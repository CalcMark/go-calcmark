package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// evalNapkinConversion evaluates a napkin conversion expression.
// Returns a rounded numeric value that can be used in calculations.
// The value is rounded to 2 significant figures (adaptable).
func (interp *Interpreter) evalNapkinConversion(n *ast.NapkinConversion) (types.Type, error) {
	// Evaluate the expression
	value, err := interp.evalNode(n.Expression)
	if err != nil {
		return nil, err
	}

	// Extract numeric value from the result
	var numValue decimal.Decimal

	switch v := value.(type) {
	case *types.Number:
		numValue = v.Value
	case *types.Quantity:
		// For quantities, just use the numeric value
		numValue = v.Value
	case *types.Currency:
		numValue = v.Value
	case *types.Duration:
		// Convert to seconds for napkin formatting
		numValue = v.ToSeconds()
	case *types.Rate:
		// Use the amount of the rate
		numValue = v.Amount.Value
	default:
		return nil, fmt.Errorf("napkin conversion requires a numeric value, got %T", value)
	}

	// Round the value to napkin precision (2 sig figs by default)
	floatVal, _ := numValue.Abs().Float64()

	// Round based on magnitude
	roundedFloat := floatVal
	if floatVal >= 1000 {
		// Determine scale
		var magnitude float64
		if floatVal >= 1e12 {
			magnitude = 1e12
		} else if floatVal >= 1e9 {
			magnitude = 1e9
		} else if floatVal >= 1e6 {
			magnitude = 1e6
		} else {
			magnitude = 1000
		}

		// Scale, round to 2 sig figs, scale back
		scaled := floatVal / magnitude
		rounded := roundToSignificantFigures(scaled, 2)

		// Convert rounded value back to float
		var roundedScaled float64
		switch r := rounded.(type) {
		case string:
			if d, err := decimal.NewFromString(r); err == nil {
				roundedScaled, _ = d.Float64()
			} else {
				roundedScaled = scaled
			}
		case int:
			roundedScaled = float64(r)
		default:
			roundedScaled = scaled
		}

		roundedFloat = roundedScaled * magnitude
	} else if floatVal > 0 {
		// For smaller numbers < 1000, round to 2 sig figs
		rounded := roundToSignificantFigures(floatVal, 2)
		switch r := rounded.(type) {
		case string:
			if d, err := decimal.NewFromString(r); err == nil {
				roundedFloat, _ = d.Float64()
			}
		case int:
			roundedFloat = float64(r)
		}
	}

	// Preserve sign
	if numValue.IsNegative() {
		roundedFloat = -roundedFloat
	}

	// Return as Number with rounded value (can be used in calculations)
	return types.NewNumber(decimal.NewFromFloat(roundedFloat)), nil
}
