package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Global flags shared across commands.
var (
	flagPath         string
	flagBranch       string
	flagCommit       string
	flagConfig       string
	flagOutput       string
	flagShowVariable string
	flagShowConfig   bool
	flagExplain      bool
	flagVerbosity    string
)

// rootCmd is the top-level command for gitsemver.
var rootCmd = &cobra.Command{
	Use:   "gitsemver",
	Short: "Semantic versioning from git history",
	Long:  "gitsemver calculates the next semantic version based on git history, tags, and branch conventions.",
	// Default action is calculate.
	RunE: calculateRunE,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagPath, "path", "p", ".", "path to the git repository")
	rootCmd.PersistentFlags().StringVarP(&flagBranch, "branch", "b", "", "target branch (default: current HEAD)")
	rootCmd.PersistentFlags().StringVarP(&flagCommit, "commit", "c", "", "target commit SHA (default: branch tip)")
	rootCmd.PersistentFlags().StringVar(&flagConfig, "config", "", "path to config file (default: auto-detect)")
	rootCmd.PersistentFlags().StringVarP(&flagOutput, "output", "o", "", "output format: json, buildserver, or empty for default")
	rootCmd.PersistentFlags().StringVar(&flagShowVariable, "show-variable", "", "output a single variable (e.g. SemVer, FullSemVer)")
	rootCmd.PersistentFlags().BoolVar(&flagShowConfig, "show-config", false, "display the effective configuration and exit")
	rootCmd.PersistentFlags().BoolVar(&flagExplain, "explain", false, "show how the version was calculated")
	rootCmd.PersistentFlags().StringVarP(&flagVerbosity, "verbosity", "v", "info", "log verbosity: quiet, info, debug")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
