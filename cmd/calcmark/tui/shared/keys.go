package shared

import "charm.land/bubbles/v2/key"

// KeyMap defines key bindings for the TUI.
// Centralized here for consistency across all modes.
type KeyMap struct {
	// Global keys (work in all modes)
	Quit key.Binding
	Help key.Binding

	// Navigation keys
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	WordLeft  key.Binding
	WordRight key.Binding
	LineStart key.Binding
	LineEnd   key.Binding
	PageUp    key.Binding
	PageDown  key.Binding
	Home      key.Binding
	End       key.Binding

	// Action keys
	Enter  key.Binding
	Escape key.Binding
	Tab    key.Binding

	// Editor-specific keys (used in editor mode)
	Edit       key.Binding
	Save       key.Binding
	Open       key.Binding
	Undo       key.Binding
	Redo       key.Binding
	InsertLine key.Binding
	DeleteLine key.Binding
	NewLine    key.Binding
	Backspace  key.Binding
	DeleteWord key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("Ctrl+C", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("f1", "ctrl+h"),
			key.WithHelp("Ctrl+H/F1", "help/commands"),
		),
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("Up", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("Down", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("Left", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("Right", "right"),
		),
		WordLeft: key.NewBinding(
			key.WithKeys("ctrl+left", "alt+left"),
			key.WithHelp("Ctrl/Alt+←", "word left"),
		),
		WordRight: key.NewBinding(
			key.WithKeys("ctrl+right", "alt+right"),
			key.WithHelp("Ctrl/Alt+→", "word right"),
		),
		LineStart: key.NewBinding(
			key.WithKeys("ctrl+a", "home"),
			key.WithHelp("Ctrl+A/Home", "line start"),
		),
		LineEnd: key.NewBinding(
			key.WithKeys("ctrl+e", "end"),
			key.WithHelp("Ctrl+E/End", "line end"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("PgUp", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("PgDn", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("ctrl+home"),
			key.WithHelp("Ctrl+Home", "go to top"),
		),
		End: key.NewBinding(
			key.WithKeys("ctrl+end"),
			key.WithHelp("Ctrl+End", "go to bottom"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "new line"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("Esc", "close/cancel"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "complete"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e", "i"),
			key.WithHelp("e/i", "edit"),
		),
		Save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("Ctrl+S", "save"),
		),
		Open: key.NewBinding(
			key.WithKeys("ctrl+o"),
			key.WithHelp("Ctrl+O", "open"),
		),
		Undo: key.NewBinding(
			key.WithKeys("ctrl+z"),
			key.WithHelp("Ctrl+Z", "undo"),
		),
		Redo: key.NewBinding(
			key.WithKeys("ctrl+y"),
			key.WithHelp("Ctrl+Y", "redo"),
		),
		InsertLine: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "insert line below"),
		),
		DeleteLine: key.NewBinding(
			key.WithKeys("ctrl+k"),
			key.WithHelp("Ctrl+K", "delete line"),
		),
		NewLine: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "new line"),
		),
		Backspace: key.NewBinding(
			key.WithKeys("backspace"),
			key.WithHelp("Backspace", "delete char"),
		),
		DeleteWord: key.NewBinding(
			key.WithKeys("ctrl+backspace"),
			key.WithHelp("Ctrl+Backspace", "delete word"),
		),
	}
}

// ShortHelp returns key bindings to show in short help.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns key bindings to show in full help.
// Organized by category: Navigation, Word Navigation, Editing, File, Other.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		// Navigation
		{k.Up, k.Down, k.Left, k.Right},
		{k.WordLeft, k.WordRight, k.LineStart, k.LineEnd},
		{k.PageUp, k.PageDown, k.Home, k.End},
		// Editing
		{k.NewLine, k.Backspace, k.DeleteWord, k.DeleteLine},
		// File
		{k.Save, k.Undo, k.Redo},
		// Other
		{k.Help, k.Escape, k.Quit},
	}
}
