package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/CalcMark/go-calcmark/format"
	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/spf13/cobra"
)

var evalVerbose bool

var evalCmd = &cobra.Command{
	Use:   "eval [file.cm]",
	Short: "Evaluate CalcMark and print the result",
	Long: `Evaluate a CalcMark file or stdin and print the result.

Examples:
  cm eval calc.cm           Evaluate file and print result
  cm eval -v calc.cm        Evaluate with verbose output (all values)
  echo "x = 10" | cm eval   Evaluate from stdin`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEval(args)
	},
}

func init() {
	evalCmd.Flags().BoolVarP(&evalVerbose, "verbose", "v", false, "Show all intermediate values")
	rootCmd.AddCommand(evalCmd)
}

// runEval handles the eval subcommand - evaluates and prints the result
func runEval(args []string) error {
	var input string
	var hasFile bool

	if len(args) > 0 {
		filename := args[0]
		hasFile = true

		// Read from file
		if err := validateFilePath(filename); err != nil {
			return fmt.Errorf("invalid file: %w", err)
		}

		bytes, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
		input = string(bytes)
	}

	if !hasFile {
		// Read from stdin
		bytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		input = string(bytes)

		if strings.TrimSpace(input) == "" {
			return fmt.Errorf("no input provided")
		}
	}

	// Parse and evaluate
	doc, err := document.NewDocument(input)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		return fmt.Errorf("evaluation error: %w", err)
	}

	// Use text formatter for eval output
	formatter := format.GetFormatter("text", "")

	opts := format.Options{
		Verbose: evalVerbose,
	}

	if err := formatter.Format(os.Stdout, doc, opts); err != nil {
		return fmt.Errorf("format error: %w", err)
	}

	return nil
}
