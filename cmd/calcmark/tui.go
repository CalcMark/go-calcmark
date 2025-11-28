package main

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config"
	"github.com/CalcMark/go-calcmark/format/display"
	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/CalcMark/go-calcmark/spec/features"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// styles holds the pre-built lipgloss styles from configuration.
// Initialized in init() after config is loaded.
var styles config.Styles

func init() {
	// CRITICAL: Set color profile and background explicitly to avoid terminal queries.
	// Terminal color detection sends OSC sequences (like OSC 11 for background color)
	// that can leak into input buffers, causing garbage text like "]11;rgb:ffff/ffff/ffff".
	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(true)

	// Load configuration and build styles
	if _, err := config.Load(); err != nil {
		// Config load failed - styles will use zero values
		// This is a development error, should not happen in production
		panic("failed to load config: " + err.Error())
	}
	styles = config.GetStyles()
}

// model represents the TUI state
type model struct {
	doc             *document.Document
	eval            *implDoc.Evaluator  // Persistent evaluator (holds variable environment)
	registry        *features.Registry  // Feature registry for help and autosuggest
	input           textinput.Model     // Single-line input for calc expressions
	mdInput         textarea.Model      // Multi-line input for markdown mode
	pinnedVars      map[string]bool     // Which variables are pinned
	changedVars     map[string]bool     // Variables that changed in last update
	history         []string            // Command history (for ↑↓ navigation)
	outputHistory   []outputHistoryItem // Display history (commands + results)
	historyIdx      int                 // Current position in history (-1 = not browsing)
	lastSuggestions []features.Feature  // Last suggestions shown (persists while typing)
	width           int
	height          int
	err             error
	lastInputted    string
	quitting        bool
	markdownMode    bool // In multi-line markdown mode
	slashMode       bool // In slash command mode (triggered by / key)
}

// outputHistoryItem represents a command and its result for display
type outputHistoryItem struct {
	input   string // The command entered
	output  string // The result/output
	isError bool   // Whether this was an error
}

// slashCommand defines a slash command with its syntax and description
type slashCommand struct {
	name        string // Command name without /
	syntax      string // Full syntax example
	description string // Brief description
}

// slashCommands is the list of available slash commands for autosuggestion
var slashCommands = []slashCommand{
	{"pin", "/pin [var]", "Pin variable(s) to panel"},
	{"unpin", "/unpin [var]", "Unpin variable(s)"},
	{"open", "/open <file>", "Load CalcMark file"},
	{"save", "/save <file>", "Save as CalcMark (.cm)"},
	{"output", "/output <file>", "Export with results"},
	{"md", "/md", "Multi-line markdown mode"},
	{"quit", "/quit", "Exit the REPL"},
	{"q", "/q", "Exit (shortcut)"},
	{"help", "/help [topic]", "Show help"},
}

// newTUIModel creates initial TUI model
func newTUIModel(doc *document.Document) model {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "Enter CalcMark expression or /command"
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 50

	// Create textarea for markdown mode with high-contrast styling
	ta := textarea.New()
	ta.Placeholder = "Enter markdown... (Esc to finish)"
	ta.ShowLineNumbers = false
	ta.CharLimit = 10000

	// Style the textarea for visibility on dark backgrounds
	// CursorLine style applies to the active line, so it needs both fg and bg
	ta.FocusedStyle.Text = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#333333"))
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	ta.BlurredStyle = ta.FocusedStyle

	// Create persistent evaluator and evaluate the loaded document
	eval := implDoc.NewEvaluator()
	_ = eval.Evaluate(doc) // Ignore error - blocks will show their own errors

	// Create feature registry for help and autosuggest
	registry := features.NewRegistry()

	m := model{
		doc:           doc,
		eval:          eval,
		registry:      registry,
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
						output = fmt.Sprintf("= %s", display.Format(calcBlock.LastValue()))
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

		case tea.KeyTab:
			// Tab completion for variable names
			if !m.markdownMode && !m.slashMode {
				if completed := m.tryCompleteVariable(); completed {
					return m, nil
				}
			}
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
	title := styles.Title.Render("CalcMark REPL")
	b.WriteString(title)
	b.WriteString("\n")

	// Markdown mode: split view with editor on left, preview on right
	if m.markdownMode {
		b.WriteString(m.renderMarkdownMode())
		return b.String()
	}

	// Mode indicator for slash mode
	if m.slashMode {
		modeIndicator := styles.ModeIndicator.Render("💬 COMMAND MODE - Type command or Esc to exit")
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
		leftContent.WriteString(styles.Separator.Render("─────────────────────────────────────────"))
		leftContent.WriteString("\n")
	}

	// Input (textinput handles its own prompt)
	leftContent.WriteString(m.input.View())
	leftContent.WriteString("\n")

	// Autosuggest line (below input)
	if m.slashMode {
		// Slash mode: show command suggestions
		slashSuggestions := getSlashCommandSuggestions(m.input.Value())
		if len(slashSuggestions) > 0 {
			leftContent.WriteString(renderSlashSuggestions(slashSuggestions))
			leftContent.WriteString("\n")
		}
	} else {
		// Normal mode: show variable completions and feature hints
		suggestions := m.updateSuggestions()
		varCompletions := m.getVariableCompletions()
		if len(suggestions) > 0 || len(varCompletions) > 0 {
			leftContent.WriteString(renderSuggestions(suggestions, varCompletions))
			leftContent.WriteString("\n")
		}
	}

	// Error display
	if m.err != nil {
		leftContent.WriteString(styles.Error.Render(fmt.Sprintf("Error: %v", m.err)))
		leftContent.WriteString("\n")
	}

	// Help text
	help := m.renderHelp()
	leftContent.WriteString(styles.Help.Render(help))

	// Layout: Always show pinned panel on the right
	// Right panel: Pinned variables (40% width)
	rightWidth := max(m.width*2/5, 25)

	pinnedContent := m.renderPinnedVars()
	rightPanel := styles.PinnedPanel.
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
	cfg := config.Get()
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color(cfg.TUI.Theme.Warning)).
		Bold(true).
		Render("📝 MARKDOWN MODE - Esc to save and exit")
	helpLink := lipgloss.NewStyle().
		Foreground(lipgloss.Color(cfg.TUI.Theme.Muted)).
		Render("  Help: https://commonmark.org/help/")
	headerLine := header + helpLink + "\n\n"

	// Left panel: markdown input
	leftTitle := lipgloss.NewStyle().Bold(true).Render("Edit")
	leftPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(cfg.TUI.Theme.Warning)).
		Padding(0, 1).
		Width(halfWidth - 4).
		Height(m.height - 8).
		Render(leftTitle + "\n" + m.mdInput.View())

	// Right panel: rendered preview
	rightTitle := lipgloss.NewStyle().Bold(true).Render("Preview")
	preview := m.renderMarkdownPreview()
	rightPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(cfg.TUI.Theme.Accent)).
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
	m.lastSuggestions = nil               // Clear suggestions after submit

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
		// Clear the error since it's now in history - prevents duplicate display
		result.err = nil
	} else {
		// Find the last value
		blocks := result.doc.GetBlocks()
		if len(blocks) > 0 {
			if calcBlock, ok := blocks[len(blocks)-1].Block.(*document.CalcBlock); ok {
				if calcBlock.LastValue() != nil {
					output = fmt.Sprintf("= %s", display.Format(calcBlock.LastValue()))
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
		// Handle /help or /help <topic>
		if len(parts) == 1 {
			// Show general help with dynamic formatting based on terminal width
			helpText := renderHelpText(m.width)
			m.outputHistory = append(m.outputHistory, outputHistoryItem{
				input:   "/help",
				output:  helpText,
				isError: false,
			})
		} else {
			// Search for topic
			topic := strings.ToLower(parts[1])
			m = m.showHelpTopic(topic)
		}
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

// showHelpTopic displays help for a specific topic or search term.
func (m model) showHelpTopic(topic string) model {
	var output strings.Builder

	// Use pre-built styles from config
	syntaxStyle := styles.Syntax
	descStyle := styles.Output
	exampleStyle := styles.Example
	headerStyle := styles.Header

	// Check for category names first
	var cat features.Category
	switch topic {
	case "function", "functions", "func":
		cat = features.CategoryFunction
	case "unit", "units":
		cat = features.CategoryUnit
	case "date", "dates", "time":
		cat = features.CategoryDate
	case "network", "net":
		cat = features.CategoryNetwork
	case "storage", "disk":
		cat = features.CategoryStorage
	case "compression", "compress":
		cat = features.CategoryCompression
	case "keyword", "keywords":
		cat = features.CategoryKeyword
	case "operator", "operators", "ops":
		cat = features.CategoryOperator
	}

	var results []features.Feature
	if cat != "" {
		// Get all features in category
		results = m.registry.ByCategory(cat)
		output.WriteString(headerStyle.Render("== "+strings.ToUpper(string(cat))+" ==") + "\n\n")
	} else {
		// Search by prefix
		results = m.registry.Search(topic)
		if len(results) == 0 {
			m.outputHistory = append(m.outputHistory, outputHistoryItem{
				input:   "/help " + topic,
				output:  fmt.Sprintf("No matches for '%s'. Try: /help functions, /help units", topic),
				isError: false,
			})
			return m
		}
		output.WriteString(headerStyle.Render("== Search: "+topic+" ==") + "\n\n")
	}

	// Format results with styled syntax (bold) and examples (italic)
	for _, f := range results {
		output.WriteString(syntaxStyle.Render(f.Syntax))
		output.WriteString("  ")
		output.WriteString(descStyle.Render(f.Description))
		output.WriteString("\n")
		if f.Example != "" {
			output.WriteString("  ")
			output.WriteString(exampleStyle.Render(f.Example))
			output.WriteString("\n")
		}
	}

	if len(results) > 10 {
		output.WriteString(fmt.Sprintf("\n(%d items)", len(results)))
	}

	m.outputHistory = append(m.outputHistory, outputHistoryItem{
		input:   "/help " + topic,
		output:  output.String(),
		isError: false,
	})
	return m
}

// updateSuggestions searches for new suggestions based on current input.
// Returns new suggestions if found, otherwise returns previous suggestions.
// This keeps hints visible while the user continues typing.
func (m *model) updateSuggestions() []features.Feature {
	if m.slashMode || m.markdownMode {
		return nil
	}

	input := strings.TrimSpace(m.input.Value())
	if input == "" {
		// Keep showing last suggestions when input is cleared
		return m.lastSuggestions
	}

	// Extract the last word being typed
	words := strings.Fields(input)
	if len(words) == 0 {
		return m.lastSuggestions
	}
	lastWord := words[len(words)-1]

	// Only search if the last word is at least 2 characters
	if len(lastWord) < 2 {
		return m.lastSuggestions
	}

	// Search for matches
	matches := m.registry.Search(lastWord)

	// Limit to top 3 suggestions
	if len(matches) > 3 {
		matches = matches[:3]
	}

	// Update stored suggestions if we found new ones
	if len(matches) > 0 {
		m.lastSuggestions = matches
	}

	return m.lastSuggestions
}

// renderSuggestions formats the autosuggest line.
func renderSuggestions(suggestions []features.Feature, varCompletions []string) string {
	var parts []string

	// Add variable completions first (Tab to complete)
	for _, v := range varCompletions {
		parts = append(parts, v+" [Tab]")
	}

	// Add feature suggestions
	for _, s := range suggestions {
		parts = append(parts, fmt.Sprintf("%s (%s)", s.Name, s.Syntax))
	}

	if len(parts) == 0 {
		return ""
	}

	// Use Hint style from config
	return styles.Hint.Render("Hints: " + strings.Join(parts, " | "))
}

// getSlashCommandSuggestions returns slash commands that match the current input.
func getSlashCommandSuggestions(input string) []slashCommand {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		// Show all commands when input is empty
		return slashCommands
	}

	var matches []slashCommand
	for _, cmd := range slashCommands {
		if strings.HasPrefix(cmd.name, input) {
			matches = append(matches, cmd)
		}
	}

	// Limit to top 4 suggestions
	if len(matches) > 4 {
		matches = matches[:4]
	}

	return matches
}

// renderSlashSuggestions formats slash command suggestions.
func renderSlashSuggestions(suggestions []slashCommand) string {
	if len(suggestions) == 0 {
		return ""
	}

	var parts []string
	for _, cmd := range suggestions {
		parts = append(parts, fmt.Sprintf("%s (%s)", cmd.syntax, cmd.description))
	}

	return styles.Hint.Render(strings.Join(parts, " │ "))
}

// getVariableCompletions returns variable names that match the current partial input.
func (m model) getVariableCompletions() []string {
	input := m.input.Value()
	if input == "" {
		return nil
	}

	// Find the last word being typed (potential variable name)
	words := strings.Fields(input)
	if len(words) == 0 {
		return nil
	}

	// Get the last token - could be after an operator
	lastWord := extractLastToken(input)
	if len(lastWord) < 2 {
		return nil
	}

	// Get all defined variables from the environment
	env := m.eval.GetEnvironment()
	allVars := env.GetAllVariables()

	// Find matching variables
	var matches []string
	prefix := strings.ToLower(lastWord)
	for varName := range allVars {
		if strings.HasPrefix(strings.ToLower(varName), prefix) && varName != lastWord {
			matches = append(matches, varName)
		}
	}

	// Sort and limit
	if len(matches) > 3 {
		matches = matches[:3]
	}

	return matches
}

// extractLastToken gets the last identifier-like token from input.
// Handles cases like "b = 2 * aaaa" -> "aaaa"
func extractLastToken(input string) string {
	// Walk backwards to find the start of the last identifier
	end := len(input)
	start := end

	for i := len(input) - 1; i >= 0; i-- {
		ch := input[i]
		if isIdentifierChar(ch) {
			start = i
		} else {
			break
		}
	}

	if start >= end {
		return ""
	}
	return input[start:end]
}

// isIdentifierChar returns true if ch can be part of a variable name.
func isIdentifierChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_'
}

// tryCompleteVariable attempts to complete the current variable name with Tab.
// Returns true if a completion was made.
func (m *model) tryCompleteVariable() bool {
	completions := m.getVariableCompletions()
	if len(completions) == 0 {
		return false
	}

	// Complete with the first match
	completion := completions[0]
	input := m.input.Value()
	lastToken := extractLastToken(input)

	if lastToken == "" {
		return false
	}

	// Replace the partial token with the full variable name
	newInput := input[:len(input)-len(lastToken)] + completion
	m.input.SetValue(newInput)
	m.input.SetCursor(len(newInput))

	return true
}
