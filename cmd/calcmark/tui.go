package main

import (
	"fmt"
	"strings"

	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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
	input         textinput.Model
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
	markdownMode  bool     // In multi-line markdown mode
	markdownLines []string // Accumulated markdown lines
	slashMode     bool     // In slash command mode (triggered by / key)
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

	m := model{
		doc:           doc,
		input:         ti,
		pinnedVars:    make(map[string]bool),
		changedVars:   make(map[string]bool),
		history:       []string{},
		outputHistory: []outputHistoryItem{},
		historyIdx:    -1,
		width:         80,
		height:        24,
		markdownMode:  false,
		markdownLines: []string{},
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
			m.quitting = true
			return m, tea.Quit

		case tea.KeyEsc:
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
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View implements tea.Model
func (m model) View() string {
	var b strings.Builder

	// Title
	title := titleStyle.Render("CalcMark REPL")
	b.WriteString(title)
	b.WriteString("\n")

	// Mode indicator
	var modeIndicator string
	if m.markdownMode {
		modeIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF9800")).
			Bold(true).
			Render(fmt.Sprintf("📝 MARKDOWN MODE (%d lines) - Type /end to finish", len(m.markdownLines)))
		b.WriteString(modeIndicator)
		b.WriteString("\n")
	} else if m.slashMode {
		modeIndicator = lipgloss.NewStyle().
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

	// Layout: If pinned vars exist, show side-by-side
	if len(m.pinnedVars) > 0 {
		// Right panel: Pinned variables (50% width)
		rightWidth := max(m.width/2, 30)

		pinnedContent := m.renderPinnedVars()
		rightPanel := pinnedPanelStyle.
			Width(rightWidth - 4).
			Height(m.height - 10).
			Render(pinnedContent)

		// Left panel: Main content (50% width)
		leftWidth := max(m.width-rightWidth, 30)

		leftPanel := lipgloss.NewStyle().
			Width(leftWidth).
			Render(leftContent.String())

		// Join horizontally
		mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
		b.WriteString(mainContent)
	} else {
		// No pinned vars: full width for main content
		b.WriteString(leftContent.String())
	}

	return b.String()
}

// handleInput processes user input
func (m model) handleInput() model {
	input := strings.TrimSpace(m.input.Value())
	m.err = nil

	if input == "" {
		return m
	}

	// Special handling for markdown mode
	if m.markdownMode {
		// Check for end command
		if input == "/end" || input == "end" {
			// Finish markdown block
			if len(m.markdownLines) > 0 {
				m = m.insertMarkdownBlock()
			}
			m.markdownMode = false
			m.markdownLines = []string{}
			m.lastInputted = ""
			return m
		}

		// Accumulate markdown line
		m.markdownLines = append(m.markdownLines, m.input.Value()) // Keep original spacing
		return m
	}

	// Slash mode: treat input as command
	if m.slashMode {
		m = m.handleCommand(input)
		m.slashMode = false // Exit slash mode after command
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
			// Unpin all
			m.pinnedVars = make(map[string]bool)
		} else {
			// Unpin specific variable
			varName := parts[1]
			delete(m.pinnedVars, varName)
		}

	case "open":
		if len(parts) < 2 {
			m.err = fmt.Errorf("usage: open <filename>")
			return m
		}

		filename := parts[1]
		m = m.openFile(filename)

	case "md", "markdown":
		// Enter multi-line markdown mode
		m.markdownMode = true
		m.markdownLines = []string{}

	case "quit", "q":
		m.quitting = true
		return m

	case "help", "h":
		// Help is shown in View
		return m

	default:
		m.err = fmt.Errorf("unknown command: %s", parts[0])
	}

	return m
}

// renderHistory renders the scrollback history of commands and results
func (m model) renderHistory() string {
	if len(m.outputHistory) == 0 {
		return ""
	}

	var b strings.Builder

	// Calculate how many history items to show based on terminal height
	// Reserve space for: title(2) + mode(1) + separator(1) + input(2) + error(1) + help(2) = ~9 lines
	maxHistoryLines := max(m.height-12, 5)

	// Count lines needed for history
	historyLines := 0
	startIdx := 0
	for i := len(m.outputHistory) - 1; i >= 0; i-- {
		linesForItem := 1 // input line
		if m.outputHistory[i].output != "" {
			linesForItem++ // output line
		}
		if historyLines+linesForItem > maxHistoryLines {
			startIdx = i + 1
			break
		}
		historyLines += linesForItem
	}

	// Render visible history items
	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))
	outputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#5FD7FF"))
	errorOutputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))

	for i := startIdx; i < len(m.outputHistory); i++ {
		item := m.outputHistory[i]

		// Show input
		b.WriteString(promptStyle.Render("> "))
		b.WriteString(item.input)
		b.WriteString("\n")

		// Show output if any
		if item.output != "" {
			if item.isError {
				b.WriteString(errorOutputStyle.Render("  " + item.output))
			} else {
				b.WriteString(outputStyle.Render("  " + item.output))
			}
			b.WriteString("\n")
		}
	}

	return b.String()
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

		// Evaluate the new block
		eval := implDoc.NewEvaluator()
		err = eval.EvaluateBlock(m.doc, result.ModifiedBlockID)
		if err != nil {
			m.err = fmt.Errorf("evaluate: %w", err)
			return m
		}

		// Track which variables changed
		for _, id := range result.AffectedBlockIDs {
			node, ok := m.doc.GetBlock(id)
			if !ok {
				continue
			}
			if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
				for _, varName := range calcBlock.Variables() {
					m.changedVars[varName] = true
				}
			}
		}
	} else {
		// First block
		doc, err := document.NewDocument(expr + "\n")
		if err != nil {
			m.err = fmt.Errorf("parse: %w", err)
			return m
		}

		eval := implDoc.NewEvaluator()
		err = eval.Evaluate(doc)
		if err != nil {
			m.err = fmt.Errorf("evaluate: %w", err)
			return m
		}

		m.doc = doc
	}

	return m
}

// renderPinnedVars renders the pinned variables panel content
func (m model) renderPinnedVars() string {
	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().Bold(true).Render("📌 Pinned Variables"))
	b.WriteString("\n\n")

	count := 0
	// Use document block order (which is stable and based on UUIDs)
	for _, node := range m.doc.GetBlocks() {
		if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
			for _, varName := range calcBlock.Variables() {
				if !m.pinnedVars[varName] {
					continue
				}

				value := calcBlock.LastValue()

				// Format variable display
				varStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

				if m.changedVars[varName] {
					// Highlight changed variables
					b.WriteString(varStyle.Bold(true).Render(varName))
					b.WriteString(" = ")
					b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF9800")).Render(fmt.Sprintf("%v", value)))
					b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888")).Render(" [CHANGED]"))
				} else {
					b.WriteString(varStyle.Render(varName))
					b.WriteString(" = ")
					b.WriteString(fmt.Sprintf("%v", value))
				}
				b.WriteString("\n")
				count++
			}
		}
	}

	if count == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888")).Render("(no variables pinned)"))
	}

	return b.String()
}

// renderHelp renders help text responsively
func (m model) renderHelp() string {
	// Compact help that fits most terminal widths
	if m.width < 80 {
		// Very compact for narrow terminals
		return "/:Commands │ Esc:Exit cmd │ Ctrl+C:Quit"
	} else if m.width < 120 {
		// Medium terminals
		return "↑↓:History │ /:Commands (pin,open,md,quit) │ Esc:Exit cmd mode │ Ctrl+C:Quit"
	} else {
		// Wide terminals - full help
		if m.slashMode {
			// In slash mode, show available commands
			helps := []string{
				"pin - Pin all vars",
				"open <file> - Load file",
				"md - Markdown mode",
				"quit - Exit REPL",
				"Esc - Exit command mode",
			}
			return strings.Join(helps, "  │  ")
		} else {
			helps := []string{
				"↑↓ History",
				"/ Enter command mode",
				"Ctrl+C Quit",
			}
			return strings.Join(helps, "  │  ")
		}
	}
}
