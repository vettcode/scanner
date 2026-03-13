package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanCommand_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"scan", "--help"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Scan one or more directories")
	assert.Contains(t, output, "--output")
	assert.Contains(t, output, "--label")
	assert.Contains(t, output, "--offline")
	assert.Contains(t, output, "--no-color")
	assert.Contains(t, output, "--quiet")
	assert.Contains(t, output, "--format")
	assert.Contains(t, output, "--verbose")
	assert.Contains(t, output, "--no-git")
	assert.Contains(t, output, "--timeout")

	rootCmd.SetArgs(nil)
	rootCmd.SetOut(nil)
	rootCmd.SetErr(nil)
}

func TestScanCommand_DefaultFlags(t *testing.T) {
	// Reset flags to defaults before checking
	resetScanFlags()

	f := scanCmd.Flags()

	// Check DefValue (registered default) which is stable
	assert.Equal(t, "./vettcode-scan-result.json", f.Lookup("output").DefValue)
	assert.Equal(t, "both", f.Lookup("format").DefValue)
	assert.Equal(t, "false", f.Lookup("offline").DefValue)
	assert.Equal(t, "false", f.Lookup("quiet").DefValue)
	assert.Equal(t, "false", f.Lookup("verbose").DefValue)
	assert.Equal(t, "false", f.Lookup("no-git").DefValue)
}
