// Package document provides document structure and parsing for CalcMark.
package document

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// ParsedGlobals contains parsed global variable values ready for injection
// into the interpreter environment.
type ParsedGlobals struct {
	Values map[string]types.Type
}

// ParseGlobals parses the raw string values from frontmatter into typed CalcMark values.
// Only literal values are allowed (no expressions or calculations):
//   - Numbers: 42, 3.14, 1.5K, 25%
//   - Quantities: 10 meters, 5 kg, 100 MB
//   - Currencies: $100, 50 EUR
//   - Dates: Jan 15 2025, today
//   - Durations: 5 days, 2 weeks
//   - Booleans: true, false
//   - Rates: 100 MB/s, $50/hour
//
// Returns an error if any value cannot be parsed or is not a literal.
func ParseGlobals(rawGlobals map[string]string) (*ParsedGlobals, error) {
	result := &ParsedGlobals{
		Values: make(map[string]types.Type),
	}

	for name, exprStr := range rawGlobals {
		value, err := parseGlobalValue(name, exprStr)
		if err != nil {
			return nil, err
		}
		result.Values[name] = value
	}

	return result, nil
}

// parseGlobalValue parses a single global variable value as a CalcMark literal.
func parseGlobalValue(name, exprStr string) (types.Type, error) {
	// Ensure expression ends with newline for parser
	input := exprStr
	if len(input) > 0 && input[len(input)-1] != '\n' {
		input = input + "\n"
	}

	// Parse the expression
	nodes, err := parser.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("invalid value for '%s': %w", name, err)
	}

	if len(nodes) != 1 {
		return nil, fmt.Errorf("invalid value for '%s': expected a single literal value", name)
	}

	// Check that it's a literal (not an expression with operators)
	if !isLiteralNode(nodes[0]) {
		return nil, fmt.Errorf("invalid value for '%s': only literal values allowed (numbers, quantities, currencies, dates, durations, rates), not expressions", name)
	}

	// Evaluate the literal using the lightweight literal evaluator
	value, err := evalLiteral(nodes[0])
	if err != nil {
		return nil, fmt.Errorf("invalid value for '%s': %w", name, err)
	}

	return value, nil
}

// isLiteralNode checks if an AST node is a literal value (not an expression).
func isLiteralNode(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.NumberLiteral,
		*ast.QuantityLiteral,
		*ast.CurrencyLiteral,
		*ast.DateLiteral,
		*ast.RelativeDateLiteral,
		*ast.DurationLiteral,
		*ast.TimeLiteral,
		*ast.BooleanLiteral,
		*ast.RateLiteral:
		return true
	case *ast.Expression:
		// Unwrap Expression and check inner node
		return isLiteralNode(n.Expr)
	default:
		return false
	}
}
