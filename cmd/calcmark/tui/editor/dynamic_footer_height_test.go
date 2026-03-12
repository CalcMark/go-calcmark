package editor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestFooterShowsFullHintOnErrorLine verifies that when the cursor is on an
// error line, the rendered view shows the full diagnostic hint — not truncated.
// This is the core reason the footer expands: hints were being cut off.
func TestFooterShowsFullHintOnErrorLine(t *testing.T) {
	doc, _ := document.NewDocument("result = undefined_var * 2\n")
	m := New(doc)
	m.width = 80
	m.height = 24
	m.cursorLine = 0

	view := m.View().Content
	plain := stripAnsi(view)

	// The semantic checker sets Detailed to "Defined variables: E, PI"
	// for undefined variable errors. The footer should show this hint.
	if !strings.Contains(plain, "Defined variables") {
		t.Errorf("Footer should show the full diagnostic hint.\nView:\n%s", plain)
	}
}

// TestFooterHintNotTruncatedAtNarrowWidth verifies that at narrow widths the
// hint word-wraps across multiple footer lines rather than being cut off.
func TestFooterHintNotTruncatedAtNarrowWidth(t *testing.T) {
	doc, _ := document.NewDocument("result = undefined_var * 2\n")
	m := New(doc)
	m.width = 40
	m.height = 24
	m.cursorLine = 0

	view := m.View().Content
	plain := stripAnsi(view)

	// Even at narrow width, the hint text should appear fully
	if !strings.Contains(plain, "Defined variables") {
		t.Errorf("Hint should be visible even at narrow width.\nView:\n%s", plain)
	}
}

// TestViewHasCorrectLineCountAlways verifies that the rendered view always
// produces exactly m.height lines — no more, no fewer. Extra or missing lines
// cause bubbletea rendering artifacts (flickering, ghost lines).
func TestViewHasCorrectLineCountAlways(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		cursorLine int
		height     int
	}{
		{"normal doc", "x = 10\ny = 20\n", 0, 24},
		{"cursor on error", "result = bad * 2\n", 0, 24},
		{"error in multi-line doc", "x = 10\nresult = bad * 2\n", 1, 24},
		{"small terminal no error", "x = 10\n", 0, 12},
		{"small terminal with error", "result = bad * 2\n", 0, 12},
		{"tall terminal", "x = 10\n", 0, 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, _ := document.NewDocument(tt.source)
			m := New(doc)
			m.width = 80
			m.height = tt.height
			m.cursorLine = tt.cursorLine

			view := m.View().Content
			lines := strings.Split(view, "\n")
			if len(lines) != tt.height {
				t.Errorf("View should have exactly %d lines, got %d", tt.height, len(lines))
			}
		})
	}
}

// TestCursorVisibleWhenFooterExpands verifies that the cursor line remains
// visible in the rendered output when the footer expands. The footer expansion
// steals content rows, so the scroll system must compensate.
func TestCursorVisibleWhenFooterExpands(t *testing.T) {
	// Build a document where the error is on the last visible line,
	// so footer expansion could push it off-screen.
	var lines []string
	for i := range 20 {
		if i == 19 {
			lines = append(lines, "result = undefined_var * 2")
		} else {
			lines = append(lines, "x = 10")
		}
	}
	source := strings.Join(lines, "\n") + "\n"

	doc, _ := document.NewDocument(source)
	m := New(doc)
	m.width = 80
	m.height = 16
	m.cursorLine = 19 // last line (error line)
	m.adjustScrollForCursor()

	view := m.View().Content
	plain := stripAnsi(view)

	// The error line's content should be visible in the rendered output
	if !strings.Contains(plain, "undefined_var") {
		t.Errorf("Cursor line should be visible even when footer expands.\nView:\n%s", plain)
	}
}

// TestFooterDoesNotExpandOnNonErrorLine verifies that the view uses the full
// content area when the cursor is on a non-error line (footer stays compact).
func TestFooterDoesNotExpandOnNonErrorLine(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\nresult = undefined_var * 2\n")
	m := New(doc)
	m.width = 80
	m.height = 24
	m.cursorLine = 0 // non-error line

	view := m.View().Content
	plain := stripAnsi(view)

	// The footer should NOT show a diagnostic hint when cursor is on a good line
	if strings.Contains(plain, "Defined variables") {
		t.Errorf("Footer should not show diagnostic hint when cursor is on non-error line.\nView:\n%s", plain)
	}
}

// TestFooterTransitionOnCursorMovement verifies that moving the cursor between
// error and non-error lines produces different rendered views — the footer
// expands and contracts without breaking the total line count.
func TestFooterTransitionOnCursorMovement(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\nresult = undefined_var * 2\n")
	m := New(doc)
	m.width = 80
	m.height = 24

	// Render with cursor on good line
	m.cursorLine = 0
	viewGood := m.View().Content
	linesGood := strings.Split(viewGood, "\n")

	// Render with cursor on error line
	m.cursorLine = 1
	viewError := m.View().Content
	linesError := strings.Split(viewError, "\n")

	// Both must have exactly m.height lines
	if len(linesGood) != m.height {
		t.Errorf("Good line view: expected %d lines, got %d", m.height, len(linesGood))
	}
	if len(linesError) != m.height {
		t.Errorf("Error line view: expected %d lines, got %d", m.height, len(linesError))
	}

	// Error view should contain the hint, good view should not
	plainError := stripAnsi(viewError)
	plainGood := stripAnsi(viewGood)
	if !strings.Contains(plainError, "Defined variables") {
		t.Error("Error line view should show diagnostic hint")
	}
	if strings.Contains(plainGood, "Defined variables") {
		t.Error("Good line view should not show diagnostic hint")
	}
}

// TestAutocompleteFooterDoesNotExpand verifies that the footer stays compact
// when autocomplete is active, even if the cursor is on an error line.
// Autocomplete needs the screen space for the popup.
func TestAutocompleteFooterDoesNotExpand(t *testing.T) {
	doc, _ := document.NewDocument("result = undefined_var * 2\n")
	m := New(doc)
	m.width = 80
	m.height = 24
	m.cursorLine = 0
	m.mode = StateAutocomplete
	m.autocompleteState.Visible = true

	view := m.View().Content
	lines := strings.Split(view, "\n")

	// Total line count must still be exact
	if len(lines) != m.height {
		t.Errorf("View should have %d lines during autocomplete, got %d", m.height, len(lines))
	}
}

// getFooterText extracts the context footer area from a rendered view.
// Layout: [content panes] [empty] [context footer] [empty] [separator ────] [status bar] [pad]
// The context footer is the area ABOVE the separator line.
func getFooterText(view string) string {
	lines := strings.Split(view, "\n")
	// Find the separator line (contains ────)
	sepIdx := -1
	for i, line := range lines {
		plain := stripAnsi(line)
		if strings.Contains(plain, "────") {
			sepIdx = i
			break
		}
	}
	if sepIdx < 2 {
		return ""
	}
	// Context footer is the 2-4 lines before the separator (plus empty line above separator)
	// Grab up to 5 lines before separator to cover max footer + empty line
	start := max(0, sepIdx-5)
	footerLines := lines[start:sepIdx]
	return stripAnsi(strings.Join(footerLines, "\n"))
}

// TestFrontmatterErrorVisibleFromAnyLineInBlock verifies that the diagnostic
// footer shows the frontmatter error when the cursor is on ANY frontmatter
// line, not just the closing ---. The error belongs to the whole block.
func TestFrontmatterErrorVisibleFromAnyLineInBlock(t *testing.T) {
	// Start with valid frontmatter, then simulate an edit that breaks YAML.
	// This mirrors how the TUI works: NewDocument succeeds initially, then
	// the user types something that makes the YAML invalid — the editor
	// sets frontmatterErr and keeps the old doc with raw source updated.
	source := "---\nhello: there\nconvert_to: si\n---\n\na = 10 lb\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Valid frontmatter should parse: %v", err)
	}
	m := New(doc)
	m.width = 80
	m.height = 24

	// Simulate a frontmatter edit that breaks YAML (like adding "scale" without a value).
	// The TUI sets frontmatterErr when document.NewDocument fails after an edit.
	m.frontmatterErr = fmt.Errorf("frontmatter: could not find expected ':'")

	// Cursor on the closing --- (line 3, 0-indexed) — this is where the error
	// is attached in results.go. This should definitely show the error in the footer.
	m.cursorLine = 3
	footer := getFooterText(m.View().Content)
	if !strings.Contains(footer, "could not find") && !strings.Contains(footer, "frontmatter") {
		t.Errorf("Closing --- line: Footer should show frontmatter error.\nFooter:\n%s", footer)
	}

	// Cursor on a middle frontmatter line (line 1, "hello: there").
	// The error should ALSO be visible in the footer since it belongs to the whole block.
	m.cursorLine = 1
	footer = getFooterText(m.View().Content)
	if !strings.Contains(footer, "could not find") && !strings.Contains(footer, "frontmatter") {
		t.Errorf("Middle frontmatter line: Footer should show frontmatter error.\nFooter:\n%s", footer)
	}

	// Non-frontmatter line should NOT show the frontmatter error in the footer.
	m.cursorLine = 5 // "a = 10 lb"
	footer = getFooterText(m.View().Content)
	if strings.Contains(footer, "could not find") {
		t.Errorf("Calc line: Footer should not show frontmatter error.\nFooter:\n%s", footer)
	}
}

// TestCountWrappedLines verifies line counting for word-wrapped text.
func TestCountWrappedLines(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		width int
		want  int
	}{
		{"empty", "", 80, 1},
		{"short text", "hello world", 80, 1},
		{"fits in width", "1234567890123456789", 20, 1},
		{"wraps once", "123456789012345678901", 20, 2},
		{"wraps twice", "1234567890123456789012345678901234567890x", 20, 3},
		{"explicit newline", "line1\nline2", 80, 2},
		{"zero width", "text", 0, 1},
		{"negative width", "text", -5, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countWrappedLines(tt.text, tt.width)
			if got != tt.want {
				t.Errorf("countWrappedLines(%q, %d) = %d, want %d",
					tt.text, tt.width, got, tt.want)
			}
		})
	}
}
