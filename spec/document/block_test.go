package document

import (
	"strings"
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

func TestInterpolatedHTMLSourceText(t *testing.T) {
	tb := NewTextBlock([]string{"Total: {{cost}}"})
	tb.SetInterpolatedSource([]string{"Total: $250"})
	tb.SetInterpolatedHTMLSource([]string{"Total: \x02$250\x03"})

	// HTML source has sentinels
	got := tb.InterpolatedHTMLSourceText()
	if got != "Total: \x02$250\x03" {
		t.Errorf("InterpolatedHTMLSourceText() = %q, want sentinel-wrapped", got)
	}

	// Plain source unchanged
	plain := tb.InterpolatedSourceText()
	if plain != "Total: $250" {
		t.Errorf("InterpolatedSourceText() = %q, want plain", plain)
	}
}

func TestInterpolatedHTMLSourceTextFallback(t *testing.T) {
	tb := NewTextBlock([]string{"Total: $250"})
	tb.SetInterpolatedSource([]string{"Total: $250"})

	// Without HTML source set, falls back to interpolated source
	got := tb.InterpolatedHTMLSourceText()
	if got != "Total: $250" {
		t.Errorf("InterpolatedHTMLSourceText() should fall back, got %q", got)
	}
}

func TestClearInterpolatedSourceClearsHTML(t *testing.T) {
	tb := NewTextBlock([]string{"{{x}}"})
	tb.SetInterpolatedSource([]string{"42"})
	tb.SetInterpolatedHTMLSource([]string{"\x0242\x03"})

	tb.ClearInterpolatedSource()

	// Both should be cleared
	if tb.InterpolatedHTMLSourceText() != "{{x}}" {
		t.Errorf("after Clear, InterpolatedHTMLSourceText() should fall back to Source()")
	}
}

func TestRenderInterpolatedSpan(t *testing.T) {
	tb := NewTextBlock([]string{"Revenue is {{rev}}"})
	tb.SetInterpolatedSource([]string{"Revenue is $4.2M"})
	tb.SetInterpolatedHTMLSource([]string{"Revenue is \x02$4.2M\x03"})
	tb.SetDirty(true)

	html := tb.Render()
	if !strings.Contains(html, `<span class="cm-interpolated">$4.2M</span>`) {
		t.Errorf("Render() should contain span-wrapped value, got %q", html)
	}
	// Should not contain raw sentinels
	if strings.Contains(html, "\x02") || strings.Contains(html, "\x03") {
		t.Errorf("Render() should not contain raw sentinels")
	}
}
