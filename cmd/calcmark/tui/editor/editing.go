package editor

// editing.go — Text editing operations (insert, delete, line manipulation).

import (
	"strings"
	"time"

	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
	tea "github.com/charmbracelet/bubbletea"
)

// handleRuneInput handles character input - regular typing only.
// All vim keys have been removed - user is ALWAYS in editing mode.
func (m Model) handleRuneInput(runes []rune) (tea.Model, tea.Cmd) {
	if len(runes) == 0 {
		return m, nil
	}

	// Typing clears selection
	m.ClearSelection()

	// Capture state BEFORE the edit for undo
	beforeLine := m.cursorLine
	beforeCol := m.cursorCol
	beforeScroll := m.scrollOffset

	// Insert all characters at cursor position
	m.transitionToEditing()

	for _, r := range runes {
		m.insertRune(r)
	}

	// Record the insert operation
	insertText := string(runes)
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

	// Check for autocomplete suggestions after typing (use pointer to modify)
	(&m).updateAutocompleteState()

	// Return batch of commands: debounce for evaluation + undo grouping timer
	debounceCmd := tea.Tick(evalDebounceDelay, func(t time.Time) tea.Msg {
		return evalDebounceMsg{editBufSnapshot: m.editBuf}
	})

	return m, tea.Batch(debounceCmd, undoCmd)
}

func (m Model) handleEnterKey() (tea.Model, tea.Cmd) {
	// Typing clears selection
	m.ClearSelection()

	// Capture state BEFORE the edit for undo
	beforeLine := m.cursorLine
	beforeCol := m.cursorCol
	beforeScroll := m.scrollOffset

	m.loadCurrentLineIntoEditBuffer()

	// Enter always creates immediate boundary per CONTEXT.md
	m.undoManager.ForceBoundary()

	// Split line at cursor position (UTF-8 safe)
	textBefore, textAfter := runeSlice(m.editBuf, m.cursorCol)

	// Record as InsertLine operation (line split creates new line)
	// OldText = original full line content (for restoration on undo)
	// NewText = the part that moves to new line (textAfter)
	// Col = cursor position where split occurred
	op := EditOperation{
		Type:         OpInsertLine,
		Line:         beforeLine,
		Col:          beforeCol,
		OldText:      m.editBuf, // Original line content before split
		NewText:      textAfter, // Content moved to new line
		CursorLine:   beforeLine,
		CursorCol:    beforeCol,
		ScrollOffset: beforeScroll,
	}

	// Frontmatter lines need atomic handling: the three-step sequence
	// (updateCurrentLine → insertLineBelow → updateCurrentLine) rebuilds the
	// document three times, and an intermediate state can produce invalid YAML
	// that silently fails, leaving stale data in the document.
	// Instead, do the line split as a single document rebuild.
	if m.cursorLine < m.frontmatterLineCount() {
		lines := m.GetLines()
		// Split: replace current line with textBefore, insert textAfter below
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:m.cursorLine]...)
		newLines = append(newLines, textBefore)
		newLines = append(newLines, textAfter)
		newLines = append(newLines, lines[m.cursorLine+1:]...)

		content := strings.Join(newLines, "\n") + "\n"
		newDoc, err := document.NewDocument(content)
		if err != nil {
			// YAML is invalid, but Enter must still work — preserve the split
			// in rawSource so the user can keep editing.
			m.frontmatterErr = err
			if fm := m.doc.GetFrontmatter(); fm != nil {
				fmCount := m.frontmatterLineCount()
				// newLines has one extra line from the split, so frontmatter is fmCount+1
				newRaw := strings.Join(newLines[:fmCount+1], "\n") + "\n"
				fm.SetRawSource(newRaw)
			}
		} else {
			m.frontmatterErr = nil
			m.doc = newDoc
			m.eval = implDoc.NewEvaluator()
			_ = m.eval.Evaluate(m.doc)
		}
		m.cursorLine = beforeLine + 1
		m.cursorCol = 0
		m.editBuf = textAfter
		m.adjustScrollForCursor()
		m.autoPinVariables()
	} else {
		// Non-frontmatter: use the standard three-step approach
		m.editBuf = textBefore
		m.updateCurrentLine(m.editBuf)
		m.insertLineBelow()
		m.editBuf = textAfter
		m.cursorCol = 0
		m.updateCurrentLine(m.editBuf)
	}

	// Process document changes immediately on ENTER
	m.redetectBlockTypes()
	m.reEvaluate()
	m.modified = true
	m.userIsTyping = false

	// Record the operation (commits immediately due to ForceBoundary)
	m.undoManager.AddOperation(op)
	m.undoManager.CommitCurrentBatch()

	return m, nil
}

func (m Model) handleBackspaceKey() (tea.Model, tea.Cmd) {
	// Typing clears selection
	m.ClearSelection()

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

		// Record the delete operation
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

		debounceCmd := tea.Tick(evalDebounceDelay, func(t time.Time) tea.Msg {
			return evalDebounceMsg{editBufSnapshot: m.editBuf}
		})
		return m, tea.Batch(debounceCmd, undoCmd)
	} else if m.cursorCol == 0 && m.cursorLine > 0 {
		// At column 0 - join current line with previous line
		// This is a line join - force boundary per CONTEXT.md
		m.undoManager.ForceBoundary()

		currentContent := m.editBuf
		prevLine := m.cursorLine - 1

		// Get previous line content
		lines := m.GetLines()
		prevContent := ""
		if prevLine < len(lines) {
			prevContent = lines[prevLine]
		}

		// Record as Replace operation (more complex than simple delete)
		op := EditOperation{
			Type:         OpReplace,
			Line:         prevLine,
			Col:          runeLen(prevContent),  // Rune position, not byte position
			OldText:      "\n" + currentContent, // Conceptually: newline + current line content
			NewText:      currentContent,        // Joined content
			CursorLine:   beforeLine,
			CursorCol:    beforeCol,
			ScrollOffset: beforeScroll,
		}

		// Delete current line
		m.deleteLine()

		// Move to previous line and append current content
		m.cursorLine = prevLine
		m.cursorCol = runeLen(prevContent) // Rune position, not byte position
		m.editBuf = prevContent + currentContent

		m.transitionToEditing()
		undoCmd := m.recordEdit(op)

		debounceCmd := tea.Tick(evalDebounceDelay, func(t time.Time) tea.Msg {
			return evalDebounceMsg{editBufSnapshot: m.editBuf}
		})
		return m, tea.Batch(debounceCmd, undoCmd)
	}

	return m, nil
}

func (m Model) handleDeleteKey() (tea.Model, tea.Cmd) {
	// Typing clears selection
	m.ClearSelection()

	// Capture state BEFORE the edit for undo
	beforeLine := m.cursorLine
	beforeCol := m.cursorCol
	beforeScroll := m.scrollOffset

	// Transition to editing state BEFORE modifying editBuf.
	// This ensures editBuf is loaded and state is set correctly.
	// CRITICAL: Must be called first, because transitionToEditing reloads
	// editBuf if it's empty, which would undo any deletion.
	m.transitionToEditing()

	if m.cursorCol < runeLen(m.editBuf) {
		// Delete character at cursor (UTF-8 safe)
		var deletedChar string
		m.editBuf, deletedChar = runeDelete(m.editBuf, m.cursorCol, 1)

		// Record the delete operation
		op := EditOperation{
			Type:         OpDelete,
			Line:         beforeLine,
			Col:          beforeCol, // Rune position, not byte position
			OldText:      deletedChar,
			NewText:      "",
			CursorLine:   beforeLine,
			CursorCol:    beforeCol,
			ScrollOffset: beforeScroll,
		}
		undoCmd := m.recordEdit(op)

		debounceCmd := tea.Tick(evalDebounceDelay, func(t time.Time) tea.Msg {
			return evalDebounceMsg{editBufSnapshot: m.editBuf}
		})
		return m, tea.Batch(debounceCmd, undoCmd)
	} else if m.cursorLine < m.TotalLines()-1 {
		// At end of line - join with next line
		// This is a line join - force boundary per CONTEXT.md
		m.undoManager.ForceBoundary()

		nextLine := m.cursorLine + 1
		lines := m.GetLines()
		nextContent := ""
		if nextLine < len(lines) {
			nextContent = lines[nextLine]
		}

		// Record as Replace operation
		op := EditOperation{
			Type:         OpReplace,
			Line:         beforeLine,
			Col:          beforeCol,
			OldText:      "\n" + nextContent, // Conceptually: newline + next line content
			NewText:      nextContent,        // Joined content
			CursorLine:   beforeLine,
			CursorCol:    beforeCol,
			ScrollOffset: beforeScroll,
		}

		// Save current position
		currentCol := m.cursorCol

		// Save current line content before manipulating
		m.saveCurrentLine(true)
		m.cursorLine = nextLine
		m.deleteLine()

		// Move back and append next line content
		m.cursorLine = nextLine - 1
		lines = m.GetLines()
		if m.cursorLine < len(lines) {
			m.editBuf = lines[m.cursorLine] + nextContent
		}
		m.cursorCol = currentCol

		undoCmd := m.recordEdit(op)
		debounceCmd := tea.Tick(evalDebounceDelay, func(t time.Time) tea.Msg {
			return evalDebounceMsg{editBufSnapshot: m.editBuf}
		})
		return m, tea.Batch(debounceCmd, undoCmd)
	}

	return m, nil
}

func (m Model) handleSpaceKey() (tea.Model, tea.Cmd) {
	// Capture state BEFORE the edit for undo
	beforeLine := m.cursorLine
	beforeCol := m.cursorCol
	beforeScroll := m.scrollOffset

	m.loadCurrentLineIntoEditBuffer()
	m.editBuf = runeInsert(m.editBuf, m.cursorCol, " ")
	m.cursorCol++
	m.transitionToEditing()

	// Record the insert operation (space is just a character)
	op := EditOperation{
		Type:         OpInsert,
		Line:         beforeLine,
		Col:          beforeCol,
		OldText:      "",
		NewText:      " ",
		CursorLine:   beforeLine,
		CursorCol:    beforeCol,
		ScrollOffset: beforeScroll,
	}
	undoCmd := m.recordEdit(op)

	debounceCmd := tea.Tick(evalDebounceDelay, func(t time.Time) tea.Msg {
		return evalDebounceMsg{editBufSnapshot: m.editBuf}
	})
	return m, tea.Batch(debounceCmd, undoCmd)
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
	m.editBuf = runeInsert(m.editBuf, m.cursorCol, string(r))
	m.cursorCol++
}

func (m Model) debounceUpdate() (tea.Model, tea.Cmd) {
	snapshot := m.editBuf
	return m, tea.Tick(evalDebounceDelay, func(t time.Time) tea.Msg {
		return evalDebounceMsg{editBufSnapshot: snapshot}
	})
}

// recordEdit adds an operation to the undo history and starts a grouping timer.
// Returns a tea.Cmd for the grouping timer.
func (m *Model) recordEdit(op EditOperation) tea.Cmd {
	m.undoManager.AddOperation(op)
	m.undoGroupID = m.undoManager.GetGroupID()

	return tea.Tick(undoGroupingDelay, func(t time.Time) tea.Msg {
		return undoGroupMsg{batchID: m.undoGroupID}
	})
}

// loadCurrentLineIntoEditBuffer ensures editBuf is loaded with current line content.
// This makes the user ALWAYS able to edit - no mode switching needed.
// Uses editBufLoaded to distinguish "not yet loaded" from "user emptied the line".
func (m *Model) loadCurrentLineIntoEditBuffer() {
	if m.editBufLoaded {
		return // Already loaded — editBuf="" means user intentionally emptied the line
	}
	lines := m.GetLines()
	if m.cursorLine < len(lines) {
		m.editBuf = lines[m.cursorLine]
	}
	m.editBufLoaded = true
}

// saveCurrentLine saves the edit buffer to the current line without changing mode.
// The user is ALWAYS able to edit - no mode switching needed.
func (m *Model) saveCurrentLine(save bool) {
	if save && m.editBufLoaded {
		// Special case: empty document with content in edit buffer
		// Create the document from the buffer content
		if len(m.GetLines()) == 0 {
			newDoc, err := document.NewDocument(m.editBuf)
			if err == nil {
				m.doc = newDoc
				m.eval = implDoc.NewEvaluator()
				_ = m.eval.Evaluate(m.doc)
				m.modified = true
				m.autoPinVariables()
			}
		} else if len(m.GetLines()) > 0 {
			// Normal case: update existing line
			m.updateCurrentLine(m.editBuf)
			m.modified = true

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
	m.editBufLoaded = true // Explicitly loaded for the new line

	// Try to preserve column position, clamp to line length
	m.cursorCol = min(savedCol, runeLen(m.editBuf))

	// Adjust scroll to keep cursor visible with margin
	m.adjustScrollForCursor()

	// Stay in edit mode (don't change m.mode)
}

// redetectBlockTypes rebuilds the document to properly detect block types.
// This is needed when editing changes a line from calculation to markdown or vice versa.
// Also captures frontmatter parsing errors for diagnostics display.
func (m *Model) redetectBlockTypes() {
	// Get current document content
	content := m.getDocumentContent()

	// Rebuild document with proper block detection
	newDoc, err := document.NewDocument(content)
	if err != nil {
		// Capture frontmatter errors for diagnostics — keep the old document
		m.frontmatterErr = err
		return
	}

	// Clear frontmatter error on successful parse
	m.frontmatterErr = nil

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
// Accounts for frontmatter lines: if cursor is in frontmatter region,
// triggers a full document rebuild instead of single-block update.
func (m *Model) updateCurrentLine(newContent string) {
	fmCount := m.frontmatterLineCount()
	targetLine := m.cursorLine - fmCount

	// If cursor is on a frontmatter line, rebuild the entire document with the
	// modified line. Frontmatter is managed by the document parser, not individual
	// blocks, so targeted updates aren't possible — we must reparse.
	if targetLine < 0 {
		lines := m.GetLines()
		if m.cursorLine >= len(lines) {
			return
		}
		lines[m.cursorLine] = newContent
		content := strings.Join(lines, "\n") + "\n"
		newDoc, err := document.NewDocument(content)
		if err != nil {
			// YAML is invalid, but we must preserve the user's text.
			// Update the raw source so GetLines()/Serialize() reflects the edit,
			// while keeping the old parsed data (globals, exchange) intact.
			m.frontmatterErr = err
			if fm := m.doc.GetFrontmatter(); fm != nil {
				newRaw := strings.Join(lines[:fmCount], "\n") + "\n"
				fm.SetRawSource(newRaw)
			}
			return
		}
		m.frontmatterErr = nil
		m.doc = newDoc
		m.eval = implDoc.NewEvaluator()
		_ = m.eval.Evaluate(m.doc)
		m.autoPinVariables()
		return
	}

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
			if lineIdx == targetLine {
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

	// Adjust scroll to keep cursor visible with margin
	m.adjustScrollForCursor()

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

// deleteLine deletes the current line (dd command).
func (m *Model) deleteLine() {
	lines := m.GetLines()
	if m.cursorLine >= len(lines) {
		return
	}

	// Copy to yank buffer first
	m.yankBuffer = lines[m.cursorLine]

	// If cursor is on a frontmatter line, rebuild the whole document
	fmCount := m.frontmatterLineCount()
	if m.cursorLine < fmCount {
		// Rebuild without this line
		newLines := make([]string, 0, len(lines)-1)
		newLines = append(newLines, lines[:m.cursorLine]...)
		newLines = append(newLines, lines[m.cursorLine+1:]...)
		content := strings.Join(newLines, "\n") + "\n"
		newDoc, err := document.NewDocument(content)
		if err != nil {
			m.frontmatterErr = err
			return
		}
		m.frontmatterErr = nil
		m.doc = newDoc
		m.eval = implDoc.NewEvaluator()
		_ = m.eval.Evaluate(m.doc)
		m.modified = true
		// Adjust cursor
		total := m.TotalLines()
		if m.cursorLine >= total && total > 0 {
			m.cursorLine = total - 1
		}
		m.autoPinVariables()
		return
	}

	// Find and update the block containing this line
	targetLine := m.cursorLine - fmCount
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
			if lineIdx == targetLine {
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
				m.reEvaluate()
				m.InvalidateAlignedCache()

				// Adjust cursor if needed
				total := m.TotalLines()
				if m.cursorLine >= total && total > 0 {
					m.cursorLine = total - 1
				}

				// Adjust scroll offset if it's now past document end
				if m.scrollOffset > 0 && m.scrollOffset >= total {
					m.scrollOffset = max(total-1, 0)
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
	m.reEvaluate()
	m.statusMsg = "Line pasted"
}

// insertFrontmatter inserts a default YAML frontmatter block at the top of the document.
// Returns the updated model and command. No-op if frontmatter already exists.
func (m Model) insertFrontmatter() (tea.Model, tea.Cmd) {
	if m.doc.GetFrontmatter() != nil {
		m.statusMsg = "Frontmatter already exists"
		return m, nil
	}

	// Build new content with default frontmatter prepended
	fmBlock := "---\nexchange:\n  USD_EUR: 0.92\nglobals:\n  my_var: 42\n---\n"
	content := fmBlock + m.getDocumentContent()

	// Rebuild document via the spec layer (parsing stays in spec/document)
	newDoc, err := document.NewDocument(content)
	if err != nil {
		m.statusMsg = "Error inserting frontmatter"
		m.statusIsErr = true
		return m, nil
	}

	m.doc = newDoc
	m.frontmatterErr = nil
	m.eval = implDoc.NewEvaluator()
	_ = m.eval.Evaluate(m.doc)

	// Set cursor to the exchange rate value line (line 2, 0-indexed) so user can edit immediately
	m.cursorLine = 2
	m.cursorCol = 0
	m.editBuf = ""
	m.editBufLoaded = false // New line — needs loading

	m.globalsExpanded = true
	m.modified = true
	m.autoPinVariables()
	m.InvalidateAlignedCache()
	m.statusMsg = "Frontmatter inserted"
	return m, nil
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
	m.reEvaluate()
	m.statusMsg = "Line pasted above"
}
