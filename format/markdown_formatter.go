package format

import (
	"fmt"
	"io"

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
	blocks := doc.GetBlocks()

	for _, node := range blocks {
		switch block := node.Block.(type) {
		case *document.CalcBlock:
			// Format calculation blocks in fenced code blocks
			fmt.Fprintf(w, "```calcmark\n")
			for _, line := range block.Source() {
				fmt.Fprintln(w, line)
			}
			fmt.Fprintf(w, "```\n\n")

			if block.Error() != nil {
				fmt.Fprintf(w, "**Error:** %v\n\n", block.Error())
			} else if block.LastValue() != nil {
				fmt.Fprintf(w, "**Result:** %s\n\n", block.LastValue().String())
			}

		case *document.TextBlock:
			// Pass through markdown text as-is
			for _, line := range block.Source() {
				fmt.Fprintln(w, line)
			}
			fmt.Fprintln(w)
		}
	}

	return nil
}
