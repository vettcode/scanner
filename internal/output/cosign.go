package output

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/vettcode/scanner/pkg/models"
)

const (
	defaultCosignBaseURL = "https://api.vettcode.com"
	cosignInitPath       = "/api/v1/cosign/init"
	cosignCompletePath   = "/api/v1/cosign/complete"

	cosignReadTimeout = 10 * time.Second
	maxRetries           = 2
)

// CosignClient handles the remote co-signing flow with the VettCode platform.
type CosignClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewCosignClient creates a co-signing client with appropriate timeouts.
func NewCosignClient() *CosignClient {
	return &CosignClient{
		BaseURL: defaultCosignBaseURL,
		HTTPClient: &http.Client{
			Timeout: cosignReadTimeout,
		},
	}
}

type cosignInitResponse struct {
	SessionID string `json:"session_id"`
	Nonce     string `json:"nonce"`
	ExpiresAt string `json:"expires_at"`
}

type cosignCompleteRequest struct {
	SessionID          string `json:"session_id"`
	ScanChecksum       string `json:"scan_checksum"`
	ScannerSignature   string `json:"scanner_signature"`
	ScannerPublicKeyID string `json:"scanner_public_key_id"`
}

type cosignCompleteResponse struct {
	PlatformCosignature string `json:"platform_cosignature"`
	PlatformPublicKeyID string `json:"platform_public_key_id"`
}

type cosignErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// CosignResult is the outcome of a co-signing attempt.
type CosignResult struct {
	Success bool
	Warning string // non-empty if fell back to offline
}

// Cosign performs the full co-signing flow:
//  1. POST /cosign/init → get nonce
//  2. Embed nonce in integrity block, re-sign
//  3. POST /cosign/complete → get platform co-signature
//
// On failure (network, rate limit, server error), falls back to offline mode
// with a warning. Fatal errors (invalid signature, unknown/revoked key) return error.
func (c *CosignClient) Cosign(ctx context.Context, result *models.ScanResult) (*CosignResult, error) {
	// Step 1: Init — get nonce
	initResp, err := c.initWithRetry(ctx)
	if err != nil {
		return &CosignResult{
			Success: false,
			Warning: fmt.Sprintf("Co-signing unavailable (%v) — scan will be self-reported. Use --offline to suppress this warning.", err),
		}, nil
	}

	// Step 2: Embed nonce and re-sign (nonce is preserved by SignScanResult)
	nonce := initResp.Nonce
	result.Integrity.CosignNonce = &nonce

	if err := SignScanResult(result); err != nil {
		return nil, fmt.Errorf("re-sign with nonce: %w", err)
	}

	// Step 3: Complete — get platform co-signature
	completeResp, err := c.completeWithRetry(ctx, initResp.SessionID, result)
	if err != nil {
		// Check if this is a fatal error
		if isFatalCosignError(err) {
			return nil, err
		}
		// Recoverable — fall back to offline
		result.Integrity.CosignNonce = nil
		if signErr := SignScanResult(result); signErr != nil {
			return nil, fmt.Errorf("re-sign after cosign failure: %w", signErr)
		}
		return &CosignResult{
			Success: false,
			Warning: fmt.Sprintf("Co-signing unavailable (%v) — scan will be self-reported. Use --offline to suppress this warning.", err),
		}, nil
	}

	// Success — embed platform co-signature
	result.Integrity.PlatformCosignature = &completeResp.PlatformCosignature
	result.Integrity.PlatformPublicKeyID = &completeResp.PlatformPublicKeyID
	result.Integrity.Cosigned = true

	return &CosignResult{Success: true}, nil
}

func (c *CosignClient) initWithRetry(ctx context.Context) (*cosignInitResponse, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			if err := backoff(ctx, attempt); err != nil {
				return nil, err
			}
		}

		resp, err := c.doInit(ctx)
		if err == nil {
			return resp, nil
		}

		if re, ok := err.(*retryableError); ok {
			lastErr = re
			if re.retryAfter > 0 {
				select {
				case <-time.After(time.Duration(re.retryAfter) * time.Second):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			continue
		}

		return nil, err // non-retryable
	}
	return nil, fmt.Errorf("init failed after retries: %w", lastErr)
}

func (c *CosignClient) doInit(ctx context.Context) (*cosignInitResponse, error) {
	url := c.BaseURL + cosignInitPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, fmt.Errorf("create init request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, &retryableError{cause: err}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))

	if resp.StatusCode == http.StatusCreated {
		var r cosignInitResponse
		if err := json.Unmarshal(body, &r); err != nil {
			return nil, fmt.Errorf("decode init response: %w", err)
		}
		if r.SessionID == "" || r.Nonce == "" {
			return nil, fmt.Errorf("init response missing session_id or nonce")
		}
		return &r, nil
	}

	return nil, classifyHTTPError(resp.StatusCode, body, resp.Header)
}

func (c *CosignClient) completeWithRetry(ctx context.Context, sessionID string, result *models.ScanResult) (*cosignCompleteResponse, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			if err := backoff(ctx, attempt); err != nil {
				return nil, err
			}
		}

		resp, err := c.doComplete(ctx, sessionID, result)
		if err == nil {
			return resp, nil
		}

		if re, ok := err.(*retryableError); ok {
			lastErr = err
			if re.retryAfter > 0 {
				select {
				case <-time.After(time.Duration(re.retryAfter) * time.Second):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			continue
		}

		// Non-retryable errors
		if fe, ok := err.(*fatalCosignError); ok {
			// Special cases: expired nonce or unknown session → restart from init (once)
			if (fe.code == "nonce_expired" || fe.code == "session_not_found") && attempt == 0 {
				initResp, initErr := c.doInit(ctx)
				if initErr != nil {
					return nil, fmt.Errorf("restart init: %w", initErr)
				}
				sessionID = initResp.SessionID
				nonce := initResp.Nonce
				result.Integrity.CosignNonce = &nonce
				if signErr := SignScanResult(result); signErr != nil {
					return nil, signErr
				}
				continue
			}
			return nil, fe
		}

		return nil, err
	}
	return nil, fmt.Errorf("complete failed after retries: %w", lastErr)
}

func (c *CosignClient) doComplete(ctx context.Context, sessionID string, result *models.ScanResult) (*cosignCompleteResponse, error) {
	url := c.BaseURL + cosignCompletePath
	reqBody := cosignCompleteRequest{
		SessionID:          sessionID,
		ScanChecksum:       result.Integrity.ScanChecksum,
		ScannerSignature:   result.Integrity.ScannerSignature,
		ScannerPublicKeyID: result.Integrity.ScannerPublicKeyID,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal complete request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create complete request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, &retryableError{cause: err}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))

	if resp.StatusCode == http.StatusOK {
		var r cosignCompleteResponse
		if err := json.Unmarshal(body, &r); err != nil {
			return nil, fmt.Errorf("decode complete response: %w", err)
		}
		if r.PlatformCosignature == "" || r.PlatformPublicKeyID == "" {
			return nil, fmt.Errorf("complete response missing platform_cosignature or platform_public_key_id")
		}
		return &r, nil
	}

	return nil, classifyHTTPError(resp.StatusCode, body, resp.Header)
}

// Error types

type retryableError struct {
	cause      error
	retryAfter int
}

func (e *retryableError) Error() string {
	if e.cause != nil {
		return e.cause.Error()
	}
	return "retryable error"
}

func (e *retryableError) Unwrap() error { return e.cause }

type fatalCosignError struct {
	code    string
	message string
}

func (e *fatalCosignError) Error() string { return e.message }

func isFatalCosignError(err error) bool {
	_, ok := err.(*fatalCosignError)
	return ok
}

// classifyHTTPError maps HTTP status codes to retryable or fatal errors
// per the co-sign API contract in Section 5.8.
func classifyHTTPError(status int, body []byte, header http.Header) error {
	var apiErr cosignErrorResponse
	json.Unmarshal(body, &apiErr) // best-effort parse

	switch status {
	case http.StatusTooManyRequests: // 429
		retryAfter := 0
		if ra := header.Get("Retry-After"); ra != "" {
			if v, err := strconv.Atoi(ra); err == nil {
				retryAfter = v
			}
		}
		return &retryableError{
			cause:      fmt.Errorf("rate limited: %s", apiErr.Message),
			retryAfter: retryAfter,
		}

	case http.StatusBadRequest: // 400
		switch apiErr.Error {
		case "invalid_signature":
			return &fatalCosignError{
				code:    "invalid_signature",
				message: "Scanner signature rejected by platform. This may indicate a corrupted or tampered scanner binary. Please re-download from vettcode.com.",
			}
		case "unknown_key_id":
			return &fatalCosignError{
				code:    "unknown_key_id",
				message: "Scanner signing key not recognized. Please update to the latest version.",
			}
		case "revoked_key":
			return &fatalCosignError{
				code:    "revoked_key",
				message: "This scanner version has been revoked. Please update to the latest version immediately.",
			}
		default:
			return &fatalCosignError{
				code:    apiErr.Error,
				message: fmt.Sprintf("Co-sign error: %s", apiErr.Message),
			}
		}

	case http.StatusNotFound: // 404
		return &fatalCosignError{
			code:    "session_not_found",
			message: "Co-sign session not found",
		}

	case http.StatusGone: // 410
		switch apiErr.Error {
		case "nonce_already_used":
			return &fatalCosignError{
				code:    "nonce_already_used",
				message: "Co-sign nonce conflict. Please retry the scan.",
			}
		default:
			return &fatalCosignError{
				code:    "nonce_expired",
				message: "Co-sign nonce expired",
			}
		}

	default:
		if status >= 500 {
			return &retryableError{
				cause: fmt.Errorf("server error %d: %s", status, apiErr.Message),
			}
		}
		return fmt.Errorf("unexpected status %d: %s", status, apiErr.Message)
	}
}

func backoff(ctx context.Context, attempt int) error {
	delays := []time.Duration{1 * time.Second, 3 * time.Second}
	idx := attempt - 1
	if idx >= len(delays) {
		idx = len(delays) - 1
	}
	select {
	case <-time.After(delays[idx]):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
