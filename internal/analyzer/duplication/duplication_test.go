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

// --- Line-hash fallback tests (no token data) ---

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

func TestHashLineWindow(t *testing.T) {
	lines1 := []string{"a", "b", "c"}
	lines2 := []string{"a", "b", "c"}
	lines3 := []string{"x", "y", "z"}

	assert.Equal(t, hashLineWindow(lines1), hashLineWindow(lines2))
	assert.NotEqual(t, hashLineWindow(lines1), hashLineWindow(lines3))
}

func TestReadNormalizedLines(t *testing.T) {
	dir := t.TempDir()
	path := createFile(t, dir, "test.txt", "  hello world  \n\n  foo   bar  \nbaz\n")
	lines := readNormalizedLines(path)
	assert.Equal(t, []string{"hello world", "foo bar", "baz"}, lines)
}

// --- Token-based Rabin-Karp tests ---

// makeTokenStream creates a token stream from a pattern of token values,
// assigning sequential line numbers.
func makeTokenStream(values []string) []Token {
	tokens := make([]Token, len(values))
	line := 1
	for i, v := range values {
		tokens[i] = Token{Value: v, Line: line}
		// Advance line every ~5 tokens to simulate real code
		if (i+1)%5 == 0 {
			line++
		}
	}
	return tokens
}

func TestTokenDuplication_ExactMatch(t *testing.T) {
	// Two files with identical 60-token streams → should detect duplication
	pattern := make([]string, 60)
	for i := range pattern {
		if i%3 == 0 {
			pattern[i] = "if"
		} else if i%3 == 1 {
			pattern[i] = "$ID"
		} else {
			pattern[i] = "$LIT"
		}
	}

	tokens1 := makeTokenStream(pattern)
	tokens2 := makeTokenStream(pattern)

	files := []walker.FileInfo{
		{Path: "/a.js", LOC: 12},
		{Path: "/b.js", LOC: 12},
	}
	ts := map[string][]Token{
		"/a.js": tokens1,
		"/b.js": tokens2,
	}
	r := Analyze(files, ts)
	assert.Greater(t, r.DuplicationPct, 0.0)
	assert.Greater(t, r.DuplicateBlocks, 0)
}

func TestTokenDuplication_RenamedVariables(t *testing.T) {
	// Two files with same structure but different variable names.
	// After normalization (identifiers → $ID), they should match.
	base := make([]string, 60)
	for i := range base {
		switch i % 4 {
		case 0:
			base[i] = "func"
		case 1:
			base[i] = "$ID" // all identifiers normalized
		case 2:
			base[i] = "{"
		case 3:
			base[i] = "}"
		}
	}

	tokens1 := makeTokenStream(base)
	tokens2 := makeTokenStream(base) // same normalized form

	files := []walker.FileInfo{
		{Path: "/x.js", LOC: 12},
		{Path: "/y.js", LOC: 12},
	}
	ts := map[string][]Token{
		"/x.js": tokens1,
		"/y.js": tokens2,
	}
	r := Analyze(files, ts)
	assert.Greater(t, r.DuplicationPct, 0.0, "renamed variables should still detect duplication")
}

func TestTokenDuplication_NoDuplication(t *testing.T) {
	// Two files with completely different token streams
	tokens1 := make([]Token, 60)
	tokens2 := make([]Token, 60)
	for i := 0; i < 60; i++ {
		tokens1[i] = Token{Value: "if", Line: i/5 + 1}
		tokens2[i] = Token{Value: "for", Line: i/5 + 1}
	}

	files := []walker.FileInfo{
		{Path: "/a.js", LOC: 12},
		{Path: "/b.js", LOC: 12},
	}
	ts := map[string][]Token{
		"/a.js": tokens1,
		"/b.js": tokens2,
	}
	r := Analyze(files, ts)
	assert.Equal(t, 0, r.DuplicatedLOC)
}

func TestTokenDuplication_BlockFiltering(t *testing.T) {
	// Duplicate tokens that span only 3 lines (< minBlockLines=6)
	// should be filtered out
	tokens := make([]Token, 60)
	for i := range tokens {
		tokens[i] = Token{Value: "$ID", Line: (i / 20) + 1} // 3 lines total
	}

	files := []walker.FileInfo{
		{Path: "/a.js", LOC: 3},
		{Path: "/b.js", LOC: 3},
	}
	ts := map[string][]Token{
		"/a.js": tokens,
		"/b.js": tokens,
	}
	r := Analyze(files, ts)
	assert.Equal(t, 0, r.DuplicateBlocks, "blocks spanning < 6 lines should be filtered")
}

func TestMixedTokenAndLineFiles(t *testing.T) {
	dir := t.TempDir()

	// Tier 2 file (no tokens) with duplication
	lineContent := `package main
func handler() {
	data := fetchData()
	result := transform(data)
	validate(result)
	save(result)
	log("done")
	cleanup()
}
`
	p1 := createFile(t, dir, "a.go", lineContent)
	p2 := createFile(t, dir, "b.go", lineContent)

	// Tier 1 file with tokens but no duplication
	uniqueTokens := make([]Token, 60)
	for i := range uniqueTokens {
		uniqueTokens[i] = Token{Value: string(rune('A' + i%26)), Line: i/5 + 1}
	}

	files := []walker.FileInfo{
		{Path: p1, LOC: 9},
		{Path: p2, LOC: 9},
		{Path: "/tier1.js", LOC: 12},
	}
	ts := map[string][]Token{
		"/tier1.js": uniqueTokens,
	}
	r := Analyze(files, ts)
	// Line-based should find duplication in a.go/b.go
	assert.Greater(t, r.DuplicatedLOC, 0)
	// TotalLOC should include all 3 files
	assert.Equal(t, 30, r.TotalLOC)
}

func TestCountBlocksAndLOC(t *testing.T) {
	// Two contiguous blocks: lines 1-8 and lines 15-22
	lines := map[int]map[int]bool{
		0: {
			1: true, 2: true, 3: true, 4: true,
			5: true, 6: true, 7: true, 8: true,
			15: true, 16: true, 17: true, 18: true,
			19: true, 20: true, 21: true, 22: true,
		},
	}
	loc, blocks := countBlocksAndLOC(lines)
	assert.Equal(t, 16, loc)
	assert.Equal(t, 2, blocks)
}

func TestCountBlocksAndLOC_FilterSmall(t *testing.T) {
	// One block of 3 lines (< minBlockLines) should be filtered
	lines := map[int]map[int]bool{
		0: {1: true, 2: true, 3: true},
	}
	loc, blocks := countBlocksAndLOC(lines)
	assert.Equal(t, 0, loc)
	assert.Equal(t, 0, blocks)
}

// --- Partial duplication with known percentage ---

func TestAnalyze_PartialDuplication(t *testing.T) {
	dir := t.TempDir()

	// 8 lines of duplicated code + 8 lines of unique code in each file
	// Total = 32 lines, duplicated = 16 (8 in each), expected ~50%
	common := `	data := fetchData()
	result := transform(data)
	validate(result)
	save(result)
	log("complete")
	cleanup()
	notify()
	finalize()
`
	p1 := createFile(t, dir, "a.go", "package main\nfunc a() {\n"+common+`	uniqueA1()
	uniqueA2()
	uniqueA3()
	uniqueA4()
	uniqueA5()
	uniqueA6()
	uniqueA7()
	uniqueA8()
}
`)
	p2 := createFile(t, dir, "b.go", "package main\nfunc b() {\n"+common+`	uniqueB1()
	uniqueB2()
	uniqueB3()
	uniqueB4()
	uniqueB5()
	uniqueB6()
	uniqueB7()
	uniqueB8()
}
`)

	files := []walker.FileInfo{
		{Path: p1, LOC: 19},
		{Path: p2, LOC: 19},
	}
	r := Analyze(files)
	// Should detect the common 8-line block
	assert.Greater(t, r.DuplicationPct, 0.0, "should detect partial duplication")
	assert.Greater(t, r.DuplicateBlocks, 0)
}

func TestAnalyze_SingleFile_NoDuplication(t *testing.T) {
	dir := t.TempDir()
	p := createFile(t, dir, "only.go", `package main
func only() {
	a := 1
	b := 2
	c := a + b
	println(c)
}
`)
	files := []walker.FileInfo{{Path: p, LOC: 7}}
	r := Analyze(files)
	assert.Equal(t, 0.0, r.DuplicationPct, "single file should not duplicate with itself")
}

func TestAnalyze_ThreeFilesWithDuplication(t *testing.T) {
	dir := t.TempDir()

	// Same 8-line block appears in all 3 files
	common := `	data := fetchData()
	result := transform(data)
	validate(result)
	save(result)
	log("done")
	cleanup()
	notify()
	finalize()
`
	for _, name := range []string{"x.go", "y.go", "z.go"} {
		createFile(t, dir, name, "package main\nfunc f() {\n"+common+"}\n")
	}

	files := []walker.FileInfo{
		{Path: filepath.Join(dir, "x.go"), LOC: 11},
		{Path: filepath.Join(dir, "y.go"), LOC: 11},
		{Path: filepath.Join(dir, "z.go"), LOC: 11},
	}
	r := Analyze(files)
	assert.Greater(t, r.DuplicationPct, 0.0, "duplication across 3 files")
	assert.Greater(t, r.DuplicatedLOC, 0)
}
