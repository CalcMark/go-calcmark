package types

import "github.com/shopspring/decimal"

// Number represents an arbitrary-precision decimal number.
// It uses the shopspring/decimal package for accurate decimal arithmetic.
type Number struct {
	Value decimal.Decimal
}

// NewNumber creates a new Number from a decimal.Decimal value.
func NewNumber(value decimal.Decimal) *Number {
	return &Number{Value: value}
}

// NewNumberFromString creates a Number from a string representation.
// Returns an error if the string cannot be parsed as a valid number.
func NewNumberFromString(s string) (*Number, error) {
	value, err := decimal.NewFromString(s)
	if err != nil {
		return nil, err
	}
	return &Number{Value: value}, nil
}

// String returns the string representation of the number.
func (n *Number) String() string {
	return n.Value.String()
}
