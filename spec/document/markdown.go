//go:build !wasm

package document

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
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

// renderMarkdown converts markdown source to HTML using goldmark.
// Raw HTML in the source is escaped by default (goldmark's safe mode).
func renderMarkdown(source string) string {
	if source == "" {
		return ""
	}

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		return ""
	}

	return buf.String()
}
