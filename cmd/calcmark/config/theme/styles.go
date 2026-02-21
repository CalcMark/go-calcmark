package theme

import "github.com/charmbracelet/lipgloss"

// Pre-built styles using the semantic palette.
// These should be used by TUI components instead of hardcoded colors.
var (
	// Text styles
	TextStyle = lipgloss.NewStyle().
			Foreground(Text)

	MutedStyle = lipgloss.NewStyle().
			Foreground(TextMuted)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(Header).
			Bold(true)

	// Result styles (preview pane)
	ResultStyle = lipgloss.NewStyle().
			Foreground(Result)

	ResultNameStyle = lipgloss.NewStyle().
			Foreground(ResultMuted)

	// Error style
	ErrorStyle = lipgloss.NewStyle().
			Foreground(Error)

	// Hint style (autosuggestions)
	HintStyle = lipgloss.NewStyle().
			Foreground(Hint)

	// Command style (REPL command mode)
	CommandStyle = lipgloss.NewStyle().
			Foreground(Command)

	// Success style
	SuccessStyle = lipgloss.NewStyle().
			Foreground(Success)

	// Cursor line (subtle highlight)
	CursorLineStyle = lipgloss.NewStyle().
			Background(Cursor)

	// Selection
	SelectionStyle = lipgloss.NewStyle().
			Background(Selection)

	// Borders
	BorderStyle = lipgloss.NewStyle().
			BorderForeground(Border)

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Background(StatusBg).
			Foreground(StatusFg).
			Padding(0, 1)
)

// PaneBorder returns a border style for panes.
func PaneBorder() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Border)
}
