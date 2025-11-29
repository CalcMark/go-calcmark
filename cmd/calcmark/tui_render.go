package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config"
	"github.com/CalcMark/go-calcmark/format/display"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	glamourStyles "github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Pure rendering functions for the TUI.
// These functions take data and return strings without side effects.

// ansiEscapePrefix is the escape sequence that starts ANSI terminal codes.
// Format: ESC (0x1B) followed by '[' starts a Control Sequence Introducer (CSI).
// Used to detect if text already contains terminal styling (colors, bold, etc).
const ansiEscapePrefix = "\x1b["

// hasANSICodes checks if text contains ANSI terminal escape codes.
func hasANSICodes(text string) bool {
	return strings.Contains(text, ansiEscapePrefix)
}

// indentLines adds a prefix to each non-empty line of text.
func indentLines(text, prefix string) string {
	lines := strings.Split(text, "\n")
	var result strings.Builder
	for i, line := range lines {
		if line != "" {
			result.WriteString(prefix)
			result.WriteString(line)
		}
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}
	return result.String()
}

// pinnedVariable represents a variable to display in the pinned panel.
type pinnedVariable struct {
	Name          string
	Value         any
	Changed       bool
	IsFrontmatter bool // Visual indicator only - shows @ prefix for frontmatter vars
}

// renderPinnedPanel renders the pinned variables panel as a string.
// Pure function: takes data, returns string.
// Uses package-level styles from config.
func renderPinnedPanel(vars []pinnedVariable) string {
	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().Bold(true).Render("📌 Pinned Variables"))
	b.WriteString("\n\n")

	if len(vars) == 0 {
		b.WriteString(styles.Hint.Render("(no variables pinned)"))
		return b.String()
	}

	for _, v := range vars {
		// Format value using display package if it's a CalcMark type
		valueStr := formatPinnedValue(v.Value)

		// Build the name with optional @ prefix for frontmatter vars
		displayName := v.Name
		if v.IsFrontmatter {
			displayName = "@" + v.Name
		}

		if v.Changed {
			// Mark changed variables with * prefix
			b.WriteString(styles.Changed.Render("* "))
			b.WriteString(styles.Var.Bold(true).Render(displayName))
			b.WriteString(" = ")
			b.WriteString(styles.Changed.Render(valueStr))
		} else {
			b.WriteString("  ") // Align with * prefix
			b.WriteString(styles.Var.Render(displayName))
			b.WriteString(" = ")
			b.WriteString(valueStr)
		}
		b.WriteString("\n")
	}

	return b.String()
}

// formatPinnedValue formats a pinned variable value using human-readable display.
func formatPinnedValue(value any) string {
	if t, ok := value.(types.Type); ok {
		return display.Format(t)
	}
	return fmt.Sprintf("%v", value)
}

// markdownRenderer is a cached glamour renderer to avoid repeated initialization
// and terminal escape sequence leaks from WithAutoStyle().
var (
	markdownRenderer     *glamour.TermRenderer
	markdownRendererOnce sync.Once
)

// buildGlamourStyle returns a glamour style based on DarkStyleConfig with
// only the colors overridden from the theme config. This preserves all the
// default formatting (margins, indentation, list formatting, etc.).
func buildGlamourStyle(theme config.ThemeConfig) ansi.StyleConfig {
	// Start with glamour's default dark style to preserve all formatting
	style := glamourStyles.DarkStyleConfig

	// Override only the colors from our theme config
	text := theme.MdText
	h1Bg := theme.MdH1Bg
	h2Bg := theme.MdH2Bg
	heading := theme.MdHeading
	link := theme.MdLink
	quote := theme.MdQuote
	code := theme.MdCode
	codeBg := theme.MdCodeBg
	bright := theme.Bright

	// Document text color
	style.Document.Color = &text

	// Heading colors
	style.Heading.Color = &heading
	style.H1.Color = &bright
	style.H1.BackgroundColor = &h1Bg
	style.H2.Color = &bright
	style.H2.BackgroundColor = &h2Bg
	// H3-H6 inherit from Heading color via the base style

	// Link colors
	style.Link.Color = &link
	style.LinkText.Color = &link

	// Block quote color
	style.BlockQuote.Color = &quote

	// Code colors
	style.Code.Color = &code
	style.Code.BackgroundColor = &codeBg
	style.CodeBlock.StyleBlock.Color = &code

	return style
}

// initMarkdownRenderer returns the cached glamour renderer.
// Uses lazy initialization with sync.Once to ensure config is loaded first.
// This avoids terminal queries during bubbletea's View() loop.
func initMarkdownRenderer() *glamour.TermRenderer {
	markdownRendererOnce.Do(func() {
		// Config must be loaded before this is called (done in tui.go)
		theme := config.Get().TUI.Theme
		r, err := glamour.NewTermRenderer(
			glamour.WithStyles(buildGlamourStyle(theme)),
			glamour.WithColorProfile(termenv.TrueColor),
			glamour.WithWordWrap(70),
		)
		if err == nil {
			markdownRenderer = r
		}
	})
	return markdownRenderer
}

// renderMarkdownPreviewContent renders markdown content using glamour.
// Pure function: takes content and width, returns rendered string.
// Uses cached renderer to avoid terminal escape sequence leaks.
func renderMarkdownPreviewContent(content string, _ int) string {
	if strings.TrimSpace(content) == "" {
		return styles.Hint.Render("(preview will appear here)")
	}

	r := initMarkdownRenderer()
	if r == nil {
		// Fallback: just return the content as-is if renderer fails
		return content
	}

	rendered, err := r.Render(content)
	if err != nil {
		// Fallback: return content with error note
		return content
	}

	return strings.TrimSpace(rendered)
}

// renderHistoryItems renders output history as a string.
// Pure function: takes history items and display constraints, returns string.
// Uses package-level styles from config.
func renderHistoryItems(items []outputHistoryItem, maxLines int) string {
	if len(items) == 0 {
		return ""
	}

	var b strings.Builder

	// Calculate which items fit in maxLines
	historyLines := 0
	startIdx := 0
	for i := len(items) - 1; i >= 0; i-- {
		linesForItem := 1 // input line
		if items[i].output != "" {
			linesForItem++ // output line
		}
		if historyLines+linesForItem > maxLines {
			startIdx = i + 1
			break
		}
		historyLines += linesForItem
	}

	for i := startIdx; i < len(items); i++ {
		item := items[i]

		// Show input
		b.WriteString(styles.Prompt.Render("> "))
		b.WriteString(item.input)
		b.WriteString("\n")

		// Show output if any
		if item.output != "" {
			if item.isError {
				b.WriteString(styles.ErrorOutput.Render("  " + item.output))
			} else if hasANSICodes(item.output) {
				// Pre-styled output - just indent, don't apply additional style
				b.WriteString(indentLines(item.output, "  "))
			} else {
				b.WriteString(styles.Output.Render("  " + item.output))
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

// helpCommand represents a slash command for help display.
type helpCommand struct {
	cmd  string
	desc string
}

// renderHelpText renders the /help output with proper formatting for the given width.
// Uses lipgloss for consistent column alignment and word wrapping.
func renderHelpText(width int) string {
	commands := []helpCommand{
		{"/pin [var]", "Pin all or specific variable"},
		{"/unpin [var]", "Unpin all or specific variable"},
		{"/open <file>", "Load a CalcMark file"},
		{"/save <file>", "Save as CalcMark (.cm)"},
		{"/output <file>", "Export with results (.html, .md, .json)"},
		{"/md", "Enter multi-line markdown mode"},
		{"/quit, /q", "Exit the REPL"},
		{"/help [topic]", "Show help or search for a topic"},
	}

	var b strings.Builder

	// Calculate column width - command column is fixed, description wraps
	cmdWidth := 16
	descWidth := max(width-cmdWidth-4, 20) // Leave some margin, minimum 20

	// Use Var style for commands (inherits color), add fixed width
	cmdStyle := styles.Var.Width(cmdWidth)
	descStyle := lipgloss.NewStyle().Width(descWidth)

	for _, cmd := range commands {
		b.WriteString(cmdStyle.Render(cmd.cmd))
		b.WriteString(descStyle.Render(cmd.desc))
		b.WriteString("\n")
	}

	// Add topics section
	b.WriteString("\n")
	b.WriteString(styles.Header.Render("Help topics:"))
	b.WriteString("\n")

	topics := "functions, units, dates, network, storage, compression, keywords, operators"
	topicsStyle := lipgloss.NewStyle().Width(width - 2).PaddingLeft(2)
	b.WriteString(topicsStyle.Render(topics))
	b.WriteString("\n")

	b.WriteString("  ")
	b.WriteString(styles.Hint.Render("Example: /help meter, /help avg"))

	return b.String()
}

// renderHelpLine renders context-sensitive help text.
// Pure function: takes state flags and width, returns string.
func renderHelpLine(slashMode bool, width int) string {
	if slashMode {
		if width < 80 {
			return "/help │ /quit │ Esc to exit"
		}
		return "/help for commands │ /quit or Ctrl+C to exit │ Esc to cancel"
	}

	// Normal mode
	if width < 80 {
		return "↑↓ History │ PgUp/Dn Scroll │ / Cmds"
	}
	return "↑↓ History │ PgUp/PgDn scroll pinned │ / for commands │ /help for features"
}
