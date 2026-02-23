package editor

import (
	"strings"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// BlockRenderResult maps rendered preview lines back to their source lines for alignment.
type BlockRenderResult struct {
	// SourceLineIndex is the 0-based index of the source line within the block
	SourceLineIndex int
	// PreviewLines are the rendered/wrapped lines for this source line
	PreviewLines []string
}

// RenderTextBlockAligned renders an entire TextBlock as markdown and maps the output
// back to source lines for 1:1 alignment in the TUI.
//
// This function is functionally pure - it takes a TextBlock and rendering width,
// and returns deterministic output without side effects.
//
// The function handles multi-line markdown constructs (ordered lists, unordered lists,
// blockquotes, etc.) correctly by rendering the entire block at once using glamour,
// then mapping the output back to source lines.
func RenderTextBlockAligned(block *document.TextBlock, renderer *MarkdownRenderer) []BlockRenderResult {
	if block == nil {
		return nil
	}

	source := block.Source()
	if len(source) == 0 {
		return nil
	}

	// Render the entire block as a single markdown document
	// This allows glamour to correctly handle multi-line constructs like ordered lists
	blockText := strings.Join(source, "\n")
	renderedLines := renderer.RenderLine(blockText)

	// Now we need to map rendered lines back to source lines
	// This is the tricky part - we need to maintain alignment

	// Strategy: Count non-empty source lines and distribute rendered output proportionally
	results := make([]BlockRenderResult, 0, len(source))

	renderedIdx := 0
	for srcIdx, srcLine := range source {
		trimmed := strings.TrimSpace(srcLine)

		if trimmed == "" {
			// Empty source line -> empty preview line
			results = append(results, BlockRenderResult{
				SourceLineIndex: srcIdx,
				PreviewLines:    []string{""},
			})
		} else {
			// Non-empty source line -> take next rendered line(s)
			previewLines := []string{}

			if renderedIdx < len(renderedLines) {
				// 1:1 mapping — wrapping is handled by the alignment layer (aligned.go).
				previewLines = []string{renderedLines[renderedIdx]}
				renderedIdx++
			} else {
				// Ran out of rendered lines - use source as fallback
				previewLines = []string{srcLine}
			}

			results = append(results, BlockRenderResult{
				SourceLineIndex: srcIdx,
				PreviewLines:    previewLines,
			})
		}
	}

	return results
}

// RenderTextBlockSimple renders an entire TextBlock and returns just the preview lines.
// This is a simpler interface for testing and cases where alignment isn't needed.
func RenderTextBlockSimple(block *document.TextBlock, renderer *MarkdownRenderer) []string {
	if block == nil {
		return nil
	}

	source := block.Source()
	if len(source) == 0 {
		return nil
	}

	blockText := strings.Join(source, "\n")
	return renderer.RenderLine(blockText)
}
