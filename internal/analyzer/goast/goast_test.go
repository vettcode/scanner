package goast

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeGoFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
	return path
}

func TestParseFile(t *testing.T) {
	dir := t.TempDir()
	path := writeGoFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")

	file, fset, err := ParseFile(path)
	require.NoError(t, err)
	assert.NotNil(t, file)
	assert.NotNil(t, fset)
}

func TestParseFile_Invalid(t *testing.T) {
	dir := t.TempDir()
	path := writeGoFile(t, dir, "bad.go", "not valid go code")

	_, _, err := ParseFile(path)
	assert.Error(t, err)
}

func TestExtractFunctions_Simple(t *testing.T) {
	dir := t.TempDir()
	src := `package main

func main() {
	println("hello")
}

func add(a, b int) int {
	return a + b
}
`
	path := writeGoFile(t, dir, "main.go", src)
	file, fset, err := ParseFile(path)
	require.NoError(t, err)

	funcs := ExtractFunctions(file, fset)
	assert.Len(t, funcs, 2)
	assert.Equal(t, "main", funcs[0].Name)
	assert.Equal(t, 1, funcs[0].Complexity) // no branches
	assert.Equal(t, "add", funcs[1].Name)
	assert.Equal(t, 1, funcs[1].Complexity) // no branches
}

func TestComplexity_IfElse(t *testing.T) {
	dir := t.TempDir()
	src := `package main

func check(x int) string {
	if x > 10 {
		return "big"
	} else if x > 5 {
		return "medium"
	} else {
		return "small"
	}
}
`
	path := writeGoFile(t, dir, "check.go", src)
	file, fset, err := ParseFile(path)
	require.NoError(t, err)

	funcs := ExtractFunctions(file, fset)
	require.Len(t, funcs, 1)
	// 1 (base) + 1 (if) + 1 (else if) = 3
	assert.Equal(t, 3, funcs[0].Complexity)
}

func TestComplexity_ForAndBooleanOps(t *testing.T) {
	dir := t.TempDir()
	src := `package main

func process(items []int) int {
	sum := 0
	for _, v := range items {
		if v > 0 && v < 100 {
			sum += v
		}
	}
	return sum
}
`
	path := writeGoFile(t, dir, "process.go", src)
	file, fset, err := ParseFile(path)
	require.NoError(t, err)

	funcs := ExtractFunctions(file, fset)
	require.Len(t, funcs, 1)
	// 1 (base) + 1 (range) + 1 (if) + 1 (&&) = 4
	assert.Equal(t, 4, funcs[0].Complexity)
}

func TestComplexity_Switch(t *testing.T) {
	dir := t.TempDir()
	src := `package main

func classify(x int) string {
	switch {
	case x > 100:
		return "huge"
	case x > 10:
		return "big"
	default:
		return "small"
	}
}
`
	path := writeGoFile(t, dir, "switch.go", src)
	file, fset, err := ParseFile(path)
	require.NoError(t, err)

	funcs := ExtractFunctions(file, fset)
	require.Len(t, funcs, 1)
	// 1 (base) + 2 (cases, default excluded) = 3
	assert.Equal(t, 3, funcs[0].Complexity)
}

func TestNesting_Simple(t *testing.T) {
	dir := t.TempDir()
	src := `package main

func deep() {
	if true {
		for i := 0; i < 10; i++ {
			if i > 5 {
				println(i)
			}
		}
	}
}
`
	path := writeGoFile(t, dir, "deep.go", src)
	file, fset, err := ParseFile(path)
	require.NoError(t, err)

	funcs := ExtractFunctions(file, fset)
	require.Len(t, funcs, 1)
	assert.Equal(t, 3, funcs[0].MaxNesting) // if > for > if
}

func TestNesting_Flat(t *testing.T) {
	dir := t.TempDir()
	src := `package main

func flat() {
	println("a")
	println("b")
}
`
	path := writeGoFile(t, dir, "flat.go", src)
	file, fset, err := ParseFile(path)
	require.NoError(t, err)

	funcs := ExtractFunctions(file, fset)
	require.Len(t, funcs, 1)
	assert.Equal(t, 0, funcs[0].MaxNesting)
}

func TestComplexity_EmptyFunction(t *testing.T) {
	dir := t.TempDir()
	src := `package main

func empty() {}
`
	path := writeGoFile(t, dir, "empty.go", src)
	file, fset, err := ParseFile(path)
	require.NoError(t, err)

	funcs := ExtractFunctions(file, fset)
	require.Len(t, funcs, 1)
	assert.Equal(t, "empty", funcs[0].Name)
	assert.Equal(t, 1, funcs[0].Complexity) // base complexity
	assert.Equal(t, 0, funcs[0].MaxNesting)
}

func TestComplexity_FuncLiteral(t *testing.T) {
	dir := t.TempDir()
	src := `package main

func withCallback() {
	fn := func(x int) bool {
		if x > 0 {
			return true
		}
		return false
	}
	_ = fn
}
`
	path := writeGoFile(t, dir, "funclit.go", src)
	file, fset, err := ParseFile(path)
	require.NoError(t, err)

	funcs := ExtractFunctions(file, fset)
	// Should find the outer func and the func literal
	require.Len(t, funcs, 2)

	// withCallback: base 1, no branches in outer body
	assert.Equal(t, "withCallback", funcs[0].Name)
	assert.Equal(t, 1, funcs[0].Complexity)

	// anonymous func: base 1 + 1 (if) = 2
	assert.Equal(t, "(anonymous)", funcs[1].Name)
	assert.Equal(t, 2, funcs[1].Complexity)
}
