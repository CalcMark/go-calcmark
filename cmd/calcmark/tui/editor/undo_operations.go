package editor

// undo_operations.go — Undo/redo operation application on the Model.
// The UndoManager type and EditOperation types are defined in undo.go.

import (
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

// handleUndo handles Ctrl+Z - undo last edit batch.
func (m Model) handleUndo() (tea.Model, tea.Cmd) {
	// Clear any active selection — undo changes document content, so stale
	// anchors would reference invalid positions and cause visual artifacts.
	m.ClearSelection()

	// Flush pending edits to document (CRITICAL - Pitfall 4)
	m.transitionToProcessing()

	// Commit any pending batch before undoing
	m.undoManager.CommitCurrentBatch()

	// Get batch to undo
	batch, ok := m.undoManager.Undo()
	if !ok {
		m.statusMsg = "Nothing to undo"
		return m, nil
	}

	// Apply operations in reverse order.
	// IMPORTANT: Do NOT clear frontmatter rawSource before applying operations.
	// The undo operations reference line numbers from the document state when they
	// were recorded. rawSource preserves the exact line structure (including empty
	// lines, formatting, etc.) that the operations were recorded against. Clearing
	// rawSource forces Serialize() to reconstruct from maps, which can change the
	// line count (e.g., losing empty lines added by Enter), causing operations to
	// apply to the wrong lines and producing duplicated content.
	// After all operations are applied, redetectBlockTypes() rebuilds the document
	// from scratch, producing a clean state.
	for _, v := range slices.Backward(batch.Operations) {
		m.applyOperationReverse(v)
	}

	// Restore cursor from first operation (chronologically first - has pre-batch state)
	if len(batch.Operations) > 0 {
		op := batch.Operations[0]
		m.cursorLine = op.CursorLine
		m.cursorCol = op.CursorCol
		m.scrollOffset = op.ScrollOffset

		// Clamp cursor to valid range after undo
		totalLines := m.TotalLines()
		if m.cursorLine >= totalLines {
			m.cursorLine = totalLines - 1
		}
		if m.cursorLine < 0 {
			m.cursorLine = 0
		}
		lines := m.GetLines()
		if m.cursorLine < len(lines) {
			lineLen := runeLen(lines[m.cursorLine])
			if m.cursorCol > lineLen {
				m.cursorCol = lineLen
			}
		}
		if m.cursorCol < 0 {
			m.cursorCol = 0
		}
	}

	// Re-evaluate document.
	// IMPORTANT: redetectBlockTypes MUST run before loadCurrentLineIntoEditBuffer.
	// redetectBlockTypes() rebuilds the document via NewDocument(), which can change
	// the line count (e.g., splitting a line with an embedded "\n" from an OpReplace
	// undo of a line join). If editBuf is loaded before the rebuild, it will hold
	// content from the pre-rebuild line numbering, and the next transitionToProcessing
	// call will flush that stale content to the wrong line, corrupting the document.
	m.editBufLoaded = false
	m.redetectBlockTypes()
	m.reEvaluate()

	// NOW load editBuf from the rebuilt document for the restored cursor line.
	m.loadCurrentLineIntoEditBuffer()

	m.statusMsg = "Undo"
	m.modified = true
	return m, nil
}

// handleRedo handles Ctrl+Y - redo last undone edit batch.
func (m Model) handleRedo() (tea.Model, tea.Cmd) {
	// Clear any active selection — redo changes document content, so stale
	// anchors would reference invalid positions and cause visual artifacts.
	m.ClearSelection()

	// Flush pending edits to document (CRITICAL - Pitfall 4)
	m.transitionToProcessing()

	// Commit any pending batch before redoing
	m.undoManager.CommitCurrentBatch()

	// Get batch to redo
	batch, ok := m.undoManager.Redo()
	if !ok {
		m.statusMsg = "Nothing to redo"
		return m, nil
	}

	// Apply operations in forward order (original execution order).
	// Do NOT clear rawSource — same rationale as handleUndo: operations reference
	// line numbers from the document state when they were recorded.
	for _, op := range batch.Operations {
		m.applyOperationForward(op)
	}

	// Restore cursor to end state (last operation's position after edit)
	if len(batch.Operations) > 0 {
		lastOp := batch.Operations[len(batch.Operations)-1]
		m.cursorLine = lastOp.Line
		// Clamp cursorLine to valid range
		totalLines := m.TotalLines()
		if m.cursorLine >= totalLines {
			m.cursorLine = totalLines - 1
		}
		if m.cursorLine < 0 {
			m.cursorLine = 0
		}
		// Use runeLen for UTF-8 safety
		m.cursorCol = lastOp.Col + runeLen(lastOp.NewText)
		// Clamp cursorCol to valid range
		lines := m.GetLines()
		if m.cursorLine < len(lines) {
			lineLen := runeLen(lines[m.cursorLine])
			if m.cursorCol > lineLen {
				m.cursorCol = lineLen
			}
		}
		if m.cursorCol < 0 {
			m.cursorCol = 0
		}
	}

	// Re-evaluate document.
	// IMPORTANT: redetectBlockTypes MUST run before loadCurrentLineIntoEditBuffer.
	// Same rationale as handleUndo — the rebuild can change line numbering.
	m.editBufLoaded = false
	m.redetectBlockTypes()
	m.reEvaluate()

	// NOW load editBuf from the rebuilt document for the restored cursor line.
	m.loadCurrentLineIntoEditBuffer()

	m.statusMsg = "Redo"
	m.modified = true
	return m, nil
}

// applyDocReplace rebuilds the document from the given content string.
// Used by OpDocReplace undo/redo to atomically replace the entire document.
func (m *Model) applyDocReplace(content string) {
	newDoc, err := document.NewDocument(content)
	if err != nil {
		// Should not fail for content that was previously valid,
		// but handle gracefully by keeping the current document.
		m.frontmatterErr = err
		return
	}
	m.doc = newDoc
	m.frontmatterErr = nil
	m.fullReEvaluate()
	m.autoPinVariables()
	m.editBufLoaded = false
}

// applyOperationReverse reverses a single edit operation (for undo).
func (m *Model) applyOperationReverse(op EditOperation) {
	if op.Type == OpDocReplace {
		// Atomic document replacement — rebuild from OldText (the pre-edit content)
		m.applyDocReplace(op.OldText)
		return
	}

	lines := m.GetLines()

	switch op.Type {
	case OpInsertLine:
		// Undoing line insert: delete the created line and restore original
		// The new line was created at Line+1
		if op.Line+1 < len(lines) {
			// Delete line at Line+1
			m.cursorLine = op.Line + 1
			m.deleteLine()
		}
		// Restore original line content
		if op.Line < len(m.GetLines()) {
			m.cursorLine = op.Line
			m.editBuf = op.OldText
			m.updateCurrentLine(op.OldText)
		}
		return

	case OpDeleteLine:
		// Undoing line delete: insert the line back
		// First ensure we're at the right position
		m.cursorLine = op.Line
		if op.Line > 0 {
			m.cursorLine = op.Line - 1
		}
		// Insert a new line
		m.insertLineBelow()
		// Set its content
		m.editBuf = op.OldText
		m.updateCurrentLine(op.OldText)
		return
	}

	// For single-line operations, validate bounds
	if op.Line < 0 || op.Line >= len(lines) {
		return
	}

	line := lines[op.Line]
	var newLine string
	lineRuneLen := runeLen(line)

	switch op.Type {
	case OpInsert:
		// Undoing insert: delete the NewText at position (UTF-8 safe)
		if op.Col >= 0 && op.Col <= lineRuneLen {
			newLine, _ = runeDelete(line, op.Col, runeLen(op.NewText))
		} else {
			newLine = line
		}
	case OpDelete:
		// Undoing delete: insert the OldText at position (UTF-8 safe)
		if op.Col >= 0 && op.Col <= lineRuneLen {
			newLine = runeInsert(line, op.Col, op.OldText)
		} else {
			newLine = line
		}
	case OpReplace:
		// Undoing replace: replace NewText with OldText (UTF-8 safe)
		if op.Col >= 0 && op.Col <= lineRuneLen {
			// Delete NewText length, then insert OldText
			temp, _ := runeDelete(line, op.Col, runeLen(op.NewText))
			newLine = runeInsert(temp, op.Col, op.OldText)
		} else {
			newLine = line
		}
	}

	// Update the line in the document
	m.cursorLine = op.Line
	m.editBuf = newLine
	m.updateCurrentLine(newLine)
}

// applyOperationForward applies a single edit operation (for redo).
func (m *Model) applyOperationForward(op EditOperation) {
	if op.Type == OpDocReplace {
		// Atomic document replacement — rebuild from NewText (the post-edit content)
		m.applyDocReplace(op.NewText)
		return
	}

	lines := m.GetLines()

	switch op.Type {
	case OpInsertLine:
		// Redo line insert: split the line again (UTF-8 safe)
		if op.Line < len(lines) {
			// Set line to content before split
			textBefore, _ := runeSlice(op.OldText, op.Col)
			m.cursorLine = op.Line
			m.editBuf = textBefore
			m.updateCurrentLine(textBefore)
			// Insert new line with content after split
			m.insertLineBelow()
			m.editBuf = op.NewText
			m.updateCurrentLine(op.NewText)
		}
		return

	case OpDeleteLine:
		// Redo line delete: delete the line again
		if op.Line < len(lines) {
			m.cursorLine = op.Line
			m.deleteLine()
		}
		return
	}

	// For single-line operations, validate bounds
	if op.Line < 0 || op.Line >= len(lines) {
		return
	}

	line := lines[op.Line]
	var newLine string
	lineRuneLen := runeLen(line)

	switch op.Type {
	case OpInsert:
		// Redo insert: insert the NewText at position (UTF-8 safe)
		if op.Col >= 0 && op.Col <= lineRuneLen {
			newLine = runeInsert(line, op.Col, op.NewText)
		} else {
			newLine = line
		}
	case OpDelete:
		// Redo delete: delete OldText.length chars at position (UTF-8 safe)
		if op.Col >= 0 && op.Col <= lineRuneLen {
			newLine, _ = runeDelete(line, op.Col, runeLen(op.OldText))
		} else {
			newLine = line
		}
	case OpReplace:
		// Redo replace: replace OldText with NewText (UTF-8 safe)
		if op.Col >= 0 && op.Col <= lineRuneLen {
			// Delete OldText length, then insert NewText
			temp, _ := runeDelete(line, op.Col, runeLen(op.OldText))
			newLine = runeInsert(temp, op.Col, op.NewText)
		} else {
			newLine = line
		}
	}

	// Update the line in the document
	m.cursorLine = op.Line
	m.editBuf = newLine
	m.updateCurrentLine(newLine)
}
