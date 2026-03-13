package duplication

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vettcode/scanner/internal/walker"
)

func createFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	os.MkdirAll(filepath.Dir(path), 0755)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestAnalyze_NoDuplication(t *testing.T) {
	dir := t.TempDir()
	p1 := createFile(t, dir, "a.go", `package main
func a() {
	println("unique code here")
	x := 1
	y := 2
	z := x + y
	println(z)
}
`)
	p2 := createFile(t, dir, "b.go", `package main
func b() {
	println("completely different code")
	a := "hello"
	b := "world"
	c := a + b
	println(c)
}
`)
	files := []walker.FileInfo{
		{Path: p1, LOC: 8},
		{Path: p2, LOC: 8},
	}
	r := Analyze(files)
	assert.Equal(t, 0.0, r.DuplicationPct)
}

func TestAnalyze_ExactDuplication(t *testing.T) {
	dir := t.TempDir()
	duplicated := `package main
func process() {
	data := fetchData()
	result := transform(data)
	validate(result)
	save(result)
	log("done")
	cleanup()
}
`
	p1 := createFile(t, dir, "a.go", duplicated)
	p2 := createFile(t, dir, "b.go", duplicated)

	files := []walker.FileInfo{
		{Path: p1, LOC: 9},
		{Path: p2, LOC: 9},
	}
	r := Analyze(files)
	assert.Greater(t, r.DuplicationPct, 0.0)
	assert.Greater(t, r.DuplicatedLOC, 0)
}

func TestAnalyze_SkipsTestFiles(t *testing.T) {
	dir := t.TempDir()
	code := `package main
func process() {
	data := fetchData()
	result := transform(data)
	validate(result)
	save(result)
	log("done")
}
`
	p1 := createFile(t, dir, "a.go", code)
	p2 := createFile(t, dir, "a_test.go", code)

	files := []walker.FileInfo{
		{Path: p1, LOC: 8},
		{Path: p2, LOC: 8, IsTest: true},
	}
	r := Analyze(files)
	assert.Equal(t, 0.0, r.DuplicationPct) // test file skipped
}

func TestAnalyze_Empty(t *testing.T) {
	r := Analyze(nil)
	assert.Equal(t, 0.0, r.DuplicationPct)
	assert.Equal(t, 0, r.TotalLOC)
}

func TestHashWindow(t *testing.T) {
	lines1 := []string{"a", "b", "c"}
	lines2 := []string{"a", "b", "c"}
	lines3 := []string{"x", "y", "z"}

	assert.Equal(t, hashWindow(lines1), hashWindow(lines2))
	assert.NotEqual(t, hashWindow(lines1), hashWindow(lines3))
}

func TestReadNormalizedLines(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "  hello world  \n\n  foo   bar  \nbaz\n")
	lines := readNormalizedLines(path)
	assert.Equal(t, []string{"hello world", "foo bar", "baz"}, lines)
}
