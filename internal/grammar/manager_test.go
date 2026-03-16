package grammar

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	m := NewManager("/tmp/vettcode-test", false)
	assert.Contains(t, m.cacheDir, "grammars")
	assert.False(t, m.offline)
}

func TestEnsureGrammar_UnsupportedLanguage(t *testing.T) {
	m := NewManager(t.TempDir(), false)
	_, err := m.EnsureGrammar("brainfuck")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported grammar")
}

func TestEnsureGrammar_OfflineNotCached(t *testing.T) {
	m := NewManager(t.TempDir(), true)
	_, err := m.EnsureGrammar("javascript")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "offline mode")
}

func TestEnsureGrammar_CachedFile(t *testing.T) {
	home := t.TempDir()
	m := NewManager(home, true) // offline but cached

	// Create a fake cached grammar
	cacheDir := filepath.Join(home, "grammars", GrammarVersion)
	os.MkdirAll(cacheDir, 0755)
	grammarPath := filepath.Join(cacheDir, "tree-sitter-javascript.wasm")
	os.WriteFile(grammarPath, []byte("fake wasm content"), 0644)

	path, err := m.EnsureGrammar("javascript")
	require.NoError(t, err)
	assert.Equal(t, grammarPath, path)
}

func TestGrammarPath(t *testing.T) {
	m := NewManager("/tmp/vettcode", false)
	path := m.GrammarPath("javascript")
	assert.Contains(t, path, "tree-sitter-javascript.wasm")

	path = m.GrammarPath("unknown")
	assert.Empty(t, path)
}

func TestVerifyChecksum(t *testing.T) {
	dir := t.TempDir()
	content := []byte("test content for checksum")
	path := filepath.Join(dir, "test.wasm")
	os.WriteFile(path, content, 0644)

	h := sha256.Sum256(content)
	expected := hex.EncodeToString(h[:])

	err := verifyChecksum(path, expected)
	assert.NoError(t, err)

	err = verifyChecksum(path, "badhash")
	assert.Error(t, err)
}

func TestVerifyChecksum_NonExistentFile(t *testing.T) {
	err := verifyChecksum("/nonexistent/file", "abc123")
	assert.Error(t, err)
}

func TestEnsureGrammar_MockDownload_CorrectChecksum(t *testing.T) {
	content := []byte("fake wasm grammar content for test")
	h := sha256.Sum256(content)
	expectedHash := hex.EncodeToString(h[:])

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer srv.Close()

	home := t.TempDir()
	m := NewManager(home, false)
	m.SetBaseURL(srv.URL)

	// Temporarily set a checksum for JavaScript grammar
	origEntry := GrammarManifest["javascript"]
	GrammarManifest["javascript"] = GrammarEntry{
		Language: "javascript",
		Filename: "tree-sitter-javascript.wasm",
		SHA256:   expectedHash,
	}
	defer func() { GrammarManifest["javascript"] = origEntry }()

	path, err := m.EnsureGrammar("javascript")
	require.NoError(t, err)
	assert.FileExists(t, path)

	// Verify content was saved correctly
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestEnsureGrammar_MockDownload_WrongChecksum(t *testing.T) {
	content := []byte("fake wasm grammar content")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer srv.Close()

	home := t.TempDir()
	m := NewManager(home, false)
	m.SetBaseURL(srv.URL)

	// Set a wrong checksum
	origEntry := GrammarManifest["javascript"]
	GrammarManifest["javascript"] = GrammarEntry{
		Language: "javascript",
		Filename: "tree-sitter-javascript.wasm",
		SHA256:   "0000000000000000000000000000000000000000000000000000000000000000",
	}
	defer func() { GrammarManifest["javascript"] = origEntry }()

	_, err := m.EnsureGrammar("javascript")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum verification failed")
}

func TestEnsureGrammar_MockDownload_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	home := t.TempDir()
	m := NewManager(home, false)
	m.SetBaseURL(srv.URL)

	_, err := m.EnsureGrammar("javascript")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
}

func TestEnsureGrammar_CacheHit_NoDownload(t *testing.T) {
	downloadCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloadCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("wasm content"))
	}))
	defer srv.Close()

	home := t.TempDir()
	m := NewManager(home, false)
	m.SetBaseURL(srv.URL)

	// First call downloads
	path1, err := m.EnsureGrammar("javascript")
	require.NoError(t, err)
	assert.Equal(t, 1, downloadCount)

	// Second call should use cache (no SHA256 set, so no verification needed)
	path2, err := m.EnsureGrammar("javascript")
	require.NoError(t, err)
	assert.Equal(t, path1, path2)
	assert.Equal(t, 1, downloadCount, "second call should use cache, no download")
}

func TestGrammarManifest_AllLanguages(t *testing.T) {
	expected := []string{"javascript", "typescript", "python", "php", "ruby", "java"}
	for _, lang := range expected {
		entry, ok := GrammarManifest[lang]
		assert.True(t, ok, "missing grammar for %s", lang)
		assert.NotEmpty(t, entry.Filename)
		assert.Contains(t, entry.Filename, ".wasm")
	}
}
