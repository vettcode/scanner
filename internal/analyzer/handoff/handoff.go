package handoff

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/vettcode/scanner/internal/language"
	"github.com/vettcode/scanner/internal/walker"
)

// Result holds the handoff readiness analysis results.
type Result struct {
	EstTestCoveragePct float64
	DocDensity         string // "high", "medium", "low"
	EnvVarCount        int
	HasReadme          bool
	HasContributing    bool
	HasEnvTemplate     bool
	HasSetupScript     bool
	HasTestConfig      bool
	HasCoverageConfig  bool
}

// Analyze computes handoff readiness metrics.
func Analyze(root string, walkResult *walker.WalkResult) *Result {
	r := &Result{}

	r.EstTestCoveragePct = computeTestCoverage(walkResult)
	r.DocDensity = computeDocDensity(root, walkResult)
	r.EnvVarCount = countEnvVars(root)

	// Boolean flags
	r.HasReadme = fileExists(root, "README.md") || fileExists(root, "README") ||
		fileExists(root, "readme.md") || fileExists(root, "README.rst") ||
		fileExists(root, "README.txt")
	r.HasContributing = fileExists(root, "CONTRIBUTING.md") || fileExists(root, "CONTRIBUTING")
	r.HasEnvTemplate = fileExists(root, ".env.example") || fileExists(root, ".env.template") ||
		fileExists(root, ".env.sample")
	r.HasSetupScript = fileExists(root, "setup.sh") || fileExists(root, "setup.py") ||
		fileExists(root, "Makefile") || fileExists(root, "docker-compose.yml") ||
		fileExists(root, "docker-compose.yaml")

	// Test framework config detection
	r.HasTestConfig = detectTestConfig(root)
	r.HasCoverageConfig = detectCoverageConfig(root)

	return r
}

// computeTestCoverage estimates test coverage using a test-effectiveness
// multiplier for Tier 1 languages only. Each line of test code typically
// exercises ~4 lines of source (through setup, assertions, mocking), so
// the raw testLOC/sourceLOC ratio is multiplied by 4 to approximate actual
// coverage. Tier 2 files (Markdown, YAML, etc.) are excluded because they
// have no test conventions and would dilute the estimate.
func computeTestCoverage(wr *walker.WalkResult) float64 {
	if wr == nil || len(wr.Files) == 0 {
		return 0
	}

	var testLOC, sourceLOC int
	for _, f := range wr.Files {
		if f.Tier != language.Tier1 {
			continue
		}
		if f.IsTest {
			testLOC += f.LOC
		} else {
			sourceLOC += f.LOC
		}
	}

	if sourceLOC == 0 {
		return 0
	}

	// Test effectiveness multiplier: each test line covers ~4 source lines.
	pct := float64(testLOC) / float64(sourceLOC) * 4.0 * 100.0
	if pct > 100 {
		pct = 100
	}
	return pct
}

// computeDocDensity classifies documentation density.
func computeDocDensity(root string, wr *walker.WalkResult) string {
	score := 0

	// README present
	if fileExists(root, "README.md") || fileExists(root, "README") ||
		fileExists(root, "readme.md") {
		score += 2
	}

	// Count doc files
	docFileCount := 0
	if wr != nil {
		for _, f := range wr.Files {
			if f.Language == "Markdown" {
				docFileCount++
			}
		}
	}
	if docFileCount >= 5 {
		score += 2
	} else if docFileCount >= 2 {
		score += 1
	}

	// Check for docs directory
	if dirExists(root, "docs") || dirExists(root, "doc") || dirExists(root, "documentation") {
		score += 1
	}

	switch {
	case score >= 4:
		return "high"
	case score >= 2:
		return "medium"
	default:
		return "low"
	}
}

// countEnvVars counts environment variables from .env.example or .env.template.
func countEnvVars(root string) int {
	for _, name := range []string{".env.example", ".env.template", ".env.sample"} {
		path := filepath.Join(root, name)
		if count := countEnvVarsInFile(path); count > 0 {
			return count
		}
	}
	return 0
}

func countEnvVarsInFile(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "=") {
			count++
		}
	}
	return count
}

func detectTestConfig(root string) bool {
	patterns := []string{
		"jest.config.js", "jest.config.ts", "jest.config.cjs", "jest.config.mjs",
		"vitest.config.ts", "vitest.config.js",
		"pytest.ini", ".pytest.ini",
		"phpunit.xml", "phpunit.xml.dist",
		".rspec",
	}
	for _, p := range patterns {
		if fileExists(root, p) {
			return true
		}
	}
	return false
}

func detectCoverageConfig(root string) bool {
	patterns := []string{
		".nycrc", ".nycrc.json",
		"istanbul.yml",
		".coveragerc",
	}
	for _, p := range patterns {
		if fileExists(root, p) {
			return true
		}
	}
	return false
}

func fileExists(root, name string) bool {
	_, err := os.Stat(filepath.Join(root, name))
	return err == nil
}

func dirExists(root, name string) bool {
	info, err := os.Stat(filepath.Join(root, name))
	return err == nil && info.IsDir()
}
