package grammar

import (
	"crypto/sha256"
	"encoding/hex"
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

func TestGrammarManifest_AllLanguages(t *testing.T) {
	expected := []string{"javascript", "typescript", "python", "php", "ruby", "java"}
	for _, lang := range expected {
		entry, ok := GrammarManifest[lang]
		assert.True(t, ok, "missing grammar for %s", lang)
		assert.NotEmpty(t, entry.Filename)
		assert.Contains(t, entry.Filename, ".wasm")
	}
}
