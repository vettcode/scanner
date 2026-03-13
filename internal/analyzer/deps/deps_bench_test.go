package deps

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// benchDepsSink prevents compiler optimization of discarded results.
var benchDepsSink *ParseResult

func BenchmarkDepParsing(b *testing.B) {
	dir := b.TempDir()

	// npm: package.json with 50 dependencies
	var npmDeps strings.Builder
	for i := 0; i < 50; i++ {
		if i > 0 {
			npmDeps.WriteString(",\n")
		}
		fmt.Fprintf(&npmDeps, `    "pkg-%d": "^%d.0.0"`, i, i%10+1)
	}
	packageJSON := fmt.Sprintf(`{"name":"bench","dependencies":{%s}}`, npmDeps.String())
	require.NoError(b, os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0644))

	// Python: requirements.txt with 30 packages
	var pyDeps strings.Builder
	for i := 0; i < 30; i++ {
		fmt.Fprintf(&pyDeps, "pylib-%d==%d.%d.0\n", i, i%5+1, i%10)
	}
	require.NoError(b, os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(pyDeps.String()), 0644))

	// Go: go.mod with 20 modules
	var goMod strings.Builder
	goMod.WriteString("module bench\n\ngo 1.21\n\nrequire (\n")
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&goMod, "\tgithub.com/example/mod%d v1.%d.0\n", i, i)
	}
	goMod.WriteString(")\n")
	require.NoError(b, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod.String()), 0644))

	// PHP: composer.json with 25 packages
	var phpDeps strings.Builder
	for i := 0; i < 25; i++ {
		if i > 0 {
			phpDeps.WriteString(",\n")
		}
		fmt.Fprintf(&phpDeps, `    "vendor/pkg-%d": "^%d.0"`, i, i%8+1)
	}
	composerJSON := fmt.Sprintf(`{"require":{%s}}`, phpDeps.String())
	require.NoError(b, os.WriteFile(filepath.Join(dir, "composer.json"), []byte(composerJSON), 0644))

	// Ruby: Gemfile with 20 gems
	var gemfile strings.Builder
	gemfile.WriteString("source 'https://rubygems.org'\n\n")
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&gemfile, "gem 'mygem-%d', '~> %d.0'\n", i, i%6+1)
	}
	require.NoError(b, os.WriteFile(filepath.Join(dir, "Gemfile"), []byte(gemfile.String()), 0644))

	// Java: pom.xml with 30 dependencies
	var pomDeps strings.Builder
	for i := 0; i < 30; i++ {
		fmt.Fprintf(&pomDeps, "    <dependency>\n      <groupId>com.example</groupId>\n      <artifactId>lib-%d</artifactId>\n      <version>%d.0.0</version>\n    </dependency>\n", i, i%10+1)
	}
	pomXML := fmt.Sprintf(`<?xml version="1.0"?>
<project>
  <dependencies>
%s  </dependencies>
</project>`, pomDeps.String())
	require.NoError(b, os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pomXML), 0644))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchDepsSink = ParseDependencies(dir)
	}
}
