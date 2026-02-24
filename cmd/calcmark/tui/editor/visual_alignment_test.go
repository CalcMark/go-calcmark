package editor

// visual_alignment_test.go — Source-to-visual mapping, scroll sync, alignment cache.

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

func TestSourceToVisualMapping_BasicCase(t *testing.T) {
	// Simple case: no wrapping, each source line maps to one visual line
	// Note: trailing newline creates an empty 4th line
	doc, _ := document.NewDocument("x = 10\ny = 20\nz = 30\n")
	m := New(doc)
	m.width = 100
	m.height = 24

	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	totalSourceLines := m.TotalLines()
	t.Logf("Document has %d source lines", totalSourceLines)

	// With no wrapping, sourceToVisual should map each source line to sequential visual lines
	for i := range totalSourceLines {
		visualIdx, ok := aligned.sourceToVisual[i]
		if !ok {
			t.Errorf("sourceToVisual missing entry for source line %d", i)
			continue
		}
		if visualIdx != i {
			t.Errorf("Source line %d: expected visual index %d, got %d", i, i, visualIdx)
		}
	}

	// With no wrapping, visual lines should equal source lines
	if len(aligned.sourceLines) != totalSourceLines {
		t.Errorf("Expected %d source lines, got %d", totalSourceLines, len(aligned.sourceLines))
	}
	if len(aligned.previewLines) != totalSourceLines {
		t.Errorf("Expected %d preview lines, got %d", totalSourceLines, len(aligned.previewLines))
	}

	// Verify source and preview counts match (critical invariant)
	if len(aligned.sourceLines) != len(aligned.previewLines) {
		t.Errorf("Source lines (%d) != Preview lines (%d)",
			len(aligned.sourceLines), len(aligned.previewLines))
	}
}

func TestSourceToVisualMapping_PreviewWraps(t *testing.T) {
	// Create a document where preview content will wrap
	// Use a narrow preview width to force wrapping
	doc, _ := document.NewDocument("a = 1\nb = 2\n")
	m := New(doc)
	m.width = 60 // Narrow total width
	m.height = 24

	// Force a very narrow preview to cause wrapping
	// We need to test with actual preview content that wraps
	// The preview shows "a  1" and "b  2" which are short
	// Let's use a document with longer variable names

	doc2, _ := document.NewDocument("short = 1\nthis_is_a_very_long_variable_name = 2\n")
	m2 := New(doc2)
	m2.width = 50 // Narrow
	m2.height = 24

	leftWidth, rightWidth := m2.GetPaneWidths(m2.width)
	aligned := m2.computeAlignedPanes(leftWidth, rightWidth)

	t.Logf("Source lines: %d, Preview lines: %d", len(aligned.sourceLines), len(aligned.previewLines))
	t.Logf("sourceToVisual map: %v", aligned.sourceToVisual)

	for i, sl := range aligned.sourceLines {
		t.Logf("  Source[%d]: lineNum=%d, sourceLineIdx=%d, isPadding=%v, content=%q",
			i, sl.lineNum, sl.sourceLineIdx, sl.isPadding, sl.content)
	}

	// Key invariant: source lines and preview lines must have same count
	if len(aligned.sourceLines) != len(aligned.previewLines) {
		t.Errorf("Source lines (%d) != Preview lines (%d) - alignment broken",
			len(aligned.sourceLines), len(aligned.previewLines))
	}

	// Verify sourceToVisual contains entries for all source lines
	sourceLineCount := m2.TotalLines()
	for i := range sourceLineCount {
		if _, ok := aligned.sourceToVisual[i]; !ok {
			t.Errorf("sourceToVisual missing entry for source line %d", i)
		}
	}
}

func TestSourceToVisualMapping_WithPaddingLines(t *testing.T) {
	// Create a scenario where preview wraps more than source
	// This forces padding lines to be added to source pane

	// We need markdown content that renders to multiple lines
	// A long heading or text line should do it
	content := `# Short heading
x = 100`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 40 // Very narrow to force wrapping
	m.height = 24

	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	t.Logf("Total source lines in doc: %d", m.TotalLines())
	t.Logf("Visual source lines: %d, Visual preview lines: %d",
		len(aligned.sourceLines), len(aligned.previewLines))

	for i, sl := range aligned.sourceLines {
		t.Logf("  Source[%d]: lineNum=%d, idx=%d, padding=%v, wrapped=%v, content=%q",
			i, sl.lineNum, sl.sourceLineIdx, sl.isPadding, sl.isWrapped, sl.content)
	}

	// Verify alignment
	if len(aligned.sourceLines) != len(aligned.previewLines) {
		t.Errorf("Source lines (%d) != Preview lines (%d)",
			len(aligned.sourceLines), len(aligned.previewLines))
	}

	// If there are padding lines in source, they should have isPadding=true
	paddingCount := 0
	for _, sl := range aligned.sourceLines {
		if sl.isPadding {
			paddingCount++
		}
	}
	t.Logf("Padding lines in source: %d", paddingCount)

	// Verify the mapping skips padding appropriately
	// Source line 0 should map to visual line 0
	if idx, ok := aligned.sourceToVisual[0]; ok {
		if idx != 0 {
			t.Errorf("Source line 0 should map to visual line 0, got %d", idx)
		}
	}
}

func TestScrollSyncWithPadding(t *testing.T) {
	// Test that when cursor is on a line with padding below it,
	// both panes compute the same scroll offset

	content := `line1 = 1
line2_with_a_much_longer_name_that_might_wrap = 2
line3 = 3
line4 = 4
line5 = 5`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 50
	m.height = 10 // Small height to test scrolling

	// Move cursor to line 2 (the long one)
	m.cursorLine = 1

	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	// Get the visual line index for cursor
	cursorVisualLine, ok := aligned.sourceToVisual[m.cursorLine]
	if !ok {
		t.Fatalf("No visual mapping for cursor line %d", m.cursorLine)
	}

	t.Logf("Cursor on source line %d, visual line %d", m.cursorLine, cursorVisualLine)
	t.Logf("sourceToVisual: %v", aligned.sourceToVisual)

	// The key test: when we render, both panes should use the same visual scroll offset
	// We can't easily test the internal scroll calculation, but we can verify
	// that the sourceToVisual mapping is monotonically increasing
	lastVisual := -1
	for srcLine := 0; srcLine < m.TotalLines(); srcLine++ {
		visualLine, ok := aligned.sourceToVisual[srcLine]
		if !ok {
			t.Errorf("Missing mapping for source line %d", srcLine)
			continue
		}
		if visualLine <= lastVisual {
			t.Errorf("sourceToVisual not monotonically increasing: line %d maps to %d, but previous was %d",
				srcLine, visualLine, lastVisual)
		}
		lastVisual = visualLine
	}
}

func TestVisualLineCalculation_Deterministic(t *testing.T) {
	// Test with exact, predictable dimensions
	// Source pane: 40 chars wide (after line numbers)
	// Preview pane: 30 chars wide
	// This allows us to predict exactly how content wraps

	content := `x = 1
this_is_a_longer_line = 2
z = 3`

	doc, _ := document.NewDocument(content)
	m := New(doc)

	// Set exact dimensions
	// With width=80, PreviewFull gives 55% source (44) and 45% preview (36)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	t.Logf("Pane widths: source=%d, preview=%d", leftWidth, rightWidth)

	// Calculate expected source content width (after line numbers)
	// Line number width is 4, plus 2 for spacing
	sourceContentWidth := leftWidth - 4 - 2
	t.Logf("Source content width: %d", sourceContentWidth)

	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	// Log all visual lines for debugging
	t.Log("Source visual lines:")
	for i, sl := range aligned.sourceLines {
		t.Logf("  [%d] srcIdx=%d lineNum=%d wrap=%v pad=%v content=%q",
			i, sl.sourceLineIdx, sl.lineNum, sl.isWrapped, sl.isPadding, sl.content)
	}

	t.Log("Preview visual lines:")
	for i, pl := range aligned.previewLines {
		t.Logf("  [%d] srcNum=%d content=%q", i, pl.sourceLineNum, pl.content)
	}

	t.Logf("sourceToVisual: %v", aligned.sourceToVisual)

	// Verify the mapping
	// Line 0 "x = 1" is short, should be visual line 0
	if v, ok := aligned.sourceToVisual[0]; !ok || v != 0 {
		t.Errorf("Source line 0 should map to visual 0, got %v (ok=%v)", v, ok)
	}

	// Line 1 "this_is_a_longer_line = 2" - check if it wraps
	line1 := "this_is_a_longer_line = 2"
	line1VisualWidth := len(line1) // ASCII, so len == visual width
	if line1VisualWidth > sourceContentWidth {
		t.Logf("Line 1 (%d chars) should wrap at width %d", line1VisualWidth, sourceContentWidth)
	}

	// Source line 2 should map to visual index after line 1's visual lines
	v1 := aligned.sourceToVisual[1]
	v2 := aligned.sourceToVisual[2]
	if v2 <= v1 {
		t.Errorf("Source line 2 visual (%d) should be > source line 1 visual (%d)", v2, v1)
	}
}

func TestScrollOffset_Deterministic(t *testing.T) {
	// Test scroll offset calculation with exact dimensions
	// Create a document with enough lines to require scrolling

	content := `line0 = 0
line1 = 1
line2 = 2
line3 = 3
line4 = 4
line5 = 5
line6 = 6
line7 = 7
line8 = 8
line9 = 9`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 80
	m.height = 10 // Small height to force scrolling
	m.previewMode = PreviewFull

	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	t.Logf("Total visual lines: %d", len(aligned.sourceLines))
	t.Logf("Visible height: ~%d (height=%d minus headers/footers)", m.height-6, m.height)

	// Test cursor at different positions
	testCases := []struct {
		cursorLine         int
		expectedVisualLine int
	}{
		{0, 0},
		{1, 1},
		{5, 5},
		{9, 9},
	}

	for _, tc := range testCases {
		m.cursorLine = tc.cursorLine
		visualLine, ok := aligned.sourceToVisual[tc.cursorLine]
		if !ok {
			t.Errorf("Cursor line %d: no visual mapping", tc.cursorLine)
			continue
		}
		if visualLine != tc.expectedVisualLine {
			t.Errorf("Cursor line %d: expected visual %d, got %d",
				tc.cursorLine, tc.expectedVisualLine, visualLine)
		}
	}
}

func TestScrollOffset_WithWrapping(t *testing.T) {
	// Test scroll calculation when some lines wrap
	// Use very narrow width to force wrapping

	content := `short = 1
this_is_a_very_long_variable_name_that_will_definitely_wrap = 2
another_short = 3`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 60
	m.height = 20
	m.previewMode = PreviewFull

	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	t.Logf("Widths: left=%d, right=%d", leftWidth, rightWidth)

	// Source content width after line numbers
	sourceContentWidth := leftWidth - 4 - 2
	t.Logf("Source content width: %d", sourceContentWidth)

	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	// Log the visual structure
	t.Log("Visual line structure:")
	for i, sl := range aligned.sourceLines {
		t.Logf("  visual[%d]: srcIdx=%d wrap=%v pad=%v len=%d content=%q",
			i, sl.sourceLineIdx, sl.isWrapped, sl.isPadding, len(sl.content), sl.content)
	}

	// The long line (source line 1) should wrap
	line1Content := "this_is_a_very_long_variable_name_that_will_definitely_wrap = 2"
	expectedWraps := (len(line1Content) + sourceContentWidth - 1) / sourceContentWidth
	t.Logf("Line 1 length: %d, expected wraps: %d", len(line1Content), expectedWraps)

	// Count visual lines for source line 1
	line1VisualCount := 0
	for _, sl := range aligned.sourceLines {
		if sl.sourceLineIdx == 1 && !sl.isPadding {
			line1VisualCount++
		}
	}
	t.Logf("Line 1 actual visual lines: %d", line1VisualCount)

	// Source line 2 should start at visual index = line0 visuals + line1 visuals
	v0 := aligned.sourceToVisual[0]
	v1 := aligned.sourceToVisual[1]
	v2 := aligned.sourceToVisual[2]

	t.Logf("Visual indices: line0=%d, line1=%d, line2=%d", v0, v1, v2)

	// Verify gaps match wrapping
	gapBetween1And2 := v2 - v1
	if gapBetween1And2 < line1VisualCount {
		t.Errorf("Gap between line1 and line2 (%d) should be >= line1 visual count (%d)",
			gapBetween1And2, line1VisualCount)
	}
}

func TestScrollOffset_VisualVsSource(t *testing.T) {
	// The bug: scrollOffset is set using source line indices
	// but used as visual line indices in rendering.
	// When there are wrapped lines, these diverge!

	content := `short = 1
this_is_a_long_line_that_will_wrap = 2
short = 3
this_is_another_long_line_that_wraps = 4
short = 5`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 50  // Narrow to cause wrapping
	m.height = 10 // Short to trigger scrolling
	m.previewMode = PreviewFull

	leftWidth, rightWidth := m.GetPaneWidths(m.width)

	// Check initial visual structure
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)
	t.Logf("Visual structure (%d visual lines for %d source lines):",
		len(aligned.sourceLines), m.TotalLines())
	for i, sl := range aligned.sourceLines {
		t.Logf("  Visual[%d]: srcIdx=%d, content=%q",
			i, sl.sourceLineIdx, truncate(sl.content, 30))
	}

	// The key insight: if we have wrapping, visual line count > source line count
	// scrollOffset is set based on source lines but used as visual lines
	t.Logf("scrollOffset before navigation: %d", m.scrollOffset)

	// Navigate down - this sets scrollOffset based on cursorLine (source)
	for range 4 {
		m.cursorLine++
		// Simulate the moveCursor scroll adjustment
		visibleHeight := m.height - 6
		if m.cursorLine >= m.scrollOffset+visibleHeight {
			m.scrollOffset = m.cursorLine - visibleHeight + 1
		}
	}

	t.Logf("After navigating to source line %d: scrollOffset=%d",
		m.cursorLine, m.scrollOffset)

	// Now check: what visual line does this source line map to?
	visualIdx := aligned.sourceToVisual[m.cursorLine]
	t.Logf("Source line %d maps to visual line %d", m.cursorLine, visualIdx)

	// THE BUG: if scrollOffset=3 (source), but source line 3 is visual line 5,
	// then rendering will start at visual line 3 (wrong!)
	// It should start at a visual line that makes the cursor visible

	// In a correct implementation, scrollOffset should be in visual space
	// OR the render should convert scrollOffset from source to visual

	// For now, just verify that scrollOffset stays sensible
	if m.scrollOffset > visualIdx {
		t.Errorf("BUG: scrollOffset (%d) > cursor visual line (%d)",
			m.scrollOffset, visualIdx)
	}

	// Also verify scrollOffset isn't larger than visual line count
	if m.scrollOffset >= len(aligned.sourceLines) {
		t.Errorf("scrollOffset (%d) >= visual line count (%d)",
			m.scrollOffset, len(aligned.sourceLines))
	}
}

func TestPaneAlignment_ExactDimensions(t *testing.T) {
	// Test with exact dimensions to verify pixel-perfect alignment

	testCases := []struct {
		name           string
		content        string
		width          int
		height         int
		expectedVisual int // expected total visual lines
	}{
		{
			name:           "simple no wrap",
			content:        "a = 1\nb = 2",
			width:          100,
			height:         24,
			expectedVisual: 2,
		},
		{
			name:           "single line wraps at narrow width",
			content:        "abcdefghij = 12345", // 18 chars
			width:          40,                   // Very narrow
			height:         24,
			expectedVisual: -1, // Calculate based on actual wrapping
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, _ := document.NewDocument(tc.content)
			m := New(doc)
			m.width = tc.width
			m.height = tc.height

			leftWidth, rightWidth := m.GetPaneWidths(m.width)
			aligned := m.computeAlignedPanes(leftWidth, rightWidth)

			t.Logf("Width=%d -> left=%d, right=%d", tc.width, leftWidth, rightWidth)
			t.Logf("Source lines: %d, Preview lines: %d",
				len(aligned.sourceLines), len(aligned.previewLines))

			// Critical: counts must match
			if len(aligned.sourceLines) != len(aligned.previewLines) {
				t.Errorf("Alignment broken: source=%d, preview=%d",
					len(aligned.sourceLines), len(aligned.previewLines))
			}

			if tc.expectedVisual > 0 && len(aligned.sourceLines) != tc.expectedVisual {
				t.Errorf("Expected %d visual lines, got %d",
					tc.expectedVisual, len(aligned.sourceLines))
			}

			// Log structure for debugging
			for i := range aligned.sourceLines {
				sl := aligned.sourceLines[i]
				pl := aligned.previewLines[i]
				t.Logf("  [%d] src=%q | preview=%q",
					i, sl.content, pl.content)
			}
		})
	}
}

func TestAlignedPanesCountsMatch(t *testing.T) {
	// Critical invariant: source and preview lines must always have same count
	testCases := []struct {
		name    string
		content string
		width   int
	}{
		{
			name:    "simple calc",
			content: "x = 10\ny = 20\n",
			width:   80,
		},
		{
			name:    "narrow width",
			content: "x = 10\ny = 20\n",
			width:   30,
		},
		{
			name:    "mixed content",
			content: "# Header\nx = 10\nSome text\ny = x * 2\n",
			width:   60,
		},
		{
			name:    "long variable names",
			content: "very_long_variable_name_here = 12345\nanother_long_one = 67890\n",
			width:   40,
		},
		{
			name:    "empty lines",
			content: "x = 1\n\ny = 2\n",
			width:   80,
		},
		{
			name: "compression scenario - long function calls",
			content: `# Use in calculations
storage_savings = 10 GB - compress(10 GB, gzip)
compressed_transfer = transfer_time(compress(1 GB, lz4), global, gigabit)`,
			width: 80,
		},
		{
			name: "very narrow compression scenario",
			content: `# Use in calculations
storage_savings = 10 GB - compress(10 GB, gzip)
compressed_transfer = transfer_time(compress(1 GB, lz4), global, gigabit)`,
			width: 50,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := document.NewDocument(tc.content)
			if err != nil {
				t.Fatalf("Failed to create document: %v", err)
			}

			m := New(doc)
			m.width = tc.width
			m.height = 24

			leftWidth, rightWidth := m.GetPaneWidths(m.width)
			aligned := m.computeAlignedPanes(leftWidth, rightWidth)

			if len(aligned.sourceLines) != len(aligned.previewLines) {
				t.Errorf("Source lines (%d) != Preview lines (%d)",
					len(aligned.sourceLines), len(aligned.previewLines))

				t.Log("Source lines:")
				for i, sl := range aligned.sourceLines {
					t.Logf("  [%d] idx=%d padding=%v content=%q",
						i, sl.sourceLineIdx, sl.isPadding, sl.content)
				}
				t.Log("Preview lines:")
				for i, pl := range aligned.previewLines {
					t.Logf("  [%d] srcNum=%d content=%q",
						i, pl.sourceLineNum, pl.content)
				}
			}
		})
	}
}

// Tests for bug: Navigation broken after pressing 'o' to insert a line
// Cursor highlights wrong visual line after insert operations

func TestAlignedModelCache(t *testing.T) {
	content := `# Header
x = 10
y = 20`
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// First call should compute fresh
	aligned1 := m.GetAlignedModel(40, 40)
	if aligned1 == nil {
		t.Fatal("GetAlignedModel returned nil")
	}

	// Second call with same params should return cached
	aligned2 := m.GetAlignedModel(40, 40)
	if aligned2 != aligned1 {
		t.Error("Cache miss: expected same pointer for identical inputs")
	}

	// Change cursor - should invalidate cache
	m.cursorLine = 1
	aligned3 := m.GetAlignedModel(40, 40)
	if aligned3 == aligned1 {
		t.Error("Cache should have been invalidated when cursor changed")
	}

	// Different width - should recompute
	aligned4 := m.GetAlignedModel(50, 40)
	if aligned4 == aligned3 {
		t.Error("Cache should have been invalidated when width changed")
	}

	// Back to same params as aligned3 should still be fresh (cursor changed)
	m.cursorLine = 1 // same as before
	aligned5 := m.GetAlignedModel(40, 40)
	// This is a fresh computation since we went to different width in between
	// The cache now has width 50, not 40
	if aligned5 == aligned4 {
		t.Error("Different widths should produce different cache entries")
	}
}
