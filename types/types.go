// Package types defines the CalcMark type system
package types

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// Type is the interface that all CalcMark types implement
type Type interface {
	String() string
	Equal(other Type) bool
	TypeName() string
}

// Number represents a numeric value with arbitrary precision
type Number struct {
	Value decimal.Decimal
}

// NewNumber creates a Number from various input types
func NewNumber(value interface{}) (*Number, error) {
	var d decimal.Decimal
	var err error

	switch v := value.(type) {
	case int:
		d = decimal.NewFromInt(int64(v))
	case int64:
		d = decimal.NewFromInt(v)
	case float64:
		d = decimal.NewFromFloat(v)
	case string:
		d, err = decimal.NewFromString(v)
		if err != nil {
			return nil, fmt.Errorf("invalid number string: %w", err)
		}
	case decimal.Decimal:
		d = v
	default:
		return nil, fmt.Errorf("cannot create Number from type %T", value)
	}

	return &Number{Value: d}, nil
}

// String returns the string representation of the number
// Removes trailing zeros and unnecessary decimal points
func (n *Number) String() string {
	s := n.Value.String()

	// If there's no decimal point, return as-is
	if !strings.Contains(s, ".") {
		return s
	}

	// Remove trailing zeros and decimal point if needed
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")

	return s
}

// TypeName returns "Number"
func (n *Number) TypeName() string {
	return "Number"
}

// Equal checks equality with another Type
func (n *Number) Equal(other Type) bool {
	otherNum, ok := other.(*Number)
	if !ok {
		return false
	}
	return n.Value.Equal(otherNum.Value)
}

// ToDecimal converts to decimal.Decimal
func (n *Number) ToDecimal() decimal.Decimal {
	return n.Value
}

// Currency represents a currency value with a symbol
type Currency struct {
	Value  decimal.Decimal
	Symbol string
}

// NewCurrency creates a Currency from various input types
func NewCurrency(value interface{}, symbol string) (*Currency, error) {
	if symbol == "" {
		symbol = "$"
	}

	var d decimal.Decimal
	var err error

	switch v := value.(type) {
	case int:
		d = decimal.NewFromInt(int64(v))
	case int64:
		d = decimal.NewFromInt(v)
	case float64:
		d = decimal.NewFromFloat(v)
	case string:
		d, err = decimal.NewFromString(v)
		if err != nil {
			return nil, fmt.Errorf("invalid currency string: %w", err)
		}
	case decimal.Decimal:
		d = v
	default:
		return nil, fmt.Errorf("cannot create Currency from type %T", value)
	}

	return &Currency{Value: d, Symbol: symbol}, nil
}

// String returns the string representation with currency symbol
// Format: $1,000.00
func (c *Currency) String() string {
	// Round to 2 decimal places
	rounded := c.Value.Round(2)

	// Format with thousands separator and 2 decimal places
	// The shopspring/decimal library doesn't have built-in formatting,
	// so we need to handle this manually
	intPart := rounded.IntPart()
	fracPart := rounded.Sub(decimal.NewFromInt(intPart)).Abs().Mul(decimal.NewFromInt(100)).IntPart()

	// Add thousands separators
	intStr := fmt.Sprintf("%d", intPart)
	if intPart < 0 {
		intStr = fmt.Sprintf("%d", -intPart)
		return fmt.Sprintf("-%s%s.%02d", c.Symbol, addThousandsSeparators(intStr), fracPart)
	}

	return fmt.Sprintf("%s%s.%02d", c.Symbol, addThousandsSeparators(intStr), fracPart)
}

// addThousandsSeparators adds commas to a numeric string
func addThousandsSeparators(s string) string {
	// Handle negative numbers
	negative := false
	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	}

	// Add commas from right to left
	var result strings.Builder
	for i := len(s) - 1; i >= 0; i-- {
		if (len(s)-i)%3 == 1 && i != len(s)-1 {
			result.WriteByte(',')
		}
		result.WriteByte(s[i])
	}

	// Reverse the result
	reversed := result.String()
	runes := []rune(reversed)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	if negative {
		return "-" + string(runes)
	}
	return string(runes)
}

// TypeName returns "Currency"
func (c *Currency) TypeName() string {
	return "Currency"
}

// Equal checks equality with another Type
func (c *Currency) Equal(other Type) bool {
	otherCurr, ok := other.(*Currency)
	if !ok {
		return false
	}
	return c.Value.Equal(otherCurr.Value) && c.Symbol == otherCurr.Symbol
}

// ToDecimal converts to decimal.Decimal
func (c *Currency) ToDecimal() decimal.Decimal {
	return c.Value
}

// Boolean represents a boolean value
type Boolean struct {
	Value bool
}

// NewBoolean creates a Boolean from various input types
func NewBoolean(value interface{}) (*Boolean, error) {
	switch v := value.(type) {
	case bool:
		return &Boolean{Value: v}, nil
	case string:
		// Normalize and check against known values
		switch v {
		case "true", "yes", "t", "y", "1", "True", "Yes", "T", "Y":
			return &Boolean{Value: true}, nil
		case "false", "no", "f", "n", "0", "False", "No", "F", "N":
			return &Boolean{Value: false}, nil
		default:
			return nil, fmt.Errorf("cannot convert '%s' to boolean", v)
		}
	case int:
		return &Boolean{Value: v != 0}, nil
	case int64:
		return &Boolean{Value: v != 0}, nil
	default:
		return nil, fmt.Errorf("cannot create Boolean from type %T", value)
	}
}

// String returns "true" or "false"
func (b *Boolean) String() string {
	if b.Value {
		return "true"
	}
	return "false"
}

// TypeName returns "Boolean"
func (b *Boolean) TypeName() string {
	return "Boolean"
}

// Equal checks equality with another Type
func (b *Boolean) Equal(other Type) bool {
	otherBool, ok := other.(*Boolean)
	if !ok {
		return false
	}
	return b.Value == otherBool.Value
}

// ToBool converts to Go bool
func (b *Boolean) ToBool() bool {
	return b.Value
}
