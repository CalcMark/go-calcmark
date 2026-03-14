//go:build !wasm

package document

import (
	"strings"

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

	rendered := renderMarkdown(tb.InterpolatedHTMLSourceText())

	// Post-process sentinel markers into <span> tags for interpolated values.
	// Sentinels (\x02 and \x03) survive markdown rendering as literal characters.
	rendered = strings.ReplaceAll(rendered, "\x02", `<span class="cm-interpolated">`)
	rendered = strings.ReplaceAll(rendered, "\x03", `</span>`)

	tb.html = rendered
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
	// - SmartypantsFractions: converts prose 1/2→½, 1/4→¼, 3/4→¾ (desirable for text)
	htmlFlags := html.CommonFlags | html.HrefTargetBlank | html.SkipHTML | html.Safelink
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	// Render to HTML
	htmlBytes := markdown.Render(doc, renderer)

	return string(htmlBytes)
}
