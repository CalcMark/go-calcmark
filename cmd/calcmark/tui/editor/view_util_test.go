package editor

import (
	"strings"
	"testing"
)

// ansi helpers for building test inputs matching renderCalcLine output.
const (
	resetCode = "\x1b[0m"
)

func fgCode(r, g, b int) string {
	return "\x1b[38;2;" + ansiItoa(r) + ";" + ansiItoa(g) + ";" + ansiItoa(b) + "m"
}

func bgCode(r, g, b int) string {
	return "\x1b[48;2;" + ansiItoa(r) + ";" + ansiItoa(g) + ";" + ansiItoa(b) + "m"
}

func ansiItoa(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return string(rune('0'+i/100)) + string(rune('0'+(i/10)%10)) + string(rune('0'+i%10))
}

func styled(fg, bg, text string) string {
	return fg + bg + text + resetCode
}

func TestWrapStyledLine(t *testing.T) {
	// Colors matching the Pearish theme (approximate)
	varNameFg := fgCode(156, 163, 175) // ResultMuted
	arrowFg := fgCode(107, 114, 128)   // ResultArrow
	valueFg := fgCode(168, 201, 64)    // Result (green)
	pvBg := bgCode(26, 23, 20)         // PreviewPaneBg

	tests := []struct {
		name     string
		input    string
		maxWidth int
		// wantLen: expected number of lines after wrapping
		wantLen int
		// wantPlain: expected plain text content of each wrapped line (nil = don't check)
		wantPlain []string
		// checkStyled: if true, verify ALL wrapped lines have ANSI codes
		checkStyled bool
	}{
		{
			name:      "empty string",
			input:     "",
			maxWidth:  20,
			wantLen:   1,
			wantPlain: []string{""},
		},
		{
			name:      "zero max width returns original",
			input:     "hello",
			maxWidth:  0,
			wantLen:   1,
			wantPlain: []string{"hello"},
		},
		{
			name:      "plain text fits - no wrap",
			input:     "short",
			maxWidth:  20,
			wantLen:   1,
			wantPlain: []string{"short"},
		},
		{
			name: "plain text wraps - baseline behavior",
			// WrapText hard-breaks at 10 since no space before position 10
			input:     "hello_world foo",
			maxWidth:  10,
			wantLen:   2,
			wantPlain: []string{"hello_worl", "d foo"},
		},
		{
			name: "styled text fits - returns original unchanged",
			input: styled(varNameFg, pvBg, "a") + " " +
				styled(arrowFg, pvBg, "→") + " " +
				styled(valueFg, pvBg, "42"),
			maxWidth:  20,
			wantLen:   1,
			wantPlain: []string{"a → 42"},
		},
		{
			name: "styled multi-segment wraps at variable name boundary",
			// "very_long_variable_name → 42" at width 15: hard-breaks (no space within 15)
			input: styled(varNameFg, pvBg, "very_long_variable_name") + " " +
				styled(arrowFg, pvBg, "→") + " " +
				styled(valueFg, pvBg, "42"),
			maxWidth:    15,
			wantLen:     2,
			wantPlain:   []string{"very_long_varia", "ble_name → 42"},
			checkStyled: true,
		},
		{
			name: "styled multi-segment wraps at value boundary",
			// "x → 1234567890.123456" at width 15: breaks at space, then hard-breaks value
			input: styled(varNameFg, pvBg, "x") + " " +
				styled(arrowFg, pvBg, "→") + " " +
				styled(valueFg, pvBg, "1234567890.123456"),
			maxWidth:    15,
			wantLen:     3,
			wantPlain:   []string{"x → ", "1234567890.1234", "56"},
			checkStyled: true,
		},
		{
			name: "single-segment styled wraps (PreviewMinimal mode)",
			// "→ long_value_text_here" at width 15: breaks at space after "→ "
			input:       styled(valueFg, pvBg, "→ long_value_text_here"),
			maxWidth:    15,
			wantLen:     3,
			wantPlain:   []string{"→ ", "long_value_text", "_here"},
			checkStyled: true,
		},
		{
			name: "three-way wrap preserves styling on all lines",
			// "very_long_variable_name_that_will_wrap → 42" at width 15
			input: styled(varNameFg, pvBg, "very_long_variable_name_that_will_wrap") + " " +
				styled(arrowFg, pvBg, "→") + " " +
				styled(valueFg, pvBg, "42"),
			maxWidth:    15,
			wantLen:     3,
			wantPlain:   []string{"very_long_varia", "ble_name_that_w", "ill_wrap → 42"},
			checkStyled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapStyledLine(tt.input, tt.maxWidth)

			if len(result) != tt.wantLen {
				t.Errorf("wrapStyledLine() returned %d lines, want %d", len(result), tt.wantLen)
				for i, line := range result {
					t.Logf("  line[%d]: %q (plain: %q)", i, line, stripANSI(line))
				}
			}

			// Check plain text content matches
			if tt.wantPlain != nil {
				for i, want := range tt.wantPlain {
					if i >= len(result) {
						t.Errorf("line[%d]: missing (want plain %q)", i, want)
						continue
					}
					got := stripANSI(result[i])
					if got != want {
						t.Errorf("line[%d] plain text = %q, want %q", i, got, want)
					}
				}
			}

			// Verify ALL wrapped lines have ANSI codes (the core bug fix)
			if tt.checkStyled {
				for i, line := range result {
					if !strings.Contains(line, "\x1b[") {
						t.Errorf("line[%d]: missing ANSI codes (got plain text %q)", i, line)
					}
				}
			}

			// Verify non-wrapping styled input returns byte-for-byte identical output
			if tt.wantLen == 1 && len(result) == 1 && strings.Contains(tt.input, "\x1b[") {
				if result[0] != tt.input {
					t.Errorf("non-wrapping styled line was modified:\n  got:  %q\n  want: %q", result[0], tt.input)
				}
			}
		})
	}
}

// TestWrapStyledLine_ANSIStateReplay verifies that ANSI state from earlier
// segments is replayed on continuation lines, matching overlayStringAt semantics.
func TestWrapStyledLine_ANSIStateReplay(t *testing.T) {
	// Build a styled string: FG1+BG+"hello " + RESET + FG2+BG+"world" + RESET
	fg1 := fgCode(255, 0, 0) // red
	fg2 := fgCode(0, 255, 0) // green
	bg := bgCode(26, 23, 20) // dark bg
	input := fg1 + bg + "hello " + resetCode + fg2 + bg + "world" + resetCode

	// Width 6: "hello " fits (6 chars including trailing space), "world" on next line
	result := wrapStyledLine(input, 6)

	if len(result) != 2 {
		for i, line := range result {
			t.Logf("  line[%d]: %q (plain: %q)", i, line, stripANSI(line))
		}
		t.Fatalf("expected 2 lines, got %d", len(result))
	}

	// First line should contain fg1 (red) styling
	if !strings.Contains(result[0], fg1) {
		t.Errorf("line[0] missing fg1 style: %q", result[0])
	}

	// Second line ("world") should contain ANSI codes — either fg2 or replayed state
	if !strings.Contains(result[1], "\x1b[") {
		t.Errorf("line[1] has no ANSI codes (state not replayed): %q", result[1])
	}

	// Verify plain text
	if got := stripANSI(result[0]); got != "hello " {
		t.Errorf("line[0] plain = %q, want %q", got, "hello ")
	}
	if got := stripANSI(result[1]); got != "world" {
		t.Errorf("line[1] plain = %q, want %q", got, "world")
	}
}
