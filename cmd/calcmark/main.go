package main

import (
	"fmt"
	"os"

	"github.com/CalcMark/go-calcmark/spec/document"
	tea "github.com/charmbracelet/bubbletea"
)

const usage = `CalcMark - A reactive calculation and markdown language

Usage:
  cm                    Start interactive REPL
  cm <file>             Start REPL with file loaded
  cm eval <file>        Evaluate file and print results
  cm eval               Evaluate from stdin (pipe support)
  cm help               Show this help

Options:
  --json                Output results as JSON (with eval)
  --no-tui              Use basic REPL instead of TUI

Examples:
  cm                              # Interactive REPL
  cm budget.cm                    # Load file in REPL
  cm eval calc.cm                 # Evaluate and print
  echo "x = 10" | cm eval         # Pipe evaluation
  echo "3 + 4 * 5" | cm eval      # Simple expression
  cm eval data.cm --json          # JSON output

Valid CalcMark Syntax:
  x = 10                          # Assignment
  3 + 4 * 5                       # Expression
  price = 100 USD                 # With units
  total = price * 1.1             # Variable reference
  
Note: Natural language like "average of 3, 4, 5" is not yet supported.
      Use: (3 + 4 + 5) / 3

Commands in REPL:
  /                     Enter command mode
  /pin                  Pin all variables
  /open <file>          Load file
  /md                   Multi-line markdown
  /quit                 Exit
  Esc                   Exit command mode
  Ctrl+C                Quit
`

func main() {
	args := os.Args[1:]

	// No arguments: interactive REPL
	if len(args) == 0 {
		runREPL(nil)
		return
	}

	// Parse command
	cmd := args[0]

	switch cmd {
	case "help", "-h", "--help":
		fmt.Print(usage)
		return

	case "eval", "evaluate":
		// Evaluate mode
		if err := runEval(args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return

	default:
		// Assume it's a filename
		if _, err := os.Stat(cmd); err == nil {
			// File exists, load in REPL
			doc, err := loadAndEvaluate(cmd)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading file: %v\n", err)
				os.Exit(1)
			}
			runREPL(doc)
		} else {
			// Unknown command
			fmt.Fprintf(os.Stderr, "Unknown command or file not found: %s\n\n", cmd)
			fmt.Print(usage)
			os.Exit(1)
		}
	}
}

// runREPL starts the interactive TUI REPL
func runREPL(doc *document.Document) {
	if doc == nil {
		doc, _ = document.NewDocument("")
	}

	p := tea.NewProgram(newTUIModel(doc), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}

// loadAndEvaluate loads a file and evaluates it
func loadAndEvaluate(path string) (*document.Document, error) {
	if err := validateFilePath(path); err != nil {
		return nil, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	doc, err := document.NewDocument(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse document: %w", err)
	}

	if err := doc.Evaluate(); err != nil {
		return nil, fmt.Errorf("evaluate: %w", err)
	}

	return doc, nil
}
