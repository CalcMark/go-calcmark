package geometry

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
			name:     "exactly maxWidth",
			text:     "1234567890",
			maxWidth: 10,
			want:     []string{"1234567890"},
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
			name:     "negative width returns original",
			text:     "hello",
			maxWidth: -5,
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
			// Total = 12, must wrap. 你好世界te = 10 (8+2), st = 2
			want: []string{"你好世界te", "st"},
		},
		{
			name:     "emoji double-width",
			text:     "hello 🎉 world",
			maxWidth: 10,
			// hello = 5, space = 1, 🎉 = 2, so "hello 🎉 " fits in 10
			want: []string{"hello 🎉 ", "world"},
		},
		{
			name:     "mixed unicode and ASCII",
			text:     "cafe\u0301 \u2615 time",
			maxWidth: 8,
			// cafe\u0301 = 4 (accented chars are single-width), space = 1, \u2615 = 2
			// "cafe\u0301 \u2615 " = 8 fits
			want: []string{"cafe\u0301 \u2615 ", "time"},
		},
		{
			name:     "single char wider than maxWidth",
			text:     "你",
			maxWidth: 1,
			// CJK char is 2 wide, maxWidth is 1 -- include it anyway
			want: []string{"你"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WrapText(tt.text, tt.maxWidth)
			if len(got) != len(tt.want) {
				t.Errorf("WrapText() returned %d lines, want %d", len(got), len(tt.want))
				t.Logf("got:  %v", got)
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

func TestCalculateRowGeometry(t *testing.T) {
	tests := []struct {
		name       string
		srcLine    string
		result     string
		leftWidth  int
		rightWidth int
		wantHeight int
		wantLeft   []string
		wantRight  []string
	}{
		{
			name:       "single line both sides",
			srcLine:    "hello",
			result:     "world",
			leftWidth:  10,
			rightWidth: 10,
			wantHeight: 1,
			wantLeft:   []string{"hello"},
			wantRight:  []string{"world"},
		},
		{
			name:       "left wraps right does not",
			srcLine:    "this is a longer line that wraps",
			result:     "short",
			leftWidth:  10,
			rightWidth: 10,
			wantHeight: 4,
			wantLeft:   []string{"this is a ", "longer ", "line that ", "wraps"},
			wantRight:  []string{"short", "", "", ""},
		},
		{
			name:       "right wraps left does not",
			srcLine:    "short",
			result:     "1234567890",
			leftWidth:  10,
			rightWidth: 5,
			wantHeight: 2,
			wantLeft:   []string{"short", ""},
			wantRight:  []string{"12345", "67890"},
		},
		{
			name:       "both wrap asymmetrically",
			srcLine:    "long source content here",
			result:     "even longer result that wraps more than source does",
			leftWidth:  10,
			rightWidth: 10,
			wantHeight: 8,
			wantLeft:   []string{"long ", "source ", "content ", "here", "", "", "", ""},
			wantRight:  []string{"even ", "longer ", "result ", "that ", "wraps ", "more than ", "source ", "does"},
		},
		{
			name:       "empty result",
			srcLine:    "hello",
			result:     "",
			leftWidth:  10,
			rightWidth: 10,
			wantHeight: 1,
			wantLeft:   []string{"hello"},
			wantRight:  []string{""},
		},
		{
			name:       "both empty",
			srcLine:    "",
			result:     "",
			leftWidth:  10,
			rightWidth: 10,
			wantHeight: 1,
			wantLeft:   []string{""},
			wantRight:  []string{""},
		},
		{
			name:       "very narrow widths with long text",
			srcLine:    "hello world",
			result:     "goodbye world",
			leftWidth:  5,
			rightWidth: 5,
			wantHeight: 3,
			wantLeft:   []string{"hello", " worl", "d"},
			wantRight:  []string{"goodb", "ye ", "world"},
		},
		{
			name:       "zero left width",
			srcLine:    "hello",
			result:     "world",
			leftWidth:  0,
			rightWidth: 10,
			wantHeight: 1,
			wantLeft:   []string{"hello"},
			wantRight:  []string{"world"},
		},
		{
			name:       "zero right width",
			srcLine:    "hello",
			result:     "world",
			leftWidth:  10,
			rightWidth: 0,
			wantHeight: 1,
			wantLeft:   []string{"hello"},
			wantRight:  []string{"world"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateRowGeometry(tt.srcLine, tt.result, tt.leftWidth, tt.rightWidth)

			if got.Height != tt.wantHeight {
				t.Errorf("Height = %d, want %d", got.Height, tt.wantHeight)
			}

			if len(got.LeftLines) != tt.wantHeight {
				t.Errorf("len(LeftLines) = %d, want %d", len(got.LeftLines), tt.wantHeight)
			}
			if len(got.RightLines) != tt.wantHeight {
				t.Errorf("len(RightLines) = %d, want %d", len(got.RightLines), tt.wantHeight)
			}

			if tt.wantLeft != nil {
				for i := range tt.wantLeft {
					if i < len(got.LeftLines) && got.LeftLines[i] != tt.wantLeft[i] {
						t.Errorf("LeftLines[%d] = %q, want %q", i, got.LeftLines[i], tt.wantLeft[i])
					}
				}
			}

			if tt.wantRight != nil {
				for i := range tt.wantRight {
					if i < len(got.RightLines) && got.RightLines[i] != tt.wantRight[i] {
						t.Errorf("RightLines[%d] = %q, want %q", i, got.RightLines[i], tt.wantRight[i])
					}
				}
			}
		})
	}
}

func TestStringWidth(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want int
	}{
		{
			name: "ASCII text",
			s:    "hello",
			want: 5,
		},
		{
			name: "CJK characters double-width",
			s:    "你好",
			want: 4,
		},
		{
			name: "empty string",
			s:    "",
			want: 0,
		},
		{
			name: "mixed ASCII and CJK",
			s:    "hi你好",
			want: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringWidth(tt.s)
			if got != tt.want {
				t.Errorf("StringWidth(%q) = %d, want %d", tt.s, got, tt.want)
			}
		})
	}
}
