package cmd

import (
	"fmt"
	"os"

	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui"
	tea "github.com/charmbracelet/bubbletea"
)

// runREPL starts the interactive TUI REPL
func runREPL() {
	doc, _ := document.NewDocument("")
	app := tui.NewApp(doc)
	runTUIApp(app)
}

// runEdit starts the editor mode, optionally with a file
func runEdit(filepath string) {
	var doc *document.Document
	var err error

	if filepath != "" {
		doc, err = loadAndEvaluate(filepath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading file: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Start with empty document
		doc, _ = document.NewDocument("")
	}

	// Always use Editor app for edit command
	app := tui.NewEditorApp(doc, filepath)
	runTUIApp(app)
}

// runTUIApp starts the TUI with the given app model
func runTUIApp(app *tui.App) {
	p := tea.NewProgram(app, tea.WithAltScreen())
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

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		return nil, fmt.Errorf("evaluate: %w", err)
	}

	return doc, nil
}
