package goast

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vettcode/scanner/testdata"
)

// TestAccuracy_Go_JavaEnterprise validates Go complexity against hand-computed
// values from the fixture. Go complexity uses go/ast, cross-checked against gocyclo.
// Tolerance: +/- 1 per function per testing plan.
func TestAccuracy_Go_JavaEnterprise(t *testing.T) {
	root := testdata.FixturePath(testdata.JavaEnterprise)
	path := filepath.Join(root, "worker", "internal", "processor", "processor.go")

	file, fset, err := ParseFile(path)
	require.NoError(t, err)

	funcs := ExtractFunctions(file, fset)
	require.Greater(t, len(funcs), 0, "should find Go functions")

	funcMap := make(map[string]int)
	for _, fn := range funcs {
		funcMap[fn.Name] = fn.Complexity
	}

	// Fixture annotations: ProcessJob=5, handleResult=4, validateJob=3
	require.Contains(t, funcMap, "ProcessJob", "should find ProcessJob")
	assert.InDelta(t, 5, funcMap["ProcessJob"], 1, "ProcessJob complexity")

	require.Contains(t, funcMap, "handleResult", "should find handleResult")
	assert.InDelta(t, 4, funcMap["handleResult"], 1, "handleResult complexity")

	require.Contains(t, funcMap, "validateJob", "should find validateJob")
	assert.InDelta(t, 3, funcMap["validateJob"], 1, "validateJob complexity")
}
