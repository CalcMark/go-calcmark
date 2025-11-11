package validator

import (
	"github.com/CalcMark/go-calcmark/impl/evaluator"
	"github.com/CalcMark/go-calcmark/spec/lexer"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// ValidateCalculation performs semantic validation on a calculation without evaluating it
func ValidateCalculation(line string, context *evaluator.Context) *ValidationResult {
	if context == nil {
		context = evaluator.NewContext()
	}

	var diagnostics []*Diagnostic

	// Tokenize for pre-parse checks
	tokens, tokErr := lexer.Tokenize(line)
	if tokErr == nil {
		// Check for ambiguous modulus usage
		diagnostics = append(diagnostics, checkAmbiguousModulus(tokens)...)

		// Check for invalid currency format (100USD instead of USD100)
		if diag := checkInvalidCurrencyFormat(tokens); diag != nil {
			diagnostics = append(diagnostics, diag)
		}
	}

	// Try to parse
	nodes, err := parser.Parse(line)
	if err != nil {
		diagnostics = append(diagnostics, createSyntaxErrorDiagnostic(err, len(line)))
		return NewValidationResult(diagnostics)
	}

	if len(nodes) == 0 {
		return NewValidationResult(diagnostics)
	}

	// Run all validation checks on each node
	for _, node := range nodes {
		diagnostics = append(diagnostics, findInvalidPercentageOperations(node)...)

		for _, ident := range findUndefinedIdentifiers(node, context) {
			diagnostics = append(diagnostics, createUndefinedVariableDiagnostic(ident, len(line)))
		}
	}

	return NewValidationResult(diagnostics)
}

// ValidateDocument validates an entire document line by line
func ValidateDocument(document string, initialContext *evaluator.Context) map[int]*ValidationResult {
	context := initialContext
	if context == nil {
		context = evaluator.NewContext()
	}

	results := make(map[int]*ValidationResult)
	lines := splitDocumentIntoLines(document)

	for lineNum := range lines {
		if shouldSkipLine(lines[lineNum]) {
			continue
		}

		ctx := createLineContext(lineNum, lines)
		result := validateSingleLine(ctx, lines, context)

		// Store results if there are any diagnostics
		if len(result.Diagnostics) > 0 {
			results[lineNum+1] = result // 1-indexed line numbers
		}
	}

	return results
}
