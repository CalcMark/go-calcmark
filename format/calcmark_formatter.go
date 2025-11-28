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
		source := node.Block.Source()
		isLastBlock := i == len(blocks)-1

		// Write source lines exactly as stored (lines don't include \n)
		for j, line := range source {
			isLastLine := j == len(source)-1

			// Skip the trailing empty line for the last block
			// (it was created by splitLines and represents "trailing \n")
			// The previous line's \n already provides the final newline
			if isLastBlock && isLastLine && line == "" {
				continue
			}

			fmt.Fprint(w, line)
			fmt.Fprint(w, "\n")
		}

		// Optionally add result as comment (verbose mode only)
		if opts.Verbose {
			if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
				if calcBlock.LastValue() != nil {
					fmt.Fprintf(w, "  # = %s\n", calcBlock.LastValue().String())
				}
			}
		}

		// Add block boundary between blocks (except after the last one)
		// Block boundary = 2 consecutive empty lines = \n\n after content\n
		// Source already includes one empty line from trailing \n, so add one more \n
		if !isLastBlock {
			fmt.Fprint(w, "\n")
		}
	}

	return nil
}
