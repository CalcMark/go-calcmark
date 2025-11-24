package evaluator

import (
	"fmt"
	"math"
	"strings"

	"github.com/CalcMark/go-calcmark/impl/types"
	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/shopspring/decimal"
)

// Binary Operations Module
//
// This module handles all binary operations (+, -, *, /, %, ^) in CalcMark.
// It implements special semantics for percentage literals and type coercion rules.
//
// Key responsibilities:
// 1. Percentage handling: "100 + 20%" means 100 + (100 * 0.20) = 120
// 2. Type preservation: Operations on Currency preserve the currency symbol
// 3. Mixed unit handling: Different currencies drop to Number type

// isPercentageLiteral checks if a type represents a percentage literal (e.g., "20%")
// Percentage literals are stored as Numbers with "%" in their SourceFormat field.
func isPercentageLiteral(t types.Type) bool {
	if num, ok := t.(*types.Number); ok {
		return strings.Contains(num.SourceFormat, "%")
	}
	return false
}

// applyPercentageAddition computes x + p% = x + (x * p)
// Example: 100 + 20% = 100 + (100 * 0.20) = 120
// This is pure functional - no side effects, just decimal math.
func applyPercentageAddition(base, percentage decimal.Decimal) decimal.Decimal {
	return base.Add(base.Mul(percentage))
}

// applyPercentageSubtraction computes x - p% = x - (x * p)
// Example: 120 - 20% = 120 - (120 * 0.20) = 96
// This is pure functional - no side effects, just decimal math.
func applyPercentageSubtraction(base, percentage decimal.Decimal) decimal.Decimal {
	return base.Sub(base.Mul(percentage))
}

// computeBinaryOp performs the actual mathematical operation.
// This is the core computation engine - takes operator and values, returns result.
// Pure function with no side effects except error returns for division by zero.
//
// rightIsPercentage flag activates special percentage semantics for + and -.
func computeBinaryOp(operator string, left, right decimal.Decimal, rightIsPercentage bool, nodeRange *ast.Range) (decimal.Decimal, error) {
	switch operator {
	case "+":
		if rightIsPercentage {
			return applyPercentageAddition(left, right), nil
		}
		return left.Add(right), nil

	case "-":
		if rightIsPercentage {
			return applyPercentageSubtraction(left, right), nil
		}
		return left.Sub(right), nil

	case "*", "×":
		return left.Mul(right), nil

	case "/", "÷":
		if right.IsZero() {
			return decimal.Zero, &EvaluationError{Message: "Division by zero", Range: nodeRange}
		}
		return left.Div(right), nil

	case "%":
		if right.IsZero() {
			return decimal.Zero, &EvaluationError{Message: "Division by zero", Range: nodeRange}
		}
		return left.Mod(right), nil

	case "**", "^":
		return computeExponentiation(left, right), nil

	default:
		return decimal.Zero, &EvaluationError{
			Message: fmt.Sprintf("Unknown operator: %s", operator),
			Range:   nodeRange,
		}
	}
}

// computeExponentiation handles both integer and floating-point exponents.
// Uses decimal.Pow for integers to maintain precision.
// Falls back to math.Pow for non-integer exponents (with precision loss).
//
// Pure function - no side effects.
func computeExponentiation(base, exponent decimal.Decimal) decimal.Decimal {
	if exponent.IsInteger() {
		expInt := exponent.IntPart()
		if expInt >= 0 {
			// Positive integer exponent: use decimal.Pow for precision
			return base.Pow(decimal.NewFromInt(expInt))
		}
		// Negative exponent: 1 / x^|exp|
		posExp := decimal.NewFromInt(-expInt)
		return decimal.NewFromInt(1).Div(base.Pow(posExp))
	}

	// Non-integer exponent: must use math.Pow (loses precision)
	baseFloat, _ := base.Float64()
	expFloat, _ := exponent.Float64()
	result := math.Pow(baseFloat, expFloat)
	return decimal.NewFromFloat(result)
}

// determineBinaryResultType determines the output type based on input types.
// Implements CalcMark's type coercion rules:
// - Same currency: preserve the currency
// - Different currencies: drop to Number
// - One currency: preserve it (except for number / currency = Number)
// - No currency: return Number
//
// Special case: number / currency returns Number (inverse rate, unit drops)
//
// Returns (resultTypeName, currencySymbol) where symbol is empty for Number type.
// Pure function - no side effects, just type analysis.
func determineBinaryResultType(left, right types.Type, operator string) (string, string) {
	leftCurrency, leftIsCurrency := left.(*types.Currency)
	rightCurrency, rightIsCurrency := right.(*types.Currency)

	// Both are currencies
	if leftIsCurrency && rightIsCurrency {
		if leftCurrency.Symbol == rightCurrency.Symbol {
			return "Currency", leftCurrency.Symbol
		}
		// Different currency units - return Number (no units)
		return "Number", ""
	}

	// Special case: number / currency = Number (inverse rate, drop unit)
	if !leftIsCurrency && rightIsCurrency && (operator == "/" || operator == "÷") {
		return "Number", ""
	}

	// Only left is currency
	if leftIsCurrency {
		return "Currency", leftCurrency.Symbol
	}

	// Only right is currency
	if rightIsCurrency {
		return "Currency", rightCurrency.Symbol
	}

	// Neither is currency
	return "Number", ""
}

// createResultType constructs the appropriate Type based on type name and value.
// Factory function that wraps decimal values in the correct Type.
// Pure function - always returns a new Type instance.
func createResultType(typeName, symbol string, value decimal.Decimal) types.Type {
	if typeName == "Currency" {
		return &types.Currency{Value: value, Symbol: symbol}
	}
	return &types.Number{Value: value}
}
