package document

import (
	"testing"
)

func TestInterpolatedSourceFallback(t *testing.T) {
	tb := NewTextBlock([]string{"hello", "world"})

	got := tb.InterpolatedSource()
	if len(got) != 2 || got[0] != "hello" || got[1] != "world" {
		t.Errorf("InterpolatedSource() should fall back to Source(), got %v", got)
	}
}

func TestInterpolatedSourceAfterSet(t *testing.T) {
	tb := NewTextBlock([]string{"Total: {{cost}}"})
	tb.SetInterpolatedSource([]string{"Total: $250"})

	got := tb.InterpolatedSource()
	if len(got) != 1 || got[0] != "Total: $250" {
		t.Errorf("InterpolatedSource() = %v, want [Total: $250]", got)
	}

	// Raw source unchanged
	raw := tb.Source()
	if len(raw) != 1 || raw[0] != "Total: {{cost}}" {
		t.Errorf("Source() should be unchanged, got %v", raw)
	}
}

func TestClearInterpolatedSource(t *testing.T) {
	tb := NewTextBlock([]string{"{{x}}"})
	tb.SetInterpolatedSource([]string{"42"})

	tb.ClearInterpolatedSource()

	got := tb.InterpolatedSource()
	if len(got) != 1 || got[0] != "{{x}}" {
		t.Errorf("after Clear, InterpolatedSource() should fall back to Source(), got %v", got)
	}
}

func TestInterpolatedSourceText(t *testing.T) {
	tb := NewTextBlock([]string{"line1", "line2"})
	tb.SetInterpolatedSource([]string{"resolved1", "resolved2"})

	got := tb.InterpolatedSourceText()
	want := "resolved1\nresolved2"
	if got != want {
		t.Errorf("InterpolatedSourceText() = %q, want %q", got, want)
	}
}

func TestInterpolatedSourceTextFallback(t *testing.T) {
	tb := NewTextBlock([]string{"a", "b"})

	got := tb.InterpolatedSourceText()
	want := "a\nb"
	if got != want {
		t.Errorf("InterpolatedSourceText() = %q, want %q", got, want)
	}
}
