package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vettcode",
	Short: "VettCode — Privacy-first technical due diligence for software M&A.",
	Long: `VettCode — Privacy-first technical due diligence for software M&A.
Scan your codebase locally. No source code ever leaves your machine.

Get Started:
  1. Scan your code     vettcode scan .
  2. Upload the JSON    Upload vettcode-scan-result.json at https://vettcode.com/upload
  3. Pay for report     Get a signed, verifiable report to share with buyers

Common Examples:
  vettcode scan .                              Scan current directory
  vettcode scan ./backend ./frontend           Scan multiple repos as one project
  vettcode scan . -o my-scan.json              Custom output file name
  vettcode scan . --offline                    Fully local, no network calls

Guides & Help:
  Scanner guide         https://vettcode.com/guide#scanner
  Report guide          https://vettcode.com/guide#reports
  Full documentation    https://vettcode.com/guide`,
}

func init() {
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
