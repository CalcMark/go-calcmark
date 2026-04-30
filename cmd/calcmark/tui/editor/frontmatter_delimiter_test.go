package editor

// frontmatter_delimiter_test.go — Tests for editing frontmatter delimiters.
// Ensures that deleting characters from the closing --- delimiter does NOT
// cause automatic removal of the frontmatter block or any text loss.
//
// The core invariant being tested: text is NEVER deleted automatically unless
// the user explicitly deletes with backspace/delete or uses ctrl-z/ctrl-y.

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

// simulateDebounce simulates the debounce timer firing by sending an
// evalDebounceMsg with the current editBuf as the snapshot. In the real app,
// this message fires after evalDebounceDelay (100ms). Since tests don't run
// the Bubble Tea event loop, we must send it manually.
func simulateDebounce(t *testing.T, model tea.Model) tea.Model {
	t.Helper()
	ed := model.(Model)
	msg := evalDebounceMsg{editBufSnapshot: ed.editBuf}
	newM, _ := ed.Update(msg)
	return newM
}

// assertLinesContain checks that all expected strings appear in lines.
func assertLinesContain(t *testing.T, lines []string, expected []string, context string) {
	t.Helper()
	for _, exp := range expected {
		if !slices.Contains(lines, exp) {
			t.Errorf("BUG [%s]: Expected line %q not found in lines: %v", context, exp, lines)
		}
	}
}

// TestFrontmatterDelimiterEdit_BackspaceOnClosingDelimiter tests that deleting
// a single '-' from the closing '---' delimiter does NOT remove the frontmatter block.
//
// Scenario: Open empty editor -> Ctrl+F to insert frontmatter -> navigate to
// closing '---' line -> End key -> Backspace -> debounce.
// Expectation: Text is modified (--- -> --) but frontmatter block is preserved.
func TestFrontmatterDelimiterEdit_BackspaceOnClosingDelimiter(t *testing.T) {
	doc, err := document.NewDocument("\n")
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	var model tea.Model = m

	// Step 1: Insert frontmatter (Ctrl+F)
	model = sendKey(t, model, "ctrl+f")

	ed := model.(Model)
	initialFmCount := ed.frontmatterLineCount()
	initialTotal := ed.TotalLines()
	t.Logf("After Ctrl+F: totalLines=%d fmCount=%d cursorLine=%d cursorCol=%d",
		initialTotal, initialFmCount, ed.cursorLine, ed.cursorCol)
	t.Logf("Lines: %v", ed.GetLines())

	if ed.doc.GetFrontmatter() == nil {
		t.Fatal("Expected frontmatter to be present after Ctrl+F")
	}
	if initialFmCount != 10 {
		t.Fatalf("Expected 10 frontmatter lines, got %d", initialFmCount)
	}
	if initialTotal != 11 {
		t.Fatalf("Expected 11 total lines, got %d", initialTotal)
	}

	// Step 2: Navigate to closing --- line (line 9)
	for range 7 {
		model = sendKey(t, model, "down")
	}

	ed = model.(Model)
	if ed.cursorLine != 9 {
		t.Fatalf("Expected cursor on line 9 (closing ---), got line %d", ed.cursorLine)
	}

	// Step 3: Go to end of line
	model = sendKey(t, model, "end")

	ed = model.(Model)
	if ed.editBuf != "---" {
		t.Fatalf("Expected editBuf='---', got %q", ed.editBuf)
	}

	// Step 4: Press backspace to delete single '-'
	model = sendKey(t, model, "backspace")

	ed = model.(Model)
	if ed.editBuf != "--" {
		t.Fatalf("Expected editBuf='--' after backspace, got %q", ed.editBuf)
	}

	// Step 5: Simulate debounce (triggers transitionToProcessing -> transitionToReady)
	model = simulateDebounce(t, model)

	ed = model.(Model)
	lines := ed.GetLines()
	t.Logf("After debounce: totalLines=%d fmCount=%d cursorLine=%d",
		len(lines), ed.frontmatterLineCount(), ed.cursorLine)
	t.Logf("Lines: %v", lines)

	if ed.doc.GetFrontmatter() == nil {
		t.Error("BUG: Frontmatter was removed! Expected it to be preserved.")
	}
	if len(lines) < initialTotal-1 {
		t.Errorf("BUG: Too many lines lost. Expected at least %d lines, got %d",
			initialTotal-1, len(lines))
	}
	if !slices.Contains(lines, "--") {
		t.Error("BUG: Modified closing delimiter '--' not found in lines")
		t.Errorf("Lines: %v", lines)
	}
	assertLinesContain(t, lines,
		[]string{"exchange:", "  USD_EUR: 0.92", "globals:", "  tax_rate: 0.085", "scale:", "  factor: 2", "convert_to: si"},
		"after backspace on closing delimiter")
}

// TestFrontmatterDelimiterEdit_BodyBlockRemovedThenDebounce tests the critical
// scenario where the only body block is removed, leaving frontmatter with 0
// blocks, and then the debounce fires triggering transitionToReady.
//
// This is the primary reproduction case: transitionToReady unconditionally
// replaces a 0-block document with NewDocument("\n"), discarding all frontmatter.
func TestFrontmatterDelimiterEdit_BodyBlockRemovedThenDebounce(t *testing.T) {
	doc, err := document.NewDocument("\n")
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	var model tea.Model = m

	// Insert frontmatter
	model = sendKey(t, model, "ctrl+f")

	ed := model.(Model)
	t.Logf("After Ctrl+F: totalLines=%d fmCount=%d", ed.TotalLines(), ed.frontmatterLineCount())
	t.Logf("Lines: %v", ed.GetLines())

	if ed.doc.GetFrontmatter() == nil {
		t.Fatal("Expected frontmatter after Ctrl+F")
	}

	// Navigate to last line (empty body line, line 10)
	for range 8 {
		model = sendKey(t, model, "down")
	}
	ed = model.(Model)
	t.Logf("Before body removal: cursorLine=%d totalLines=%d", ed.cursorLine, ed.TotalLines())

	// Press backspace at col 0 of the body line — this joins with the closing ---
	// which triggers deleteLine() on the body block, leaving 0 blocks.
	model = sendKey(t, model, "backspace")

	ed = model.(Model)
	t.Logf("After line join: cursorLine=%d editBuf=%q totalLines=%d fmCount=%d blocks=%d",
		ed.cursorLine, ed.editBuf, ed.TotalLines(), ed.frontmatterLineCount(),
		len(ed.doc.GetBlocks()))

	// NOW simulate the debounce firing — this is where transitionToReady runs
	model = simulateDebounce(t, model)

	ed = model.(Model)
	lines := ed.GetLines()
	t.Logf("After debounce: totalLines=%d fmCount=%d blocks=%d",
		len(lines), ed.frontmatterLineCount(), len(ed.doc.GetBlocks()))
	t.Logf("Lines: %v", lines)

	// CRITICAL: Frontmatter must survive the debounce!
	if ed.doc.GetFrontmatter() == nil {
		t.Error("BUG: Frontmatter was destroyed by transitionToReady!")
		t.Error("This happens because transitionToReady replaces 0-block documents")
		t.Error("with NewDocument(\"\\n\") which has no frontmatter.")
	}

	assertLinesContain(t, lines,
		[]string{"---", "exchange:", "  USD_EUR: 0.92", "globals:", "  tax_rate: 0.085", "scale:", "  factor: 2", "convert_to: si"},
		"after body block removal + debounce")
}

// TestFrontmatterDelimiterEdit_DeleteClosingDelimiterEndOfDoc tests the
// exact user scenario: empty editor -> Ctrl+F -> go to end of doc ->
// delete the closing '---' character by character.
func TestFrontmatterDelimiterEdit_DeleteClosingDelimiterEndOfDoc(t *testing.T) {
	doc, err := document.NewDocument("\n")
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	var model tea.Model = m

	// Insert frontmatter
	model = sendKey(t, model, "ctrl+f")

	// Navigate to the closing --- line (line 9), go to end
	for range 7 {
		model = sendKey(t, model, "down")
	}
	model = sendKey(t, model, "end")

	ed := model.(Model)
	t.Logf("Starting deletion: cursorLine=%d cursorCol=%d editBuf=%q",
		ed.cursorLine, ed.cursorCol, ed.editBuf)

	// Delete each '-' one by one, simulating debounce after each
	for i := range 3 {
		model = sendKey(t, model, "backspace")

		ed = model.(Model)
		expectedBuf := strings.Repeat("-", 2-i)
		t.Logf("After backspace %d: editBuf=%q (expected %q)",
			i+1, ed.editBuf, expectedBuf)

		// Simulate debounce after each deletion
		model = simulateDebounce(t, model)

		ed = model.(Model)
		lines := ed.GetLines()
		t.Logf("After debounce %d: totalLines=%d fmCount=%d blocks=%d",
			i+1, len(lines), ed.frontmatterLineCount(), len(ed.doc.GetBlocks()))
		t.Logf("Lines: %v", lines)

		if ed.doc.GetFrontmatter() == nil {
			t.Errorf("BUG: Frontmatter lost after deleting %d dashes!", i+1)
			break
		}

		assertLinesContain(t, lines,
			[]string{"exchange:", "  USD_EUR: 0.92", "globals:", "  tax_rate: 0.085", "scale:", "  factor: 2", "convert_to: si"},
			"after "+strings.Repeat("backspace+debounce", i+1))
	}
}

// TestFrontmatterDelimiterEdit_DeleteAllDashes tests deleting ALL dashes from
// the closing delimiter one at a time, flushing via navigation.
func TestFrontmatterDelimiterEdit_DeleteAllDashes(t *testing.T) {
	content := "---\nexchange:\n  USD_EUR: 0.92\nglobals:\n  my_var: 42\n---\nx = 10"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	var model tea.Model = m

	initialLines := m.GetLines()
	t.Logf("Initial lines (%d): %v", len(initialLines), initialLines)

	// Navigate to closing --- line (line 5)
	for range 5 {
		model = sendKey(t, model, "down")
	}
	model = sendKey(t, model, "end")

	for i := range 3 {
		model = sendKey(t, model, "backspace")

		ed := model.(Model)
		expected := strings.Repeat("-", 2-i)
		t.Logf("After backspace %d: editBuf=%q (expected %q)",
			i+1, ed.editBuf, expected)

		// Flush by navigating down then back up
		model = sendKey(t, model, "down")
		model = sendKey(t, model, "up")

		ed = model.(Model)
		lines := ed.GetLines()
		t.Logf("After flush %d: totalLines=%d fmCount=%d",
			i+1, len(lines), ed.frontmatterLineCount())
		t.Logf("Lines: %v", lines)

		if ed.doc.GetFrontmatter() == nil {
			t.Errorf("BUG: Frontmatter was removed after deleting %d dashes!", i+1)
			break
		}

		assertLinesContain(t, lines,
			[]string{"---", "exchange:", "  USD_EUR: 0.92", "globals:", "  my_var: 42"},
			"after "+strings.Repeat("backspace", i+1))

		if !slices.Contains(lines, "x = 10") {
			t.Errorf("BUG: Body line 'x = 10' was lost after %d backspaces!", i+1)
		}

		// Navigate back to the closing delimiter line for the next backspace
		model = sendKey(t, model, "end")
	}
}

// TestFrontmatterDelimiterEdit_NoAutoRemoval verifies the core principle:
// text is NEVER deleted automatically unless the user explicitly deletes with
// backspace/delete or uses ctrl-z/ctrl-y.
func TestFrontmatterDelimiterEdit_NoAutoRemoval(t *testing.T) {
	content := "---\nexchange:\n  USD_EUR: 0.92\nglobals:\n  my_var: 42\n---\nhello world"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	var model tea.Model = m

	initialLines := m.GetLines()
	t.Logf("Initial: %v", initialLines)

	// Navigate to closing --- (line 5), go to end, type 'x' (making it "---x")
	for range 5 {
		model = sendKey(t, model, "down")
	}
	model = sendKey(t, model, "end")
	model = typeText(t, model, "x")

	// Simulate debounce to flush
	model = simulateDebounce(t, model)

	ed := model.(Model)
	lines := ed.GetLines()
	t.Logf("After typing 'x' on closing delimiter: %v", lines)

	if ed.doc.GetFrontmatter() == nil {
		t.Error("BUG: Frontmatter was removed after typing on closing delimiter!")
	}
	if len(lines) < len(initialLines) {
		t.Errorf("BUG: Lines were automatically deleted! Had %d, now have %d",
			len(initialLines), len(lines))
		t.Errorf("Before: %v", initialLines)
		t.Errorf("After:  %v", lines)
	}
	if !slices.Contains(lines, "hello world") {
		t.Errorf("BUG: Body line 'hello world' was auto-removed!")
		t.Errorf("Lines: %v", lines)
	}
}
