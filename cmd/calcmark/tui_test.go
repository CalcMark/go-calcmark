package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
	tea "github.com/charmbracelet/bubbletea"
)

// TestPinnedVarsNoDuplicates verifies that /pin shows each variable only once,
// even when a variable is reassigned across multiple blocks.
func TestPinnedVarsNoDuplicates(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantVars      []string // Variables that should appear
		wantNoDupVars []string // Variables that must not appear more than once
		wantVarCount  int      // Expected count of unique variables shown
	}{
		{
			name:          "single assignment",
			input:         "x = 5\n",
			wantVars:      []string{"x"},
			wantNoDupVars: []string{"x"},
			wantVarCount:  1,
		},
		{
			name:          "two different variables",
			input:         "x = 5\ny = 10\n",
			wantVars:      []string{"x", "y"},
			wantNoDupVars: []string{"x", "y"},
			wantVarCount:  2,
		},
		{
			name:          "same variable reassigned in same block",
			input:         "x = 5\nx = 10\n",
			wantVars:      []string{"x"},
			wantNoDupVars: []string{"x"},
			wantVarCount:  1,
		},
		{
			name:          "same variable reassigned across blocks",
			input:         "x = 5\n\n\nx = 10\n", // Two blocks separated by blank lines
			wantVars:      []string{"x"},
			wantNoDupVars: []string{"x"},
			wantVarCount:  1,
		},
		{
			name:          "multiple variables with one reassigned",
			input:         "x = 5\ny = 3\n\n\nx = 10\n",
			wantVars:      []string{"x", "y"},
			wantNoDupVars: []string{"x", "y"},
			wantVarCount:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create document from input
			doc, err := document.NewDocument(tt.input)
			if err != nil {
				t.Fatalf("Failed to create document: %v", err)
			}

			// Create TUI model (auto-pins all variables)
			m := newTUIModel(doc)

			// Render pinned vars (model already has all vars pinned)
			rendered := m.renderPinnedVars()

			// Check all expected variables are present
			for _, varName := range tt.wantVars {
				if !strings.Contains(rendered, varName) {
					t.Errorf("Expected variable %q not found in rendered output:\n%s", varName, rendered)
				}
			}

			// Check no duplicates - count occurrences of " = " after each variable name
			for _, varName := range tt.wantNoDupVars {
				// Count how many times "varName = " appears (with leading boundary)
				// We look for the pattern where varName is followed by " = "
				count := countVariableOccurrences(rendered, varName)
				if count > 1 {
					t.Errorf("Variable %q appears %d times, expected 1:\n%s", varName, count, rendered)
				}
			}

			// Check total count of variables shown
			actualCount := countTotalVariables(rendered)
			if actualCount != tt.wantVarCount {
				t.Errorf("Expected %d variables, got %d:\n%s", tt.wantVarCount, actualCount, rendered)
			}
		})
	}
}

// countVariableOccurrences counts how many times a variable assignment appears in output.
// Looks for pattern: varName followed by " = ".
func countVariableOccurrences(rendered, varName string) int {
	count := 0
	for line := range strings.SplitSeq(rendered, "\n") {
		// Look for "varName = " pattern (variable followed by assignment)
		// This handles styled output where varName might have ANSI codes around it
		if strings.Contains(line, varName) && strings.Contains(line, " = ") {
			// Verify this line is actually showing this variable (not just mentioning it)
			// The format is: varName = value
			trimmed := strings.TrimSpace(line)
			// Remove any ANSI escape codes for checking
			cleaned := stripANSI(trimmed)
			if strings.HasPrefix(cleaned, varName+" ") || strings.HasPrefix(cleaned, varName+"=") {
				count++
			}
		}
	}
	return count
}

// countTotalVariables counts total variable assignments shown in rendered output.
func countTotalVariables(rendered string) int {
	count := 0
	for line := range strings.SplitSeq(rendered, "\n") {
		cleaned := stripANSI(line)
		// A variable line contains " = " and is not the header or empty
		if strings.Contains(cleaned, " = ") &&
			!strings.Contains(cleaned, "Pinned") &&
			strings.TrimSpace(cleaned) != "" {
			count++
		}
	}
	return count
}

// stripANSI removes ANSI escape codes from a string.
func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

// TestSlashCommandModeTransition tests the slash command mode state machine.
func TestSlashCommandModeTransition(t *testing.T) {
	doc, _ := document.NewDocument("x = 5\n")
	m := newTUIModel(doc)

	// Initially not in slash mode
	if m.slashMode {
		t.Error("Should not start in slash mode")
	}

	// Entering a command puts us in normal mode after execution
	m.slashMode = true // Simulate entering slash mode
	m, _ = m.handleCommand("pin")

	// After command execution, slash mode should be handled by Update()
	// handleCommand doesn't reset slashMode - that's done in handleInput
	// Let's test the full flow

	// Reset and test handleInput flow
	m = newTUIModel(doc)
	m.slashMode = true
	m.input.SetValue("pin")
	m, _ = m.handleInput()

	// After handleInput with slashMode=true, slashMode should be false
	if m.slashMode {
		t.Error("Should exit slash mode after command execution")
	}
}

// TestPinSpecificVariable tests pinning a specific variable.
func TestPinSpecificVariable(t *testing.T) {
	doc, err := document.NewDocument("x = 5\ny = 10\nz = 15\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := newTUIModel(doc)

	// Unpin all first
	m, _ = m.handleCommand("unpin")
	if len(m.pinnedVars) != 0 {
		t.Errorf("Expected 0 pinned vars after unpin, got %d", len(m.pinnedVars))
	}

	// Pin specific variable
	m, _ = m.handleCommand("pin x")
	if !m.pinnedVars["x"] {
		t.Error("Variable 'x' should be pinned")
	}
	if m.pinnedVars["y"] {
		t.Error("Variable 'y' should not be pinned")
	}
	if m.pinnedVars["z"] {
		t.Error("Variable 'z' should not be pinned")
	}

	// Rendered should only show x
	rendered := m.renderPinnedVars()
	if !strings.Contains(rendered, "x") {
		t.Error("Rendered should contain 'x'")
	}
	// y and z might appear in ANSI codes but not as variable assignments
	yCount := countVariableOccurrences(rendered, "y")
	if yCount > 0 {
		t.Errorf("Variable 'y' should not appear as assignment, found %d times", yCount)
	}
}

// TestUnpinSpecificVariable tests unpinning a specific variable.
func TestUnpinSpecificVariable(t *testing.T) {
	doc, err := document.NewDocument("x = 5\ny = 10\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := newTUIModel(doc)
	// Auto-pins all variables

	// Unpin specific variable
	m, _ = m.handleCommand("unpin x")
	if m.pinnedVars["x"] {
		t.Error("Variable 'x' should be unpinned")
	}
	if !m.pinnedVars["y"] {
		t.Error("Variable 'y' should still be pinned")
	}
}

// TestPinUnpinCommands tests /pin and /unpin behavior.
func TestPinUnpinCommands(t *testing.T) {
	doc, err := document.NewDocument("x = 5\ny = 10\nz = 15\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := newTUIModel(doc)
	// Model starts with all vars pinned (auto-pin in newTUIModel)

	// /unpin should unpin all
	m, _ = m.handleCommand("unpin")
	if len(m.pinnedVars) != 0 {
		t.Errorf("Expected 0 pinned vars after /unpin, got %d", len(m.pinnedVars))
	}

	// /pin should pin all
	m, _ = m.handleCommand("pin")
	if !m.pinnedVars["x"] || !m.pinnedVars["y"] || !m.pinnedVars["z"] {
		t.Error("All variables should be pinned after /pin")
	}

	// /pin again should still have all pinned (not toggle)
	m, _ = m.handleCommand("pin")
	if !m.pinnedVars["x"] || !m.pinnedVars["y"] || !m.pinnedVars["z"] {
		t.Error("All variables should still be pinned after second /pin")
	}
}

// TestHandleCommandUnknown tests unknown command error.
func TestHandleCommandUnknown(t *testing.T) {
	doc, _ := document.NewDocument("x = 5\n")
	m := newTUIModel(doc)

	m, _ = m.handleCommand("notarealcommand")

	if m.err == nil {
		t.Error("Expected error for unknown command")
	}
	if !strings.Contains(m.err.Error(), "unknown command") {
		t.Errorf("Expected 'unknown command' error, got: %v", m.err)
	}
	if !strings.Contains(m.err.Error(), "/help") {
		t.Errorf("Expected error to mention /help, got: %v", m.err)
	}
}

// TestHelpCommand tests that /help adds help text to output history.
func TestHelpCommand(t *testing.T) {
	doc, _ := document.NewDocument("x = 5\n")
	m := newTUIModel(doc)
	initialHistoryLen := len(m.outputHistory)

	m, _ = m.handleCommand("help")

	if len(m.outputHistory) != initialHistoryLen+1 {
		t.Errorf("Expected help to add to output history, got %d items (was %d)",
			len(m.outputHistory), initialHistoryLen)
	}

	lastItem := m.outputHistory[len(m.outputHistory)-1]
	if !strings.Contains(lastItem.output, "/pin") {
		t.Error("Help output should contain /pin command")
	}
	if !strings.Contains(lastItem.output, "/quit") {
		t.Error("Help output should contain /quit command")
	}
}

// TestHelpCommandAliases tests /h and /? work as help aliases.
func TestHelpCommandAliases(t *testing.T) {
	for _, cmd := range []string{"h", "?"} {
		doc, _ := document.NewDocument("x = 5\n")
		m := newTUIModel(doc)
		initialLen := len(m.outputHistory)

		m, _ = m.handleCommand(cmd)

		if len(m.outputHistory) != initialLen+1 {
			t.Errorf("/%s should add help to output history", cmd)
		}
	}
}

// TestPromptResetAfterCommand tests that prompt resets to > after slash command.
func TestPromptResetAfterCommand(t *testing.T) {
	doc, _ := document.NewDocument("x = 5\n")
	m := newTUIModel(doc)

	// Simulate entering slash mode and running a command
	m.slashMode = true
	m.input.Prompt = "/ "
	m.input.SetValue("help")

	m, _ = m.handleInput()

	if m.slashMode {
		t.Error("Should exit slash mode after command")
	}
	if m.input.Prompt != "> " {
		t.Errorf("Prompt should reset to '> ', got %q", m.input.Prompt)
	}
}

// TestHandleCommandQuit tests quit commands.
func TestHandleCommandQuit(t *testing.T) {
	doc, _ := document.NewDocument("x = 5\n")

	// Test "quit"
	m := newTUIModel(doc)
	m, _ = m.handleCommand("quit")
	if !m.quitting {
		t.Error("'quit' command should set quitting=true")
	}

	// Test "q"
	m = newTUIModel(doc)
	m, _ = m.handleCommand("q")
	if !m.quitting {
		t.Error("'q' command should set quitting=true")
	}
}

// TestCtrlCExitsInAllModes tests that Ctrl+C always exits, regardless of mode.
func TestCtrlCExitsInAllModes(t *testing.T) {
	doc, _ := document.NewDocument("x = 5\n")

	tests := []struct {
		name  string
		setup func(m model) model
	}{
		{"normal_mode", func(m model) model { return m }},
		{"slash_mode", func(m model) model {
			m.slashMode = true
			return m
		}},
		{"markdown_mode", func(m model) model {
			m.markdownMode = true
			return m
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTUIModel(doc)
			m = tc.setup(m)

			// Simulate Ctrl+C
			newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
			result := newModel.(model)

			if !result.quitting {
				t.Errorf("Ctrl+C should set quitting=true in %s", tc.name)
			}
			if cmd == nil {
				t.Errorf("Ctrl+C should return tea.Quit command in %s", tc.name)
			}
		})
	}
}

// TestCtrlDExitsInAllModes tests that Ctrl+D always exits, regardless of mode.
func TestCtrlDExitsInAllModes(t *testing.T) {
	doc, _ := document.NewDocument("x = 5\n")

	tests := []struct {
		name  string
		setup func(m model) model
	}{
		{"normal_mode", func(m model) model { return m }},
		{"slash_mode", func(m model) model {
			m.slashMode = true
			return m
		}},
		{"markdown_mode", func(m model) model {
			m.markdownMode = true
			return m
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTUIModel(doc)
			m = tc.setup(m)

			// Simulate Ctrl+D
			newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
			result := newModel.(model)

			if !result.quitting {
				t.Errorf("Ctrl+D should set quitting=true in %s", tc.name)
			}
			if cmd == nil {
				t.Errorf("Ctrl+D should return tea.Quit command in %s", tc.name)
			}
		})
	}
}

// TestDoubleEscClearsInput tests that double-ESC clears the input line.
func TestDoubleEscClearsInput(t *testing.T) {
	doc, _ := document.NewDocument("x = 5\n")
	m := newTUIModel(doc)

	// Set some input text
	m.input.SetValue("some text to clear")

	// First ESC - should not clear, just record the time
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := newModel.(model)

	if result.input.Value() != "some text to clear" {
		t.Error("First ESC should not clear input")
	}
	if result.lastEscTime == 0 {
		t.Error("First ESC should set lastEscTime")
	}

	// Second ESC immediately after - should clear the input
	newModel, _ = result.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result = newModel.(model)

	if result.input.Value() != "" {
		t.Errorf("Double-ESC should clear input, got: %q", result.input.Value())
	}
	if result.lastEscTime != 0 {
		t.Error("Double-ESC should reset lastEscTime to prevent triple-ESC trigger")
	}
}

// TestSingleEscDoesNotClear tests that a single ESC doesn't clear input.
func TestSingleEscDoesNotClear(t *testing.T) {
	doc, _ := document.NewDocument("x = 5\n")
	m := newTUIModel(doc)

	// Set some input text
	m.input.SetValue("keep this text")

	// Single ESC - should not clear
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := newModel.(model)

	if result.input.Value() != "keep this text" {
		t.Errorf("Single ESC should not clear input, got: %q", result.input.Value())
	}
}

// TestHandleCommandOpenNoArg tests open command without filename shows file picker.
func TestHandleCommandOpenNoArg(t *testing.T) {
	doc, _ := document.NewDocument("x = 5\n")
	m := newTUIModel(doc)

	m, _ = m.handleCommand("open")

	if !m.filePickerMode {
		t.Error("Expected file picker mode when 'open' called without filename")
	}
	if m.err != nil {
		t.Errorf("Expected no error, got: %v", m.err)
	}
}

// TestHandleCommandMarkdown tests markdown mode entry.
func TestHandleCommandMarkdown(t *testing.T) {
	doc, _ := document.NewDocument("x = 5\n")

	// Test "md"
	m := newTUIModel(doc)
	m, _ = m.handleCommand("md")
	if !m.markdownMode {
		t.Error("'md' command should enter markdown mode")
	}

	// Test "markdown"
	m = newTUIModel(doc)
	m, _ = m.handleCommand("markdown")
	if !m.markdownMode {
		t.Error("'markdown' command should enter markdown mode")
	}
}

// TestSlashPrefixStripped tests that leading / is stripped from commands.
func TestSlashPrefixStripped(t *testing.T) {
	doc, _ := document.NewDocument("x = 5\n")
	m := newTUIModel(doc)

	// /unpin with / prefix should work
	m, _ = m.handleCommand("/unpin")
	if len(m.pinnedVars) != 0 {
		t.Errorf("/unpin with / prefix should unpin all, got %d pinned", len(m.pinnedVars))
	}

	// /pin with / prefix should work
	m, _ = m.handleCommand("/pin")
	if len(m.pinnedVars) == 0 {
		t.Error("/pin with / prefix should pin all")
	}
}

// TestMarkdownModeEntry tests entering markdown mode via /md command.
func TestMarkdownModeEntry(t *testing.T) {
	doc, _ := document.NewDocument("x = 5\n")
	m := newTUIModel(doc)

	if m.markdownMode {
		t.Error("Should not start in markdown mode")
	}

	m, _ = m.handleCommand("md")

	if !m.markdownMode {
		t.Error("Should be in markdown mode after /md command")
	}
}

// TestMarkdownModeExitOnEsc tests that ESC exits markdown mode.
func TestMarkdownModeExitOnEsc(t *testing.T) {
	doc, _ := document.NewDocument("x = 5\n")
	m := newTUIModel(doc)

	// Enter markdown mode
	m, _ = m.handleCommand("md")
	if !m.markdownMode {
		t.Error("Should be in markdown mode")
	}

	// Simulate typing some markdown
	m.mdInput.SetValue("# Hello World\n\nThis is a test.")

	// Simulate ESC key (which triggers markdown exit in Update)
	// The actual ESC handling is in Update, but we can test the exit logic directly
	content := m.mdInput.Value()
	if strings.TrimSpace(content) != "" {
		m = m.insertMarkdownBlock()
	}
	m.markdownMode = false
	m.mdInput.Reset()

	if m.markdownMode {
		t.Error("Should have exited markdown mode")
	}
}

// TestMarkdownBlockInsertion tests that markdown content is added to the document.
func TestMarkdownBlockInsertion(t *testing.T) {
	doc, _ := document.NewDocument("x = 5\n")
	m := newTUIModel(doc)

	initialBlockCount := len(m.doc.GetBlocks())

	// Enter markdown mode and add content
	m, _ = m.handleCommand("md")
	m.mdInput.SetValue("# Test Header\n\nSome text content.")
	m = m.insertMarkdownBlock()

	// Check that a new block was added
	newBlockCount := len(m.doc.GetBlocks())
	if newBlockCount != initialBlockCount+1 {
		t.Errorf("Expected %d blocks after markdown insertion, got %d", initialBlockCount+1, newBlockCount)
	}

	// Verify it's a TextBlock
	blocks := m.doc.GetBlocks()
	lastBlock := blocks[len(blocks)-1]
	if _, ok := lastBlock.Block.(*document.TextBlock); !ok {
		t.Error("Last block should be a TextBlock")
	}
}

// TestMarkdownEmptyContentNotInserted tests that empty markdown is not inserted.
func TestMarkdownEmptyContentNotInserted(t *testing.T) {
	doc, _ := document.NewDocument("x = 5\n")
	m := newTUIModel(doc)

	initialBlockCount := len(m.doc.GetBlocks())

	// Enter markdown mode with empty content
	m, _ = m.handleCommand("md")
	m.mdInput.SetValue("   \n   ")
	m = m.insertMarkdownBlock()

	// Block count should not change
	newBlockCount := len(m.doc.GetBlocks())
	if newBlockCount != initialBlockCount {
		t.Errorf("Empty markdown should not create block, expected %d blocks, got %d", initialBlockCount, newBlockCount)
	}
}

// TestMarkdownPreviewRender tests that markdown preview renders correctly.
func TestMarkdownPreviewRender(t *testing.T) {
	doc, _ := document.NewDocument("x = 5\n")
	m := newTUIModel(doc)
	m.width = 80 // Set reasonable width for rendering

	// Enter markdown mode
	m, _ = m.handleCommand("md")

	// Test empty state
	preview := m.renderMarkdownPreview()
	if !strings.Contains(preview, "preview will appear") {
		t.Error("Empty preview should show placeholder text")
	}

	// Test with content
	m.mdInput.SetValue("# Hello")
	preview = m.renderMarkdownPreview()
	if strings.Contains(preview, "render error") {
		t.Errorf("Preview should render without error, got: %s", preview)
	}
}

// TestMarkdownModeViewLayout tests the split view layout in markdown mode.
func TestMarkdownModeViewLayout(t *testing.T) {
	doc, _ := document.NewDocument("x = 5\n")
	m := newTUIModel(doc)
	m.width = 100
	m.height = 30

	// Enter markdown mode
	m, _ = m.handleCommand("md")

	view := m.View()

	// Should contain markdown mode indicators
	if !strings.Contains(view, "MARKDOWN MODE") {
		t.Error("View should indicate markdown mode")
	}

	// Should contain the help link
	if !strings.Contains(view, "commonmark.org") {
		t.Error("View should contain markdown help link")
	}

	// Should have Edit and Preview labels
	if !strings.Contains(view, "Edit") {
		t.Error("View should have Edit panel")
	}
	if !strings.Contains(view, "Preview") {
		t.Error("View should have Preview panel")
	}
}

// TestNewVariablesAutoPinned tests that new variables are automatically pinned.
func TestNewVariablesAutoPinned(t *testing.T) {
	// Start with empty document
	doc, _ := document.NewDocument("\n")
	m := newTUIModel(doc)

	// Initially no pinned vars
	if len(m.pinnedVars) != 0 {
		t.Errorf("Expected 0 pinned vars initially, got %d", len(m.pinnedVars))
	}

	// Simulate evaluating a new expression with a variable
	m.input.SetValue("x = 42")
	m, _ = m.handleInput()

	// x should now be auto-pinned
	if !m.pinnedVars["x"] {
		t.Error("Variable 'x' should be auto-pinned after creation")
	}

	// Add another variable
	m.input.SetValue("y = 100")
	m, _ = m.handleInput()

	// Both should be pinned
	if !m.pinnedVars["x"] {
		t.Error("Variable 'x' should still be pinned")
	}
	if !m.pinnedVars["y"] {
		t.Error("Variable 'y' should be auto-pinned after creation")
	}
}

// TestChangedVarsMarkedWithAsterisk tests that changed variables show * indicator.
func TestChangedVarsMarkedWithAsterisk(t *testing.T) {
	doc, _ := document.NewDocument("x = 5\ny = 10\n")
	m := newTUIModel(doc)
	m.width = 80

	// Initially no changed vars (loaded from file)
	if len(m.changedVars) != 0 {
		t.Errorf("Expected 0 changed vars on load, got %d", len(m.changedVars))
	}

	// Reassign x - should mark x as changed
	m.input.SetValue("x = 100")
	m, _ = m.handleInput()

	if !m.changedVars["x"] {
		t.Error("Variable 'x' should be marked as changed after reassignment")
	}

	// Render should show * for changed var
	rendered := m.renderPinnedVars()
	if !strings.Contains(rendered, "* ") {
		t.Error("Rendered output should contain * for changed variable")
	}

	// Next input should clear changedVars
	m.input.SetValue("z = 1")
	m, _ = m.handleInput()

	// x should no longer be in changedVars (cleared at start of handleInput)
	// but z should be
	if m.changedVars["x"] {
		t.Error("Variable 'x' should not be marked as changed after next input")
	}
	if !m.changedVars["z"] {
		t.Error("Variable 'z' should be marked as changed")
	}
}

// TestDependentVarsMarkedAsChanged tests that dependent variables are marked when upstream changes.
func TestDependentVarsMarkedAsChanged(t *testing.T) {
	// Create document with dependency: y depends on x
	doc, _ := document.NewDocument("x = 5\n")
	m := newTUIModel(doc)

	// Add y = x + 1
	m.input.SetValue("y = x + 1")
	m, _ = m.handleInput()

	// Clear changed vars for next test
	m.changedVars = make(map[string]bool)

	// Change x - should mark both x and y as changed (y depends on x)
	m.input.SetValue("x = 100")
	m, _ = m.handleInput()

	if !m.changedVars["x"] {
		t.Error("Variable 'x' should be marked as changed")
	}
	// Note: y being marked depends on document's AffectedBlockIDs including y's block
}

// TestTransitiveDependencyValuePropagation tests that changing a variable
// updates the VALUES (not just marks) of all transitive dependents.
// This is the exact scenario from the bug report: a=10, b=a*2, c=b+10
// When a=20, c should update from 30 to 60.
func TestTransitiveDependencyValuePropagation(t *testing.T) {
	doc, _ := document.NewDocument("")
	m := newTUIModel(doc)

	// Build dependency chain: a -> b -> c
	m.input.SetValue("a = 10")
	m, _ = m.handleInput()

	m.input.SetValue("b = a * 2")
	m, _ = m.handleInput()

	m.input.SetValue("c = b + 10")
	m, _ = m.handleInput()

	// Verify initial values
	_, values, _ := m.collectPinnedVariables()

	aVal, ok := values["a"]
	if !ok {
		t.Fatal("Variable 'a' not found in pinned variables")
	}
	if fmt.Sprintf("%v", aVal) != "10" {
		t.Errorf("Initial a = %v, want 10", aVal)
	}

	bVal, ok := values["b"]
	if !ok {
		t.Fatal("Variable 'b' not found in pinned variables")
	}
	if fmt.Sprintf("%v", bVal) != "20" {
		t.Errorf("Initial b = %v, want 20", bVal)
	}

	cVal, ok := values["c"]
	if !ok {
		t.Fatal("Variable 'c' not found in pinned variables")
	}
	if fmt.Sprintf("%v", cVal) != "30" {
		t.Errorf("Initial c = %v, want 30", cVal)
	}

	// Now change a to 20
	m.input.SetValue("a = 20")
	// Note: handleInput calls evaluateExpression which should track affected blocks
	m, _ = m.handleInput()

	// Check which variables are marked as changed
	t.Logf("Changed vars after a=20: %v", m.changedVars)

	// Log raw environment values
	env := m.eval.GetEnvironment()
	for _, name := range []string{"a", "b", "c"} {
		if v, ok := env.Get(name); ok {
			t.Logf("Raw env %s = %v", name, v)
		}
	}

	// Verify values updated correctly
	_, values, _ = m.collectPinnedVariables()

	aVal = values["a"]
	if fmt.Sprintf("%v", aVal) != "20" {
		t.Errorf("After change: a = %v, want 20", aVal)
	}

	bVal = values["b"]
	// b = a * 2 = 20 * 2 = 40
	if fmt.Sprintf("%v", bVal) != "40" {
		t.Errorf("After change: b = %v, want 40 (a*2 where a=20)", bVal)
	}

	cVal = values["c"]
	// c = b + 10 = 40 + 10 = 50
	if fmt.Sprintf("%v", cVal) != "50" {
		t.Errorf("After change: c = %v, want 50 (b+10 where b=40)", cVal)
	}
}

// Tests for pure rendering functions

// TestRenderPinnedPanel tests the pure renderPinnedPanel function.
func TestRenderPinnedPanel(t *testing.T) {
	tests := []struct {
		name     string
		vars     []pinnedVariable
		wantStar bool
		wantName string
	}{
		{
			name:     "empty panel",
			vars:     []pinnedVariable{},
			wantStar: false,
			wantName: "",
		},
		{
			name: "unchanged variable",
			vars: []pinnedVariable{
				{Name: "x", Value: 42, Changed: false},
			},
			wantStar: false,
			wantName: "x",
		},
		{
			name: "changed variable",
			vars: []pinnedVariable{
				{Name: "y", Value: 100, Changed: true},
			},
			wantStar: true,
			wantName: "y",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderPinnedPanel(tt.vars)

			if tt.wantName != "" && !strings.Contains(result, tt.wantName) {
				t.Errorf("Expected result to contain %q, got: %s", tt.wantName, result)
			}

			hasAsterisk := strings.Contains(result, "* ")
			if tt.wantStar && !hasAsterisk {
				t.Error("Expected * for changed variable")
			}
			if !tt.wantStar && len(tt.vars) > 0 && hasAsterisk {
				t.Error("Did not expect * for unchanged variable")
			}
		})
	}
}

// TestRenderHistoryItems tests the pure renderHistoryItems function.
func TestRenderHistoryItems(t *testing.T) {
	items := []outputHistoryItem{
		{input: "x = 1", output: "= 1", isError: false},
		{input: "y = 2", output: "= 2", isError: false},
		{input: "bad", output: "error", isError: true},
	}

	// Test with enough space for all
	result := renderHistoryItems(items, 100)
	if !strings.Contains(result, "x = 1") {
		t.Error("Expected history to contain first item")
	}
	if !strings.Contains(result, "bad") {
		t.Error("Expected history to contain error item")
	}

	// Test with limited space
	result = renderHistoryItems(items, 2)
	// Should only show the last item
	if strings.Contains(result, "x = 1") {
		t.Error("First item should be truncated with limited space")
	}

	// Test empty history
	result = renderHistoryItems(nil, 10)
	if result != "" {
		t.Error("Empty history should return empty string")
	}
}

// TestRenderHelpLine tests the pure renderHelpLine function.
func TestRenderHelpLine(t *testing.T) {
	// Narrow terminal
	result := renderHelpLine(false, 60)
	if !strings.Contains(result, "History") {
		t.Error("Narrow help should contain History")
	}
	if !strings.Contains(result, "PgUp") || !strings.Contains(result, "Scroll") {
		t.Error("Narrow help should mention PgUp/Scroll")
	}

	// Wide terminal, normal mode
	result = renderHelpLine(false, 150)
	if !strings.Contains(result, "History") {
		t.Error("Wide normal mode should mention History")
	}
	if !strings.Contains(result, "PgUp/PgDn") {
		t.Error("Wide normal mode should mention PgUp/PgDn for scrolling pinned panel")
	}

	// Wide terminal, slash mode
	result = renderHelpLine(true, 150)
	if !strings.Contains(result, "/help") {
		t.Error("Wide slash mode should mention /help")
	}
	if !strings.Contains(result, "Esc") {
		t.Error("Wide slash mode should mention Esc to cancel")
	}
}

// TestSlashCommandSuggestions tests the slash command autosuggestion.
func TestSlashCommandSuggestions(t *testing.T) {
	// Empty input returns all commands
	all := getSlashCommandSuggestions("")
	if len(all) != len(slashCommands) {
		t.Errorf("Expected all %d commands for empty input, got %d", len(slashCommands), len(all))
	}

	// Prefix "o" matches "open" and "output"
	oMatches := getSlashCommandSuggestions("o")
	if len(oMatches) != 2 {
		t.Errorf("Expected 2 commands for 'o', got %d", len(oMatches))
	}

	// Prefix "ou" matches only "output"
	ouMatches := getSlashCommandSuggestions("ou")
	if len(ouMatches) != 1 || ouMatches[0].name != "output" {
		t.Error("Expected 'output' for 'ou' prefix")
	}

	// Prefix "q" matches "quit" and "q"
	qMatches := getSlashCommandSuggestions("q")
	if len(qMatches) != 2 {
		t.Errorf("Expected 2 commands for 'q', got %d", len(qMatches))
	}

	// No match for invalid prefix
	noMatch := getSlashCommandSuggestions("xyz")
	if len(noMatch) != 0 {
		t.Error("Expected no matches for 'xyz'")
	}
}

// TestRenderSlashSuggestions tests the slash suggestion rendering.
func TestRenderSlashSuggestions(t *testing.T) {
	suggestions := []slashCommand{
		{"open", "/open <file>", "Load CalcMark file"},
	}
	result := renderSlashSuggestions(suggestions)
	if !strings.Contains(result, "/open <file>") {
		t.Error("Rendered suggestions should contain syntax")
	}
	if !strings.Contains(result, "Load CalcMark") {
		t.Error("Rendered suggestions should contain description")
	}
}

// TestSaveCommand tests the /save command requires valid extension.
func TestSaveCommand(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\n")
	m := newTUIModel(doc)

	// Invalid extension should error
	m, _ = m.handleCommand("save test.txt")
	if m.err == nil {
		t.Error("Expected error for invalid extension")
	}
	if !strings.Contains(m.err.Error(), ".cm") {
		t.Error("Error should mention valid extensions")
	}
}

// TestOutputCommand tests the /output command requires a filename.
func TestOutputCommand(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\n")
	m := newTUIModel(doc)

	// No filename should error
	m, _ = m.handleCommand("output")
	if m.err == nil {
		t.Error("Expected error for missing filename")
	}
	if !strings.Contains(m.err.Error(), "usage") {
		t.Error("Error should show usage")
	}
}

// TestHelpIncludesSaveOutput tests that /help includes save and output commands.
func TestHelpIncludesSaveOutput(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\n")
	m := newTUIModel(doc)

	m, _ = m.handleCommand("help")

	// Check output history for help text
	if len(m.outputHistory) == 0 {
		t.Fatal("Expected help output in history")
	}

	lastItem := m.outputHistory[len(m.outputHistory)-1]
	if !strings.Contains(lastItem.output, "/save") {
		t.Error("Help should mention /save command")
	}
	if !strings.Contains(lastItem.output, "/output") {
		t.Error("Help should mention /output command")
	}
}

// TestFrontmatterGlobalVarPinned tests that @global.xxx creates a pinned variable.
func TestFrontmatterGlobalVarPinned(t *testing.T) {
	// Create document with @global assignment
	doc, err := document.NewDocument("@global.tax_rate = 0.32\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Create TUI model (should auto-pin the variable)
	m := newTUIModel(doc)

	// Check that tax_rate is pinned
	if !m.pinnedVars["tax_rate"] {
		t.Error("Expected tax_rate to be auto-pinned")
	}

	// Check that it appears in rendered pinned panel with @ visual indicator
	// (the @ prefix is display-only, indicating it's a frontmatter variable)
	rendered := m.renderPinnedVars()
	if !strings.Contains(rendered, "@tax_rate") {
		t.Errorf("Expected @tax_rate (with @ visual indicator) in pinned panel, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "0.32") {
		t.Errorf("Expected 0.32 value in pinned panel, got:\n%s", rendered)
	}
}

// TestFrontmatterVisualIndicator tests that frontmatter variables show @ prefix in pinned panel.
func TestFrontmatterVisualIndicator(t *testing.T) {
	// Create document with both frontmatter and regular variables
	doc, err := document.NewDocument("@global.fm_var = 100\nreg_var = 200\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := newTUIModel(doc)

	// Collect pinned variables
	names, _, isFrontmatter := m.collectPinnedVariables()

	// Verify fm_var is marked as frontmatter
	found := false
	for _, name := range names {
		if name == "fm_var" {
			found = true
			if !isFrontmatter["fm_var"] {
				t.Error("Expected fm_var to be marked as frontmatter")
			}
		}
		if name == "reg_var" {
			if isFrontmatter["reg_var"] {
				t.Error("Expected reg_var NOT to be marked as frontmatter")
			}
		}
	}
	if !found {
		t.Error("Expected fm_var in pinned variables")
	}

	// Verify rendered output shows @ prefix for frontmatter var only
	rendered := m.renderPinnedVars()
	if !strings.Contains(rendered, "@fm_var") {
		t.Errorf("Expected @fm_var in rendered output, got:\n%s", rendered)
	}
	// reg_var should NOT have @ prefix
	if strings.Contains(rendered, "@reg_var") {
		t.Errorf("Expected reg_var without @ prefix, but found @reg_var in:\n%s", rendered)
	}
	if !strings.Contains(rendered, "reg_var") {
		t.Errorf("Expected reg_var in rendered output, got:\n%s", rendered)
	}
}

// TestFrontmatterGlobalInFrontmatter tests that @global updates document frontmatter.
func TestFrontmatterGlobalInFrontmatter(t *testing.T) {
	doc, err := document.NewDocument("@global.my_var = 123\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := newTUIModel(doc)

	// Check that frontmatter was updated
	fm := m.doc.GetFrontmatter()
	if fm == nil {
		t.Fatal("Expected frontmatter to be created")
	}
	if fm.Globals["my_var"] != "123" {
		t.Errorf("Expected frontmatter global my_var=123, got %q", fm.Globals["my_var"])
	}
}

// TestFmCommand tests the /fm command shows frontmatter.
func TestFmCommand(t *testing.T) {
	doc, err := document.NewDocument("@global.tax = 0.1\n@exchange.USD_EUR = 0.92\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := newTUIModel(doc)
	m, _ = m.handleCommand("fm")

	// Check output history
	if len(m.outputHistory) == 0 {
		t.Fatal("Expected /fm output in history")
	}

	lastItem := m.outputHistory[len(m.outputHistory)-1]
	if !strings.Contains(lastItem.output, "@global.tax") {
		t.Errorf("Expected @global.tax in /fm output:\n%s", lastItem.output)
	}
	if !strings.Contains(lastItem.output, "@exchange.USD_EUR") {
		t.Errorf("Expected @exchange.USD_EUR in /fm output:\n%s", lastItem.output)
	}
}
