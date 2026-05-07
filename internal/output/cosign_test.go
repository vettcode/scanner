package output

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCosign_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case cosignInitPath:
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(cosignInitResponse{
				SessionID: "sess-123",
				Nonce:     "nonce-abc",
				ExpiresAt: "2026-03-13T11:00:00Z",
			})
		case cosignCompletePath:
			var req cosignCompleteRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, "sess-123", req.SessionID)
			assert.NotEmpty(t, req.ScanChecksum)
			assert.NotEmpty(t, req.ScannerSignature)
			assert.Equal(t, ScannerKeyID, req.ScannerPublicKeyID)

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(cosignCompleteResponse{
				PlatformCosignature: "platform-sig-xyz",
				PlatformPublicKeyID: "vettcode-platform-key-2026-03",
			})
		}
	}))
	defer server.Close()

	client := &CosignClient{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	result := newTestScanResult()
	require.NoError(t, SignScanResult(result))

	cr, err := client.Cosign(context.Background(), result)
	require.NoError(t, err)
	assert.True(t, cr.Success)
	assert.Empty(t, cr.Warning)

	assert.True(t, result.Integrity.Cosigned)
	assert.NotNil(t, result.Integrity.CosignNonce)
	assert.Equal(t, "nonce-abc", *result.Integrity.CosignNonce)
	assert.NotNil(t, result.Integrity.PlatformCosignature)
	assert.Equal(t, "platform-sig-xyz", *result.Integrity.PlatformCosignature)
	assert.NotNil(t, result.Integrity.PlatformPublicKeyID)
	assert.Equal(t, "vettcode-platform-key-2026-03", *result.Integrity.PlatformPublicKeyID)
}

func TestCosign_NetworkError_FallsBackToOffline(t *testing.T) {
	// Server that immediately closes connections
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer server.Close()

	client := &CosignClient{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	result := newTestScanResult()
	require.NoError(t, SignScanResult(result))

	cr, err := client.Cosign(context.Background(), result)
	require.NoError(t, err) // not a fatal error
	assert.False(t, cr.Success)
	assert.Contains(t, cr.Warning, "Co-signing unavailable")
	assert.Contains(t, cr.Warning, "self-reported")
}

func TestCosign_RateLimited_RetriesThenFallback(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(cosignErrorResponse{
			Error:   "rate_limit_exceeded",
			Message: "Too many requests",
		})
	}))
	defer server.Close()

	client := &CosignClient{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	result := newTestScanResult()
	require.NoError(t, SignScanResult(result))

	cr, err := client.Cosign(context.Background(), result)
	require.NoError(t, err)
	assert.False(t, cr.Success)
	assert.Contains(t, cr.Warning, "Co-signing unavailable")
	// Should have retried
	assert.GreaterOrEqual(t, int(atomic.LoadInt32(&attempts)), 2)
}

func TestCosign_InvalidSignature_Fatal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case cosignInitPath:
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(cosignInitResponse{
				SessionID: "sess-123",
				Nonce:     "nonce-abc",
			})
		case cosignCompletePath:
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(cosignErrorResponse{
				Error:   "invalid_signature",
				Message: "Signature verification failed",
			})
		}
	}))
	defer server.Close()

	client := &CosignClient{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	result := newTestScanResult()
	require.NoError(t, SignScanResult(result))

	_, err := client.Cosign(context.Background(), result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "corrupted or tampered")
}

func TestCosign_UnknownKeyID_Fatal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case cosignInitPath:
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(cosignInitResponse{
				SessionID: "sess-123",
				Nonce:     "nonce-abc",
			})
		case cosignCompletePath:
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(cosignErrorResponse{
				Error:   "unknown_key_id",
				Message: "Key not recognized",
			})
		}
	}))
	defer server.Close()

	client := &CosignClient{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	result := newTestScanResult()
	require.NoError(t, SignScanResult(result))

	_, err := client.Cosign(context.Background(), result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update to the latest version")
}

func TestCosign_NonceExpired_RestartsOnce(t *testing.T) {
	var initCalls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case cosignInitPath:
			count := atomic.AddInt32(&initCalls, 1)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(cosignInitResponse{
				SessionID: "sess-" + string(rune('0'+count)),
				Nonce:     "nonce-" + string(rune('0'+count)),
			})
		case cosignCompletePath:
			if atomic.LoadInt32(&initCalls) == 1 {
				// First complete → nonce expired
				w.WriteHeader(http.StatusGone)
				json.NewEncoder(w).Encode(cosignErrorResponse{
					Error:   "nonce_expired",
					Message: "Nonce has expired",
				})
			} else {
				// Second complete → success
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(cosignCompleteResponse{
					PlatformCosignature: "platform-sig",
					PlatformPublicKeyID: "platform-key",
				})
			}
		}
	}))
	defer server.Close()

	client := &CosignClient{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	result := newTestScanResult()
	require.NoError(t, SignScanResult(result))

	cr, err := client.Cosign(context.Background(), result)
	require.NoError(t, err)
	assert.True(t, cr.Success)
	assert.Equal(t, int32(2), atomic.LoadInt32(&initCalls))
}

func TestCosign_ServerError_RetriesThenFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case cosignInitPath:
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(cosignInitResponse{
				SessionID: "sess-123",
				Nonce:     "nonce-abc",
			})
		case cosignCompletePath:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(cosignErrorResponse{
				Error:   "internal_error",
				Message: "Something went wrong",
			})
		}
	}))
	defer server.Close()

	client := &CosignClient{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	result := newTestScanResult()
	require.NoError(t, SignScanResult(result))

	cr, err := client.Cosign(context.Background(), result)
	require.NoError(t, err)
	assert.False(t, cr.Success)
	assert.Contains(t, cr.Warning, "Co-signing unavailable")
}

func TestClassifyHTTPError_RevokedKey(t *testing.T) {
	body, _ := json.Marshal(cosignErrorResponse{Error: "revoked_key", Message: "Key revoked"})
	err := classifyHTTPError(400, body, http.Header{})
	assert.True(t, isFatalCosignError(err))
	assert.Contains(t, err.Error(), "revoked")
}

func TestClassifyHTTPError_NonceAlreadyUsed(t *testing.T) {
	body, _ := json.Marshal(cosignErrorResponse{Error: "nonce_already_used", Message: "Already used"})
	err := classifyHTTPError(410, body, http.Header{})
	assert.True(t, isFatalCosignError(err))
	assert.Contains(t, err.Error(), "nonce conflict")
}
