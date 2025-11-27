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
