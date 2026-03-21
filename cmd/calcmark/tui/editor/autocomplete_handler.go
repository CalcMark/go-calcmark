package editor

// autocomplete_handler.go — Autocomplete key handling, state management, and acceptance.

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/spec/features"
)

// minAutocompletePrefix is the minimum number of characters needed to trigger autosuggest.
const minAutocompletePrefix = 2

// handleAutocompleteKey processes keys when autocomplete popup is visible.
// Typing continues to work normally — we just update suggestions.
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
		m.exitAutocomplete()
		return m, nil
	case "tab":
		return m.acceptAutocomplete()
	case "backspace":
		beforeLine := m.cursorLine
		beforeCol := m.cursorCol
		beforeScroll := m.scrollOffset

		m.transitionToEditing()
		if m.cursorCol > 0 && runeLen(m.editBuf) > 0 {
			var deletedChar string
			m.editBuf, deletedChar = runeDelete(m.editBuf, m.cursorCol-1, 1)
			m.cursorCol--

			op := EditOperation{
				Type:         OpDelete,
				Line:         beforeLine,
				Col:          m.cursorCol,
				OldText:      deletedChar,
				NewText:      "",
				CursorLine:   beforeLine,
				CursorCol:    beforeCol,
				ScrollOffset: beforeScroll,
			}
			undoCmd := m.recordEdit(op)

			m.updateAutocompleteState()
			debounceModel, debounceCmd := m.debounceUpdate()
			return debounceModel, tea.Batch(debounceCmd, undoCmd)
		}
		m.updateAutocompleteState()
		return m.debounceUpdate()
	case "space":
		m.exitAutocomplete()
		return m.handleRuneInput([]rune{' '})
	case "enter":
		if len(m.autocompleteState.Suggestions) > 0 {
			return m.acceptAutocomplete()
		}
		m.exitAutocomplete()
		return m.handleEnterKey()
	default:
		if msg.Text != "" {
			beforeLine := m.cursorLine
			beforeCol := m.cursorCol
			beforeScroll := m.scrollOffset

			m.transitionToEditing()
			for _, r := range msg.Text {
				m.insertRune(r)
			}

			op := EditOperation{
				Type:         OpInsert,
				Line:         beforeLine,
				Col:          beforeCol,
				OldText:      "",
				NewText:      msg.Text,
				CursorLine:   beforeLine,
				CursorCol:    beforeCol,
				ScrollOffset: beforeScroll,
			}
			undoCmd := m.recordEdit(op)

			m.updateAutocompleteState()
			debounceModel, debounceCmd := m.debounceUpdate()
			return debounceModel, tea.Batch(debounceCmd, undoCmd)
		}
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
// Called after every character typed to show/hide the popup automatically.
func (m *Model) updateAutocompleteState() {
	// No autocomplete inside frontmatter — it's YAML, not CalcMark.
	if m.cursorLine < m.frontmatterLineCount() {
		if m.mode == StateAutocomplete {
			m.exitAutocomplete()
		}
		return
	}

	m.loadCurrentLineIntoEditBuffer()
	prefix := features.ExtractPrefix(m.editBuf, m.cursorCol)

	if len(prefix) < minAutocompletePrefix {
		if m.mode == StateAutocomplete {
			m.exitAutocomplete()
		}
		return
	}

	if m.suggestionSource == nil {
		return
	}

	if m.combinedSource != nil {
		m.combinedSource.CursorLine = m.cursorLine
	}

	suggestions := m.suggestionSource.GetSuggestions(prefix)

	// Inside a function call, suppress function/NL suggestions — the status bar
	// already shows parameter help. Keep variable and unit suggestions.
	cursorCtx := GetCursorContext(m.editBuf, m.cursorCol)
	if cursorCtx.InFunctionCall {
		suggestions = filterNonFunctionSuggestions(suggestions)
	}

	if len(suggestions) == 0 {
		if m.mode == StateAutocomplete {
			m.exitAutocomplete()
		}
		return
	}

	popupWidth, popupHeight := m.calculatePopupDimensions(suggestions)

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

// isFunctionSuggestion returns true for function and NL example suggestions.
func isFunctionSuggestion(s components.Suggestion) bool {
	tag := suggestionTag(s.Category)
	return tag == "fn" || tag == "nl"
}

// filterNonFunctionSuggestions removes function and NL example suggestions,
// keeping only variables and units.
func filterNonFunctionSuggestions(suggestions []components.Suggestion) []components.Suggestion {
	filtered := suggestions[:0:0]
	for _, s := range suggestions {
		if !isFunctionSuggestion(s) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// calculatePopupDimensions determines the popup size based on suggestions.
func (m *Model) calculatePopupDimensions(suggestions []components.Suggestion) (width, height int) {
	width = 30 // minimum width for readability
	for _, s := range suggestions {
		tag := suggestionTag(s.Category)
		w := len(tag) + 1
		if s.Syntax != "" {
			w += len(s.Syntax)
			if idx := strings.Index(s.Name, " ("); idx >= 0 {
				w += 1 + len(s.Name[idx+1:])
			}
		} else {
			w += len(s.Name)
		}
		if w+6 > width {
			width = w + 6
		}
	}

	maxWidth := max(m.width*7/10, 40)
	if width > maxWidth {
		width = maxWidth
	}

	height = min(len(suggestions), 8)
	return width, height
}

// acceptAutocomplete inserts the selected suggestion at the cursor.
// Records an OpReplace on the undo stack so the acceptance can be undone.
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

	// For functions, add opening paren so parameter help is shown
	if strings.Contains(selected.Syntax, "(") {
		insertText += "("
	}

	beforeCol := m.cursorCol
	beforeScroll := m.scrollOffset

	prefix := m.autocompleteState.Prefix
	prefixStart := max(m.cursorCol-runeLen(prefix), 0)

	m.undoManager.ForceBoundary()

	beforePrefix, _ := runeSlice(m.editBuf, prefixStart)
	_, afterCursor := runeSlice(m.editBuf, m.cursorCol)
	m.editBuf = beforePrefix + insertText + afterCursor
	m.cursorCol = prefixStart + runeLen(insertText)

	m.undoManager.AddOperation(EditOperation{
		Type:         OpReplace,
		Line:         m.cursorLine,
		Col:          prefixStart,
		OldText:      prefix,
		NewText:      insertText,
		CursorLine:   m.cursorLine,
		CursorCol:    beforeCol,
		ScrollOffset: beforeScroll,
	})

	m.undoManager.ForceBoundary()

	m.modified = true
	m.exitAutocomplete()
	m.transitionToEditing()
	return m.debounceUpdate()
}
