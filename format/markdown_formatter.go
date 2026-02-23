package format

import (
	"fmt"
	"io"
	"strings"

	"github.com/CalcMark/go-calcmark/format/display"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// MarkdownFormatter formats CalcMark documents as Markdown.
// Calculation blocks are shown in code fences with results.
type MarkdownFormatter struct{}

// Extensions returns the file extensions handled by this formatter.
func (f *MarkdownFormatter) Extensions() []string {
	return []string{".md", ".markdown"}
}

// Format writes the document as Markdown to the writer.
func (f *MarkdownFormatter) Format(w io.Writer, doc *document.Document, opts Options) error {
	// Serialize frontmatter first (if present)
	if fm := doc.GetFrontmatter(); fm != nil {
		fmStr := fm.Serialize()
		if fmStr != "" {
			fmt.Fprint(w, fmStr)
		}
	}

	blocks := doc.GetBlocks()

	for _, node := range blocks {
		switch block := node.Block.(type) {
		case *document.CalcBlock:
			// Skip blocks that only contain frontmatter @ assignments
			if isOnlyFrontmatterBlockMd(block) {
				continue
			}

			// Format calculation blocks in fenced code blocks with per-line results
			fmt.Fprintf(w, "```calcmark\n")
			sourceLines := block.Source()
			results := block.Results()

			// Trim trailing empty lines to avoid blank line before closing fence
			for len(sourceLines) > 0 && strings.TrimSpace(sourceLines[len(sourceLines)-1]) == "" {
				sourceLines = sourceLines[:len(sourceLines)-1]
			}

			for i, line := range sourceLines {
				// Skip result lines from previous saves
				if isResultLine(line) {
					continue
				}
				if line == "" {
					fmt.Fprintln(w)
					continue
				}
				fmt.Fprint(w, line)
				// Append inline result if available
				if i < len(results) && results[i] != nil {
					fmt.Fprintf(w, " → %s", display.Format(results[i]))
				}
				fmt.Fprintln(w)
			}
			fmt.Fprintf(w, "```\n\n")

			if block.Error() != nil {
				fmt.Fprintf(w, "**Error:** %v\n\n", block.Error())
			}

		case *document.TextBlock:
			// Skip text blocks that are just result lines from verbose saves
			if isResultBlock(block) {
				continue
			}
			// Pass through markdown text as-is
			for _, line := range block.Source() {
				fmt.Fprintln(w, line)
			}
			fmt.Fprintln(w)
		}
	}

	return nil
}

// isOnlyFrontmatterBlockMd returns true if the block only contains @ assignments.
func isOnlyFrontmatterBlockMd(block *document.CalcBlock) bool {
	for _, line := range block.Source() {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, "@") {
			return false
		}
	}
	return true
}

// isResultBlock checks if a text block only contains result lines.
// These appear when verbose-saved .cm files have result lines that
// get detected as text blocks on reload.
func isResultBlock(block *document.TextBlock) bool {
	for _, line := range block.Source() {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, "# =") && !strings.HasPrefix(trimmed, "→") {
			return false
		}
	}
	return true
}
