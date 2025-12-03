package tui

import (
	"github.com/CalcMark/go-calcmark/cmd/calcmark/config"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/editor"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/repl"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/shared"
	"github.com/CalcMark/go-calcmark/spec/document"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// initializeColorProfile sets up lipgloss color settings.
// MUST be called after alternate screen is entered to avoid terminal artifacts.
func initializeColorProfile() {
	// Set color profile and background explicitly to avoid terminal queries.
	// We set TrueColor without querying the terminal.
	lipgloss.SetColorProfile(termenv.TrueColor)

	// Use the color mode from config (already loaded at this point)
	// This avoids any terminal queries for background detection
	lipgloss.SetHasDarkBackground(config.IsDarkMode())
}

// App represents the root TUI application.
// It manages the current mode and delegates to mode-specific models.
type App struct {
	mode   shared.Mode
	repl   repl.Model
	editor editor.Model

	width    int
	height   int
	quitting bool
}

// NewApp creates a new TUI application in REPL mode.
func NewApp(doc *document.Document) *App {
	return &App{
		mode: shared.ModeREPL,
		repl: repl.New(doc),
	}
}

// NewEditorApp creates a new TUI application in Editor mode.
func NewEditorApp(doc *document.Document, filepath string) *App {
	var ed editor.Model
	if filepath != "" {
		ed = editor.NewWithFile(filepath, doc)
	} else {
		ed = editor.New(doc)
	}

	return &App{
		mode:   shared.ModeEditor,
		editor: ed,
	}
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	// Initialize lipgloss color profile AFTER alternate screen is entered
	// to avoid terminal queries that cause artifacts
	initializeColorProfile()

	switch a.mode {
	case shared.ModeREPL:
		return a.repl.Init()
	case shared.ModeEditor:
		return a.editor.Init()
	}
	return nil
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case shared.SwitchModeMsg:
		// Switch to a different mode
		return a.switchMode(msg)
	}

	// Delegate to current mode
	switch a.mode {
	case shared.ModeREPL:
		newModel, cmd := a.repl.Update(msg)
		a.repl = newModel.(repl.Model)
		if a.repl.Quitting() {
			a.quitting = true
		}
		return a, cmd

	case shared.ModeEditor:
		newModel, cmd := a.editor.Update(msg)
		a.editor = newModel.(editor.Model)
		if a.editor.Quitting() {
			a.quitting = true
		}
		return a, cmd
	}

	return a, nil
}

// switchMode handles switching between REPL and Editor modes.
func (a *App) switchMode(msg shared.SwitchModeMsg) (tea.Model, tea.Cmd) {
	switch msg.Mode {
	case shared.ModeEditor:
		// Switch to editor mode, carrying over the current document
		doc := a.repl.Document()
		if msg.Filepath != "" {
			// Load file if specified
			a.editor = editor.NewWithFile(msg.Filepath, doc)
		} else {
			a.editor = editor.New(doc)
		}
		a.mode = shared.ModeEditor

	case shared.ModeREPL:
		// Switch back to REPL mode
		doc := a.editor.Document()
		a.repl = repl.New(doc)
		a.mode = shared.ModeREPL
	}

	return a, nil
}

// View implements tea.Model.
func (a *App) View() string {
	if a.quitting {
		return ""
	}

	switch a.mode {
	case shared.ModeREPL:
		return a.repl.View()
	case shared.ModeEditor:
		return a.editor.View()
	}

	return "Unknown mode"
}

// SetMode switches to a different mode.
func (a *App) SetMode(mode shared.Mode) {
	a.mode = mode
}

// Document returns the current document.
func (a *App) Document() *document.Document {
	switch a.mode {
	case shared.ModeREPL:
		return a.repl.Document()
	case shared.ModeEditor:
		return a.editor.Document()
	}
	return nil
}
