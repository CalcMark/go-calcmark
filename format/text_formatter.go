package format

import (
	"fmt"
	"io"

	"github.com/CalcMark/go-calcmark/format/display"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// TextFormatter formats CalcMark documents as plain text.
// This is the primary formatter for interactive use (REPL, CLI).
type TextFormatter struct{}

// Extensions returns the file extensions handled by this formatter.
func (f *TextFormatter) Extensions() []string {
	return []string{".txt"}
}

// Format writes the document as plain text to the writer.
// In verbose mode, it shows source with intermediate results for each line.
// All output uses the centralized Type.String() methods for display.
func (f *TextFormatter) Format(w io.Writer, doc *document.Document, opts Options) error {
	blocks := doc.GetBlocks()

	// In non-verbose mode, only output calc block results (one per line, no extra spacing)
	if !opts.Verbose {
		for _, node := range blocks {
			if block, ok := node.Block.(*document.CalcBlock); ok {
				if block.Error() != nil {
					fmt.Fprintf(w, "Error: %v\n", block.Error())
				} else if block.LastValue() != nil {
					fmt.Fprintln(w, display.Format(block.LastValue()))
				}
			}
		}
		return nil
	}

	// Verbose mode: show source with results and spacing between blocks
	for i, node := range blocks {
		switch block := node.Block.(type) {
		case *document.CalcBlock:
			// Show source with inline results for each line
			sourceLines := block.Source()
			results := block.Results()

			for j, line := range sourceLines {
				if line == "" {
					continue
				}
				fmt.Fprint(w, line)
				// Add result if available for this line
				if j < len(results) && results[j] != nil {
					fmt.Fprintf(w, " → %s", display.Format(results[j]))
				}
				fmt.Fprintln(w)
			}

			// Show error in verbose mode too
			if block.Error() != nil {
				fmt.Fprintf(w, "Error: %v\n", block.Error())
			}

		case *document.TextBlock:
			// For text blocks, show markdown content
			for _, line := range block.Source() {
				fmt.Fprintln(w, line)
			}
		}

		// Add spacing between blocks (except after the last one)
		if i < len(blocks)-1 {
			fmt.Fprintln(w)
		}
	}

	return nil
}
