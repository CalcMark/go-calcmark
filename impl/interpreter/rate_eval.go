package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// evalRateLiteral evaluates a rate literal and returns a Rate type.
// Examples: "100 MB/s", "5 GB per day", "$0.10 per hour"
func (interp *Interpreter) evalRateLiteral(node *ast.RateLiteral) (types.Type, error) {
	// Evaluate the amount (numerator)
	amountVal, err := interp.evalNode(node.Amount)
	if err != nil {
		return nil, fmt.Errorf("rate amount evaluation failed: %w", err)
	}

	// Convert amount to Quantity
	var amountQty *types.Quantity
	switch v := amountVal.(type) {
	case *types.Quantity:
		amountQty = v
	case *types.Number:
		// Plain number without unit (e.g., "60" in "60 per second")
		amountQty = &types.Quantity{
			Value: v.Value,
			Unit:  "", // Unitless
		}
	case *types.Currency:
		// Currency as rate amount (e.g., "$50 per hour", "100 USD/day")
		// Use the symbol for display (preserves $ vs USD)
		amountQty = &types.Quantity{
			Value: v.Value,
			Unit:  v.Symbol,
		}
	default:
		return nil, fmt.Errorf("rate amount must be a number, quantity, or currency, got %T", amountVal)
	}

	// Create the Rate
	rate := types.NewRate(amountQty, node.PerUnit)

	return rate, nil
}
