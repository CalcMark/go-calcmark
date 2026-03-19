package embedded

import (
	"testing"
)

func TestBacktickFenceCM(t *testing.T) {
	input := "```cm\na = 1\n```\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d: %+v", len(got), got)
	}
	if got[0].Kind != CalcMarkBlock {
		t.Errorf("expected CalcMarkBlock, got %d", got[0].Kind)
	}
	if got[0].Text != "a = 1\n" {
		t.Errorf("expected %q, got %q", "a = 1\n", got[0].Text)
	}
	if got[0].OpenLine != 1 {
		t.Errorf("expected OpenLine=1, got %d", got[0].OpenLine)
	}
}

func TestBacktickFenceCalcmark(t *testing.T) {
	input := "```calcmark\nb = 2\n```\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d: %+v", len(got), got)
	}
	if got[0].Kind != CalcMarkBlock {
		t.Errorf("expected CalcMarkBlock, got %d", got[0].Kind)
	}
	if got[0].Text != "b = 2\n" {
		t.Errorf("expected %q, got %q", "b = 2\n", got[0].Text)
	}
}

func TestTildeFenceCM(t *testing.T) {
	input := "~~~cm\na = 1\n~~~\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d: %+v", len(got), got)
	}
	if got[0].Kind != CalcMarkBlock {
		t.Errorf("expected CalcMarkBlock, got %d", got[0].Kind)
	}
	if got[0].Text != "a = 1\n" {
		t.Errorf("expected %q, got %q", "a = 1\n", got[0].Text)
	}
}

func TestTildeFenceCalcmark(t *testing.T) {
	input := "~~~calcmark\nb = 2\n~~~\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d: %+v", len(got), got)
	}
	if got[0].Kind != CalcMarkBlock {
		t.Errorf("expected CalcMarkBlock, got %d", got[0].Kind)
	}
	if got[0].Text != "b = 2\n" {
		t.Errorf("expected %q, got %q", "b = 2\n", got[0].Text)
	}
}

func TestFourBacktickFence(t *testing.T) {
	input := "````cm\na = 1\n````\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d: %+v", len(got), got)
	}
	if got[0].Kind != CalcMarkBlock {
		t.Errorf("expected CalcMarkBlock, got %d", got[0].Kind)
	}
	if got[0].Text != "a = 1\n" {
		t.Errorf("expected %q, got %q", "a = 1\n", got[0].Text)
	}
}

func TestFenceLengthMismatch(t *testing.T) {
	// Opens with 4 backticks, 3 backticks inside is NOT a closer.
	input := "````cm\na = 1\n```\nb = 2\n````\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d: %+v", len(got), got)
	}
	if got[0].Kind != CalcMarkBlock {
		t.Errorf("expected CalcMarkBlock, got %d", got[0].Kind)
	}
	// Content should include the 3-backtick line as content.
	expected := "a = 1\n```\nb = 2\n"
	if got[0].Text != expected {
		t.Errorf("expected %q, got %q", expected, got[0].Text)
	}
}

func TestIndent0Space(t *testing.T) {
	input := "```cm\na = 1\n```\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(got))
	}
	if got[0].Kind != CalcMarkBlock {
		t.Errorf("expected CalcMarkBlock")
	}
}

func TestIndent1Space(t *testing.T) {
	input := " ```cm\na = 1\n ```\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d: %+v", len(got), got)
	}
	if got[0].Kind != CalcMarkBlock {
		t.Errorf("expected CalcMarkBlock")
	}
}

func TestIndent2Spaces(t *testing.T) {
	input := "  ```cm\na = 1\n  ```\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d: %+v", len(got), got)
	}
	if got[0].Kind != CalcMarkBlock {
		t.Errorf("expected CalcMarkBlock")
	}
}

func TestIndent3Spaces(t *testing.T) {
	input := "   ```cm\na = 1\n   ```\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d: %+v", len(got), got)
	}
	if got[0].Kind != CalcMarkBlock {
		t.Errorf("expected CalcMarkBlock")
	}
}

func TestIndent4SpacesNotFence(t *testing.T) {
	input := "    ```cm\na = 1\n    ```\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d: %+v", len(got), got)
	}
	if got[0].Kind != Passthrough {
		t.Errorf("expected Passthrough (4-space indent is not a fence), got %d", got[0].Kind)
	}
	if got[0].Text != input {
		t.Errorf("expected full input as passthrough, got %q", got[0].Text)
	}
}

func TestInfoStringCmakeNotMatched(t *testing.T) {
	input := "```cmake\na = 1\n```\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(got))
	}
	if got[0].Kind != Passthrough {
		t.Errorf("expected Passthrough for cmake info string")
	}
}

func TestInfoStringCmExtraNotMatched(t *testing.T) {
	input := "```cm-extra\na = 1\n```\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(got))
	}
	if got[0].Kind != Passthrough {
		t.Errorf("expected Passthrough for cm-extra info string")
	}
}

func TestInfoStringUppercaseCMNotMatched(t *testing.T) {
	input := "```CM\na = 1\n```\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(got))
	}
	if got[0].Kind != Passthrough {
		t.Errorf("expected Passthrough for CM (case-sensitive)")
	}
}

func TestInfoStringCmWithAttributes(t *testing.T) {
	input := "```cm {.highlight}\na = 1\n```\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d: %+v", len(got), got)
	}
	if got[0].Kind != CalcMarkBlock {
		t.Errorf("expected CalcMarkBlock for 'cm {.highlight}'")
	}
	if got[0].Text != "a = 1\n" {
		t.Errorf("expected %q, got %q", "a = 1\n", got[0].Text)
	}
}

func TestInfoStringCalcmarkWithTitle(t *testing.T) {
	input := "```calcmark title=\"foo\"\nb = 2\n```\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d: %+v", len(got), got)
	}
	if got[0].Kind != CalcMarkBlock {
		t.Errorf("expected CalcMarkBlock for 'calcmark title=\"foo\"'")
	}
	if got[0].Text != "b = 2\n" {
		t.Errorf("expected %q, got %q", "b = 2\n", got[0].Text)
	}
}

func TestUnclosedFenceIsPassthrough(t *testing.T) {
	input := "```cm\na = 1\nb = 2\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d: %+v", len(got), got)
	}
	if got[0].Kind != Passthrough {
		t.Errorf("expected Passthrough for unclosed fence")
	}
	if got[0].Text != input {
		t.Errorf("expected full input as passthrough, got %q", got[0].Text)
	}
}

func TestEmptyBlock(t *testing.T) {
	input := "```cm\n```\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d: %+v", len(got), got)
	}
	if got[0].Kind != CalcMarkBlock {
		t.Errorf("expected CalcMarkBlock")
	}
	if got[0].Text != "" {
		t.Errorf("expected empty text, got %q", got[0].Text)
	}
	if got[0].OpenLine != 1 {
		t.Errorf("expected OpenLine=1, got %d", got[0].OpenLine)
	}
}

func TestNestedFencesDifferentChar(t *testing.T) {
	// ~~~ opener, ``` inside is NOT a closer.
	input := "~~~cm\n```\na = 1\n~~~\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d: %+v", len(got), got)
	}
	if got[0].Kind != CalcMarkBlock {
		t.Errorf("expected CalcMarkBlock")
	}
	expected := "```\na = 1\n"
	if got[0].Text != expected {
		t.Errorf("expected %q, got %q", expected, got[0].Text)
	}
}

func TestMultipleBlocksWithProse(t *testing.T) {
	input := "# Title\n\n```cm\na = 1\n```\n\nSome prose.\n\n```calcmark\nb = 2\n```\n\nEnd.\n"
	got := Scan(input)
	if len(got) != 5 {
		t.Fatalf("expected 5 segments, got %d: %+v", len(got), got)
	}
	// Segment 0: passthrough (title + blank line)
	if got[0].Kind != Passthrough {
		t.Errorf("segment 0: expected Passthrough, got %d", got[0].Kind)
	}
	if got[0].Text != "# Title\n\n" {
		t.Errorf("segment 0: expected %q, got %q", "# Title\n\n", got[0].Text)
	}
	// Segment 1: CalcMarkBlock
	if got[1].Kind != CalcMarkBlock {
		t.Errorf("segment 1: expected CalcMarkBlock")
	}
	if got[1].Text != "a = 1\n" {
		t.Errorf("segment 1: expected %q, got %q", "a = 1\n", got[1].Text)
	}
	if got[1].OpenLine != 3 {
		t.Errorf("segment 1: expected OpenLine=3, got %d", got[1].OpenLine)
	}
	// Segment 2: passthrough (blank + prose + blank)
	if got[2].Kind != Passthrough {
		t.Errorf("segment 2: expected Passthrough")
	}
	if got[2].Text != "\nSome prose.\n\n" {
		t.Errorf("segment 2: expected %q, got %q", "\nSome prose.\n\n", got[2].Text)
	}
	// Segment 3: CalcMarkBlock
	if got[3].Kind != CalcMarkBlock {
		t.Errorf("segment 3: expected CalcMarkBlock")
	}
	if got[3].Text != "b = 2\n" {
		t.Errorf("segment 3: expected %q, got %q", "b = 2\n", got[3].Text)
	}
	if got[3].OpenLine != 9 {
		t.Errorf("segment 3: expected OpenLine=9, got %d", got[3].OpenLine)
	}
	// Segment 4: passthrough (blank + End.)
	if got[4].Kind != Passthrough {
		t.Errorf("segment 4: expected Passthrough")
	}
	if got[4].Text != "\nEnd.\n" {
		t.Errorf("segment 4: expected %q, got %q", "\nEnd.\n", got[4].Text)
	}
}

func TestNoBlocks(t *testing.T) {
	input := "Just some text.\nNothing special.\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(got))
	}
	if got[0].Kind != Passthrough {
		t.Errorf("expected Passthrough")
	}
	if got[0].Text != input {
		t.Errorf("expected %q, got %q", input, got[0].Text)
	}
}

func TestSingleBlockOnly(t *testing.T) {
	input := "```cm\nx = 42\n```\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(got))
	}
	if got[0].Kind != CalcMarkBlock {
		t.Errorf("expected CalcMarkBlock")
	}
	if got[0].Text != "x = 42\n" {
		t.Errorf("expected %q, got %q", "x = 42\n", got[0].Text)
	}
}

func TestPassthroughPreservesContentByteForByte(t *testing.T) {
	// Include tabs, trailing spaces, CRLF-style endings mixed in.
	input := "line one\t \nline two\n\n```go\nfmt.Println()\n```\nfinal\n"
	got := Scan(input)
	// The go block is not cm/calcmark, so everything is passthrough.
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d: %+v", len(got), got)
	}
	if got[0].Text != input {
		t.Errorf("passthrough did not preserve content byte-for-byte:\nexpected: %q\ngot:      %q", input, got[0].Text)
	}
}

func TestOpenLineTracking(t *testing.T) {
	input := "prose\n\n```cm\na = 1\n```\n\n```calcmark\nb = 2\n```\n"
	got := Scan(input)
	// Expect: passthrough, block@line3, passthrough, block@line7
	if len(got) != 4 {
		t.Fatalf("expected 4 segments, got %d: %+v", len(got), got)
	}
	if got[0].Kind != Passthrough || got[0].OpenLine != 0 {
		t.Errorf("segment 0: expected Passthrough with OpenLine=0, got Kind=%d OpenLine=%d", got[0].Kind, got[0].OpenLine)
	}
	if got[1].Kind != CalcMarkBlock || got[1].OpenLine != 3 {
		t.Errorf("segment 1: expected CalcMarkBlock with OpenLine=3, got Kind=%d OpenLine=%d", got[1].Kind, got[1].OpenLine)
	}
	if got[2].Kind != Passthrough || got[2].OpenLine != 0 {
		t.Errorf("segment 2: expected Passthrough with OpenLine=0, got Kind=%d OpenLine=%d", got[2].Kind, got[2].OpenLine)
	}
	if got[3].Kind != CalcMarkBlock || got[3].OpenLine != 7 {
		t.Errorf("segment 3: expected CalcMarkBlock with OpenLine=7, got Kind=%d OpenLine=%d", got[3].Kind, got[3].OpenLine)
	}
}

func TestEmptyInput(t *testing.T) {
	got := Scan("")
	if len(got) != 0 {
		t.Fatalf("expected 0 segments for empty input, got %d: %+v", len(got), got)
	}
}

func TestClosingFenceWithTrailingSpaces(t *testing.T) {
	input := "```cm\na = 1\n```   \n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d: %+v", len(got), got)
	}
	if got[0].Kind != CalcMarkBlock {
		t.Errorf("expected CalcMarkBlock (closing fence with trailing spaces)")
	}
	if got[0].Text != "a = 1\n" {
		t.Errorf("expected %q, got %q", "a = 1\n", got[0].Text)
	}
}

func TestClosingFenceWithNonSpaceCharsIsNotClose(t *testing.T) {
	// Closing fence with text after backticks is not a valid close.
	input := "```cm\na = 1\n``` text\n```\n"
	got := Scan(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 segment, got %d: %+v", len(got), got)
	}
	if got[0].Kind != CalcMarkBlock {
		t.Errorf("expected CalcMarkBlock")
	}
	// "``` text" is content, not a closer. The real closer is the last ```.
	expected := "a = 1\n``` text\n"
	if got[0].Text != expected {
		t.Errorf("expected %q, got %q", expected, got[0].Text)
	}
}
