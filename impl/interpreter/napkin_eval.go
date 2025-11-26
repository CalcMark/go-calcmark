package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// evalNapkinConversion evaluates a napkin conversion expression.
// Converts numbers to human-readable format with K/M/B/T suffixes.
// Returns a string type with the formatted number.
func (interp *Interpreter) evalNapkinConversion(n *ast.NapkinConversion) (types.Type, error) {
	// Evaluate the expression
	value, err := interp.evalNode(n.Expression)
	if err != nil {
		return nil, err
	}

	// Extract numeric value from the result
	var numValue decimal.Decimal

	switch v := value.(type) {
	case *types.NumberLiteral:
		numValue = v.Value
	case *types.Quantity:
		// For quantities, just format the numeric value (ignore units)
		numValue = v.Value
	case *types.CurrencyValue:
		numValue = v.Amount
	case *types.Duration:
		// Convert to seconds for napkin formatting
		numValue = v.ToSeconds()
	case *types.Rate:
		// Format the amount of the rate
		numValue = v.Amount.Value
	default:
		return nil, fmt.Errorf("napkin conversion requires a numeric value, got %T", value)
	}

	// Format with napkin style (2 sig figs by default, adaptable)
	formatted := formatNapkin(numValue, 2)

	// Return as a string - napkin formatting produces string output
	return &types.StringValue{Value: formatted}, nil
}
