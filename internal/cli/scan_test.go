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
}

func TestScanCommand_DefaultFlags(t *testing.T) {
	f := scanCmd.Flags()

	output, err := f.GetString("output")
	require.NoError(t, err)
	assert.Equal(t, "./vettcode-scan-result.json", output)

	format, err := f.GetString("format")
	require.NoError(t, err)
	assert.Equal(t, "both", format)

	offline, err := f.GetBool("offline")
	require.NoError(t, err)
	assert.False(t, offline)

	quiet, err := f.GetBool("quiet")
	require.NoError(t, err)
	assert.False(t, quiet)

	verbose, err := f.GetBool("verbose")
	require.NoError(t, err)
	assert.False(t, verbose)

	noGit, err := f.GetBool("no-git")
	require.NoError(t, err)
	assert.False(t, noGit)
}
