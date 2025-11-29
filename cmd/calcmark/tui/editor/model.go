package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/shared"
	"github.com/CalcMark/go-calcmark/format/display"
	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/document"
	tea "github.com/charmbracelet/bubbletea"
)

// Debounce delay for re-evaluation after typing (per spec: ~50ms)
const evalDebounceDelay = 50 * time.Millisecond

// evalDebounceMsg is sent after the debounce delay to trigger evaluation.
type evalDebounceMsg struct {
	editBufSnapshot string // Snapshot of editBuf when timer was started
}

// EditorMode represents the current editor mode.
type EditorMode int

const (
	ModeNormal  EditorMode = iota // Normal navigation mode
	ModeEditing                   // Line editing mode
	ModeCommand                   // Command palette mode
	ModeGlobals                   // Globals panel focused
	ModeHelp                      // Help viewer
)

// PreviewMode represents the preview pane display mode.
type PreviewMode int

const (
	PreviewFull    PreviewMode = iota // Show variable name + value
	PreviewMinimal                    // Show just arrow + value (left-aligned, narrower)
	PreviewHidden                     // No preview pane
)

// PaneWidthConfig defines the source/preview width ratios for each preview mode.
// Widths are expressed as percentages (source + preview = 100).
type PaneWidthConfig struct {
	SourcePercent  int // Source pane width percentage
	PreviewPercent int // Preview pane width percentage
}

// DefaultPaneWidths returns the default pane width configurations for each preview mode.
var DefaultPaneWidths = map[PreviewMode]PaneWidthConfig{
	PreviewFull:    {SourcePercent: 55, PreviewPercent: 45},
	PreviewMinimal: {SourcePercent: 75, PreviewPercent: 25},
	PreviewHidden:  {SourcePercent: 100, PreviewPercent: 0},
}

// GetPaneWidths returns the source and preview pane widths for the given total width.
func (m Model) GetPaneWidths(totalWidth int) (sourceWidth, previewWidth int) {
	cfg := DefaultPaneWidths[m.previewMode]
	sourceWidth = totalWidth * cfg.SourcePercent / 100
	previewWidth = totalWidth - sourceWidth
	return
}

// Model represents the document editor state.
type Model struct {
	// Core document (from spec/document)
	doc      *document.Document
	eval     *implDoc.Evaluator
	filepath string
	modified bool

	// Cursor and navigation
	cursorLine   int // Current line (0-indexed)
	cursorCol    int // Current column (0-indexed)
	scrollOffset int // Vertical scroll offset

	// Editor state
	mode            EditorMode
	editBuf         string          // Buffer for line being edited
	lineWrap        bool            // Whether to wrap long lines
	changedBlockIDs map[string]bool // Track changed blocks for highlighting

	// Undo/redo
	undoStack []string // Document content snapshots
	redoStack []string

	// Command palette
	cmdInput   string
	cmdHistory []string

	// Globals panel
	globalsExpanded bool
	globalsFocusIdx int

	// Pinned variables
	pinnedVars  map[string]bool
	changedVars map[string]bool

	// UI state
	width       int
	height      int
	lastEscTime int64       // For double-ESC detection
	quitting    bool
	previewMode PreviewMode // Preview pane mode: Full, Minimal, Hidden
	pendingKey  rune        // For two-key sequences like gg, dd, yy
	yankBuffer  string      // Yanked line content for paste

	// Search state
	searchTerm    string // Current search term
	searchMatches []int  // Line numbers with matches
	searchIdx     int    // Current match index

	// Status message
	statusMsg   string
	statusIsErr bool

	// Styles
	styles config.Styles
}

// New creates a new editor model with an optional document.
func New(doc *document.Document) Model {
	if doc == nil {
		doc, _ = document.NewDocument("")
	}

	eval := implDoc.NewEvaluator()
	_ = eval.Evaluate(doc)

	m := Model{
		doc:             doc,
		eval:            eval,
		mode:            ModeNormal,
		pinnedVars:      make(map[string]bool),
		changedVars:     make(map[string]bool),
		changedBlockIDs: make(map[string]bool),
		undoStack:       []string{},
		redoStack:       []string{},
		cmdHistory:      []string{},
		width:           80,
		height:          24,
		previewMode:     PreviewFull,
		lineWrap:        true,
		styles:          config.GetStyles(),
	}

	// Auto-pin all variables
	m.autoPinVariables()

	// Save initial state for undo
	m.pushUndoState()

	return m
}

// NewWithFile creates an editor with a file loaded.
func NewWithFile(filepath string, doc *document.Document) Model {
	m := New(doc)
	m.filepath = filepath
	return m
}

// autoPinVariables pins all variables in the document.
func (m *Model) autoPinVariables() {
	for _, node := range m.doc.GetBlocks() {
		if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
			for _, varName := range calcBlock.Variables() {
				m.pinnedVars[varName] = true
			}
		}
	}
}

// pushUndoState saves current document state for undo.
func (m *Model) pushUndoState() {
	content := m.getDocumentContent()
	if len(m.undoStack) == 0 || m.undoStack[len(m.undoStack)-1] != content {
		m.undoStack = append(m.undoStack, content)
		// Limit undo stack size
		if len(m.undoStack) > 100 {
			m.undoStack = m.undoStack[1:]
		}
		// Clear redo stack on new change
		m.redoStack = nil
	}
}

// getDocumentContent returns the document as a string.
func (m *Model) getDocumentContent() string {
	var lines []string
	for _, node := range m.doc.GetBlocks() {
		switch b := node.Block.(type) {
		case *document.CalcBlock:
			lines = append(lines, b.Source()...)
		case *document.TextBlock:
			lines = append(lines, b.Source()...)
		}
	}
	return strings.Join(lines, "\n")
}

// GetLines returns all lines in the document.
func (m *Model) GetLines() []string {
	var lines []string
	for _, node := range m.doc.GetBlocks() {
		switch b := node.Block.(type) {
		case *document.CalcBlock:
			lines = append(lines, b.Source()...)
		case *document.TextBlock:
			lines = append(lines, b.Source()...)
		}
	}
	return lines
}

// TotalLines returns the total number of lines.
func (m *Model) TotalLines() int {
	return len(m.GetLines())
}

// CalcBlockCount returns the number of calculation blocks.
func (m *Model) CalcBlockCount() int {
	count := 0
	for _, node := range m.doc.GetBlocks() {
		if _, ok := node.Block.(*document.CalcBlock); ok {
			count++
		}
	}
	return count
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case evalDebounceMsg:
		// Only evaluate if editBuf hasn't changed since the timer was started
		// This ensures we don't evaluate stale content
		if m.mode == ModeEditing && m.editBuf == msg.editBufSnapshot {
			m.liveUpdateCurrentLine()
		}
	}

	return m, nil
}

// handleKey processes keyboard input.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Clear status message on any key
	m.statusMsg = ""
	m.statusIsErr = false

	// Global quit handlers
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyCtrlD:
		m.quitting = true
		return m, tea.Quit
	case tea.KeyCtrlS:
		// Save (Ctrl+S works in all modes)
		m.saveFile("")
		return m, nil
	}

	// Mode-specific handling
	switch m.mode {
	case ModeEditing:
		return m.handleEditKey(msg)
	case ModeCommand:
		return m.handleCommandKey(msg)
	case ModeGlobals:
		return m.handleGlobalsKey(msg)
	default:
		return m.handleNormalKey(msg)
	}
}

// handleNormalKey processes keys in normal navigation mode.
func (m Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		m.moveCursor(-1, 0)
	case tea.KeyDown:
		m.moveCursor(1, 0)
	case tea.KeyLeft:
		m.moveCursor(0, -1)
	case tea.KeyRight:
		m.moveCursor(0, 1)
	case tea.KeyPgUp:
		m.moveCursor(-(m.height - 4), 0)
	case tea.KeyPgDown:
		m.moveCursor(m.height-4, 0)
	case tea.KeyHome:
		m.cursorLine = 0
		m.cursorCol = 0
		m.scrollOffset = 0
	case tea.KeyEnd:
		total := m.TotalLines()
		if total > 0 {
			m.cursorLine = total - 1
		}
	case tea.KeyEnter:
		m.enterEditMode()
	case tea.KeyEsc:
		return m.handleEscape()
	case tea.KeyTab:
		// Tab cycles preview mode: Full → Minimal → Hidden → Full
		m.cyclePreviewMode()
	case tea.KeyCtrlD:
		// Half-page down
		m.moveCursor(m.height/2, 0)
	case tea.KeyCtrlU:
		// Half-page up
		m.moveCursor(-m.height/2, 0)
	case tea.KeyRunes:
		return m.handleNormalRune(msg.Runes)
	}

	return m, nil
}

// handleNormalRune handles character input in normal mode.
func (m Model) handleNormalRune(runes []rune) (tea.Model, tea.Cmd) {
	if len(runes) == 0 {
		return m, nil
	}

	key := runes[0]

	// Handle two-key sequences
	if m.pendingKey != 0 {
		pending := m.pendingKey
		m.pendingKey = 0

		switch pending {
		case 'g':
			if key == 'g' {
				// gg: go to top
				m.cursorLine = 0
				m.cursorCol = 0
				m.scrollOffset = 0
				return m, nil
			}
			// g followed by anything else: enter globals mode then process key
			m.mode = ModeGlobals
			m.globalsExpanded = true
			return m, nil
		case 'd':
			if key == 'd' {
				// dd: delete current line
				m.deleteLine()
				return m, nil
			}
		case 'y':
			if key == 'y' {
				// yy: yank (copy) current line
				m.yankLine()
				return m, nil
			}
		}
		// Invalid sequence, ignore
		return m, nil
	}

	// Check for start of two-key sequence
	switch key {
	case 'g':
		m.pendingKey = 'g'
		return m, nil
	case 'd':
		m.pendingKey = 'd'
		return m, nil
	case 'y':
		m.pendingKey = 'y'
		return m, nil
	}

	// Single key commands
	switch key {
	case 'j': // Down
		m.moveCursor(1, 0)
	case 'k': // Up
		m.moveCursor(-1, 0)
	case 'h': // Left
		m.moveCursor(0, -1)
	case 'l': // Right
		m.moveCursor(0, 1)
	case 'G': // Go to bottom
		total := m.TotalLines()
		if total > 0 {
			m.cursorLine = total - 1
		}
	case 'e', 'i': // Enter edit mode
		m.enterEditMode()
	case 'o': // Insert line below and enter edit mode
		m.insertLineBelow()
		m.enterEditMode()
	case 'O': // Insert line above and enter edit mode
		m.insertLineAbove()
		m.enterEditMode()
	case 'u': // Undo
		m.undo()
	case 'r': // Redo
		m.redo()
	case '/': // Enter command mode
		m.mode = ModeCommand
		m.cmdInput = ""
	case 'p': // Paste below (if yank buffer has content) OR cycle preview
		if m.yankBuffer != "" {
			m.pasteLine()
		} else {
			m.cyclePreviewMode()
		}
	case 'P': // Paste above
		m.pasteLineAbove()
	case 'v': // Toggle preview (alternate to Tab)
		m.cyclePreviewMode()
	case 'n': // Next search match
		m.nextSearchMatch()
	case 'N': // Previous search match
		m.prevSearchMatch()
	case '?': // Help
		m.mode = ModeHelp
	}

	return m, nil
}

// handleEditKey processes keys in edit mode.
func (m Model) handleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	contentChanged := false

	switch msg.Type {
	case tea.KeyEsc:
		m.exitEditMode(true) // Save changes
	case tea.KeyEnter:
		m.exitEditMode(true)
		m.insertLineBelow()
		m.enterEditMode()
	case tea.KeyBackspace:
		if len(m.editBuf) > 0 && m.cursorCol > 0 {
			m.editBuf = m.editBuf[:m.cursorCol-1] + m.editBuf[m.cursorCol:]
			m.cursorCol--
			contentChanged = true
		}
	case tea.KeyDelete:
		if m.cursorCol < len(m.editBuf) {
			m.editBuf = m.editBuf[:m.cursorCol] + m.editBuf[m.cursorCol+1:]
			contentChanged = true
		}
	case tea.KeyLeft:
		if m.cursorCol > 0 {
			m.cursorCol--
		}
	case tea.KeyRight:
		if m.cursorCol < len(m.editBuf) {
			m.cursorCol++
		}
	case tea.KeyHome:
		m.cursorCol = 0
	case tea.KeyEnd:
		m.cursorCol = len(m.editBuf)
	case tea.KeySpace:
		// Insert space at cursor
		m.editBuf = m.editBuf[:m.cursorCol] + " " + m.editBuf[m.cursorCol:]
		m.cursorCol++
		contentChanged = true
	case tea.KeyRunes:
		// Insert character at cursor
		for _, r := range msg.Runes {
			m.editBuf = m.editBuf[:m.cursorCol] + string(r) + m.editBuf[m.cursorCol:]
			m.cursorCol++
		}
		contentChanged = true
	}

	// Schedule debounced re-evaluation on content changes
	// This prevents re-evaluating on every keystroke (per spec: ~50ms debounce)
	if contentChanged {
		snapshot := m.editBuf
		return m, tea.Tick(evalDebounceDelay, func(t time.Time) tea.Msg {
			return evalDebounceMsg{editBufSnapshot: snapshot}
		})
	}

	return m, nil
}

// liveUpdateCurrentLine updates the current line and re-evaluates for live preview.
func (m *Model) liveUpdateCurrentLine() {
	// Update the line in the document
	m.updateCurrentLine(m.editBuf)
	// Re-evaluate to update preview
	m.reEvaluate()
}

// handleCommandKey processes keys in command mode.
func (m Model) handleCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = ModeNormal
		m.cmdInput = ""
	case tea.KeyEnter:
		m.executeCommand(m.cmdInput)
		m.mode = ModeNormal
		m.cmdInput = ""
		// Check if command requested quit
		if m.quitting {
			return m, tea.Quit
		}
	case tea.KeyBackspace:
		if len(m.cmdInput) > 0 {
			m.cmdInput = m.cmdInput[:len(m.cmdInput)-1]
		}
	case tea.KeyRunes:
		for _, r := range msg.Runes {
			m.cmdInput += string(r)
		}
	}

	return m, nil
}

// handleGlobalsKey processes keys when globals panel is focused.
func (m Model) handleGlobalsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = ModeNormal
		m.globalsExpanded = false
	case tea.KeyUp, tea.KeyRunes:
		if msg.Type == tea.KeyUp || (len(msg.Runes) > 0 && msg.Runes[0] == 'k') {
			if m.globalsFocusIdx > 0 {
				m.globalsFocusIdx--
			}
		}
	case tea.KeyDown:
		globalsCount := m.getGlobalsCount()
		if m.globalsFocusIdx < globalsCount-1 {
			m.globalsFocusIdx++
		}
	case tea.KeyEnter:
		// Could edit focused global
		m.mode = ModeNormal
	}

	// Handle 'j' for down
	if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 && msg.Runes[0] == 'j' {
		globalsCount := m.getGlobalsCount()
		if m.globalsFocusIdx < globalsCount-1 {
			m.globalsFocusIdx++
		}
	}

	return m, nil
}

// handleEscape processes escape key.
func (m Model) handleEscape() (tea.Model, tea.Cmd) {
	now := time.Now().UnixNano()
	if m.lastEscTime > 0 && (now-m.lastEscTime) < 500_000_000 {
		// Double ESC - quit
		m.quitting = true
		return m, tea.Quit
	}
	m.lastEscTime = now
	return m, nil
}

// moveCursor moves the cursor by delta lines and columns.
func (m *Model) moveCursor(dLine, dCol int) {
	total := m.TotalLines()
	if total == 0 {
		return
	}

	// Move line
	m.cursorLine += dLine
	if m.cursorLine < 0 {
		m.cursorLine = 0
	}
	if m.cursorLine >= total {
		m.cursorLine = total - 1
	}

	// Move column
	lines := m.GetLines()
	if m.cursorLine < len(lines) {
		lineLen := len(lines[m.cursorLine])
		m.cursorCol += dCol
		if m.cursorCol < 0 {
			m.cursorCol = 0
		}
		if m.cursorCol > lineLen {
			m.cursorCol = lineLen
		}
	}

	// Adjust scroll
	visibleHeight := m.height - 6 // Account for status bar etc
	if m.cursorLine < m.scrollOffset {
		m.scrollOffset = m.cursorLine
	}
	if m.cursorLine >= m.scrollOffset+visibleHeight {
		m.scrollOffset = m.cursorLine - visibleHeight + 1
	}
}

// enterEditMode enters line editing mode.
func (m *Model) enterEditMode() {
	lines := m.GetLines()
	isNewDocument := len(lines) == 0

	// Handle empty document - create a new document with a placeholder line
	if isNewDocument {
		// Create a new document with underscore (minimal valid calc block)
		newDoc, err := document.NewDocument("_")
		if err == nil {
			m.doc = newDoc
			m.eval = implDoc.NewEvaluator()
			_ = m.eval.Evaluate(m.doc)
			m.pushUndoState()
			lines = m.GetLines()
		}
	}

	if m.cursorLine >= len(lines) {
		m.cursorLine = len(lines) - 1
	}
	if m.cursorLine < 0 {
		m.cursorLine = 0
	}

	if m.cursorLine < len(lines) {
		lineContent := lines[m.cursorLine]
		// For newly created document, start with empty buffer (not the placeholder)
		if isNewDocument {
			m.editBuf = ""
		} else {
			m.editBuf = lineContent
		}
		m.mode = ModeEditing
		m.cursorCol = len(m.editBuf) // Position cursor at end
	}
}

// exitEditMode exits line editing mode.
func (m *Model) exitEditMode(save bool) {
	if save && m.mode == ModeEditing {
		// Find and update the block containing this line
		m.updateCurrentLine(m.editBuf)
		m.modified = true
		m.pushUndoState()

		// Re-detect block types in case content changed from calc to text or vice versa
		m.redetectBlockTypes()

		// Re-evaluate affected blocks
		m.reEvaluate()
	}
	m.mode = ModeNormal
	m.editBuf = ""
}

// redetectBlockTypes rebuilds the document to properly detect block types.
// This is needed when editing changes a line from calculation to markdown or vice versa.
func (m *Model) redetectBlockTypes() {
	// Get current document content
	content := m.getDocumentContent()

	// Rebuild document with proper block detection
	newDoc, err := document.NewDocument(content)
	if err != nil {
		// If parsing fails, keep the old document
		return
	}

	// Preserve cursor position
	cursorLine := m.cursorLine
	cursorCol := m.cursorCol

	// Replace document
	m.doc = newDoc

	// Re-evaluate the new document
	m.eval = implDoc.NewEvaluator()
	_ = m.eval.Evaluate(m.doc)

	// Restore cursor (clamped to valid range)
	total := m.TotalLines()
	if cursorLine >= total {
		cursorLine = total - 1
	}
	if cursorLine < 0 {
		cursorLine = 0
	}
	m.cursorLine = cursorLine
	m.cursorCol = cursorCol

	// Auto-pin any new variables
	m.autoPinVariables()
}

// updateCurrentLine updates the line at cursorLine with new content.
func (m *Model) updateCurrentLine(newContent string) {
	lineIdx := 0
	for _, node := range m.doc.GetBlocks() {
		var blockLines []string
		switch b := node.Block.(type) {
		case *document.CalcBlock:
			blockLines = b.Source()
		case *document.TextBlock:
			blockLines = b.Source()
		}

		for i := range blockLines {
			if lineIdx == m.cursorLine {
				// This is the line to update
				blockLines[i] = newContent

				// Replace block source
				result, err := m.doc.ReplaceBlockSource(node.ID, blockLines)
				if err != nil {
					return
				}

				// Track affected blocks
				for _, id := range result.AffectedBlockIDs {
					m.changedBlockIDs[id] = true
				}
				return
			}
			lineIdx++
		}
	}
}

// insertLineBelow inserts a new line below the cursor.
func (m *Model) insertLineBelow() {
	m.insertLine(m.cursorLine + 1)
}

// insertLineAbove inserts a new line above the cursor.
func (m *Model) insertLineAbove() {
	m.insertLine(m.cursorLine)
}

// insertLine inserts a new line at the given position.
func (m *Model) insertLine(at int) {
	// For now, just add an empty calc block
	blocks := m.doc.GetBlocks()
	if len(blocks) == 0 {
		return
	}

	// Find block at position
	lineIdx := 0
	for _, node := range blocks {
		var blockLines []string
		switch b := node.Block.(type) {
		case *document.CalcBlock:
			blockLines = b.Source()
		case *document.TextBlock:
			blockLines = b.Source()
		}

		if lineIdx+len(blockLines) >= at {
			// Insert after this block
			_, _ = m.doc.InsertBlock(node.ID, document.BlockCalculation, []string{""})
			m.cursorLine = at
			m.cursorCol = 0
			m.modified = true
			m.pushUndoState()
			return
		}
		lineIdx += len(blockLines)
	}
}

// reEvaluate re-evaluates affected blocks after an edit.
func (m *Model) reEvaluate() {
	m.changedVars = make(map[string]bool)

	// Use EvaluateAffectedBlocks for incremental evaluation
	if len(m.changedBlockIDs) > 0 {
		affectedIDs := make([]string, 0, len(m.changedBlockIDs))
		for id := range m.changedBlockIDs {
			affectedIDs = append(affectedIDs, id)
		}

		orderedBlocks := m.doc.GetBlocksInDependencyOrder(affectedIDs)
		m.eval.EvaluateAffectedBlocks(m.doc, orderedBlocks)

		// Track changed variables
		for _, id := range orderedBlocks {
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
	}

	m.changedBlockIDs = make(map[string]bool)
}

// undo reverts to the previous state.
func (m *Model) undo() {
	if len(m.undoStack) <= 1 {
		return
	}

	// Save current state to redo
	current := m.undoStack[len(m.undoStack)-1]
	m.redoStack = append(m.redoStack, current)

	// Pop and restore previous state
	m.undoStack = m.undoStack[:len(m.undoStack)-1]
	prev := m.undoStack[len(m.undoStack)-1]

	// Restore document
	doc, err := document.NewDocument(prev)
	if err != nil {
		return
	}
	m.doc = doc
	m.eval = implDoc.NewEvaluator()
	_ = m.eval.Evaluate(m.doc)
	m.modified = true
}

// redo re-applies an undone change.
func (m *Model) redo() {
	if len(m.redoStack) == 0 {
		return
	}

	// Pop from redo and apply
	content := m.redoStack[len(m.redoStack)-1]
	m.redoStack = m.redoStack[:len(m.redoStack)-1]

	doc, err := document.NewDocument(content)
	if err != nil {
		return
	}
	m.doc = doc
	m.eval = implDoc.NewEvaluator()
	_ = m.eval.Evaluate(m.doc)

	m.undoStack = append(m.undoStack, content)
	m.modified = true
}

// executeCommand executes a slash command.
func (m *Model) executeCommand(cmd string) {
	cmd = strings.TrimPrefix(cmd, "/")
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "save", "w":
		var filename string
		if len(parts) > 1 {
			filename = parts[1]
		}
		m.saveFile(filename)
	case "open", "o":
		if len(parts) > 1 {
			m.openFile(parts[1])
		} else {
			m.statusMsg = "Usage: /open <filename>"
			m.statusIsErr = true
		}
	case "quit", "q":
		m.quitting = true
	case "wq":
		// Save and quit
		m.saveFile("")
		if !m.statusIsErr {
			m.quitting = true
		}
	case "preview":
		// /preview cycles, /preview full|minimal|hidden sets specific mode
		if len(parts) == 1 {
			m.cyclePreviewMode()
		} else {
			switch parts[1] {
			case "full":
				m.previewMode = PreviewFull
			case "minimal", "min":
				m.previewMode = PreviewMinimal
			case "hidden", "hide", "off":
				m.previewMode = PreviewHidden
			default:
				m.statusMsg = "Usage: /preview [full|minimal|hidden]"
				m.statusIsErr = true
			}
		}
	case "undo", "u":
		m.undo()
	case "redo":
		m.redo()
	case "find", "f", "search":
		if len(parts) > 1 {
			term := strings.Join(parts[1:], " ")
			m.searchDocument(term)
		} else {
			m.statusMsg = "Usage: /find <term>"
			m.statusIsErr = true
		}
	case "goto", "go":
		if len(parts) > 1 {
			m.gotoLine(parts[1])
		} else {
			m.statusMsg = "Usage: /goto <line>"
			m.statusIsErr = true
		}
	case "help", "h", "?":
		m.statusMsg = "e=edit j/k=nav n/N=search /save /open /quit /preview /find /goto"
	default:
		m.statusMsg = fmt.Sprintf("Unknown command: %s", parts[0])
		m.statusIsErr = true
	}
}

// saveFile saves the document to a file.
func (m *Model) saveFile(filename string) {
	// Use provided filename or current filepath
	if filename == "" {
		filename = m.filepath
	}
	if filename == "" {
		m.statusMsg = "No filename. Use /save <filename>"
		m.statusIsErr = true
		return
	}

	// Ensure .cm extension
	if !strings.HasSuffix(filename, ".cm") {
		filename = filename + ".cm"
	}

	// Get absolute path
	absPath, err := filepath.Abs(filename)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Invalid path: %v", err)
		m.statusIsErr = true
		return
	}

	// Get document content
	content := m.getDocumentContent()

	// Write file
	err = os.WriteFile(absPath, []byte(content), 0644)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Save failed: %v", err)
		m.statusIsErr = true
		return
	}

	// Update state
	m.filepath = absPath
	m.modified = false
	m.statusMsg = fmt.Sprintf("Saved: %s", filepath.Base(absPath))
}

// openFile opens a file into the editor.
func (m *Model) openFile(filename string) {
	// Get absolute path
	absPath, err := filepath.Abs(filename)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Invalid path: %v", err)
		m.statusIsErr = true
		return
	}

	// Read file
	content, err := os.ReadFile(absPath)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Open failed: %v", err)
		m.statusIsErr = true
		return
	}

	// Parse document
	doc, err := document.NewDocument(string(content))
	if err != nil {
		m.statusMsg = fmt.Sprintf("Parse error: %v", err)
		m.statusIsErr = true
		return
	}

	// Evaluate
	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		// Non-fatal - document loaded but has evaluation errors
		m.statusMsg = fmt.Sprintf("Opened with errors: %v", err)
		m.statusIsErr = true
	} else {
		m.statusMsg = fmt.Sprintf("Opened: %s", filepath.Base(absPath))
	}

	// Update model state
	m.doc = doc
	m.eval = eval
	m.filepath = absPath
	m.modified = false
	m.cursorLine = 0
	m.cursorCol = 0
	m.scrollOffset = 0

	// Reset undo stack
	m.undoStack = []string{}
	m.redoStack = []string{}
	m.pushUndoState()

	// Auto-pin variables
	m.pinnedVars = make(map[string]bool)
	m.changedVars = make(map[string]bool)
	m.autoPinVariables()
}

// getGlobalsCount returns the number of global variables.
func (m *Model) getGlobalsCount() int {
	fm := m.doc.GetFrontmatter()
	if fm == nil {
		return 0
	}
	return len(fm.Globals) + len(fm.Exchange)
}

// GetStatusBarState returns state for the status bar.
func (m *Model) GetStatusBarState() components.StatusBarState {
	modeStr := ""
	switch m.mode {
	case ModeNormal:
		modeStr = "NORMAL"
	case ModeEditing:
		modeStr = "EDITING"
	case ModeCommand:
		modeStr = "COMMAND"
	case ModeGlobals:
		modeStr = "GLOBALS"
	case ModeHelp:
		modeStr = "HELP"
	}

	// Build hints with preview mode indicator
	previewHint := ""
	switch m.previewMode {
	case PreviewFull:
		previewHint = "Tab:min"
	case PreviewMinimal:
		previewHint = "Tab:hide"
	case PreviewHidden:
		previewHint = "Tab:full"
	}

	hints := ""
	switch m.mode {
	case ModeNormal:
		hints = fmt.Sprintf("e=edit j/k=↑↓ %s /=cmd", previewHint)
	case ModeEditing:
		hints = "Esc=done"
	case ModeCommand:
		hints = "Enter=run Esc=cancel"
	}

	return components.StatusBarState{
		Filename:    m.filepath,
		Line:        m.cursorLine + 1,
		TotalLines:  m.TotalLines(),
		CalcCount:   m.CalcBlockCount(),
		Modified:    m.modified,
		Mode:        modeStr,
		Hints:       hints,
		StatusMsg:   m.statusMsg,
		StatusIsErr: m.statusIsErr,
	}
}

// GetPinnedPanelState returns state for the pinned panel.
func (m *Model) GetPinnedPanelState(height int) components.PinnedPanelState {
	vars := m.collectPinnedVariables()
	return components.PinnedPanelState{
		Variables: vars,
		ScrollY:   0,
		Height:    height,
	}
}

// collectPinnedVariables gathers pinned variables for display.
func (m *Model) collectPinnedVariables() []components.PinnedVar {
	var result []components.PinnedVar
	seen := make(map[string]bool)

	// Track frontmatter variables
	fmVars := make(map[string]bool)
	if fm := m.doc.GetFrontmatter(); fm != nil {
		for name := range fm.Globals {
			fmVars[name] = true
		}
	}

	// Collect in document order
	for _, node := range m.doc.GetBlocks() {
		if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
			for _, varName := range calcBlock.Variables() {
				if !m.pinnedVars[varName] || seen[varName] {
					continue
				}
				seen[varName] = true

				valueStr := "?"
				if m.eval != nil {
					env := m.eval.GetEnvironment()
					if val, ok := env.Get(varName); ok {
						valueStr = display.Format(val)
					}
				}

				result = append(result, components.PinnedVar{
					Name:          varName,
					Value:         valueStr,
					Changed:       m.changedVars[varName],
					IsFrontmatter: fmVars[varName],
				})
			}
		}
	}

	return result
}

// GetGlobalsPanelState returns state for the globals panel.
func (m *Model) GetGlobalsPanelState() components.GlobalsPanelState {
	var globals []components.GlobalVar

	fm := m.doc.GetFrontmatter()
	if fm != nil {
		for name, value := range fm.Globals {
			globals = append(globals, components.GlobalVar{
				Name:       name,
				Value:      fmt.Sprintf("%v", value),
				IsExchange: false,
			})
		}
		for name, rate := range fm.Exchange {
			globals = append(globals, components.GlobalVar{
				Name:       name,
				Value:      rate.StringFixed(4),
				IsExchange: true,
			})
		}
	}

	return components.GlobalsPanelState{
		Globals:    globals,
		Expanded:   m.globalsExpanded,
		FocusIndex: m.globalsFocusIdx,
		Focused:    m.mode == ModeGlobals,
	}
}

// LineResult represents a line's evaluation result.
type LineResult struct {
	LineNum    int
	Source     string
	IsCalc     bool
	VarName    string
	Value      string
	Error      string
	BlockID    string
	WasChanged bool
}

// GetLineResults returns evaluation results for all lines.
// Each source line maps to its corresponding statement result when available.
func (m *Model) GetLineResults() []LineResult {
	var results []LineResult
	lineNum := 0

	for _, node := range m.doc.GetBlocks() {
		switch b := node.Block.(type) {
		case *document.CalcBlock:
			sourceLines := b.Source()
			vars := b.Variables()
			stmtResults := b.Results()    // Per-statement results
			statements := b.Statements()  // Parsed AST nodes
			blockError := b.Error()

			// Build a map of variable index for lookup
			// Variables are in definition order, so vars[i] corresponds to the
			// i-th assignment statement
			varIndex := 0

			for i, line := range sourceLines {
				lr := LineResult{
					LineNum:    lineNum,
					Source:     line,
					IsCalc:     true,
					BlockID:    node.ID,
					WasChanged: m.changedBlockIDs[node.ID],
				}

				// Skip empty/whitespace-only lines (no result to show)
				trimmed := line
				for _, c := range line {
					if c != ' ' && c != '\t' {
						trimmed = line
						break
					}
				}
				if len(trimmed) == 0 || trimmed == "" {
					results = append(results, lr)
					lineNum++
					continue
				}

				// If there's a block-level error, show it on first non-empty line
				if blockError != nil && i == 0 {
					lr.Error = blockError.Error()
					results = append(results, lr)
					lineNum++
					continue
				}

				// Map source line index to statement index
				// This is approximate - assumes 1:1 mapping of non-empty lines to statements
				stmtIdx := m.countNonEmptyLinesBefore(sourceLines, i)

				// Get result for this statement if available
				if stmtIdx < len(stmtResults) && stmtResults[stmtIdx] != nil {
					lr.Value = display.Format(stmtResults[stmtIdx])
				}

				// Get variable name if this statement defines one
				if stmtIdx < len(statements) {
					if varName := m.getAssignmentVarName(statements[stmtIdx]); varName != "" {
						lr.VarName = varName
					} else if varIndex < len(vars) {
						// Fallback: use vars list in order
						lr.VarName = vars[varIndex]
						varIndex++
					}
				}

				results = append(results, lr)
				lineNum++
			}

		case *document.TextBlock:
			for _, line := range b.Source() {
				results = append(results, LineResult{
					LineNum: lineNum,
					Source:  line,
					IsCalc:  false,
					BlockID: node.ID,
				})
				lineNum++
			}
		}
	}

	return results
}

// countNonEmptyLinesBefore counts non-empty lines before index i.
func (m *Model) countNonEmptyLinesBefore(lines []string, i int) int {
	count := 0
	for j := 0; j < i; j++ {
		if strings.TrimSpace(lines[j]) != "" {
			count++
		}
	}
	return count
}

// getAssignmentVarName extracts the variable name from an assignment AST node.
func (m *Model) getAssignmentVarName(node ast.Node) string {
	switch n := node.(type) {
	case *ast.Assignment:
		return n.Name
	case *ast.FrontmatterAssignment:
		return n.Property
	}
	return ""
}

// Quitting returns whether the editor should quit.
func (m Model) Quitting() bool {
	return m.quitting
}

// Document returns the current document.
func (m Model) Document() *document.Document {
	return m.doc
}

// Mode returns the current editor mode.
func (m Model) Mode() EditorMode {
	return m.mode
}

// CursorLine returns the current cursor line.
func (m Model) CursorLine() int {
	return m.cursorLine
}

// CursorCol returns the current cursor column.
func (m Model) CursorCol() int {
	return m.cursorCol
}

// ScrollOffset returns the current scroll offset.
func (m Model) ScrollOffset() int {
	return m.scrollOffset
}

// ShowPreview returns whether preview pane is visible (not hidden).
func (m Model) ShowPreview() bool {
	return m.previewMode != PreviewHidden
}

// PreviewModeValue returns the current preview mode.
func (m Model) PreviewModeValue() PreviewMode {
	return m.previewMode
}

// cyclePreviewMode cycles through preview modes: Full → Minimal → Hidden → Full
func (m *Model) cyclePreviewMode() {
	switch m.previewMode {
	case PreviewFull:
		m.previewMode = PreviewMinimal
	case PreviewMinimal:
		m.previewMode = PreviewHidden
	case PreviewHidden:
		m.previewMode = PreviewFull
	}
}

// Width returns the current width.
func (m Model) Width() int {
	return m.width
}

// Height returns the current height.
func (m Model) Height() int {
	return m.height
}

// SetMode sets the editor mode.
func (m *Model) SetMode(mode EditorMode) {
	m.mode = mode
}

// CommandInput returns the current command input.
func (m Model) CommandInput() string {
	return m.cmdInput
}

// IsModified returns whether the document has unsaved changes.
func (m Model) IsModified() bool {
	return m.modified
}

// FilePath returns the current file path.
func (m Model) FilePath() string {
	return m.filepath
}

// Key returns the key map.
func (m Model) Key() shared.KeyMap {
	return shared.DefaultKeyMap()
}

// deleteLine deletes the current line (dd command).
func (m *Model) deleteLine() {
	lines := m.GetLines()
	if m.cursorLine >= len(lines) {
		return
	}

	// Copy to yank buffer first
	m.yankBuffer = lines[m.cursorLine]

	// Find and update the block containing this line
	lineIdx := 0
	for _, node := range m.doc.GetBlocks() {
		var blockLines []string
		switch b := node.Block.(type) {
		case *document.CalcBlock:
			blockLines = b.Source()
		case *document.TextBlock:
			blockLines = b.Source()
		}

		for i := range blockLines {
			if lineIdx == m.cursorLine {
				// Remove this line from the block
				newLines := make([]string, 0, len(blockLines)-1)
				newLines = append(newLines, blockLines[:i]...)
				newLines = append(newLines, blockLines[i+1:]...)

				if len(newLines) == 0 {
					// Block is now empty - delete it
					m.doc.DeleteBlock(node.ID)
				} else {
					// Replace block source
					m.doc.ReplaceBlockSource(node.ID, newLines)
				}

				m.modified = true
				m.pushUndoState()
				m.reEvaluate()

				// Adjust cursor if needed
				total := m.TotalLines()
				if m.cursorLine >= total && total > 0 {
					m.cursorLine = total - 1
				}
				m.statusMsg = "Line deleted"
				return
			}
			lineIdx++
		}
	}
}

// yankLine copies the current line to the yank buffer (yy command).
func (m *Model) yankLine() {
	lines := m.GetLines()
	if m.cursorLine >= len(lines) {
		return
	}
	m.yankBuffer = lines[m.cursorLine]
	m.statusMsg = "Line yanked"
}

// pasteLine pastes the yank buffer below the current line (p command).
func (m *Model) pasteLine() {
	if m.yankBuffer == "" {
		return
	}

	// Insert a new line below cursor with yanked content
	m.insertLineBelow()
	m.updateCurrentLine(m.yankBuffer)
	m.modified = true
	m.pushUndoState()
	m.reEvaluate()
	m.statusMsg = "Line pasted"
}

// pasteLineAbove pastes the yank buffer above the current line (P command).
func (m *Model) pasteLineAbove() {
	if m.yankBuffer == "" {
		return
	}

	// Insert a new line above cursor with yanked content
	m.insertLineAbove()
	m.updateCurrentLine(m.yankBuffer)
	m.modified = true
	m.pushUndoState()
	m.reEvaluate()
	m.statusMsg = "Line pasted above"
}

// searchDocument searches for a term and highlights matches.
func (m *Model) searchDocument(term string) {
	m.searchTerm = term
	m.searchMatches = nil
	m.searchIdx = -1

	lines := m.GetLines()
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), strings.ToLower(term)) {
			m.searchMatches = append(m.searchMatches, i)
		}
	}

	if len(m.searchMatches) == 0 {
		m.statusMsg = fmt.Sprintf("No matches for: %s", term)
		m.statusIsErr = true
		return
	}

	// Jump to first match at or after cursor
	for i, lineNum := range m.searchMatches {
		if lineNum >= m.cursorLine {
			m.searchIdx = i
			m.cursorLine = lineNum
			m.adjustScroll()
			break
		}
	}
	if m.searchIdx == -1 {
		// All matches are before cursor, go to first
		m.searchIdx = 0
		m.cursorLine = m.searchMatches[0]
		m.adjustScroll()
	}

	m.statusMsg = fmt.Sprintf("Match %d of %d: %s", m.searchIdx+1, len(m.searchMatches), term)
}

// nextSearchMatch jumps to the next search match.
func (m *Model) nextSearchMatch() {
	if len(m.searchMatches) == 0 {
		m.statusMsg = "No search active (use /find <term>)"
		m.statusIsErr = true
		return
	}

	m.searchIdx = (m.searchIdx + 1) % len(m.searchMatches)
	m.cursorLine = m.searchMatches[m.searchIdx]
	m.adjustScroll()
	m.statusMsg = fmt.Sprintf("Match %d of %d: %s", m.searchIdx+1, len(m.searchMatches), m.searchTerm)
}

// prevSearchMatch jumps to the previous search match.
func (m *Model) prevSearchMatch() {
	if len(m.searchMatches) == 0 {
		m.statusMsg = "No search active (use /find <term>)"
		m.statusIsErr = true
		return
	}

	m.searchIdx--
	if m.searchIdx < 0 {
		m.searchIdx = len(m.searchMatches) - 1
	}
	m.cursorLine = m.searchMatches[m.searchIdx]
	m.adjustScroll()
	m.statusMsg = fmt.Sprintf("Match %d of %d: %s", m.searchIdx+1, len(m.searchMatches), m.searchTerm)
}

// gotoLine jumps to a specific line number.
func (m *Model) gotoLine(lineStr string) {
	var lineNum int
	if _, err := fmt.Sscanf(lineStr, "%d", &lineNum); err != nil {
		m.statusMsg = fmt.Sprintf("Invalid line number: %s", lineStr)
		m.statusIsErr = true
		return
	}

	// Convert to 0-indexed
	lineNum--

	total := m.TotalLines()
	if lineNum < 0 {
		lineNum = 0
	}
	if lineNum >= total {
		lineNum = total - 1
	}

	m.cursorLine = lineNum
	m.adjustScroll()
	m.statusMsg = fmt.Sprintf("Line %d", lineNum+1)
}

// adjustScroll ensures cursor is visible.
func (m *Model) adjustScroll() {
	visibleHeight := m.height - 6
	if m.cursorLine < m.scrollOffset {
		m.scrollOffset = m.cursorLine
	}
	if m.cursorLine >= m.scrollOffset+visibleHeight {
		m.scrollOffset = m.cursorLine - visibleHeight + 1
	}
}
