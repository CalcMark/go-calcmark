package repl

import (
	"fmt"
	"strings"
	"time"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/shared"
	"github.com/CalcMark/go-calcmark/format/display"
	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/CalcMark/go-calcmark/spec/features"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the Simple REPL mode state.
// The Simple REPL is a minimal, scrolling history view.
// No split panes, no pinned panel - just input → output in a list.
// Implements tea.Model for bubbletea.
type Model struct {
	// Document and evaluation
	doc      *document.Document
	eval     *implDoc.Evaluator
	registry *features.Registry

	// UI components
	input textinput.Model

	// State
	history       []string              // Command history for ↑↓
	outputHistory []shared.HistoryEntry // Display history (input/output pairs)
	pinnedVars    map[string]bool       // Variables (kept for /vars command)
	changedVars   map[string]bool       // Variables changed in last update
	historyIdx    int                   // Current position in history (-1 = not browsing)
	lastEscTime   int64                 // For double-ESC detection
	lastSuggest   []features.Feature    // Cached suggestions
	slashCommands []shared.SlashCommand // Available commands

	// Modes
	inputMode shared.InputMode
	quitting  bool

	// Dimensions
	width  int
	height int

	// Error state
	err error

	// Styles (from config)
	styles config.Styles
}

// New creates a new Simple REPL model with an optional initial document.
func New(doc *document.Document) Model {
	if doc == nil {
		doc, _ = document.NewDocument("")
	}

	// Initialize text input
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "Enter expression (e.g., salary = $85000)"
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 70

	// Initialize evaluator
	eval := implDoc.NewEvaluator()
	_ = eval.Evaluate(doc)

	m := Model{
		doc:           doc,
		eval:          eval,
		registry:      features.NewRegistry(),
		input:         ti,
		pinnedVars:    make(map[string]bool),
		changedVars:   make(map[string]bool),
		history:       []string{},
		outputHistory: []shared.HistoryEntry{},
		historyIdx:    -1,
		slashCommands: shared.DefaultSlashCommands(),
		inputMode:     shared.InputNormal,
		width:         80,
		height:        24,
		styles:        config.GetStyles(),
	}

	// Track variables from the loaded document
	m.populateFromDocument()

	return m
}

// populateFromDocument initializes UI state from the document.
func (m *Model) populateFromDocument() {
	m.pinnedVars = make(map[string]bool)
	m.changedVars = make(map[string]bool)
	m.outputHistory = []shared.HistoryEntry{}
	m.history = []string{}
	m.historyIdx = -1

	for _, node := range m.doc.GetBlocks() {
		calcBlock, ok := node.Block.(*document.CalcBlock)
		if !ok {
			continue
		}

		// Auto-pin all variables
		for _, varName := range calcBlock.Variables() {
			m.pinnedVars[varName] = true
		}

		// Add to output history
		for _, line := range calcBlock.Source() {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}

			var output string
			if calcBlock.LastValue() != nil {
				output = fmt.Sprintf("= %s", display.Format(calcBlock.LastValue()))
			}
			m.outputHistory = append(m.outputHistory, shared.HistoryEntry{
				Input:   trimmed,
				Output:  output,
				IsError: calcBlock.Error() != nil,
			})
			m.history = append(m.history, trimmed)
		}
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = m.width - 6
	}

	// Forward to text input
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// handleKey processes keyboard input.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyCtrlD:
		m.quitting = true
		return m, tea.Quit

	case tea.KeyEsc:
		return m.handleEscape()

	case tea.KeyUp:
		return m.handleHistoryUp()

	case tea.KeyDown:
		return m.handleHistoryDown()

	case tea.KeyPgUp, tea.KeyPgDown:
		// No action needed in Simple REPL - history scrolls automatically
		return m, nil

	case tea.KeyEnter:
		return m.handleEnter()

	case tea.KeyTab:
		if m.tryCompleteVariable() {
			return m, nil
		}

	case tea.KeyRunes:
		// Check for '/' to enter slash mode
		if m.inputMode == shared.InputNormal &&
			len(msg.Runes) == 1 &&
			msg.Runes[0] == '/' &&
			m.input.Value() == "" {
			m.inputMode = shared.InputSlash
			m.input.SetValue("")
			m.input.Prompt = "/ "
			return m, nil
		}
	}

	// Default: forward to text input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// handleEscape processes the escape key.
func (m Model) handleEscape() (tea.Model, tea.Cmd) {
	// Exit slash mode
	if m.inputMode == shared.InputSlash {
		m.inputMode = shared.InputNormal
		m.input.SetValue("")
		m.input.Prompt = "> "
		return m, nil
	}

	// Double-ESC to clear input
	now := time.Now().UnixNano()
	if m.lastEscTime > 0 && (now-m.lastEscTime) < 500_000_000 {
		m.input.SetValue("")
		m.lastEscTime = 0
		m.lastSuggest = nil
		return m, nil
	}
	m.lastEscTime = now
	return m, nil
}

// handleHistoryUp navigates history backward.
func (m Model) handleHistoryUp() (tea.Model, tea.Cmd) {
	if len(m.history) == 0 {
		return m, nil
	}

	if m.historyIdx == -1 {
		m.historyIdx = len(m.history) - 1
	} else if m.historyIdx > 0 {
		m.historyIdx--
	}

	if m.historyIdx >= 0 && m.historyIdx < len(m.history) {
		m.input.SetValue(m.history[m.historyIdx])
	}
	return m, nil
}

// handleHistoryDown navigates history forward.
func (m Model) handleHistoryDown() (tea.Model, tea.Cmd) {
	if m.historyIdx == -1 {
		return m, nil
	}

	m.historyIdx++
	if m.historyIdx >= len(m.history) {
		m.historyIdx = -1
		m.input.SetValue("")
	} else {
		m.input.SetValue(m.history[m.historyIdx])
	}
	return m, nil
}

// handleEnter processes input submission.
func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.input.Value())
	m.err = nil

	if input == "" {
		return m, nil
	}

	// Slash mode: treat as command
	if m.inputMode == shared.InputSlash {
		m, cmd := m.handleCommand(input)
		m.inputMode = shared.InputNormal
		m.input.Prompt = "> "
		m.input.SetValue("")
		m.historyIdx = -1

		if m.quitting {
			return m, tea.Quit
		}
		return m, cmd
	}

	// Normal mode: evaluate expression
	m.changedVars = make(map[string]bool)
	m.lastSuggest = nil

	// Add to history
	if len(m.history) == 0 || m.history[len(m.history)-1] != input {
		m.history = append(m.history, input)
		if len(m.history) > 100 {
			m.history = m.history[1:]
		}
	}

	// Evaluate
	m = m.evaluateExpression(input)

	m.input.SetValue("")
	m.historyIdx = -1
	return m, nil
}

// handleCommand processes slash commands.
func (m Model) handleCommand(cmd string) (Model, tea.Cmd) {
	cmd = strings.TrimPrefix(cmd, "/")
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return m, nil
	}

	switch parts[0] {
	case "vars", "v":
		// List all defined variables
		varsOutput := m.formatVariables()
		m.outputHistory = append(m.outputHistory, shared.HistoryEntry{
			Input:   "/vars",
			Output:  varsOutput,
			IsError: false,
		})

	case "clear", "c":
		// Clear screen (keep variables)
		m.outputHistory = []shared.HistoryEntry{}

	case "reset":
		// Clear everything
		m.outputHistory = []shared.HistoryEntry{}
		m.history = []string{}
		m.pinnedVars = make(map[string]bool)
		m.changedVars = make(map[string]bool)
		m.doc, _ = document.NewDocument("")
		m.eval = implDoc.NewEvaluator()
		_ = m.eval.Evaluate(m.doc)

	case "pin":
		if len(parts) == 1 {
			// Pin all
			for _, node := range m.doc.GetBlocks() {
				if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
					for _, varName := range calcBlock.Variables() {
						m.pinnedVars[varName] = true
					}
				}
			}
		} else {
			m.pinnedVars[parts[1]] = true
		}

	case "unpin":
		if len(parts) == 1 {
			m.pinnedVars = make(map[string]bool)
		} else {
			delete(m.pinnedVars, parts[1])
		}

	case "edit", "e":
		// Switch to editor mode
		var filepath string
		if len(parts) > 1 {
			filepath = parts[1]
		}
		return m, func() tea.Msg {
			return shared.SwitchModeMsg{
				Mode:     shared.ModeEditor,
				Filepath: filepath,
			}
		}

	case "quit", "q":
		m.quitting = true

	case "help", "h", "?":
		helpText := RenderHelpText(m.width)
		m.outputHistory = append(m.outputHistory, shared.HistoryEntry{
			Input:   "/help",
			Output:  helpText,
			IsError: false,
		})

	default:
		m.err = fmt.Errorf("unknown command: /%s (type /help)", parts[0])
	}

	return m, nil
}

// formatVariables formats all variables for the /vars command.
func (m Model) formatVariables() string {
	env := m.eval.GetEnvironment()
	allVars := env.GetAllVariables()

	if len(allVars) == 0 {
		return "(no variables defined)"
	}

	var b strings.Builder
	b.WriteString("Defined variables:\n")
	for name, val := range allVars {
		b.WriteString(fmt.Sprintf("  %s = %s\n", name, display.Format(val)))
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// evaluateExpression evaluates a CalcMark expression.
func (m Model) evaluateExpression(expr string) Model {
	blocks := m.doc.GetBlocks()

	var afterID string
	if len(blocks) > 0 {
		afterID = blocks[len(blocks)-1].ID
		result, err := m.doc.InsertBlock(afterID, document.BlockCalculation, []string{expr})
		if err != nil {
			m.err = fmt.Errorf("insert block: %w", err)
			return m
		}

		err = m.eval.EvaluateBlock(m.doc, result.ModifiedBlockID)
		if err != nil {
			m.addErrorToHistory(expr, err)
			return m
		}

		// Track changed variables
		for _, id := range result.AffectedBlockIDs {
			node, ok := m.doc.GetBlock(id)
			if !ok {
				continue
			}
			if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
				for _, varName := range calcBlock.Variables() {
					m.changedVars[varName] = true
					m.pinnedVars[varName] = true
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

		m.eval = implDoc.NewEvaluator()
		if err := m.eval.Evaluate(doc); err != nil {
			m.addErrorToHistory(expr, err)
			return m
		}

		m.doc = doc

		// Auto-pin variables
		for _, node := range doc.GetBlocks() {
			if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
				for _, varName := range calcBlock.Variables() {
					m.changedVars[varName] = true
					m.pinnedVars[varName] = true
				}
			}
		}
	}

	// Add result to history
	m.addResultToHistory(expr)
	return m
}

// addResultToHistory adds the evaluation result to output history.
func (m *Model) addResultToHistory(expr string) {
	blocks := m.doc.GetBlocks()
	if len(blocks) == 0 {
		return
	}

	lastBlock := blocks[len(blocks)-1]
	calcBlock, ok := lastBlock.Block.(*document.CalcBlock)
	if !ok {
		return
	}

	if calcBlock.Error() != nil {
		m.outputHistory = append(m.outputHistory, shared.HistoryEntry{
			Input:   expr,
			Output:  calcBlock.Error().Error(),
			IsError: true,
		})
		return
	}

	if calcBlock.LastValue() != nil {
		m.outputHistory = append(m.outputHistory, shared.HistoryEntry{
			Input:   expr,
			Output:  fmt.Sprintf("= %s", display.Format(calcBlock.LastValue())),
			IsError: false,
		})
	}
}

// addErrorToHistory adds an error to output history.
func (m *Model) addErrorToHistory(expr string, err error) {
	m.outputHistory = append(m.outputHistory, shared.HistoryEntry{
		Input:   expr,
		Output:  err.Error(),
		IsError: true,
	})
}

// tryCompleteVariable attempts tab completion.
func (m *Model) tryCompleteVariable() bool {
	input := m.input.Value()
	if input == "" {
		return false
	}

	// Extract last token
	lastToken := extractLastToken(input)
	if len(lastToken) < 2 {
		return false
	}

	// Find matching variables
	env := m.eval.GetEnvironment()
	allVars := env.GetAllVariables()

	var matches []string
	prefix := strings.ToLower(lastToken)
	for varName := range allVars {
		if strings.HasPrefix(strings.ToLower(varName), prefix) && varName != lastToken {
			matches = append(matches, varName)
		}
	}

	if len(matches) == 0 {
		return false
	}

	// Complete with first match
	completion := matches[0]
	newInput := input[:len(input)-len(lastToken)] + completion
	m.input.SetValue(newInput)
	m.input.SetCursor(len(newInput))
	return true
}

// extractLastToken gets the last identifier from input.
func extractLastToken(input string) string {
	end := len(input)
	start := end

	for i := len(input) - 1; i >= 0; i-- {
		ch := input[i]
		if isIdentChar(ch) {
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

func isIdentChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_'
}

// Quitting returns whether the REPL should quit.
func (m Model) Quitting() bool {
	return m.quitting
}

// Document returns the current document.
func (m Model) Document() *document.Document {
	return m.doc
}

// SetDocument sets a new document.
func (m *Model) SetDocument(doc *document.Document) {
	m.doc = doc
	m.eval = implDoc.NewEvaluator()
	_ = m.eval.Evaluate(doc)
	m.populateFromDocument()
}
