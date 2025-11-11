package validator

import (
	"strings"

	"github.com/CalcMark/go-calcmark/constants"
	"github.com/CalcMark/go-calcmark/impl/evaluator"
	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// lineContext holds context information for validating a single line
type lineContext struct {
	line       string
	lineNum    int
	isBlank    bool
	isFirstLine bool
	isLastLine bool
}

// createBlankLineHints generates hints for missing blank lines around calculations
func createBlankLineHints(ctx lineContext, lines []string) []*Diagnostic {
	var hints []*Diagnostic

	// Only suggest blank lines for valid calculations
	if ctx.isBlank || !canParseAsCalculation(ctx.line) {
		return hints
	}

	hasPrevBlank := ctx.isFirstLine || isBlankLine(lines[ctx.lineNum-1])
	hasNextBlank := ctx.isLastLine || isBlankLine(lines[ctx.lineNum+1])

	if !hasPrevBlank {
		hints = append(hints, &Diagnostic{
			Severity: Hint,
			Code:     BlankLineIsolation,
			Message:  "Consider adding a blank line before this calculation for better readability",
			Range:    nil,
		})
	}

	if !hasNextBlank {
		hints = append(hints, &Diagnostic{
			Severity: Hint,
			Code:     BlankLineIsolation,
			Message:  "Consider adding a blank line after this calculation for better readability",
			Range:    nil,
		})
	}

	return hints
}

// isBlankLine checks if a line is blank per ENCODING_SPEC.md
func isBlankLine(s string) bool {
	return constants.IsBlankLine(s)
}

// canParseAsCalculation checks if a line can be parsed as a calculation
func canParseAsCalculation(s string) bool {
	if isBlankLine(s) {
		return false
	}
	_, err := parser.Parse(s)
	return err == nil
}

// extractAssignments finds all variable assignments in a list of AST nodes
func extractAssignments(nodes []ast.Node) []string {
	var assignments []string

	for _, node := range nodes {
		if assign, ok := node.(*ast.Assignment); ok {
			assignments = append(assignments, assign.Name)
		}
	}

	return assignments
}

// updateContextWithAssignments adds assigned variables to the context
// This is for tracking variable existence, not evaluating values
func updateContextWithAssignments(context *evaluator.Context, assignments []string) {
	// Get a dummy value to represent "variable exists"
	dummyValue, _ := evaluator.NewContext().Get("true")

	for _, varName := range assignments {
		context.Set(varName, dummyValue)
	}
}

// validateSingleLine validates one line and returns its result
func validateSingleLine(ctx lineContext, lines []string, context *evaluator.Context) *ValidationResult {
	// Validate the line
	result := ValidateCalculation(ctx.line, context)

	// Add blank line hints for valid calculations
	if result.IsValid() {
		result.Diagnostics = append(result.Diagnostics, createBlankLineHints(ctx, lines)...)
	}

	// Update context if the line is valid and has assignments
	if result.IsValid() {
		nodes, err := parser.Parse(ctx.line)
		if err == nil {
			assignments := extractAssignments(nodes)
			updateContextWithAssignments(context, assignments)
		}
	}

	return result
}

// splitDocumentIntoLines splits a document and returns lines with metadata
func splitDocumentIntoLines(document string) []string {
	return strings.Split(document, constants.Newline)
}

// shouldSkipLine determines if a line should be skipped during validation
func shouldSkipLine(line string) bool {
	return isBlankLine(line)
}

// createLineContext creates context for a specific line
func createLineContext(lineNum int, lines []string) lineContext {
	return lineContext{
		line:        lines[lineNum],
		lineNum:     lineNum,
		isBlank:     isBlankLine(lines[lineNum]),
		isFirstLine: lineNum == 0,
		isLastLine:  lineNum == len(lines)-1,
	}
}
