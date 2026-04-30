package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
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
	case *types.Duration:
		// Duration as rate amount (e.g., "40 hours / week", "30 minutes
		// per session"). Stored as Quantity{value, time-unit} so the
		// Rate × Duration / Rate × Rate arithmetic in operators.go can
		// detect time-unit cancellation against the PerUnit. The
		// Duration value is already canonicalised by the parser, so
		// `40 hours` and `40 hour` both land here as Unit="hour".
		amountQty = &types.Quantity{
			Value: v.Value,
			Unit:  v.Unit,
		}
	default:
		return nil, fmt.Errorf("rate amount must be a number, quantity, currency, or duration, got %T", amountVal)
	}

	// Create the Rate
	rate := types.NewRate(amountQty, node.PerUnit)

	return rate, nil
}
