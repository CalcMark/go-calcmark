package editor

// model.go — Model definition, constructors, types, accessors, Init/Update.
//
// Method implementations are split across focused files:
//   key_dispatch.go      — handleKey, handleDefaultKey
//   navigation.go        — Arrow keys, scroll, search, goto
//   editing.go           — Text editing (rune input, Enter, Backspace, Delete, line ops)
//   undo_operations.go   — handleUndo/Redo, apply ops
//   file_operations.go   — Save, Open, Export, hasUnsavedChanges, cyclePreviewMode
//   file_picker_handler.go — handleFilePickerKey
//   globals_handler.go   — handleGlobalsKey, handleSavePromptKey
//   view_state.go        — Status bar, panels, aligned model cache, GetAutocompleteState
//   selection.go         — Selection operations (SelectAll, ClearSelection, etc.)
//   clipboard.go         — Cut/Copy/Paste
//   command_menu.go      — Command menu handling
//   help_overlay.go      — Help overlay handling
//   export_overlay.go    — Export overlay handling
//   autocomplete.go      — Suggestion sources

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"charm.land/bubbles/v2/filepicker"
	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/config"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/shared"
	"github.com/CalcMark/go-calcmark/format/display"
	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// ========================================
// Utility functions (UTF-8 safe string operations)
// ========================================

// runeSlice splits a string at a rune position, returning (before, after).
// This is UTF-8 safe - runePos is a character index, not a byte index.
// If runePos is out of bounds, it's clamped to valid range.
func runeSlice(s string, runePos int) (before, after string) {
	runes := []rune(s)
	if runePos < 0 {
		runePos = 0
	}
	if runePos > len(runes) {
		runePos = len(runes)
	}
	return string(runes[:runePos]), string(runes[runePos:])
}

// runeLen returns the number of runes (characters) in a string.
// This is UTF-8 safe and returns character count, not byte count.
func runeLen(s string) int {
	return utf8.RuneCountInString(s)
}

// runeInsert inserts a string at a rune position.
// This is UTF-8 safe - runePos is a character index, not a byte index.
func runeInsert(s string, runePos int, insert string) string {
	before, after := runeSlice(s, runePos)
	return before + insert + after
}

// runeDelete removes count runes starting at runePos.
// This is UTF-8 safe - runePos is a character index, not a byte index.
func runeDelete(s string, runePos, count int) (result string, deleted string) {
	runes := []rune(s)
	if runePos < 0 {
		runePos = 0
	}
	if runePos >= len(runes) {
		return s, ""
	}
	endPos := min(runePos+count, len(runes))
	deleted = string(runes[runePos:endPos])
	result = string(runes[:runePos]) + string(runes[endPos:])
	return result, deleted
}

// expandTilde expands ~ to the user's home directory in file paths.
// The shell doesn't expand tilde for us when we get user input.
func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	} else if path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home
		}
	}
	return path
}

// ========================================
// Constants
// ========================================

// evalDebounceDelay is the time to wait after typing stops before re-evaluating.
// This prevents excessive evaluation during rapid typing while keeping results responsive.
// Can be tuned lower if interpreter performance allows (100ms is conservative).
const evalDebounceDelay = 100 * time.Millisecond

// scrollMargin is the number of lines to keep between cursor and viewport edge.
// This provides visual context around the cursor position.
const scrollMargin = 3

// ========================================
// Types
// ========================================

// evalDebounceMsg is sent after the debounce delay to trigger evaluation.
type evalDebounceMsg struct {
	editBufSnapshot string // Snapshot of editBuf when timer was started
}

// alignedCacheKey stores the inputs that determine aligned pane output.
// Used to avoid recomputing alignment when nothing has changed.
type alignedCacheKey struct {
	contentHash uint64      // Hash of document content
	cursorLine  int         // Cursor position affects highlighting
	previewMode PreviewMode // Affects rendering
	totalLines  int         // Quick check for document changes
	editBuf     string      // EditBuf changes should invalidate cache
}

// shareResultMsg carries the result of a Share To Gist operation.
type shareResultMsg struct {
	url    string
	copied bool // True if URL was successfully written to clipboard
	err    error
}

// openFromResultMsg carries the result of an Open From Gist operation.
type openFromResultMsg struct {
	content  string
	filename string
	err      error
}

// retryShareMsg triggers a retry of the share operation after interactive auth.
type retryShareMsg struct{}

// retryOpenFromMsg triggers a retry of the open operation after interactive auth.
type retryOpenFromMsg struct {
	identifier string
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
	StateAutocomplete                   // Autocomplete dropdown active
	StateGlobals                        // Globals panel active, preview shows global values
	StateHelp                           // Interactive help overlay with actionable/advisory items
	StateCommandMenu                    // Command menu popup active
	StateFilePicker                     // File picker for save/open operations
	StateExport                         // Export overlay (format + filename in one modal)
	StateShareTo                        // Share To Gist overlay (visibility + description)
	StateOpenFrom                       // Open From Gist overlay (URL/ID input)
	StateSavePrompt                     // Save confirmation dialog active, normal UI paused
)

// String returns the string name of the InputState for debugging.
func (s InputState) String() string {
	switch s {
	case StateDefault:
		return "StateDefault"
	case StateAutocomplete:
		return "StateAutocomplete"
	case StateGlobals:
		return "StateGlobals"
	case StateHelp:
		return "StateHelp"
	case StateCommandMenu:
		return "StateCommandMenu"
	case StateFilePicker:
		return "StateFilePicker"
	case StateExport:
		return "StateExport"
	case StateShareTo:
		return "StateShareTo"
	case StateOpenFrom:
		return "StateOpenFrom"
	case StateSavePrompt:
		return "StateSavePrompt"
	default:
		return fmt.Sprintf("InputState(%d)", s)
	}
}

// PendingAction tracks what action triggered the save prompt.
// The save prompt handler uses this to decide what to do after
// the user responds (quit vs open file vs new file).
type PendingAction int

const (
	PendingNone           PendingAction = iota // No pending action (default — normal save)
	PendingQuit                                // Save prompt was triggered by Ctrl+Q
	PendingOpen                                // Save prompt was triggered by Ctrl+O
	PendingNew                                 // Save prompt was triggered by Ctrl+N
	PendingOpenFromRemote                      // Save prompt was triggered by Open From Gist
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
	PreviewFull:    {SourcePercent: 60, PreviewPercent: 40},
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

// ========================================
// Model struct
// ========================================

// Model represents the document editor state.
type Model struct {
	// Core document (from spec/document)
	doc            *document.Document
	eval           *implDoc.Evaluator
	filepath       string
	modified       bool  // True if document has unsaved changes
	frontmatterErr error // Non-nil when frontmatter YAML is malformed

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
	editBufLoaded   bool            // True when editBuf has been loaded for current line (distinguishes "" from "not loaded")
	lineWrap        bool            // Whether to wrap long lines
	changedBlockIDs map[string]bool // Track changed blocks for highlighting

	// Undo/redo - UndoManager handles operation-based undo/redo with timer grouping
	undoManager *UndoManager
	undoGroupID int // Mirrors UndoManager.groupID for stale timer detection

	// Export state
	exportState      ExportOverlayState // State for the export modal overlay
	exportFormatOpts []string           // Available export formats

	// Share To overlay state
	shareVisibility  int // 0 = public, 1 = secret
	shareDescription string
	shareField       int // 0 = visibility select, 1 = description input

	// Open From overlay state
	openFromInput string

	// Help overlay state
	helpState HelpOverlayState // Navigation state for interactive help

	// File picker purpose
	filePickerPurpose FilePickerPurpose // Why the file picker was opened (save or open)

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

	// Selection state
	selectionAnchorLine int // Line of selection anchor, -1 if no selection
	selectionAnchorCol  int // Column of selection anchor

	// Status message
	statusMsg   string
	statusIsErr bool

	// Display formatting (locale-aware)
	formatter display.Formatter

	// Styles
	styles config.Styles

	// Keybindings
	keys shared.KeyMap

	// Cached alignment model - computed once and invalidated on changes
	alignedCache       *AlignedModel
	alignedCacheKey    alignedCacheKey // Key for cache validation
	alignedCacheWidths [2]int          // [sourceWidth, previewWidth] used for cache

	// Autocomplete state
	autocompleteState components.AutosuggestState
	suggestionSource  components.SuggestionSource

	// Command menu state
	commandMenuState CommandMenuState

	// File picker state
	filePicker      filepicker.Model
	filePickerFocus FilePickerFocus // Which part of save dialog has focus
	newFileName     string          // Filename being typed

	// Save prompt context — tracks what action triggered the save prompt
	// so the handler knows what to do after the user responds.
	pendingSaveAction PendingAction
}

// ========================================
// Constructors
// ========================================

// New creates a new editor model with an optional document.
// This is the ONLY place where editor state is initialized.
// After this, the editor is ALWAYS in StateReady with all invariants satisfied.
func New(doc *document.Document) Model {
	// User is ALWAYS able to edit - mode only represents temporary UI overlays
	m := Model{
		doc:                 doc,
		eval:                nil,
		mode:                StateDefault,
		userIsTyping:        false,
		pinnedVars:          make(map[string]bool),
		changedVars:         make(map[string]bool),
		changedBlockIDs:     make(map[string]bool),
		undoManager:         NewUndoManager(1000),
		exportFormatOpts:    []string{"text", "cm", "json", "html", "md"},
		width:               80,
		height:              24,
		previewMode:         PreviewFull,
		lineWrap:            true,
		styles:              config.GetStyles(),
		keys:                shared.DefaultKeyMap(),
		selectionAnchorLine: -1, // No selection initially
		selectionAnchorCol:  -1,
	}

	// Initialize autocomplete suggestion sources
	funcSource := NewFunctionSuggestionSource()
	unitSource := NewUnitSuggestionSource()
	// Variable source captures 'm' by closure to access current environment
	varSource := NewVariableSuggestionSource(func() map[string]string {
		if m.eval == nil {
			return nil
		}
		env := m.eval.GetEnvironment()
		vars := env.GetAllVariables()
		result := make(map[string]string)
		for name, val := range vars {
			result[name] = fmt.Sprintf("%v", val)
		}
		return result
	})
	m.suggestionSource = NewCombinedSuggestionSource(funcSource, unitSource, varSource)

	// CRITICAL: Transition to StateReady - establishes all invariants
	// This is the ONLY state transition during initialization
	m.transitionToReady()

	// Auto-pin all variables
	m.autoPinVariables()

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

// SetFormatter sets the locale-aware display formatter.
// If not called, the zero-value Formatter falls back to en-US defaults.
func (m *Model) SetFormatter(f display.Formatter) {
	m.formatter = f
}

// displayFormat formats a value using the model's locale-aware formatter.
// Falls back to DefaultFormatter if no formatter was set.
func (m Model) displayFormat(t types.Type) string {
	if m.formatter.Config().DecimalSep == "" {
		return display.Format(t)
	}
	return m.formatter.Format(t)
}

// ========================================
// Document accessors
// ========================================

// builtinConstants contains mathematical constants that should not be pinned.
// These are always available in the environment and showing them clutters the display.
var builtinConstants = map[string]bool{
	"PI": true,
	"E":  true,
}

// autoPinVariables pins all user-defined variables in the document.
// Built-in constants (PI, E) are excluded to avoid cluttering the pinned panel.
func (m *Model) autoPinVariables() {
	for _, node := range m.doc.GetBlocks() {
		if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
			for _, varName := range calcBlock.Variables() {
				if !builtinConstants[varName] {
					m.pinnedVars[varName] = true
				}
			}
		}
	}
}

// getDocumentContent returns the document as a string, including frontmatter.
// CRITICAL: Returns content with trailing newline to preserve line count.
// See unicode.go fix - trailing newlines no longer create extra lines,
// so we MUST include them when reconstructing to preserve N lines.
func (m *Model) getDocumentContent() string {
	lines := m.GetLines()
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

// splitFrontmatterLines splits a serialized frontmatter string into its
// constituent lines, trimming the trailing newline that Serialize() appends.
func splitFrontmatterLines(serialized string) []string {
	return strings.Split(strings.TrimRight(serialized, "\n"), "\n")
}

// GetLines returns all lines in the document, including frontmatter.
func (m *Model) GetLines() []string {
	var lines []string

	// Prepend frontmatter lines if present
	if fm := m.doc.GetFrontmatter(); fm != nil {
		serialized := fm.Serialize()
		if serialized != "" {
			lines = append(lines, splitFrontmatterLines(serialized)...)
		}
	}

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

// frontmatterLineCount returns the number of lines occupied by frontmatter.
// Returns 0 if no frontmatter exists.
func (m *Model) frontmatterLineCount() int {
	fm := m.doc.GetFrontmatter()
	if fm == nil {
		return 0
	}
	serialized := fm.Serialize()
	if serialized == "" {
		return 0
	}
	return len(splitFrontmatterLines(serialized))
}

// TotalLines returns the total number of lines without materializing a []string.
func (m *Model) TotalLines() int {
	count := m.frontmatterLineCount()
	for _, node := range m.doc.GetBlocks() {
		switch b := node.Block.(type) {
		case *document.CalcBlock:
			count += len(b.Source())
		case *document.TextBlock:
			count += len(b.Source())
		}
	}
	return count
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

// ========================================
// Bubble Tea interface (Init + Update)
// ========================================

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// When file picker is active, pass ALL messages to it (not just KeyPressMsg)
	// The filepicker needs to receive its internal messages (directory read results)
	if m.mode == StateFilePicker {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			return m.handleKey(msg)
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			m.InvalidateAlignedCache()
		default:
			// Pass other messages to filepicker (e.g., directory read results)
			var cmd tea.Cmd
			m.filePicker, cmd = m.filePicker.Update(msg)
			return m, cmd
		}
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.PasteMsg:
		// Bracketed paste: terminal intercepted Cmd+V (or middle-click) and
		// sent the clipboard content as a paste event. Handle identically to
		// Ctrl+V / Cmd+V key shortcut.
		return m.handleBracketedPaste(msg.Content)

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

	case undoGroupMsg:
		// Timer fired for undo grouping - commit if batchID matches current groupID
		// (Stale timers have mismatched batchIDs and are ignored)
		if msg.batchID == m.undoGroupID {
			m.undoManager.CommitCurrentBatch()
		}

	case shareResultMsg:
		if msg.err != nil {
			m.statusMsg = msg.err.Error()
			m.statusIsErr = true
		} else if msg.copied {
			m.statusMsg = "Shared: " + msg.url + " (copied)"
		} else {
			m.statusMsg = "Shared: " + msg.url
		}
		m.exitOverlay()

	case openFromResultMsg:
		if msg.err != nil {
			m.statusMsg = msg.err.Error()
			m.statusIsErr = true
		} else {
			m.loadDocumentFromString(msg.content, msg.filename)
		}
		m.exitOverlay()

	case retryShareMsg:
		return m.executeShareToGist()

	case retryOpenFromMsg:
		return m.executeOpenFromGist(msg.identifier)
	}

	return m, nil
}

// ========================================
// Exported accessors
// ========================================

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

// Width returns the current width.
func (m Model) Width() int {
	return m.width
}

// Height returns the current height.
func (m Model) Height() int {
	return m.height
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

// ========================================
// Debug accessors (for catwalk tests)
// ========================================

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
			"selectionAnchorLine=%d selectionAnchorCol=%d "+
			"sourcePreviewMatch=%v cursorInBounds=%v highlightMatch=%v mappingComplete=%v",
		m.mode, m.cursorLine, m.cursorCol, cursorVisual, cursorHighlightAt,
		m.scrollOffset, m.TotalLines(), len(aligned.sourceLines), m.editBuf,
		m.selectionAnchorLine, m.selectionAnchorCol,
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

// ========================================
// Autocomplete functions
// ========================================

// handleAutocompleteKey processes keys when autocomplete popup is visible.
// IMPORTANT: Typing continues to work normally - we just update suggestions.
func (m Model) handleAutocompleteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		if m.autocompleteState.Selected > 0 {
			m.autocompleteState.Selected--
		}
		return m, nil
	case "down":
		if m.autocompleteState.Selected < len(m.autocompleteState.Suggestions)-1 {
			m.autocompleteState.Selected++
		}
		return m, nil
	case "esc":
		// Dismiss autocomplete without inserting
		m.exitAutocomplete()
		return m, nil
	case "tab":
		// Accept current selection
		return m.acceptAutocomplete()
	case "backspace":
		// Allow backspace to edit the prefix
		// Capture state BEFORE the edit for undo
		beforeLine := m.cursorLine
		beforeCol := m.cursorCol
		beforeScroll := m.scrollOffset

		m.transitionToEditing()
		if m.cursorCol > 0 && runeLen(m.editBuf) > 0 {
			// Delete character before cursor (UTF-8 safe)
			var deletedChar string
			m.editBuf, deletedChar = runeDelete(m.editBuf, m.cursorCol-1, 1)
			m.cursorCol--

			// Record the delete operation for undo
			op := EditOperation{
				Type:         OpDelete,
				Line:         beforeLine,
				Col:          m.cursorCol, // Position where deletion occurred (rune position)
				OldText:      deletedChar,
				NewText:      "",
				CursorLine:   beforeLine,
				CursorCol:    beforeCol,
				ScrollOffset: beforeScroll,
			}
			undoCmd := m.recordEdit(op)

			// Update suggestions with new prefix (use pointer to modify)
			m.updateAutocompleteState()
			debounceModel, debounceCmd := m.debounceUpdate()
			return debounceModel, tea.Batch(debounceCmd, undoCmd)
		}
		// If no character to delete, just update autocomplete state
		m.updateAutocompleteState()
		return m.debounceUpdate()
	case "space":
		// Space typically ends a word - dismiss and insert space
		m.exitAutocomplete()
		return m.handleRuneInput([]rune{' '})
	case "enter":
		// Enter accepts if there's a selection, otherwise just inserts newline
		if len(m.autocompleteState.Suggestions) > 0 {
			return m.acceptAutocomplete()
		}
		m.exitAutocomplete()
		return m.handleEnterKey()
	default:
		if msg.Text != "" {
			// Continue typing - insert characters and update suggestions
			// Capture state BEFORE the edit for undo
			beforeLine := m.cursorLine
			beforeCol := m.cursorCol
			beforeScroll := m.scrollOffset

			m.transitionToEditing()
			for _, r := range msg.Text {
				m.insertRune(r)
			}

			// Record the insert operation for undo
			insertText := msg.Text
			op := EditOperation{
				Type:         OpInsert,
				Line:         beforeLine,
				Col:          beforeCol,
				OldText:      "",
				NewText:      insertText,
				CursorLine:   beforeLine,
				CursorCol:    beforeCol,
				ScrollOffset: beforeScroll,
			}
			undoCmd := m.recordEdit(op)

			// Update suggestions with new prefix (use pointer to modify)
			m.updateAutocompleteState()
			debounceModel, debounceCmd := m.debounceUpdate()
			return debounceModel, tea.Batch(debounceCmd, undoCmd)
		}
		// Navigation and other keys dismiss autocomplete
		m.exitAutocomplete()
		return m.handleDefaultKey(msg)
	}
}

// triggerAutocomplete initiates autocomplete mode (called explicitly by TAB).
func (m Model) triggerAutocomplete() (tea.Model, tea.Cmd) {
	m.updateAutocompleteState()
	return m, nil
}

// updateAutocompleteState checks for suggestions at current prefix and updates popup state.
// This is called after every character typed to show/hide the popup automatically.
// Uses pointer receiver because it modifies mode and autocompleteState.
func (m *Model) updateAutocompleteState() {
	// Extract word prefix at cursor position
	prefix := m.getCurrentWordPrefix()

	// No prefix - dismiss any visible popup
	if prefix == "" {
		if m.mode == StateAutocomplete {
			m.mode = StateDefault
			m.autocompleteState = components.AutosuggestState{}
		}
		return
	}

	// Check if we have a suggestion source
	if m.suggestionSource == nil {
		return
	}

	suggestions := m.suggestionSource.GetSuggestions(prefix)

	// No suggestions - dismiss popup if visible
	if len(suggestions) == 0 {
		if m.mode == StateAutocomplete {
			m.mode = StateDefault
			m.autocompleteState = components.AutosuggestState{}
		}
		return
	}

	// Calculate popup position based on cursor
	popupWidth, popupHeight := m.calculatePopupDimensions(suggestions)

	// We have suggestions - show/update the popup
	m.mode = StateAutocomplete
	m.autocompleteState = components.AutosuggestState{
		Suggestions: suggestions,
		Selected:    0,
		Visible:     true,
		Prefix:      prefix,
		PopupWidth:  popupWidth,
		PopupHeight: popupHeight,
	}
}

// calculatePopupDimensions determines the popup size based on suggestions.
func (m *Model) calculatePopupDimensions(suggestions []components.Suggestion) (width, height int) {
	// Calculate width based on longest suggestion name + syntax
	width = 30 // minimum width for readability
	for _, s := range suggestions {
		// Name + syntax is more important than description for width
		w := len(s.Name)
		if s.Syntax != "" {
			w += 1 + len(s.Syntax)
		}
		if w+6 > width { // +6 for padding, selection indicator, borders
			width = w + 6
		}
	}

	// Allow up to 70% of screen width for function signatures
	maxWidth := max(m.width*7/10, 40) // minimum usable width
	if width > maxWidth {
		width = maxWidth
	}

	// Height is number of items, capped at 8
	height = min(len(suggestions), 8)

	return width, height
}

// getCurrentWordPrefix extracts the word being typed at cursor (UTF-8 safe).
func (m *Model) getCurrentWordPrefix() string {
	m.loadCurrentLineIntoEditBuffer()
	if m.cursorCol == 0 {
		return ""
	}

	// Convert to runes for UTF-8 safe iteration
	runes := []rune(m.editBuf)
	if m.cursorCol > len(runes) {
		return ""
	}

	// Walk backwards to find word start
	start := m.cursorCol
	for start > 0 {
		ch := runes[start-1]
		if !isWordRune(ch) {
			break
		}
		start--
	}

	if start >= m.cursorCol {
		return ""
	}
	return string(runes[start:m.cursorCol])
}

// isWordRune returns true if the rune is a valid word character for autocomplete.
// This is UTF-8 safe and handles Unicode letters/digits.
func isWordRune(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
}

// acceptAutocomplete inserts the selected suggestion at the cursor.
func (m Model) acceptAutocomplete() (tea.Model, tea.Cmd) {
	if m.autocompleteState.Selected < 0 ||
		m.autocompleteState.Selected >= len(m.autocompleteState.Suggestions) {
		m.exitAutocomplete()
		return m, nil
	}

	selected := m.autocompleteState.Suggestions[m.autocompleteState.Selected]
	insertText := selected.InsertText
	if insertText == "" {
		insertText = selected.Name
	}

	// For functions (identified by having a Syntax like "func(...)"), add opening paren
	// This positions the cursor inside the function call so parameter help is shown.
	isFunction := strings.Contains(selected.Syntax, "(")
	if isFunction {
		insertText += "("
	}

	// Replace prefix with selected suggestion (UTF-8 safe)
	prefix := m.autocompleteState.Prefix
	prefixStart := max(m.cursorCol-runeLen(prefix), 0)

	// Delete the prefix, then insert the completion text
	beforePrefix, _ := runeSlice(m.editBuf, prefixStart)
	_, afterCursor := runeSlice(m.editBuf, m.cursorCol)
	m.editBuf = beforePrefix + insertText + afterCursor
	m.cursorCol = prefixStart + runeLen(insertText)

	m.exitAutocomplete()
	m.transitionToEditing()
	return m.debounceUpdate()
}
