package format

import (
	"fmt"
	"io"

	"github.com/CalcMark/go-calcmark/v2/spec/document"
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
func (f *TextFormatter) Format(w io.Writer, doc *document.Document, opts Options) error {
	blocks := doc.GetBlocks()
	df := opts.getFormatter()

	// In non-verbose mode, output each per-statement result (one per line, no extra spacing)
	if !opts.Verbose {
		for _, node := range blocks {
			if block, ok := node.Block.(*document.CalcBlock); ok {
				if block.Error() != nil {
					fmt.Fprintf(w, "Error: %v\n", block.Error())
				} else {
					for _, result := range block.Results() {
						if result != nil {
							fmt.Fprintln(w, df.Format(result))
						}
					}
				}
			}
		}
		return nil
	}

	// Verbose mode: show frontmatter and source with results
	if fm := doc.GetFrontmatter(); fm != nil {
		hasFrontmatter := len(fm.Globals) > 0 || len(fm.Exchange) > 0 || fm.Scale != nil || fm.ConvertTo != nil
		if hasFrontmatter {
			fmt.Fprintln(w, "--- Frontmatter ---")
			for _, name := range fm.GlobalKeys() {
				fmt.Fprintf(w, "  %s = %s\n", name, fm.Globals[name])
			}
			for _, key := range fm.ExchangeKeys() {
				rate := fm.Exchange[key]
				from, to, err := document.ParseExchangeRateKey(key)
				if err == nil {
					fmt.Fprintf(w, "  %s → %s: %s\n", from, to, rate.StringFixed(4))
				} else {
					fmt.Fprintf(w, "  %s: %s\n", key, rate.StringFixed(4))
				}
			}
			if fm.Scale != nil {
				fmt.Fprintf(w, "  scale: %s\n", fm.Scale.Factor.String())
			}
			if fm.ConvertTo != nil {
				fmt.Fprintf(w, "  convert_to: %s\n", fm.ConvertTo.System)
			}
			fmt.Fprintln(w)
		}
	}

	for i, node := range blocks {
		switch block := node.Block.(type) {
		case *document.CalcBlock:
			stmts := AlignResults(block)
			for _, stmt := range stmts {
				if stmt.IsBlank {
					continue
				}
				fmt.Fprint(w, stmt.Source)
				if stmt.Result != nil {
					fmt.Fprintf(w, " → %s", df.Format(stmt.Result))
				}
				fmt.Fprintln(w)
			}

			if block.Error() != nil {
				fmt.Fprintf(w, "Error: %v\n", block.Error())
			}

		case *document.TextBlock:
			for _, line := range block.InterpolatedSource() {
				fmt.Fprintln(w, line)
			}
		}

		if i < len(blocks)-1 {
			fmt.Fprintln(w)
		}
	}

	return nil
}
