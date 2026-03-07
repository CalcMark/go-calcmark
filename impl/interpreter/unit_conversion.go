package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/CalcMark/go-calcmark/spec/units"
	"github.com/shopspring/decimal"
)

// evalQuantityOperation handles quantity + quantity with unit conversion
// USER REQUIREMENT: First-unit-wins rule
func evalQuantityOperation(left, right *types.Quantity, operator string) (types.Type, error) {
	if operator != "+" && operator != "-" {
		return nil, fmt.Errorf("unsupported quantity operation: %s", operator)
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
