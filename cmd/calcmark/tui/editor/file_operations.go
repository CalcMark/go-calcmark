package editor

// file_operations.go — Save, open, export, and preview mode operations.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/filecheck"
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
		Verbose:          false,
		IncludeErrors:    true,
		DisplayFormatter: m.formatter,
	}

	err = formatter.Format(file, m.doc, opts)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Export failed: %v", err)
		m.statusIsErr = true
		return
	}

	m.statusMsg = fmt.Sprintf("Exported to: %s (%s)", filepath.Base(absPath), formatName)
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

	// Security: Reject binary/non-text content before parsing
	if err := filecheck.ValidateContent(content); err != nil {
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

	// Reset all mutable editor state for the new document.
	m.resetForNewDocument(doc, eval, absPath, string(content))
}

// newFile resets the editor to a blank, untitled document.
func (m *Model) newFile() {
	doc, err := document.NewDocument("\n")
	if err != nil {
		m.statusMsg = fmt.Sprintf("New file failed: %v", err)
		m.statusIsErr = true
		return
	}

	eval := implDoc.NewEvaluator()
	_ = eval.Evaluate(doc) // empty doc — no errors expected

	m.resetForNewDocument(doc, eval, "", "\n")
	m.statusMsg = "New document"
}

// loadDocumentFromString loads a document from raw content (e.g., fetched from a remote store).
// Similar to openFile but skips the file read — content is already in memory.
// The document is treated as a new untitled file (no filepath set).
func (m *Model) loadDocumentFromString(content, suggestedFilename string) {
	// Security: Reject binary/non-text content before parsing
	if err := filecheck.ValidateContent([]byte(content)); err != nil {
		m.statusMsg = fmt.Sprintf("Open failed: %v", err)
		m.statusIsErr = true
		return
	}

	// Parse document
	doc, err := document.NewDocument(content)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Parse error: %v", err)
		m.statusIsErr = true
		return
	}

	// Evaluate
	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		m.statusMsg = fmt.Sprintf("Opened with errors: %v", err)
		m.statusIsErr = true
	} else {
		m.statusMsg = fmt.Sprintf("Opened from Gist: %s", suggestedFilename)
	}

	// Reset all mutable editor state for the new document.
	// Empty filepath signals this is an untitled document (not yet saved locally).
	m.resetForNewDocument(doc, eval, "", content)
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
