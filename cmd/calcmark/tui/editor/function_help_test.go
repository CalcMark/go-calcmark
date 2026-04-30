package editor

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/v2/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

// TestFunctionHelp_AfterAutocompleteAccept verifies that function parameter help
// is shown in the context footer after accepting a function from autocomplete.
//
// User scenario:
// 1. Type "acc" - autocomplete shows "accumulate"
// 2. Press TAB to accept
// 3. Text becomes "accumulate(" with cursor inside parentheses
// 4. Context footer SHOULD show parameter help for "rate" parameter
//
// Bug report: "I do NOT see that status message at all after TAB to accept"
func TestFunctionHelp_AfterAutocompleteAccept(t *testing.T) {
	doc, err := document.NewDocument("")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	// Step 1: Type "acc" to trigger autocomplete
	for _, r := range "acc" {
		m2, _ := m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = m2.(Model)
	}

	t.Logf("After typing 'acc':")
	t.Logf("  mode: %v", m.mode)
	t.Logf("  editBuf: %q", m.editBuf)
	t.Logf("  cursorCol: %d", m.cursorCol)
	t.Logf("  autocompleteState.Visible: %v", m.autocompleteState.Visible)
	t.Logf("  autocompleteState.Suggestions count: %d", len(m.autocompleteState.Suggestions))

	// Verify autocomplete is showing
	if !m.autocompleteState.Visible {
		t.Fatalf("Expected autocomplete to be visible after typing 'acc'")
	}

	// Find "accumulate" in suggestions
	foundAccumulate := false
	for _, s := range m.autocompleteState.Suggestions {
		if s.InsertText == "accumulate" || s.Name == "accumulate" {
			foundAccumulate = true
			break
		}
	}
	if !foundAccumulate {
		t.Logf("Suggestions: %+v", m.autocompleteState.Suggestions)
		t.Fatalf("Expected 'accumulate' in autocomplete suggestions")
	}

	// Step 2: Press TAB to accept the autocomplete suggestion
	m3, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = m3.(Model)

	t.Logf("After pressing TAB:")
	t.Logf("  mode: %v", m.mode)
	t.Logf("  editBuf: %q", m.editBuf)
	t.Logf("  cursorCol: %d", m.cursorCol)
	t.Logf("  autocompleteState.Visible: %v", m.autocompleteState.Visible)

	// Verify the text was inserted correctly
	expectedText := "accumulate("
	if m.editBuf != expectedText {
		t.Errorf("Expected editBuf=%q after TAB, got %q", expectedText, m.editBuf)
	}

	// Verify cursor is inside the parentheses
	expectedCursorCol := len("accumulate(")
	if m.cursorCol != expectedCursorCol {
		t.Errorf("Expected cursorCol=%d after TAB, got %d", expectedCursorCol, m.cursorCol)
	}

	// Verify autocomplete is now hidden
	if m.autocompleteState.Visible {
		t.Errorf("Expected autocomplete to be hidden after TAB")
	}

	// Step 3: Check that GetCursorContext detects we're inside a function call
	cursorCtx := GetCursorContext(m.editBuf, m.cursorCol)

	t.Logf("CursorContext:")
	t.Logf("  InFunctionCall: %v", cursorCtx.InFunctionCall)
	t.Logf("  FunctionName: %q", cursorCtx.FunctionName)
	t.Logf("  ArgIndex: %d", cursorCtx.ArgIndex)
	if cursorCtx.ParamSpec != nil {
		t.Logf("  ParamSpec.Name: %q", cursorCtx.ParamSpec.Name)
	} else {
		t.Logf("  ParamSpec: nil")
	}

	if !cursorCtx.InFunctionCall {
		t.Errorf("Expected InFunctionCall=true for editBuf=%q, cursorCol=%d",
			m.editBuf, m.cursorCol)
	}

	if cursorCtx.FunctionName != "accumulate" {
		t.Errorf("Expected FunctionName='accumulate', got %q", cursorCtx.FunctionName)
	}

	if cursorCtx.ParamSpec == nil {
		t.Errorf("Expected ParamSpec to be non-nil for first parameter")
	} else if cursorCtx.ParamSpec.Name != "rate" {
		t.Errorf("Expected first param name='rate', got %q", cursorCtx.ParamSpec.Name)
	}

	// Step 4: THE REAL TEST - Check the actual View() output contains function help
	// This is what the user actually sees!
	view := m.View().Content

	t.Logf("View output length: %d bytes", len(view))

	// The context footer should contain function parameter help
	// Look for indicators that function help is being shown:
	// - The function name "accumulate"
	// - The parameter name "rate"
	// - Parameter examples like "10 MB/s"

	if !strings.Contains(view, "rate") {
		t.Errorf("FAIL: View() output does not contain 'rate' - function parameter help is NOT being rendered!")
		t.Logf("View output:\n%s", view)
	}

	if !strings.Contains(view, "accumulate") {
		t.Errorf("FAIL: View() output does not contain 'accumulate'")
	}

	// Check for parameter examples
	if !strings.Contains(view, "MB/s") && !strings.Contains(view, "requests") {
		t.Errorf("FAIL: View() output does not contain parameter examples like 'MB/s' or 'requests'")
		t.Logf("View output:\n%s", view)
	}
}

// TestFunctionHelp_ViewRendering tests that the View() method actually renders
// function parameter help in the context footer.
func TestFunctionHelp_ViewRendering(t *testing.T) {
	doc, err := document.NewDocument("")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	// Directly set up the state as if user just accepted "accumulate(" from autocomplete
	m.editBuf = "accumulate("
	m.editBufLoaded = true
	m.cursorCol = len("accumulate(")
	m.mode = StateDefault

	t.Logf("Setup:")
	t.Logf("  editBuf: %q", m.editBuf)
	t.Logf("  cursorCol: %d", m.cursorCol)
	t.Logf("  mode: %v", m.mode)

	// Get the actual rendered view
	view := m.View().Content

	t.Logf("View length: %d bytes", len(view))

	// Check what renderContextFooter would produce
	// First, let's trace through the logic manually
	results := m.GetLineResults()
	t.Logf("GetLineResults count: %d", len(results))
	if len(results) > 0 {
		t.Logf("  results[0].IsCalc: %v", results[0].IsCalc)
		t.Logf("  results[0].Error: %q", results[0].Error)
	}

	// Check cursor context
	ctx := GetCursorContext(m.editBuf, m.cursorCol)
	t.Logf("GetCursorContext:")
	t.Logf("  InFunctionCall: %v", ctx.InFunctionCall)
	t.Logf("  FunctionName: %q", ctx.FunctionName)
	if ctx.ParamSpec != nil {
		t.Logf("  ParamSpec.Name: %q", ctx.ParamSpec.Name)
	}

	// The view MUST contain function help
	if !strings.Contains(view, "rate") {
		t.Errorf("FAIL: View does NOT contain 'rate' parameter name")
		// Print a portion of the view to debug
		lines := strings.Split(view, "\n")
		t.Logf("Last 10 lines of view:")
		start := max(len(lines)-10, 0)
		for i := start; i < len(lines); i++ {
			t.Logf("  [%d] %q", i, lines[i])
		}
	}

	if !strings.Contains(view, "MB/s") {
		t.Errorf("FAIL: View does NOT contain 'MB/s' parameter example")
	}
}

// TestFunctionHelp_ContextFooterState tests the ContextFooterState that gets
// passed to RenderContextFooter.
func TestFunctionHelp_ContextFooterState(t *testing.T) {
	doc, err := document.NewDocument("")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	// Set up state
	m.editBuf = "accumulate("
	m.editBufLoaded = true
	m.cursorCol = len("accumulate(")
	m.mode = StateDefault

	// Now trace through exactly what renderContextFooter does
	// This is copied from view.go to understand the actual flow

	results := m.GetLineResults()
	state := components.ContextFooterState{}

	t.Logf("Building ContextFooterState:")

	// Check bounds
	if m.cursorLine < len(results) {
		currentResult := results[m.cursorLine]
		state.IsCalcLine = currentResult.IsCalc
		t.Logf("  IsCalcLine: %v", state.IsCalcLine)

		if currentResult.IsCalc && currentResult.Error != "" {
			state.HasError = true
			t.Logf("  HasError: %v (error: %q)", state.HasError, currentResult.Error)
		}
	} else {
		t.Logf("  cursorLine (%d) >= len(results) (%d)", m.cursorLine, len(results))
	}

	// Check autocomplete state
	if m.mode == StateAutocomplete && m.autocompleteState.Visible {
		state.AutocompleteActive = true
		t.Logf("  AutocompleteActive: true")
	} else {
		t.Logf("  AutocompleteActive: false (mode=%v, visible=%v)", m.mode, m.autocompleteState.Visible)
	}

	// Check for function argument context
	// NOTE: This is the key part - we should enter this block
	if !state.AutocompleteActive {
		t.Logf("  Checking function argument context...")
		m.loadCurrentLineIntoEditBuffer()
		t.Logf("    editBuf after load: %q", m.editBuf)
		t.Logf("    cursorCol: %d", m.cursorCol)

		cursorCtx := GetCursorContext(m.editBuf, m.cursorCol)
		t.Logf("    InFunctionCall: %v", cursorCtx.InFunctionCall)
		t.Logf("    FunctionName: %q", cursorCtx.FunctionName)
		if cursorCtx.ParamSpec != nil {
			t.Logf("    ParamSpec.Name: %q", cursorCtx.ParamSpec.Name)
		} else {
			t.Logf("    ParamSpec: nil")
		}

		if cursorCtx.InFunctionCall && cursorCtx.ParamSpec != nil {
			state.InFunctionCall = true
			state.FunctionName = cursorCtx.FunctionName
			state.ParamName = cursorCtx.ParamSpec.Name
			state.ParamExamples = FormatParamHelp(cursorCtx.ParamSpec)
			state.ArgIndex = cursorCtx.ArgIndex
			t.Logf("  SET: InFunctionCall=true, FunctionName=%q, ParamName=%q",
				state.FunctionName, state.ParamName)
		} else {
			t.Logf("  NOT setting function context (InFunctionCall=%v, ParamSpec=%v)",
				cursorCtx.InFunctionCall, cursorCtx.ParamSpec != nil)
		}
	}

	t.Logf("Final state:")
	t.Logf("  InFunctionCall: %v", state.InFunctionCall)
	t.Logf("  FunctionName: %q", state.FunctionName)
	t.Logf("  ParamName: %q", state.ParamName)
	t.Logf("  ParamExamples: %q", state.ParamExamples)

	// Assertions
	if !state.InFunctionCall {
		t.Errorf("FAIL: state.InFunctionCall should be true")
	}
	if state.ParamName != "rate" {
		t.Errorf("FAIL: state.ParamName should be 'rate', got %q", state.ParamName)
	}
}

// TestFunctionHelp_WhileTypingArgs verifies that function parameter help
// continues to show while typing arguments.
func TestFunctionHelp_WhileTypingArgs(t *testing.T) {
	doc, err := document.NewDocument("")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	// Simulate having already typed "accumulate(10 M" - in the middle of first arg
	m.editBuf = "accumulate(10 M"
	m.cursorCol = len(m.editBuf)

	t.Logf("Setup: editBuf=%q, cursorCol=%d", m.editBuf, m.cursorCol)

	// Check cursor context
	cursorCtx := GetCursorContext(m.editBuf, m.cursorCol)

	t.Logf("CursorContext:")
	t.Logf("  InFunctionCall: %v", cursorCtx.InFunctionCall)
	t.Logf("  FunctionName: %q", cursorCtx.FunctionName)
	t.Logf("  ArgIndex: %d", cursorCtx.ArgIndex)
	if cursorCtx.ParamSpec != nil {
		t.Logf("  ParamSpec.Name: %q", cursorCtx.ParamSpec.Name)
	}

	if !cursorCtx.InFunctionCall {
		t.Errorf("Expected InFunctionCall=true")
	}
	if cursorCtx.FunctionName != "accumulate" {
		t.Errorf("Expected FunctionName='accumulate', got %q", cursorCtx.FunctionName)
	}
	if cursorCtx.ArgIndex != 0 {
		t.Errorf("Expected ArgIndex=0 (first arg), got %d", cursorCtx.ArgIndex)
	}
	if cursorCtx.ParamSpec == nil || cursorCtx.ParamSpec.Name != "rate" {
		t.Errorf("Expected ParamSpec for 'rate' parameter")
	}
}

// TestFunctionHelp_SecondArg verifies parameter help shows for second argument.
func TestFunctionHelp_SecondArg(t *testing.T) {
	doc, err := document.NewDocument("")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	// Simulate having typed "accumulate(10 MB/s, " - starting second arg
	m.editBuf = "accumulate(10 MB/s, "
	m.cursorCol = len(m.editBuf)

	t.Logf("Setup: editBuf=%q, cursorCol=%d", m.editBuf, m.cursorCol)

	cursorCtx := GetCursorContext(m.editBuf, m.cursorCol)

	t.Logf("CursorContext:")
	t.Logf("  InFunctionCall: %v", cursorCtx.InFunctionCall)
	t.Logf("  FunctionName: %q", cursorCtx.FunctionName)
	t.Logf("  ArgIndex: %d", cursorCtx.ArgIndex)
	if cursorCtx.ParamSpec != nil {
		t.Logf("  ParamSpec.Name: %q", cursorCtx.ParamSpec.Name)
	}

	if !cursorCtx.InFunctionCall {
		t.Errorf("Expected InFunctionCall=true")
	}
	if cursorCtx.ArgIndex != 1 {
		t.Errorf("Expected ArgIndex=1 (second arg), got %d", cursorCtx.ArgIndex)
	}
	if cursorCtx.ParamSpec == nil || cursorCtx.ParamSpec.Name != "duration" {
		paramName := ""
		if cursorCtx.ParamSpec != nil {
			paramName = cursorCtx.ParamSpec.Name
		}
		t.Errorf("Expected ParamSpec for 'duration' parameter, got %q", paramName)
	}
}

// TestFunctionHelp_AfterClosingParen verifies help is NOT shown after function is complete.
func TestFunctionHelp_AfterClosingParen(t *testing.T) {
	doc, err := document.NewDocument("")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	// Simulate completed function call "accumulate(10 MB/s, 1 hour)"
	m.editBuf = "accumulate(10 MB/s, 1 hour)"
	m.cursorCol = len(m.editBuf) // cursor after closing paren

	t.Logf("Setup: editBuf=%q, cursorCol=%d", m.editBuf, m.cursorCol)

	cursorCtx := GetCursorContext(m.editBuf, m.cursorCol)

	t.Logf("CursorContext:")
	t.Logf("  InFunctionCall: %v", cursorCtx.InFunctionCall)

	// After the closing paren, we should NOT be inside the function call
	if cursorCtx.InFunctionCall {
		t.Errorf("Expected InFunctionCall=false after closing paren")
	}
}
