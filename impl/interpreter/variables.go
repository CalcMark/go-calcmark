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
		// Reject zero and negative rates (same validation as frontmatter parser)
		if rate.IsZero() || rate.IsNegative() {
			return nil, fmt.Errorf("@exchange.%s: exchange rate must be positive, got %s", f.Property, rate.String())
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
// NOTE: This mirrors document.ParseExchangeRateKey in spec/document/frontmatter.go.
// We cannot import spec/document here because spec/document/eval_flow_debug_test.go
// (internal test package) imports impl/interpreter, which would create a cycle.
func parseExchangeKey(key string) (from, to string, err error) {
	parts := strings.Split(key, "_")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid exchange rate key '%s': expected format 'FROM_TO' (e.g., 'USD_EUR')", key)
	}
	from = strings.TrimSpace(strings.ToUpper(parts[0]))
	to = strings.TrimSpace(strings.ToUpper(parts[1]))
	if from == "" || to == "" {
		return "", "", fmt.Errorf("invalid exchange rate key '%s': currency codes cannot be empty", key)
	}
	if !isValidCurrencyCode(from) {
		return "", "", fmt.Errorf("invalid currency code '%s' in exchange rate key '%s': must be exactly 3 letters (e.g., USD, EUR)", from, key)
	}
	if !isValidCurrencyCode(to) {
		return "", "", fmt.Errorf("invalid currency code '%s' in exchange rate key '%s': must be exactly 3 letters (e.g., USD, EUR)", to, key)
	}
	return from, to, nil
}

// isValidCurrencyCode checks if a string looks like a valid currency code:
// exactly 3 ASCII letters.
func isValidCurrencyCode(code string) bool {
	if len(code) != 3 {
		return false
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
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
