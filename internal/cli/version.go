package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// These are set via ldflags at build time.
var (
	version = "dev"
	date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version, build date, and platform info",
	Run: func(cmd *cobra.Command, args []string) {
		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "vettcode %s\n", version)
		fmt.Fprintf(out, "  built:    %s\n", date)
		fmt.Fprintf(out, "  platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

// Version returns the current scanner version string.
func Version() string {
	return version
}
