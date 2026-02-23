package editor

// undo_operations.go — Undo/redo operation application on the Model.
// The UndoManager type and EditOperation types are defined in undo.go.

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleUndo handles Ctrl+Z - undo last edit batch.
func (m Model) handleUndo() (tea.Model, tea.Cmd) {
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

	// Apply operations in reverse order
	for i := len(batch.Operations) - 1; i >= 0; i-- {
		m.applyOperationReverse(batch.Operations[i])
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

	// Reload editBuf from the document for the restored cursor line.
	// applyOperationReverse may have set editBuf for a different line than
	// cursorLine, so we clear and eagerly reload to keep them in sync.
	m.editBufLoaded = false
	m.loadCurrentLineIntoEditBuffer()

	// Re-evaluate document
	m.redetectBlockTypes()
	m.reEvaluate()

	m.statusMsg = "Undo"
	m.modified = true
	return m, nil
}

// handleRedo handles Ctrl+Y - redo last undone edit batch.
func (m Model) handleRedo() (tea.Model, tea.Cmd) {
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

	// Apply operations in forward order (original execution order)
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

	// Reload editBuf from the document for the restored cursor line.
	// applyOperationForward may have set editBuf for a different line than
	// cursorLine, so we clear and eagerly reload to keep them in sync.
	m.editBufLoaded = false
	m.loadCurrentLineIntoEditBuffer()

	// Re-evaluate document
	m.redetectBlockTypes()
	m.reEvaluate()

	m.statusMsg = "Redo"
	m.modified = true
	return m, nil
}

// applyOperationReverse reverses a single edit operation (for undo).
func (m *Model) applyOperationReverse(op EditOperation) {
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
