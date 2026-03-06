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
	//
	// IMPORTANT: When frontmatter exists but all body blocks have been removed
	// (e.g., line join of the only body line with a frontmatter line), we must
	// preserve the frontmatter. Unconditionally replacing with NewDocument("\n")
	// would discard all frontmatter content.
	if m.doc == nil || len(m.doc.GetBlocks()) == 0 {
		if m.doc != nil && m.doc.GetFrontmatter() != nil {
			// Preserve frontmatter: rebuild document from current content + empty body line
			content := m.getDocumentContent()
			if content == "" {
				content = "\n"
			}
			newDoc, err := document.NewDocument(content + "\n")
			if err == nil {
				m.doc = newDoc
				m.fullReEvaluate()
			}
			// If parse fails (e.g., invalid YAML in rawSource), keep the existing
			// document. The 0-blocks state is temporarily violated but safe: all
			// block iteration loops handle empty slices gracefully, and the user
			// can fix the frontmatter to restore a valid state.
		} else {
			newDoc, err := document.NewDocument("\n")
			if err != nil {
				// Should never happen for a literal newline, but surface it defensively
				m.statusMsg = "Internal error creating empty document"
				m.statusIsErr = true
			} else {
				m.doc = newDoc
			}
		}
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

	// Entry action: Load current line if not yet loaded
	if !m.editBufLoaded {
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
	// Only save when editBufLoaded is true — this means editBuf has been loaded
	// or modified for the current line. When editBufLoaded is false (e.g., after
	// undo/redo reset), there is nothing to save and writing editBuf="" would
	// destroy the document's actual content.
	if m.editBufLoaded {
		m.updateCurrentLine(m.editBuf)
		m.modified = true
	}

	// Process document: re-detect block types and re-evaluate
	m.redetectBlockTypes()
	m.reEvaluate()

	// Transition back to Ready
	m.transitionToReady()
}
