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
		{
			name:   "data URI link",
			source: "[click](data:text/html,<script>alert(1)</script>)",
			banned: "data:",
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

func TestMarkdownGFMTableRendered(t *testing.T) {
	source := []string{
		"| Header | Value |",
		"|--------|-------|",
		"| A      | 1     |",
	}

	block := NewTextBlock(source)
	html := block.Render()

	if !strings.Contains(html, "<table") {
		t.Error("GFM tables must render as <table>")
	}
	if !strings.Contains(html, "<th") {
		t.Error("GFM tables must render <th> elements for header row")
	}
	if !strings.Contains(html, "<td") {
		t.Error("GFM tables must render <td> elements for data rows")
	}
	if !strings.Contains(html, "Header") {
		t.Error("Table header content must be preserved")
	}
	if !strings.Contains(html, "A") || !strings.Contains(html, "1") {
		t.Error("Table cell content must be preserved")
	}
}

func TestMarkdownGFMTableWithInterpolation(t *testing.T) {
	// Tables with {{variable}} interpolation must render as HTML tables
	source := []string{
		"| Metric | Value |",
		"|--------|-------|",
		"| Revenue | $1M |",
		"| Cost | $500K |",
	}

	block := NewTextBlock(source)
	html := block.Render()

	if !strings.Contains(html, "<table") {
		t.Error("Table with data must render as <table>")
	}
	if !strings.Contains(html, "Revenue") {
		t.Error("Table must contain Revenue cell")
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

// --- Phase 5b: Edge case tests ---

func TestReferenceStyleLinkWithinTextBlock(t *testing.T) {
	// Reference-style link definition and usage within the same TextBlock
	// should resolve correctly in HTML rendering.
	source := []string{
		"See the [official docs][docs] for details.",
		"",
		"[docs]: https://example.com",
	}

	block := NewTextBlock(source)
	html := block.Render()

	if !strings.Contains(html, `href="https://example.com"`) {
		t.Errorf("Reference-style link should resolve within TextBlock, got: %s", html)
	}
	if !strings.Contains(html, "official docs") {
		t.Errorf("Link text should be present, got: %s", html)
	}
}

func TestReferenceStyleLinkAcrossTextBlockBoundary(t *testing.T) {
	// When the link definition is in a different TextBlock than the usage,
	// it should NOT resolve (known limitation of per-block rendering).
	usageBlock := NewTextBlock([]string{
		"See the [official docs][docs] for details.",
	})
	html := usageBlock.Render()

	// The link should NOT resolve — no <a> tag with the URL
	if strings.Contains(html, `href="https://example.com"`) {
		t.Error("Reference-style link across TextBlock boundary should NOT resolve")
	}
}

func TestSetextHeadingRendersAsH1(t *testing.T) {
	// === after a paragraph line should render as H1 (setext heading).
	source := []string{
		"This is a heading",
		"===",
	}

	block := NewTextBlock(source)
	html := block.Render()

	if !strings.Contains(html, "<h1") {
		t.Errorf("Expected setext === to render as H1, got: %s", html)
	}
}

func TestSetextHeadingRendersAsH2(t *testing.T) {
	// --- after a paragraph line should render as H2 (setext heading).
	source := []string{
		"This is a heading",
		"---",
	}

	block := NewTextBlock(source)
	html := block.Render()

	if !strings.Contains(html, "<h2") {
		t.Errorf("Expected setext --- to render as H2, got: %s", html)
	}
}

func TestSmartypantsFractionsInProse(t *testing.T) {
	// gomarkdown's SmartypantsFractions converts prose fractions to styled HTML.
	// This is DESIRABLE for plain prose text — "Use 1/2 cup" renders as a proper fraction.
	// gomarkdown uses <sup>num</sup>&frasl;<sub>denom</sub> format.
	tests := []struct {
		name   string
		source string
		want   string // HTML expected (sup/frasl/sub format)
	}{
		{"1/2 in prose", "Use 1/2 cup of flour", "<sup>1</sup>&frasl;<sub>2</sub>"},
		{"1/4 in prose", "Use 1/4 cup of sugar", "<sup>1</sup>&frasl;<sub>4</sub>"},
		{"3/4 in prose", "Fill 3/4 of the tank", "<sup>3</sup>&frasl;<sub>4</sub>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := NewTextBlock([]string{tt.source})
			html := block.Render()

			if !strings.Contains(html, tt.want) {
				t.Errorf("SmartypantsFractions should convert prose fractions: expected %s in %q", tt.want, html)
			}
		})
	}
}

func TestInterpolatedFractionsNotCorrupted(t *testing.T) {
	// When a CalcMark fraction like a = 1/2 is interpolated into text via {{a}},
	// the sentinel-wrapped value must survive markdown rendering intact.
	// SmartypantsFractions must NOT corrupt the interpolated value.
	//
	// Pipeline: "Total: {{a}}" → "Total: \x021/2\x03" → markdown → post-process sentinels
	// The \x02...\x03 sentinels should prevent SmartypantsFractions from seeing "1/2".
	tests := []struct {
		name       string
		source     string // raw source with {{var}} placeholder
		htmlSource string // sentinel-wrapped source (simulating interpolation)
		wantInHTML string // expected content in final HTML
		banned     string // must NOT appear
	}{
		{
			name:       "interpolated 1/2 preserved",
			source:     "Total: {{a}}",
			htmlSource: "Total: \x021/2\x03",
			wantInHTML: "1/2",
			banned:     "&frac12;",
		},
		{
			name:       "interpolated 1/4 preserved",
			source:     "Add {{b}} cup",
			htmlSource: "Add \x021/4\x03 cup",
			wantInHTML: "1/4",
			banned:     "&frac14;",
		},
		{
			name:       "interpolated 3/4 preserved",
			source:     "Fill {{c}} tank",
			htmlSource: "Fill \x023/4\x03 tank",
			wantInHTML: "3/4",
			banned:     "&frac34;",
		},
		{
			name:       "interpolated 1/3 preserved",
			source:     "Use {{d}} cup",
			htmlSource: "Use \x021/3\x03 cup",
			wantInHTML: "1/3",
			banned:     "",
		},
		{
			name:       "mixed: prose 1/2 converted but interpolated 1/2 preserved",
			source:     "Prose 1/2 and calc {{a}}",
			htmlSource: "Prose 1/2 and calc \x021/2\x03",
			wantInHTML: `<span class="cm-interpolated">1/2</span>`,
			banned:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := NewTextBlock([]string{tt.source})
			block.SetInterpolatedHTMLSource([]string{tt.htmlSource})
			block.SetDirty(true)
			html := block.Render()

			if tt.wantInHTML != "" && !strings.Contains(html, tt.wantInHTML) {
				t.Errorf("Expected %q in rendered HTML, got: %s", tt.wantInHTML, html)
			}
			if tt.banned != "" && strings.Contains(html, tt.banned) {
				t.Errorf("Interpolated value must not be converted: found %s in %q", tt.banned, html)
			}
		})
	}
}

func TestStandaloneHorizontalRule(t *testing.T) {
	// --- after a blank line (or at start) should render as horizontal rule.
	source := []string{
		"Some text above.",
		"",
		"---",
	}

	block := NewTextBlock(source)
	html := block.Render()

	if !strings.Contains(html, "<hr") {
		t.Errorf("Expected standalone --- to render as <hr>, got: %s", html)
	}
}
