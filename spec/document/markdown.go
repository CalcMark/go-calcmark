//go:build !wasm

package document

import (
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// Render converts the markdown source to HTML.
// This implementation is only available in native Go builds (not WASM).
func (tb *TextBlock) Render() string {
	if !tb.dirty && tb.html != "" {
		return tb.html // Return cached HTML
	}

	tb.html = renderMarkdown(tb.InterpolatedSourceText())
	tb.dirty = false

	return tb.html
}

// renderMarkdown converts markdown source to HTML using gomarkdown.
func renderMarkdown(source string) string {
	if source == "" {
		return ""
	}

	// gomarkdown requires a trailing newline to correctly parse certain
	// constructs (e.g., setext headings). SourceText() joins lines with \n
	// but does not append a final newline.
	if source[len(source)-1] != '\n' {
		source += "\n"
	}

	// CommonMark-only parser extensions (no GFM: Tables, Strikethrough, DefinitionLists excluded)
	extensions := parser.NoIntraEmphasis | parser.FencedCode | parser.Autolink |
		parser.SpaceHeadings | parser.HeadingIDs | parser.BackslashLineBreak |
		parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)

	// Parse markdown to AST
	doc := p.Parse([]byte(source))

	// Create HTML renderer with security flags:
	// - SkipHTML: strip raw HTML to prevent XSS
	// - Safelink: block javascript:, vbscript:, data: URI schemes
	// - HrefTargetBlank: open links in new tab
	htmlFlags := html.CommonFlags | html.HrefTargetBlank | html.SkipHTML | html.Safelink
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	// Render to HTML
	htmlBytes := markdown.Render(doc, renderer)

	return string(htmlBytes)
}
