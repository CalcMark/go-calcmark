package format

import (
	"strings"

	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// AlignedStatement maps a source line to its evaluated result.
// This is the shared logic extracted from all formatters to prevent
// the recurring source-line/result-index alignment bugs.
type AlignedStatement struct {
	// Source is the original source line text.
	Source string

	// Result is the raw, unformatted evaluation result.
	// Nil if this line has no result (blank, result-comment, etc.).
	Result types.Type

	// Variable is the name assigned by this statement (e.g., "x" from "x = 10").
	// Empty for anonymous expressions.
	Variable string

	// IsBlank is true for empty/whitespace-only source lines.
	IsBlank bool

	// IsResultLine is true for previous result comments (e.g., "# = 42", "→ 42").
	// Formatters should typically skip these.
	IsResultLine bool
}

// AlignResults maps source lines of a calc block to their corresponding
// evaluation results. It handles the shared alignment logic that was
// previously duplicated across all four formatters:
//   - Result lines (from previous saves) are marked but not assigned results
//   - Blank lines are marked but not assigned results
//   - Non-blank, non-result lines consume the next result from block.Results()
//   - Variable names are extracted from assignment expressions
func AlignResults(block *document.CalcBlock) []AlignedStatement {
	sourceLines := block.Source()
	results := block.Results()

	aligned := make([]AlignedStatement, 0, len(sourceLines))
	resultIdx := 0

	for _, line := range sourceLines {
		stmt := AlignedStatement{Source: line}

		trimmed := strings.TrimSpace(line)

		// Mark result lines from previous saves
		if isResultLine(line) {
			stmt.IsResultLine = true
			aligned = append(aligned, stmt)
			continue
		}

		// Mark blank lines
		if trimmed == "" {
			stmt.IsBlank = true
			aligned = append(aligned, stmt)
			continue
		}

		// Non-blank, non-result line: consume the next evaluation result
		if resultIdx < len(results) && results[resultIdx] != nil {
			stmt.Result = results[resultIdx]
		}
		resultIdx++

		// Extract variable name from assignment (left of '=')
		if eqIdx := strings.Index(line, "="); eqIdx > 0 {
			// Exclude == and !=
			if eqIdx+1 < len(line) && line[eqIdx+1] != '=' {
				if eqIdx == 0 || line[eqIdx-1] != '!' {
					varName := strings.TrimSpace(line[:eqIdx])
					if varName != "" && !strings.ContainsAny(varName, " \t+*/-") {
						stmt.Variable = varName
					}
				}
			}
		}

		aligned = append(aligned, stmt)
	}

	return aligned
}
