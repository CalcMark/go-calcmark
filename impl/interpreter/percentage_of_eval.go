package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// evalPercentageOf evaluates a "X% of Y" expression.
// The percentage value is converted from percentage form (e.g., 10% becomes 0.10)
// and multiplied by the target value.
func (interp *Interpreter) evalPercentageOf(n *ast.PercentageOf) (types.Type, error) {
	// Evaluate the percentage (should be a NUMBER_PERCENT like "10%")
	percentResult, err := interp.evalNode(n.Percentage)
	if err != nil {
		return nil, err
	}

	// Evaluate the value to take percentage of
	valueResult, err := interp.evalNode(n.Value)
	if err != nil {
		return nil, err
	}

	// Extract the percentage as a decimal (already converted from % form)
	var percentDecimal decimal.Decimal
	switch p := percentResult.(type) {
	case *types.Percentage:
		percentDecimal = p.Value
	case *types.Number:
		percentDecimal = p.Value
	default:
		return nil, fmt.Errorf("percentage must be a number, got %T", percentResult)
	}

	// Apply percentage to the value based on its type
	switch v := valueResult.(type) {
	case *types.Number:
		result := v.Value.Mul(percentDecimal)
		return types.NewNumber(result), nil

	case *types.Quantity:
		result := v.Value.Mul(percentDecimal)
		return types.NewQuantity(result, v.Unit), nil

	case *types.Currency:
		result := v.Value.Mul(percentDecimal)
		return types.NewCurrency(result, v.Symbol), nil

	default:
		return nil, fmt.Errorf("cannot take percentage of %T", valueResult)
	}
}

// evalAsPercentOf evaluates "X as % of Y" — computes the ratio X/Y as a Percentage.
// Both operands must be the same type family. This is the inverse of evalPercentageOf:
//
//	"20% of $500" = $100   (forward: apply percentage)
//	"$100 as % of $500" = 20%  (reverse: compute percentage)
func (interp *Interpreter) evalAsPercentOf(n *ast.AsPercentOf) (types.Type, error) {
	numerator, err := interp.evalNode(n.Numerator)
	if err != nil {
		return nil, err
	}
	denominator, err := interp.evalNode(n.Denominator)
	if err != nil {
		return nil, err
	}

	numVal, numDesc, err := extractDecimalForRatio(numerator)
	if err != nil {
		return nil, fmt.Errorf("cannot use %s in 'as %% of': %w", formatTypeForError(numerator), err)
	}
	denVal, denDesc, err := extractDecimalForRatio(denominator)
	if err != nil {
		return nil, fmt.Errorf("cannot use %s in 'as %% of': %w", formatTypeForError(denominator), err)
	}

	// Type compatibility check
	if numDesc != denDesc {
		return nil, fmt.Errorf("cannot compute %s as %% of %s — both values must be the same type. "+
			"Got %s and %s",
			formatTypeForError(numerator), formatTypeForError(denominator),
			numDesc, denDesc)
	}

	if denVal.IsZero() {
		return nil, fmt.Errorf("division by zero in 'as %% of'")
	}

	ratio := numVal.Div(denVal)
	return types.NewPercentage(ratio), nil
}

// extractDecimalForRatio extracts a decimal value and a type descriptor string
// from a typed value. The descriptor is used for same-type checking.
// Duration values are normalized to seconds for cross-unit comparison.
func extractDecimalForRatio(t types.Type) (decimal.Decimal, string, error) {
	switch v := t.(type) {
	case *types.Number:
		return v.Value, "number", nil
	case *types.Percentage:
		return v.Value, "percentage", nil
	case *types.Currency:
		return v.Value, "currency:" + v.Symbol, nil
	case *types.Quantity:
		return v.Value, "quantity:" + v.Unit, nil
	case *types.Duration:
		return v.ToSeconds(), "duration", nil
	case *types.Fraction:
		dec := decimal.NewFromBigRat(v.Value, 15)
		if v.Unit != "" {
			return dec, "quantity:" + v.Unit, nil
		}
		return dec, "number", nil
	default:
		return decimal.Zero, "", fmt.Errorf("type %s cannot be expressed as a percentage", formatTypeForError(t))
	}
}
