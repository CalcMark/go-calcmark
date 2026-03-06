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

func TestMarkdownRawHTMLStripped(t *testing.T) {
	// Raw HTML in markdown source must be stripped for security (XSS prevention)
	source := []string{
		"<script>alert('xss')</script>",
		"",
		"Normal paragraph.",
	}

	block := NewTextBlock(source)
	html := block.Render()

	if strings.Contains(html, "<script>") {
		t.Error("Raw <script> tags must be stripped from rendered HTML")
	}
	if strings.Contains(html, "alert") {
		t.Error("Script content must be stripped from rendered HTML")
	}
	if !strings.Contains(html, "Normal paragraph") {
		t.Error("Non-HTML content should still render")
	}
}

func TestMarkdownUnsafeLinkSchemes(t *testing.T) {
	// javascript: and other unsafe URI schemes must be blocked
	tests := []struct {
		name   string
		source string
		banned string
	}{
		{
			name:   "javascript link",
			source: "[click](javascript:alert(1))",
			banned: "javascript:",
		},
		{
			name:   "vbscript link",
			source: "[click](vbscript:MsgBox)",
			banned: "vbscript:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := NewTextBlock([]string{tt.source})
			html := block.Render()

			if strings.Contains(html, tt.banned) {
				t.Errorf("Unsafe link scheme %q must be blocked, got: %s", tt.banned, html)
			}
		})
	}
}

func TestMarkdownRawHTMLBlocks(t *testing.T) {
	// Various raw HTML patterns must be stripped
	tests := []struct {
		name   string
		source string
		banned string
	}{
		{"div tag", "<div>content</div>", "<div>"},
		{"iframe", "<iframe src='evil.com'></iframe>", "<iframe"},
		{"img onerror", "<img src=x onerror=alert(1)>", "onerror"},
		{"style tag", "<style>body{display:none}</style>", "<style>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := NewTextBlock([]string{tt.source})
			html := block.Render()

			if strings.Contains(html, tt.banned) {
				t.Errorf("Raw HTML %q must be stripped, got: %s", tt.banned, html)
			}
		})
	}
}

func TestMarkdownGFMTableNotRendered(t *testing.T) {
	// GFM tables should not render as HTML tables (CommonMark only)
	source := []string{
		"| Header | Value |",
		"|--------|-------|",
		"| A      | 1     |",
	}

	block := NewTextBlock(source)
	html := block.Render()

	if strings.Contains(html, "<table") {
		t.Error("GFM tables must not render as <table> (CommonMark only)")
	}
	if strings.Contains(html, "<th") {
		t.Error("GFM tables must not render <th> elements")
	}
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
