package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RepoInput represents a validated scan target.
type RepoInput struct {
	Name string // user-provided label or directory name
	Path string // absolute path
}

// ParsePaths validates scan paths and labels, returning RepoInput entries.
// If no paths are given, defaults to the current directory.
func ParsePaths(args []string, labels []string) ([]RepoInput, error) {
	// Build label map: name -> path (ordered by label name for deterministic output)
	labelMap := make(map[string]string)
	var labelNames []string
	for _, l := range labels {
		parts := strings.SplitN(l, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid --label format %q: expected name:path (e.g., --label frontend:./fe)", l)
		}
		labelMap[parts[0]] = parts[1]
		labelNames = append(labelNames, parts[0])
	}

	// If labels are provided but no positional args, use labels in sorted order
	if len(args) == 0 && len(labelMap) > 0 {
		sort.Strings(labelNames)
		var repos []RepoInput
		seen := make(map[string]bool)
		for _, name := range labelNames {
			rawPath := labelMap[name]
			absPath, err := validatePath(rawPath)
			if err != nil {
				return nil, err
			}
			if seen[absPath] {
				return nil, fmt.Errorf("duplicate path: %s", rawPath)
			}
			seen[absPath] = true
			repos = append(repos, RepoInput{Name: name, Path: absPath})
		}
		return repos, nil
	}

	// Default to current directory if no args
	if len(args) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("cannot determine current directory: %w", err)
		}
		args = []string{cwd}
	}

	// Build reverse label map: path -> name
	pathToName := make(map[string]string)
	for name, path := range labelMap {
		absPath, err := filepath.Abs(path)
		if err == nil {
			pathToName[absPath] = name
		}
	}

	var repos []RepoInput
	seen := make(map[string]bool)

	for _, rawPath := range args {
		absPath, err := validatePath(rawPath)
		if err != nil {
			return nil, err
		}

		if seen[absPath] {
			return nil, fmt.Errorf("duplicate path: %s", rawPath)
		}
		seen[absPath] = true

		name := pathToName[absPath]
		if name == "" {
			name = filepath.Base(absPath)
		}

		repos = append(repos, RepoInput{Name: name, Path: absPath})
	}

	return repos, nil
}

// ValidateOutputPath checks that the output file's parent directory exists and is writable.
func ValidateOutputPath(outputPath string) error {
	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("invalid output path %q: %w", outputPath, err)
	}

	dir := filepath.Dir(absPath)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("output directory does not exist: %s\n  Fix: Create the directory or choose a different path with --output", dir)
		}
		return fmt.Errorf("cannot access output directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("output parent path is not a directory: %s", dir)
	}

	// Test write permission by creating a temp file
	testFile := absPath + ".vettcode-write-test"
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("cannot write to output path: %s\n  Fix: Choose a writable output path with --output flag:\n       vettcode scan ./my-project --output ~/vettcode-scan.json", absPath)
	}
	f.Close()
	os.Remove(testFile)

	return nil
}

func validatePath(rawPath string) (string, error) {
	absPath, err := filepath.Abs(rawPath)
	if err != nil {
		return "", fmt.Errorf("invalid path %q: %w", rawPath, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("path does not exist: %s", rawPath)
		}
		return "", fmt.Errorf("cannot access path %s: %w", rawPath, err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", rawPath)
	}

	return absPath, nil
}
