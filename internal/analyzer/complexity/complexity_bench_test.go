package complexity

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// generateJS generates n LOC of JavaScript with moderate complexity.
func generateJS(n int) string {
	var b strings.Builder
	funcs := n / 20 // ~20 LOC per function
	if funcs == 0 {
		funcs = 1
	}
	for i := 0; i < funcs; i++ {
		fmt.Fprintf(&b, "function fn%d(x, items) {\n", i)
		b.WriteString("  if (x > 0) {\n")
		b.WriteString("    for (let i = 0; i < items.length; i++) {\n")
		b.WriteString("      if (items[i] > 0 && items[i] < 100) {\n")
		b.WriteString("        console.log(items[i]);\n")
		for j := 0; j < 10; j++ {
			fmt.Fprintf(&b, "        let v%d = items[i] + %d;\n", j, j)
		}
		b.WriteString("      }\n")
		b.WriteString("    }\n")
		b.WriteString("  } else if (x < -10) {\n")
		b.WriteString("    return null;\n")
		b.WriteString("  }\n")
		b.WriteString("  return x;\n")
		b.WriteString("}\n\n")
	}
	return b.String()
}

// generatePHP generates n LOC of PHP with moderate complexity.
func generatePHP(n int) string {
	var b strings.Builder
	b.WriteString("<?php\n")
	funcs := n / 20
	if funcs == 0 {
		funcs = 1
	}
	for i := 0; i < funcs; i++ {
		fmt.Fprintf(&b, "function fn%d($x, $items) {\n", i)
		b.WriteString("  if ($x > 0) {\n")
		b.WriteString("    foreach ($items as $item) {\n")
		b.WriteString("      if ($item > 0 && $item < 100) {\n")
		b.WriteString("        echo $item;\n")
		for j := 0; j < 10; j++ {
			fmt.Fprintf(&b, "        $v%d = $item + %d;\n", j, j)
		}
		b.WriteString("      }\n")
		b.WriteString("    }\n")
		b.WriteString("  } elseif ($x < -10) {\n")
		b.WriteString("    return null;\n")
		b.WriteString("  }\n")
		b.WriteString("  return $x;\n")
		b.WriteString("}\n\n")
	}
	return b.String()
}

// generateJava generates n LOC of Java with moderate complexity.
func generateJava(n int) string {
	var b strings.Builder
	b.WriteString("class Generated {\n")
	funcs := n / 20
	if funcs == 0 {
		funcs = 1
	}
	for i := 0; i < funcs; i++ {
		fmt.Fprintf(&b, "  public int method%d(int x, int[] items) {\n", i)
		b.WriteString("    int sum = 0;\n")
		b.WriteString("    for (int item : items) {\n")
		b.WriteString("      if (item > 0 && item < 1000) {\n")
		b.WriteString("        sum += item;\n")
		for j := 0; j < 10; j++ {
			fmt.Fprintf(&b, "        int v%d = item + %d;\n", j, j)
		}
		b.WriteString("      }\n")
		b.WriteString("    }\n")
		b.WriteString("    if (sum > x) { return sum; }\n")
		b.WriteString("    return x;\n")
		b.WriteString("  }\n\n")
	}
	b.WriteString("}\n")
	return b.String()
}

// generatePython generates n LOC of Python with moderate complexity.
func generatePython(n int) string {
	var b strings.Builder
	funcs := n / 20
	if funcs == 0 {
		funcs = 1
	}
	for i := 0; i < funcs; i++ {
		fmt.Fprintf(&b, "def fn%d(x, items):\n", i)
		b.WriteString("    if x > 0:\n")
		b.WriteString("        for item in items:\n")
		b.WriteString("            if item > 0 and item < 100:\n")
		b.WriteString("                print(item)\n")
		for j := 0; j < 10; j++ {
			fmt.Fprintf(&b, "                v%d = item + %d\n", j, j)
		}
		b.WriteString("    elif x < -10:\n")
		b.WriteString("        return None\n")
		b.WriteString("    return x\n\n")
	}
	return b.String()
}

// generateRuby generates n LOC of Ruby with moderate complexity.
func generateRuby(n int) string {
	var b strings.Builder
	funcs := n / 20
	if funcs == 0 {
		funcs = 1
	}
	for i := 0; i < funcs; i++ {
		fmt.Fprintf(&b, "def fn%d(x, items)\n", i)
		b.WriteString("  if x > 0\n")
		b.WriteString("    items.each do |item|\n")
		b.WriteString("      if item > 0 && item < 100\n")
		b.WriteString("        puts item\n")
		for j := 0; j < 10; j++ {
			fmt.Fprintf(&b, "        v%d = item + %d\n", j, j)
		}
		b.WriteString("      end\n")
		b.WriteString("    end\n")
		b.WriteString("  elsif x < -10\n")
		b.WriteString("    return nil\n")
		b.WriteString("  end\n")
		b.WriteString("  x\n")
		b.WriteString("end\n\n")
	}
	return b.String()
}

// benchComplexitySink prevents compiler optimization of discarded results.
var benchComplexitySink *FileResult

func BenchmarkComplexityJS1K(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "gen.js")
	require.NoError(b, os.WriteFile(path, []byte(generateJS(1000)), 0644))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchComplexitySink, _ = AnalyzeFile(path, "JavaScript")
	}
}

func BenchmarkComplexityJS10K(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "gen.js")
	require.NoError(b, os.WriteFile(path, []byte(generateJS(10000)), 0644))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchComplexitySink, _ = AnalyzeFile(path, "JavaScript")
	}
}

func BenchmarkComplexityPHP10K(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "gen.php")
	require.NoError(b, os.WriteFile(path, []byte(generatePHP(10000)), 0644))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchComplexitySink, _ = AnalyzeFile(path, "PHP")
	}
}

func BenchmarkComplexityJava10K(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "Gen.java")
	require.NoError(b, os.WriteFile(path, []byte(generateJava(10000)), 0644))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchComplexitySink, _ = AnalyzeFile(path, "Java")
	}
}

func BenchmarkComplexity100K(b *testing.B) {
	dir := b.TempDir()

	// Generate ~100K LOC across 5 tree-sitter languages (Go uses goast, excluded)
	type langFile struct {
		name    string
		lang    string
		genFunc func(int) string
	}
	langs := []langFile{
		{"gen.js", "JavaScript", generateJS},
		{"Gen.java", "Java", generateJava},
		{"gen.php", "PHP", generatePHP},
		{"gen.py", "Python", generatePython},
		{"gen.rb", "Ruby", generateRuby},
	}

	var paths []struct {
		path string
		lang string
	}
	for _, lf := range langs {
		// ~20K LOC per language = ~100K total
		path := filepath.Join(dir, lf.name)
		require.NoError(b, os.WriteFile(path, []byte(lf.genFunc(20000)), 0644))
		paths = append(paths, struct {
			path string
			lang string
		}{path, lf.lang})
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range paths {
			benchComplexitySink, _ = AnalyzeFile(p.path, p.lang)
		}
	}
}
