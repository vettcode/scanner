package secrets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vettcode/scanner/internal/walker"
)

// generateCodeLines generates n lines of realistic code with occasional secret-like patterns.
// Every ~500 lines, a planted secret is inserted to exercise the match path.
func generateCodeLines(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		switch {
		case i%500 == 100:
			fmt.Fprintf(&b, "\tapiKey := \"AKIAIOSFODNN7BENCH%02d\"\n", i/500)
		case i%500 == 300:
			fmt.Fprintf(&b, "\ttoken := \"ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZbench%04d\"\n", i/500)
		case i%50 == 0:
			fmt.Fprintf(&b, "func handler%d(w http.ResponseWriter, r *http.Request) {\n", i/50)
		case i%50 == 49:
			b.WriteString("}\n")
		case i%50 == 10:
			b.WriteString("\tlogger.Info(\"processing request\")\n")
		case i%50 == 20:
			fmt.Fprintf(&b, "\tresult := process(data%d)\n", i)
		default:
			fmt.Fprintf(&b, "\tv%d := compute(input, %d)\n", i, i*17)
		}
	}
	return b.String()
}

// benchSecretsSink prevents compiler optimization of discarded results.
var benchSecretsSink *Result

func BenchmarkSecrets1K(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "code.go")
	require.NoError(b, os.WriteFile(path, []byte(generateCodeLines(1000)), 0644))
	files := []walker.FileInfo{{Path: path, RelPath: "code.go"}}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSecretsSink = Scan(files)
	}
}

func BenchmarkSecrets10K(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "code.go")
	require.NoError(b, os.WriteFile(path, []byte(generateCodeLines(10000)), 0644))
	files := []walker.FileInfo{{Path: path, RelPath: "code.go"}}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSecretsSink = Scan(files)
	}
}

func BenchmarkSecrets100K(b *testing.B) {
	dir := b.TempDir()
	// Spread 100K lines across 10 files
	var files []walker.FileInfo
	for f := 0; f < 10; f++ {
		path := filepath.Join(dir, fmt.Sprintf("code%d.go", f))
		require.NoError(b, os.WriteFile(path, []byte(generateCodeLines(10000)), 0644))
		files = append(files, walker.FileInfo{Path: path, RelPath: fmt.Sprintf("code%d.go", f)})
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSecretsSink = Scan(files)
	}
}
