// Package updater implements a non-blocking, 24h-throttled version check
// mechanism that caches results in ~/.vettcode/version-check.json.
package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// CheckInterval is the minimum time between version checks.
const CheckInterval = 24 * time.Hour

// RequestTimeout is the HTTP timeout for the version check endpoint.
const RequestTimeout = 2 * time.Second

// DefaultEndpoint is the default version check URL.
const DefaultEndpoint = "https://api.vettcode.com/api/v1/scanner/latest-version"

// VersionInfo holds the response from the version check endpoint.
type VersionInfo struct {
	Version      string `json:"version"`
	MinSupported string `json:"min_supported"`
	DownloadURL  string `json:"download_url"`
}

// CachedCheck holds cached version check data.
type CachedCheck struct {
	VersionInfo
	CheckedAt time.Time `json:"checked_at"`
}

// Checker performs version checks with caching and throttling.
type Checker struct {
	Endpoint       string
	CacheDir       string
	CurrentVersion string
	Client         *http.Client
}

// NewChecker creates a version checker with the given config directory and current version.
func NewChecker(cacheDir, currentVersion string) *Checker {
	return &Checker{
		Endpoint:       DefaultEndpoint,
		CacheDir:       cacheDir,
		CurrentVersion: currentVersion,
		Client: &http.Client{
			Timeout: RequestTimeout,
		},
	}
}

// Result describes the outcome of a version check.
type Result struct {
	// UpdateAvailable is true when a newer version exists.
	UpdateAvailable bool
	// Unsupported is true when current version is below min_supported.
	Unsupported bool
	// LatestVersion is the latest available version string.
	LatestVersion string
	// DownloadURL is the URL to download the latest version.
	DownloadURL string
}

// Check performs a throttled version check. Returns nil if no check was needed
// or if the check failed (failures are silent).
// Respects VETTCODE_NO_UPDATE_CHECK=true environment variable.
func (c *Checker) Check(ctx context.Context) *Result {
	if v := os.Getenv("VETTCODE_NO_UPDATE_CHECK"); v == "true" || v == "1" {
		return nil
	}

	cached, err := c.loadCache()
	if err == nil && time.Since(cached.CheckedAt) < CheckInterval {
		return c.compare(&cached.VersionInfo)
	}

	info, err := c.fetch(ctx)
	if err != nil {
		return nil
	}

	_ = c.saveCache(&CachedCheck{
		VersionInfo: *info,
		CheckedAt:   time.Now(),
	})

	return c.compare(info)
}

func (c *Checker) compare(info *VersionInfo) *Result {
	current := normalizeVersion(c.CurrentVersion)
	latest := normalizeVersion(info.Version)
	minSupported := normalizeVersion(info.MinSupported)

	result := &Result{
		LatestVersion: info.Version,
		DownloadURL:   info.DownloadURL,
	}

	if current == "dev" || current == "" {
		return nil // dev builds don't check
	}

	if compareSemver(current, latest) < 0 {
		result.UpdateAvailable = true
	}
	if minSupported != "" && compareSemver(current, minSupported) < 0 {
		result.Unsupported = true
	}

	if !result.UpdateAvailable && !result.Unsupported {
		return nil
	}
	return result
}

func (c *Checker) fetch(ctx context.Context) (*VersionInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "vettcode-scanner/"+c.CurrentVersion)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("version check returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return nil, err
	}

	var info VersionInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	if info.Version == "" {
		return nil, fmt.Errorf("empty version in response")
	}
	return &info, nil
}

func (c *Checker) cachePath() string {
	return filepath.Join(c.CacheDir, "version-check.json")
}

func (c *Checker) loadCache() (*CachedCheck, error) {
	data, err := os.ReadFile(c.cachePath())
	if err != nil {
		return nil, err
	}
	var cached CachedCheck
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, err
	}
	return &cached, nil
}

func (c *Checker) saveCache(cached *CachedCheck) error {
	if err := os.MkdirAll(c.CacheDir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}

	// Atomic write
	tmp := c.cachePath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, c.cachePath())
}

// FormatNotice returns a user-facing string describing the version check result.
// Returns empty string if no notice is needed.
func FormatNotice(r *Result, currentVersion string) string {
	if r == nil {
		return ""
	}
	if r.Unsupported {
		return fmt.Sprintf("WARN: This scanner version (%s) is no longer supported. "+
			"Please upgrade to %s or later.", currentVersion, r.LatestVersion)
	}
	if r.UpdateAvailable {
		dl := r.DownloadURL
		if dl == "" {
			dl = "https://vettcode.com/download"
		}
		return fmt.Sprintf("NOTE: A newer scanner version is available (%s → %s). "+
			"Run 'brew upgrade vettcode' or download from %s",
			currentVersion, r.LatestVersion, dl)
	}
	return ""
}

// normalizeVersion strips the leading 'v' prefix.
func normalizeVersion(v string) string {
	return strings.TrimPrefix(v, "v")
}

// compareSemver compares two semver strings (without 'v' prefix).
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareSemver(a, b string) int {
	aParts := parseSemver(a)
	bParts := parseSemver(b)

	for i := 0; i < 3; i++ {
		if aParts[i] < bParts[i] {
			return -1
		}
		if aParts[i] > bParts[i] {
			return 1
		}
	}
	return 0
}

func parseSemver(v string) [3]int {
	var parts [3]int
	segments := strings.SplitN(v, ".", 3)
	for i, s := range segments {
		if i >= 3 {
			break
		}
		// Strip any pre-release suffix (e.g., "1-beta" → "1")
		if idx := strings.IndexAny(s, "-+"); idx >= 0 {
			s = s[:idx]
		}
		n, _ := strconv.Atoi(s)
		parts[i] = n
	}
	return parts
}
