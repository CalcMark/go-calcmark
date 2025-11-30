package editor

import (
	"testing"
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
	return WrapText(line, width)
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
	for i := 0; i < 3; i++ {
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
	for i := 0; i < 6; i++ {
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
			got := WrapText(tt.line, tt.maxWidth)
			if len(got) != tt.want {
				t.Errorf("WrapText(%q, %d) = %d lines, want %d", tt.line, tt.maxWidth, len(got), tt.want)
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
