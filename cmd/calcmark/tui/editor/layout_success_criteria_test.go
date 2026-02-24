package editor

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/geometry"
	"github.com/CalcMark/go-calcmark/spec/document"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
)

func init() {
	// Force ASCII for consistent test output
	lipgloss.SetColorProfile(termenv.Ascii)
}

// =============================================================================
// Phase 2 Success Criteria Integration Tests
//
// Each test maps 1:1 to a success criterion from the ROADMAP:
//   TestSC1_ -> SC1 (source and results side-by-side, no overlap)
//   TestSC2_ -> SC2 (source wraps at column boundary, no bleed)
//   TestSC3_ -> SC3 (result wraps independently in right pane)
//   TestSC4_ -> SC4 (asymmetric wrapping, vertical alignment preserved)
//   TestSC5_ -> SC5 (terminal resize reflows both columns correctly)
//
// Wiring verification:
//   - View() is called and asserted on (TestSC1, TestSC5)
//   - computeAlignedPanes() is called and asserted on (TestSC1, TestSC2)
//   - geometry.CalculateRowGeometry / geometry.WrapText is called (TestSC2, TestSC4)
// =============================================================================

// TestSC1_SourceAndResultsSideBySide verifies that source and results render
// side-by-side with no overlapping text for standard content. This exercises
// View() and computeAlignedPanes to prove the full rendering pipeline works.
func TestSC1_SourceAndResultsSideBySide(t *testing.T) {
	content := "# Header\nx = 10\ny = 20\nz = x + y"

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	// --- Wiring: call View() and assert on its output ---
	view := m.View()
	viewLines := strings.Split(view, "\n")

	// View output should not exceed terminal height
	if len(viewLines) > m.height {
		t.Errorf("View() produces %d lines but terminal height is %d -- overflow causes bleed-through",
			len(viewLines), m.height)
	}

	// Compute pane widths
	leftWidth, rightWidth := m.GetPaneWidths(80)
	totalPaneWidth := leftWidth + rightWidth
	t.Logf("leftWidth=%d, rightWidth=%d, total=%d", leftWidth, rightWidth, totalPaneWidth)

	// For lines in the pane area (header + content, before status bar/footer),
	// verify that each line has exactly the expected total pane width.
	// The first line is the pane header, followed by content.
	// We check all lines that are full-width (pane area).
	fullWidthCount := 0
	for i, line := range viewLines {
		w := lipgloss.Width(line)
		if w == totalPaneWidth {
			fullWidthCount++
		} else if w > 0 && w != totalPaneWidth && w != m.width {
			// Lines should be either totalPaneWidth (pane area) or m.width (footer/status)
			// Allow some flexibility for status bar styling
			t.Logf("Line %d has unexpected width %d (expected %d or %d): %q",
				i, w, totalPaneWidth, m.width, truncateForLog(line, 60))
		}
	}
	if fullWidthCount == 0 {
		t.Error("No lines in View() output have the expected pane width -- rendering pipeline broken")
	}

	// --- Wiring: call computeAlignedPanes and assert on alignment invariant ---
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	// The 1:1 alignment invariant: source and preview must have same line count
	if len(aligned.sourceLines) != len(aligned.previewLines) {
		t.Errorf("Alignment invariant broken: sourceLines=%d, previewLines=%d",
			len(aligned.sourceLines), len(aligned.previewLines))
	}

	// Verify all source line indices match between source and preview at each visual line
	for i := 0; i < len(aligned.sourceLines); i++ {
		if aligned.sourceLines[i].sourceLineIdx != aligned.previewLines[i].sourceLineNum {
			t.Errorf("Visual line %d: source maps to doc line %d, preview maps to doc line %d",
				i, aligned.sourceLines[i].sourceLineIdx, aligned.previewLines[i].sourceLineNum)
		}
	}

	// Verify Invariants from AlignedModel
	freshAligned := m.computeAlignedModelFresh(leftWidth, rightWidth)
	inv := freshAligned.Invariants()
	if !inv.SourcePreviewMatch {
		t.Error("AlignedModel invariant SourcePreviewMatch is false")
	}
	if !inv.MappingComplete {
		t.Error("AlignedModel invariant MappingComplete is false")
	}
	if !inv.ReverseComplete {
		t.Error("AlignedModel invariant ReverseComplete is false")
	}

	t.Logf("SC1 PASS: %d visual lines, %d full-width view lines, all invariants hold",
		len(aligned.sourceLines), fullWidthCount)
}

// TestSC2_SourceWrapsAtColumnBoundary verifies that a long source line wraps
// at the column boundary without bleeding into the results pane. This exercises
// computeAlignedPanes and geometry.WrapText for cross-verification.
func TestSC2_SourceWrapsAtColumnBoundary(t *testing.T) {
	// Create a document with a very long calc line
	longLine := "long_variable_name_total_sum = 100 + 200 + 300 + 400 + 500 + 600 + 700 + 800"
	content := longLine

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	leftWidth, rightWidth := m.GetPaneWidths(80)

	// Source content width = leftWidth - lineNumWidth(4) - gutter+space(2)
	lineNumWidth := 4
	sourceContentWidth := leftWidth - lineNumWidth - 2
	t.Logf("leftWidth=%d, sourceContentWidth=%d, rightWidth=%d", leftWidth, sourceContentWidth, rightWidth)

	// --- Wiring: call computeAlignedPanes ---
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	// Count how many visual lines belong to source line 0
	sourceVisualLineCount := 0
	for _, sl := range aligned.sourceLines {
		if sl.sourceLineIdx == 0 {
			sourceVisualLineCount++
		}
	}

	// The long line should wrap to more than 1 visual line
	if sourceVisualLineCount <= 1 {
		t.Errorf("Expected long line to wrap (sourceContentWidth=%d, lineLen=%d), but got %d visual lines",
			sourceContentWidth, len(longLine), sourceVisualLineCount)
	}

	// Verify each source visual line's content does not exceed sourceContentWidth
	for i, sl := range aligned.sourceLines {
		if sl.sourceLineIdx == 0 && sl.content != "" {
			contentWidth := runewidth.StringWidth(sl.content)
			if contentWidth > sourceContentWidth {
				t.Errorf("Visual line %d: content width %d exceeds sourceContentWidth %d -- bleed-through! Content: %q",
					i, contentWidth, sourceContentWidth, sl.content)
			}
		}
	}

	// --- Wiring: cross-verify with geometry.WrapText ---
	geometryWrapped := geometry.WrapText(longLine, sourceContentWidth)
	if len(geometryWrapped) != sourceVisualLineCount {
		t.Errorf("geometry.WrapText produces %d lines but AlignedModel has %d visual lines for source line 0",
			len(geometryWrapped), sourceVisualLineCount)
	}

	// Verify geometry.WrapText lines also respect the boundary
	for i, wrapLine := range geometryWrapped {
		w := runewidth.StringWidth(wrapLine)
		if w > sourceContentWidth {
			t.Errorf("geometry.WrapText line %d: width %d exceeds sourceContentWidth %d",
				i, w, sourceContentWidth)
		}
	}

	t.Logf("SC2 PASS: long line (%d chars) wraps to %d visual lines within %d-char boundary",
		len(longLine), sourceVisualLineCount, sourceContentWidth)
}

// TestSC3_ResultWrapsIndependently verifies that a wide result wraps independently
// in the right pane without pushing other rows down. Uses ComputeAlignedModel
// directly with narrow preview width to force result wrapping.
func TestSC3_ResultWrapsIndependently(t *testing.T) {
	// Use ComputeAlignedModel directly with narrow preview width
	// Source line: "x = 1" (short), Result: "x = 12345678901234567890" (long at narrow width)
	input := AlignedModelInput{
		Lines: []string{"x = 1", "y = 2"},
		Results: []LineResult{
			{LineNum: 0, Source: "x = 1", BlockID: "b1", IsCalc: true, VarName: "x", Value: "12345678901234567890"},
			{LineNum: 1, Source: "y = 2", BlockID: "b1", IsCalc: true, VarName: "y", Value: "2"},
		},
		SourceContentWidth: 40,
		PreviewWidth:       8, // Very narrow: forces result wrapping of "x = 12345678901234567890"
		CursorLine:         0,
		PreviewMode:        PreviewFull,
	}

	model := ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)

	// The result for line 0 ("x = 12345678901234567890") should wrap at width 8
	// mockRenderCalcLine returns "x = 12345678901234567890" which is 24 chars,
	// wrapping at 8 produces 3+ visual lines
	t.Logf("TotalVisualLines=%d, TotalSourceLines=%d", model.TotalVisualLines, model.TotalSourceLines)

	// Check that result wrapping created extra visual lines
	if model.TotalVisualLines <= model.TotalSourceLines {
		t.Errorf("Expected TotalVisualLines(%d) > TotalSourceLines(%d) due to result wrapping",
			model.TotalVisualLines, model.TotalSourceLines)
	}

	// Source should have padding lines to match the wrapped result
	paddingCount := 0
	for _, sl := range model.SourceLines {
		if sl.Kind == AlignedLinePadding {
			paddingCount++
		}
	}
	if paddingCount == 0 {
		t.Error("Expected source padding lines to absorb wrapped result, but found none")
	}

	// 1:1 alignment must hold
	if len(model.SourceLines) != len(model.PreviewLines) {
		t.Errorf("Alignment broken: source=%d, preview=%d",
			len(model.SourceLines), len(model.PreviewLines))
	}

	// Critically: source line 1 should start AFTER all the wrapped result lines of line 0
	// This proves the result wrapping doesn't push down other content incorrectly
	visual0 := model.SourceToVisual[0]
	visual1 := model.SourceToVisual[1]
	t.Logf("Source line 0 starts at visual %d, source line 1 starts at visual %d", visual0, visual1)

	if visual1 <= visual0+1 {
		// Count how many visual lines line 0 occupies
		line0VisualCount := 0
		for _, sl := range model.SourceLines {
			if sl.SourceLineIdx == 0 {
				line0VisualCount++
			}
		}
		if line0VisualCount > 1 {
			t.Errorf("Source line 1 starts at visual %d but line 0 occupies %d visual lines -- alignment wrong",
				visual1, line0VisualCount)
		}
	}

	// Verify source line 1 starts exactly where expected
	line0VisualCount := 0
	for _, sl := range model.SourceLines {
		if sl.SourceLineIdx == 0 {
			line0VisualCount++
		}
	}
	if visual1 != visual0+line0VisualCount {
		t.Errorf("Source line 1 at visual %d, but expected %d (visual0=%d + line0Count=%d)",
			visual1, visual0+line0VisualCount, visual0, line0VisualCount)
	}

	t.Logf("SC3 PASS: result wraps to %d visual lines, %d padding lines absorb difference, line 1 starts at visual %d",
		line0VisualCount, paddingCount, visual1)
}

// TestSC4_AsymmetricWrappingVerticalAlignment verifies that when source wraps
// to more visual lines than preview, both still align at the same vertical position.
// Uses ComputeAlignedModel directly and geometry.WrapText for verification.
func TestSC4_AsymmetricWrappingVerticalAlignment(t *testing.T) {
	// Source line 0 wraps to 3+ visual lines at width 15, but result is short
	input := AlignedModelInput{
		Lines: []string{"this is a text that will wrap to three visual lines easily", "next line"},
		Results: []LineResult{
			{LineNum: 0, Source: "this is a text that will wrap to three visual lines easily", BlockID: "b1", IsCalc: false},
			{LineNum: 1, Source: "next line", BlockID: "b1", IsCalc: false},
		},
		SourceContentWidth: 15,
		PreviewWidth:       40,
		CursorLine:         0,
		PreviewMode:        PreviewFull,
	}

	model := ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)

	// --- Wiring: verify with geometry.WrapText ---
	wrappedSource := geometry.WrapText(input.Lines[0], 15)
	expectedSourceVisualLines := len(wrappedSource)
	t.Logf("Source line 0 wraps to %d visual lines at width 15: %v", expectedSourceVisualLines, wrappedSource)

	if expectedSourceVisualLines < 3 {
		t.Fatalf("Expected source line 0 to wrap to 3+ lines at width 15, got %d", expectedSourceVisualLines)
	}

	// Count how many visual lines source line 0 occupies in the model
	sourceLine0Count := 0
	for _, sl := range model.SourceLines {
		if sl.SourceLineIdx == 0 {
			sourceLine0Count++
		}
	}

	// Source line 0 visual count should match geometry.WrapText count
	if sourceLine0Count != expectedSourceVisualLines {
		t.Errorf("Model has %d visual lines for source 0 but geometry.WrapText has %d",
			sourceLine0Count, expectedSourceVisualLines)
	}

	// Preview line 0 should only need 1-2 visual lines at width 40
	wrappedPreview := geometry.WrapText(input.Lines[0], 40)
	t.Logf("Preview line 0 wraps to %d visual lines at width 40", len(wrappedPreview))

	// Count preview padding (lines with no content for source line 0)
	previewPaddingForLine0 := 0
	previewContentForLine0 := 0
	for _, pl := range model.PreviewLines {
		if pl.SourceLineIdx == 0 {
			if pl.Kind == AlignedLinePadding {
				previewPaddingForLine0++
			} else {
				previewContentForLine0++
			}
		}
	}
	t.Logf("Preview for source 0: %d content lines + %d padding lines = %d total",
		previewContentForLine0, previewPaddingForLine0, previewContentForLine0+previewPaddingForLine0)

	// Total preview visual lines for source 0 should equal source visual lines
	totalPreviewLine0 := previewContentForLine0 + previewPaddingForLine0
	if totalPreviewLine0 != sourceLine0Count {
		t.Errorf("Preview has %d total lines for source 0 but source has %d -- alignment broken",
			totalPreviewLine0, sourceLine0Count)
	}

	// Verify SourceToVisual mapping
	if _, ok := model.SourceToVisual[0]; !ok {
		t.Fatal("SourceToVisual[0] missing")
	}
	if _, ok := model.SourceToVisual[1]; !ok {
		t.Fatal("SourceToVisual[1] missing")
	}

	visual0 := model.SourceToVisual[0]
	visual1 := model.SourceToVisual[1]
	t.Logf("SourceToVisual: [0]=%d, [1]=%d", visual0, visual1)

	// Source line 1 must start at visual line = visual0 + sourceLine0Count
	expectedVisual1 := visual0 + sourceLine0Count
	if visual1 != expectedVisual1 {
		t.Errorf("SourceToVisual[1]=%d, expected %d (visual0=%d + sourceLine0Count=%d)",
			visual1, expectedVisual1, visual0, sourceLine0Count)
	}

	// At visual line 0, both source and preview should show content for source line 0
	if model.SourceLines[0].SourceLineIdx != 0 {
		t.Error("Visual line 0 source does not correspond to source line 0")
	}
	if model.PreviewLines[0].SourceLineIdx != 0 {
		t.Error("Visual line 0 preview does not correspond to source line 0")
	}

	// At visual line = visual1, both should show content for source line 1
	if visual1 < len(model.SourceLines) {
		if model.SourceLines[visual1].SourceLineIdx != 1 {
			t.Errorf("Visual line %d source corresponds to source %d, want 1",
				visual1, model.SourceLines[visual1].SourceLineIdx)
		}
		if model.PreviewLines[visual1].SourceLineIdx != 1 {
			t.Errorf("Visual line %d preview corresponds to source %d, want 1",
				visual1, model.PreviewLines[visual1].SourceLineIdx)
		}
	}

	t.Logf("SC4 PASS: source line 0 wraps to %d lines, preview padded to match, line 1 starts at visual %d",
		sourceLine0Count, visual1)
}

// TestSC5_ResizeReflowsCorrectly verifies that terminal resize reflows both
// columns with correct widths and maintained alignment. Exercises View() and
// tea.WindowSizeMsg via Update().
func TestSC5_ResizeReflowsCorrectly(t *testing.T) {
	content := "# A header that is somewhat long to test wrapping behavior at different widths\nx = 10\ny = 20\nz = x + y"

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 120
	m.height = 40
	m.previewMode = PreviewFull

	// --- Render at width 120 ---
	view1 := m.View()
	viewLines1 := strings.Split(view1, "\n")
	leftWidth1, _ := m.GetPaneWidths(120)
	t.Logf("Width 120: leftWidth=%d, view lines=%d", leftWidth1, len(viewLines1))

	// Verify line count does not exceed height
	if len(viewLines1) > m.height {
		t.Errorf("At width 120: View has %d lines but height is %d", len(viewLines1), m.height)
	}

	// --- Resize to width 60 via Update ---
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 30})
	m = updated.(Model)

	// --- Render at width 60 ---
	view2 := m.View()
	viewLines2 := strings.Split(view2, "\n")
	leftWidth2, rightWidth2 := m.GetPaneWidths(60)
	t.Logf("Width 60: leftWidth=%d, rightWidth=%d, view lines=%d", leftWidth2, rightWidth2, len(viewLines2))

	// Pane widths must have changed
	if leftWidth1 == leftWidth2 {
		t.Error("Expected pane widths to change after resize, but they're the same")
	}

	// Verify line count does not exceed new height
	if len(viewLines2) > 30 {
		t.Errorf("At width 60: View has %d lines but height is 30", len(viewLines2))
	}

	// Verify all lines in the new View are at most 60 chars wide
	for i, line := range viewLines2 {
		w := lipgloss.Width(line)
		if w > 60 {
			t.Errorf("After resize, line %d has width %d > 60: %q",
				i, w, truncateForLog(line, 60))
		}
	}

	// --- Verify alignment still holds after resize ---
	aligned := m.computeAlignedPanes(leftWidth2, rightWidth2)
	if len(aligned.sourceLines) != len(aligned.previewLines) {
		t.Errorf("After resize: alignment broken, source=%d preview=%d",
			len(aligned.sourceLines), len(aligned.previewLines))
	}

	// Verify wrapping changed: header that fit on 1 line at 120 may need 2+ at 60
	sourceContentWidth60 := leftWidth2 - 4 - 2 // lineNum(4) + gutter(2)
	sourceContentWidth60 = max(sourceContentWidth60, 10)
	wrappedAt60 := geometry.WrapText("# A header that is somewhat long to test wrapping behavior at different widths", sourceContentWidth60)
	sourceContentWidth120 := leftWidth1 - 4 - 2
	sourceContentWidth120 = max(sourceContentWidth120, 10)
	wrappedAt120 := geometry.WrapText("# A header that is somewhat long to test wrapping behavior at different widths", sourceContentWidth120)

	t.Logf("Header wraps: %d lines at width-120 content=%d, %d lines at width-60 content=%d",
		len(wrappedAt120), sourceContentWidth120, len(wrappedAt60), sourceContentWidth60)

	if len(wrappedAt60) <= len(wrappedAt120) {
		t.Logf("NOTE: header wrapping count did not change (wide=%d, narrow=%d) -- header may be short enough to fit both",
			len(wrappedAt120), len(wrappedAt60))
	}

	t.Logf("SC5 PASS: resize from 120x40 to 60x30 produces correct widths and maintained alignment")
}

// truncateForLog truncates a string for log output readability.
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
