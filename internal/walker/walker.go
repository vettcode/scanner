package walker

import (
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/vettcode/scanner/internal/exclusion"
	"github.com/vettcode/scanner/internal/language"
)

// FileInfo holds metadata about a walked file.
type FileInfo struct {
	Path     string
	RelPath  string
	Language string
	Tier     language.Tier
	IsTest   bool
	LOC      int
}

// WalkResult holds the results of walking a directory.
type WalkResult struct {
	Files       []FileInfo
	LanguageLOC map[string]int // language -> total LOC
	TotalLOC    int
	TotalFiles  int
	TestFiles   int
	SourceFiles int
}

// Walk traverses a directory tree, applying exclusion filters and classifying files.
// Symlinks are not followed (filepath.WalkDir default behavior).
func Walk(root string) (*WalkResult, error) {
	result := &WalkResult{
		LanguageLOC: make(map[string]int),
	}

	var mu sync.Mutex
	var files []FileInfo

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			slog.Warn("skipping inaccessible path", "path", path, "error", err)
			return nil
		}

		if d.IsDir() {
			if exclusion.ShouldExcludeDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip excluded file patterns
		if exclusion.ShouldExcludeFile(path) {
			return nil
		}

		// Skip binary files
		if exclusion.IsBinaryFile(path) {
			return nil
		}

		// Skip generated files
		if exclusion.IsGeneratedFile(path) {
			return nil
		}

		// Classify the file
		classification := language.ClassifyFile(path)
		if classification == nil {
			return nil // unrecognized language, skip
		}

		loc := countLOC(path)
		if loc == 0 {
			return nil
		}

		relPath, relErr := filepath.Rel(root, path)
		if relErr != nil {
			relPath = path // fall back to absolute path
		}

		fi := FileInfo{
			Path:     path,
			RelPath:  relPath,
			Language: classification.Language,
			Tier:     classification.Tier,
			IsTest:   classification.IsTest,
			LOC:      loc,
		}

		mu.Lock()
		files = append(files, fi)
		mu.Unlock()

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Aggregate results
	for _, f := range files {
		result.LanguageLOC[f.Language] += f.LOC
		result.TotalLOC += f.LOC
		result.TotalFiles++
		if f.IsTest {
			result.TestFiles++
		} else {
			result.SourceFiles++
		}
	}
	result.Files = files

	return result, nil
}

// countLOC counts non-blank lines in a file.
// Uses a chunked byte reader instead of bufio.Scanner to handle
// files with arbitrarily long lines (e.g. minified JS bundles).
func countLOC(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	buf := make([]byte, 32*1024)
	count := 0
	lineHasContent := false

	for {
		n, err := f.Read(buf)
		for i := 0; i < n; i++ {
			if buf[i] == '\n' {
				if lineHasContent {
					count++
				}
				lineHasContent = false
			} else if buf[i] != '\r' && buf[i] != ' ' && buf[i] != '\t' {
				lineHasContent = true
			}
		}
		if err != nil {
			break
		}
	}
	// Count last line if it has content and no trailing newline
	if lineHasContent {
		count++
	}
	return count
}
