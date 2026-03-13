package goast

import (
	"go/ast"
	"go/parser"
	goscanner "go/scanner"
	"go/token"
	"os"
)

// FunctionInfo holds metadata about a Go function.
type FunctionInfo struct {
	Name       string
	StartLine  int
	EndLine    int
	Complexity int
	MaxNesting int
}

// Summary holds aggregate complexity statistics for Go files.
type Summary struct {
	TotalFunctions int
	AvgComplexity  float64
	AvgNesting     float64
	MaxComplexity  int
	MaxNesting     int
	HighComplexity int // functions with complexity > 10
}

// Summarize computes aggregate metrics from multiple Go function results.
func Summarize(funcs []FunctionInfo) Summary {
	var s Summary
	totalComplexity := 0
	totalNesting := 0

	for _, fn := range funcs {
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

	if s.TotalFunctions > 0 {
		s.AvgComplexity = float64(totalComplexity) / float64(s.TotalFunctions)
		s.AvgNesting = float64(totalNesting) / float64(s.TotalFunctions)
	}
	return s
}

// Token represents a normalized token for duplication detection.
type Token struct {
	Value string
	Line  int // 1-based line number
}

// ExtractTokens tokenizes a Go source file into a normalized token stream.
// Identifiers → "$ID", literals → "$LIT", keywords/operators → verbatim.
func ExtractTokens(path string) ([]Token, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	file := fset.AddFile(path, -1, len(source))

	var s goscanner.Scanner
	s.Init(file, source, nil, 0) // mode 0 = skip comments

	var tokens []Token
	for {
		pos, tok, _ := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.COMMENT {
			continue
		}

		line := fset.Position(pos).Line
		var value string
		switch {
		case tok == token.IDENT:
			value = "$ID"
		case tok == token.INT || tok == token.FLOAT || tok == token.IMAG ||
			tok == token.CHAR || tok == token.STRING:
			value = "$LIT"
		default:
			value = tok.String()
		}
		tokens = append(tokens, Token{Value: value, Line: line})
	}

	return tokens, nil
}

// ParseFile parses a Go source file and returns its AST.
func ParseFile(path string) (*ast.File, *token.FileSet, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}
	return file, fset, nil
}

// ExtractFunctions returns all function/method declarations in a Go file.
func ExtractFunctions(file *ast.File, fset *token.FileSet) []FunctionInfo {
	var funcs []FunctionInfo

	ast.Inspect(file, func(n ast.Node) bool {
		switch fn := n.(type) {
		case *ast.FuncDecl:
			info := FunctionInfo{
				Name:      fn.Name.Name,
				StartLine: fset.Position(fn.Pos()).Line,
				EndLine:   fset.Position(fn.End()).Line,
			}
			info.Complexity = computeComplexity(fn.Body)
			info.MaxNesting = computeMaxNesting(fn.Body, 0)
			funcs = append(funcs, info)
		case *ast.FuncLit:
			info := FunctionInfo{
				Name:      "(anonymous)",
				StartLine: fset.Position(fn.Pos()).Line,
				EndLine:   fset.Position(fn.End()).Line,
			}
			info.Complexity = computeComplexity(fn.Body)
			info.MaxNesting = computeMaxNesting(fn.Body, 0)
			funcs = append(funcs, info)
			return false // don't recurse into func literal body again
		}
		return true
	})

	return funcs
}

// computeComplexity calculates the cyclomatic complexity of a function body.
// Base complexity = 1 + number of decision points.
func computeComplexity(body *ast.BlockStmt) int {
	if body == nil {
		return 1
	}
	complexity := 1
	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.IfStmt:
			complexity++
		case *ast.ForStmt:
			complexity++
		case *ast.RangeStmt:
			complexity++
		case *ast.CaseClause:
			// Each case in a switch adds a decision point
			if node.List != nil { // skip default case
				complexity++
			}
		case *ast.CommClause:
			// Each case in a select
			if node.Comm != nil { // skip default case
				complexity++
			}
		case *ast.BinaryExpr:
			if node.Op == token.LAND || node.Op == token.LOR {
				complexity++
			}
		case *ast.FuncLit:
			// Don't recurse into nested function literals
			return false
		}
		return true
	})
	return complexity
}

// computeMaxNesting computes the maximum nesting depth inside a block.
func computeMaxNesting(body *ast.BlockStmt, currentDepth int) int {
	if body == nil {
		return currentDepth
	}
	maxDepth := currentDepth
	for _, stmt := range body.List {
		depth := nestingDepthOfStmt(stmt, currentDepth)
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	return maxDepth
}

func nestingDepthOfStmt(stmt ast.Stmt, depth int) int {
	maxDepth := depth

	switch s := stmt.(type) {
	case *ast.IfStmt:
		d := depth + 1
		if d > maxDepth {
			maxDepth = d
		}
		if s.Body != nil {
			for _, inner := range s.Body.List {
				nd := nestingDepthOfStmt(inner, d)
				if nd > maxDepth {
					maxDepth = nd
				}
			}
		}
		if s.Else != nil {
			nd := nestingDepthOfStmt(s.Else, d)
			if nd > maxDepth {
				maxDepth = nd
			}
		}
	case *ast.ForStmt:
		d := depth + 1
		if d > maxDepth {
			maxDepth = d
		}
		if s.Body != nil {
			for _, inner := range s.Body.List {
				nd := nestingDepthOfStmt(inner, d)
				if nd > maxDepth {
					maxDepth = nd
				}
			}
		}
	case *ast.RangeStmt:
		d := depth + 1
		if d > maxDepth {
			maxDepth = d
		}
		if s.Body != nil {
			for _, inner := range s.Body.List {
				nd := nestingDepthOfStmt(inner, d)
				if nd > maxDepth {
					maxDepth = nd
				}
			}
		}
	case *ast.SwitchStmt:
		d := depth + 1
		if d > maxDepth {
			maxDepth = d
		}
		if s.Body != nil {
			for _, inner := range s.Body.List {
				nd := nestingDepthOfStmt(inner, d)
				if nd > maxDepth {
					maxDepth = nd
				}
			}
		}
	case *ast.TypeSwitchStmt:
		d := depth + 1
		if d > maxDepth {
			maxDepth = d
		}
		if s.Body != nil {
			for _, inner := range s.Body.List {
				nd := nestingDepthOfStmt(inner, d)
				if nd > maxDepth {
					maxDepth = nd
				}
			}
		}
	case *ast.SelectStmt:
		d := depth + 1
		if d > maxDepth {
			maxDepth = d
		}
		if s.Body != nil {
			for _, inner := range s.Body.List {
				nd := nestingDepthOfStmt(inner, d)
				if nd > maxDepth {
					maxDepth = nd
				}
			}
		}
	case *ast.CaseClause:
		for _, inner := range s.Body {
			nd := nestingDepthOfStmt(inner, depth)
			if nd > maxDepth {
				maxDepth = nd
			}
		}
	case *ast.CommClause:
		for _, inner := range s.Body {
			nd := nestingDepthOfStmt(inner, depth)
			if nd > maxDepth {
				maxDepth = nd
			}
		}
	case *ast.BlockStmt:
		for _, inner := range s.List {
			nd := nestingDepthOfStmt(inner, depth)
			if nd > maxDepth {
				maxDepth = nd
			}
		}
	}

	return maxDepth
}
