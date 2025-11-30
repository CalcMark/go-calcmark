package format

import (
	"fmt"
	"io"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// CalcMarkFormatter formats CalcMark documents back to CalcMark format.
// Useful for round-tripping and saving documents with frontmatter.
type CalcMarkFormatter struct{}

// Extensions returns the file extensions handled by this formatter.
func (f *CalcMarkFormatter) Extensions() []string {
	return []string{".cm", ".calcmark"}
}

// Format writes the document as CalcMark source to the writer.
// Output is exactly the source as typed, ensuring reproducibility.
func (f *CalcMarkFormatter) Format(w io.Writer, doc *document.Document, opts Options) error {
	// Serialize frontmatter first (if present)
	if fm := doc.GetFrontmatter(); fm != nil {
		fmStr := fm.Serialize()
		if fmStr != "" {
			fmt.Fprint(w, fmStr)
		}
	}

	blocks := doc.GetBlocks()

	// Filter out blocks that only contain frontmatter assignments
	// (they're now serialized in the YAML frontmatter above)
	filteredBlocks := make([]*document.BlockNode, 0, len(blocks))
	for _, node := range blocks {
		if !isOnlyFrontmatterBlock(node) {
			filteredBlocks = append(filteredBlocks, node)
		}
	}

	for i, node := range filteredBlocks {
		source := node.Block.Source()
		isLastBlock := i == len(filteredBlocks)-1

		// Write source lines, filtering out legacy "# = ..." result comments
		for j, line := range source {
			isLastLine := j == len(source)-1

			// Skip the trailing empty line for the last block
			if isLastBlock && isLastLine && line == "" {
				continue
			}

			// Skip result lines from previous saves (both legacy "# =" and new "→")
			if isResultLine(line) {
				continue
			}

			fmt.Fprint(w, line)
			fmt.Fprint(w, "\n")
		}

		// Add block boundary between blocks (except after the last one)
		// Skip if the block already ends with empty lines (they serve as the separator)
		if !isLastBlock {
			endsWithEmpty := len(source) > 0 && source[len(source)-1] == ""
			if !endsWithEmpty {
				fmt.Fprint(w, "\n")
			}
		}
	}

	return nil
}

// isResultLine checks if a line is a result output from previous saves.
// Handles both legacy "# = ..." format and new "→ ..." format.
func isResultLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "# =") || strings.HasPrefix(trimmed, "→")
}

// isOnlyFrontmatterBlock returns true if the block only contains @ assignments.
// These blocks should be filtered out when saving since their content is
// now in the YAML frontmatter.
func isOnlyFrontmatterBlock(node *document.BlockNode) bool {
	calcBlock, ok := node.Block.(*document.CalcBlock)
	if !ok {
		return false
	}

	// Check if all non-empty lines start with @
	for _, line := range calcBlock.Source() {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, "@") {
			return false
		}
	}

	// All lines are either empty or @ assignments
	return true
}
