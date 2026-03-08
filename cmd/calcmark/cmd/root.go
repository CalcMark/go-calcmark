package cmd

import (
	"fmt"
	"os"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config"
	"github.com/CalcMark/go-calcmark/format/display"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cm [file]",
	Short: "CalcMark - A reactive calculation and markdown language",
	Long: `  ♪
 ('>    CalcMark — reactive calculations meet markdown.
 /V\
(| |)

Examples:
  cm                              Open editor with new document
  cm budget.cm                    Open file in editor
  echo "x = 42" | cm             Evaluate piped input
  echo "x = 42" | cm --format json  Evaluate piped input as JSON
  cm eval calc.cm                 Evaluate file and print result
  cm eval < input.cm              Evaluate from stdin
  cm convert doc.cm --to=html     Convert to HTML
  cm remote --gist abc123         Open a GitHub Gist
  cm remote --http https://...    Open a public URL

GitHub Gist support requires the gh CLI: https://cli.github.com`,
	// Don't dump usage on every error — just show the error message.
	SilenceUsage: true,
	// Allow 0 or 1 file argument
	Args: cobra.MaximumNArgs(1),
	// When called without subcommand, open editor — unless stdin is piped,
	// in which case evaluate and print the result (issue #23).
	RunE: func(cmd *cobra.Command, args []string) error {
		// Piped stdin without a file argument → evaluate like `cm eval`.
		if stdinIsPiped() && len(args) == 0 {
			evalFormat = rootFormat
			evalVerbose = rootVerbose
			return runEval(nil)
		}

		// If a file argument is provided, open in editor mode
		if len(args) > 0 {
			runEdit(args[0])
			return nil
		}
		// Otherwise open editor with empty document
		runEdit("")
		return nil
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// cobra already printed "Error: ..." to stderr; no need to repeat.
		os.Exit(1)
	}
}

var (
	colorModeFlag string
	localeFlag    string
	rootFormat    string
	rootVerbose   bool
	debugKeysFlag bool
)

func init() {
	// Help command is registered in help.go via rootCmd.SetHelpCommand(helpCmd)

	// Persistent flags available to all subcommands
	rootCmd.PersistentFlags().StringVar(&colorModeFlag, "color-mode", "",
		"Color mode: 'auto' (detect from terminal), 'light', or 'dark'")
	rootCmd.PersistentFlags().StringVar(&localeFlag, "locale", "",
		"Display locale for number formatting (e.g., 'en-US', 'de-DE', 'fr-FR')")
	rootCmd.PersistentFlags().BoolVar(&debugKeysFlag, "debug-keys", false,
		"Log raw key events to stderr for debugging keyboard issues")

	// --format applies when stdin is piped (e.g. `echo "1+1" | cm --format json`)
	rootCmd.Flags().StringVar(&rootFormat, "format", "text",
		"Output format for piped input: text, json, html, md, cm")
	rootCmd.Flags().BoolVarP(&rootVerbose, "verbose", "v", false,
		"Show all intermediate values (piped input)")

	// Load config before any command runs
	// NOTE: localeFormatter() below depends on config being loaded first
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

		// Override locale if flag is set
		// Precedence: --locale flag > config.toml > "en-US" default
		if localeFlag != "" {
			cfg.Locale = localeFlag
		}

		return nil
	}

}

// localeFormatter builds a display.Formatter from the resolved config locale.
// Must be called after config.Load() (i.e., inside or after PersistentPreRunE).
// Invalid locale falls back to en-US with a warning to stderr.
func localeFormatter() display.Formatter {
	locale := config.Get().Locale
	if locale == "" {
		return display.DefaultFormatter()
	}
	dcfg, err := display.NewConfig(locale)
	if err != nil {
		fmt.Fprintf(os.Stderr, "calcmark: invalid locale %q, using en-US: %v\n", locale, err)
		return display.DefaultFormatter()
	}
	return display.NewFormatter(dcfg)
}
