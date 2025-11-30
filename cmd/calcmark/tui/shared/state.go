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
	InputSlash                     // Slash command entry
	InputMarkdown                  // Multi-line markdown entry
	InputEditing                   // Line editing in editor
)

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

// SlashCommand defines a slash command with its syntax and description.
type SlashCommand struct {
	Name        string // Command name without /
	Syntax      string // Full syntax example
	Description string // Brief description
}

// DefaultSlashCommands returns the list of available slash commands for Simple REPL.
func DefaultSlashCommands() []SlashCommand {
	return []SlashCommand{
		{"help", "/help", "Show help"},
		{"vars", "/vars", "List all variables"},
		{"clear", "/clear", "Clear screen (keep variables)"},
		{"reset", "/reset", "Clear everything"},
		{"edit", "/edit [file]", "Switch to editor mode"},
		{"quit", "/quit", "Exit REPL"},
		{"q", "/q", "Exit (shortcut)"},
		{"h", "/h", "Help (shortcut)"},
		{"?", "/?", "Help (shortcut)"},
	}
}

// EditorSlashCommands returns the list of slash commands for Document Editor.
func EditorSlashCommands() []SlashCommand {
	return []SlashCommand{
		{"save", "/save", "Save document"},
		{"saveas", "/saveas <name>", "Save as new file"},
		{"open", "/open [file]", "Open file"},
		{"quit", "/quit", "Quit (warns if unsaved)"},
		{"help", "/help", "Show help"},
		{"globals", "/globals", "Toggle globals panel"},
		{"preview", "/preview [mode]", "Cycle preview mode"},
		{"find", "/find <term>", "Search document"},
		{"goto", "/goto <line>", "Jump to line"},
		{"eval", "/eval <expr>", "Quick evaluate"},
		{"undo", "/undo", "Undo change"},
		{"redo", "/redo", "Redo change"},
		{"wq", "/wq", "Save and quit"},
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
