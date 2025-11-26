package interpreter

import (
	"fmt"
	"math"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// requiresCapacity calculates how many units of capacity are needed for a given load.
// Performs ceiling division with optional buffer percentage.
//
// Examples:
//   - requiresCapacity(10000, 450, 0) → 23  (⌈10000÷450⌉)
//   - requiresCapacity(10000, 450, 20) → 28 (⌈(10000×1.2)÷450⌉)
//   - requiresCapacity(10 TB, 2 TB, 10) → 6
func requiresCapacity(load, capacity types.Type, bufferPercent decimal.Decimal) (*types.Number, error) {
	// Extract numeric values from load and capacity
	var loadValue, capacityValue decimal.Decimal

	switch l := load.(type) {
	case *types.Number:
		loadValue = l.Value
	case *types.Quantity:
		loadValue = l.Value
	case *types.Rate:
		loadValue = l.Amount.Value
	default:
		return nil, fmt.Errorf("requires() load must be a number, quantity, or rate, got %T", load)
	}

	switch c := capacity.(type) {
	case *types.Number:
		capacityValue = c.Value
	case *types.Quantity:
		capacityValue = c.Value
	case *types.Rate:
		capacityValue = c.Amount.Value
	default:
		return nil, fmt.Errorf("requires() capacity must be a number, quantity, or rate, got %T", capacity)
	}

	// Validate capacity is not zero
	if capacityValue.IsZero() {
		return nil, fmt.Errorf("requires() capacity cannot be zero")
	}

	// Validate capacity is positive
	if capacityValue.IsNegative() {
		return nil, fmt.Errorf("requires() capacity must be positive")
	}

	// Validate buffer percentage (only negative is invalid)
	if bufferPercent.IsNegative() {
		return nil, fmt.Errorf("requires() buffer percentage cannot be negative")
	}

	// Buffer is a decimal fraction (0.20 = 20%, 1.0 = 100%, 1.2 = 120%)
	// Use percentage literal syntax (20%) or decimal (0.20), NOT plain integers
	bufferMultiplier := decimal.NewFromInt(1).Add(bufferPercent)
	adjustedLoad := loadValue.Mul(bufferMultiplier)

	// Divide: raw_result = adjusted_load ÷ capacity
	rawResult := adjustedLoad.Div(capacityValue)

	// Apply ceiling
	f, _ := rawResult.Float64()
	result := decimal.NewFromFloat(math.Ceil(f))

	return types.NewNumber(result), nil
}

// requiresCapacityNoBuffer is a convenience wrapper for requires() without buffer.
func requiresCapacityNoBuffer(load, capacity types.Type) (*types.Number, error) {
	// Extract numeric values
	var loadValue, capacityValue decimal.Decimal

	switch l := load.(type) {
	case *types.Number:
		loadValue = l.Value
	case *types.Quantity:
		loadValue = l.Value
	case *types.Rate:
		loadValue = l.Amount.Value
	default:
		return nil, fmt.Errorf("requires() load must be a number, quantity, or rate, got %T", load)
	}

	switch c := capacity.(type) {
	case *types.Number:
		capacityValue = c.Value
	case *types.Quantity:
		capacityValue = c.Value
	case *types.Rate:
		capacityValue = c.Amount.Value
	default:
		return nil, fmt.Errorf("requires() capacity must be a number, quantity, or rate, got %T", capacity)
	}

	// Validate capacity
	if capacityValue.IsZero() {
		return nil, fmt.Errorf("requires() capacity cannot be zero")
	}
	if capacityValue.IsNegative() {
		return nil, fmt.Errorf("requires() capacity must be positive")
	}

	// Divide and ceiling
	rawResult := loadValue.Div(capacityValue)
	f, _ := rawResult.Float64()
	result := decimal.NewFromFloat(math.Ceil(f))

	return types.NewNumber(result), nil
}
