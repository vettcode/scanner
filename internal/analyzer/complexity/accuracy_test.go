package complexity

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vettcode/scanner/testdata"
)

// Accuracy validation tests compare VettCode's complexity results against
// hand-computed values from the fixture files. These serve as cross-checks
// against reference tools (ESLint, radon, gocyclo, phpmetrics, flog, PMD).
// Tolerance: +/- 1 per function per the testing plan.
//
// Scanner values may differ from the hand-computed annotations in fixture
// files because tree-sitter counts case/when/elsif/catch/boolean ops as
// separate decision points.

// buildFuncMap creates a map of function name → complexity from analysis results.
func buildFuncMap(t *testing.T, result *FileResult) map[string]int {
	t.Helper()
	funcMap := make(map[string]int)
	for _, fn := range result.Functions {
		funcMap[fn.Name] = fn.Complexity
	}
	return funcMap
}

// assertComplexity asserts that a named function exists and has expected complexity +/- 1.
func assertComplexity(t *testing.T, funcMap map[string]int, name string, expected int) {
	t.Helper()
	require.Contains(t, funcMap, name, "function %q should be found by analyzer", name)
	assert.InDelta(t, expected, funcMap[name], 1, "%s complexity", name)
}

// TestAccuracy_JavaScript_HealthySaas validates JS complexity against scanner values.
func TestAccuracy_JavaScript_HealthySaas(t *testing.T) {
	root := testdata.FixturePath(testdata.HealthySaas)
	path := filepath.Join(root, "frontend", "src", "services", "payment.ts")

	result, err := AnalyzeFile(path, "JavaScript") // TS uses JS grammar
	require.NoError(t, err)
	require.NotNil(t, result)

	funcMap := buildFuncMap(t, result)
	assertComplexity(t, funcMap, "processPayment", 7)
	assertComplexity(t, funcMap, "validateCard", 7)
	assertComplexity(t, funcMap, "formatReceipt", 2)
}

// TestAccuracy_JavaScript_Dashboard validates TSX component complexity.
func TestAccuracy_JavaScript_Dashboard(t *testing.T) {
	root := testdata.FixturePath(testdata.HealthySaas)
	path := filepath.Join(root, "frontend", "src", "components", "Dashboard.tsx")

	result, err := AnalyzeFile(path, "JavaScript")
	require.NoError(t, err)
	require.NotNil(t, result)

	funcMap := buildFuncMap(t, result)
	assertComplexity(t, funcMap, "Dashboard", 4)
	assertComplexity(t, funcMap, "formatMetric", 3)
}

// TestAccuracy_PHP_NeglectedProject validates PHP complexity.
func TestAccuracy_PHP_NeglectedProject(t *testing.T) {
	root := testdata.FixturePath(testdata.NeglectedProject)
	path := filepath.Join(root, "src", "controllers", "UserController.php")

	result, err := AnalyzeFile(path, "PHP")
	require.NoError(t, err)
	require.NotNil(t, result)

	funcMap := buildFuncMap(t, result)
	// Tree-sitter counts case branches as decision points
	assertComplexity(t, funcMap, "handleRequest", 10)
	assertComplexity(t, funcMap, "processForm", 12)
	assertComplexity(t, funcMap, "validateInput", 12)
}

// TestAccuracy_Ruby_SecurityNightmare validates Ruby complexity.
func TestAccuracy_Ruby_SecurityNightmare(t *testing.T) {
	root := testdata.FixturePath(testdata.SecurityNightmare)
	path := filepath.Join(root, "app", "controllers", "api_controller.rb")

	result, err := AnalyzeFile(path, "Ruby")
	require.NoError(t, err)
	require.NotNil(t, result)

	funcMap := buildFuncMap(t, result)
	// Tree-sitter counts case/when/elsif as decision points
	assertComplexity(t, funcMap, "process_request", 13)
	assertComplexity(t, funcMap, "handle_webhook", 8)
}

// TestAccuracy_Java_Enterprise validates Java complexity.
func TestAccuracy_Java_Enterprise(t *testing.T) {
	root := testdata.FixturePath(testdata.JavaEnterprise)
	path := filepath.Join(root, "api", "src", "main", "java", "com", "example", "controllers", "UserController.java")

	result, err := AnalyzeFile(path, "Java")
	require.NoError(t, err)
	require.NotNil(t, result)

	funcMap := buildFuncMap(t, result)
	// Tree-sitter counts catch, boolean ops as decision points
	assertComplexity(t, funcMap, "createUser", 8)
	assertComplexity(t, funcMap, "getUser", 2)
	assertComplexity(t, funcMap, "updateUser", 9)
	assertComplexity(t, funcMap, "handleBulkOperation", 14)
}

// TestAccuracy_Python_HealthySaas validates Python complexity with per-function values.
func TestAccuracy_Python_HealthySaas(t *testing.T) {
	root := testdata.FixturePath(testdata.HealthySaas)
	path := filepath.Join(root, "backend", "app", "routes", "users.py")

	result, err := AnalyzeFile(path, "Python")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Greater(t, len(result.Functions), 0, "should find Python functions")

	funcMap := buildFuncMap(t, result)
	// Scanner values: tree-sitter counts 'or'/'and' as separate decision points
	// create_user: base + 4 if + 2 or/not_in = 7; get_user: base + 1 if = 2
	// update_user: base + 3 if + 2 and + 1 for = 7; list_users: base + 2 if = 3
	assertComplexity(t, funcMap, "create_user", 7)
	assertComplexity(t, funcMap, "get_user", 2)
	assertComplexity(t, funcMap, "update_user", 7)
	assertComplexity(t, funcMap, "list_users", 3)
}
