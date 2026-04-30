package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
)

// evalPreciseConversion evaluates a precise conversion expression.
// Returns the value with IsPrecise set to skip display rounding.
// Works on any numeric type, mirroring evalNapkinConversion.
func (interp *Interpreter) evalPreciseConversion(n *ast.PreciseConversion) (types.Type, error) {
	value, err := interp.evalNode(n.Expression)
	if err != nil {
		return nil, err
	}

	switch v := value.(type) {
	case *types.Number:
		return v, nil

	case *types.Quantity:
		return &types.Quantity{
			Value:      v.Value,
			Unit:       v.Unit,
			IsExplicit: v.IsExplicit,
			IsPrecise:  true,
		}, nil

	case *types.Currency:
		return v, nil

	case *types.Duration:
		return v, nil

	case *types.Rate:
		return v, nil

	case *types.Fraction:
		// Fractions are already exact — no-op
		return v, nil

	case *types.Percentage:
		// Percentages pass through unchanged.
		return v, nil

	default:
		return nil, fmt.Errorf("precise conversion requires a numeric value, got %T", value)
	}
}
