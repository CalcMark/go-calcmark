package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/geometry"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/shared"
	"github.com/CalcMark/go-calcmark/format"
	"github.com/CalcMark/go-calcmark/format/display"
	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
	tea "github.com/charmbracelet/bubbletea"
)

// Debounce delay for re-evaluation after typing (per spec: ~50ms)
const evalDebounceDelay = 50 * time.Millisecond

// evalDebounceMsg is sent after the debounce delay to trigger evaluation.
type evalDebounceMsg struct {
	editBufSnapshot string // Snapshot of editBuf when timer was started
}

// InputState determines the UI context: what receives input and what auxiliary UI should display.
// IMPORTANT: This is NOT a modal editing system (like vim's normal/insert modes).
// The user is ALWAYS editing the document - typing and navigation work continuously.
// InputState controls:
//   - Which UI component processes keyboard input
//   - What auxiliary UI elements are shown (preview pane updates, errors, help)
//   - What auxiliary UI elements are hidden (irrelevant errors/help for the current context)
type InputState int

const (
	StateDefault      InputState = iota // Normal document editing with live preview and error display
	StateGlobals                        // Globals panel active, preview shows global values
	StateHelp                           // Help viewer active, preview shows help content
	StateExportFormat                   // Export dialog active, normal UI paused
	StateExportPath                     // Export path input active, normal UI paused
	StateSavePrompt                     // Save confirmation dialog active, normal UI paused
	StateSaveAsPath                     // Save-as filename input active, normal UI paused
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
	modified bool // True if document has unsaved changes

	// Save state tracking
	savedContent string // Content as it was at last save (for detecting changes)

	// Cursor and navigation
	cursorLine   int // Current line (0-indexed)
	cursorCol    int // Current column (0-indexed)
	scrollOffset int // Vertical scroll offset

	// Editor state
	state           EditorState     // Current editing state (StateReady, StateEditing, StateProcessing)
	mode            InputState      // Which UI component receives input (NOT a vim-style editing mode)
	userIsTyping    bool            // True when user is actively typing (for debounce)
	editBuf         string          // Buffer for line being edited
	lineWrap        bool            // Whether to wrap long lines
	changedBlockIDs map[string]bool // Track changed blocks for highlighting

	// Undo/redo
	undoStack []string // Document content snapshots
	redoStack []string

	// Export state
	exportFormat     string   // Selected export format (text, json, html, md)
	exportPath       string   // Path being entered for export
	exportFormatOpts []string // Available export formats

	// Save state
	saveAsPath string // Path being entered for save-as

	// Globals panel
	globalsExpanded bool
	globalsFocusIdx int

	// Pinned variables
	pinnedVars  map[string]bool
	changedVars map[string]bool

	// UI state
	width       int
	height      int
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

	// Cached alignment model - computed once and invalidated on changes
	alignedCache       *AlignedModel
	alignedCacheKey    alignedCacheKey // Key for cache validation
	alignedCacheWidths [2]int          // [sourceWidth, previewWidth] used for cache
}

// New creates a new editor model with an optional document.
// This is the ONLY place where editor state is initialized.
// After this, the editor is ALWAYS in StateReady with all invariants satisfied.
func New(doc *document.Document) Model {
	// User is ALWAYS able to edit - mode only represents temporary UI overlays
	m := Model{
		doc:              doc,
		eval:             nil,
		mode:             StateDefault,
		userIsTyping:     false,
		pinnedVars:       make(map[string]bool),
		changedVars:      make(map[string]bool),
		changedBlockIDs:  make(map[string]bool),
		undoStack:        []string{},
		redoStack:        []string{},
		exportFormatOpts: []string{"text", "cm", "json", "html", "md"},
		width:            80,
		height:           24,
		previewMode:      PreviewFull,
		lineWrap:         true,
		styles:           config.GetStyles(),
	}

	// CRITICAL: Transition to StateReady - establishes all invariants
	// This is the ONLY state transition during initialization
	m.transitionToReady()

	// Auto-pin all variables
	m.autoPinVariables()

	// Save initial state for undo
	m.pushUndoState()

	// Initialize savedContent to current content (new documents start "saved")
	m.savedContent = m.getDocumentContent()

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
// CRITICAL: Returns content with trailing newline to preserve line count.
// See unicode.go fix - trailing newlines no longer create extra lines,
// so we MUST include them when reconstructing to preserve N lines.
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
	if len(lines) == 0 {
		return ""
	}
	// Append trailing newline to preserve last line
	return strings.Join(lines, "\n") + "\n"
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
		m.InvalidateAlignedCache()

	case evalDebounceMsg:
		// Only evaluate if editBuf hasn't changed since the timer was started
		// This ensures we don't evaluate stale content
		if m.editBuf == msg.editBufSnapshot {
			// Transition to processing - this will update the line, re-evaluate, and transition to ready
			m.transitionToProcessing()
		}
	}

	return m, nil
}

// handleKey processes keyboard input.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Invalidate aligned model cache - state may change
	m.InvalidateAlignedCache()

	// Clear status message on any key
	m.statusMsg = ""
	m.statusIsErr = false

	// Global quit handlers
	switch msg.Type {
	case tea.KeyCtrlC:
		// Ctrl+C is a standard Unix interrupt signal - quit immediately without prompts
		// This is the emergency exit - users expect this to always work
		m.quitting = true
		return m, tea.Quit
	case tea.KeyCtrlQ:
		// Ctrl+Q is the dedicated quit command
		// Check for unsaved changes before quitting
		if m.hasUnsavedChanges() {
			m.mode = StateSavePrompt
			m.statusMsg = "Unsaved changes! Save before quit? (y/n/c)"
			return m, nil
		}
		m.quitting = true
		return m, tea.Quit
	case tea.KeyCtrlS:
		// Save (Ctrl+S works in all modes)
		// If no filename, prompt for one
		if m.filepath == "" {
			m.mode = StateSaveAsPath
			m.saveAsPath = ""
			m.statusMsg = "Save as (filename):"
			return m, nil
		}
		m.saveFile("")
		return m, nil
	case tea.KeyCtrlE:
		// Export (Ctrl+E works in all modes)
		m.enterExportMode()
		return m, nil
	}

	// Mode-specific handling for UI overlays
	switch m.mode {
	case StateGlobals:
		return m.handleGlobalsKey(msg)
	case StateExportFormat:
		return m.handleExportFormatKey(msg)
	case StateExportPath:
		return m.handleExportPathKey(msg)
	case StateSavePrompt:
		return m.handleSavePromptKey(msg)
	case StateSaveAsPath:
		return m.handleSaveAsPathKey(msg)
	default:
		// StateDefault - user is always editing
		return m.handleDefaultKey(msg)
	}
}

// handleDefaultKey processes keys in the default editing mode.
// The user is ALWAYS able to type and edit - this is the only mode they experience.
func (m Model) handleDefaultKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		return m.handleUpKey()
	case tea.KeyDown:
		return m.handleDownKey()
	case tea.KeyLeft:
		return m.handleLeftKey()
	case tea.KeyRight:
		return m.handleRightKey()
	case tea.KeyPgUp:
		return m.handlePageUpKey()
	case tea.KeyPgDown:
		return m.handlePageDownKey()
	case tea.KeyHome:
		return m.handleHomeKey()
	case tea.KeyEnd:
		return m.handleEndKey()
	case tea.KeyEsc:
		// ESC does nothing in normal editing mode - it's only for canceling special modes
		// (like globals panel, export mode, save-as dialog, etc.)
		return m, nil
	case tea.KeyEnter:
		return m.handleEnterKey()
	case tea.KeyBackspace:
		return m.handleBackspaceKey()
	case tea.KeyDelete:
		return m.handleDeleteKey()
	case tea.KeyCtrlP:
		return m.handleCtrlP()
	case tea.KeyCtrlD:
		return m.handleCtrlD()
	case tea.KeyCtrlU:
		return m.handleCtrlU()
	case tea.KeySpace:
		return m.handleSpaceKey()
	case tea.KeyRunes:
		return m.handleRuneInput(msg.Runes)
	}

	return m, nil
}

// handleRuneInput handles character input - regular typing only.
// All vim keys have been removed - user is ALWAYS in editing mode.
func (m Model) handleRuneInput(runes []rune) (tea.Model, tea.Cmd) {
	if len(runes) == 0 {
		return m, nil
	}

	// Insert all characters at cursor position
	m.transitionToEditing()

	for _, r := range runes {
		m.insertRune(r)
	}

	return m.debounceUpdate()
}

// ========================================
// Clean key handler helper functions
// ========================================

// Navigation keys
func (m Model) handleUpKey() (tea.Model, tea.Cmd) {
	m.loadCurrentLineIntoEditBuffer()
	if m.cursorLine > 0 {
		m.saveCurrentLineAndMoveTo(m.cursorLine - 1)
	}
	return m, nil
}

func (m Model) handleDownKey() (tea.Model, tea.Cmd) {
	m.loadCurrentLineIntoEditBuffer()
	if m.cursorLine < m.TotalLines()-1 {
		m.saveCurrentLineAndMoveTo(m.cursorLine + 1)
	}
	return m, nil
}

func (m Model) handleLeftKey() (tea.Model, tea.Cmd) {
	m.loadCurrentLineIntoEditBuffer()
	if m.cursorCol > 0 {
		m.cursorCol--
	} else if m.cursorLine > 0 {
		// At start of line - move to end of previous line
		m.saveCurrentLineAndMoveTo(m.cursorLine - 1)
		m.cursorCol = len(m.editBuf)
	}
	return m, nil
}

func (m Model) handleRightKey() (tea.Model, tea.Cmd) {
	m.loadCurrentLineIntoEditBuffer()
	if m.cursorCol < len(m.editBuf) {
		m.cursorCol++
	} else if m.cursorLine < m.TotalLines()-1 {
		// At end of line - move to start of next line
		m.saveCurrentLineAndMoveTo(m.cursorLine + 1)
		m.cursorCol = 0
	}
	return m, nil
}

func (m Model) handlePageUpKey() (tea.Model, tea.Cmd) {
	m.loadCurrentLineIntoEditBuffer()
	m.moveCursor(-(m.height - 4), 0)
	return m, nil
}

func (m Model) handlePageDownKey() (tea.Model, tea.Cmd) {
	m.loadCurrentLineIntoEditBuffer()
	m.moveCursor(m.height-4, 0)
	return m, nil
}

func (m Model) handleHomeKey() (tea.Model, tea.Cmd) {
	m.loadCurrentLineIntoEditBuffer()
	m.cursorCol = 0
	return m, nil
}

func (m Model) handleEndKey() (tea.Model, tea.Cmd) {
	m.loadCurrentLineIntoEditBuffer()
	m.cursorCol = len(m.editBuf)
	return m, nil
}

// Content modification keys
func (m Model) handleEscKey() (tea.Model, tea.Cmd) {
	m.loadCurrentLineIntoEditBuffer()

	// Save current line
	m.updateCurrentLine(m.editBuf)

	// Insert new line below
	// insertLineBelow() sets cursor to the new line, so no need to increment
	m.insertLineBelow()
	m.editBuf = ""
	m.cursorCol = 0

	// Process document changes immediately on ESC
	m.redetectBlockTypes()
	m.reEvaluate()
	m.pushUndoState()
	m.modified = true
	m.userIsTyping = false

	return m, nil
}

func (m Model) handleEnterKey() (tea.Model, tea.Cmd) {
	m.loadCurrentLineIntoEditBuffer()

	// Split line at cursor position
	textBefore := m.editBuf[:m.cursorCol]
	textAfter := m.editBuf[m.cursorCol:]

	// Save current line with text before cursor
	m.editBuf = textBefore
	m.updateCurrentLine(m.editBuf)

	// Insert new line below with text after cursor
	// insertLineBelow() sets cursor to the new line, so no need to increment
	m.insertLineBelow()
	m.editBuf = textAfter
	m.cursorCol = 0

	// Process document changes immediately on ENTER
	m.redetectBlockTypes()
	m.reEvaluate()
	m.pushUndoState()
	m.modified = true
	m.userIsTyping = false

	return m, nil
}

func (m Model) handleBackspaceKey() (tea.Model, tea.Cmd) {
	m.transitionToEditing()

	if m.cursorCol > 0 && len(m.editBuf) > 0 {
		// Delete character before cursor
		m.editBuf = m.editBuf[:m.cursorCol-1] + m.editBuf[m.cursorCol:]
		m.cursorCol--
		return m.debounceUpdate()
	} else if len(m.editBuf) == 0 && m.cursorLine > 0 {
		// Empty line - join with previous line
		prevLine := m.cursorLine - 1
		m.deleteLine()
		m.cursorLine = prevLine
		lines := m.GetLines()
		if m.cursorLine < len(lines) {
			m.editBuf = lines[m.cursorLine]
			m.cursorCol = len(m.editBuf)
		}
		m.transitionToEditing()
		return m.debounceUpdate()
	}

	return m, nil
}

func (m Model) handleDeleteKey() (tea.Model, tea.Cmd) {
	m.loadCurrentLineIntoEditBuffer()

	if m.cursorCol < len(m.editBuf) {
		// Delete character at cursor
		m.editBuf = m.editBuf[:m.cursorCol] + m.editBuf[m.cursorCol+1:]
		m.transitionToEditing()
		return m.debounceUpdate()
	} else if len(m.editBuf) == 0 {
		// Empty line - delete it
		m.deleteLine()
		if m.TotalLines() > 0 {
			if m.cursorLine >= m.TotalLines() {
				m.cursorLine = m.TotalLines() - 1
			}
			lines := m.GetLines()
			if m.cursorLine < len(lines) {
				m.editBuf = lines[m.cursorLine]
			} else {
				m.editBuf = ""
			}
			m.cursorCol = 0
		}
		m.transitionToEditing()
		return m.debounceUpdate()
	}

	return m, nil
}

func (m Model) handleSpaceKey() (tea.Model, tea.Cmd) {
	m.loadCurrentLineIntoEditBuffer()
	m.editBuf = m.editBuf[:m.cursorCol] + " " + m.editBuf[m.cursorCol:]
	m.cursorCol++
	m.transitionToEditing()
	return m.debounceUpdate()
}

// Control keys
func (m Model) handleCtrlP() (tea.Model, tea.Cmd) {
	m.cyclePreviewMode()
	return m, nil
}

func (m Model) handleCtrlD() (tea.Model, tea.Cmd) {
	m.loadCurrentLineIntoEditBuffer()
	m.moveCursor(m.height/2, 0)
	return m, nil
}

func (m Model) handleCtrlU() (tea.Model, tea.Cmd) {
	m.loadCurrentLineIntoEditBuffer()
	m.moveCursor(-m.height/2, 0)
	return m, nil
}

// insertRune inserts a single character at the cursor position.
func (m *Model) insertRune(r rune) {
	m.loadCurrentLineIntoEditBuffer()
	m.editBuf = m.editBuf[:m.cursorCol] + string(r) + m.editBuf[m.cursorCol:]
	m.cursorCol++
}

func (m Model) debounceUpdate() (tea.Model, tea.Cmd) {
	snapshot := m.editBuf
	return m, tea.Tick(evalDebounceDelay, func(t time.Time) tea.Msg {
		return evalDebounceMsg{editBufSnapshot: snapshot}
	})
}

// loadCurrentLineIntoEditBuffer ensures editBuf is loaded with current line content.
// This makes the user ALWAYS able to edit - no mode switching needed.
func (m *Model) loadCurrentLineIntoEditBuffer() {
	if m.editBuf == "" {
		lines := m.GetLines()
		if m.cursorLine < len(lines) {
			m.editBuf = lines[m.cursorLine]
		}
	}
}

// handleGlobalsKey processes keys when globals panel is focused.
func (m Model) handleGlobalsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = StateDefault
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
		m.mode = StateDefault
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

// handleExportFormatKey processes keys in export format selection mode.
func (m Model) handleExportFormatKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = StateDefault
		m.exportFormat = ""
		m.statusMsg = "Export cancelled"
	case tea.KeyEnter:
		// Move to path input
		if m.exportFormat != "" {
			m.mode = StateExportPath
			m.exportPath = ""
			m.statusMsg = "Enter filename (without extension):"
		}
	case tea.KeyRunes:
		if len(msg.Runes) > 0 {
			key := msg.Runes[0]
			// Select format by number key (1-5)
			if key >= '1' && key <= '5' {
				idx := int(key - '1')
				if idx < len(m.exportFormatOpts) {
					m.exportFormat = m.exportFormatOpts[idx]
					// Auto-advance to path input
					m.mode = StateExportPath
					m.exportPath = ""
					m.statusMsg = fmt.Sprintf("Exporting as %s. Enter filename:", m.exportFormat)
				}
			}
		}
	}
	return m, nil
}

// handleExportPathKey processes keys in export path input mode.
func (m Model) handleExportPathKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = StateDefault
		m.exportFormat = ""
		m.exportPath = ""
		m.statusMsg = "Export cancelled"
	case tea.KeyEnter:
		if m.exportPath != "" {
			m.exportFile(m.exportPath, m.exportFormat)
			m.mode = StateDefault
			m.exportFormat = ""
			m.exportPath = ""
		}
	case tea.KeyBackspace:
		if len(m.exportPath) > 0 {
			m.exportPath = m.exportPath[:len(m.exportPath)-1]
		}
	case tea.KeySpace:
		m.exportPath += " "
	case tea.KeyRunes:
		for _, r := range msg.Runes {
			m.exportPath += string(r)
		}
	}
	return m, nil
}

// handleSavePromptKey processes keys in save prompt mode (before quit).
func (m Model) handleSavePromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
		switch msg.Runes[0] {
		case 'y', 'Y':
			// Save and quit - but if no filename, prompt for one first
			if m.filepath == "" {
				m.mode = StateSaveAsPath
				m.saveAsPath = ""
				m.statusMsg = "Save as (filename):"
				return m, nil
			}
			m.saveFile("")
			if !m.statusIsErr {
				m.quitting = true
				return m, tea.Quit
			}
			// If save failed, stay in prompt mode
			return m, nil
		case 'n', 'N':
			// Quit without saving
			m.quitting = true
			return m, tea.Quit
		case 'c', 'C':
			// Cancel quit
			m.mode = StateDefault
			m.statusMsg = "Quit cancelled"
		}
	} else if msg.Type == tea.KeyEsc {
		// Cancel quit
		m.mode = StateDefault
		m.statusMsg = "Quit cancelled"
	}
	return m, nil
}

// handleSaveAsPathKey processes keys in save-as filename input mode.
func (m Model) handleSaveAsPathKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = StateDefault
		m.saveAsPath = ""
		m.quitting = false // Clear quit flag if canceling save
		m.statusMsg = "Save cancelled"
	case tea.KeyEnter:
		if m.saveAsPath != "" {
			m.saveFile(m.saveAsPath)
			if !m.statusIsErr {
				m.mode = StateDefault
				// If we were trying to quit, quit now
				if m.quitting {
					return m, tea.Quit
				}
			}
			m.saveAsPath = ""
		}
	case tea.KeyBackspace:
		if len(m.saveAsPath) > 0 {
			m.saveAsPath = m.saveAsPath[:len(m.saveAsPath)-1]
		}
	case tea.KeySpace:
		m.saveAsPath += " "
	case tea.KeyRunes:
		for _, r := range msg.Runes {
			m.saveAsPath += string(r)
		}
	}
	return m, nil
}

// handleEscape processes escape key.
// ESC is only for canceling modes, not for quitting the application.
// Use Ctrl+Q to quit.
func (m Model) handleEscape() (tea.Model, tea.Cmd) {
	// ESC does nothing in normal mode - it's just for canceling other modes
	// The mode-specific handlers will handle ESC appropriately
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

// saveCurrentLine saves the edit buffer to the current line without changing mode.
// The user is ALWAYS able to edit - no mode switching needed.
func (m *Model) saveCurrentLine(save bool) {
	if save && m.editBuf != "" {
		// Special case: empty document with content in edit buffer
		// Create the document from the buffer content
		if len(m.GetLines()) == 0 {
			newDoc, err := document.NewDocument(m.editBuf)
			if err == nil {
				m.doc = newDoc
				m.eval = implDoc.NewEvaluator()
				_ = m.eval.Evaluate(m.doc)
				m.modified = true
				m.pushUndoState()
				m.autoPinVariables()
			}
		} else if len(m.GetLines()) > 0 {
			// Normal case: update existing line
			m.updateCurrentLine(m.editBuf)
			m.modified = true
			m.pushUndoState()

			// Re-detect block types in case content changed from calc to text or vice versa
			m.redetectBlockTypes()

			// Re-evaluate affected blocks
			m.reEvaluate()
		}
	}
}

// saveCurrentLineAndMoveTo saves the current edit buffer and moves to a new line,
// staying in edit mode. Used for up/down navigation while editing.
func (m *Model) saveCurrentLineAndMoveTo(newLine int) {
	// Save current line content
	m.updateCurrentLine(m.editBuf)
	m.modified = true

	// CRITICAL: Re-evaluate after saving line so preview shows updated results
	m.redetectBlockTypes()
	m.reEvaluate()
	m.pushUndoState()
	m.userIsTyping = false // Navigation commits the typing

	// Remember cursor column to try to preserve it
	savedCol := m.cursorCol

	// Move to new line
	m.cursorLine = newLine

	// Load new line into edit buffer
	lines := m.GetLines()
	if m.cursorLine < len(lines) {
		m.editBuf = lines[m.cursorLine]
	} else {
		m.editBuf = ""
	}

	// Try to preserve column position, clamp to line length
	if savedCol > len(m.editBuf) {
		m.cursorCol = len(m.editBuf)
	} else {
		m.cursorCol = savedCol
	}

	// Stay in edit mode (don't change m.mode)
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

// insertLine inserts a new empty line at the given position.
// This rebuilds the document with the new line inserted at the correct position.
func (m *Model) insertLine(at int) {
	lines := m.GetLines()

	// Clamp position
	if at < 0 {
		at = 0
	}
	if at > len(lines) {
		at = len(lines)
	}

	// Insert empty line at position
	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:at]...)
	newLines = append(newLines, "")
	newLines = append(newLines, lines[at:]...)

	// Rebuild document with new content
	// CRITICAL: strings.Join(lines, "\n") is NOT round-trippable!
	// To preserve N lines, we need N-1 separators PLUS a trailing newline.
	// Example: ["", ""] needs "\n\n" not "\n" to preserve 2 lines.
	content := strings.Join(newLines, "\n") + "\n"
	newDoc, err := document.NewDocument(content)
	if err != nil {
		// DEBUG: log error if document creation fails
		// In production, this shouldn't happen but helps debug
		_ = err // Keep for future debugging
		return
	}

	// Replace document
	m.doc = newDoc
	m.eval = implDoc.NewEvaluator()
	_ = m.eval.Evaluate(m.doc)

	// Set cursor to new line
	m.cursorLine = at
	m.cursorCol = 0
	m.modified = true
	m.pushUndoState()

	// Adjust scroll to keep cursor visible
	visibleHeight := m.height - 6 // Account for status bar etc
	if visibleHeight < 1 {
		visibleHeight = 1
	}
	if m.cursorLine < m.scrollOffset {
		m.scrollOffset = m.cursorLine
	}
	if m.cursorLine >= m.scrollOffset+visibleHeight {
		m.scrollOffset = m.cursorLine - visibleHeight + 1
	}

	// Auto-pin any new variables
	m.autoPinVariables()
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

		// Update changedBlockIDs to include ALL affected blocks (including dependents)
		// This allows the view to show visual feedback for cascading changes
		m.changedBlockIDs = make(map[string]bool)
		for _, id := range orderedBlocks {
			m.changedBlockIDs[id] = true
		}

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
	// Note: changedBlockIDs is NOT cleared here - it persists until the next edit
	// so the view can show which blocks were affected by the last change
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
	case "export", "x":
		// Enter export mode
		m.enterExportMode()
	case "open", "o":
		if len(parts) > 1 {
			m.openFile(parts[1])
		} else {
			m.statusMsg = "Usage: /open <filename>"
			m.statusIsErr = true
		}
	case "quit", "q":
		// Check for unsaved changes
		if m.hasUnsavedChanges() {
			m.mode = StateSavePrompt
			m.statusMsg = "Unsaved changes! Save before quit? (y/n/c)"
		} else {
			m.quitting = true
		}
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

	// Update state - record the saved content for change detection
	m.filepath = absPath
	m.savedContent = content
	m.modified = false
	m.statusMsg = fmt.Sprintf("Saved: %s", filepath.Base(absPath))
	m.statusIsErr = false
}

// exportFile exports the document to a file in the specified format.
func (m *Model) exportFile(filename, formatName string) {
	if filename == "" {
		m.statusMsg = "No filename specified"
		m.statusIsErr = true
		return
	}

	// Get absolute path and add appropriate extension
	absPath, err := filepath.Abs(filename)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Invalid path: %v", err)
		m.statusIsErr = true
		return
	}

	// Add extension based on format if not present
	ext := filepath.Ext(absPath)
	if ext == "" {
		switch formatName {
		case "cm":
			absPath += ".cm"
		case "json":
			absPath += ".json"
		case "html":
			absPath += ".html"
		case "md":
			absPath += ".md"
		case "text":
			absPath += ".txt"
		default:
			absPath += ".txt"
		}
	}

	// Create file
	file, err := os.Create(absPath)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Export failed: %v", err)
		m.statusIsErr = true
		return
	}
	defer file.Close()

	// Get formatter from registry
	formatter := format.GetFormatter(formatName, absPath)

	// Format and write to file
	opts := format.Options{
		Verbose:       false,
		IncludeErrors: true,
	}

	err = formatter.Format(file, m.doc, opts)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Export failed: %v", err)
		m.statusIsErr = true
		return
	}

	m.statusMsg = fmt.Sprintf("Exported to: %s (%s)", filepath.Base(absPath), formatName)
}

// enterExportMode enters export format selection mode.
func (m *Model) enterExportMode() {
	m.mode = StateExportFormat
	m.exportFormat = ""
	m.statusMsg = "Select export format: 1)text 2)cm 3)json 4)html 5)md"
}

// hasUnsavedChanges returns true if there are unsaved changes.
func (m *Model) hasUnsavedChanges() bool {
	// Compare current document content with last saved content
	currentContent := m.getDocumentContent()
	return currentContent != m.savedContent
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

	// Record file content as saved state
	m.savedContent = string(content)

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
	// Note: mode is an internal implementation detail and not shown to users

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
	case StateDefault:
		// User is always able to edit - show all available commands
		hints = fmt.Sprintf("Ctrl+S=save Ctrl+E=export Ctrl+Q=quit Arrows=navigate %s", previewHint)
	case StateExportFormat:
		hints = "1-5=select Esc=cancel"
	case StateExportPath:
		hints = "Enter=export Esc=cancel"
	case StateSavePrompt:
		hints = "y=save&quit n=quit c=cancel"
	case StateSaveAsPath:
		hints = "Enter=save Esc=cancel"
	}

	return components.StatusBarState{
		Filename:    m.filepath,
		Line:        m.cursorLine + 1,
		TotalLines:  m.TotalLines(),
		CalcCount:   m.CalcBlockCount(),
		Modified:    m.modified,
		Mode:        "", // Mode is internal - not shown to users
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
		Focused:    m.mode == StateGlobals,
	}
}

// alignedCacheKey captures the inputs that affect AlignedModel computation.
// If any of these change, the cache must be invalidated.
type alignedCacheKey struct {
	contentHash uint64      // Hash of document content
	cursorLine  int         // Cursor position affects highlighting
	previewMode PreviewMode // Affects rendering
	totalLines  int         // Quick check for document changes
	editBuf     string      // EditBuf changes should invalidate cache
}

// computeCacheKey computes a cache key from current model state.
func (m *Model) computeCacheKey() alignedCacheKey {
	// Simple hash of content - just use length and first/last chars for speed
	// A proper implementation would use a real hash, but this catches most changes
	lines := m.GetLines()
	var contentHash uint64
	for i, line := range lines {
		contentHash ^= uint64(len(line)) << (uint(i%8) * 8)
		if len(line) > 0 {
			contentHash ^= uint64(line[0]) << 32
			contentHash ^= uint64(line[len(line)-1]) << 40
		}
	}

	return alignedCacheKey{
		contentHash: contentHash,
		cursorLine:  m.cursorLine,
		previewMode: m.previewMode,
		totalLines:  len(lines),
		editBuf:     m.editBuf, // Include editBuf so cache updates while typing
	}
}

// GetAlignedModel returns the cached aligned model, computing it if necessary.
// This is the single source of truth for visual line alignment.
// The cache is automatically invalidated when inputs change.
func (m *Model) GetAlignedModel(sourceWidth, previewWidth int) *AlignedModel {
	currentKey := m.computeCacheKey()

	// Check if cache is valid: same key and same widths
	if m.alignedCache != nil &&
		m.alignedCacheKey == currentKey &&
		m.alignedCacheWidths[0] == sourceWidth &&
		m.alignedCacheWidths[1] == previewWidth {
		return m.alignedCache
	}

	// Cache miss - recompute
	// Calculate content width for source pane (accounting for line numbers)
	lineNumWidth := 4
	sourceContentWidth := sourceWidth - lineNumWidth - 2
	if sourceContentWidth < 10 {
		sourceContentWidth = 10
	}

	input := AlignedModelInput{
		Lines:              m.GetLines(),
		Results:            m.GetLineResults(),
		SourceContentWidth: sourceContentWidth,
		PreviewWidth:       previewWidth,
		CursorLine:         m.cursorLine,
		PreviewMode:        m.previewMode,
		EditBuf:            m.editBuf,
		EditBufLine:        m.cursorLine, // EditBuf applies to current cursor line
	}

	// Compute with render functions that match view.go behavior
	aligned := ComputeAlignedModel(input, m.renderCalcLine, func(line string, width int) []string {
		mdRenderer, _ := NewMarkdownRenderer(width)
		if mdRenderer != nil {
			return mdRenderer.RenderLine(line)
		}
		return geometry.WrapText(line, width)
	})

	// Update cache
	m.alignedCache = &aligned
	m.alignedCacheKey = currentKey
	m.alignedCacheWidths = [2]int{sourceWidth, previewWidth}

	return m.alignedCache
}

// InvalidateAlignedCache explicitly invalidates the cache.
// This is called on key presses, but the cache will also auto-invalidate
// when computeCacheKey() detects changed inputs.
func (m *Model) InvalidateAlignedCache() {
	m.alignedCache = nil
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
func (m Model) Mode() InputState {
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
func (m *Model) SetMode(mode InputState) {
	m.mode = mode
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

// Debug returns a string representation of the model's alignment state.
// This is used by catwalk tests to verify visual/source line consistency.
func (m Model) Debug() string {
	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	// Get cursor's visual position from the mapping
	cursorVisual := -1
	if v, ok := aligned.sourceToVisual[m.cursorLine]; ok {
		cursorVisual = v
	}

	// Find where cursor is actually highlighted in the visual structure
	cursorHighlightAt := -1
	for i, sl := range aligned.sourceLines {
		if sl.isCursorLine {
			cursorHighlightAt = i
			break
		}
	}

	// Check invariants
	sourcePreviewMatch := len(aligned.sourceLines) == len(aligned.previewLines)
	cursorInBounds := cursorVisual >= 0 && cursorVisual < len(aligned.sourceLines)
	highlightMatchesMapping := cursorHighlightAt == cursorVisual
	mappingComplete := true
	for i := 0; i < m.TotalLines(); i++ {
		if _, ok := aligned.sourceToVisual[i]; !ok {
			mappingComplete = false
			break
		}
	}

	return fmt.Sprintf(
		"mode=%v cursorLine=%d cursorCol=%d cursorVisual=%d cursorHighlight=%d "+
			"scrollOffset=%d totalSource=%d totalVisual=%d editBuf=%q "+
			"sourcePreviewMatch=%v cursorInBounds=%v highlightMatch=%v mappingComplete=%v",
		m.mode, m.cursorLine, m.cursorCol, cursorVisual, cursorHighlightAt,
		m.scrollOffset, m.TotalLines(), len(aligned.sourceLines), m.editBuf,
		sourcePreviewMatch, cursorInBounds, highlightMatchesMapping, mappingComplete,
	)
}

// DebugLines returns a detailed breakdown of the visual line structure.
// This is used for debugging alignment issues.
func (m Model) DebugLines() string {
	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("sourceToVisual: %v\n", aligned.sourceToVisual))
	b.WriteString("Visual lines:\n")
	for i, sl := range aligned.sourceLines {
		cursor := ""
		if sl.isCursorLine {
			cursor = " <CURSOR>"
		}
		b.WriteString(fmt.Sprintf("  [%d] srcIdx=%d lineNum=%d wrap=%v pad=%v content=%q%s\n",
			i, sl.sourceLineIdx, sl.lineNum, sl.isWrapped, sl.isPadding,
			truncateStr(sl.content, 30), cursor))
	}
	return b.String()
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
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
				m.InvalidateAlignedCache()

				// Adjust cursor if needed
				total := m.TotalLines()
				if m.cursorLine >= total && total > 0 {
					m.cursorLine = total - 1
				}

				// Adjust scroll offset if it's now past document end
				if m.scrollOffset > 0 && m.scrollOffset >= total {
					m.scrollOffset = total - 1
					if m.scrollOffset < 0 {
						m.scrollOffset = 0
					}
				}

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
