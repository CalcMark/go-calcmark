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

// ToDecimal extracts a decimal.Decimal value from various numeric types.
// Supports Number, Currency (returns the value), and Quantity (returns the value).
// Returns an error for non-numeric types.
func ToDecimal(t Type) (decimal.Decimal, error) {
	switch v := t.(type) {
	case *Number:
		return v.Value, nil
	case *Currency:
		return v.Value, nil
	case *Quantity:
		return v.Value, nil
	default:
		return decimal.Zero, &TypeError{Message: "expected numeric type, got " + typeName(t)}
	}
}

// typeName returns the name of a Type for error messages.
func typeName(t Type) string {
	if t == nil {
		return "nil"
	}
	switch t.(type) {
	case *Number:
		return "Number"
	case *Currency:
		return "Currency"
	case *Quantity:
		return "Quantity"
	case *Boolean:
		return "Boolean"
	case *Date:
		return "Date"
	case *Duration:
		return "Duration"
	case *Rate:
		return "Rate"
	default:
		return "unknown"
	}
}

// TypeError represents a type error during evaluation.
type TypeError struct {
	Message string
}

func (e *TypeError) Error() string {
	return e.Message
}
