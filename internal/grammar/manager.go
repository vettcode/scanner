package grammar

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	// GCSBaseURL is the base URL for grammar downloads.
	GCSBaseURL = "https://storage.googleapis.com/vettcode-grammars"

	// DownloadTimeout is the maximum time for a grammar download.
	DownloadTimeout = 60 * time.Second
)

// GrammarVersion maps each scanner version to the expected grammar version.
// Updated with each scanner release.
var GrammarVersion = "0.1.0"

// GrammarManifest maps language name to expected SHA-256 checksum.
// In production, these would be hardcoded per scanner version.
// For MVP, we'll use a placeholder that gets populated from the server.
var GrammarManifest = map[string]GrammarEntry{
	"javascript": {Language: "javascript", Filename: "tree-sitter-javascript.wasm"},
	"typescript": {Language: "typescript", Filename: "tree-sitter-typescript.wasm"},
	"python":     {Language: "python", Filename: "tree-sitter-python.wasm"},
	"php":        {Language: "php", Filename: "tree-sitter-php.wasm"},
	"ruby":       {Language: "ruby", Filename: "tree-sitter-ruby.wasm"},
	"java":       {Language: "java", Filename: "tree-sitter-java.wasm"},
}

// GrammarEntry describes a downloadable grammar.
type GrammarEntry struct {
	Language string
	Filename string
	SHA256   string // expected checksum (empty = skip verification in dev)
}

// Manager handles grammar downloading, caching, and verification.
type Manager struct {
	cacheDir string
	offline  bool
	baseURL  string
}

// NewManager creates a grammar manager.
func NewManager(vettcodeHome string, offline bool) *Manager {
	return &Manager{
		cacheDir: filepath.Join(vettcodeHome, "grammars", GrammarVersion),
		offline:  offline,
		baseURL:  GCSBaseURL,
	}
}

// SetBaseURL overrides the grammar download base URL (used for testing).
func (m *Manager) SetBaseURL(url string) {
	m.baseURL = url
}

// CacheDir returns the grammar cache directory path.
func (m *Manager) CacheDir() string {
	return m.cacheDir
}

// EnsureGrammar ensures a grammar is cached locally, downloading if needed.
func (m *Manager) EnsureGrammar(lang string) (string, error) {
	entry, ok := GrammarManifest[lang]
	if !ok {
		return "", fmt.Errorf("unsupported grammar language: %s", lang)
	}

	cachedPath := filepath.Join(m.cacheDir, entry.Filename)

	// Check if cached
	if _, err := os.Stat(cachedPath); err == nil {
		// Verify checksum if available
		if entry.SHA256 != "" {
			if err := verifyChecksum(cachedPath, entry.SHA256); err != nil {
				slog.Warn("cached grammar checksum mismatch, re-downloading",
					"language", lang, "error", err)
				os.Remove(cachedPath)
			} else {
				return cachedPath, nil
			}
		} else {
			return cachedPath, nil
		}
	}

	// Download if not offline
	if m.offline {
		return "", fmt.Errorf("grammar not cached for %s and running in offline mode.\n"+
			"  Fix: Run a scan with network access first to download grammars,\n"+
			"       or use the Docker image which bundles all grammars.", lang)
	}

	return m.download(entry)
}

// GrammarPath returns the expected cache path for a grammar.
func (m *Manager) GrammarPath(lang string) string {
	entry, ok := GrammarManifest[lang]
	if !ok {
		return ""
	}
	return filepath.Join(m.cacheDir, entry.Filename)
}

// download fetches a grammar from GCS and caches it.
func (m *Manager) download(entry GrammarEntry) (string, error) {
	url := fmt.Sprintf("%s/%s/%s", m.baseURL, GrammarVersion, entry.Filename)
	slog.Info("downloading grammar", "language", entry.Language, "url", url)

	fmt.Fprintf(os.Stderr, "Downloading %s grammar... ", entry.Language)

	client := &http.Client{Timeout: DownloadTimeout}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed")
		return "", fmt.Errorf("failed to download %s grammar: %w", entry.Language, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "failed")
		return "", fmt.Errorf("failed to download %s grammar: HTTP %d", entry.Language, resp.StatusCode)
	}

	// Create cache directory
	if err := os.MkdirAll(m.cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create grammar cache directory: %w", err)
	}

	// Write to temp file first, then rename
	tmpPath := filepath.Join(m.cacheDir, entry.Filename+".tmp")
	f, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tmpPath)
		fmt.Fprintln(os.Stderr, "failed")
		return "", fmt.Errorf("failed to write grammar: %w", err)
	}

	// Verify checksum if available
	if entry.SHA256 != "" {
		if err := verifyChecksum(tmpPath, entry.SHA256); err != nil {
			os.Remove(tmpPath)
			fmt.Fprintln(os.Stderr, "failed (checksum)")
			return "", fmt.Errorf("grammar checksum verification failed for %s: %w", entry.Language, err)
		}
	}

	// Atomic rename
	finalPath := filepath.Join(m.cacheDir, entry.Filename)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to cache grammar: %w", err)
	}

	fmt.Fprintln(os.Stderr, "done")
	return finalPath, nil
}

func verifyChecksum(path, expectedHex string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if actual != expectedHex {
		return fmt.Errorf("expected %s, got %s", expectedHex, actual)
	}
	return nil
}
