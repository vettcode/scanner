package duplication

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vettcode/scanner/internal/walker"
)

func generateCodeFile(n int, seed int) string {
	var b strings.Builder
	b.WriteString("package main\n\n")
	for i := 0; i < n/10; i++ {
		fmt.Fprintf(&b, "func fn%d_%d() {\n", seed, i)
		for j := 0; j < 7; j++ {
			fmt.Fprintf(&b, "\tv%d := compute(%d, %d)\n", j, i+seed, j)
		}
		b.WriteString("}\n\n")
	}
	return b.String()
}

// benchDupSink prevents compiler optimization of discarded results.
var benchDupSink *Result

func BenchmarkDuplication1K(b *testing.B) {
	dir := b.TempDir()
	var files []walker.FileInfo
	// Include a common block so duplication matching is exercised
	common := strings.Repeat("\tdata := fetchData()\n\tresult := transform(data)\n\tvalidate(result)\n\tsave(result)\n\tlog(\"done\")\n\tcleanup()\n\tnotify()\n\tfinalize()\n", 2)
	for i := 0; i < 5; i++ {
		var content strings.Builder
		content.WriteString("package main\n\n")
		content.WriteString(generateCodeFile(150, i))
		fmt.Fprintf(&content, "func shared%d() {\n%s}\n", i, common)
		path := filepath.Join(dir, fmt.Sprintf("file%d.go", i))
		require.NoError(b, os.WriteFile(path, []byte(content.String()), 0644))
		files = append(files, walker.FileInfo{Path: path, LOC: 200})
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchDupSink = Analyze(files)
	}
}

func BenchmarkDuplication10K(b *testing.B) {
	dir := b.TempDir()
	var files []walker.FileInfo

	// Create 10 files of ~1000 LOC each, some with duplication
	common := strings.Repeat("\tdata := fetchData()\n\tresult := transform(data)\n\tvalidate(result)\n\tsave(result)\n\tlog(\"done\")\n\tcleanup()\n\tnotify()\n\tfinalize()\n", 10)

	for i := 0; i < 10; i++ {
		var content strings.Builder
		content.WriteString("package main\n\n")
		content.WriteString(generateCodeFile(800, i))
		fmt.Fprintf(&content, "func common%d() {\n%s}\n", i, common)
		path := filepath.Join(dir, fmt.Sprintf("file%d.go", i))
		require.NoError(b, os.WriteFile(path, []byte(content.String()), 0644))
		files = append(files, walker.FileInfo{Path: path, LOC: 1000})
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchDupSink = Analyze(files)
	}
}

func BenchmarkDuplication100K(b *testing.B) {
	dir := b.TempDir()
	var files []walker.FileInfo

	// Create 100 files of ~1000 LOC each, with shared duplicate blocks
	common := strings.Repeat("\tdata := fetchData()\n\tresult := transform(data)\n\tvalidate(result)\n\tsave(result)\n\tlog(\"done\")\n\tcleanup()\n\tnotify()\n\tfinalize()\n", 10)

	for i := 0; i < 100; i++ {
		var content strings.Builder
		content.WriteString("package main\n\n")
		content.WriteString(generateCodeFile(800, i))
		fmt.Fprintf(&content, "func common%d() {\n%s}\n", i, common)
		path := filepath.Join(dir, fmt.Sprintf("file%d.go", i))
		require.NoError(b, os.WriteFile(path, []byte(content.String()), 0644))
		files = append(files, walker.FileInfo{Path: path, LOC: 1000})
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchDupSink = Analyze(files)
	}
}
