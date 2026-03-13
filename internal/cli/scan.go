package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan [paths...]",
	Short: "Scan one or more directories for technical health assessment",
	Long: `Scan one or more directories and produce a technical health assessment.

Results are displayed in the terminal and saved as a JSON file. To get a
signed report you can share with buyers, upload the JSON at vettcode.com.

Multi-Repo Scanning:
  If your product spans multiple repositories, pass all paths in a single
  command. VettCode aggregates metrics across repos into one combined scan
  — grades reflect the full codebase, not individual repos.

  Example: vettcode scan ./api ./web ./worker

  Use --label to give each repo a human-readable name in the output:
    vettcode scan --label api:./api --label web:./web --label worker:./worker`,
	Example: `  # Scan a single project
  vettcode scan .

  # Scan multiple repos as one product (combined report)
  vettcode scan ./backend ./frontend ./infra

  # Label repos for clarity in the report
  vettcode scan --label api:./backend --label web:./frontend --label infra:./deploy

  # Scan with custom output path
  vettcode scan . -o ~/Desktop/my-scan.json

  # Fully offline — no network calls, no co-signing
  vettcode scan . --offline

  # JSON output only (no terminal display)
  vettcode scan . --format json -q`,
	Args: cobra.ArbitraryArgs,
	RunE: runScan,
}

func init() {
	f := scanCmd.Flags()
	f.StringP("output", "o", "./vettcode-scan-result.json", "Output JSON file path")
	f.StringSlice("label", nil, "Label repos as name:path (e.g., --label frontend:./fe)")
	f.Bool("offline", false, "Skip remote co-signing (fully local, no network calls)")
	f.Bool("no-color", false, "Disable color terminal output")
	f.BoolP("quiet", "q", false, "Suppress terminal output (JSON only)")
	f.String("format", "both", "Output format: terminal, json, both")
	f.BoolP("verbose", "v", false, "Enable verbose/debug logging")
	f.Bool("no-git", false, "Skip git-based analysis (activity, contributors)")
	f.Duration("timeout", 30*time.Minute, "Maximum scan duration")
}

func runScan(cmd *cobra.Command, args []string) error {
	// Placeholder — will be implemented in subsequent tickets
	fmt.Fprintln(cmd.OutOrStdout(), "Scan not yet implemented. Use 'vettcode help scan' for usage.")
	return nil
}
