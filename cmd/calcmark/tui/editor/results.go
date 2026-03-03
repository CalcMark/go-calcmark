package editor

import (
	"strings"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// LineResult represents a line's evaluation result.
// This is the bridge between the document model and the view layer.
type LineResult struct {
	LineNum        int
	Source         string
	IsCalc         bool
	IsFrontmatter  bool // True if this line is part of the YAML frontmatter block
	VarName        string
	Value          string
	Error          string               // Legacy error string (for backwards compatibility)
	Diagnostic     *document.Diagnostic // Structured diagnostic with code, message, position
	BlockID        string
	WasChanged     bool
	IsBlocked      bool     // True if this error is caused by an undefined variable from a prior error
	ReferencedVars []string // Variable names referenced by this statement (AST-derived, sorted)
}

// GetLineResults returns evaluation results for all lines.
// Each source line maps to its corresponding statement result when available.
// Frontmatter lines are prepended as empty (non-calc) results to match GetLines().
func (m *Model) GetLineResults() []LineResult {
	allLines := m.GetLines()
	results := make([]LineResult, 0, len(allLines))
	lineNum := 0

	// Prepend empty results for frontmatter lines to maintain alignment
	// with GetLines() which includes frontmatter
	fmCount := m.frontmatterLineCount()
	if fmCount > 0 {
		for i := range fmCount {
			source := ""
			if i < len(allLines) {
				source = allLines[i]
			}
			results = append(results, LineResult{
				LineNum:       lineNum,
				Source:        source,
				IsCalc:        false,
				IsFrontmatter: true,
			})
			lineNum++
		}
	}

	// Track variables that failed to evaluate (for cascading error detection)
	// Per CONTEXT.md: "Cascading errors: show root cause only, dependents show 'blocked'"
	blockedVars := make(map[string]bool)

	for _, node := range m.doc.GetBlocks() {
		switch b := node.Block.(type) {
		case *document.CalcBlock:
			sourceLines := b.Source()
			stmtResults := b.Results()   // Per-statement results
			statements := b.Statements() // Parsed AST nodes
			blockError := b.Error()

			// Build error line map from diagnostics (which have proper position info)
			// Map from 1-indexed block line to structured diagnostic
			diagnostics := b.Diagnostics()
			diagByLine := make(map[int]*document.Diagnostic)
			for i := range diagnostics {
				diag := &diagnostics[i]
				if diag.Line > 0 {
					diagByLine[diag.Line] = diag
				}
			}

			// If we have a block error but no diagnostics with position, fall back to heuristics
			errorLineIdx := -1
			if blockError != nil && len(diagByLine) == 0 {
				errorLineIdx = findErrorLine(sourceLines, blockError.Error())
			}

			for i, line := range sourceLines {
				lr := LineResult{
					LineNum:    lineNum,
					Source:     line,
					IsCalc:     true,
					BlockID:    node.ID,
					WasChanged: m.changedBlockIDs[node.ID],
				}

				// Skip empty/whitespace-only lines (no result to show)
				trimmed := line
				for _, c := range line {
					if c != ' ' && c != '\t' {
						trimmed = line
						break
					}
				}
				if len(trimmed) == 0 || trimmed == "" {
					results = append(results, lr)
					lineNum++
					continue
				}

				// Map source line index to statement index
				// This is approximate - assumes 1:1 mapping of non-empty lines to statements
				stmtIdx := countNonEmptyLinesBefore(sourceLines, i)

				// Check for diagnostic errors on this line (i+1 because diag.Line is 1-indexed)
				blockLineNum := i + 1
				if diag, hasError := diagByLine[blockLineNum]; hasError {
					lr.Diagnostic = diag
					lr.Error = diag.Code + ": " + diag.Message // Legacy string for backwards compat

					// Check if this is a cascading (blocked) error
					if isUndefinedVarError(lr.Error) {
						varName := extractVarFromUndefinedError(lr.Error)
						if varName != "" && blockedVars[varName] {
							lr.IsBlocked = true
						}
					}

					// If this line defines a variable and has an error, add it to blockedVars
					// Note: Don't set lr.VarName here to preserve original behavior (error lines
					// don't report VarName)
					if stmtIdx < len(statements) {
						if varName := getAssignmentVarName(statements[stmtIdx]); varName != "" {
							blockedVars[varName] = true
						}
					}

					results = append(results, lr)
					lineNum++
					continue
				}

				// Fallback: If there's a block-level error without position info
				if blockError != nil && len(diagByLine) == 0 {
					showErrorHere := false
					if errorLineIdx >= 0 {
						showErrorHere = (i == errorLineIdx)
					} else {
						// Last resort: show on first non-empty line (stmtIdx == 0)
						showErrorHere = (stmtIdx == 0)
					}
					if showErrorHere {
						lr.Error = blockError.Error()

						// Check if this is a cascading (blocked) error
						if isUndefinedVarError(lr.Error) {
							varName := extractVarFromUndefinedError(lr.Error)
							if varName != "" && blockedVars[varName] {
								lr.IsBlocked = true
							}
						}

						// If this line defines a variable and has an error, add it to blockedVars
						// Note: Don't set lr.VarName here to preserve original behavior (error lines
						// don't report VarName)
						if stmtIdx < len(statements) {
							if varName := getAssignmentVarName(statements[stmtIdx]); varName != "" {
								blockedVars[varName] = true
							}
						}

						results = append(results, lr)
						lineNum++
						continue
					}
				}

				// Get result for this statement if available
				if stmtIdx < len(stmtResults) && stmtResults[stmtIdx] != nil {
					lr.Value = m.displayFormat(stmtResults[stmtIdx])
				}

				// Get variable name if this statement defines one (assignment)
				// Anonymous calculations (like "2 + 2") don't have a variable name
				if stmtIdx < len(statements) {
					if varName := getAssignmentVarName(statements[stmtIdx]); varName != "" {
						lr.VarName = varName
					}
					lr.ReferencedVars = document.ExtractStatementReferences(statements[stmtIdx])
				}

				results = append(results, lr)
				lineNum++
			}

		case *document.TextBlock:
			for _, line := range b.Source() {
				results = append(results, LineResult{
					LineNum: lineNum,
					Source:  line,
					IsCalc:  false,
					BlockID: node.ID,
				})
				lineNum++
			}
		}
	}

	return results
}

// countNonEmptyLinesBefore counts non-empty lines before index i.
func countNonEmptyLinesBefore(lines []string, i int) int {
	count := 0
	for j := range i {
		if strings.TrimSpace(lines[j]) != "" {
			count++
		}
	}
	return count
}

// findErrorLine tries to determine which source line caused the error by
// searching for context clues in the error message.
// Returns the source line index, or -1 if not determinable.
func findErrorLine(sourceLines []string, errMsg string) int {
	// Common pattern: "undefined variable: \"varname\"" (case-insensitive)
	// Extract the variable name and find which line references it
	lowerErr := strings.ToLower(errMsg)
	if strings.Contains(lowerErr, "undefined variable") {
		// Extract variable name from error (format: "... \"varname\" ...")
		start := strings.Index(errMsg, "\"")
		if start >= 0 {
			end := strings.Index(errMsg[start+1:], "\"")
			if end >= 0 {
				varName := errMsg[start+1 : start+1+end]
				// Find which line references this variable (not defines it)
				for i, line := range sourceLines {
					trimmed := strings.TrimSpace(line)
					if trimmed == "" {
						continue
					}
					// Skip if this line defines the variable (left side of =)
					if strings.HasPrefix(trimmed, varName+" ") && strings.Contains(trimmed, "=") {
						continue
					}
					// Check if line contains the variable name
					if strings.Contains(line, varName) {
						return i
					}
				}
			}
		}
	}

	// Common pattern: syntax errors often include line/column info
	// For now, just return -1 and fall back to first line
	return -1
}

// getAssignmentVarName extracts the variable name from an assignment AST node.
func getAssignmentVarName(node ast.Node) string {
	switch n := node.(type) {
	case *ast.Assignment:
		return n.Name
	}
	return ""
}

// isUndefinedVarError checks if an error message indicates an undefined variable.
func isUndefinedVarError(errMsg string) bool {
	lowerErr := strings.ToLower(errMsg)
	return strings.Contains(lowerErr, "undefined variable") ||
		strings.Contains(lowerErr, "undefined_variable")
}

// extractVarFromUndefinedError extracts the variable name from an undefined variable error.
// Returns empty string if no variable name found.
func extractVarFromUndefinedError(errMsg string) string {
	// Common patterns:
	// - `undefined_variable: Undefined variable "varname" - ...`
	// - `undefined variable: "varname"`
	start := strings.Index(errMsg, "\"")
	if start >= 0 {
		end := strings.Index(errMsg[start+1:], "\"")
		if end >= 0 {
			return errMsg[start+1 : start+1+end]
		}
	}
	return ""
}
