package format

import (
	"fmt"
	"io"
	"strings"

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
	// Serialize frontmatter first (if present and not suppressed)
	if !opts.SuppressFrontmatter {
		if fm := doc.GetFrontmatter(); fm != nil {
			fmStr := fm.Serialize()
			if fmStr != "" {
				if opts.FrontmatterAsCodeFence {
					// Render as a visible code fence instead of raw --- delimiters.
					// This avoids collisions when the output is embedded in a Hugo
					// page that has its own YAML frontmatter.
					inner := strings.TrimSpace(fmStr)
					inner = strings.TrimPrefix(inner, "---")
					inner = strings.TrimSuffix(inner, "---")
					inner = strings.TrimSpace(inner)
					if inner != "" {
						fmt.Fprintf(w, "```yaml\n%s\n```\n\n", inner)
					}
				} else {
					fmt.Fprint(w, fmStr)
				}
			}
		}
	}

	blocks := doc.GetBlocks()
	df := opts.getFormatter()

	for _, node := range blocks {
		switch block := node.Block.(type) {
		case *document.CalcBlock:
			// Skip blocks that only contain frontmatter @ assignments
			if isOnlyFrontmatterBlockMd(block) {
				continue
			}

			fmt.Fprintf(w, "```calcmark\n")
			stmts := AlignResults(block)

			// Trim trailing blank statements
			for len(stmts) > 0 && stmts[len(stmts)-1].IsBlank {
				stmts = stmts[:len(stmts)-1]
			}

			for _, stmt := range stmts {
				if stmt.IsResultLine {
					continue
				}
				if stmt.IsBlank {
					fmt.Fprintln(w)
					continue
				}
				fmt.Fprint(w, stmt.Source)
				if stmt.Result != nil {
					fmt.Fprintf(w, " → %s", df.Format(stmt.Result))
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
			for _, line := range block.InterpolatedSource() {
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
