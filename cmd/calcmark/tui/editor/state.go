package editor

import (
	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// EditorState represents the core editing state of the editor.
// State transitions are EXPLICIT - you call a transition function.
// Each state has clear invariants that must hold.
type EditorState int

const (
	// StateReady: Editor has valid document and user can type.
	// Invariants:
	// - doc != nil && len(doc.GetBlocks()) > 0
	// - eval != nil
	// - cursorLine >= 0 && cursorLine < TotalLines()
	// - userIsTyping reflects actual typing state
	StateReady EditorState = iota

	// StateEditing: User is actively typing (userIsTyping = true)
	// Same invariants as StateReady, plus:
	// - editBuf is populated with current line content
	// - debounce timer may be active
	StateEditing

	// StateProcessing: Processing document after user stopped typing
	// Invariants:
	// - userIsTyping = false
	// - document is being re-evaluated
	StateProcessing
)

// State Transition Functions
// These are the ONLY ways to change state explicitly.

// transitionToReady moves the editor to StateReady.
// Called during initialization and after processing completes.
func (m *Model) transitionToReady() {
	// Establish invariants for StateReady

	// INVARIANT: Document must exist with at least 1 block
	// Empty document ("") has 0 blocks, so we create a document with a single newline.
	// After the unicode.go fix, "\n" correctly creates 1 block with 1 empty line (not 2).
	if m.doc == nil || len(m.doc.GetBlocks()) == 0 {
		m.doc, _ = document.NewDocument("\n")
	}

	// INVARIANT: Evaluator must exist
	if m.eval == nil {
		m.eval = implDoc.NewEvaluator()
		if err := m.eval.Evaluate(m.doc); err != nil {
			// Show a brief notice that the document has errors.
			// The preview pane displays detailed error diagnostics —
			// the status bar just flags there's a problem.
			m.statusMsg = "Document has errors — see preview pane"
			m.statusIsErr = true
		}
	}

	// INVARIANT: Cursor at valid position
	if m.cursorLine < 0 {
		m.cursorLine = 0
	}
	totalLines := m.TotalLines()
	if totalLines > 0 && m.cursorLine >= totalLines {
		m.cursorLine = totalLines - 1
	}

	// INVARIANT: Not actively typing
	m.userIsTyping = false

	m.state = StateReady
}

// transitionToEditing moves the editor to StateEditing.
// Called when user starts typing or editing a line.
func (m *Model) transitionToEditing() {
	// Only transition if not already editing
	if m.state == StateEditing {
		return
	}

	// Entry action: Load current line if editBuf is empty
	if m.editBuf == "" {
		m.loadCurrentLineIntoEditBuffer()
	}

	m.userIsTyping = true
	m.state = StateEditing
}

// transitionToProcessing moves the editor to StateProcessing.
// Called when debounce timer fires, ENTER is pressed, or navigation happens.
func (m *Model) transitionToProcessing() {
	m.userIsTyping = false
	m.state = StateProcessing

	// Save current editBuf to document and mark as modified.
	// CRITICAL: Must save even when editBuf is empty - user may have deleted
	// all content from the line. Without this, the document retains old content
	// and transitionToEditing() will reload stale data.
	m.updateCurrentLine(m.editBuf)
	m.modified = true

	// Process document: re-detect block types and re-evaluate
	m.redetectBlockTypes()
	m.reEvaluate()

	// Transition back to Ready
	m.transitionToReady()
}

// checkInvariants verifies that all editor invariants hold.
// This is used in tests and can be called in development builds.
// In production, invariants should ALWAYS hold after ensureReadyState().
func (m *Model) checkInvariants() []string {
	var violations []string

	if m.doc == nil {
		violations = append(violations, "document is nil")
	}

	if m.eval == nil {
		violations = append(violations, "evaluator is nil")
	}

	if m.doc != nil && len(m.doc.GetBlocks()) == 0 {
		violations = append(violations, "document has 0 blocks")
	}

	totalLines := m.TotalLines()
	if totalLines < 0 {
		violations = append(violations, "TotalLines() returned negative value")
	}

	if m.cursorLine < 0 {
		violations = append(violations, "cursorLine is negative")
	}

	if totalLines > 0 && m.cursorLine >= totalLines {
		violations = append(violations, "cursorLine is out of bounds")
	}

	return violations
}
