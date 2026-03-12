package editor

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
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
			name:         "empty prefix returns all functions plus NL rows",
			prefix:       "",
			wantMinCount: 24, // 15 builtin functions + NL example rows (parseable aliases + NLExample fallbacks)
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
	}, nil)

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
	}, nil)

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

	// 1. Type "av" — triggers autocomplete (2-char minimum prefix)
	for _, ch := range "av" {
		keyMsg := tea.KeyPressMsg{Code: ch, Text: string(ch)}
		result, _ := m.Update(keyMsg)
		m = result.(Model)
	}

	if m.editBuf != "avx = 10" {
		t.Fatalf("After typing 'av': editBuf=%q, want %q", m.editBuf, "avx = 10")
	}
	if m.mode != StateAutocomplete {
		t.Fatalf("After typing 'av': mode=%v, want StateAutocomplete", m.mode)
	}

	// 2. Simulate debounce firing (as happens in real app between key presses)
	m.transitionToProcessing() // This is what the debounce does

	// Document should now have "avx = 10" saved
	lines := m.GetLines()
	if lines[0] != "avx = 10" {
		t.Fatalf("After debounce: line 0=%q, want %q", lines[0], "avx = 10")
	}

	// editBuf should still have the content
	if m.editBuf != "avx = 10" {
		t.Fatalf("After debounce: editBuf=%q, want %q", m.editBuf, "avx = 10")
	}

	// 3. Re-trigger autocomplete mode (debounce transitions to Ready, which
	// clears autocomplete mode; simulate user state where popup is still showing)
	m.mode = StateAutocomplete
	m.autocompleteState.Visible = true
	m.autocompleteState.Prefix = "av"

	// 4. Press backspace to delete "v" — prefix becomes "a" (below 2-char minimum)
	bsMsg := tea.KeyPressMsg{Code: tea.KeyBackspace}
	result, _ := m.Update(bsMsg)
	m = result.(Model)

	// editBuf MUST be "ax = 10" (the "v" deleted)
	if m.editBuf != "ax = 10" {
		t.Errorf("After backspace: editBuf=%q, want %q", m.editBuf, "ax = 10")
	}

	// Cursor must be at col 1
	if m.cursorCol != 1 {
		t.Errorf("After backspace: cursorCol=%d, want 1", m.cursorCol)
	}

	// Mode must be StateDefault (autocomplete dismissed, prefix "a" is below 2-char minimum)
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
	keyMsg := tea.KeyPressMsg{Code: 'a', Text: "a"}
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
	bsMsg := tea.KeyPressMsg{Code: tea.KeyBackspace}
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

func TestFunctionSuggestionSource_NLRows(t *testing.T) {
	source := NewFunctionSuggestionSource()

	t.Run("avg prefix returns fn and nl rows", func(t *testing.T) {
		suggestions := source.GetSuggestions("av")
		var fnRow, nlRow *components.Suggestion
		for i := range suggestions {
			if suggestions[i].InsertText == "avg" {
				fnRow = &suggestions[i]
			}
			if suggestions[i].Category == "example" && suggestions[i].InsertText == "average of 1, 2, 3" {
				nlRow = &suggestions[i]
			}
		}
		if fnRow == nil {
			t.Error("expected fn row for avg")
		}
		if nlRow == nil {
			t.Fatal("expected nl row for avg")
		}
		if nlRow.SortCategory != "Math" {
			t.Errorf("NL row SortCategory = %q, want %q", nlRow.SortCategory, "Math")
		}
		if nlRow.Name != "average of" {
			t.Errorf("NL row Name = %q, want %q", nlRow.Name, "average of")
		}
	})

	t.Run("NL keyword prefix matches NL row", func(t *testing.T) {
		suggestions := source.GetSuggestions("aver")
		found := false
		for _, s := range suggestions {
			if s.Category == "example" && s.InsertText == "average of 1, 2, 3" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected NL row when typing NL keyword prefix 'aver'")
		}
	})

	t.Run("compound prefix returns fn and nl rows", func(t *testing.T) {
		suggestions := source.GetSuggestions("comp")
		var fnRow, basicNL, freqNL *components.Suggestion
		for i := range suggestions {
			if suggestions[i].InsertText == "compound" {
				fnRow = &suggestions[i]
			}
			if suggestions[i].Category == "example" && suggestions[i].InsertText == "compound $1000 by 5% over 10 years" {
				basicNL = &suggestions[i]
			}
			if suggestions[i].Category == "example" && suggestions[i].InsertText == "compound $1000 by 5% monthly over 10 years" {
				freqNL = &suggestions[i]
			}
		}
		if fnRow == nil {
			t.Error("expected fn row for compound")
		}
		if basicNL == nil {
			t.Fatal("expected basic NL row for compound (without modifier)")
		}
		if freqNL == nil {
			t.Fatal("expected frequency NL row for compound (with monthly)")
		}
	})

	t.Run("NL keyword only matches when fn name does not", func(t *testing.T) {
		// "squa" matches "square root of" but not "sqrt"
		suggestions := source.GetSuggestions("squa")
		var hasFn, hasNL bool
		for _, s := range suggestions {
			if s.InsertText == "sqrt" {
				hasFn = true
			}
			if s.Category == "example" && s.InsertText == "square root of 16" {
				hasNL = true
			}
		}
		if hasFn {
			t.Error("fn row for sqrt should NOT match prefix 'squa'")
		}
		if !hasNL {
			t.Error("expected NL row for 'square root of' with prefix 'squa'")
		}
	})

	t.Run("template alias cleaned for display", func(t *testing.T) {
		suggestions := source.GetSuggestions("transfer")
		for _, s := range suggestions {
			if s.Category == "example" {
				if s.Name != "transfer across" {
					t.Errorf("expected cleaned alias name %q, got %q", "transfer across", s.Name)
				}
				if s.InsertText != "transfer 1 GB across regional gigabit" {
					t.Errorf("expected example InsertText, got %q", s.InsertText)
				}
				break
			}
		}
	})

	t.Run("NL row syntax has no parentheses", func(t *testing.T) {
		suggestions := source.GetSuggestions("av")
		for _, s := range suggestions {
			if s.Category == "example" {
				if strings.Contains(s.Syntax, "(") {
					t.Errorf("NL row Syntax should not contain '(', got %q", s.Syntax)
				}
			}
		}
	})

	t.Run("functions with NLExample but no parseable alias", func(t *testing.T) {
		suggestions := source.GetSuggestions("accum")
		var nlRow *components.Suggestion
		for i := range suggestions {
			if suggestions[i].Category == "example" {
				nlRow = &suggestions[i]
				break
			}
		}
		if nlRow == nil {
			t.Fatal("expected NL row for accumulate via NLExample")
		}
		if nlRow.InsertText != "100 MB/s over 1 day" {
			t.Errorf("NL InsertText = %q, want %q", nlRow.InsertText, "100 MB/s over 1 day")
		}
	})
}

func TestCombinedSuggestionSource_NLRowOrdering(t *testing.T) {
	funcSource := NewFunctionSuggestionSource()
	unitSource := NewUnitSuggestionSource()
	varSource := NewVariableSuggestionSource(func() map[string]string {
		return nil
	}, nil)

	combined := NewCombinedSuggestionSource(funcSource, unitSource, varSource)

	// "av" should produce fn row for avg, then nl row for avg, grouped together
	suggestions := combined.GetSuggestions("av")

	fnIdx := -1
	nlIdx := -1
	for i, s := range suggestions {
		if s.InsertText == "avg" && s.Category != "example" {
			fnIdx = i
		}
		if s.Category == "example" && s.InsertText == "average of 1, 2, 3" {
			nlIdx = i
		}
	}

	if fnIdx == -1 {
		t.Fatal("expected fn row for avg")
	}
	if nlIdx == -1 {
		t.Fatal("expected nl row for avg")
	}
	if nlIdx != fnIdx+1 {
		t.Errorf("NL row at index %d should immediately follow fn row at index %d", nlIdx, fnIdx)
	}
}

func TestCleanAliasName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"transfer...across", "transfer across"},
		{"read...from", "read from"},
		{"average of", "average of"},
		{"compress...using", "compress using"},
	}
	for _, tt := range tests {
		if got := cleanAliasName(tt.input); got != tt.want {
			t.Errorf("cleanAliasName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFirstWord(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"average of", "average"},
		{"square root of", "square"},
		{"transfer...across", "transfer"},
		{"read...from", "read"},
		{"accumulate", "accumulate"},
	}
	for _, tt := range tests {
		if got := firstWord(tt.input); got != tt.want {
			t.Errorf("firstWord(%q) = %q, want %q", tt.input, got, tt.want)
		}
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

// TestPositionAwareVariableSuggestions verifies that the autocomplete popup
// only shows variables defined above the cursor line, not below it.
func TestPositionAwareVariableSuggestions(t *testing.T) {
	content := `price = 100
quantity = 5
total = price * quantity
result = total + 10`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Move cursor to line 2 ("total = price * quantity")
	m.cursorLine = 2
	m.cursorCol = 0

	// Set cursor position for variable filtering
	m.varSource.CursorLine = m.cursorLine

	// Get variable suggestions directly from the combined source
	suggestions := m.suggestionSource.GetSuggestions("pr")

	foundPrice := false
	for _, s := range suggestions {
		if s.Category == "variable" && s.InsertText == "price" {
			foundPrice = true
		}
		if s.Category == "variable" && s.InsertText == "result" {
			t.Error("should NOT suggest 'result' — it's defined below cursor")
		}
	}
	if !foundPrice {
		t.Error("should suggest 'price' — it's defined above cursor")
	}

	// Move cursor to line 0 — no user variables visible above
	m.cursorLine = 0
	m.varSource.CursorLine = m.cursorLine
	suggestions = m.suggestionSource.GetSuggestions("pr")
	for _, s := range suggestions {
		if s.Category == "variable" && s.InsertText == "price" {
			t.Error("should NOT suggest 'price' when cursor is on its definition line")
		}
	}

	// PI and E should always be available regardless of cursor position
	m.cursorLine = 0
	m.varSource.CursorLine = m.cursorLine
	suggestions = m.suggestionSource.GetSuggestions("PI")
	foundPI := false
	for _, s := range suggestions {
		if s.Category == "variable" && s.InsertText == "PI" {
			foundPI = true
		}
	}
	if !foundPI {
		t.Error("should always suggest built-in constant 'PI'")
	}
}

// TestAutocompleteSuppressesFunctionsInsideFunctionCall verifies that function
// suggestions are suppressed when typing inside a function call's arguments,
// but variable/unit suggestions still appear.
// Bug: typing "comp" inside compound($10K, 5%, 10, comp) triggers the popup
// for the "compound" function, but the user is already inside that call.
func TestAutocompleteSuppressesFunctionsInsideFunctionCall(t *testing.T) {
	content := `a = 1`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Simulate typing "compound($10K, 5%, 10, comp" — no closing paren yet
	// Cursor is at col 27, right after 'p' of the 4th arg "comp"
	m.cursorLine = 0
	m.cursorCol = 27
	m.editBuf = "compound($10K, 5%, 10, comp"
	m.editBufLoaded = true // prevent loadCurrentLineIntoEditBuffer from overwriting

	// Call updateAutocompleteState which is what triggers after each keystroke
	m.updateAutocompleteState()

	// "comp" only matches function suggestions (compound), so with function
	// filtering the popup should not appear at all
	if m.mode == StateAutocomplete {
		// Check that no function suggestions leaked through
		for _, s := range m.autocompleteState.Suggestions {
			tag := suggestionTag(s.Category)
			if tag == "fn" || tag == "nl" {
				t.Errorf("function suggestion %q should be suppressed inside function call", s.InsertText)
			}
		}
	}
}

// TestAutocompleteVariablesStillShowInsideFunctionCall verifies that variable
// suggestions still appear when typing inside a function call's arguments.
func TestAutocompleteVariablesStillShowInsideFunctionCall(t *testing.T) {
	content := `price = 100
total = avg(pr`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Simulate typing "avg(pr" — cursor inside avg() call, typing "pr"
	m.cursorLine = 1
	m.cursorCol = 14
	m.editBuf = "total = avg(pr"
	m.editBufLoaded = true

	m.updateAutocompleteState()

	if m.mode != StateAutocomplete {
		t.Error("autocomplete popup SHOULD appear for variable 'price' inside function call")
		return
	}

	// Verify only non-function suggestions are present
	for _, s := range m.autocompleteState.Suggestions {
		tag := suggestionTag(s.Category)
		if tag == "fn" || tag == "nl" {
			t.Errorf("function suggestion %q should be suppressed inside function call", s.InsertText)
		}
	}

	// Verify 'price' variable is suggested
	foundPrice := false
	for _, s := range m.autocompleteState.Suggestions {
		if s.Category == "variable" && s.InsertText == "price" {
			foundPrice = true
		}
	}
	if !foundPrice {
		t.Error("should suggest variable 'price' inside function call")
	}
}

// TestAutocompleteNotSuppressedOutsideFunctionCall verifies that the
// autocomplete popup still appears normally when NOT inside a function call.
func TestAutocompleteNotSuppressedOutsideFunctionCall(t *testing.T) {
	content := `comp`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Cursor at end of "comp" — NOT inside any function call
	m.cursorLine = 0
	m.cursorCol = 4
	m.editBuf = "comp"

	m.updateAutocompleteState()

	if m.mode != StateAutocomplete {
		t.Error("autocomplete popup SHOULD appear when typing 'comp' outside any function call")
	}

	// Should include function suggestions
	foundFn := false
	for _, s := range m.autocompleteState.Suggestions {
		if suggestionTag(s.Category) == "fn" {
			foundFn = true
			break
		}
	}
	if !foundFn {
		t.Error("function suggestions should appear outside function calls")
	}
}

// TestAutocompleteSuppressedInFrontmatter verifies that autocomplete does not
// trigger when the cursor is on a frontmatter line. Frontmatter is YAML, not
// CalcMark — autocomplete suggestions (functions, variables) are irrelevant.
func TestAutocompleteSuppressedInFrontmatter(t *testing.T) {
	source := "---\nscale: 9\nconvert_to: si\n---\n\nv = 10 lb\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)
	m.width = 80
	m.height = 24

	// Move cursor to frontmatter line 2 ("convert_to: si") and type "conv"
	m.cursorLine = 2
	m.loadCurrentLineIntoEditBuffer()
	m.cursorCol = len("conv") // simulate having typed "conv"

	// Trigger autocomplete — should NOT activate on frontmatter lines
	m.updateAutocompleteState()

	if m.mode == StateAutocomplete {
		t.Error("Autocomplete should not activate on frontmatter lines")
	}
	if m.autocompleteState.Visible {
		t.Error("Autocomplete popup should not be visible on frontmatter lines")
	}

	// Same prefix on a calc line SHOULD trigger autocomplete
	m.cursorLine = 5 // "v = 10 lb"
	m.editBuf = "conv"
	m.cursorCol = 4
	m.updateAutocompleteState()

	if m.mode != StateAutocomplete {
		t.Error("Autocomplete should activate on calc lines with matching prefix")
	}
}
