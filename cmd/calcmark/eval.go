package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/CalcMark/go-calcmark/format"
	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// runEval handles the eval subcommand with formatter support
func runEval(args []string) error {
	var input string
	var hasFile bool
	var formatName string
	var verbose bool

	// Parse flags and get filename
	filename := ""
	for _, arg := range args {
		// Backward compatibility: --json maps to --format=json
		if arg == "--json" {
			formatName = "json"
		} else if strings.HasPrefix(arg, "--format=") {
			formatName = strings.TrimPrefix(arg, "--format=")
		} else if arg == "--verbose" || arg == "-v" {
			verbose = true
		} else if !strings.HasPrefix(arg, "-") {
			filename = arg
			hasFile = true
		}
	}

	if hasFile {
		// Read from file
		if err := validateFilePath(filename); err != nil {
			return fmt.Errorf("invalid file: %w", err)
		}

		bytes, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
		input = string(bytes)
	} else {
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

	// Get formatter (defaults to text if not specified)
	formatter := format.GetFormatter(formatName, "")

	// Format options
	opts := format.Options{
		Verbose: verbose,
	}

	// Format and output
	if err := formatter.Format(os.Stdout, doc, opts); err != nil {
		return fmt.Errorf("format error: %w", err)
	}

	return nil
}
