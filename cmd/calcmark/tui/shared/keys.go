package shared

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines key bindings for the TUI.
// Centralized here for consistency across all modes.
type KeyMap struct {
	// Global keys (work in all modes)
	Quit      key.Binding
	ForceQuit key.Binding
	Help      key.Binding

	// Navigation keys
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding

	// Action keys
	Enter  key.Binding
	Escape key.Binding
	Tab    key.Binding

	// Mode switching
	SlashCommand key.Binding

	// Editor-specific keys (used in editor mode)
	Edit       key.Binding
	Save       key.Binding
	Open       key.Binding
	Undo       key.Binding
	Redo       key.Binding
	InsertLine key.Binding
	DeleteLine key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		ForceQuit: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "force quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("home/g", "go to top"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("end/G", "go to bottom"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel/exit mode"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "complete"),
		),
		SlashCommand: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "command mode"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e", "i"),
			key.WithHelp("e/i", "edit"),
		),
		Save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "save"),
		),
		Open: key.NewBinding(
			key.WithKeys("ctrl+o"),
			key.WithHelp("ctrl+o", "open"),
		),
		Undo: key.NewBinding(
			key.WithKeys("u", "ctrl+z"),
			key.WithHelp("u", "undo"),
		),
		Redo: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "redo"),
		),
		InsertLine: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "insert line below"),
		),
		DeleteLine: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("dd", "delete line"),
		),
	}
}

// ShortHelp returns key bindings to show in short help.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns key bindings to show in full help.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.PageUp, k.PageDown, k.Home, k.End},
		{k.Enter, k.Escape, k.Tab},
		{k.Help, k.Quit},
	}
}
