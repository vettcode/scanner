package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vettcode/scanner/testdata"
)

// E2E tests for CLI commands.
//
// NOTE: These tests share the package-level rootCmd and MUST NOT use
// t.Parallel() — cobra's global state is not safe for concurrent access.

// execCLI is a helper that sets up a fresh output buffer and runs the root command
// with the given args. It resets args, output, and flag values afterward to avoid
// state leakage between tests.
func execCLI(t *testing.T, args ...string) (output string, err error) {
	t.Helper()

	// Reset scan command flags to defaults before each execution
	resetScanFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	err = rootCmd.Execute()
	// Reset state to prevent leakage between tests
	rootCmd.SetArgs(nil)
	rootCmd.SetOut(nil)
	rootCmd.SetErr(nil)

	// Reset flags again after execution
	resetScanFlags()

	return buf.String(), err
}

// resetScanFlags resets all scan command flags to their default values.
func resetScanFlags() {
	scanCmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

func TestCLI_HelpCommand(t *testing.T) {
	output, err := execCLI(t, "help")
	require.NoError(t, err)
	assert.Contains(t, output, "VettCode")
	assert.Contains(t, output, "scan")
	assert.Contains(t, output, "version")
}

func TestCLI_ScanHelp(t *testing.T) {
	output, err := execCLI(t, "scan", "--help")
	require.NoError(t, err)
	assert.Contains(t, output, "Scan one or more directories")
	assert.Contains(t, output, "--output")
	assert.Contains(t, output, "--offline")
	assert.Contains(t, output, "--no-color")
	assert.Contains(t, output, "--quiet")
	assert.Contains(t, output, "--label")
	assert.Contains(t, output, "--format")
	assert.Contains(t, output, "--verbose")
	assert.Contains(t, output, "--no-git")
	assert.Contains(t, output, "--timeout")
	assert.Contains(t, output, "--ci")
	assert.Contains(t, output, "--ci-threshold")
	assert.Contains(t, output, "--ci-fail-on")
}

func TestCLI_VersionOutput(t *testing.T) {
	output, err := execCLI(t, "version")
	require.NoError(t, err)
	output = strings.TrimSpace(output)
	assert.True(t, len(output) > 0, "version output should not be empty")
	assert.Contains(t, output, "vettcode", "version output should mention vettcode")
	assert.Contains(t, output, "commit:", "version output should include commit")
	assert.Contains(t, output, "built:", "version output should include build time")
	assert.Contains(t, output, runtime.GOOS+"/"+runtime.GOARCH, "version output should include platform")
}

func TestCLI_UnknownCommand(t *testing.T) {
	output, err := execCLI(t, "nonexistent-command")
	assert.Error(t, err, "unknown command should return error")
	assert.Contains(t, output, "nonexistent-command", "error should mention the unknown command")
}

func TestCLI_ScanFlags(t *testing.T) {
	// Verify all expected flags are registered on the scan command.
	// This checks DefValue (initial defaults), which is stable regardless
	// of prior Execute() calls — DefValue is set at registration time.
	flags := scanCmd.Flags()

	outputFlag := flags.Lookup("output")
	require.NotNil(t, outputFlag, "--output flag should exist")
	assert.Equal(t, "./vettcode-scan-result.json", outputFlag.DefValue)

	offlineFlag := flags.Lookup("offline")
	require.NotNil(t, offlineFlag)
	assert.Equal(t, "false", offlineFlag.DefValue)

	noColorFlag := flags.Lookup("no-color")
	require.NotNil(t, noColorFlag)

	quietFlag := flags.Lookup("quiet")
	require.NotNil(t, quietFlag)

	formatFlag := flags.Lookup("format")
	require.NotNil(t, formatFlag)
	assert.Equal(t, "both", formatFlag.DefValue)

	verboseFlag := flags.Lookup("verbose")
	require.NotNil(t, verboseFlag)

	noGitFlag := flags.Lookup("no-git")
	require.NotNil(t, noGitFlag)

	timeoutFlag := flags.Lookup("timeout")
	require.NotNil(t, timeoutFlag)
	assert.Equal(t, "30m0s", timeoutFlag.DefValue)

	labelFlag := flags.Lookup("label")
	require.NotNil(t, labelFlag)
}

func TestCLI_ScanNoArgs(t *testing.T) {
	// With no arguments, scan defaults to current directory.
	// This should succeed (we're in a valid Go project directory).
	tmpOut := filepath.Join(t.TempDir(), "scan.json")
	_, err := execCLI(t, "scan", "--offline", "--format", "json", "-q", "-o", tmpOut)
	assert.NoError(t, err, "scan with no args should not error")
}

// --- E2E tests that exercise the full scan pipeline ---

func TestCLI_ScanNonexistentPath(t *testing.T) {
	_, err := execCLI(t, "scan", "/nonexistent/path/12345", "--offline", "-q")
	assert.Error(t, err, "scan with nonexistent path should return error")
}

func TestCLI_ScanEmptyDir(t *testing.T) {
	emptyDir := t.TempDir()
	tmpOut := filepath.Join(t.TempDir(), "scan.json")
	_, err := execCLI(t, "scan", emptyDir, "--offline", "--format", "json", "-q", "-o", tmpOut)
	// TODO: The spec says `vettcode scan <empty-dir>` should exit code 1 with
	// "no supported languages", but the current implementation exits 0 with 0 LOC.
	// This test documents the current behavior; update when the implementation
	// is changed to match the spec.
	require.NoError(t, err)
	data, err := os.ReadFile(tmpOut)
	require.NoError(t, err)
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &result))
	assert.Equal(t, float64(0), result["total_loc"], "empty dir should have 0 LOC")
}

func TestCLI_ScanQuietMode(t *testing.T) {
	fixture := testdata.FixturePath(testdata.HealthySaas)
	tmpOut := filepath.Join(t.TempDir(), "scan.json")
	out, err := execCLI(t, "scan", fixture, "--offline", "-q", "-o", tmpOut)
	require.NoError(t, err)
	// Quiet mode should produce no terminal output on stdout
	assert.Empty(t, out, "quiet mode should suppress terminal output")
	// But JSON should be written
	_, statErr := os.Stat(tmpOut)
	assert.NoError(t, statErr, "JSON output file should exist")
}

func TestCLI_ScanNoColorMode(t *testing.T) {
	fixture := testdata.FixturePath(testdata.HealthySaas)
	tmpOut := filepath.Join(t.TempDir(), "scan.json")
	out, err := execCLI(t, "scan", fixture, "--offline", "--no-color", "--format", "terminal", "-o", tmpOut)
	require.NoError(t, err)
	// No ANSI escape codes should appear
	assert.NotContains(t, out, "\033[", "no-color mode should have no ANSI codes")
}

func TestCLI_ScanOfflineCached(t *testing.T) {
	fixture := testdata.FixturePath(testdata.HealthySaas)
	tmpOut := filepath.Join(t.TempDir(), "scan.json")
	_, err := execCLI(t, "scan", fixture, "--offline", "--format", "json", "-q", "-o", tmpOut)
	require.NoError(t, err)

	// Read and validate JSON output
	data, err := os.ReadFile(tmpOut)
	require.NoError(t, err)
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &result))

	// Should have basic structure
	assert.Contains(t, result, "version")
	assert.Contains(t, result, "scan_id")
	assert.Contains(t, result, "timestamp")
	assert.Contains(t, result, "integrity")
	assert.Equal(t, "1.0", result["version"])
	assert.False(t, result["integrity"].(map[string]interface{})["cosigned"].(bool),
		"offline scan should not be cosigned")
}

func TestCLI_ScanFixtureHealthySaas(t *testing.T) {
	fixture := testdata.FixturePath(testdata.HealthySaas)
	tmpOut := filepath.Join(t.TempDir(), "scan.json")
	_, err := execCLI(t, "scan", fixture, "--offline", "--format", "json", "-q", "-o", tmpOut)
	require.NoError(t, err)

	data, err := os.ReadFile(tmpOut)
	require.NoError(t, err)
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &result))

	// Should detect multiple languages
	repos := result["repositories"].([]interface{})
	require.Len(t, repos, 1)
	repo := repos[0].(map[string]interface{})
	langs := repo["detected_languages"].([]interface{})
	assert.True(t, len(langs) >= 2, "healthy-saas should have multiple languages, got %v", langs)

	// Should have positive LOC
	totalLOC := result["total_loc"].(float64)
	assert.True(t, totalLOC > 0, "total_loc should be positive")

	// Should have metrics
	metrics := result["metrics"].(map[string]interface{})
	assert.Contains(t, metrics, "maintainability")
	assert.Contains(t, metrics, "security")

	// Maintainability should get a decent score for healthy-saas (>= 60)
	maint := metrics["maintainability"].(map[string]interface{})
	if grade, ok := maint["grade"].(string); ok {
		// Grade should be C or better (A, A-, B+, B, B-, C+, C)
		decentGrades := map[string]bool{"A": true, "A-": true, "B+": true, "B": true, "B-": true, "C+": true, "C": true}
		assert.True(t, decentGrades[grade],
			"healthy-saas maintainability grade should be C or better, got %s", grade)
	}

	// Should have integrity block
	integrity := result["integrity"].(map[string]interface{})
	assert.NotEmpty(t, integrity["scan_checksum"])
	assert.NotEmpty(t, integrity["scanner_signature"])
}

func TestCLI_ScanFixtureNeglectedProject(t *testing.T) {
	fixture := testdata.FixturePath(testdata.NeglectedProject)
	tmpOut := filepath.Join(t.TempDir(), "scan.json")
	_, err := execCLI(t, "scan", fixture, "--offline", "--format", "json", "-q", "-o", tmpOut)
	require.NoError(t, err)

	data, err := os.ReadFile(tmpOut)
	require.NoError(t, err)
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &result))

	// Neglected project should have red flags
	redFlags := result["red_flags"].(map[string]interface{})
	flagCount := redFlags["count"].(float64)
	assert.True(t, flagCount > 0, "neglected project should have red flags")

	// Neglected project should have multiple red flags (at least 2: no_readme + others)
	assert.True(t, flagCount >= 2,
		"neglected project should have multiple red flags, got %v", flagCount)

	// Should have no README red flag
	flags := redFlags["flags"].([]interface{})
	var flagCodes []string
	for _, f := range flags {
		flag := f.(map[string]interface{})
		flagCodes = append(flagCodes, flag["flag"].(string))
	}
	assert.Contains(t, flagCodes, "no_readme", "neglected project should flag missing README")
}

func TestCLI_ScanFixtureSecurityNightmare(t *testing.T) {
	// The secrets scanner skips files under testdata/fixtures/ paths,
	// so we copy the fixture to a temp directory for E2E testing.
	fixture := testdata.FixturePath(testdata.SecurityNightmare)
	tmpDir := t.TempDir()
	scanDir := filepath.Join(tmpDir, "project")
	copyDir(t, fixture, scanDir)

	tmpOut := filepath.Join(t.TempDir(), "scan.json")
	_, err := execCLI(t, "scan", scanDir, "--offline", "--format", "json", "-q", "-o", tmpOut)
	require.NoError(t, err)

	data, err := os.ReadFile(tmpOut)
	require.NoError(t, err)
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &result))

	// Security nightmare should have secrets detected
	metrics := result["metrics"].(map[string]interface{})
	sec := metrics["security"].(map[string]interface{})
	secretsFound := sec["secrets_found"].(float64)
	assert.True(t, secretsFound > 0, "security-nightmare should have secrets")

	// Should have secrets_detected red flag
	redFlags := result["red_flags"].(map[string]interface{})
	flags := redFlags["flags"].([]interface{})
	var flagCodes []string
	for _, f := range flags {
		flag := f.(map[string]interface{})
		flagCodes = append(flagCodes, flag["flag"].(string))
	}
	assert.Contains(t, flagCodes, "secrets_detected", "security-nightmare should flag secrets")
}

func TestCLI_ScanFixtureTier2Only(t *testing.T) {
	fixture := testdata.FixturePath(testdata.Tier2Only)
	tmpOut := filepath.Join(t.TempDir(), "scan.json")
	_, err := execCLI(t, "scan", fixture, "--offline", "--format", "json", "-q", "-o", tmpOut)
	// Tier2-only has IaC files (Terraform, K8s, Docker) but no Tier 1 languages.
	// Should still succeed (exit 0) because there are supported files.
	require.NoError(t, err, "tier2-only fixture should succeed even without Tier 1 languages")

	data, err := os.ReadFile(tmpOut)
	require.NoError(t, err)
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &result))

	// Should have positive LOC from Tier 2 files
	totalLOC := result["total_loc"].(float64)
	assert.True(t, totalLOC > 0, "tier2-only should have positive LOC from IaC files")
}

func TestCLI_ScanFixtureJavaEnterprise(t *testing.T) {
	fixture := testdata.FixturePath(testdata.JavaEnterprise)
	tmpOut := filepath.Join(t.TempDir(), "scan.json")
	_, err := execCLI(t, "scan", fixture, "--offline", "--format", "json", "-q", "-o", tmpOut)
	require.NoError(t, err, "java-enterprise fixture should scan successfully")

	data, err := os.ReadFile(tmpOut)
	require.NoError(t, err)
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &result))

	// Verify multi-language detection (Java + Go)
	repos := result["repositories"].([]interface{})
	require.Len(t, repos, 1)
	repo := repos[0].(map[string]interface{})
	langs := repo["detected_languages"].([]interface{})
	assert.True(t, len(langs) >= 2, "java-enterprise should detect multiple languages, got %v", langs)

	// Verify Java is detected
	var langNames []string
	for _, l := range langs {
		langNames = append(langNames, l.(string))
	}
	assert.Contains(t, langNames, "Java", "should detect Java")
}

// --- Grammar command tests ---

func TestCLI_GrammarHelp(t *testing.T) {
	output, err := execCLI(t, "grammar", "--help")
	require.NoError(t, err)
	assert.Contains(t, output, "tree-sitter grammars")
	assert.Contains(t, output, "install")
	assert.Contains(t, output, "list")
	assert.Contains(t, output, "update")
}

func TestCLI_GrammarList(t *testing.T) {
	output, err := execCLI(t, "grammar", "list")
	require.NoError(t, err)
	assert.Contains(t, output, "LANGUAGE")
	assert.Contains(t, output, "javascript")
	assert.Contains(t, output, "python")
	assert.Contains(t, output, "Grammar version")
}

func TestCLI_GrammarInstallUnknown(t *testing.T) {
	_, err := execCLI(t, "grammar", "install", "brainfuck")
	assert.Error(t, err, "installing unknown language should fail")
}

// --- CI mode tests ---

func TestCLI_CIModePass(t *testing.T) {
	fixture := testdata.FixturePath(testdata.HealthySaas)
	tmpOut := filepath.Join(t.TempDir(), "scan.json")
	out, err := execCLI(t, "scan", fixture, "--offline", "--ci", "--format", "json", "-q", "-o", tmpOut)
	require.NoError(t, err, "CI mode should pass for healthy-saas with default threshold")
	// JSON should still be written
	_, statErr := os.Stat(tmpOut)
	assert.NoError(t, statErr, "JSON output file should exist in CI mode")
	_ = out
}

func TestCLI_CIModeFailGrade(t *testing.T) {
	fixture := testdata.FixturePath(testdata.HealthySaas)
	tmpOut := filepath.Join(t.TempDir(), "scan.json")
	_, err := execCLI(t, "scan", fixture, "--offline", "--ci", "--ci-threshold", "A", "--format", "json", "-q", "-o", tmpOut)
	require.Error(t, err, "CI mode should fail when threshold is A and grade is lower")
	var ciErr *CIGateError
	assert.ErrorAs(t, err, &ciErr, "error should be CIGateError")
	assert.Contains(t, ciErr.Error(), "below threshold A")
	// JSON should still be written even on CI failure
	_, statErr := os.Stat(tmpOut)
	assert.NoError(t, statErr, "JSON output should exist even when CI gate fails")
}

func TestCLI_CIModeFailRedFlags(t *testing.T) {
	// Copy security-nightmare to temp dir (secrets scanner skips testdata/ paths)
	fixture := testdata.FixturePath(testdata.SecurityNightmare)
	tmpDir := t.TempDir()
	scanDir := filepath.Join(tmpDir, "project")
	copyDir(t, fixture, scanDir)

	tmpOut := filepath.Join(t.TempDir(), "scan.json")
	_, err := execCLI(t, "scan", scanDir, "--offline", "--ci", "--ci-threshold", "F", "--format", "json", "-q", "-o", tmpOut)
	require.Error(t, err, "CI mode should fail on critical red flags even with F threshold")
	var ciErr *CIGateError
	assert.ErrorAs(t, err, &ciErr)
	assert.Contains(t, ciErr.Error(), "secrets_detected")
}

func TestCLI_CIModeInvalidThreshold(t *testing.T) {
	_, err := execCLI(t, "scan", ".", "--offline", "--ci", "--ci-threshold", "Z", "-q")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --ci-threshold")
}

func TestCLI_CIModeInvalidFailOn(t *testing.T) {
	_, err := execCLI(t, "scan", ".", "--offline", "--ci", "--ci-fail-on", "none", "-q")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --ci-fail-on")
}

func TestCLI_ScanInvalidOutputPath(t *testing.T) {
	fixture := testdata.FixturePath(testdata.HealthySaas)
	_, err := execCLI(t, "scan", fixture, "--offline", "--format", "json", "-q", "-o", "/nonexistent/dir/scan.json")
	assert.Error(t, err, "scan with invalid output path should return error")
	assert.Contains(t, err.Error(), "output directory does not exist",
		"error should mention that the output directory does not exist")
}

// copyDir recursively copies a directory tree.
func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, info.Mode())
	})
	require.NoError(t, err)
}
