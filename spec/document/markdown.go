//go:build !wasm

package document

import (
	"bytes"
	"fmt"
	"regexp"
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

// crossBlockMarkerFmt is the per-block delimiter goldmark sees when
// joined-rendering. The ALL-CAPS form contains no characters
// goldmark treats as inline markdown (no underscores, asterisks,
// brackets, backticks) — surrounding blank lines force each marker
// into its own paragraph, which the post-render splitter can
// recognise without a state machine. The `%d` anchors the marker to
// its block index so the same pattern can't collide with anything
// the user might type.
const crossBlockMarkerFmt = "CMTBSPLITxX%dXxCMTBSPLIT"

// crossBlockMarkerRE matches a rendered marker — `<p>CMTBSPLITxXNXxCMTBSPLIT</p>`,
// possibly surrounded by whitespace. Used by RenderTextBlocksJoined
// to split the joined render back into per-block HTML.
var crossBlockMarkerRE = regexp.MustCompile(`(?m)^\s*<p>CMTBSPLITxX(\d+)XxCMTBSPLIT</p>\s*$`)

// RenderTextBlocksJoined renders multiple text blocks as one
// markdown pass and returns the per-block HTML plus any
// document-wide trailing section (footnotes definitions, etc.) that
// only goldmark's whole-document pipeline can produce.
//
// Why this exists: goldmark's footnote extension only resolves
// references whose definitions appear in the SAME markdown document.
// When CalcMark splits prose into separate TextBlocks around calc
// blocks, an in-text `[^note]` reference and its trailing
// `[^note]: definition` paragraph end up in distinct renders and the
// reference falls back to literal text. Issue #129. The fix is to
// join all blocks' sources with unique delimiter markers, render
// once so goldmark sees the whole document, then split the rendered
// HTML back into per-block pieces.
//
// Returns:
//   - blockHTML[i] is the HTML for blocks[i], with `\x02`/`\x03`
//     sentinels left in place so the caller can do its own
//     interpolation-span post-processing.
//   - trailing is the document-level HTML that follows the last
//     block in the joined render (typically the
//     `<section class="footnotes">…</section>` from goldmark's
//     footnote extension). Callers attach it wherever footnotes
//     should appear — usually appended to the last text block or
//     rendered as a doc-level footer.
//
// `ok` reports whether the joined render succeeded. When ok is
// false, callers should fall back to per-block rendering — the
// per-block HTML values returned are still safe to use but
// footnotes etc. won't resolve across blocks.
//
// Empty input returns (nil, "", true).
func RenderTextBlocksJoined(blocks []*TextBlock) (blockHTML []string, trailing string, ok bool) {
	if len(blocks) == 0 {
		return nil, "", true
	}

	// Step 1: join all block sources with paragraph-isolated markers.
	// Blank lines around each marker force goldmark to render it as
	// its own paragraph rather than splicing it into adjacent text.
	var b strings.Builder
	for i, blk := range blocks {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(blk.InterpolatedHTMLSourceText())
		b.WriteString("\n\n")
		fmt.Fprintf(&b, crossBlockMarkerFmt, i)
	}

	rendered := renderMarkdown(b.String())

	// Step 2: split on each rendered marker paragraph. The regexp
	// captures the block index so we can verify the markers came
	// out in order (defence against goldmark reordering blocks).
	matches := crossBlockMarkerRE.FindAllStringSubmatchIndex(rendered, -1)
	if len(matches) != len(blocks) {
		// Marker survival failed (likely a goldmark version change
		// or a future extension that mangles raw-text paragraphs).
		// Signal failure so the caller falls back to per-block
		// rendering — footnotes won't resolve across blocks, but
		// the document still renders.
		return nil, "", false
	}

	blockHTML = make([]string, len(blocks))
	cursor := 0
	for i, m := range matches {
		blockStart := m[0] // index of the marker paragraph
		blockHTML[i] = strings.TrimSpace(rendered[cursor:blockStart])
		cursor = m[1] // index just past the marker paragraph
	}
	// Anything after the last marker (the footnotes section) is the
	// document-level trailing block.
	trailing = strings.TrimSpace(rendered[cursor:])

	return blockHTML, trailing, true
}
