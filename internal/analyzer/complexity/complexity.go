package complexity

import (
	"context"
	"os"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	tsTypescript "github.com/smacker/go-tree-sitter/typescript/typescript"
)

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

// langConfig holds language-specific tree-sitter configuration.
type langConfig struct {
	language          *sitter.Language
	functionNodeTypes []string
	decisionNodes     map[string]bool
	booleanOps        map[string]bool
	nestingNodes      map[string]bool
}

var configs = map[string]*langConfig{
	"JavaScript": {
		language: javascript.GetLanguage(),
		functionNodeTypes: []string{
			"function_declaration", "method_definition",
			"arrow_function", "function_expression", "function",
		},
		decisionNodes: map[string]bool{
			"if_statement": true, "for_statement": true,
			"for_in_statement": true, "while_statement": true,
			"do_statement": true, "switch_case": true,
			"catch_clause": true, "ternary_expression": true,
		},
		booleanOps: map[string]bool{
			"&&": true, "||": true, "??": true,
		},
		nestingNodes: map[string]bool{
			"if_statement": true, "for_statement": true,
			"for_in_statement": true, "while_statement": true,
			"do_statement": true, "switch_statement": true,
		},
	},
	"TypeScript": {
		language: tsTypescript.GetLanguage(),
		functionNodeTypes: []string{
			"function_declaration", "method_definition",
			"arrow_function", "function_expression", "function",
		},
		decisionNodes: map[string]bool{
			"if_statement": true, "for_statement": true,
			"for_in_statement": true, "while_statement": true,
			"do_statement": true, "switch_case": true,
			"catch_clause": true, "ternary_expression": true,
		},
		booleanOps: map[string]bool{
			"&&": true, "||": true, "??": true,
		},
		nestingNodes: map[string]bool{
			"if_statement": true, "for_statement": true,
			"for_in_statement": true, "while_statement": true,
			"do_statement": true, "switch_statement": true,
		},
	},
	"Python": {
		language: python.GetLanguage(),
		functionNodeTypes: []string{
			"function_definition", "lambda",
		},
		decisionNodes: map[string]bool{
			"if_statement": true, "elif_clause": true,
			"for_statement": true, "while_statement": true,
			"except_clause": true, "conditional_expression": true,
			"case_clause": true,
		},
		booleanOps: map[string]bool{
			"and": true, "or": true,
		},
		nestingNodes: map[string]bool{
			"if_statement": true, "for_statement": true,
			"while_statement": true, "try_statement": true,
			"with_statement": true,
		},
	},
	"PHP": {
		language: php.GetLanguage(),
		functionNodeTypes: []string{
			"function_definition", "method_declaration",
			"anonymous_function_creation_expression",
		},
		decisionNodes: map[string]bool{
			"if_statement": true, "for_statement": true,
			"foreach_statement": true, "while_statement": true,
			"do_statement": true, "switch_case": true,
			"catch_clause": true,
		},
		booleanOps: map[string]bool{
			"&&": true, "||": true, "and": true, "or": true,
		},
		nestingNodes: map[string]bool{
			"if_statement": true, "for_statement": true,
			"foreach_statement": true, "while_statement": true,
			"do_statement": true, "switch_statement": true,
		},
	},
	"Ruby": {
		language: ruby.GetLanguage(),
		functionNodeTypes: []string{
			"method", "singleton_method", "lambda",
		},
		decisionNodes: map[string]bool{
			"if":    true, "elsif": true,
			"unless":   true, "while": true,
			"until":    true, "for":   true,
			"when":     true, "rescue": true,
			"conditional": true,
		},
		booleanOps: map[string]bool{
			"&&": true, "||": true, "and": true, "or": true,
		},
		nestingNodes: map[string]bool{
			"if":    true, "unless": true,
			"while": true, "until":  true,
			"for":   true, "begin":  true,
			"case":  true,
		},
	},
	"Java": {
		language: java.GetLanguage(),
		functionNodeTypes: []string{
			"method_declaration", "constructor_declaration",
			"lambda_expression",
		},
		decisionNodes: map[string]bool{
			"if_statement": true, "for_statement": true,
			"enhanced_for_statement": true, "while_statement": true,
			"do_statement": true, "switch_block_statement_group": true,
			"catch_clause": true, "ternary_expression": true,
		},
		booleanOps: map[string]bool{
			"&&": true, "||": true,
		},
		nestingNodes: map[string]bool{
			"if_statement": true, "for_statement": true,
			"enhanced_for_statement": true, "while_statement": true,
			"do_statement": true, "switch_expression": true,
		},
	},
}

// AnalyzeFile computes complexity for all functions in a file.
func AnalyzeFile(path, lang string) (*FileResult, error) {
	cfg, ok := configs[lang]
	if !ok {
		return nil, nil // unsupported language
	}

	source, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	parser := sitter.NewParser()
	parser.SetLanguage(cfg.language)

	tree, err := parser.ParseCtx(context.Background(), nil, source)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	result := &FileResult{
		Path:     path,
		Language: lang,
	}

	root := tree.RootNode()
	extractFunctions(root, source, cfg, result)
	collectTokens(root, source, result)

	return result, nil
}

// extractFunctions walks the AST to find function nodes and compute complexity.
func extractFunctions(node *sitter.Node, source []byte, cfg *langConfig, result *FileResult) {
	nodeType := node.Type()

	if isFunctionNode(nodeType, cfg) {
		name := extractFuncName(node, source)
		body := findBodyNode(node)

		complexity := 1 // base complexity
		maxNesting := 0

		if body != nil {
			complexity += countDecisionPoints(body, source, cfg)
			maxNesting = computeNesting(body, cfg, 0)
		}

		result.Functions = append(result.Functions, FunctionResult{
			Name:       name,
			StartLine:  int(node.StartPoint().Row) + 1,
			EndLine:    int(node.EndPoint().Row) + 1,
			Complexity: complexity,
			MaxNesting: maxNesting,
		})
	}

	// Recurse into children (nested functions get their own entry)
	for i := 0; i < int(node.ChildCount()); i++ {
		extractFunctions(node.Child(i), source, cfg, result)
	}
}

func countDecisionPoints(node *sitter.Node, source []byte, cfg *langConfig) int {
	count := 0
	walkNode(node, source, cfg, func(n *sitter.Node) {
		nodeType := n.Type()

		// Check decision nodes
		if cfg.decisionNodes[nodeType] {
			// For switch cases, skip default
			if nodeType == "switch_case" || nodeType == "switch_block_statement_group" {
				// Check if this is a default case
				for i := 0; i < int(n.ChildCount()); i++ {
					child := n.Child(i)
					if child.Type() == "default" {
						return // skip default
					}
				}
			}
			count++
		}

		// Check boolean operators
		if nodeType == "binary_expression" || nodeType == "boolean_operator" ||
			nodeType == "logical_and" || nodeType == "logical_or" {
			op := ""
			for i := 0; i < int(n.ChildCount()); i++ {
				child := n.Child(i)
				if child.IsNamed() {
					continue
				}
				op = child.Content(source)
			}
			if cfg.booleanOps[op] {
				count++
			}
		}
	})
	return count
}

func computeNesting(node *sitter.Node, cfg *langConfig, depth int) int {
	maxDepth := depth
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		childType := child.Type()
		if cfg.nestingNodes[childType] {
			nested := computeNesting(child, cfg, depth+1)
			if nested > maxDepth {
				maxDepth = nested
			}
		} else {
			nested := computeNesting(child, cfg, depth)
			if nested > maxDepth {
				maxDepth = nested
			}
		}
	}
	return maxDepth
}

func walkNode(node *sitter.Node, source []byte, cfg *langConfig, fn func(*sitter.Node)) {
	fn(node)
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		// Don't walk into nested functions
		if !isFunctionNode(child.Type(), cfg) {
			walkNode(child, source, cfg, fn)
		}
	}
}

func isFunctionNode(nodeType string, cfg *langConfig) bool {
	for _, ft := range cfg.functionNodeTypes {
		if nodeType == ft {
			return true
		}
	}
	return false
}

func extractFuncName(node *sitter.Node, source []byte) string {
	// Look for a name child node
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		ct := child.Type()
		if ct == "identifier" || ct == "property_identifier" || ct == "name" {
			return child.Content(source)
		}
	}
	return "(anonymous)"
}

func findBodyNode(node *sitter.Node) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		ct := child.Type()
		if ct == "statement_block" || ct == "block" || ct == "body" ||
			ct == "compound_statement" || ct == "method_body" ||
			ct == "function_body" {
			return child
		}
	}
	// For arrow functions/lambdas, the body might be the expression itself
	return node
}

// --- Token extraction for duplication detection ---

// identifierNodeTypes are tree-sitter node types representing identifiers.
var identifierNodeTypes = map[string]bool{
	"identifier": true, "property_identifier": true,
	"shorthand_property_identifier": true, "shorthand_property_identifier_pattern": true,
	"type_identifier": true, "name": true, "variable_name": true,
	"constant": true,
}

// literalNodeTypes are tree-sitter node types representing literals.
var literalNodeTypes = map[string]bool{
	"number": true, "integer": true, "float": true,
	"integer_literal": true, "float_literal": true,
	"decimal_integer_literal": true, "decimal_floating_point_literal": true,
	"hex_integer_literal": true, "octal_integer_literal": true,
	"string": true, "string_literal": true, "string_content": true,
	"string_fragment": true, "template_string": true,
	"encapsed_string": true, "character_literal": true,
	"true": true, "false": true, "null": true, "nil": true, "none": true,
	"null_literal": true, "boolean": true, "regex": true, "regex_pattern": true,
}

// commentNodeTypes are tree-sitter node types to skip.
var commentNodeTypes = map[string]bool{
	"comment": true, "line_comment": true, "block_comment": true,
}

// collectTokens walks all leaf nodes of the AST to produce a normalized token stream.
func collectTokens(node *sitter.Node, source []byte, result *FileResult) {
	if node.ChildCount() == 0 {
		tok := normalizeNode(node, source)
		if tok != "" {
			result.Tokens = append(result.Tokens, Token{
				Value: tok,
				Line:  int(node.StartPoint().Row) + 1,
			})
		}
		return
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		collectTokens(node.Child(i), source, result)
	}
}

// normalizeNode converts a leaf AST node to a normalized token string.
// Identifiers → "$ID", literals → "$LIT", comments → "" (skip),
// everything else (keywords, operators, punctuation) → verbatim content.
func normalizeNode(node *sitter.Node, source []byte) string {
	nodeType := node.Type()
	if commentNodeTypes[nodeType] {
		return ""
	}
	if identifierNodeTypes[nodeType] {
		return "$ID"
	}
	if literalNodeTypes[nodeType] {
		return "$LIT"
	}
	return node.Content(source)
}
