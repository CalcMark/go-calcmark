package validator

import (
	"strings"

	"github.com/CalcMark/go-calcmark/impl/evaluator"
	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/lexer"
	"golang.org/x/text/currency"
)

// findUndefinedIdentifiers walks the AST and finds all undefined identifier references
func findUndefinedIdentifiers(node ast.Node, context *evaluator.Context) []*ast.Identifier {
	var undefined []*ast.Identifier

	walkAST(node, func(n ast.Node) {
		if ident, ok := n.(*ast.Identifier); ok {
			if !context.Has(ident.Name) && !isBooleanKeyword(ident.Name) {
				undefined = append(undefined, ident)
			}
		}
	})

	return undefined
}

// findInvalidPercentageOperations walks the AST and finds percentage literals on left side of +/-
func findInvalidPercentageOperations(node ast.Node) []*Diagnostic {
	var diagnostics []*Diagnostic

	walkAST(node, func(n ast.Node) {
		if binOp, ok := n.(*ast.BinaryOp); ok {
			if isInvalidPercentageOperation(binOp) {
				diagnostics = append(diagnostics, createPercentageDiagnostic(binOp))
			}
		}
	})

	return diagnostics
}

// isInvalidPercentageOperation checks if a binary operation has percentage on left of +/-
func isInvalidPercentageOperation(binOp *ast.BinaryOp) bool {
	if binOp.Operator != "+" && binOp.Operator != "-" {
		return false
	}

	numLit, ok := binOp.Left.(*ast.NumberLiteral)
	if !ok {
		return false
	}

	return strings.Contains(numLit.SourceText, "%")
}

// createPercentageDiagnostic creates a diagnostic for invalid percentage operation
func createPercentageDiagnostic(binOp *ast.BinaryOp) *Diagnostic {
	var message string
	if binOp.Operator == "+" {
		message = "Cannot add to a percentage literal. Did you mean to write the expression in reverse order (e.g., '5 + 20%' equals 6)?"
	} else {
		message = "Cannot subtract from a percentage literal. Percentages can only be applied to other values (e.g., '100 - 20%' equals 80)."
	}

	return &Diagnostic{
		Severity: Error,
		Code:     PercentageOnLeftOfAdditionSubtraction,
		Message:  message,
		Range:    binOp.Range,
	}
}

// checkAmbiguousModulus checks for modulus operators without spaces (e.g., "10%3")
func checkAmbiguousModulus(tokens []lexer.Token) []*Diagnostic {
	var diagnostics []*Diagnostic

	for i := 0; i < len(tokens)-1; i++ {
		if isAmbiguousModulusPattern(tokens, i) {
			diagnostics = append(diagnostics, createAmbiguousModulusDiagnostic(tokens, i))
		}
	}

	return diagnostics
}

// isAmbiguousModulusPattern checks if token pattern matches ambiguous modulus
func isAmbiguousModulusPattern(tokens []lexer.Token, i int) bool {
	if tokens[i].Type != lexer.MODULUS || tokens[i+1].Type != lexer.NUMBER {
		return false
	}

	// Check if there's NO space between them (EndPos == StartPos)
	return tokens[i].EndPos == tokens[i+1].StartPos
}

// createAmbiguousModulusDiagnostic creates a diagnostic for ambiguous modulus
func createAmbiguousModulusDiagnostic(tokens []lexer.Token, i int) *Diagnostic {
	precedingNum := ""
	if i > 0 && tokens[i-1].Type == lexer.NUMBER {
		precedingNum = tokens[i-1].Value + " "
	}

	message := "Modulus operator (%) immediately followed by number may be confused with percentage notation. Consider adding a space: '" +
		precedingNum + "% " + tokens[i+1].Value + "'"

	return &Diagnostic{
		Severity: Hint,
		Code:     AmbiguousModulus,
		Message:  message,
		Range: &ast.Range{
			Start: ast.Position{Line: 1, Column: tokens[i].Column},
			End:   ast.Position{Line: 1, Column: tokens[i+1].Column + len(tokens[i+1].Value)},
		},
	}
}

// createUndefinedVariableDiagnostic creates a diagnostic for undefined variable
func createUndefinedVariableDiagnostic(ident *ast.Identifier, lineLength int) *Diagnostic {
	errorRange := ident.Range
	if errorRange == nil {
		// Fallback to whole line
		errorRange = &ast.Range{
			Start: ast.Position{Line: 1, Column: 1},
			End:   ast.Position{Line: 1, Column: lineLength + 1},
		}
	}

	return &Diagnostic{
		Severity:     Error,
		Code:         UndefinedVariable,
		Message:      "Undefined variable: " + ident.Name,
		Range:        errorRange,
		VariableName: ident.Name,
	}
}

// createSyntaxErrorDiagnostic creates a diagnostic for parse/syntax errors
func createSyntaxErrorDiagnostic(err error, lineLength int) *Diagnostic {
	return &Diagnostic{
		Severity: Error,
		Code:     SyntaxError,
		Message:  err.Error(),
		Range: &ast.Range{
			Start: ast.Position{Line: 1, Column: 1},
			End:   ast.Position{Line: 1, Column: lineLength + 1},
		},
	}
}

// checkInvalidCurrencyFormat detects NUMBER followed by currency code IDENTIFIER
// and provides a helpful diagnostic for common mistakes like "100USD"
func checkInvalidCurrencyFormat(tokens []lexer.Token) *Diagnostic {
	for i := 0; i < len(tokens)-1; i++ {
		if tokens[i].Type != lexer.NUMBER {
			continue
		}

		nextToken := tokens[i+1]
		if nextToken.Type != lexer.IDENTIFIER {
			continue
		}

		// Check if the identifier is a valid currency code
		if !isValidCurrencyCode(nextToken.Value) {
			continue
		}

		// Found NUMBER followed by currency code
		var message string
		if tokens[i].EndPos == nextToken.StartPos {
			// No space between: "100USD"
			message = "Invalid currency format. Use prefix format (e.g., " +
				nextToken.Value + tokens[i].Value + ") or currency symbols (e.g., $" + tokens[i].Value + ")"
		} else {
			// With space: "100 USD"
			message = "Postfix currency format not yet supported. Use prefix format " +
				nextToken.Value + tokens[i].Value + " instead of " + tokens[i].Value + " " + nextToken.Value
		}

		return &Diagnostic{
			Severity: Error,
			Code:     InvalidCurrencyFormat,
			Message:  message,
			Range: &ast.Range{
				Start: ast.Position{Line: 1, Column: tokens[i].Column},
				End:   ast.Position{Line: 1, Column: nextToken.Column + len(nextToken.Value)},
			},
		}
	}

	return nil
}

// isValidCurrencyCode checks if a string is a valid ISO 4217 currency code
func isValidCurrencyCode(s string) bool {
	if len(s) != 3 {
		return false
	}

	// Must be all uppercase
	for _, ch := range s {
		if ch < 'A' || ch > 'Z' {
			return false
		}
	}

	// Validate against ISO 4217
	unit, err := currency.ParseISO(s)
	if err != nil {
		return false
	}

	// Verify it's a real currency (not XXX or similar)
	return unit.String() != "XXX"
}
