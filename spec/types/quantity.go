package types

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// Quantity represents a value with a physical unit.
// This is a stub for future implementation with full unit support.
// For now, it simply stores the value and unit string.
type Quantity struct {
	Value decimal.Decimal
	Unit  string
}

// NewQuantity creates a new Quantity with the given value and unit.
func NewQuantity(value decimal.Decimal, unit string) *Quantity {
	return &Quantity{
		Value: value,
		Unit:  unit,
	}
}

// NewQuantityFromString creates a Quantity from a string value and unit.
func NewQuantityFromString(s string, unit string) (*Quantity, error) {
	value, err := decimal.NewFromString(s)
	if err != nil {
		return nil, err
	}
	return &Quantity{
		Value: value,
		Unit:  unit,
	}, nil
}

// String returns the string representation.
func (q *Quantity) String() string {
	return fmt.Sprintf("%s %s", q.Value.String(), q.Unit)
}
