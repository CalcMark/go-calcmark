package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2/compat"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/config"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/editor"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/repl"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/shared"
	"github.com/CalcMark/go-calcmark/format/display"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// initializeColorProfile sets up lipgloss color settings.
// MUST be called after alternate screen is entered to avoid terminal artifacts.
func initializeColorProfile() {
	// Set the background mode from config. In lipgloss v2, adaptive colors
	// use compat.HasDarkBackground to resolve Light/Dark slots.
	compat.HasDarkBackground = config.IsDarkMode()
}

// App represents the root TUI application.
// It manages the current mode and delegates to mode-specific models.
type App struct {
	mode   shared.Mode
	repl   repl.Model
	editor editor.Model

	// Display formatting (locale-aware), preserved across mode switches
	formatter display.Formatter

	width    int
	height   int
	quitting bool
}

// NewApp creates a new TUI application in REPL mode.
func NewApp(doc *document.Document) *App {
	r := repl.New(doc)
	return &App{
		mode: shared.ModeREPL,
		repl: r,
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

// SetFormatter sets the locale-aware display formatter on the App and its sub-models.
// Must be called after construction but before Init().
func (a *App) SetFormatter(f display.Formatter) {
	a.formatter = f
	a.repl.SetFormatter(f)
	a.editor.SetFormatter(f)
}

// SetDebugKeys enables logging of raw key events to stderr.
func (a *App) SetDebugKeys(enabled bool) {
	a.editor.SetDebugKeys(enabled)
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
// Preserves the locale-aware formatter across mode transitions.
func (a *App) switchMode(msg shared.SwitchModeMsg) (tea.Model, tea.Cmd) {
	switch msg.Mode {
	case shared.ModeEditor:
		// Switch to editor mode, carrying over the current document
		doc := a.repl.Document()
		if msg.Filepath != "" {
			a.editor = editor.NewWithFile(msg.Filepath, doc)
		} else {
			a.editor = editor.New(doc)
		}
		a.editor.SetFormatter(a.formatter)
		a.mode = shared.ModeEditor

	case shared.ModeREPL:
		// Switch back to REPL mode
		doc := a.editor.Document()
		a.repl = repl.New(doc)
		a.repl.SetFormatter(a.formatter)
		a.mode = shared.ModeREPL
	}

	return a, nil
}

// View implements tea.Model.
func (a *App) View() tea.View {
	if a.quitting {
		return tea.NewView("")
	}

	var v tea.View
	switch a.mode {
	case shared.ModeREPL:
		v = a.repl.View()
	case shared.ModeEditor:
		v = a.editor.View()
	default:
		v = tea.NewView("Unknown mode")
	}

	// Declarative terminal configuration (replaces v1 program options)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion

	return v
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
