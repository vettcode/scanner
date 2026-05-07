//go:build !cgo

package complexity

// AnalyzeFile returns an empty result when CGO is disabled (tree-sitter unavailable).
func AnalyzeFile(path, lang string) (*FileResult, error) {
	return &FileResult{Path: path, Language: lang}, nil
}
