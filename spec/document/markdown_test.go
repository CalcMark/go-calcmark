package document

import (
	"strings"
	"testing"
)

func TestMarkdownRendering(t *testing.T) {
	// This test only runs in native Go builds (not WASM)
	// Skip in WASM to avoid panic

	source := []string{
		"# Heading 1",
		"",
		"This is **bold** text.",
	}

	block := NewTextBlock(source)

	html := block.Render()

	if html == "" {
		t.Error("Expected rendered HTML, got empty string")
	}

	// Check that HTML contains expected elements
	if !strings.Contains(html, "<h1") {
		t.Error("Expected <h1> tag in rendered HTML")
	}

	if !strings.Contains(html, "<strong>") {
		t.Error("Expected <strong> tag for bold text")
	}

	t.Logf("Rendered HTML:\n%s", html)
}

func TestMarkdownCaching(t *testing.T) {
	source := []string{"# Test"}
	block := NewTextBlock(source)

	// First render
	html1 := block.Render()

	// Second render (should use cache)
	html2 := block.Render()

	if html1 != html2 {
		t.Error("Expected cached HTML to match")
	}

	// Mark dirty and re-render
	block.SetDirty(true)
	html3 := block.Render()

	if html3 != html1 {
		t.Error("Expected re-rendered HTML to match original")
	}
}
