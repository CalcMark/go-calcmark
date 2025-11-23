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
	Value        decimal.Decimal
	SourceFormat string // Original formatting from source (e.g., "1,000", "1k", "10000")
}

// NewNumber creates a Number from various input types
func NewNumber(value any) (*Number, error) {
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
		// Handle percentage literals (e.g., "20%" → 0.20)
		if strings.HasSuffix(v, "%") {
			percentStr := strings.TrimSuffix(v, "%")
			percent, err := decimal.NewFromString(percentStr)
			if err != nil {
				return nil, fmt.Errorf("invalid percentage string: can't convert %s to decimal", v)
			}
			// Convert percentage to decimal (20% → 0.20)
			d = percent.Div(decimal.NewFromInt(100))
		} else {
			d, err = decimal.NewFromString(v)
			if err != nil {
				return nil, fmt.Errorf("invalid number string: %w", err)
			}
		}
	case decimal.Decimal:
		d = v
	default:
		return nil, fmt.Errorf("cannot create Number from type %T", value)
	}

	return &Number{Value: d}, nil
}

// NewNumberWithFormat creates a Number with source format preservation
func NewNumberWithFormat(value any, sourceFormat string) (*Number, error) {
	n, err := NewNumber(value)
	if err != nil {
		return nil, err
	}
	n.SourceFormat = sourceFormat
	return n, nil
}

// String returns the string representation of the number
// If SourceFormat is set, returns that; otherwise formats with default rules
func (n *Number) String() string {
	// If we have the original source format, use it
	if n.SourceFormat != "" {
		return n.SourceFormat
	}

	// Otherwise, format without thousands separators (just the canonical value)
	s := n.Value.String()

	// Split into integer and fractional parts
	parts := strings.Split(s, ".")
	intPart := parts[0]

	// If there's no decimal part, return the integer
	if len(parts) == 1 {
		return intPart
	}

	// Remove trailing zeros from fractional part
	fracPart := strings.TrimRight(parts[1], "0")

	// If no significant fractional part, return just the integer
	if fracPart == "" {
		return intPart
	}

	return intPart + "." + fracPart
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
	Value        decimal.Decimal
	Symbol       string
	SourceFormat string // Original formatting from source (e.g., "$1,000", "$10000")
}

// NewCurrency creates a Currency from various input types
func NewCurrency(value any, symbol string) (*Currency, error) {
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

// NewCurrencyWithFormat creates a Currency with source format preservation
func NewCurrencyWithFormat(value any, symbol string, sourceFormat string) (*Currency, error) {
	c, err := NewCurrency(value, symbol)
	if err != nil {
		return nil, err
	}
	c.SourceFormat = sourceFormat
	return c, nil
}

// String returns the string representation with currency symbol
// If SourceFormat is set, returns that; otherwise formats as $X.XX
func (c *Currency) String() string {
	// If we have the original source format, use it
	if c.SourceFormat != "" {
		return c.SourceFormat
	}

	// Otherwise, format with symbol and 2 decimal places (no thousands separators)
	rounded := c.Value.Round(2)

	// Format with symbol and 2 decimal places
	intPart := rounded.IntPart()
	fracPart := rounded.Sub(decimal.NewFromInt(intPart)).Abs().Mul(decimal.NewFromInt(100)).IntPart()

	intStr := fmt.Sprintf("%d", intPart)
	if intPart < 0 {
		intStr = fmt.Sprintf("%d", -intPart)
		return fmt.Sprintf("-%s%s.%02d", c.Symbol, intStr, fracPart)
	}

	return fmt.Sprintf("%s%s.%02d", c.Symbol, intStr, fracPart)
}

// addThousandsSeparators adds commas to a numeric string
// Always uses commas (US format) regardless of locale
func addThousandsSeparators(s string) string {
	// Handle negative numbers
	negative := strings.HasPrefix(s, "-")
	if negative {
		s = s[1:]
	}

	n := len(s)

	// No separators needed for 3 or fewer digits
	if n <= 3 {
		if negative {
			return "-" + s
		}
		return s
	}

	// Build result left-to-right with commas every 3 digits from the right
	var result strings.Builder

	// Calculate size of first group (1-3 digits)
	firstGroupSize := n % 3
	if firstGroupSize == 0 {
		firstGroupSize = 3
	}

	// Write first group
	result.WriteString(s[:firstGroupSize])

	// Write remaining groups of 3, each preceded by comma
	for i := firstGroupSize; i < n; i += 3 {
		result.WriteByte(',')
		result.WriteString(s[i : i+3])
	}

	if negative {
		return "-" + result.String()
	}
	return result.String()
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
func NewBoolean(value any) (*Boolean, error) {
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
