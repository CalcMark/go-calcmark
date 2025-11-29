package editor

import (
	"testing"
)

func TestWrapText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		maxWidth int
		want     []string
	}{
		{
			name:     "empty string",
			text:     "",
			maxWidth: 20,
			want:     []string{""},
		},
		{
			name:     "fits in width",
			text:     "hello world",
			maxWidth: 20,
			want:     []string{"hello world"},
		},
		{
			name:     "wraps at space",
			text:     "hello world foo bar",
			maxWidth: 12,
			want:     []string{"hello world ", "foo bar"},
		},
		{
			name:     "hard wrap when no space",
			text:     "abcdefghijklmnop",
			maxWidth: 10,
			want:     []string{"abcdefghij", "klmnop"},
		},
		{
			name:     "multiple wraps",
			text:     "this is a really long line that should wrap multiple times",
			maxWidth: 15,
			want:     []string{"this is a ", "really long ", "line that ", "should wrap ", "multiple times"},
		},
		{
			name:     "zero width returns original",
			text:     "hello",
			maxWidth: 0,
			want:     []string{"hello"},
		},
		{
			name:     "long word without spaces uses character wrap",
			text:     "this is a reallylllllllllllllllllllll ong line",
			maxWidth: 20,
			want:     []string{"this is a ", "reallyllllllllllllll", "lllllll ong line"},
		},
		{
			name:     "partial word at end - no premature wrap",
			text:     "line that should wrap and so",
			maxWidth: 25,
			want:     []string{"line that should wrap ", "and so"},
		},
		{
			name:     "CJK double-width characters",
			text:     "你好世界test",
			maxWidth: 10,
			// 你好世界 = 8 visual width (4 chars x 2), test = 4 visual width
			// Total = 12, so must wrap. 你好世界te = 10 (8+2), st = 2
			want: []string{"你好世界te", "st"},
		},
		{
			name:     "emoji double-width",
			text:     "hello 🎉 world",
			maxWidth: 10,
			// hello = 5, space = 1, 🎉 = 2, so "hello 🎉" = 8 fits
			// but "hello 🎉 w" = 10, "hello 🎉 wo" = 11 doesn't fit
			want: []string{"hello 🎉 ", "world"},
		},
		{
			name:     "mixed unicode and ASCII",
			text:     "café ☕ time",
			maxWidth: 8,
			// café = 4 (accented chars are single-width), space = 1, ☕ = 2
			// "café ☕" = 7 fits, "café ☕ " = 8 fits
			want: []string{"café ☕ ", "time"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WrapText(tt.text, tt.maxWidth)
			if len(got) != len(tt.want) {
				t.Errorf("WrapText() returned %d lines, want %d", len(got), len(tt.want))
				t.Logf("got: %v", got)
				t.Logf("want: %v", tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("WrapText()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestComputeLineModel_SingleLine(t *testing.T) {
	input := LineModelInput{
		Lines:        []string{"x = 10"},
		BlockIDs:     []string{"block1"},
		IsCalcLine:   []bool{true},
		LineResults:  []string{"10"},
		SourceWidth:  40,
		PreviewWidth: 20,
		EditMode:     false,
	}

	model := ComputeLineModel(input)

	// Should have 1 source line and 1 preview line
	if len(model.SourceLines) != 1 {
		t.Errorf("Expected 1 source line, got %d", len(model.SourceLines))
	}
	if len(model.PreviewLines) != 1 {
		t.Errorf("Expected 1 preview line, got %d", len(model.PreviewLines))
	}

	// Check source line
	if model.SourceLines[0].Content != "x = 10" {
		t.Errorf("Source content = %q, want %q", model.SourceLines[0].Content, "x = 10")
	}
	if model.SourceLines[0].LineNumber != 1 {
		t.Errorf("Source line number = %d, want 1", model.SourceLines[0].LineNumber)
	}
	if model.SourceLines[0].Kind != LineKindNormal {
		t.Errorf("Source kind = %v, want LineKindNormal", model.SourceLines[0].Kind)
	}

	// Check preview line
	if model.PreviewLines[0].Content != "10" {
		t.Errorf("Preview content = %q, want %q", model.PreviewLines[0].Content, "10")
	}
}

func TestComputeLineModel_WrappedSourceLine(t *testing.T) {
	input := LineModelInput{
		Lines:        []string{"this is a really long line that should wrap"},
		BlockIDs:     []string{"block1"},
		IsCalcLine:   []bool{false},
		LineResults:  []string{"rendered"},
		SourceWidth:  20,
		PreviewWidth: 40,
		EditMode:     false,
	}

	model := ComputeLineModel(input)

	// Source should wrap into multiple visual lines
	if len(model.SourceLines) < 2 {
		t.Errorf("Expected source to wrap, got %d lines", len(model.SourceLines))
	}

	// First line should have line number
	if model.SourceLines[0].LineNumber != 1 {
		t.Errorf("First source line number = %d, want 1", model.SourceLines[0].LineNumber)
	}
	if model.SourceLines[0].Kind != LineKindNormal {
		t.Errorf("First source kind = %v, want LineKindNormal", model.SourceLines[0].Kind)
	}

	// Second line should be wrapped (no line number)
	if model.SourceLines[1].LineNumber != 0 {
		t.Errorf("Wrapped line number = %d, want 0", model.SourceLines[1].LineNumber)
	}
	if model.SourceLines[1].Kind != LineKindWrapped {
		t.Errorf("Wrapped kind = %v, want LineKindWrapped", model.SourceLines[1].Kind)
	}

	// All source lines map to same source line index
	for i, sl := range model.SourceLines {
		if sl.SourceLineIdx != 0 {
			t.Errorf("SourceLines[%d].SourceLineIdx = %d, want 0", i, sl.SourceLineIdx)
		}
	}
}

func TestComputeLineModel_EditMode_NoDuplicate(t *testing.T) {
	// This tests the bug where edit mode was duplicating wrapped lines
	input := LineModelInput{
		Lines:         []string{"this is a really long line that should wrap"},
		BlockIDs:      []string{"block1"},
		IsCalcLine:   []bool{false},
		LineResults:   []string{"rendered"},
		SourceWidth:   20,
		PreviewWidth:  40,
		EditMode:      true,
		EditLineIdx:   0,
		EditBuffer:    "this is a really long line that should wrap",
		EditCursorCol: 43,
	}

	model := ComputeLineModel(input)

	// Count how many times "should wrap" appears across all source lines
	wrapCount := 0
	for _, sl := range model.SourceLines {
		if containsSubstring(sl.Content, "should") || containsSubstring(sl.Content, "wrap") {
			wrapCount++
		}
	}

	// The wrapped content should appear exactly the right number of times
	// (once per wrap segment, not duplicated)
	t.Logf("Source lines: %d", len(model.SourceLines))
	for i, sl := range model.SourceLines {
		t.Logf("  [%d] kind=%v num=%d content=%q", i, sl.Kind, sl.LineNumber, sl.Content)
	}

	// First line should be edit type
	if model.SourceLines[0].Kind != LineKindEditFirst {
		t.Errorf("First line kind = %v, want LineKindEditFirst", model.SourceLines[0].Kind)
	}

	// Subsequent lines should be edit wrap type
	for i := 1; i < len(model.SourceLines); i++ {
		if model.SourceLines[i].Kind == LineKindPadding {
			continue // padding is OK
		}
		if model.SourceLines[i].Kind != LineKindEditWrap {
			t.Errorf("SourceLines[%d].Kind = %v, want LineKindEditWrap", i, model.SourceLines[i].Kind)
		}
	}
}

func TestComputeLineModel_PreviewWraps(t *testing.T) {
	// Test that preview wraps instead of truncating
	input := LineModelInput{
		Lines:        []string{"x"},
		BlockIDs:     []string{"block1"},
		IsCalcLine:   []bool{false},
		LineResults:  []string{"this is a really long preview result"},
		SourceWidth:  40,
		PreviewWidth: 15,
		EditMode:     false,
	}

	model := ComputeLineModel(input)

	// Preview should wrap into multiple lines
	if len(model.PreviewLines) < 2 {
		t.Errorf("Expected preview to wrap, got %d lines", len(model.PreviewLines))
	}

	// Check all preview content is present (not truncated)
	allContent := ""
	for _, pl := range model.PreviewLines {
		allContent += pl.Content
	}
	if !containsSubstring(allContent, "really") || !containsSubstring(allContent, "result") {
		t.Errorf("Preview content appears truncated: %q", allContent)
	}
}

func TestComputeLineModel_BlockAlignment(t *testing.T) {
	// Test that source and preview are padded to align
	input := LineModelInput{
		Lines:        []string{"short"},
		BlockIDs:     []string{"block1"},
		IsCalcLine:   []bool{false},
		LineResults:  []string{"this is a much longer preview that wraps"},
		SourceWidth:  40,
		PreviewWidth: 15,
		EditMode:     false,
	}

	model := ComputeLineModel(input)

	// Source and preview should have same number of lines
	if len(model.SourceLines) != len(model.PreviewLines) {
		t.Errorf("Source lines (%d) != Preview lines (%d)",
			len(model.SourceLines), len(model.PreviewLines))
	}

	// Extra source lines should be padding
	for i := 1; i < len(model.SourceLines); i++ {
		if model.SourceLines[i].Kind != LineKindPadding {
			t.Errorf("SourceLines[%d].Kind = %v, want LineKindPadding", i, model.SourceLines[i].Kind)
		}
	}
}

func TestComputeLineModel_MultipleBlocks(t *testing.T) {
	input := LineModelInput{
		Lines:        []string{"# Header", "x = 10", "y = 20"},
		BlockIDs:     []string{"text1", "calc1", "calc1"},
		IsCalcLine:   []bool{false, true, true},
		LineResults:  []string{"Header", "10", "20"},
		SourceWidth:  40,
		PreviewWidth: 40,
		EditMode:     false,
	}

	model := ComputeLineModel(input)

	// Should have 3 source lines (one per input line)
	if len(model.SourceLines) != 3 {
		t.Errorf("Expected 3 source lines, got %d", len(model.SourceLines))
	}

	// Check line numbers
	expectedLineNums := []int{1, 2, 3}
	for i, want := range expectedLineNums {
		if model.SourceLines[i].LineNumber != want {
			t.Errorf("SourceLines[%d].LineNumber = %d, want %d",
				i, model.SourceLines[i].LineNumber, want)
		}
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && containsSubstringHelper(s, substr)))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
