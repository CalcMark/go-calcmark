package validator

import (
	"strings"

	"github.com/CalcMark/go-calcmark/ast"
	"github.com/CalcMark/go-calcmark/evaluator"
	"github.com/CalcMark/go-calcmark/parser"
)

// findUndefinedIdentifiers walks the AST and finds all undefined identifier references
func findUndefinedIdentifiers(node ast.Node, context *evaluator.Context) []*ast.Identifier {
	var undefined []*ast.Identifier

	var walk func(ast.Node)
	walk = func(n ast.Node) {
		switch node := n.(type) {
		case *ast.Identifier:
			// Check if identifier is defined in context or is a boolean keyword
			if !context.Has(node.Name) && !isBooleanKeyword(node.Name) {
				undefined = append(undefined, node)
			}

		case *ast.BinaryOp:
			walk(node.Left)
			walk(node.Right)

		case *ast.ComparisonOp:
			walk(node.Left)
			walk(node.Right)

		case *ast.Assignment:
			// Only check the RHS, not the variable being assigned
			walk(node.Value)

		case *ast.Expression:
			walk(node.Expr)
		}
	}

	walk(node)
	return undefined
}

// ValidateCalculation performs semantic validation on a calculation without evaluating it
func ValidateCalculation(line string, context *evaluator.Context) *ValidationResult {
	if context == nil {
		context = evaluator.NewContext()
	}

	var diagnostics []*Diagnostic

	// Try to parse
	nodes, err := parser.Parse(line)
	if err != nil {
		// Parse/lexer error - can't validate further
		diagnostics = append(diagnostics, &Diagnostic{
			Severity: Error,
			Code:     SyntaxError,
			Message:  err.Error(),
			Range: &ast.Range{
				Start: ast.Position{Line: 1, Column: 1},
				End:   ast.Position{Line: 1, Column: len(line) + 1},
			},
		})
		return NewValidationResult(diagnostics)
	}

	// No AST nodes
	if len(nodes) == 0 {
		return NewValidationResult(diagnostics)
	}

	// Check each statement
	for _, node := range nodes {
		// Find undefined variables
		undefined := findUndefinedIdentifiers(node, context)

		for _, ident := range undefined {
			// Use the identifier's position if available
			errorRange := ident.Range
			if errorRange == nil {
				// Fallback to whole line
				errorRange = &ast.Range{
					Start: ast.Position{Line: 1, Column: 1},
					End:   ast.Position{Line: 1, Column: len(line) + 1},
				}
			}

			diagnostics = append(diagnostics, &Diagnostic{
				Severity:     Error,
				Code:         UndefinedVariable,
				Message:      "Undefined variable: " + ident.Name,
				Range:        errorRange,
				VariableName: ident.Name,
			})
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
	lines := strings.Split(document, "\n")

	for lineNum, line := range lines {
		lineNumber := lineNum + 1 // 1-indexed

		// Skip blank lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// For now, validate all non-blank lines as calculations
		// (classifier integration will be added later)
		result := ValidateCalculation(line, context)

		// Store if there are any diagnostics
		if len(result.Diagnostics) > 0 {
			results[lineNumber] = result
		}

		// If the line is valid, try to update context
		if result.IsValid() {
			// Try to parse and extract assignments
			nodes, err := parser.Parse(line)
			if err == nil {
				for _, node := range nodes {
					if assign, ok := node.(*ast.Assignment); ok {
						// Add variable to context (with dummy value)
						// We don't evaluate, just track that it exists
						num, _ := evaluator.NewContext().Get("true") // Get any valid type
						context.Set(assign.Name, num)
					}
				}
			}
		}
	}

	return results
}
