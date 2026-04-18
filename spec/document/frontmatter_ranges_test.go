package document

import (
	"errors"
	"testing"
)

// TestParseFrontmatter_KeyRanges_SingleScalar verifies a registered scalar key
// surfaces a range whose Start.Line points at the key's source line (1-based).
func TestParseFrontmatter_KeyRanges_SingleScalar(t *testing.T) {
	source := "---\nconvert_to: si\n---\n"
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("ParseFrontmatter: %v", err)
	}
	if fm == nil {
		t.Fatal("expected frontmatter, got nil")
	}
	r, ok := fm.KeyRanges["convert_to"]
	if !ok {
		t.Fatalf("KeyRanges missing 'convert_to'; got %v", fm.KeyRanges)
	}
	// Line 1 is "---", line 2 is "convert_to: si"; 1-based per ast.Position.
	if r.Start.Line != 2 {
		t.Errorf("Start.Line = %d, want 2", r.Start.Line)
	}
	if r.End.Line != 2 {
		t.Errorf("End.Line = %d, want 2", r.End.Line)
	}
	if r.Start.Column != 1 {
		t.Errorf("Start.Column = %d, want 1", r.Start.Column)
	}
}

// TestParseFrontmatter_KeyRanges_MultiLineMapping verifies a mapping value
// (globals: ...) yields a range that spans the whole block including children.
func TestParseFrontmatter_KeyRanges_MultiLineMapping(t *testing.T) {
	// Lines:
	// 1: ---
	// 2: convert_to: si
	// 3: globals:
	// 4:   foo: 1
	// 5:   bar: 2
	// 6: ---
	source := "---\nconvert_to: si\nglobals:\n  foo: 1\n  bar: 2\n---\n"
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("ParseFrontmatter: %v", err)
	}
	r, ok := fm.KeyRanges["globals"]
	if !ok {
		t.Fatalf("KeyRanges missing 'globals'; got %v", fm.KeyRanges)
	}
	if r.Start.Line != 3 {
		t.Errorf("globals Start.Line = %d, want 3", r.Start.Line)
	}
	if r.End.Line != 5 {
		t.Errorf("globals End.Line = %d, want 5 (last child line)", r.End.Line)
	}
}

// TestParseFrontmatter_KeyRanges_ExtraKey verifies non-CalcMark keys (Extra)
// also populate KeyRanges, so tooling can locate them too.
func TestParseFrontmatter_KeyRanges_ExtraKey(t *testing.T) {
	source := "---\ntitle: Hello\n---\n"
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("ParseFrontmatter: %v", err)
	}
	r, ok := fm.KeyRanges["title"]
	if !ok {
		t.Fatalf("KeyRanges missing 'title' (Extra key); got %v", fm.KeyRanges)
	}
	if r.Start.Line != 2 {
		t.Errorf("title Start.Line = %d, want 2", r.Start.Line)
	}
}

// TestParseFrontmatter_KeyRanges_MultipleKeys verifies several keys at known
// line numbers are populated correctly.
func TestParseFrontmatter_KeyRanges_MultipleKeys(t *testing.T) {
	// 1: ---
	// 2: scale: 2
	// 3: convert_to: si
	// 4: title: Demo
	// 5: ---
	source := "---\nscale: 2\nconvert_to: si\ntitle: Demo\n---\n"
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("ParseFrontmatter: %v", err)
	}
	cases := map[string]int{
		"scale":      2,
		"convert_to": 3,
		"title":      4,
	}
	for key, wantLine := range cases {
		r, ok := fm.KeyRanges[key]
		if !ok {
			t.Errorf("KeyRanges missing %q", key)
			continue
		}
		if r.Start.Line != wantLine {
			t.Errorf("%s Start.Line = %d, want %d", key, r.Start.Line, wantLine)
		}
	}
}

// TestParseFrontmatter_KeyRanges_EmptyBody verifies an empty frontmatter body
// produces an empty (or nil) KeyRanges map.
func TestParseFrontmatter_KeyRanges_EmptyBody(t *testing.T) {
	source := "---\n---\n"
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("ParseFrontmatter: %v", err)
	}
	if fm == nil {
		t.Fatal("expected non-nil frontmatter for empty body")
	}
	if len(fm.KeyRanges) != 0 {
		t.Errorf("KeyRanges should be empty for empty body, got %v", fm.KeyRanges)
	}
}

// TestParseFrontmatter_KeyRanges_UnclosedFence verifies the parser
// returns ErrFrontmatterUnclosed on a missing closing fence, no
// frontmatter is exposed (so KeyRanges is unobservable), and the
// full source is returned as body for lenient callers.
func TestParseFrontmatter_KeyRanges_UnclosedFence(t *testing.T) {
	source := "---\nconvert_to: si\n"
	fm, remaining, err := ParseFrontmatter(source)
	if !errors.Is(err, ErrFrontmatterUnclosed) {
		t.Fatalf("expected ErrFrontmatterUnclosed, got %v", err)
	}
	if fm != nil {
		t.Errorf("expected nil frontmatter when fence unclosed, got %+v", fm)
	}
	if remaining != source {
		t.Errorf("expected source to pass through, got %q", remaining)
	}
}
