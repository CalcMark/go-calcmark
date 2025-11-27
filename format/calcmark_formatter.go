package format

import (
	"fmt"
	"io"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// CalcMarkFormatter formats CalcMark documents back to CalcMark format.
// Useful for round-tripping and formatted output with optional result comments.
type CalcMarkFormatter struct{}

// Extensions returns the file extensions handled by this formatter.
func (f *CalcMarkFormatter) Extensions() []string {
	return []string{".cm", ".calcmark"}
}

// Format writes the document as CalcMark source to the writer.
// In verbose mode, adds result as comments after calculation lines.
func (f *CalcMarkFormatter) Format(w io.Writer, doc *document.Document, opts Options) error {
	blocks := doc.GetBlocks()

	for i, node := range blocks {
		// Write source lines
		for _, line := range node.Block.Source() {
			fmt.Fprintln(w, line)
		}

		// Optionally add result as comment
		if opts.Verbose {
			if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
				if calcBlock.LastValue() != nil {
					fmt.Fprintf(w, "  # = %s\n", calcBlock.LastValue().String())
				}
			}
		}

		// Add spacing between blocks (except after the last one)
		if i < len(blocks)-1 {
			fmt.Fprintln(w)
		}
	}

	return nil
}
