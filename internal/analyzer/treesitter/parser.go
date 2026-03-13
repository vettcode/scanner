package treesitter

// Parser provides an interface for tree-sitter based parsing.
// The actual tree-sitter integration will be implemented in Epic 3
// when complexity analyzers are built. This package defines the
// interface and common types used by all language analyzers.

// FunctionInfo holds complexity info for a function found via tree-sitter.
type FunctionInfo struct {
	Name       string
	StartLine  int
	EndLine    int
	Complexity int
	MaxNesting int
}

// FileAnalysis holds the analysis result for a single source file.
type FileAnalysis struct {
	Path      string
	Language  string
	Functions []FunctionInfo
	LOC       int
}

// LanguageAnalyzer is the interface for language-specific complexity analyzers.
type LanguageAnalyzer interface {
	// AnalyzeFile parses a file and returns function-level complexity data.
	AnalyzeFile(path string) (*FileAnalysis, error)

	// Language returns the language this analyzer handles.
	Language() string
}
