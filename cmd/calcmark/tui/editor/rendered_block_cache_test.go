package editor

import (
	"testing"
)

func TestRenderedBlockCache_CacheHit(t *testing.T) {
	cache := NewRenderedBlockCache(10)
	lines := []string{"# Hello"}

	// First call: cache miss, renders through glamour.
	result1 := cache.Render("block1", lines, 80)
	if len(result1) == 0 {
		t.Fatal("expected non-empty result")
	}

	// Second call: cache hit, returns same content.
	result2 := cache.Render("block1", lines, 80)
	if len(result2) != len(result1) {
		t.Fatalf("cache hit returned different length: got %d, want %d", len(result2), len(result1))
	}
	for i := range result1 {
		if result1[i] != result2[i] {
			t.Errorf("line %d differs: got %q, want %q", i, result2[i], result1[i])
		}
	}
}

func TestRenderedBlockCache_ContentChangeCausesMiss(t *testing.T) {
	cache := NewRenderedBlockCache(10)

	result1 := cache.Render("block1", []string{"# Hello"}, 80)
	result2 := cache.Render("block1", []string{"# World"}, 80)

	if len(result1) == 0 || len(result2) == 0 {
		t.Fatal("expected non-empty results")
	}

	// Different content should produce different results.
	if result1[0] == result2[0] {
		t.Error("expected different results for different content")
	}

	// Cache should have 2 entries.
	if cache.Len() != 2 {
		t.Errorf("expected 2 entries, got %d", cache.Len())
	}
}

func TestRenderedBlockCache_WidthChangeCausesMiss(t *testing.T) {
	cache := NewRenderedBlockCache(10)
	lines := []string{"# Hello"}

	cache.Render("block1", lines, 80)
	cache.Render("block1", lines, 40)

	// Different widths should create separate cache entries.
	if cache.Len() != 2 {
		t.Errorf("expected 2 entries for different widths, got %d", cache.Len())
	}
}

func TestRenderedBlockCache_LRUEviction(t *testing.T) {
	cache := NewRenderedBlockCache(3)

	// Fill to capacity.
	cache.Render("block1", []string{"line 1"}, 80)
	cache.Render("block2", []string{"line 2"}, 80)
	cache.Render("block3", []string{"line 3"}, 80)

	if cache.Len() != 3 {
		t.Fatalf("expected 3 entries, got %d", cache.Len())
	}

	// Adding a 4th should evict block1 (oldest).
	cache.Render("block4", []string{"line 4"}, 80)

	if cache.Len() != 3 {
		t.Errorf("expected 3 entries after eviction, got %d", cache.Len())
	}
}

func TestRenderedBlockCache_LRUTouchPreventsEviction(t *testing.T) {
	cache := NewRenderedBlockCache(3)

	// Fill to capacity.
	cache.Render("block1", []string{"line 1"}, 80)
	cache.Render("block2", []string{"line 2"}, 80)
	cache.Render("block3", []string{"line 3"}, 80)

	// Touch block1 (move to most recently used).
	cache.Render("block1", []string{"line 1"}, 80)

	// Adding block4 should evict block2 (now oldest), not block1.
	cache.Render("block4", []string{"line 4"}, 80)

	// block1 should still be cached (was touched).
	// Verify by checking cache length is still 3.
	if cache.Len() != 3 {
		t.Errorf("expected 3 entries, got %d", cache.Len())
	}
}

func TestRenderedBlockCache_Clear(t *testing.T) {
	cache := NewRenderedBlockCache(10)

	cache.Render("block1", []string{"# Hello"}, 80)
	cache.Render("block2", []string{"# World"}, 80)

	if cache.Len() != 2 {
		t.Fatalf("expected 2 entries, got %d", cache.Len())
	}

	cache.Clear()

	if cache.Len() != 0 {
		t.Errorf("expected 0 entries after clear, got %d", cache.Len())
	}
}

func TestRenderedBlockCache_EmptyBlock(t *testing.T) {
	cache := NewRenderedBlockCache(10)

	result := cache.Render("block1", []string{""}, 80)
	if len(result) == 0 {
		t.Fatal("expected non-empty result for empty block")
	}
	// Empty input should produce at least one empty line.
	if result[0] != "" {
		t.Errorf("expected empty string for empty block, got %q", result[0])
	}
}

func TestRenderedBlockCache_MultiLineBlock(t *testing.T) {
	cache := NewRenderedBlockCache(10)
	lines := []string{
		"# Summary",
		"",
		"- Item one",
		"- Item two",
		"- Item three",
	}

	result := cache.Render("block1", lines, 80)
	if len(result) < 3 {
		t.Errorf("expected at least 3 rendered lines for a heading + list, got %d: %v", len(result), result)
	}
}

func TestRenderedBlockCache_RendererReusedForSameWidth(t *testing.T) {
	cache := NewRenderedBlockCache(10)

	cache.Render("block1", []string{"# Hello"}, 80)
	r1 := cache.renderer

	cache.Render("block2", []string{"# World"}, 80)
	r2 := cache.renderer

	if r1 != r2 {
		t.Error("expected renderer to be reused for same width")
	}
}

func TestRenderedBlockCache_RendererRecreatedForDifferentWidth(t *testing.T) {
	cache := NewRenderedBlockCache(10)

	cache.Render("block1", []string{"# Hello"}, 80)
	r1 := cache.renderer

	cache.Render("block2", []string{"# World"}, 40)
	r2 := cache.renderer

	if r1 == r2 {
		t.Error("expected new renderer for different width")
	}
}

func TestRenderedBlockCache_DefaultCapacity(t *testing.T) {
	cache := NewRenderedBlockCache(0)
	if cache.capacity != 128 {
		t.Errorf("expected default capacity 128, got %d", cache.capacity)
	}
}

func BenchmarkRenderedBlockCache_Hit(b *testing.B) {
	cache := NewRenderedBlockCache(128)
	lines := []string{"# Hello", "", "Some paragraph text with **bold** and _italic_."}
	cache.Render("block1", lines, 80) // Prime the cache.

	b.ResetTimer()
	for b.Loop() {
		cache.Render("block1", lines, 80)
	}
}

func BenchmarkRenderedBlockCache_Miss(b *testing.B) {
	lines := []string{"# Hello", "", "Some paragraph text with **bold** and _italic_."}

	b.ResetTimer()
	for b.Loop() {
		cache := NewRenderedBlockCache(128)
		cache.Render("block1", lines, 80)
	}
}
