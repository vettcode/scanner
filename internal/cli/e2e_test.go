package cli

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2E tests for CLI commands. Full scan E2E tests are deferred until the
// scan orchestrator (runScan) is implemented.
//
// NOTE: These tests share the package-level rootCmd and MUST NOT use
// t.Parallel() — cobra's global state is not safe for concurrent access.

// execCLI is a helper that sets up a fresh output buffer and runs the root command
// with the given args. It resets args and output afterward to avoid state leakage.
func execCLI(t *testing.T, args ...string) (output string, err error) {
	t.Helper()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	err = rootCmd.Execute()
	// Reset state to prevent leakage between tests
	rootCmd.SetArgs(nil)
	rootCmd.SetOut(nil)
	rootCmd.SetErr(nil)
	return buf.String(), err
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
	// With no arguments, scan should either run the placeholder or show help
	// without erroring out. Full scan E2E tests require runScan implementation.
	_, err := execCLI(t, "scan")
	assert.NoError(t, err, "scan with no args should not error")
}

// Deferred E2E tests — require runScan implementation.

func TestCLI_ScanNonexistentPath(t *testing.T) {
	t.Skip("requires runScan implementation — scan should validate paths and return error")
}

func TestCLI_ScanEmptyDir(t *testing.T) {
	t.Skip("requires runScan implementation — scan should detect no supported languages")
}

func TestCLI_ScanQuietMode(t *testing.T) {
	t.Skip("requires runScan implementation — scan with --quiet should suppress terminal output")
}

func TestCLI_ScanNoColorMode(t *testing.T) {
	t.Skip("requires runScan implementation — scan with --no-color should have no ANSI codes")
}

func TestCLI_ScanOfflineCached(t *testing.T) {
	t.Skip("requires runScan implementation — scan with --offline and cached grammars")
}

func TestCLI_ScanFixtureHealthySaas(t *testing.T) {
	t.Skip("requires runScan implementation — full scan of healthy-saas fixture")
}

func TestCLI_ScanFixtureNeglectedProject(t *testing.T) {
	t.Skip("requires runScan implementation — full scan of neglected-project fixture")
}

func TestCLI_ScanFixtureSecurityNightmare(t *testing.T) {
	t.Skip("requires runScan implementation — full scan of security-nightmare fixture")
}
