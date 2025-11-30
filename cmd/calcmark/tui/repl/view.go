package repl

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/shared"
	"github.com/charmbracelet/lipgloss"
)

// View implements tea.Model.
// The Simple REPL is a minimal, scrolling history view.
// No split panes, no pinned panel - just input → output in a list.
func (m Model) View() string {
	var b strings.Builder

	// Title bar
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Width(m.width)
	b.WriteString(titleStyle.Render("CalcMark REPL"))
	b.WriteString("\n")

	// Calculate available height for history
	// 1 line for title, 2 lines for input area, 1 line for help footer
	historyHeight := m.height - 4 - 2
	if historyHeight < 3 {
		historyHeight = 3
	}

	// Mode indicator for slash mode (takes 1 line if shown)
	if m.inputMode == shared.InputSlash {
		modeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")).
			Italic(true)
		b.WriteString(modeStyle.Render("  COMMAND MODE - Type command or Esc to exit"))
		b.WriteString("\n")
		historyHeight--
	}

	// Scrolling history area
	historyContent := m.renderScrollingHistory(historyHeight)
	b.WriteString(historyContent)

	// Input line
	b.WriteString(m.input.View())
	b.WriteString("\n")

	// Suggestions (if any)
	if m.inputMode == shared.InputSlash {
		suggestions := GetSlashCommandSuggestions(m.input.Value(), m.slashCommands)
		if len(suggestions) > 0 {
			b.WriteString(RenderSlashSuggestions(suggestions, m.styles))
			b.WriteString("\n")
		}
	} else {
		varCompletions := m.getVariableCompletions()
		if len(varCompletions) > 0 {
			b.WriteString(m.styles.Hint.Render("Hints: " + strings.Join(varCompletions, " | ")))
			b.WriteString("\n")
		}
	}

	// Error display (if any)
	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		b.WriteString(errorStyle.Render(fmt.Sprintf("⚠ %v", m.err)))
		b.WriteString("\n")
	}

	// Separator line
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	separator := strings.Repeat("─", m.width)
	b.WriteString(separatorStyle.Render(separator))
	b.WriteString("\n")

	// Help footer
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(m.width)
	helpText := RenderHelpLine(m.inputMode == shared.InputSlash, m.width)
	b.WriteString(helpStyle.Render(helpText))

	return b.String()
}

// renderScrollingHistory renders the scrolling output history.
func (m Model) renderScrollingHistory(maxLines int) string {
	var b strings.Builder

	if len(m.outputHistory) == 0 {
		// Show welcome message for empty REPL
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
		b.WriteString(emptyStyle.Render("  Type an expression and press Enter"))
		b.WriteString("\n")
		b.WriteString(emptyStyle.Render("  Example: salary = $85000"))
		b.WriteString("\n")
		b.WriteString(emptyStyle.Render("           monthly = salary / 12"))
		b.WriteString("\n\n")
		return b.String()
	}

	// Calculate how many history entries we can show
	// Each entry takes 2 lines: input + output
	visibleEntries := m.calculateVisibleEntries(maxLines)

	// Render visible entries
	for _, entry := range visibleEntries {
		// Input line with prompt
		promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
		b.WriteString(promptStyle.Render("> "))
		b.WriteString(entry.Input)
		b.WriteString("\n")

		// Output line (indented)
		if entry.Output != "" {
			outputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
			if entry.IsError {
				outputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
				b.WriteString("  ")
				b.WriteString(outputStyle.Render("⚠ " + entry.Output))
			} else {
				b.WriteString("  ")
				b.WriteString(outputStyle.Render(entry.Output))
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

// calculateVisibleEntries returns the entries that fit in maxLines.
func (m Model) calculateVisibleEntries(maxLines int) []shared.HistoryEntry {
	if len(m.outputHistory) == 0 {
		return nil
	}

	// Calculate lines needed per entry
	var entries []shared.HistoryEntry
	linesUsed := 0

	// Work backwards from most recent
	for i := len(m.outputHistory) - 1; i >= 0; i-- {
		entry := m.outputHistory[i]
		linesNeeded := 1 // Input line
		if entry.Output != "" {
			linesNeeded++ // Output line
		}

		if linesUsed+linesNeeded > maxLines {
			break
		}

		entries = append([]shared.HistoryEntry{entry}, entries...)
		linesUsed += linesNeeded
	}

	return entries
}

// getVariableCompletions returns matching variable names for tab completion.
func (m Model) getVariableCompletions() []string {
	input := m.input.Value()
	if input == "" {
		return nil
	}

	lastToken := extractLastToken(input)
	if len(lastToken) < 2 {
		return nil
	}

	env := m.eval.GetEnvironment()
	allVars := env.GetAllVariables()

	var matches []string
	prefix := strings.ToLower(lastToken)
	for varName := range allVars {
		if strings.HasPrefix(strings.ToLower(varName), prefix) && varName != lastToken {
			matches = append(matches, varName+" [Tab]")
		}
	}

	if len(matches) > 3 {
		matches = matches[:3]
	}

	return matches
}

// Pure rendering functions below

// RenderHelpLine renders context-sensitive help text.
func RenderHelpLine(slashMode bool, width int) string {
	if slashMode {
		return "↑↓ history │ /help │ /vars │ /quit │ Esc cancel"
	}
	return "↑↓ history │ /help │ /vars │ /clear │ /quit"
}

// RenderHelpText renders the /help output.
func RenderHelpText(width int) string {
	help := `
CalcMark REPL Help

EXPRESSIONS
  salary = $85000      Define a variable with currency
  monthly = salary/12  Reference other variables
  5 * 10 + 2          Simple arithmetic
  sqrt(144)           Built-in functions

COMMANDS
  /help, /h, /?       Show this help
  /vars               List all defined variables
  /clear              Clear screen (keep variables)
  /reset              Clear everything
  /quit, /q           Exit REPL
  /edit [file]        Switch to editor mode

KEYBOARD
  ↑/↓                 Navigate command history
  Tab                 Autocomplete variable names
  Ctrl-C              Exit
`
	return strings.TrimSpace(help)
}

// GetSlashCommandSuggestions returns matching slash commands.
func GetSlashCommandSuggestions(input string, commands []shared.SlashCommand) []shared.SlashCommand {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return commands
	}

	var matches []shared.SlashCommand
	for _, cmd := range commands {
		if strings.HasPrefix(cmd.Name, input) {
			matches = append(matches, cmd)
		}
	}

	if len(matches) > 4 {
		matches = matches[:4]
	}

	return matches
}

// RenderSlashSuggestions renders slash command suggestions.
func RenderSlashSuggestions(suggestions []shared.SlashCommand, styles config.Styles) string {
	if len(suggestions) == 0 {
		return ""
	}

	var parts []string
	for _, cmd := range suggestions {
		parts = append(parts, fmt.Sprintf("%s (%s)", cmd.Syntax, cmd.Description))
	}

	return styles.Hint.Render(strings.Join(parts, " │ "))
}

// RenderHistoryItems renders history items as a string.
// This is kept for test compatibility.
func RenderHistoryItems(items []shared.HistoryEntry, maxLines int, styles config.Styles) string {
	if len(items) == 0 {
		return ""
	}

	var b strings.Builder

	// Calculate which items fit
	historyLines := 0
	startIdx := 0
	for i := len(items) - 1; i >= 0; i-- {
		linesForItem := 1
		if items[i].Output != "" {
			linesForItem++
		}
		if historyLines+linesForItem > maxLines {
			startIdx = i + 1
			break
		}
		historyLines += linesForItem
	}

	for i := startIdx; i < len(items); i++ {
		item := items[i]

		b.WriteString(styles.Prompt.Render("> "))
		b.WriteString(item.Input)
		b.WriteString("\n")

		if item.Output != "" {
			if item.IsError {
				b.WriteString(styles.ErrorOutput.Render("  " + item.Output))
			} else {
				b.WriteString(styles.Output.Render("  " + item.Output))
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

// RenderPinnedPanel is kept for backwards compatibility with tests.
// The Simple REPL doesn't use a pinned panel anymore.
func RenderPinnedPanel(vars []shared.PinnedVar, styles config.Styles) string {
	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().Bold(true).Render("📌 Pinned Variables"))
	b.WriteString("\n\n")

	if len(vars) == 0 {
		b.WriteString(styles.Hint.Render("(no variables pinned)"))
		return b.String()
	}

	for _, v := range vars {
		displayName := v.Name
		if v.IsFrontmatter {
			displayName = "@" + v.Name
		}

		valueStr := "?"
		if v.Value != nil {
			valueStr = fmt.Sprintf("%v", v.Value)
		}

		if v.Changed {
			b.WriteString(styles.Changed.Render("* "))
			b.WriteString(styles.Var.Bold(true).Render(displayName))
			b.WriteString(" = ")
			b.WriteString(styles.Changed.Render(valueStr))
		} else {
			b.WriteString("  ")
			b.WriteString(styles.Var.Render(displayName))
			b.WriteString(" = ")
			b.WriteString(valueStr)
		}
		b.WriteString("\n")
	}

	return b.String()
}
