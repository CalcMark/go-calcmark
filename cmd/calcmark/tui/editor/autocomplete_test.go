package editor

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/CalcMark/go-calcmark/spec/document"
)

func TestFunctionSuggestionSource_GetSuggestions(t *testing.T) {
	source := NewFunctionSuggestionSource()

	tests := []struct {
		name         string
		prefix       string
		wantMatch    string // Expected InsertText in results
		wantMinCount int    // Minimum number of results
		wantNotMatch string // Should NOT be in results
	}{
		{
			name:         "av prefix matches avg",
			prefix:       "av",
			wantMatch:    "avg",
			wantMinCount: 1,
		},
		{
			name:         "mean prefix matches avg via synonym",
			prefix:       "mean",
			wantMatch:    "avg", // synonym should still insert primary name
			wantMinCount: 1,
		},
		{
			name:         "average prefix matches avg via synonym",
			prefix:       "average",
			wantMatch:    "avg",
			wantMinCount: 1,
		},
		{
			name:         "sq prefix matches sqrt",
			prefix:       "sq",
			wantMatch:    "sqrt",
			wantMinCount: 1,
		},
		{
			name:         "case insensitive matching",
			prefix:       "AV",
			wantMatch:    "avg",
			wantMinCount: 1,
		},
		{
			name:         "empty prefix returns all functions",
			prefix:       "",
			wantMinCount: 12, // All 12 builtin functions
		},
		{
			name:         "xyz prefix matches nothing",
			prefix:       "xyz",
			wantMinCount: 0,
			wantNotMatch: "avg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := source.GetSuggestions(tt.prefix)

			if len(suggestions) < tt.wantMinCount {
				t.Errorf("got %d suggestions, want at least %d", len(suggestions), tt.wantMinCount)
			}

			if tt.wantMatch != "" {
				found := false
				for _, s := range suggestions {
					if s.InsertText == tt.wantMatch {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected to find InsertText=%q in suggestions", tt.wantMatch)
				}
			}

			if tt.wantNotMatch != "" {
				for _, s := range suggestions {
					if s.InsertText == tt.wantNotMatch {
						t.Errorf("did not expect to find InsertText=%q in suggestions", tt.wantNotMatch)
					}
				}
			}
		})
	}
}

func TestUnitSuggestionSource_GetSuggestions(t *testing.T) {
	source := NewUnitSuggestionSource()

	tests := []struct {
		name         string
		prefix       string
		wantMatch    string // Expected InsertText in results
		wantMinCount int
	}{
		{
			name:         "met prefix matches meter",
			prefix:       "met",
			wantMatch:    "meter",
			wantMinCount: 1,
		},
		{
			name:         "m prefix matches multiple units",
			prefix:       "m",
			wantMinCount: 3, // meter, millimeter, mile, milligram, milliliter, etc.
		},
		{
			name:         "kg prefix matches kilogram",
			prefix:       "kg",
			wantMatch:    "kilogram",
			wantMinCount: 1,
		},
		{
			name:         "case insensitive",
			prefix:       "MET",
			wantMatch:    "meter",
			wantMinCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := source.GetSuggestions(tt.prefix)

			if len(suggestions) < tt.wantMinCount {
				t.Errorf("got %d suggestions, want at least %d", len(suggestions), tt.wantMinCount)
			}

			if tt.wantMatch != "" {
				found := false
				for _, s := range suggestions {
					if s.InsertText == tt.wantMatch {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected to find InsertText=%q in suggestions", tt.wantMatch)
				}
			}
		})
	}
}

func TestVariableSuggestionSource_GetSuggestions(t *testing.T) {
	vars := map[string]string{
		"price":    "100",
		"quantity": "5",
		"total":    "500",
	}

	source := NewVariableSuggestionSource(func() map[string]string {
		return vars
	})

	tests := []struct {
		name         string
		prefix       string
		wantMatch    string
		wantMinCount int
	}{
		{
			name:         "pr prefix matches price",
			prefix:       "pr",
			wantMatch:    "price",
			wantMinCount: 1,
		},
		{
			name:         "empty prefix returns all variables",
			prefix:       "",
			wantMinCount: 3,
		},
		{
			name:         "case insensitive",
			prefix:       "PR",
			wantMatch:    "price",
			wantMinCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := source.GetSuggestions(tt.prefix)

			if len(suggestions) < tt.wantMinCount {
				t.Errorf("got %d suggestions, want at least %d", len(suggestions), tt.wantMinCount)
			}

			if tt.wantMatch != "" {
				found := false
				for _, s := range suggestions {
					if s.InsertText == tt.wantMatch {
						found = true
						// Verify description shows the value
						if s.Description != vars[tt.wantMatch] {
							t.Errorf("expected description=%q, got %q", vars[tt.wantMatch], s.Description)
						}
						break
					}
				}
				if !found {
					t.Errorf("expected to find InsertText=%q in suggestions", tt.wantMatch)
				}
			}
		})
	}
}

func TestCombinedSuggestionSource_GetSuggestions(t *testing.T) {
	funcSource := NewFunctionSuggestionSource()
	unitSource := NewUnitSuggestionSource()
	varSource := NewVariableSuggestionSource(func() map[string]string {
		return map[string]string{"avgPrice": "100"}
	})

	combined := NewCombinedSuggestionSource(funcSource, unitSource, varSource)

	// "av" should match both avg function and avgPrice variable
	suggestions := combined.GetSuggestions("av")

	if len(suggestions) < 2 {
		t.Errorf("expected at least 2 suggestions for 'av', got %d", len(suggestions))
	}

	// Verify ordering: functions should come before variables
	foundFunc := -1
	foundVar := -1
	for i, s := range suggestions {
		if s.InsertText == "avg" {
			foundFunc = i
		}
		if s.InsertText == "avgPrice" {
			foundVar = i
		}
	}

	if foundFunc == -1 {
		t.Error("expected to find avg function")
	}
	if foundVar == -1 {
		t.Error("expected to find avgPrice variable")
	}
	if foundFunc != -1 && foundVar != -1 && foundFunc > foundVar {
		t.Error("expected functions to appear before variables in suggestions")
	}
}

// TestBackspaceLastCharAfterDebounce reproduces the bug where pressing backspace
// in autocomplete mode to delete the last prefix character leaves the character
// visible. The root cause: after the debounce fires, the typed character is saved
// to the document. When backspace clears editBuf to "", the view falls back to
// document content (which still has the character) instead of showing empty.
func TestBackspaceLastCharAfterDebounce(t *testing.T) {
	content := "x = 10\n"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Move to line 0, col 0
	m.cursorLine = 0
	m.cursorCol = 0
	m.editBuf = ""

	// 1. Type "a" — triggers autocomplete
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	result, cmd := m.Update(keyMsg)
	m = result.(Model)

	if m.editBuf != "ax = 10" {
		t.Fatalf("After typing 'a': editBuf=%q, want %q", m.editBuf, "ax = 10")
	}
	if m.mode != StateAutocomplete {
		t.Fatalf("After typing 'a': mode=%v, want StateAutocomplete", m.mode)
	}

	// 2. Simulate debounce firing (as happens in real app between key presses)
	// Extract the evalDebounceMsg from the returned command
	if cmd != nil {
		// Process the batch — find and fire the evalDebounceMsg
		m.transitionToProcessing() // This is what the debounce does
	}

	// Document should now have "ax = 10" saved
	lines := m.GetLines()
	if lines[0] != "ax = 10" {
		t.Fatalf("After debounce: line 0=%q, want %q", lines[0], "ax = 10")
	}

	// editBuf should still have the content
	if m.editBuf != "ax = 10" {
		t.Fatalf("After debounce: editBuf=%q, want %q", m.editBuf, "ax = 10")
	}

	// 3. Re-trigger autocomplete mode (debounce transitions to Ready, which
	// clears autocomplete mode; simulate user state where popup is still showing)
	m.mode = StateAutocomplete
	m.autocompleteState.Visible = true
	m.autocompleteState.Prefix = "a"

	// 4. Press backspace to delete "a" — this is the bug trigger
	bsMsg := tea.KeyMsg{Type: tea.KeyBackspace}
	result, _ = m.Update(bsMsg)
	m = result.(Model)

	// editBuf MUST be "x = 10" (the "a" deleted), NOT "ax = 10"
	if m.editBuf != "x = 10" {
		t.Errorf("After backspace: editBuf=%q, want %q", m.editBuf, "x = 10")
	}

	// Cursor must be at col 0
	if m.cursorCol != 0 {
		t.Errorf("After backspace: cursorCol=%d, want 0", m.cursorCol)
	}

	// Mode must be StateDefault (autocomplete dismissed, prefix is empty)
	// Note: "x" at col 0 is not a word prefix (cursor is before it, not after)
	if m.mode != StateDefault {
		t.Errorf("After backspace: mode=%v, want StateDefault", m.mode)
	}

	// THE CRITICAL CHECK: The view must NOT show the deleted "a".
	// In the buggy code, the view falls back to GetLines() when editBuf=="",
	// but here editBuf="x = 10" (not empty), so this particular case works.
	// The REAL problem is when the typed char is the ONLY content on the line.
}

// TestBackspaceLastCharOnEmptyLine reproduces the bug on an initially empty line.
// This is the critical case: editBuf becomes "" after backspace, and the view
// falls back to document content which has the stale character.
func TestBackspaceLastCharOnEmptyLine(t *testing.T) {
	// Start with a two-line doc, second line is just a calc
	content := "x = 10\n\n" // line 0: "x = 10", line 1: ""
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Move to the empty line
	m.cursorLine = 1
	m.cursorCol = 0
	m.editBuf = ""

	// 1. Type "a" — inserts on empty line, triggers autocomplete
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	result, _ := m.Update(keyMsg)
	m = result.(Model)

	if m.editBuf != "a" {
		t.Fatalf("After typing 'a' on empty line: editBuf=%q, want %q", m.editBuf, "a")
	}

	// 2. Simulate debounce firing — saves "a" to document
	m.transitionToProcessing()

	// Document line 1 should now be "a"
	lines := m.GetLines()
	if lines[1] != "a" {
		t.Fatalf("After debounce: line 1=%q, want %q", lines[1], "a")
	}

	// 3. Re-trigger autocomplete mode
	m.mode = StateAutocomplete
	m.autocompleteState.Visible = true
	m.autocompleteState.Prefix = "a"
	m.state = StateReady // debounce transitions through Processing → Ready

	// 4. Press backspace — deletes "a", editBuf becomes ""
	bsMsg := tea.KeyMsg{Type: tea.KeyBackspace}
	result, _ = m.Update(bsMsg)
	m = result.(Model)

	// editBuf MUST be "" (the "a" was deleted)
	if m.editBuf != "" {
		t.Errorf("After backspace on empty line: editBuf=%q, want empty", m.editBuf)
	}

	// Cursor at col 0
	if m.cursorCol != 0 {
		t.Errorf("After backspace: cursorCol=%d, want 0", m.cursorCol)
	}

	// The document still has "a" because debounce hasn't fired yet — that's expected.
	// The critical fix is that editBufLoaded=true so the view renders editBuf=""
	// instead of falling back to the stale GetLines() content.
	docLines := m.GetLines()
	if docLines[m.cursorLine] != "a" {
		t.Errorf("Expected document to still have 'a' (debounce not fired), got %q", docLines[m.cursorLine])
	}

	// editBufLoaded MUST be true so the view uses editBuf="" instead of stale doc
	if !m.editBufLoaded {
		t.Error("editBufLoaded must be true so view renders editBuf instead of stale document content")
	}
}

func TestFunctionSuggestionSource_SynonymDisplay(t *testing.T) {
	source := NewFunctionSuggestionSource()

	// When typing "mean", should show avg with synonyms
	suggestions := source.GetSuggestions("mean")

	if len(suggestions) == 0 {
		t.Fatal("expected at least one suggestion for 'mean'")
	}

	s := suggestions[0]
	if s.InsertText != "avg" {
		t.Errorf("expected InsertText=avg, got %q", s.InsertText)
	}

	// Display name should mention the matched synonym
	if s.Name != "avg (mean)" {
		t.Errorf("expected Name to show matched synonym, got %q", s.Name)
	}
}
