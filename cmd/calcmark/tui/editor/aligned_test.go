package editor

import (
	"testing"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/geometry"
)

// mockRenderCalcLine returns a simple representation for testing
func mockRenderCalcLine(r LineResult, width int) string {
	if r.Value != "" {
		return r.VarName + " = " + r.Value
	}
	return ""
}

// mockRenderMarkdown returns the line as-is for testing
func mockRenderMarkdown(line string, width int) []string {
	return geometry.WrapText(line, width)
}

func TestComputeAlignedModel_Simple(t *testing.T) {
	input := AlignedModelInput{
		Lines: []string{"# Header", "x = 10", "y = 20"},
		Results: []LineResult{
			{LineNum: 0, Source: "# Header", BlockID: "b1", IsCalc: false},
			{LineNum: 1, Source: "x = 10", BlockID: "b2", IsCalc: true, VarName: "x", Value: "10"},
			{LineNum: 2, Source: "y = 20", BlockID: "b2", IsCalc: true, VarName: "y", Value: "20"},
		},
		SourceContentWidth: 40,
		PreviewWidth:       40,
		CursorLine:         1,
		PreviewMode:        PreviewFull,
	}

	model := ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)

	// Check basic structure
	if model.TotalSourceLines != 3 {
		t.Errorf("TotalSourceLines = %d, want 3", model.TotalSourceLines)
	}

	if model.TotalVisualLines != 3 {
		t.Errorf("TotalVisualLines = %d, want 3", model.TotalVisualLines)
	}

	// Check 1:1 alignment
	if len(model.SourceLines) != len(model.PreviewLines) {
		t.Errorf("SourceLines (%d) != PreviewLines (%d)", len(model.SourceLines), len(model.PreviewLines))
	}

	// Check source-to-visual mapping
	for i := range 3 {
		if v, ok := model.SourceToVisual[i]; !ok || v != i {
			t.Errorf("SourceToVisual[%d] = %d, %v, want %d, true", i, v, ok, i)
		}
	}

	// Check cursor marking
	if model.SourceLines[1].Kind != AlignedLineCursor {
		t.Errorf("Line 1 Kind = %v, want AlignedLineCursor", model.SourceLines[1].Kind)
	}

	// Check invariants
	inv := model.Invariants()
	if !inv.SourcePreviewMatch {
		t.Error("Invariant SourcePreviewMatch failed")
	}
	if !inv.MappingComplete {
		t.Error("Invariant MappingComplete failed")
	}
	if !inv.ReverseComplete {
		t.Error("Invariant ReverseComplete failed")
	}
}

func TestComputeAlignedModel_WrappedSource(t *testing.T) {
	input := AlignedModelInput{
		Lines: []string{"this is a very long line that needs wrapping", "short"},
		Results: []LineResult{
			{LineNum: 0, Source: "this is a very long line that needs wrapping", BlockID: "b1", IsCalc: false},
			{LineNum: 1, Source: "short", BlockID: "b1", IsCalc: false},
		},
		SourceContentWidth: 20, // Force wrapping
		PreviewWidth:       40,
		CursorLine:         0,
		PreviewMode:        PreviewFull,
	}

	model := ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)

	// Source line 0 should wrap into multiple visual lines
	if model.TotalVisualLines <= 2 {
		t.Errorf("TotalVisualLines = %d, expected > 2 due to wrapping", model.TotalVisualLines)
	}

	// First visual line should be normal cursor line
	if model.SourceLines[0].Kind != AlignedLineCursor {
		t.Errorf("First visual line Kind = %v, want AlignedLineCursor", model.SourceLines[0].Kind)
	}
	if model.SourceLines[0].LineNum != 1 {
		t.Errorf("First visual line LineNum = %d, want 1", model.SourceLines[0].LineNum)
	}

	// Second visual line should be wrapped continuation
	if model.SourceLines[1].Kind != AlignedLineCursorWrapped {
		t.Errorf("Second visual line Kind = %v, want AlignedLineCursorWrapped", model.SourceLines[1].Kind)
	}
	if model.SourceLines[1].LineNum != 0 {
		t.Errorf("Wrapped line LineNum = %d, want 0", model.SourceLines[1].LineNum)
	}

	// Check mapping: source line 0 maps to visual line 0
	if v := model.CursorVisualLine(0); v != 0 {
		t.Errorf("CursorVisualLine(0) = %d, want 0", v)
	}

	// Check reverse mapping: all wrapped visual lines map back to source line 0
	for i := 0; i < model.TotalVisualLines; i++ {
		srcLine := model.SourceLineAt(i)
		if srcLine < 0 || srcLine > 1 {
			t.Errorf("SourceLineAt(%d) = %d, want 0 or 1", i, srcLine)
		}
	}

	// Check invariants
	inv := model.Invariants()
	if !inv.SourcePreviewMatch {
		t.Errorf("Invariant SourcePreviewMatch failed: source=%d, preview=%d",
			len(model.SourceLines), len(model.PreviewLines))
	}
}

func TestComputeAlignedModel_PreviewWrapsMore(t *testing.T) {
	// Test case where preview wraps more than source
	// This can happen with calc results that are longer than source
	input := AlignedModelInput{
		Lines: []string{"x = 1"},
		Results: []LineResult{
			{LineNum: 0, Source: "x = 1", BlockID: "b1", IsCalc: true, VarName: "x", Value: "1"},
		},
		SourceContentWidth: 40,
		PreviewWidth:       5, // Very narrow preview forces wrapping
		CursorLine:         0,
		PreviewMode:        PreviewFull,
	}

	model := ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)

	// Source has 1 visual line, but preview needs multiple
	// Alignment should add padding to source
	if len(model.SourceLines) != len(model.PreviewLines) {
		t.Errorf("Alignment broken: source=%d, preview=%d",
			len(model.SourceLines), len(model.PreviewLines))
	}

	// Check that source has padding lines
	paddingCount := 0
	for _, sl := range model.SourceLines {
		if sl.Kind == AlignedLinePadding {
			paddingCount++
		}
	}

	// If preview wraps to > 1 line, source should have padding
	if model.TotalVisualLines > 1 && paddingCount == 0 {
		t.Error("Expected padding lines in source when preview wraps more")
	}
}

func TestComputeAlignedModel_EmptyDocument(t *testing.T) {
	input := AlignedModelInput{
		Lines:              []string{},
		Results:            []LineResult{},
		SourceContentWidth: 40,
		PreviewWidth:       40,
		CursorLine:         0,
		PreviewMode:        PreviewFull,
	}

	model := ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)

	if model.TotalSourceLines != 0 {
		t.Errorf("TotalSourceLines = %d, want 0", model.TotalSourceLines)
	}
	if model.TotalVisualLines != 0 {
		t.Errorf("TotalVisualLines = %d, want 0", model.TotalVisualLines)
	}

	inv := model.Invariants()
	if !inv.SourcePreviewMatch {
		t.Error("Empty document should have matching source/preview")
	}
}

func TestComputeAlignedModel_MultipleBlocks(t *testing.T) {
	input := AlignedModelInput{
		Lines: []string{"# Header", "", "x = 10", "y = 20", "", "# Footer"},
		Results: []LineResult{
			{LineNum: 0, Source: "# Header", BlockID: "text1", IsCalc: false},
			{LineNum: 1, Source: "", BlockID: "text1", IsCalc: false},
			{LineNum: 2, Source: "x = 10", BlockID: "calc1", IsCalc: true, VarName: "x", Value: "10"},
			{LineNum: 3, Source: "y = 20", BlockID: "calc1", IsCalc: true, VarName: "y", Value: "20"},
			{LineNum: 4, Source: "", BlockID: "calc1", IsCalc: true},
			{LineNum: 5, Source: "# Footer", BlockID: "text2", IsCalc: false},
		},
		SourceContentWidth: 40,
		PreviewWidth:       40,
		CursorLine:         2,
		PreviewMode:        PreviewFull,
	}

	model := ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)

	if model.TotalSourceLines != 6 {
		t.Errorf("TotalSourceLines = %d, want 6", model.TotalSourceLines)
	}

	// Cursor should be on line 2 (x = 10)
	cursorVisual := model.CursorVisualLine(2)
	if cursorVisual < 0 {
		t.Error("CursorVisualLine(2) returned -1")
	} else if model.SourceLines[cursorVisual].Kind != AlignedLineCursor {
		t.Errorf("Visual line %d Kind = %v, want AlignedLineCursor", cursorVisual, model.SourceLines[cursorVisual].Kind)
	}

	// Check each source line has a mapping
	for i := range 6 {
		if _, ok := model.SourceToVisual[i]; !ok {
			t.Errorf("Source line %d has no visual mapping", i)
		}
	}
}

func TestAlignedModel_VisibleRange(t *testing.T) {
	model := AlignedModel{
		TotalVisualLines: 100,
	}

	tests := []struct {
		name         string
		scrollOffset int
		height       int
		wantStart    int
		wantEnd      int
	}{
		{"normal", 10, 20, 10, 30},
		{"at_top", 0, 20, 0, 20},
		{"at_bottom", 90, 20, 90, 100},
		{"past_bottom", 95, 20, 95, 100},
		{"negative_offset", -5, 20, 0, 20},
		{"zero_height", 10, 0, 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := model.VisibleRange(tt.scrollOffset, tt.height)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Errorf("VisibleRange(%d, %d) = (%d, %d), want (%d, %d)",
					tt.scrollOffset, tt.height, start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

func TestAlignedModel_ScrollOffsetForCursor(t *testing.T) {
	model := AlignedModel{
		TotalVisualLines: 100,
		SourceToVisual:   map[int]int{0: 0, 10: 10, 50: 50, 90: 90},
	}

	tests := []struct {
		name          string
		cursorSource  int
		currentOffset int
		viewportH     int
		wantOffset    int
	}{
		{"cursor_visible", 10, 5, 20, 5},       // cursor at 10, visible in 5-25
		{"cursor_above", 10, 30, 20, 10},       // cursor at 10, scroll down
		{"cursor_below", 50, 10, 20, 31},       // cursor at 50, scroll up
		{"cursor_at_bottom", 90, 70, 20, 71},   // cursor at 90, just visible
		{"cursor_not_mapped", 999, 10, 20, 10}, // unmapped cursor, no change
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := model.ScrollOffsetForCursor(tt.cursorSource, tt.currentOffset, tt.viewportH)
			if got != tt.wantOffset {
				t.Errorf("ScrollOffsetForCursor(%d, %d, %d) = %d, want %d",
					tt.cursorSource, tt.currentOffset, tt.viewportH, got, tt.wantOffset)
			}
		})
	}
}

func TestAlignedModel_WrapTextBasic(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		maxWidth int
		want     int // expected number of wrapped lines
	}{
		{"short_line", "hello", 20, 1},
		{"exact_fit", "hello world", 11, 1},
		{"needs_wrap", "hello world test", 10, 2},
		{"long_word", "supercalifragilistic", 10, 2},
		{"zero_width", "hello", 0, 1},
		{"empty", "", 20, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := geometry.WrapText(tt.line, tt.maxWidth)
			if len(got) != tt.want {
				t.Errorf("geometry.WrapText(%q, %d) = %d lines, want %d", tt.line, tt.maxWidth, len(got), tt.want)
			}
		})
	}
}

func TestAlignedModel_InsertLineScenario(t *testing.T) {
	// Simulate the insert line scenario from the bug report
	// Start with 4 lines, cursor at line 2
	input := AlignedModelInput{
		Lines: []string{"# Header", "x = 10", "y = 20", "z = 30"},
		Results: []LineResult{
			{LineNum: 0, Source: "# Header", BlockID: "b1", IsCalc: false},
			{LineNum: 1, Source: "x = 10", BlockID: "b2", IsCalc: true, VarName: "x", Value: "10"},
			{LineNum: 2, Source: "y = 20", BlockID: "b2", IsCalc: true, VarName: "y", Value: "20"},
			{LineNum: 3, Source: "z = 30", BlockID: "b2", IsCalc: true, VarName: "z", Value: "30"},
		},
		SourceContentWidth: 40,
		PreviewWidth:       40,
		CursorLine:         2,
		PreviewMode:        PreviewFull,
	}

	before := ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)

	// Verify initial state
	if before.CursorVisualLine(2) != 2 {
		t.Errorf("Before insert: CursorVisualLine(2) = %d, want 2", before.CursorVisualLine(2))
	}

	// Simulate 'o' key: insert new line below cursor, cursor moves to new line
	input.Lines = []string{"# Header", "x = 10", "y = 20", "", "z = 30"}
	input.Results = []LineResult{
		{LineNum: 0, Source: "# Header", BlockID: "b1", IsCalc: false},
		{LineNum: 1, Source: "x = 10", BlockID: "b2", IsCalc: true, VarName: "x", Value: "10"},
		{LineNum: 2, Source: "y = 20", BlockID: "b2", IsCalc: true, VarName: "y", Value: "20"},
		{LineNum: 3, Source: "", BlockID: "b2", IsCalc: true}, // New empty line
		{LineNum: 4, Source: "z = 30", BlockID: "b2", IsCalc: true, VarName: "z", Value: "30"},
	}
	input.CursorLine = 3 // Cursor on new line

	after := ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)

	// Verify after state
	if after.TotalSourceLines != 5 {
		t.Errorf("After insert: TotalSourceLines = %d, want 5", after.TotalSourceLines)
	}

	// Cursor should be on visual line 3 (the new empty line)
	cursorVisual := after.CursorVisualLine(3)
	if cursorVisual != 3 {
		t.Errorf("After insert: CursorVisualLine(3) = %d, want 3", cursorVisual)
	}

	// The cursor line should be marked
	if after.SourceLines[cursorVisual].Kind != AlignedLineCursor {
		t.Errorf("After insert: cursor line Kind = %v, want AlignedLineCursor", after.SourceLines[cursorVisual].Kind)
	}

	// Navigation: moving down from cursor (line 3) should go to line 4 (z = 30)
	nextVisual := after.CursorVisualLine(4)
	if nextVisual != 4 {
		t.Errorf("After insert: CursorVisualLine(4) = %d, want 4", nextVisual)
	}

	// Check all invariants
	inv := after.Invariants()
	if !inv.SourcePreviewMatch {
		t.Error("After insert: SourcePreviewMatch invariant failed")
	}
	if !inv.MappingComplete {
		t.Error("After insert: MappingComplete invariant failed")
	}
	if !inv.ReverseComplete {
		t.Error("After insert: ReverseComplete invariant failed")
	}
}

func TestComputeAlignedModel_EditBufAsymmetricWidths(t *testing.T) {
	// Regression test: when the user is actively typing, the alignment
	// computation must use the live edit buffer (EditBuf) instead of the
	// committed document text. With asymmetric source/preview widths, the
	// edit buffer may wrap differently than the committed line, and both
	// panes must agree on the visual line count.
	//
	// Scenario: editBuf is 33 chars. Source width=41 (no wrap), preview
	// width=32. Without EditBuf, alignment uses committed text "x = 1".
	// With EditBuf, alignment uses the longer text which still fits in 41.

	input := AlignedModelInput{
		Lines: []string{"x = 1", "y = 20"},
		Results: []LineResult{
			{LineNum: 0, Source: "x = 1", BlockID: "b1", IsCalc: true, VarName: "x", Value: "1"},
			{LineNum: 1, Source: "y = 20", BlockID: "b1", IsCalc: true, VarName: "y", Value: "20"},
		},
		SourceContentWidth: 41,
		PreviewWidth:       32,
		CursorLine:         0,
		PreviewMode:        PreviewFull,
		EditBuf:            "total_gross = salary_1 + salary_2", // 33 chars, fits in 41
		EditBufLine:        0,
	}

	model := ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)

	// EditBuf (33 chars) fits in source width (41) → 1 source visual line
	// Preview result is short → 1 preview visual line
	// numAligned = max(1, 1) = 1 for line 0
	// Total: 1 (line 0) + 1 (line 1) = 2 visual lines
	if model.TotalVisualLines != 2 {
		t.Errorf("TotalVisualLines = %d, want 2", model.TotalVisualLines)
	}

	// The cursor line content should use the editBuf text, not committed text
	if model.SourceLines[0].Content != "total_gross = salary_1 + salary_2" {
		t.Errorf("SourceLines[0].Content = %q, want editBuf text", model.SourceLines[0].Content)
	}

	// Verify 1:1 alignment
	if len(model.SourceLines) != len(model.PreviewLines) {
		t.Errorf("SourceLines (%d) != PreviewLines (%d)",
			len(model.SourceLines), len(model.PreviewLines))
	}

	inv := model.Invariants()
	if !inv.SourcePreviewMatch {
		t.Error("Invariant SourcePreviewMatch failed")
	}
}

func TestComputeAlignedModel_EditBufWrapsInSource(t *testing.T) {
	// When the edit buffer is long enough to wrap in the source pane,
	// alignment must account for the extra visual lines. Both panes
	// should have the same count via padding.

	input := AlignedModelInput{
		Lines: []string{"x = 1", "y = 20"},
		Results: []LineResult{
			{LineNum: 0, Source: "x = 1", BlockID: "b1", IsCalc: true, VarName: "x", Value: "1"},
			{LineNum: 1, Source: "y = 20", BlockID: "b1", IsCalc: true, VarName: "y", Value: "20"},
		},
		SourceContentWidth: 25, // Narrow enough to force wrapping
		PreviewWidth:       32,
		CursorLine:         0,
		PreviewMode:        PreviewFull,
		EditBuf:            "total_gross = salary_1 + salary_2", // 33 chars > 25 → wraps
		EditBufLine:        0,
	}

	model := ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)

	// EditBuf (33 chars) wraps at source width (25) → 2 source lines
	// Preview result is short → 1 preview line
	// numAligned = max(2, 1) = 2 for line 0
	// Total: 2 (line 0 + wrap) + 1 (line 1) = 3 visual lines
	if model.TotalVisualLines != 3 {
		t.Errorf("TotalVisualLines = %d, want 3", model.TotalVisualLines)
	}

	// First visual line should be cursor, second should be cursor-wrapped
	if model.SourceLines[0].Kind != AlignedLineCursor {
		t.Errorf("SourceLines[0].Kind = %v, want AlignedLineCursor", model.SourceLines[0].Kind)
	}
	if model.SourceLines[1].Kind != AlignedLineCursorWrapped {
		t.Errorf("SourceLines[1].Kind = %v, want AlignedLineCursorWrapped", model.SourceLines[1].Kind)
	}

	// Preview should have content + padding for the wrapped source line
	if model.PreviewLines[1].Kind != AlignedLinePadding {
		t.Errorf("PreviewLines[1].Kind = %v, want AlignedLinePadding", model.PreviewLines[1].Kind)
	}

	inv := model.Invariants()
	if !inv.SourcePreviewMatch {
		t.Error("Invariant SourcePreviewMatch failed")
	}
}

func TestComputeAlignedModel_RenderedTextBlocks(t *testing.T) {
	// When RenderedTextBlocks is populated for a TextBlock, the pre-rendered
	// content should be used in preview lines instead of per-line rendering.
	// This tests Phase 3a wiring: RenderedBlockCache → AlignedModelInput → preview.

	input := AlignedModelInput{
		Lines: []string{"Hello **world**", "Another line", "x = 10"},
		Results: []LineResult{
			{LineNum: 0, Source: "Hello **world**", BlockID: "text1", IsCalc: false},
			{LineNum: 1, Source: "Another line", BlockID: "text1", IsCalc: false},
			{LineNum: 2, Source: "x = 10", BlockID: "calc1", IsCalc: true, VarName: "x", Value: "10"},
		},
		SourceContentWidth: 40,
		PreviewWidth:       40,
		CursorLine:         0,
		PreviewMode:        PreviewRendered,
		RenderedTextBlocks: map[string][]string{
			"text1": {"Hello world (rendered)", "Another line (rendered)"},
		},
	}

	model := ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)

	// Should have 3 visual lines (2 text + 1 calc), 1:1 aligned
	if model.TotalVisualLines != 3 {
		t.Errorf("TotalVisualLines = %d, want 3", model.TotalVisualLines)
	}

	// Preview lines for the TextBlock should use the pre-rendered content
	if model.PreviewLines[0].Content != "Hello world (rendered)" {
		t.Errorf("PreviewLines[0].Content = %q, want pre-rendered content", model.PreviewLines[0].Content)
	}
	if model.PreviewLines[1].Content != "Another line (rendered)" {
		t.Errorf("PreviewLines[1].Content = %q, want pre-rendered content", model.PreviewLines[1].Content)
	}

	// CalcBlock should still use the calc renderer (not affected by RenderedTextBlocks)
	if model.PreviewLines[2].Content != "x = 10" {
		t.Errorf("PreviewLines[2].Content = %q, want calc result", model.PreviewLines[2].Content)
	}

	// Invariants must hold
	inv2 := model.Invariants()
	if !inv2.SourcePreviewMatch {
		t.Error("Invariant SourcePreviewMatch failed")
	}
	if !inv2.MappingComplete {
		t.Error("Invariant MappingComplete failed")
	}
}

func TestComputeAlignedModel_RenderedTextBlocksFallback(t *testing.T) {
	// When RenderedTextBlocks is nil, fall back to existing per-line rendering.

	input := AlignedModelInput{
		Lines: []string{"# Header", "plain text"},
		Results: []LineResult{
			{LineNum: 0, Source: "# Header", BlockID: "text1", IsCalc: false},
			{LineNum: 1, Source: "plain text", BlockID: "text1", IsCalc: false},
		},
		SourceContentWidth: 40,
		PreviewWidth:       40,
		CursorLine:         0,
		PreviewMode:        PreviewRendered,
	}

	model := ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)

	if model.TotalVisualLines != 2 {
		t.Errorf("TotalVisualLines = %d, want 2", model.TotalVisualLines)
	}

	inv := model.Invariants()
	if !inv.SourcePreviewMatch {
		t.Error("Invariant SourcePreviewMatch failed")
	}
}

func TestComputeAlignedModel_RenderedTextBlocksMoreLines(t *testing.T) {
	// When glamour produces MORE rendered lines than source lines (e.g., a
	// table with borders), rendered lines are distributed per source line.
	// 4 rendered / 2 source = 2 rendered per source line.

	input := AlignedModelInput{
		Lines: []string{"| A | B |", "| 1 | 2 |"},
		Results: []LineResult{
			{LineNum: 0, Source: "| A | B |", BlockID: "text1", IsCalc: false},
			{LineNum: 1, Source: "| 1 | 2 |", BlockID: "text1", IsCalc: false},
		},
		SourceContentWidth: 40,
		PreviewWidth:       40,
		CursorLine:         0,
		PreviewMode:        PreviewRendered,
		RenderedTextBlocks: map[string][]string{
			"text1": {"| A   | B   |", "|-----|-----|", "| 1   | 2   |", "|-----|-----|"},
		},
	}

	model := ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)

	// Per-line distribution: 2 source lines, 4 rendered → 2 per source
	// Source line 0: 1 source wrap + 1 padding = 2 visual lines
	// Source line 1: 1 source wrap + 1 padding = 2 visual lines
	// Total: 4 visual lines
	if model.TotalVisualLines != 4 {
		t.Errorf("TotalVisualLines = %d, want 4", model.TotalVisualLines)
	}

	// Preview lines: distributed across source lines
	if model.PreviewLines[0].Content != "| A   | B   |" {
		t.Errorf("PreviewLines[0].Content = %q", model.PreviewLines[0].Content)
	}
	if model.PreviewLines[1].Content != "|-----|-----|" {
		t.Errorf("PreviewLines[1].Content = %q", model.PreviewLines[1].Content)
	}
	if model.PreviewLines[2].Content != "| 1   | 2   |" {
		t.Errorf("PreviewLines[2].Content = %q", model.PreviewLines[2].Content)
	}

	// Source: line 0 (cursor) + padding, line 1 (normal) + padding
	if model.SourceLines[0].Kind != AlignedLineCursor {
		t.Errorf("SourceLines[0].Kind = %v, want AlignedLineCursor", model.SourceLines[0].Kind)
	}
	if model.SourceLines[1].Kind != AlignedLinePadding {
		t.Errorf("SourceLines[1].Kind = %v, want AlignedLinePadding", model.SourceLines[1].Kind)
	}
	if model.SourceLines[2].Kind != AlignedLineNormal {
		t.Errorf("SourceLines[2].Kind = %v, want AlignedLineNormal", model.SourceLines[2].Kind)
	}

	invMoreLines := model.Invariants()
	if !invMoreLines.SourcePreviewMatch {
		t.Error("Invariant SourcePreviewMatch failed")
	}
}

func TestComputeAlignedModel_BlockLevelFewerRendered(t *testing.T) {
	// Per-line distribution: 5 source lines, 2 rendered lines.
	// 2/5 = 0 per source, remainder 2 → first 2 entries get 1 rendered line each.
	// Lines 0-1 each get 1 rendered line; lines 2-4 get fallback empty content.

	input := AlignedModelInput{
		Lines: []string{"line 1", "line 2", "line 3", "line 4", "line 5"},
		Results: []LineResult{
			{LineNum: 0, Source: "line 1", BlockID: "text1", IsCalc: false},
			{LineNum: 1, Source: "line 2", BlockID: "text1", IsCalc: false},
			{LineNum: 2, Source: "line 3", BlockID: "text1", IsCalc: false},
			{LineNum: 3, Source: "line 4", BlockID: "text1", IsCalc: false},
			{LineNum: 4, Source: "line 5", BlockID: "text1", IsCalc: false},
		},
		SourceContentWidth: 40,
		PreviewWidth:       40,
		CursorLine:         0,
		PreviewMode:        PreviewRendered,
		RenderedTextBlocks: map[string][]string{
			"text1": {"Rendered paragraph 1", "Rendered paragraph 2"},
		},
	}

	model := ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)

	// 5 source lines, each gets 1 visual line → total 5
	if model.TotalVisualLines != 5 {
		t.Errorf("TotalVisualLines = %d, want 5", model.TotalVisualLines)
	}

	// Preview: first 2 source lines get the rendered content
	if model.PreviewLines[0].Content != "Rendered paragraph 1" {
		t.Errorf("PreviewLines[0].Content = %q", model.PreviewLines[0].Content)
	}
	if model.PreviewLines[1].Content != "Rendered paragraph 2" {
		t.Errorf("PreviewLines[1].Content = %q", model.PreviewLines[1].Content)
	}
	// Remaining lines get empty fallback content (AlignedLineNormal, not Padding)
	for i := 2; i < 5; i++ {
		if model.PreviewLines[i].Content != "" {
			t.Errorf("PreviewLines[%d].Content = %q, want empty", i, model.PreviewLines[i].Content)
		}
	}

	inv := model.Invariants()
	if !inv.SourcePreviewMatch {
		t.Error("Invariant SourcePreviewMatch failed")
	}
	if !inv.MappingComplete {
		t.Error("Invariant MappingComplete failed")
	}
}

func TestComputeAlignedModel_BlockLevelWithSourceWrapping(t *testing.T) {
	// Block-level: source line wraps → wrapping counts toward source visual lines.

	input := AlignedModelInput{
		Lines: []string{"this is a long line that will wrap at width twenty", "short"},
		Results: []LineResult{
			{LineNum: 0, Source: "this is a long line that will wrap at width twenty", BlockID: "text1", IsCalc: false},
			{LineNum: 1, Source: "short", BlockID: "text1", IsCalc: false},
		},
		SourceContentWidth: 20,
		PreviewWidth:       40,
		CursorLine:         0,
		PreviewMode:        PreviewRendered,
		RenderedTextBlocks: map[string][]string{
			"text1": {"Rendered A", "Rendered B"},
		},
	}

	model := ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)

	// Line 0 wraps at width 20 → ~3 visual lines. Line 1 = 1. Total source visual ~4.
	// 2 rendered lines. numAligned = max(~4, 2) = ~4.
	if model.TotalVisualLines < 3 {
		t.Errorf("TotalVisualLines = %d, expected >= 3 (wrapping)", model.TotalVisualLines)
	}

	if model.SourceLines[0].Kind != AlignedLineCursor {
		t.Errorf("SourceLines[0].Kind = %v, want AlignedLineCursor", model.SourceLines[0].Kind)
	}

	inv := model.Invariants()
	if !inv.SourcePreviewMatch {
		t.Error("Invariant SourcePreviewMatch failed")
	}
}

func TestComputeAlignedModel_MixedBlocksCalcUnchanged(t *testing.T) {
	// Mixed document: TextBlock (block-aligned) + CalcBlock (line-aligned).
	// CalcBlock alignment must be identical whether RenderedTextBlocks is set or not.

	baseInput := AlignedModelInput{
		Lines: []string{"# Heading", "text", "x = 10", "y = 20"},
		Results: []LineResult{
			{LineNum: 0, Source: "# Heading", BlockID: "text1", IsCalc: false},
			{LineNum: 1, Source: "text", BlockID: "text1", IsCalc: false},
			{LineNum: 2, Source: "x = 10", BlockID: "calc1", IsCalc: true, VarName: "x", Value: "10"},
			{LineNum: 3, Source: "y = 20", BlockID: "calc1", IsCalc: true, VarName: "y", Value: "20"},
		},
		SourceContentWidth: 40,
		PreviewWidth:       40,
		CursorLine:         2,
		PreviewMode:        PreviewRendered,
	}

	// Without RenderedTextBlocks
	modelWithout := ComputeAlignedModel(baseInput, mockRenderCalcLine, mockRenderMarkdown)

	// With RenderedTextBlocks
	withRendered := baseInput
	withRendered.RenderedTextBlocks = map[string][]string{
		"text1": {"# Heading (rendered)", "text (rendered)"},
	}
	modelWith := ComputeAlignedModel(withRendered, mockRenderCalcLine, mockRenderMarkdown)

	// CalcBlock preview lines should be identical in both models.
	var calcWithout, calcWith []AlignedLine
	for _, pl := range modelWithout.PreviewLines {
		if pl.IsCalc {
			calcWithout = append(calcWithout, pl)
		}
	}
	for _, pl := range modelWith.PreviewLines {
		if pl.IsCalc {
			calcWith = append(calcWith, pl)
		}
	}

	if len(calcWithout) != len(calcWith) {
		t.Fatalf("CalcBlock line counts differ: %d vs %d", len(calcWithout), len(calcWith))
	}
	for i := range calcWithout {
		if calcWithout[i].Content != calcWith[i].Content {
			t.Errorf("CalcBlock line %d content differs: %q vs %q", i, calcWithout[i].Content, calcWith[i].Content)
		}
	}

	invWith := modelWith.Invariants()
	if !invWith.SourcePreviewMatch {
		t.Error("Invariant SourcePreviewMatch failed (with rendered)")
	}
	invWithout := modelWithout.Invariants()
	if !invWithout.SourcePreviewMatch {
		t.Error("Invariant SourcePreviewMatch failed (without rendered)")
	}
}

func TestComputeAlignedModel_BlockLevelEmptyRendered(t *testing.T) {
	// Empty rendered content for a TextBlock with blank lines.
	// Each source line gets fallback empty content (AlignedLineNormal).

	input := AlignedModelInput{
		Lines: []string{"", ""},
		Results: []LineResult{
			{LineNum: 0, Source: "", BlockID: "text1", IsCalc: false},
			{LineNum: 1, Source: "", BlockID: "text1", IsCalc: false},
		},
		SourceContentWidth: 40,
		PreviewWidth:       40,
		CursorLine:         0,
		PreviewMode:        PreviewRendered,
		RenderedTextBlocks: map[string][]string{
			"text1": {},
		},
	}

	model := ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)

	// 2 source lines, 0 rendered → each gets fallback empty → total 2
	if model.TotalVisualLines != 2 {
		t.Errorf("TotalVisualLines = %d, want 2", model.TotalVisualLines)
	}

	// Preview lines get empty fallback content
	for i := range 2 {
		if model.PreviewLines[i].Content != "" {
			t.Errorf("PreviewLines[%d].Content = %q, want empty", i, model.PreviewLines[i].Content)
		}
	}

	inv := model.Invariants()
	if !inv.SourcePreviewMatch {
		t.Error("Invariant SourcePreviewMatch failed")
	}
}

func TestComputeAlignedModel_BlockLevelSingleLine(t *testing.T) {
	// Single-line TextBlock with rendered content.

	input := AlignedModelInput{
		Lines: []string{"# Title"},
		Results: []LineResult{
			{LineNum: 0, Source: "# Title", BlockID: "text1", IsCalc: false},
		},
		SourceContentWidth: 40,
		PreviewWidth:       40,
		CursorLine:         0,
		PreviewMode:        PreviewRendered,
		RenderedTextBlocks: map[string][]string{
			"text1": {"Title (rendered)"},
		},
	}

	model := ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)

	if model.TotalVisualLines != 1 {
		t.Errorf("TotalVisualLines = %d, want 1", model.TotalVisualLines)
	}
	if model.PreviewLines[0].Content != "Title (rendered)" {
		t.Errorf("PreviewLines[0].Content = %q", model.PreviewLines[0].Content)
	}

	inv := model.Invariants()
	if !inv.SourcePreviewMatch {
		t.Error("Invariant SourcePreviewMatch failed")
	}
	if !inv.MappingComplete {
		t.Error("Invariant MappingComplete failed")
	}
}
