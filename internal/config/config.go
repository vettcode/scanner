package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config holds all scanner configuration.
type Config struct {
	// Scan targets (populated after ParsePaths)
	Repos []RepoInput

	// Output
	Output string
	Format string // "terminal", "json", "both"
	Quiet  bool
	NoColor bool

	// Behavior
	Offline  bool
	NoGit    bool
	Verbose  bool
	Timeout  time.Duration
	LogLevel string

	// Directories
	Home string // VETTCODE_HOME, default ~/.vettcode

	// Update check
	NoUpdateCheck bool
}

// Load builds Config from cobra flags and environment variables.
func Load(cmd *cobra.Command) (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix("VETTCODE")
	v.AutomaticEnv()

	// Bind env vars
	_ = v.BindEnv("HOME")
	_ = v.BindEnv("OFFLINE")
	_ = v.BindEnv("NO_COLOR")
	_ = v.BindEnv("LOG_LEVEL")
	_ = v.BindEnv("NO_UPDATE_CHECK")

	cfg := &Config{}

	// Home directory
	cfg.Home = v.GetString("HOME")
	if cfg.Home == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home directory: %w", err)
		}
		cfg.Home = filepath.Join(home, ".vettcode")
	}

	// Flags (flags override env vars)
	if cmd != nil {
		flags := cmd.Flags()

		if flags.Changed("output") {
			cfg.Output, _ = flags.GetString("output")
		} else {
			cfg.Output = "./vettcode-scan-result.json"
		}

		if flags.Changed("format") {
			cfg.Format, _ = flags.GetString("format")
		} else {
			cfg.Format = "both"
		}

		cfg.Quiet, _ = flags.GetBool("quiet")
		cfg.Verbose, _ = flags.GetBool("verbose")
		cfg.Timeout, _ = flags.GetDuration("timeout")

		noColor, _ := flags.GetBool("no-color")
		cfg.NoColor = noColor || v.GetBool("NO_COLOR")

		offline, _ := flags.GetBool("offline")
		cfg.Offline = offline || v.GetBool("OFFLINE")

		noGit, _ := flags.GetBool("no-git")
		cfg.NoGit = noGit
	}

	// Defaults when no cobra command is provided
	if cfg.Output == "" {
		cfg.Output = "./vettcode-scan-result.json"
	}
	if cfg.Format == "" {
		cfg.Format = "both"
	}

	// Env-only settings
	cfg.LogLevel = v.GetString("LOG_LEVEL")
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	if cfg.Verbose {
		cfg.LogLevel = "debug"
	}

	cfg.NoUpdateCheck = v.GetBool("NO_UPDATE_CHECK")

	// Validate format
	switch cfg.Format {
	case "terminal", "json", "both":
		// valid
	default:
		return nil, fmt.Errorf("invalid --format value %q: must be terminal, json, or both", cfg.Format)
	}

	return cfg, nil
}
