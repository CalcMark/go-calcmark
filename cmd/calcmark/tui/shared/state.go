package shared

import (
	"github.com/CalcMark/go-calcmark/spec/types"
)

// Mode represents the current TUI mode.
type Mode int

const (
	ModeREPL     Mode = iota // Simple REPL mode
	ModeEditor               // Document editor mode
	ModeHelp                 // Help viewer
	ModeFilePick             // File picker
)

// InputMode represents the input state within a mode.
type InputMode int

const (
	InputNormal   InputMode = iota // Normal input
	InputCommand                   // Command entry (triggered by ":")
	InputMarkdown                  // Multi-line markdown entry
	InputEditing                   // Line editing in editor
)

// Note: Slash commands (/help, /vars, etc.) were originally used for REPL
// commands but were removed because "/" is the CalcMark divide operator.
// Commands now use the ":" prefix (e.g., :help, :vars).

// HistoryEntry represents a single REPL history entry.
type HistoryEntry struct {
	Input   string // The expression or command entered
	Output  string // The result (formatted)
	IsError bool   // Whether this was an error
}

// PinnedVar represents a variable displayed in the pinned panel.
type PinnedVar struct {
	Name          string
	Value         types.Type // The actual value
	Changed       bool       // Was this variable modified in the last operation?
	IsFrontmatter bool       // Is this a frontmatter variable?
}

// Command defines a REPL command with its syntax and description.
type Command struct {
	Name        string // Command name without prefix
	Syntax      string // Full syntax example (e.g., ":help")
	Description string // Brief description
}

// DefaultCommands returns the list of available commands for Simple REPL.
func DefaultCommands() []Command {
	return []Command{
		{"help", ":help", "Show help"},
		{"vars", ":vars", "List all variables"},
		{"clear", ":clear", "Clear screen (keep variables)"},
		{"reset", ":reset", "Clear everything"},
		{"edit", ":edit [file]", "Switch to editor mode"},
		{"quit", ":quit", "Exit REPL"},
		{"q", ":q", "Exit (shortcut)"},
		{"h", ":h", "Help (shortcut)"},
		{"?", ":?", "Help (shortcut)"},
	}
}

// Dimensions holds terminal dimensions.
type Dimensions struct {
	Width  int
	Height int
}

// MinDimensions returns minimum usable dimensions.
func MinDimensions() Dimensions {
	return Dimensions{Width: 40, Height: 10}
}

// SwitchModeMsg is a message to switch to a different mode.
type SwitchModeMsg struct {
	Mode     Mode   // Target mode
	Filepath string // Optional file path for editor mode
}
