package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// runEval handles the eval subcommand
func runEval(args []string) error {
	var input string
	var useJSON bool
	var hasFile bool

	// Parse flags and get filename
	filename := ""
	for _, arg := range args {
		if arg == "--json" {
			useJSON = true
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

	if err := doc.Evaluate(); err != nil {
		return fmt.Errorf("evaluation error: %w", err)
	}

	// Output results
	if useJSON {
		outputJSON(doc)
	} else {
		outputText(doc)
	}

	return nil
}

// outputText prints results in human-readable annotated format
func outputText(doc *document.Document) {
	blocks := doc.GetBlocks()

	for i, node := range blocks {
		switch block := node.Block.(type) {
		case *document.CalcBlock:
			// Print source
			for _, line := range block.Source() {
				fmt.Println(line)
			}

			// Always show result on next line
			if block.Error() != nil {
				fmt.Printf("  # Error: %v\n", block.Error())
			} else if block.LastValue() != nil {
				// Show computed value
				fmt.Printf("  = %v\n", block.LastValue())
			}

		case *document.TextBlock:
			// Print markdown as-is
			for _, line := range block.Source() {
				fmt.Println(line)
			}
		}

		// Add spacing between blocks (except last)
		if i < len(blocks)-1 {
			fmt.Println()
		}
	}
}

// outputJSON prints results as JSON
func outputJSON(doc *document.Document) {
	type Result struct {
		Type   string            `json:"type"`
		Source string            `json:"source,omitempty"`
		Value  string            `json:"value,omitempty"`
		Error  string            `json:"error,omitempty"`
		Vars   map[string]string `json:"variables,omitempty"`
	}

	results := []Result{}
	allVars := make(map[string]string)

	blocks := doc.GetBlocks()
	for _, node := range blocks {
		switch block := node.Block.(type) {
		case *document.CalcBlock:
			r := Result{
				Type:   "calc",
				Source: strings.Join(block.Source(), "\n"),
			}

			if block.Error() != nil {
				r.Error = block.Error().Error()
			} else if block.LastValue() != nil {
				r.Value = fmt.Sprintf("%v", block.LastValue())
			}

			// Collect variables
			for _, varName := range block.Variables() {
				if block.LastValue() != nil {
					allVars[varName] = fmt.Sprintf("%v", block.LastValue())
				}
			}

			results = append(results, r)

		case *document.TextBlock:
			results = append(results, Result{
				Type:   "text",
				Source: strings.Join(block.Source(), "\n"),
			})
		}
	}

	// Add variables to last result
	if len(results) > 0 {
		results[len(results)-1].Vars = allVars
	}

	output, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))
}
