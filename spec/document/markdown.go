//go:build !wasm
// +build !wasm

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

	tb.html = renderMarkdown(tb.SourceText())
	tb.dirty = false

	return tb.html
}

// renderMarkdown converts markdown source to HTML using gomarkdown.
func renderMarkdown(source string) string {
	if source == "" {
		return ""
	}

	// Create parser with CommonMark extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)

	// Parse markdown to AST
	doc := p.Parse([]byte(source))

	// Create HTML renderer
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	// Render to HTML
	htmlBytes := markdown.Render(doc, renderer)

	return string(htmlBytes)
}
