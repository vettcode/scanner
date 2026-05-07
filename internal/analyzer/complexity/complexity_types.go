package complexity

// FunctionResult holds complexity data for a single function.
type FunctionResult struct {
	Name       string
	StartLine  int
	EndLine    int
	Complexity int
	MaxNesting int
}

// Token represents a normalized token extracted from the AST.
type Token struct {
	Value string
	Line  int // 1-based line number
}

// FileResult holds complexity analysis for a single file.
type FileResult struct {
	Path      string
	Language  string
	Functions []FunctionResult
	Tokens    []Token // normalized token stream for duplication detection
}

// Summary holds aggregate complexity statistics across all analyzed files.
type Summary struct {
	TotalFunctions int
	AvgComplexity  float64
	AvgNesting     float64
	MaxComplexity  int
	MaxNesting     int
	HighComplexity int // functions with complexity > 10
}

// Summarize computes aggregate metrics from multiple file results.
func Summarize(results []*FileResult) Summary {
	var s Summary
	totalComplexity := 0
	totalNesting := 0

	for _, fr := range results {
		if fr == nil {
			continue
		}
		for _, fn := range fr.Functions {
			s.TotalFunctions++
			totalComplexity += fn.Complexity
			totalNesting += fn.MaxNesting
			if fn.Complexity > s.MaxComplexity {
				s.MaxComplexity = fn.Complexity
			}
			if fn.MaxNesting > s.MaxNesting {
				s.MaxNesting = fn.MaxNesting
			}
			if fn.Complexity > 10 {
				s.HighComplexity++
			}
		}
	}

	if s.TotalFunctions > 0 {
		s.AvgComplexity = float64(totalComplexity) / float64(s.TotalFunctions)
		s.AvgNesting = float64(totalNesting) / float64(s.TotalFunctions)
	}
	return s
}
