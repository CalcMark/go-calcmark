package cmd

import (
	"fmt"
	"os"

	"github.com/CalcMark/go-calcmark/spec/document"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui"
	tea "github.com/charmbracelet/bubbletea"
)

// runEdit starts the editor mode, optionally with a file
func runEdit(filepath string) {
	var doc *document.Document
	var err error

	if filepath != "" {
		doc, err = loadDocument(filepath)
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
	// CRITICAL: Flush any pending output before entering alternate screen
	// This prevents terminal history from bleeding through
	os.Stdout.Sync()
	os.Stderr.Sync()

	// Ensure terminal colors are reset when we exit
	defer func() {
		fmt.Fprint(os.Stderr, "\x1b[0m") // Reset all attributes
	}()

	// Configure TUI with:
	// - tea.WithAltScreen(): Enter alternate screen (clean slate, no terminal history)
	// - tea.WithMouseCellMotion(): Enable mouse support for better UX
	// - tea.WithInput(os.Stdin): Explicitly use stdin
	// - tea.WithOutput(os.Stderr): Use stderr for TUI output (stdout is for data)
	p := tea.NewProgram(app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stderr),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}

// loadDocument loads and parses a file into a document.
// Only file-system and parse errors are fatal — evaluation errors are
// diagnostic and handled by the editor's preview pane.
func loadDocument(path string) (*document.Document, error) {
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

	return doc, nil
}
