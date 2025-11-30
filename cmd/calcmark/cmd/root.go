package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cm [file]",
	Short: "CalcMark - A reactive calculation and markdown language",
	Long: `CalcMark is an interpreted language that blends CommonMark markdown
and calculations in one document. Calculations are verifiable and reproducible.

Examples:
  cm                              Start interactive REPL
  cm budget.cm                    Open file in editor
  cm eval calc.cm                 Evaluate file and print result
  cm eval < input.cm              Evaluate from stdin
  cm convert doc.cm --to=html     Convert to HTML`,
	// Allow 0 or 1 file argument
	Args: cobra.MaximumNArgs(1),
	// When called without subcommand, run REPL
	Run: func(cmd *cobra.Command, args []string) {
		// If a file argument is provided, open in editor mode
		if len(args) > 0 {
			runEdit(args[0])
			return
		}
		// Otherwise start REPL
		runREPL()
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Disable default completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
