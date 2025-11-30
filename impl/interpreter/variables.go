package interpreter

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// Variable and identifier evaluation.

func (interp *Interpreter) evalAssignment(a *ast.Assignment) (types.Type, error) {
	value, err := interp.evalNode(a.Value)
	if err != nil {
		return nil, err
	}

	interp.env.Set(a.Name, value)
	return value, nil
}

// evalFrontmatterAssignment evaluates a frontmatter variable assignment.
// This updates the interpreter's environment with exchange rates or global variables.
// The actual Document frontmatter storage is handled by the evaluator layer.
func (interp *Interpreter) evalFrontmatterAssignment(f *ast.FrontmatterAssignment) (types.Type, error) {
	value, err := interp.evalNode(f.Value)
	if err != nil {
		return nil, err
	}

	switch f.Namespace {
	case "exchange":
		// Exchange rates require a numeric value
		rate, err := types.ToDecimal(value)
		if err != nil {
			return nil, fmt.Errorf("@exchange.%s: rate must be a number, got %T", f.Property, value)
		}
		// Parse FROM_TO format
		from, to, err := parseExchangeKey(f.Property)
		if err != nil {
			return nil, fmt.Errorf("@exchange.%s: %v", f.Property, err)
		}
		interp.env.SetExchangeRate(from, to, rate)
		return value, nil

	case "global":
		// Globals can be any type (number, currency, quantity, etc.)
		interp.env.Set(f.Property, value)
		return value, nil

	default:
		// Parser already validates this, but be safe
		return nil, fmt.Errorf("unknown frontmatter namespace: %s", f.Namespace)
	}
}

// parseExchangeKey parses an exchange key like "USD_EUR" into (from, to).
func parseExchangeKey(key string) (from, to string, err error) {
	parts := strings.Split(key, "_")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid format: expected FROM_TO (e.g., 'USD_EUR')")
	}
	from = strings.TrimSpace(strings.ToUpper(parts[0]))
	to = strings.TrimSpace(strings.ToUpper(parts[1]))
	if from == "" || to == "" {
		return "", "", fmt.Errorf("currency codes cannot be empty")
	}
	return from, to, nil
}

func (interp *Interpreter) evalIdentifier(id *ast.Identifier) (types.Type, error) {
	// Check for defined variables FIRST (variables take precedence over keywords)
	if value, ok := interp.env.Get(id.Name); ok {
		return value, nil
	}

	// Then check for boolean keywords
	if isBooleanKeyword(id.Name) {
		value, _ := parseBooleanValue(id.Name)
		return types.NewBoolean(value), nil
	}

	// Undefined variable
	return nil, fmt.Errorf("undefined variable: %q", id.Name)
}
