package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// generateSyntheticRepo creates a synthetic multi-language repo at the given
// directory with approximately targetLOC lines of code. Files are spread
// across JS, Python, PHP, Ruby, Java, and Go to simulate a real polyglot repo.
func generateSyntheticRepo(b *testing.B, dir string, targetLOC int) {
	b.Helper()

	// Allocate LOC across 6 languages
	locPerLang := targetLOC / 6
	remainder := targetLOC - locPerLang*6

	type langSpec struct {
		ext     string
		genFunc func(int) string
	}
	langs := []langSpec{
		{".js", genJS},
		{".py", genPython},
		{".php", genPHP},
		{".rb", genRuby},
		{".java", genJava},
		{".go", genGo},
	}

	// Split each language into multiple files (~500 LOC each)
	filesPerLang := locPerLang / 500
	if filesPerLang < 1 {
		filesPerLang = 1
	}
	locPerFile := locPerLang / filesPerLang

	for li, lang := range langs {
		subdir := filepath.Join(dir, fmt.Sprintf("pkg%d", li))
		require.NoError(b, os.MkdirAll(subdir, 0755))

		for fi := 0; fi < filesPerLang; fi++ {
			loc := locPerFile
			if li == 0 && fi == 0 {
				loc += remainder // give remainder to first file
			}
			content := lang.genFunc(loc)
			name := fmt.Sprintf("file%d%s", fi, lang.ext)
			require.NoError(b, os.WriteFile(filepath.Join(subdir, name), []byte(content), 0644))
		}
	}

	// Add a go.mod so Go files are parseable
	goMod := fmt.Sprintf("module synthetic\n\ngo 1.23\n")
	require.NoError(b, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644))

	// Add a package.json for dep detection
	pkgJSON := `{"name":"synthetic","version":"1.0.0","dependencies":{"express":"^4.18.0"}}`
	require.NoError(b, os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0644))
}

// --- Generator functions (variable-renamed copies to avoid cross-file duplication) ---

func genJS(n int) string {
	var b strings.Builder
	funcs := n / 20
	if funcs == 0 {
		funcs = 1
	}
	for i := 0; i < funcs; i++ {
		fmt.Fprintf(&b, "function handler%d(req, res) {\n", i)
		b.WriteString("  if (req.method === 'GET') {\n")
		b.WriteString("    for (let i = 0; i < req.params.length; i++) {\n")
		b.WriteString("      if (req.params[i] !== null && req.params[i] !== undefined) {\n")
		b.WriteString("        res.write(req.params[i]);\n")
		for j := 0; j < 10; j++ {
			fmt.Fprintf(&b, "        let val%d = req.params[i] + %d;\n", j, j)
		}
		b.WriteString("      }\n")
		b.WriteString("    }\n")
		b.WriteString("  } else if (req.method === 'POST') {\n")
		b.WriteString("    return res.status(405).end();\n")
		b.WriteString("  }\n")
		b.WriteString("  return res.end();\n")
		b.WriteString("}\n\n")
	}
	return b.String()
}

func genPython(n int) string {
	var b strings.Builder
	funcs := n / 20
	if funcs == 0 {
		funcs = 1
	}
	for i := 0; i < funcs; i++ {
		fmt.Fprintf(&b, "def process%d(data, config):\n", i)
		b.WriteString("    if data is not None:\n")
		b.WriteString("        for item in data:\n")
		b.WriteString("            if item > 0 and item < 1000:\n")
		b.WriteString("                print(item)\n")
		for j := 0; j < 10; j++ {
			fmt.Fprintf(&b, "                r%d = item * %d\n", j, j+1)
		}
		b.WriteString("    elif config.get('strict'):\n")
		b.WriteString("        raise ValueError('no data')\n")
		b.WriteString("    return data\n\n")
	}
	return b.String()
}

func genPHP(n int) string {
	var b strings.Builder
	b.WriteString("<?php\n")
	funcs := n / 20
	if funcs == 0 {
		funcs = 1
	}
	for i := 0; i < funcs; i++ {
		fmt.Fprintf(&b, "function handle%d($input, $opts) {\n", i)
		b.WriteString("  if ($input !== null) {\n")
		b.WriteString("    foreach ($input as $val) {\n")
		b.WriteString("      if ($val > 0 && $val < 500) {\n")
		b.WriteString("        echo $val;\n")
		for j := 0; j < 10; j++ {
			fmt.Fprintf(&b, "        $r%d = $val * %d;\n", j, j+1)
		}
		b.WriteString("      }\n")
		b.WriteString("    }\n")
		b.WriteString("  } elseif ($opts['strict']) {\n")
		b.WriteString("    throw new Exception('empty');\n")
		b.WriteString("  }\n")
		b.WriteString("  return $input;\n")
		b.WriteString("}\n\n")
	}
	return b.String()
}

func genRuby(n int) string {
	var b strings.Builder
	funcs := n / 20
	if funcs == 0 {
		funcs = 1
	}
	for i := 0; i < funcs; i++ {
		fmt.Fprintf(&b, "def compute%d(arr, opts)\n", i)
		b.WriteString("  if arr.nil?\n")
		b.WriteString("    raise 'nil input' if opts[:strict]\n")
		b.WriteString("    return []\n")
		b.WriteString("  end\n")
		b.WriteString("  arr.each do |el|\n")
		b.WriteString("    if el > 0 && el < 500\n")
		b.WriteString("      puts el\n")
		for j := 0; j < 8; j++ {
			fmt.Fprintf(&b, "      r%d = el * %d\n", j, j+1)
		}
		b.WriteString("    end\n")
		b.WriteString("  end\n")
		b.WriteString("  arr\n")
		b.WriteString("end\n\n")
	}
	return b.String()
}

func genJava(n int) string {
	var b strings.Builder
	b.WriteString("public class Synth {\n")
	funcs := n / 20
	if funcs == 0 {
		funcs = 1
	}
	for i := 0; i < funcs; i++ {
		fmt.Fprintf(&b, "  public int calc%d(int[] data, boolean strict) {\n", i)
		b.WriteString("    int total = 0;\n")
		b.WriteString("    for (int d : data) {\n")
		b.WriteString("      if (d > 0 && d < 1000) {\n")
		b.WriteString("        total += d;\n")
		for j := 0; j < 10; j++ {
			fmt.Fprintf(&b, "        int t%d = d + %d;\n", j, j)
		}
		b.WriteString("      }\n")
		b.WriteString("    }\n")
		b.WriteString("    if (strict && total == 0) { throw new RuntimeException(\"empty\"); }\n")
		b.WriteString("    return total;\n")
		b.WriteString("  }\n\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func genGo(n int) string {
	var b strings.Builder
	b.WriteString("package synth\n\n")
	funcs := n / 20
	if funcs == 0 {
		funcs = 1
	}
	for i := 0; i < funcs; i++ {
		fmt.Fprintf(&b, "func Transform%d(data []int, strict bool) int {\n", i)
		b.WriteString("\tsum := 0\n")
		b.WriteString("\tfor _, d := range data {\n")
		b.WriteString("\t\tif d > 0 && d < 1000 {\n")
		b.WriteString("\t\t\tsum += d\n")
		for j := 0; j < 10; j++ {
			fmt.Fprintf(&b, "\t\t\tv%d := d + %d\n", j, j)
			fmt.Fprintf(&b, "\t\t\t_ = v%d\n", j)
		}
		b.WriteString("\t\t}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tif strict && sum == 0 {\n")
		b.WriteString("\t\tpanic(\"empty\")\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn sum\n")
		b.WriteString("}\n\n")
	}
	return b.String()
}

// --- Full scan benchmarks ---

func BenchmarkFullScan30K(b *testing.B) {
	dir := b.TempDir()
	generateSyntheticRepo(b, dir, 30000)
	tmpOut := filepath.Join(b.TempDir(), "bench.json")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resetScanFlags()
		_, err := execBench(b, "scan", dir, "--offline", "--format", "json", "-q", "-o", tmpOut)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFullScan100K(b *testing.B) {
	dir := b.TempDir()
	generateSyntheticRepo(b, dir, 100000)
	tmpOut := filepath.Join(b.TempDir(), "bench.json")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resetScanFlags()
		_, err := execBench(b, "scan", dir, "--offline", "--format", "json", "-q", "-o", tmpOut)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemory100K(b *testing.B) {
	dir := b.TempDir()
	generateSyntheticRepo(b, dir, 100000)
	tmpOut := filepath.Join(b.TempDir(), "bench.json")

	b.ReportAllocs()
	b.ResetTimer()

	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	resetScanFlags()
	_, err := execBench(b, "scan", dir, "--offline", "--format", "json", "-q", "-o", tmpOut)
	if err != nil {
		b.Fatal(err)
	}

	runtime.ReadMemStats(&memAfter)
	peakMB := float64(memAfter.TotalAlloc-memBefore.TotalAlloc) / 1024 / 1024
	b.ReportMetric(peakMB, "MB-alloc")

	// Spec target: < 1 GB
	if peakMB > 1024 {
		b.Errorf("memory usage %.0f MB exceeds 1 GB target", peakMB)
	}
}

// execBench is like execCLI but for benchmarks (uses testing.B).
func execBench(b *testing.B, args ...string) (string, error) {
	b.Helper()
	resetScanFlags()

	buf := new(strings.Builder)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	rootCmd.SetArgs(nil)
	rootCmd.SetOut(nil)
	rootCmd.SetErr(nil)

	resetScanFlags()
	return buf.String(), err
}
