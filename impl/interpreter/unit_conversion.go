package interpreter

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/types"
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

// convertQuantity converts a quantity to the target unit using unit_library registry
func convertQuantity(qty *types.Quantity, targetUnit string) (*types.Quantity, error) {
	if qty.Unit == targetUnit {
		return qty, nil // No conversion needed
	}

	// Normalize unit names for lookup
	sourceNorm := strings.ToLower(qty.Unit)
	targetNorm := strings.ToLower(targetUnit)

	// Look up both units in the registry
	sourceInfo, sourceOk := GetUnitInfo(sourceNorm)
	targetInfo, targetOk := GetUnitInfo(targetNorm)

	if !sourceOk || !targetOk {
		// One or both are arbitrary units - cannot convert
		return nil, fmt.Errorf("cannot convert %s to %s (incompatible units)", qty.Unit, targetUnit)
	}

	// Check if units are in the same category
	if sourceInfo.Category != targetInfo.Category {
		return nil, fmt.Errorf("cannot convert %s to %s (different unit types: %s vs %s)",
			qty.Unit, targetUnit, sourceInfo.Category, targetInfo.Category)
	}

	// Perform conversion: source -> base unit -> target
	value, _ := qty.Value.Float64()
	baseValue := sourceInfo.ToBaseUnit(value)         // Convert to base unit (e.g., meters)
	targetValue := targetInfo.FromBaseUnit(baseValue) // Convert from base to target

	return &types.Quantity{
		Value: decimal.NewFromFloat(targetValue),
		Unit:  targetUnit, // Preserve user's target unit name
	}, nil
}
