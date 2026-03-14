package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/vettcode/scanner/internal/grammar"
)

var grammarCmd = &cobra.Command{
	Use:   "grammar",
	Short: "Manage tree-sitter grammars for code analysis",
	Long: `Manage tree-sitter grammars used by VettCode for AST-based code analysis.

Grammars are automatically downloaded on first scan, but these commands
let you manage them manually (e.g., pre-cache for air-gapped environments).`,
}

var grammarListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available grammars and their cache status",
	RunE:  runGrammarList,
}

var grammarInstallCmd = &cobra.Command{
	Use:   "install [languages...]",
	Short: "Download and cache grammars",
	Long: `Download and cache tree-sitter grammars. Specify language names to install
specific grammars, or omit arguments to install all available grammars.

Examples:
  vettcode grammar install              # install all grammars
  vettcode grammar install javascript   # install just JavaScript
  vettcode grammar install python ruby  # install Python and Ruby`,
	RunE: runGrammarInstall,
}

var grammarUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Re-download all cached grammars to get latest versions",
	RunE:  runGrammarUpdate,
}

func init() {
	grammarCmd.AddCommand(grammarListCmd)
	grammarCmd.AddCommand(grammarInstallCmd)
	grammarCmd.AddCommand(grammarUpdateCmd)
}

func grammarHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".vettcode")
	}
	return filepath.Join(home, ".vettcode")
}

func runGrammarList(cmd *cobra.Command, args []string) error {
	mgr := grammar.NewManager(grammarHome(), true) // offline=true, just check cache

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintf(w, "LANGUAGE\tFILENAME\tSTATUS\n")

	for lang, entry := range grammar.GrammarManifest {
		status := "not cached"
		path := mgr.GrammarPath(lang)
		if _, err := os.Stat(path); err == nil {
			status = "cached"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", lang, entry.Filename, status)
	}
	w.Flush()

	fmt.Fprintf(cmd.OutOrStdout(), "\nGrammar version: %s\n", grammar.GrammarVersion)
	fmt.Fprintf(cmd.OutOrStdout(), "Cache directory: %s\n", mgr.CacheDir())
	return nil
}

func runGrammarInstall(cmd *cobra.Command, args []string) error {
	mgr := grammar.NewManager(grammarHome(), false) // online mode

	langs := args
	if len(langs) == 0 {
		// Install all
		for lang := range grammar.GrammarManifest {
			langs = append(langs, lang)
		}
	}

	var errs []string
	for _, lang := range langs {
		if _, ok := grammar.GrammarManifest[lang]; !ok {
			fmt.Fprintf(cmd.ErrOrStderr(), "unknown language: %s\n", lang)
			errs = append(errs, lang)
			continue
		}
		path, err := mgr.EnsureGrammar(lang)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "failed to install %s: %v\n", lang, err)
			errs = append(errs, lang)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "installed %s → %s\n", lang, path)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to install grammars: %v", errs)
	}
	return nil
}

func runGrammarUpdate(cmd *cobra.Command, args []string) error {
	mgr := grammar.NewManager(grammarHome(), false) // online mode

	// Remove cached grammars to force re-download
	cacheDir := mgr.CacheDir()
	if _, err := os.Stat(cacheDir); err == nil {
		fmt.Fprintf(cmd.OutOrStdout(), "clearing grammar cache: %s\n", cacheDir)
		if err := os.RemoveAll(cacheDir); err != nil {
			return fmt.Errorf("failed to clear grammar cache: %w", err)
		}
	}

	// Re-download all
	return runGrammarInstall(cmd, nil)
}
