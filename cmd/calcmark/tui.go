package main

import (
	"fmt"
	"strings"

	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func init() {
	// CRITICAL: Set color profile and background explicitly to avoid terminal queries.
	// Terminal color detection sends OSC sequences (like OSC 11 for background color)
	// that can leak into input buffers, causing garbage text like "]11;rgb:ffff/ffff/ffff".
	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(true)
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			Margin(1, 0)

	pinnedPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#874BFD")).
				Padding(0, 1).
				Margin(1, 0)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			Margin(1, 0)
)

// model represents the TUI state
type model struct {
	doc           *document.Document
	eval          *implDoc.Evaluator  // Persistent evaluator (holds variable environment)
	input         textinput.Model     // Single-line input for calc expressions
	mdInput       textarea.Model      // Multi-line input for markdown mode
	pinnedVars    map[string]bool     // Which variables are pinned
	changedVars   map[string]bool     // Variables that changed in last update
	history       []string            // Command history (for ↑↓ navigation)
	outputHistory []outputHistoryItem // Display history (commands + results)
	historyIdx    int                 // Current position in history (-1 = not browsing)
	width         int
	height        int
	err           error
	lastInputted  string
	quitting      bool
	markdownMode  bool // In multi-line markdown mode
	slashMode     bool // In slash command mode (triggered by / key)
}

// outputHistoryItem represents a command and its result for display
type outputHistoryItem struct {
	input   string // The command entered
	output  string // The result/output
	isError bool   // Whether this was an error
}

// newTUIModel creates initial TUI model
func newTUIModel(doc *document.Document) model {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "Enter CalcMark expression or /command"
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 50

	// Create textarea for markdown mode
	ta := textarea.New()
	ta.Placeholder = "Enter markdown... (Esc to finish)"
	ta.ShowLineNumbers = false
	ta.CharLimit = 10000

	// Create persistent evaluator and evaluate the loaded document
	eval := implDoc.NewEvaluator()
	_ = eval.Evaluate(doc) // Ignore error - blocks will show their own errors

	m := model{
		doc:           doc,
		eval:          eval,
		input:         ti,
		mdInput:       ta,
		pinnedVars:    make(map[string]bool),
		changedVars:   make(map[string]bool),
		history:       []string{},
		outputHistory: []outputHistoryItem{},
		historyIdx:    -1,
		width:         80,
		height:        24,
		markdownMode:  false,
		slashMode:     false,
	}

	// Auto-pin all variables in the loaded document
	// Display order will be determined by document block order (via UUIDs)
	for _, node := range doc.GetBlocks() {
		if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
			for _, varName := range calcBlock.Variables() {
				m.pinnedVars[varName] = true
			}

			// Add blocks to output history for display
			for _, line := range calcBlock.Source() {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" {
					var output string
					if calcBlock.LastValue() != nil {
						output = fmt.Sprintf("= %v", calcBlock.LastValue())
					}
					m.outputHistory = append(m.outputHistory, outputHistoryItem{
						input:   trimmed,
						output:  output,
						isError: calcBlock.Error() != nil,
					})
				}
			}
		}
	}

	// Populate history with all statements from the document
	// This allows users to scroll through the loaded file with ↑/↓
	for _, node := range doc.GetBlocks() {
		if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
			// Add each source line to history
			for _, line := range calcBlock.Source() {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" {
					m.history = append(m.history, trimmed)
				}
			}
		}
	}

	// Reset history index to -1 (start at end, ready for new input)
	m.historyIdx = -1

	return m
}

// Init implements tea.Model
func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			// CRITICAL: Always allow Ctrl+C to exit, regardless of mode
			m.quitting = true
			return m, tea.Quit

		case tea.KeyCtrlD:
			// Also allow Ctrl+D to exit (common Unix behavior)
			m.quitting = true
			return m, tea.Quit

		case tea.KeyEsc:
			// ESC exits markdown mode and saves the block
			if m.markdownMode {
				content := m.mdInput.Value()
				if strings.TrimSpace(content) != "" {
					m = m.insertMarkdownBlock()
				}
				m.markdownMode = false
				m.mdInput.Reset()
				m.input.Focus()
				return m, nil
			}
			// Esc exits slash mode
			if m.slashMode {
				m.slashMode = false
				m.input.SetValue("")
				m.input.Prompt = "> "
				return m, nil
			}
			// In normal mode, Esc does nothing (prevents accidental exits)
			return m, nil

		case tea.KeyRunes:
			// Check if user typed '/' in normal mode (not markdown mode)
			if !m.markdownMode && !m.slashMode && len(msg.Runes) == 1 && msg.Runes[0] == '/' && m.input.Value() == "" {
				// Enter slash command mode
				m.slashMode = true
				m.input.SetValue("")
				m.input.Prompt = "/ "
				return m, nil
			}

		case tea.KeyUp:
			// Don't handle history in markdown mode - textarea handles it
			if m.markdownMode {
				break
			}
			// Navigate history backward
			if len(m.history) > 0 {
				if m.historyIdx == -1 {
					m.historyIdx = len(m.history) - 1
				} else if m.historyIdx > 0 {
					m.historyIdx--
				}
				if m.historyIdx >= 0 && m.historyIdx < len(m.history) {
					m.input.SetValue(m.history[m.historyIdx])
				}
			}
			return m, nil

		case tea.KeyDown:
			// Don't handle history in markdown mode - textarea handles it
			if m.markdownMode {
				break
			}
			// Navigate history forward
			if m.historyIdx != -1 {
				m.historyIdx++
				if m.historyIdx >= len(m.history) {
					m.historyIdx = -1
					m.input.SetValue("")
				} else {
					m.input.SetValue(m.history[m.historyIdx])
				}
			}
			return m, nil

		case tea.KeyEnter:
			// In markdown mode, let textarea handle enter for newlines
			if m.markdownMode {
				break
			}
			m = m.handleInput()
			m.input.SetValue("")
			m.historyIdx = -1 // Reset history browsing
			if m.quitting {
				return m, tea.Quit
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = m.width - 6
		// Update textarea size for markdown mode
		m.mdInput.SetWidth(m.width/2 - 4)
		m.mdInput.SetHeight(m.height - 10)
	}

	// Forward events to the appropriate input component
	if m.markdownMode {
		m.mdInput, cmd = m.mdInput.Update(msg)
	} else {
		m.input, cmd = m.input.Update(msg)
	}
	return m, cmd
}

// View implements tea.Model
func (m model) View() string {
	var b strings.Builder

	// Title
	title := titleStyle.Render("CalcMark REPL")
	b.WriteString(title)
	b.WriteString("\n")

	// Markdown mode: split view with editor on left, preview on right
	if m.markdownMode {
		b.WriteString(m.renderMarkdownMode())
		return b.String()
	}

	// Mode indicator for slash mode
	if m.slashMode {
		modeIndicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Render("💬 COMMAND MODE - Type command or Esc to exit")
		b.WriteString(modeIndicator)
		b.WriteString("\n")
	}

	// Main content area (left side)
	var leftContent strings.Builder

	// Scrollback history (above input)
	historyDisplay := m.renderHistory()
	leftContent.WriteString(historyDisplay)

	// Separator
	if len(m.outputHistory) > 0 {
		leftContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#444")).Render("─────────────────────────────────────────"))
		leftContent.WriteString("\n")
	}

	// Input (textinput handles its own prompt)
	leftContent.WriteString(m.input.View())
	leftContent.WriteString("\n")

	// Error display
	if m.err != nil {
		leftContent.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		leftContent.WriteString("\n")
	}

	// Help text
	help := m.renderHelp()
	leftContent.WriteString(helpStyle.Render(help))

	// Layout: Always show pinned panel on the right
	// Right panel: Pinned variables (40% width)
	rightWidth := max(m.width*2/5, 25)

	pinnedContent := m.renderPinnedVars()
	rightPanel := pinnedPanelStyle.
		Width(rightWidth - 4).
		Height(m.height - 10).
		Render(pinnedContent)

	// Left panel: Main content (60% width)
	leftWidth := max(m.width-rightWidth, 40)

	leftPanel := lipgloss.NewStyle().
		Width(leftWidth).
		Render(leftContent.String())

	// Join horizontally
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	b.WriteString(mainContent)

	return b.String()
}

// renderMarkdownMode renders the split markdown editor view.
// Left side: raw markdown editor, Right side: glamour-rendered preview.
func (m model) renderMarkdownMode() string {
	halfWidth := m.width / 2

	// Header with mode indicator and help link
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF9800")).
		Bold(true).
		Render("📝 MARKDOWN MODE - Esc to save and exit")
	helpLink := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		Render("  Help: https://commonmark.org/help/")
	headerLine := header + helpLink + "\n\n"

	// Left panel: markdown input
	leftTitle := lipgloss.NewStyle().Bold(true).Render("Edit")
	leftPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF9800")).
		Padding(0, 1).
		Width(halfWidth - 4).
		Height(m.height - 8).
		Render(leftTitle + "\n" + m.mdInput.View())

	// Right panel: rendered preview
	rightTitle := lipgloss.NewStyle().Bold(true).Render("Preview")
	preview := m.renderMarkdownPreview()
	rightPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#874BFD")).
		Padding(0, 1).
		Width(halfWidth - 4).
		Height(m.height - 8).
		Render(rightTitle + "\n" + preview)

	return headerLine + lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

// renderMarkdownPreview renders the current markdown content using glamour.
// Delegates to pure renderMarkdownPreviewContent function.
func (m model) renderMarkdownPreview() string {
	return renderMarkdownPreviewContent(m.mdInput.Value(), m.width/2-8)
}

// handleInput processes user input
func (m model) handleInput() model {
	input := strings.TrimSpace(m.input.Value())
	m.err = nil

	if input == "" {
		return m
	}

	// Slash mode: treat input as command
	if m.slashMode {
		m = m.handleCommand(input)
		m.slashMode = false
		m.input.Prompt = "> " // Reset prompt after command
		return m
	}

	// Normal mode: CalcMark expression
	m.lastInputted = input
	m.changedVars = make(map[string]bool) // Reset changed tracking

	// Add to history (skip duplicates of last command)
	if len(m.history) == 0 || m.history[len(m.history)-1] != input {
		m.history = append(m.history, input)
		// Limit history to 100 commands
		if len(m.history) > 100 {
			m.history = m.history[1:]
		}
	}

	// Evaluate expression
	result := m.evaluateExpression(input)

	// Add to output history with result
	var output string
	if result.err != nil {
		output = result.err.Error()
		result.outputHistory = append(result.outputHistory, outputHistoryItem{
			input:   input,
			output:  output,
			isError: true,
		})
	} else {
		// Find the last value
		blocks := result.doc.GetBlocks()
		if len(blocks) > 0 {
			if calcBlock, ok := blocks[len(blocks)-1].Block.(*document.CalcBlock); ok {
				if calcBlock.LastValue() != nil {
					output = fmt.Sprintf("= %v", calcBlock.LastValue())
					result.outputHistory = append(result.outputHistory, outputHistoryItem{
						input:   input,
						output:  output,
						isError: false,
					})
				}
			}
		}
	}

	return result
}

// handleCommand processes REPL commands
func (m model) handleCommand(cmd string) model {
	// Strip leading / if present (for slash mode compatibility)
	cmd = strings.TrimPrefix(cmd, "/")

	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return m
	}

	switch parts[0] {
	case "pin":
		if len(parts) == 1 {
			// Pin all variables
			m.pinnedVars = make(map[string]bool)
			for _, node := range m.doc.GetBlocks() {
				if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
					for _, varName := range calcBlock.Variables() {
						m.pinnedVars[varName] = true
					}
				}
			}
		} else {
			// Pin specific variable
			varName := parts[1]
			m.pinnedVars[varName] = true
		}

	case "unpin":
		if len(parts) == 1 {
			// Unpin all variables
			m.pinnedVars = make(map[string]bool)
		} else {
			// Unpin specific variable
			varName := parts[1]
			delete(m.pinnedVars, varName)
		}

	case "open":
		if len(parts) < 2 {
			m.err = fmt.Errorf("usage: /open <filename>")
			return m
		}

		filename := parts[1]
		m = m.openFile(filename)

	case "md", "markdown":
		// Enter multi-line markdown mode with live preview
		m.markdownMode = true
		m.mdInput.Reset()
		m.mdInput.Focus()
		m.input.Blur()

	case "save":
		if len(parts) < 2 {
			m.err = fmt.Errorf("usage: /save <filename.cm>")
			return m
		}
		m = m.saveFile(parts[1])

	case "output":
		if len(parts) < 2 {
			m.err = fmt.Errorf("usage: /output <filename.html|.md|.json>")
			return m
		}
		m = m.outputFile(parts[1])

	case "quit", "q":
		m.quitting = true
		return m

	case "help", "h", "?":
		// Add help to output history
		helpText := `/pin          Pin all variables
/pin <var>    Pin a specific variable
/unpin        Unpin all variables
/unpin <var>  Unpin a specific variable
/open <file>  Load a CalcMark file
/save <file>  Save as CalcMark (.cm)
/output <file> Export with results (.html, .md, .json)
/md           Enter multi-line markdown mode
/quit or /q   Exit the REPL
/help or /?   Show this help`
		m.outputHistory = append(m.outputHistory, outputHistoryItem{
			input:   "/help",
			output:  helpText,
			isError: false,
		})
		return m

	default:
		m.err = fmt.Errorf("unknown command: /%s (type /help for available commands)", parts[0])
	}

	return m
}

// renderHistory renders the scrollback history of commands and results.
// Delegates to pure renderHistoryItems function.
func (m model) renderHistory() string {
	// Reserve space for: title(2) + mode(1) + separator(1) + input(2) + error(1) + help(2) = ~9 lines
	maxHistoryLines := max(m.height-12, 5)
	return renderHistoryItems(m.outputHistory, maxHistoryLines)
}

// evaluateExpression adds and evaluates a new expression
func (m model) evaluateExpression(expr string) model {
	blocks := m.doc.GetBlocks()

	var afterID string
	if len(blocks) > 0 {
		afterID = blocks[len(blocks)-1].ID

		// Insert new calc block
		result, err := m.doc.InsertBlock(afterID, document.BlockCalculation, []string{expr})
		if err != nil {
			m.err = fmt.Errorf("insert block: %w", err)
			return m
		}

		// Evaluate using persistent evaluator (maintains variable environment)
		err = m.eval.EvaluateBlock(m.doc, result.ModifiedBlockID)
		if err != nil {
			m.err = err
			return m
		}

		// Track which variables changed and auto-pin new variables
		for _, id := range result.AffectedBlockIDs {
			node, ok := m.doc.GetBlock(id)
			if !ok {
				continue
			}
			if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
				for _, varName := range calcBlock.Variables() {
					m.changedVars[varName] = true
					// Auto-pin new variables
					m.pinnedVars[varName] = true
				}
			}
		}
	} else {
		// First block - create new document
		doc, err := document.NewDocument(expr + "\n")
		if err != nil {
			m.err = fmt.Errorf("parse: %w", err)
			return m
		}

		// Create fresh evaluator for the new document
		m.eval = implDoc.NewEvaluator()
		err = m.eval.Evaluate(doc)
		if err != nil {
			m.err = err
			return m
		}

		m.doc = doc

		// Auto-pin variables from the first block
		for _, node := range doc.GetBlocks() {
			if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
				for _, varName := range calcBlock.Variables() {
					m.changedVars[varName] = true
					m.pinnedVars[varName] = true
				}
			}
		}
	}

	return m
}

// renderPinnedVars renders the pinned variables panel content.
// Delegates to pure renderPinnedPanel function.
func (m model) renderPinnedVars() string {
	// Collect pinned variables in definition order with their latest values.
	names, values := m.collectPinnedVariables()

	// Convert to pinnedVariable structs for pure rendering function
	vars := make([]pinnedVariable, 0, len(names))
	for _, name := range names {
		vars = append(vars, pinnedVariable{
			Name:    name,
			Value:   values[name],
			Changed: m.changedVars[name],
		})
	}

	return renderPinnedPanel(vars)
}

// collectPinnedVariables returns pinned variables in first-definition order
// with their current values from the interpreter environment.
// Each variable appears only once.
func (m model) collectPinnedVariables() ([]string, map[string]any) {
	order := []string{}
	seen := make(map[string]bool)

	// Collect variables in document order (first definition wins for ordering)
	for _, node := range m.doc.GetBlocks() {
		if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
			for _, varName := range calcBlock.Variables() {
				if !m.pinnedVars[varName] || seen[varName] {
					continue
				}
				seen[varName] = true
				order = append(order, varName)
			}
		}
	}

	// Get current values from the interpreter's environment
	values := make(map[string]any)
	if m.eval != nil {
		env := m.eval.GetEnvironment()
		for _, varName := range order {
			if val, ok := env.Get(varName); ok {
				values[varName] = val
			}
		}
	}

	return order, values
}

// renderHelp returns context-sensitive help text.
// Delegates to pure renderHelpLine function.
func (m model) renderHelp() string {
	return renderHelpLine(m.slashMode, m.width)
}
