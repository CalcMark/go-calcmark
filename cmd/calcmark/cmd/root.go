package cmd

import (
	"fmt"
	"os"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config"
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

var (
	colorModeFlag string
)

func init() {
	// Persistent flags available to all subcommands
	rootCmd.PersistentFlags().StringVar(&colorModeFlag, "color-mode", "",
		"Color mode: 'auto' (detect from terminal), 'light', or 'dark'")

	// Load config before any command runs
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Load config (will apply color mode)
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Override color mode if flag is set
		if colorModeFlag != "" {
			cfg.TUI.ColorMode = colorModeFlag
			// Re-apply color mode with the override
			config.ApplyColorModeOverride(colorModeFlag)
		}

		return nil
	}

}
