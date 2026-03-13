package updater

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheck_UpdateAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(VersionInfo{
			Version:      "1.4.2",
			MinSupported: "1.0.0",
			DownloadURL:  "https://vettcode.com/download",
		})
	}))
	defer srv.Close()

	c := NewChecker(t.TempDir(), "1.1.0")
	c.Endpoint = srv.URL

	result := c.Check(context.Background())
	require.NotNil(t, result)
	assert.True(t, result.UpdateAvailable)
	assert.False(t, result.Unsupported)
	assert.Equal(t, "1.4.2", result.LatestVersion)
}

func TestCheck_Unsupported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(VersionInfo{
			Version:      "2.0.0",
			MinSupported: "1.5.0",
			DownloadURL:  "https://vettcode.com/download",
		})
	}))
	defer srv.Close()

	c := NewChecker(t.TempDir(), "1.1.0")
	c.Endpoint = srv.URL

	result := c.Check(context.Background())
	require.NotNil(t, result)
	assert.True(t, result.UpdateAvailable)
	assert.True(t, result.Unsupported)
}

func TestCheck_UpToDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(VersionInfo{
			Version:      "1.4.2",
			MinSupported: "1.0.0",
		})
	}))
	defer srv.Close()

	c := NewChecker(t.TempDir(), "1.4.2")
	c.Endpoint = srv.URL

	result := c.Check(context.Background())
	assert.Nil(t, result, "no notice when up to date")
}

func TestCheck_DevVersion_NoCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(VersionInfo{Version: "1.0.0"})
	}))
	defer srv.Close()

	c := NewChecker(t.TempDir(), "dev")
	c.Endpoint = srv.URL

	result := c.Check(context.Background())
	assert.Nil(t, result, "dev builds should not produce notices")
}

func TestCheck_CacheThrottling(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(VersionInfo{
			Version:      "2.0.0",
			MinSupported: "1.0.0",
		})
	}))
	defer srv.Close()

	dir := t.TempDir()
	c := NewChecker(dir, "1.0.0")
	c.Endpoint = srv.URL

	// First check hits network
	r1 := c.Check(context.Background())
	require.NotNil(t, r1)
	assert.Equal(t, 1, callCount)

	// Second check uses cache
	r2 := c.Check(context.Background())
	require.NotNil(t, r2)
	assert.Equal(t, 1, callCount, "should use cache, not hit network again")
}

func TestCheck_ExpiredCache(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(VersionInfo{
			Version:      "2.0.0",
			MinSupported: "1.0.0",
		})
	}))
	defer srv.Close()

	dir := t.TempDir()
	c := NewChecker(dir, "1.0.0")
	c.Endpoint = srv.URL

	// Write expired cache
	expired := CachedCheck{
		VersionInfo: VersionInfo{Version: "1.5.0"},
		CheckedAt:   time.Now().Add(-25 * time.Hour),
	}
	data, _ := json.Marshal(expired)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "version-check.json"), data, 0644))

	result := c.Check(context.Background())
	require.NotNil(t, result)
	assert.Equal(t, 1, callCount, "should fetch fresh data when cache expired")
	assert.Equal(t, "2.0.0", result.LatestVersion)
}

func TestCheck_NetworkError_Silent(t *testing.T) {
	c := NewChecker(t.TempDir(), "1.0.0")
	c.Endpoint = "http://127.0.0.1:1" // connection refused

	result := c.Check(context.Background())
	assert.Nil(t, result, "network errors should be silent")
}

func TestCheck_BadResponse_Silent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewChecker(t.TempDir(), "1.0.0")
	c.Endpoint = srv.URL

	result := c.Check(context.Background())
	assert.Nil(t, result, "server errors should be silent")
}

func TestCheck_CorruptedCache_Refetches(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(VersionInfo{
			Version:      "2.0.0",
			MinSupported: "1.0.0",
		})
	}))
	defer srv.Close()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "version-check.json"), []byte("not json"), 0644))

	c := NewChecker(dir, "1.0.0")
	c.Endpoint = srv.URL

	result := c.Check(context.Background())
	require.NotNil(t, result)
	assert.True(t, result.UpdateAvailable)
}

func TestFormatNotice_UpdateAvailable(t *testing.T) {
	r := &Result{
		UpdateAvailable: true,
		LatestVersion:   "1.4.2",
		DownloadURL:     "https://vettcode.com/download",
	}
	notice := FormatNotice(r, "1.1.0")
	assert.Contains(t, notice, "NOTE:")
	assert.Contains(t, notice, "1.1.0")
	assert.Contains(t, notice, "1.4.2")
	assert.Contains(t, notice, "brew upgrade")
}

func TestFormatNotice_Unsupported(t *testing.T) {
	r := &Result{
		Unsupported:   true,
		LatestVersion: "2.0.0",
	}
	notice := FormatNotice(r, "1.0.0")
	assert.Contains(t, notice, "WARN:")
	assert.Contains(t, notice, "no longer supported")
	assert.Contains(t, notice, "2.0.0")
}

func TestFormatNotice_EmptyDownloadURL_Fallback(t *testing.T) {
	r := &Result{
		UpdateAvailable: true,
		LatestVersion:   "1.4.2",
		DownloadURL:     "",
	}
	notice := FormatNotice(r, "1.1.0")
	assert.Contains(t, notice, "https://vettcode.com/download", "should use fallback URL")
}

func TestFormatNotice_Nil(t *testing.T) {
	assert.Empty(t, FormatNotice(nil, "1.0.0"))
}

func TestCheck_NoUpdateCheckEnvVar(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(VersionInfo{Version: "2.0.0", MinSupported: "1.0.0"})
	}))
	defer srv.Close()

	c := NewChecker(t.TempDir(), "1.0.0")
	c.Endpoint = srv.URL

	t.Setenv("VETTCODE_NO_UPDATE_CHECK", "true")
	result := c.Check(context.Background())
	assert.Nil(t, result, "should skip check when VETTCODE_NO_UPDATE_CHECK=true")
}

func TestCheck_NoUpdateCheckEnvVar_1(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(VersionInfo{Version: "2.0.0", MinSupported: "1.0.0"})
	}))
	defer srv.Close()

	c := NewChecker(t.TempDir(), "1.0.0")
	c.Endpoint = srv.URL

	t.Setenv("VETTCODE_NO_UPDATE_CHECK", "1")
	result := c.Check(context.Background())
	assert.Nil(t, result, "should skip check when VETTCODE_NO_UPDATE_CHECK=1")
}

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.1.0", "1.0.9", 1},
		{"2.0.0", "1.9.9", 1},
		{"0.9.0", "1.0.0", -1},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, compareSemver(tt.a, tt.b), "%s vs %s", tt.a, tt.b)
	}
}

func TestNormalizeVersion(t *testing.T) {
	assert.Equal(t, "1.0.0", normalizeVersion("v1.0.0"))
	assert.Equal(t, "1.0.0", normalizeVersion("1.0.0"))
	assert.Equal(t, "dev", normalizeVersion("dev"))
}

func TestCheck_VPrefixHandled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(VersionInfo{
			Version:      "v1.4.2",
			MinSupported: "v1.0.0",
		})
	}))
	defer srv.Close()

	c := NewChecker(t.TempDir(), "v1.1.0")
	c.Endpoint = srv.URL

	result := c.Check(context.Background())
	require.NotNil(t, result)
	assert.True(t, result.UpdateAvailable)
}
