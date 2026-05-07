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
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
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
	notify()
	finalize()
	complete()
}
`
	p1 := createFile(t, dir, "a.go", duplicated)
	p2 := createFile(t, dir, "b.go", duplicated)

	files := []walker.FileInfo{
		{Path: p1, LOC: 12},
		{Path: p2, LOC: 12},
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
	cleanup()
	notify()
	finalize()
	complete()
}
`
	p1 := createFile(t, dir, "a.go", code)
	p2 := createFile(t, dir, "a_test.go", code)

	files := []walker.FileInfo{
		{Path: p1, LOC: 12},
		{Path: p2, LOC: 12, IsTest: true},
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
	// Two files with identical 120-token streams → should detect duplication
	pattern := make([]string, 120)
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
		{Path: "/a.js", LOC: 24},
		{Path: "/b.js", LOC: 24},
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
	base := make([]string, 120)
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
		{Path: "/x.js", LOC: 24},
		{Path: "/y.js", LOC: 24},
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
	tokens1 := make([]Token, 120)
	tokens2 := make([]Token, 120)
	for i := 0; i < 120; i++ {
		tokens1[i] = Token{Value: "if", Line: i/5 + 1}
		tokens2[i] = Token{Value: "for", Line: i/5 + 1}
	}

	files := []walker.FileInfo{
		{Path: "/a.js", LOC: 24},
		{Path: "/b.js", LOC: 24},
	}
	ts := map[string][]Token{
		"/a.js": tokens1,
		"/b.js": tokens2,
	}
	r := Analyze(files, ts)
	assert.Equal(t, 0, r.DuplicatedLOC)
}

func TestTokenDuplication_BlockFiltering(t *testing.T) {
	// Duplicate tokens that span only 3 lines (< minBlockLines=10)
	// should be filtered out
	tokens := make([]Token, 120)
	for i := range tokens {
		tokens[i] = Token{Value: "$ID", Line: (i / 40) + 1} // 3 lines total
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
	assert.Equal(t, 0, r.DuplicateBlocks, "blocks spanning < 10 lines should be filtered")
}

func TestMixedTokenAndLineFiles(t *testing.T) {
	dir := t.TempDir()

	// Tier 2 file (no tokens) with duplication — needs 10+ non-blank lines
	lineContent := `package main
func handler() {
	data := fetchData()
	result := transform(data)
	validate(result)
	save(result)
	log("done")
	cleanup()
	notify()
	finalize()
	complete()
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
		{Path: p1, LOC: 12},
		{Path: p2, LOC: 12},
		{Path: "/tier1.js", LOC: 12},
	}
	ts := map[string][]Token{
		"/tier1.js": uniqueTokens,
	}
	r := Analyze(files, ts)
	// Line-based should find duplication in a.go/b.go
	assert.Greater(t, r.DuplicatedLOC, 0)
	// TotalLOC should include all 3 files
	assert.Equal(t, 36, r.TotalLOC)
}

func TestCountBlocksAndLOC(t *testing.T) {
	// Two contiguous blocks: lines 1-12 and lines 20-31
	lines := map[int]map[int]bool{
		0: {
			1: true, 2: true, 3: true, 4: true, 5: true, 6: true,
			7: true, 8: true, 9: true, 10: true, 11: true, 12: true,
			20: true, 21: true, 22: true, 23: true, 24: true, 25: true,
			26: true, 27: true, 28: true, 29: true, 30: true, 31: true,
		},
	}
	loc, blocks := countBlocksAndLOC(lines)
	assert.Equal(t, 24, loc)
	assert.Equal(t, 2, blocks)
}

func TestCountBlocksAndLOC_FilterSmall(t *testing.T) {
	// One block of 5 lines (< minBlockLines=10) should be filtered
	lines := map[int]map[int]bool{
		0: {1: true, 2: true, 3: true, 4: true, 5: true},
	}
	loc, blocks := countBlocksAndLOC(lines)
	assert.Equal(t, 0, loc)
	assert.Equal(t, 0, blocks)
}

// --- Partial duplication with known percentage ---

func TestAnalyze_PartialDuplication(t *testing.T) {
	dir := t.TempDir()

	// 12 lines of duplicated code + 12 lines of unique code in each file
	common := `	data := fetchData()
	result := transform(data)
	validate(result)
	save(result)
	log("complete")
	cleanup()
	notify()
	finalize()
	archive()
	compress()
	upload()
	done()
`
	p1 := createFile(t, dir, "a.go", "package main\nfunc a() {\n"+common+`	uniqueA1()
	uniqueA2()
	uniqueA3()
	uniqueA4()
	uniqueA5()
	uniqueA6()
	uniqueA7()
	uniqueA8()
	uniqueA9()
	uniqueA10()
	uniqueA11()
	uniqueA12()
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
	uniqueB9()
	uniqueB10()
	uniqueB11()
	uniqueB12()
}
`)

	files := []walker.FileInfo{
		{Path: p1, LOC: 27},
		{Path: p2, LOC: 27},
	}
	r := Analyze(files)
	// Should detect the common 12-line block
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

	// Same 12-line block appears in all 3 files
	common := `	data := fetchData()
	result := transform(data)
	validate(result)
	save(result)
	log("done")
	cleanup()
	notify()
	finalize()
	archive()
	compress()
	upload()
	done()
`
	for _, name := range []string{"x.go", "y.go", "z.go"} {
		createFile(t, dir, name, "package main\nfunc f() {\n"+common+"}\n")
	}

	files := []walker.FileInfo{
		{Path: filepath.Join(dir, "x.go"), LOC: 15},
		{Path: filepath.Join(dir, "y.go"), LOC: 15},
		{Path: filepath.Join(dir, "z.go"), LOC: 15},
	}
	r := Analyze(files)
	assert.Greater(t, r.DuplicationPct, 0.0, "duplication across 3 files")
	assert.Greater(t, r.DuplicatedLOC, 0)
}

// --- Sampling tests (SC-083) ---

func TestAnalyze_SamplingKicksInAbove300K(t *testing.T) {
	// Below threshold: no sampling (all files processed)
	files := make([]walker.FileInfo, 10)
	dir := t.TempDir()
	for i := range files {
		p := createFile(t, dir, filepath.Join("sub", "file"+string(rune('a'+i))+".go"),
			"package main\nfunc f() {}\n")
		files[i] = walker.FileInfo{Path: p, LOC: 29000} // 10 * 29K = 290K < 300K
	}
	r := Analyze(files)
	assert.Equal(t, 290000, r.TotalLOC, "total LOC should reflect all files")

	// Above threshold: sampling activates
	filesLarge := make([]walker.FileInfo, 20)
	dir2 := t.TempDir()
	for i := range filesLarge {
		p := createFile(t, dir2, filepath.Join("sub", "file"+string(rune('a'+i))+".go"),
			"package main\nfunc f() {}\n")
		filesLarge[i] = walker.FileInfo{Path: p, LOC: 20000} // 20 * 20K = 400K > 300K
	}
	r2 := Analyze(filesLarge)
	assert.Equal(t, 400000, r2.TotalLOC, "total LOC should reflect all files even with sampling")
}

func TestAnalyze_SamplingPreservesTotalLOC(t *testing.T) {
	// Sampling should not affect TotalLOC — it always reflects all files
	dir := t.TempDir()
	files := make([]walker.FileInfo, 100)
	for i := range files {
		p := createFile(t, dir, filepath.Join("d", string(rune(i/26+'a'))+string(rune(i%26+'a'))+".go"),
			"package main\n")
		files[i] = walker.FileInfo{Path: p, LOC: 5000} // 100 * 5K = 500K
	}
	r := Analyze(files)
	assert.Equal(t, 500000, r.TotalLOC)
}
