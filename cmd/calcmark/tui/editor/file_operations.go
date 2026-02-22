package editor

// file_operations.go — Save, open, export, and preview mode operations.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/format"
	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// saveFile saves the document to a file.
func (m *Model) saveFile(filename string) {
	// Use provided filename or current filepath
	if filename == "" {
		filename = m.filepath
	}
	if filename == "" {
		m.statusMsg = "No filename. Use Ctrl+S or Save from command menu"
		m.statusIsErr = true
		return
	}

	// Expand tilde to home directory
	filename = expandTilde(filename)

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
	previousPath := m.filepath
	m.filepath = absPath
	m.savedContent = content
	m.modified = false

	// Save As: filepath changed — make it clear the active file has changed
	if previousPath != "" && previousPath != absPath {
		m.statusMsg = fmt.Sprintf("Now editing: %s", filepath.Base(absPath))
	} else {
		m.statusMsg = fmt.Sprintf("Saved: %s", filepath.Base(absPath))
	}
	m.statusIsErr = false
}

// exportFile exports the document to a file in the specified format.
func (m *Model) exportFile(filename, formatName string) {
	if filename == "" {
		m.statusMsg = "No filename specified"
		m.statusIsErr = true
		return
	}

	// Expand tilde to home directory
	filename = expandTilde(filename)

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
	m.mode = StateExport
	m.exportState = ExportOverlayState{
		FormatIdx: 0,
	}
	m.statusMsg = ""
}

// hasUnsavedChanges returns true if there are unsaved changes.
// Checks both the flushed document content and the live edit buffer,
// since the user may have typed text that hasn't been committed to
// the document yet.
func (m *Model) hasUnsavedChanges() bool {
	// The edit buffer holds uncommitted keystrokes that aren't reflected
	// in Source() until the line is saved. If the user is actively typing,
	// compare the buffer against the current line to catch pending edits.
	if m.userIsTyping {
		lines := m.GetLines()
		if m.cursorLine < len(lines) && m.editBuf != lines[m.cursorLine] {
			return true
		}
	}
	if !m.modified {
		return false
	}
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

	// Reinitialize all mutable editor state for the new document.
	// Every field must be reset to prevent stale data from leaking
	// into the newly opened document.
	m.doc = doc
	m.eval = eval
	m.filepath = absPath
	m.modified = false
	m.savedContent = string(content)

	// Cursor and scroll
	m.cursorLine = 0
	m.cursorCol = 0
	m.scrollOffset = 0

	// Editing state
	m.editBuf = ""
	m.userIsTyping = false
	m.frontmatterErr = nil
	m.changedBlockIDs = make(map[string]bool)
	m.selectionAnchorLine = -1
	m.selectionAnchorCol = -1
	m.pendingKey = 0
	m.yankBuffer = ""

	// Search state
	m.searchTerm = ""
	m.searchMatches = nil
	m.searchIdx = 0

	// Overlay / prompt state
	m.autocompleteState = components.AutosuggestState{}
	m.pendingSaveAction = PendingNone

	// Undo history — fresh start on file open
	m.undoManager.Clear()

	// Auto-pin variables
	m.pinnedVars = make(map[string]bool)
	m.changedVars = make(map[string]bool)
	m.autoPinVariables()
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
