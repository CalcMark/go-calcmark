package editor

// view_rendering_test.go — View() output, preview pane, wrapping, height consistency.

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

func TestView(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\ny = 20\n")
	m := New(doc)
	m.width = 80
	m.height = 24

	view := m.View().Content

	// Should contain source header
	if !strings.Contains(view, "Source") {
		t.Error("View should contain 'Source' header")
	}

	// Should contain results header (preview visible by default)
	if !strings.Contains(view, "Results") {
		t.Error("View should contain 'Results' header")
	}

	// Mode is no longer displayed - it's an internal implementation detail
	// Just verify view is not empty
	if len(view) == 0 {
		t.Error("View should not be empty")
	}
}

func TestViewEmptyDocument(t *testing.T) {
	// Test that viewing an empty document doesn't crash or produce empty output
	doc, _ := document.NewDocument("")
	m := New(doc)
	m.width = 80
	m.height = 24

	view := m.View().Content

	// Should not be empty or crash
	if len(view) == 0 {
		t.Error("View of empty document should not be empty string")
	}

	// Should still have Source header
	if !strings.Contains(view, "Source") {
		t.Error("View of empty document should contain 'Source' header")
	}

	// Should not be all whitespace
	if strings.TrimSpace(view) == "" {
		t.Error("View of empty document should not be all whitespace")
	}
}

func TestViewAfterEnterEditMode(t *testing.T) {
	// Test viewing after entering edit mode on empty doc
	doc, _ := document.NewDocument("")
	m := New(doc)
	m.width = 80
	m.height = 24

	// Enter edit mode
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()

	if m.mode != StateDefault {
		t.Fatalf("Expected StateDefault, got %v", m.mode)
	}

	view := m.View().Content

	// Should not be empty
	if len(view) == 0 {
		t.Error("View after entering edit mode should not be empty")
	}

	// Mode is no longer displayed - it's an internal implementation detail
	// Editing mode is transparent to the user
}

func TestPreviewModeRendering(t *testing.T) {
	doc, _ := document.NewDocument("x = 100\n")
	m := New(doc)
	m.width = 80
	m.height = 24

	// Test Full preview mode
	m.previewMode = PreviewFull
	view := m.View().Content
	if !strings.Contains(view, "Results") {
		t.Error("Full preview mode should show Results header")
	}

	// Test Minimal preview mode
	m.previewMode = PreviewMinimal
	view = m.View().Content
	if !strings.Contains(view, "Results") {
		t.Error("Minimal preview mode should show Results header")
	}

	// Test Hidden preview mode
	m.previewMode = PreviewHidden
	view = m.View().Content
	if strings.Contains(view, "Results") {
		t.Error("Hidden preview mode should not show Results header")
	}
}

func TestEditModeWrappedLineNoDuplicate(t *testing.T) {
	content := `this is a really long line of markdown that should wrap`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	// Set narrow width to force wrapping
	m.width = 60
	m.height = 20

	// Enter edit mode on the first line
	m.mode = StateDefault
	m.cursorLine = 0
	m.editBuf = content
	m.cursorCol = len(content)

	view := m.View().Content
	t.Logf("VIEW OUTPUT:\n%s", view)

	// Count occurrences of "markdown" - it should only appear TWICE
	// (once in source pane, once in preview pane - not duplicated due to wrapping)
	// Note: "should wrap" may be split across lines due to word wrapping
	occurrences := strings.Count(view, "markdown")
	if occurrences > 2 {
		t.Errorf("Text 'markdown' appears %d times, expected 2 (duplicate wrapped line bug)", occurrences)
	}
	if occurrences == 0 {
		t.Error("Text 'markdown' not found in output")
	}

	// The line number "1" should only appear once
	lineNum1Count := strings.Count(view, "   1 ")
	if lineNum1Count > 1 {
		t.Errorf("Line number 1 appears %d times, expected 1", lineNum1Count)
	}
}

// TestLongLineWrappingInEditor tests that long lines wrap in the editor pane
// instead of being truncated with "...".
func TestLongLineWrappingInEditor(t *testing.T) {
	// Create a document with a very long variable name and expression
	content := `x = 1
heres_a_reeeeeeeeeeeeeeeeeeeeeeeeeelly_long_variable_name = x * 2`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	// Set a narrow width to force wrapping
	m.width = 60
	m.height = 20

	// Get the view output
	view := m.View().Content

	// Debug: dump the view
	t.Logf("VIEW OUTPUT:\n%s", view)

	// The SOURCE PANE should wrap long lines (not truncate with ...)
	// The wrapped line may appear across multiple visual lines, so we check
	// that the content is present even if split across lines.
	// We verify wrapping by checking:
	// 1. The expression "= x * 2" appears somewhere (it would be cut off if truncated)
	// 2. The content wraps across multiple lines

	// Check that "= x * 2" appears (the assignment expression at the end)
	// This would be cut off if truncation happened instead of wrapping
	hasExpression := strings.Contains(view, "= x * 2")
	if !hasExpression {
		t.Error("Long line in source pane is truncated instead of wrapping - missing end of expression")
	}

	// Check for continuation lines (wrapped content) - they show indented text without line numbers
	// Look for parts of the long variable name or the expression on wrapped lines
	lines := strings.Split(view, "\n")
	foundWrappedContent := false
	for _, line := range lines {
		// A wrapped line contains continuation text (parts of the long name or expression).
		// In v2, lines include ANSI codes, so use Contains instead of HasPrefix.
		if strings.Contains(line, "eeeee") || strings.Contains(line, "= x * 2") {
			foundWrappedContent = true
			break
		}
	}
	if !foundWrappedContent {
		t.Error("Expected to find wrapped continuation lines in source pane")
	}

	// The line number "2" should only appear once (wrapped lines don't get line numbers)
	lineNum2Count := strings.Count(view, "   2 ")
	if lineNum2Count > 1 {
		t.Errorf("Line number 2 appears %d times, expected 1 (wrapped lines should not have line numbers)", lineNum2Count)
	}
}

// TestLongMarkdownWrappingInEditor tests that long markdown text wraps properly.
func TestLongMarkdownWrappingInEditor(t *testing.T) {
	content := `# Header
Some really long markdown text that should wrap nicely in the editor pane without being truncated with ellipsis dots`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	m.width = 60
	m.height = 20

	view := m.View().Content

	// Long markdown should not be truncated
	if strings.Contains(view, "Some really long") && strings.Contains(view, "...") {
		if !strings.Contains(view, "ellipsis dots") {
			t.Error("Long markdown is truncated instead of wrapping")
		}
	}
}

// TestPreviewPaneShowsFullMarkdown tests that the preview pane renders
// full markdown content, not just the first character.
func TestPreviewPaneShowsFullMarkdown(t *testing.T) {
	content := `Some really long markdown that should render fully in the preview pane`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	m.width = 100
	m.height = 20
	m.previewMode = PreviewFull

	view := m.View().Content

	// Debug: log the view output
	t.Logf("VIEW OUTPUT:\n%s", view)

	// The preview pane should show more than just "S"
	// Split the view to find the preview section
	lines := strings.Split(view, "\n")

	foundFullText := false
	for _, line := range lines {
		// Strip ANSI codes before checking - v2 renders with escape codes that can split words
		plain := stripAnsiCodes(line)
		if strings.Contains(plain, "really") || strings.Contains(plain, "markdown") {
			foundFullText = true
			break
		}
	}

	if !foundFullText {
		t.Error("Preview pane does not show full markdown content")
	}
}

// TestPreviewPaneWrapsInsteadOfTruncating tests that preview pane wraps
// long content instead of truncating with "...".
func TestPreviewPaneWrapsInsteadOfTruncating(t *testing.T) {
	content := `this is a really long line of markdown that should wrap in preview`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	// Set narrow width to force wrapping
	m.width = 60
	m.height = 20
	m.previewMode = PreviewFull

	view := m.View().Content
	t.Logf("VIEW OUTPUT:\n%s", view)

	// Strip ANSI codes for text content checks
	plainView := stripAnsiCodes(view)

	// Preview should NOT contain "..." truncation
	if strings.Contains(plainView, "...") {
		// Check if it's truly truncation (cutting off content)
		if !strings.Contains(plainView, "wrap in preview") {
			t.Error("Preview pane is truncating content with '...' instead of wrapping")
		}
	}

	// The full text should be visible somewhere (possibly wrapped across lines)
	if !strings.Contains(plainView, "really") || !strings.Contains(plainView, "wrap") {
		t.Error("Preview pane does not show full content - appears to be truncated")
	}
}

// TestPreviewPaneMarkdownNotTruncatedToSingleChar tests that markdown in preview
// is not truncated to just the first character (regression test for "S" bug).
func TestPreviewPaneMarkdownNotTruncatedToSingleChar(t *testing.T) {
	// This tests the specific bug where preview showed only "S" for long markdown
	testCases := []struct {
		name        string
		content     string
		expectWords []string // Words that should appear in preview
	}{
		{
			name:        "single long line",
			content:     `Some really long markdown text`,
			expectWords: []string{"Some", "really", "long"},
		},
		{
			name:        "heading",
			content:     `# This is a heading`,
			expectWords: []string{"This", "heading"},
		},
		{
			name:        "paragraph with bold",
			content:     `This has **bold text** in it`,
			expectWords: []string{"This", "bold", "text"},
		},
		{
			name:        "multiple lines",
			content:     "First line\nSecond line\nThird line",
			expectWords: []string{"First", "Second", "Third"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := document.NewDocument(tc.content)
			if err != nil {
				t.Fatalf("Failed to create document: %v", err)
			}
			m := New(doc)
			m.width = 100
			m.height = 20
			m.previewMode = PreviewFull

			view := m.View().Content

			// Strip ANSI codes - v2 renders with escape codes that can split words
			plainView := stripAnsiCodes(view)
			for _, word := range tc.expectWords {
				if !strings.Contains(plainView, word) {
					t.Errorf("Expected word %q not found in preview output", word)
					t.Logf("VIEW:\n%s", view)
				}
			}
		})
	}
}

// TestMinimalModeLeftJustified tests that minimal mode shows results
// left-justified, not right-justified.
func TestMinimalModeLeftJustified(t *testing.T) {
	content := `x = 42`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	m.width = 100
	m.height = 20
	m.previewMode = PreviewMinimal

	view := m.View().Content

	// In minimal mode, "→ 42" should be left-justified within the preview pane
	// Use the centralized pane width configuration
	sourceWidth, _ := m.GetPaneWidths(m.width)
	previewStart := sourceWidth

	for line := range strings.SplitSeq(view, "\n") {
		if strings.Contains(line, "→") && strings.Contains(line, "42") {
			// Strip ANSI codes before measuring character positions
			plain := stripAnsiCodes(line)
			arrowIdx := strings.Index(plain, "→")

			// Arrow should be near the start of the preview pane (left-justified)
			// Allow a few characters of margin for borders/padding
			if arrowIdx > previewStart+8 {
				// Arrow is too far into the preview pane - it's right-justified
				t.Errorf("Minimal mode result is right-justified (arrow at position %d, preview starts at %d)", arrowIdx, previewStart)
			}
			if arrowIdx < previewStart-5 {
				// Arrow is before the preview pane - something's wrong
				t.Errorf("Arrow appears before preview pane (arrow at %d, preview starts at %d)", arrowIdx, previewStart)
			}
			return
		}
	}
	t.Error("Could not find arrow with result value in output")
}

// TestMinimalModeNarrowerPreview tests that the preview pane is narrower
// in minimal mode compared to full mode.
func TestMinimalModeNarrowerPreview(t *testing.T) {
	content := `x = 42`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	m.width = 100
	m.height = 20

	// Use the centralized configuration to verify width differences
	fullConfig := DefaultPaneWidths[PreviewFull]
	minimalConfig := DefaultPaneWidths[PreviewMinimal]

	// Verify minimal mode has narrower preview (smaller preview percent)
	if minimalConfig.PreviewPercent >= fullConfig.PreviewPercent {
		t.Errorf("Minimal mode preview should be narrower than full mode: minimal=%d%%, full=%d%%",
			minimalConfig.PreviewPercent, fullConfig.PreviewPercent)
	}

	// Verify the actual widths match configuration
	m.previewMode = PreviewFull
	_, fullPreviewWidth := m.GetPaneWidths(m.width)

	m.previewMode = PreviewMinimal
	_, minimalPreviewWidth := m.GetPaneWidths(m.width)

	if minimalPreviewWidth >= fullPreviewWidth {
		t.Errorf("Minimal preview width (%d) should be less than full preview width (%d)",
			minimalPreviewWidth, fullPreviewWidth)
	}

	// Verify expected percentages from configuration
	expectedMinimalPreview := m.width * minimalConfig.PreviewPercent / 100
	if minimalPreviewWidth != expectedMinimalPreview {
		t.Errorf("Minimal preview width mismatch: got %d, expected %d (from config)",
			minimalPreviewWidth, expectedMinimalPreview)
	}
}

// =============================================================================
// Visual Line Alignment Tests
// These tests verify that source and preview panes stay aligned when content
// wraps to multiple visual lines.
// =============================================================================

func TestEditModeShowsPreviewResult(t *testing.T) {
	// Test that when in edit mode, the preview pane still shows the computed result
	// for the cursor line, not blank lines
	content := `gzip_compressed = compress(1 GB, gzip)
lz4_compressed = compress(100 MB, lz4)
zstd_compressed = compress(500 MB, zstd)`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)
	m.width = 120
	m.height = 24
	m.previewMode = PreviewFull

	// Enter edit mode on line 1 (lz4_compressed)
	m.cursorLine = 1
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()

	if m.mode != StateDefault {
		t.Fatalf("Expected StateDefault, got %v", m.mode)
	}

	// Render the view
	view := m.View().Content

	// The preview pane should show the COMPUTED RESULT for lz4_compressed
	// The result is "50 MB" (compress(100 MB, lz4) = 100 * 0.5 = 50 MB)
	// This must appear in the preview pane (right side), not just the source
	if !strings.Contains(view, "50 MB") {
		t.Logf("VIEW:\n%s", view)
		t.Error("Preview should show computed result '50 MB' for lz4_compressed in edit mode")
	}

	// Also verify all three computed results are visible
	// gzip: compress(1 GB, gzip) = 1000 MB * 0.341 = 341 MB
	if !strings.Contains(view, "341 MB") {
		t.Logf("VIEW:\n%s", view)
		t.Error("Preview should show computed result '341 MB' for gzip_compressed")
	}
	// zstd: compress(500 MB, zstd) = 500 * 0.285714 ≈ 143 MB (rounded for display)
	if !strings.Contains(view, "143 MB") {
		t.Logf("VIEW:\n%s", view)
		t.Error("Preview should show computed result containing '143 MB' for zstd_compressed")
	}
}

// TestParseErrorForDisplay tests the error message parsing logic.
func TestOThenBackspaceRendersCorrectly(t *testing.T) {
	// Reproduce: press 'o' to open new line, then immediately press backspace
	// This was causing rendering issues (black bar at top, lost status bar)
	content := `gzip_compressed = compress(1 GB, gzip)
lz4_compressed = compress(100 MB, lz4)`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)
	m.width = 100
	m.height = 24
	m.previewMode = PreviewFull

	// Initial: 2 lines, cursor on line 0
	if got := m.TotalLines(); got != 2 {
		t.Fatalf("Initial: expected 2 lines, got %d", got)
	}

	// Press 'o' - insert line below and enter edit mode
	m.cursorLine = 0
	m.insertLineBelow()
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()

	// Now: 3 lines, cursor on line 1 (the new empty line), in edit mode
	if got := m.TotalLines(); got != 3 {
		t.Fatalf("After 'o': expected 3 lines, got %d", got)
	}
	if m.cursorLine != 1 {
		t.Fatalf("After 'o': expected cursor on line 1, got %d", m.cursorLine)
	}
	if m.mode != StateDefault {
		t.Fatalf("After 'o': expected StateDefault, got %v", m.mode)
	}
	if m.editBuf != "" {
		t.Fatalf("After 'o': expected empty editBuf, got %q", m.editBuf)
	}

	// Simulate backspace on empty line
	// This is the code from handleEditKey for KeyBackspace on empty line
	prevLine := m.cursorLine - 1
	m.saveCurrentLine(false)
	m.deleteLine()
	m.cursorLine = prevLine
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()
	m.cursorCol = len(m.editBuf)

	// After backspace: back to 2 lines, cursor on line 0, in edit mode
	if got := m.TotalLines(); got != 2 {
		t.Errorf("After backspace: expected 2 lines, got %d", got)
	}
	if m.cursorLine != 0 {
		t.Errorf("After backspace: expected cursor on line 0, got %d", m.cursorLine)
	}
	if m.mode != StateDefault {
		t.Errorf("After backspace: expected StateDefault, got %v", m.mode)
	}

	// Render should work without panics and produce valid output
	view := m.View().Content

	// Check for valid structure - should have Source and Results headers
	if !strings.Contains(view, "Source") {
		t.Error("View should contain 'Source' header")
	}
	if !strings.Contains(view, "Results") {
		t.Error("View should contain 'Results' header")
	}

	// Should show the computed results
	if !strings.Contains(view, "341 MB") {
		t.Logf("VIEW:\n%s", view)
		t.Error("View should show gzip result '341 MB'")
	}

	// View should have reasonable number of lines (not collapsed)
	lines := strings.Split(view, "\n")
	if len(lines) < 10 {
		t.Errorf("View has too few lines (%d), something is wrong with rendering", len(lines))
	}
}

// TestNoStatusMessageOnBackspaceDelete verifies that deleting an empty line via
// backspace does NOT set a status message. Status messages cause view height
// changes which lead to bubbletea rendering artifacts (screen "jogging").
func TestNoStatusMessageOnBackspaceDelete(t *testing.T) {
	content := `line_one = 1 + 1
line_two = 2 + 2`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)
	m.width = 100
	m.height = 24
	m.previewMode = PreviewFull

	// Get baseline view height
	baselineView := m.View().Content
	baselineLines := strings.Count(baselineView, "\n")

	// Press 'o' - insert line below and enter edit mode
	m.cursorLine = 0
	m.insertLineBelow()
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()

	// Clear any status message that might have been set
	m.statusMsg = ""
	m.statusIsErr = false

	// Get view height after 'o'
	afterOView := m.View().Content
	afterOLines := strings.Count(afterOView, "\n")
	if afterOLines != baselineLines {
		t.Errorf("View height changed after 'o': baseline=%d, afterO=%d", baselineLines, afterOLines)
	}

	// Simulate backspace on empty line (same logic as handleEditKey)
	prevLine := m.cursorLine - 1
	m.saveCurrentLine(false)
	m.deleteLine()
	m.cursorLine = prevLine
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()
	m.cursorCol = len(m.editBuf)

	// CRITICAL: No status message should be set after line deletion via backspace
	if m.statusMsg != "" {
		t.Errorf("Status message should be empty after backspace delete, got: %q", m.statusMsg)
	}

	// View height should remain constant
	afterBackspaceView := m.View().Content
	afterBackspaceLines := strings.Count(afterBackspaceView, "\n")
	if afterBackspaceLines != baselineLines {
		t.Errorf("View height changed after backspace: baseline=%d, afterBackspace=%d",
			baselineLines, afterBackspaceLines)
	}
}

// TestNoStatusMessageOnDDDelete verifies that deleting a line via 'dd' does NOT
// set a status message. The line is yanked to the buffer but no message is shown.
func TestNoStatusMessageOnDDDelete(t *testing.T) {
	content := `line_one = 1 + 1
line_two = 2 + 2
line_three = 3 + 3`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)
	m.width = 100
	m.height = 24

	// Clear any status message
	m.statusMsg = ""
	m.statusIsErr = false

	// Get baseline view height
	baselineView := m.View().Content
	baselineLines := strings.Count(baselineView, "\n")

	// Press 'dd' - delete line (this calls deleteLine())
	m.cursorLine = 1
	m.deleteLine()

	// CRITICAL: No status message should be set after 'dd'
	// (Note: 'yy' does set "Line yanked", but 'dd' should not)
	if m.statusMsg != "" {
		t.Errorf("Status message should be empty after 'dd', got: %q", m.statusMsg)
	}

	// View height should remain constant
	afterDDView := m.View().Content
	afterDDLines := strings.Count(afterDDView, "\n")
	if afterDDLines != baselineLines {
		t.Errorf("View height changed after 'dd': baseline=%d, afterDD=%d",
			baselineLines, afterDDLines)
	}
}

// TestViewHeightConsistency verifies that View() always returns the same number
// of lines regardless of model state. This is critical for bubbletea rendering -
// if line count changes between renders, the terminal can show artifacts like
// missing headers or truncated status bars.
//
// See: https://charm.land/bubbletea/v2/issues/1004
func TestViewHeightConsistency(t *testing.T) {
	content := `gzip_compressed = compress(1 GB, gzip)
lz4_compressed = compress(100 MB, lz4)
zstd_compressed = compress(500 MB, zstd)`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 100
	m.height = 30
	m.previewMode = PreviewFull

	// Get baseline line count
	baselineView := m.View().Content
	baselineLines := strings.Count(baselineView, "\n")

	// Test various state changes that previously caused height inconsistencies
	testCases := []struct {
		name   string
		mutate func(*Model)
	}{
		{
			name: "cursor on calc line",
			mutate: func(m *Model) {
				m.cursorLine = 0
			},
		},
		{
			name: "cursor on empty line after insert",
			mutate: func(m *Model) {
				m.cursorLine = 1
				m.insertLineBelow()
				m.cursorLine = 2 // now on empty line
			},
		},
		{
			name: "edit mode on calc line",
			mutate: func(m *Model) {
				m.cursorLine = 0
				// User is always able to edit - load editBuf
				m.loadCurrentLineIntoEditBuffer()
			},
		},
		{
			name: "edit mode on empty line",
			mutate: func(m *Model) {
				m.cursorLine = 1
				m.insertLineBelow()
				m.cursorLine = 2
				// User is always able to edit - load editBuf
				m.loadCurrentLineIntoEditBuffer()
			},
		},
		{
			name: "after deleting a line",
			mutate: func(m *Model) {
				m.cursorLine = 1
				m.insertLineBelow() // add line
				m.cursorLine = 2
				m.deleteLine() // delete it
			},
		},
		{
			name: "cursor past end of results",
			mutate: func(m *Model) {
				// Add empty lines at end
				m.cursorLine = m.TotalLines() - 1
				m.insertLineBelow()
				m.insertLineBelow()
				m.cursorLine = m.TotalLines() - 1
			},
		},
		{
			name: "normal mode after edit",
			mutate: func(m *Model) {
				m.cursorLine = 0
				// User is always able to edit - load editBuf
				m.loadCurrentLineIntoEditBuffer()
				m.saveCurrentLine(true)
			},
		},
		{
			name: "status message set",
			mutate: func(m *Model) {
				m.statusMsg = "Test status message"
			},
		},
		{
			name: "status message cleared",
			mutate: func(m *Model) {
				m.statusMsg = ""
			},
		},
		{
			name: "globals expanded",
			mutate: func(m *Model) {
				m.globalsExpanded = true
			},
		},
		{
			name: "globals collapsed",
			mutate: func(m *Model) {
				m.globalsExpanded = false
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create fresh model for each test
			doc, _ := document.NewDocument(content)
			m := New(doc)
			m.width = 100
			m.height = 30
			m.previewMode = PreviewFull

			// Apply mutation
			tc.mutate(&m)

			// Get view and count lines
			view := m.View().Content
			lineCount := strings.Count(view, "\n")

			if lineCount != baselineLines {
				t.Errorf("View height changed: baseline=%d, got=%d (diff=%d)",
					baselineLines, lineCount, lineCount-baselineLines)
				// Show last 5 lines of each view to compare footer
				baselineViewLines := strings.Split(baselineView, "\n")
				viewLines := strings.Split(view, "\n")
				t.Logf("Baseline last 5 lines:")
				for i := max(0, len(baselineViewLines)-5); i < len(baselineViewLines); i++ {
					t.Logf("  [%d]: %q", i, baselineViewLines[i])
				}
				t.Logf("Current last 5 lines:")
				for i := max(0, len(viewLines)-5); i < len(viewLines); i++ {
					t.Logf("  [%d]: %q", i, viewLines[i])
				}
			}
		})
	}
}

func TestViewHeightWithErrors(t *testing.T) {
	// Document with and without errors should have same view height
	goodContent := `x = 1 + 1`
	badContent := `x = undefined_var`

	goodDoc, _ := document.NewDocument(goodContent)
	badDoc, _ := document.NewDocument(badContent)

	goodModel := New(goodDoc)
	goodModel.width = 80
	goodModel.height = 24
	goodModel.previewMode = PreviewFull

	badModel := New(badDoc)
	badModel.width = 80
	badModel.height = 24
	badModel.previewMode = PreviewFull

	goodView := goodModel.View().Content
	badView := badModel.View().Content

	goodLines := strings.Count(goodView, "\n")
	badLines := strings.Count(badView, "\n")

	if goodLines != badLines {
		t.Errorf("View height differs: good=%d, bad=%d (with error)", goodLines, badLines)
		t.Logf("Good view last 5 lines:")
		for _, line := range strings.Split(goodView, "\n")[max(0, goodLines-5):] {
			t.Logf("  %q", line)
		}
		t.Logf("Bad view last 5 lines:")
		for _, line := range strings.Split(badView, "\n")[max(0, badLines-5):] {
			t.Logf("  %q", line)
		}
	}
}

// Test export functionality
