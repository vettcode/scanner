package config

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	f := cmd.Flags()
	f.StringP("output", "o", "./vettcode-scan-result.json", "")
	f.String("format", "both", "")
	f.Bool("no-color", false, "")
	f.BoolP("quiet", "q", false, "")
	f.Bool("offline", false, "")
	f.BoolP("verbose", "v", false, "")
	f.Bool("no-git", false, "")
	f.Duration("timeout", 30*time.Minute, "")
	return cmd
}

func TestLoad_Defaults(t *testing.T) {
	// Clear env vars that might interfere
	t.Setenv("VETTCODE_HOME", "")
	t.Setenv("VETTCODE_OFFLINE", "")
	t.Setenv("VETTCODE_NO_COLOR", "")
	t.Setenv("VETTCODE_LOG_LEVEL", "")
	t.Setenv("VETTCODE_NO_UPDATE_CHECK", "")

	cmd := newTestCmd()
	cfg, err := Load(cmd)
	require.NoError(t, err)

	assert.Equal(t, "./vettcode-scan-result.json", cfg.Output)
	assert.Equal(t, "both", cfg.Format)
	assert.False(t, cfg.Quiet)
	assert.False(t, cfg.NoColor)
	assert.False(t, cfg.Offline)
	assert.False(t, cfg.Verbose)
	assert.False(t, cfg.NoGit)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.False(t, cfg.NoUpdateCheck)
	assert.NotEmpty(t, cfg.Home)
}

func TestLoad_EnvVars(t *testing.T) {
	t.Setenv("VETTCODE_HOME", "/tmp/test-vettcode")
	t.Setenv("VETTCODE_OFFLINE", "true")
	t.Setenv("VETTCODE_NO_COLOR", "true")
	t.Setenv("VETTCODE_LOG_LEVEL", "debug")
	t.Setenv("VETTCODE_NO_UPDATE_CHECK", "true")

	cmd := newTestCmd()
	cfg, err := Load(cmd)
	require.NoError(t, err)

	assert.Equal(t, "/tmp/test-vettcode", cfg.Home)
	assert.True(t, cfg.Offline)
	assert.True(t, cfg.NoColor)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.True(t, cfg.NoUpdateCheck)
}

func TestLoad_VerboseOverridesLogLevel(t *testing.T) {
	t.Setenv("VETTCODE_HOME", "")
	t.Setenv("VETTCODE_OFFLINE", "")
	t.Setenv("VETTCODE_NO_COLOR", "")
	t.Setenv("VETTCODE_LOG_LEVEL", "warn")
	t.Setenv("VETTCODE_NO_UPDATE_CHECK", "")

	cmd := newTestCmd()
	_ = cmd.Flags().Set("verbose", "true")
	cfg, err := Load(cmd)
	require.NoError(t, err)

	assert.Equal(t, "debug", cfg.LogLevel)
}

func TestLoad_InvalidFormat(t *testing.T) {
	t.Setenv("VETTCODE_HOME", "")
	t.Setenv("VETTCODE_OFFLINE", "")
	t.Setenv("VETTCODE_NO_COLOR", "")
	t.Setenv("VETTCODE_LOG_LEVEL", "")
	t.Setenv("VETTCODE_NO_UPDATE_CHECK", "")

	cmd := newTestCmd()
	_ = cmd.Flags().Set("format", "xml")
	_, err := Load(cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --format")
}

func TestLoad_NilCmd(t *testing.T) {
	t.Setenv("VETTCODE_HOME", "/tmp/test")
	t.Setenv("VETTCODE_OFFLINE", "")
	t.Setenv("VETTCODE_NO_COLOR", "")
	t.Setenv("VETTCODE_LOG_LEVEL", "")
	t.Setenv("VETTCODE_NO_UPDATE_CHECK", "")

	cfg, err := Load(nil)
	require.NoError(t, err)
	assert.Equal(t, "/tmp/test", cfg.Home)
}
