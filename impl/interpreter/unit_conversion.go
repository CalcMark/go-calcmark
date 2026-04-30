package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/CalcMark/go-calcmark/v2/spec/units"
	"github.com/shopspring/decimal"
)

// evalQuantityOperation handles quantity + quantity with unit conversion
// USER REQUIREMENT: First-unit-wins rule
func evalQuantityOperation(left, right *types.Quantity, operator string) (types.Type, error) {
	if operator != "+" && operator != "-" {
		switch operator {
		case "*":
			return nil, fmt.Errorf("cannot multiply quantity by quantity — the result would be \"square %s\" which isn't a real unit. Use number() to get the raw values: number(%s) * number(%s)",
				left.Unit, left.String(), right.String())
		case "/":
			return nil, fmt.Errorf("cannot divide quantity by quantity — dividing %s by %s doesn't produce %s. Use number() to get a ratio: number(%s) / number(%s)",
				left.Unit, right.Unit, left.Unit, left.String(), right.String())
		default:
			return nil, fmt.Errorf("unsupported quantity operation: %s", operator)
		}
	}

	// First-unit-wins: convert right to left's unit
	rightConverted, err := convertQuantity(right, left.Unit)
	if err != nil {
		return nil, fmt.Errorf("cannot %s incompatible units %s and %s: %w",
			operator, left.Unit, right.Unit, err)
	}

	var result decimal.Decimal
	switch operator {
	case "+":
		result = left.Value.Add(rightConverted.Value)
	case "-":
		result = left.Value.Sub(rightConverted.Value)
	}

	// Result is in left's unit (first-unit-wins)
	return &types.Quantity{Value: result, Unit: left.Unit}, nil
}

// convertQuantity converts a quantity to the target unit.
// Delegates to the spec/units conversion registry.
func convertQuantity(qty *types.Quantity, targetUnit string) (*types.Quantity, error) {
	if qty.Unit == targetUnit {
		return qty, nil // No conversion needed
	}

	converted, err := units.Convert(qty.Value, qty.Unit, targetUnit)
	if err != nil {
		return nil, fmt.Errorf("cannot convert %s to %s (incompatible units)", qty.Unit, targetUnit)
	}

	return &types.Quantity{
		Value: converted,
		Unit:  targetUnit,
	}, nil
}
