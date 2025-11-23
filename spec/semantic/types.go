package semantic

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// TypeInfo represents type information for a node.
type TypeInfo struct {
	Type types.Type
	Kind TypeKind
}

// TypeKind represents the kind of type.
type TypeKind int

const (
	TypeNumber TypeKind = iota
	TypeCurrency
	TypeBoolean
	TypeDate
	TypeTime
	TypeDuration
	TypeQuantity
)

// CheckTypeCompatibility validates that an operation is type-compatible.
// Returns an error diagnostic if incompatible, nil if compatible.
func CheckTypeCompatibility(left, right TypeInfo, operator string, r *ast.Range) *Diagnostic {
	// Date + Duration → Date (valid)
	if left.Kind == TypeDate && right.Kind == TypeDuration && operator == "+" {
		return nil
	}

	// Date - Date → Duration (valid)
	if left.Kind == TypeDate && right.Kind == TypeDate && operator == "-" {
		return nil
	}

	// Date - Duration → Date (valid)
	if left.Kind == TypeDate && right.Kind == TypeDuration && operator == "-" {
		return nil
	}

	// Date + Date → ERROR
	if left.Kind == TypeDate && right.Kind == TypeDate && operator == "+" {
		return &Diagnostic{
			Severity: Error,
			Code:     DiagInvalidDateOperation,
			Message:  "Cannot add two dates - did you mean to subtract them to get the duration between them?",
			Range:    r,
		}
	}

	// Date * anything → ERROR
	if left.Kind == TypeDate && (operator == "*" || operator == "/" || operator == "%") {
		return &Diagnostic{
			Severity: Error,
			Code:     DiagInvalidDateOperation,
			Message:  fmt.Sprintf("Cannot %s a date - dates only support addition and subtraction with durations", operatorToVerb(operator)),
			Range:    r,
		}
	}

	// Duration - Date → ERROR
	if left.Kind == TypeDuration && right.Kind == TypeDate && operator == "-" {
		return &Diagnostic{
			Severity: Error,
			Code:     DiagInvalidDateOperation,
			Message:  "Cannot subtract a date from a duration - try reversing the order",
			Range:    r,
		}
	}

	// Duration + Duration → Duration (valid)
	// Duration - Duration → Duration (valid)
	// Duration * Number → Duration (valid)
	// Duration / Number → Duration (valid)
	if left.Kind == TypeDuration {
		if right.Kind == TypeDuration && (operator == "+" || operator == "-") {
			return nil
		}
		if right.Kind == TypeNumber && (operator == "*" || operator == "/") {
			return nil
		}
	}

	// Number * Duration → Duration (valid)
	if left.Kind == TypeNumber && right.Kind == TypeDuration && operator == "*" {
		return nil
	}

	// Currency + Number → Currency (relaxed rule)
	// Currency * Number → Currency (relaxed rule)
	if left.Kind == TypeCurrency && right.Kind == TypeNumber {
		if operator == "+" || operator == "-" || operator == "*" || operator == "/" {
			return nil
		}
	}

	// Number * Currency → Currency (relaxed rule)
	if left.Kind == TypeNumber && right.Kind == TypeCurrency && operator == "*" {
		return nil
	}

	// Currency + Currency (same code) → Currency
	if left.Kind == TypeCurrency && right.Kind == TypeCurrency {
		leftCurrency := left.Type.(*types.Currency)
		rightCurrency := right.Type.(*types.Currency)

		if leftCurrency.Code != rightCurrency.Code {
			return &Diagnostic{
				Severity: Error,
				Code:     DiagIncompatibleCurrencies,
				Message:  fmt.Sprintf("Cannot %s %s and %s directly - convert to the same currency first", operatorToVerb(operator), leftCurrency.Code, rightCurrency.Code),
				Range:    r,
			}
		}
		return nil // Same currency is okay
	}

	// Non-currency units → ERROR (not yet supported)
	if left.Kind == TypeQuantity || right.Kind == TypeQuantity {
		return &Diagnostic{
			Severity: Error,
			Code:     DiagUnsupportedUnit,
			Message:  "Unit operations are not yet supported - only currency and time units are currently available",
			Range:    r,
		}
	}

	// Number operations with same types
	if left.Kind == TypeNumber && right.Kind == TypeNumber {
		return nil
	}

	// Boolean comparisons
	if left.Kind == TypeBoolean && right.Kind == TypeBoolean {
		if operator == "==" || operator == "!=" {
			return nil
		}
	}

	// If we get here, it's a type mismatch
	return &Diagnostic{
		Severity: Error,
		Code:     DiagTypeMismatch,
		Message:  fmt.Sprintf("Cannot %s %s and %s", operatorToVerb(operator), kindToString(left.Kind), kindToString(right.Kind)),
		Range:    r,
	}
}

// operatorToVerb converts an operator symbol to a verb for error messages.
func operatorToVerb(op string) string {
	switch op {
	case "+":
		return "add"
	case "-":
		return "subtract"
	case "*":
		return "multiply"
	case "/":
		return "divide"
	case "%":
		return "modulo"
	case "^":
		return "exponentiate"
	default:
		return "operate on"
	}
}

// kindToString converts a TypeKind to a readable string.
func kindToString(kind TypeKind) string {
	switch kind {
	case TypeNumber:
		return "number"
	case TypeCurrency:
		return "currency"
	case TypeBoolean:
		return "boolean"
	case TypeDate:
		return "date"
	case TypeTime:
		return "time"
	case TypeDuration:
		return "duration"
	case TypeQuantity:
		return "quantity with unit"
	default:
		return "unknown type"
	}
}
