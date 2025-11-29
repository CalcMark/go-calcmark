package types

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// Currency represents a monetary value with a currency symbol and ISO code.
// It preserves both the display symbol (e.g., "$") and the normalized ISO code (e.g., "USD").
type Currency struct {
	Value  decimal.Decimal
	Symbol string // Display symbol: "$", "€", "£", "¥", or ISO code like "USD"
	Code   string // Normalized ISO 4217 code: "USD", "EUR", "GBP", "JPY"
}

// SymbolToCode maps currency symbols to their ISO 4217 codes.
// This allows normalization of common symbols to standard codes.
var SymbolToCode = map[string]string{
	"$": "USD",
	"€": "EUR",
	"£": "GBP",
	"¥": "JPY",
}

// NewCurrency creates a new Currency with the given value and symbol/code.
// If symbolOrCode is a known symbol ($, €, £, ¥), it's mapped to the ISO code.
// Otherwise, it's used as both the symbol and code.
func NewCurrency(value decimal.Decimal, symbolOrCode string) *Currency {
	symbol := symbolOrCode
	code := symbolOrCode

	// If this is a known symbol, map it to its ISO code
	if mappedCode, ok := SymbolToCode[symbolOrCode]; ok {
		code = mappedCode
	}

	return &Currency{
		Value:  value,
		Symbol: symbol,
		Code:   code,
	}
}

// NewCurrencyFromString creates a Currency from a string value and symbol/code.
// Returns an error if the string cannot be parsed as a valid number.
func NewCurrencyFromString(s string, symbolOrCode string) (*Currency, error) {
	value, err := decimal.NewFromString(s)
	if err != nil {
		return nil, err
	}
	return NewCurrency(value, symbolOrCode), nil
}

// String returns the string representation with symbol and value.
// For display purposes, formats the value with standard decimal formatting.
func (c *Currency) String() string {
	// Format with appropriate decimals (most currencies use 2)
	formatted := c.Value.StringFixed(2)

	// Remove trailing zeros after decimal point, but keep at least 2 decimals for cents
	// This ensures $1.00 but also allows $1.99
	return fmt.Sprintf("%s%s", c.Symbol, formatted)
}

// IsSameCurrency checks if two currencies have the same ISO code.
// This is used for validating currency arithmetic operations.
func (c *Currency) IsSameCurrency(other *Currency) bool {
	return c.Code == other.Code
}

// CodeToSymbol maps ISO 4217 codes to display symbols.
// This is the reverse of SymbolToCode.
var CodeToSymbol = map[string]string{
	"USD": "$",
	"EUR": "€",
	"GBP": "£",
	"JPY": "¥",
}

// IsCurrencyCode checks if a string is a known currency code or symbol.
func IsCurrencyCode(s string) bool {
	// Check if it's a known symbol
	if _, ok := SymbolToCode[s]; ok {
		return true
	}
	// Check if it's a known ISO code
	if _, ok := CodeToSymbol[s]; ok {
		return true
	}
	return false
}

// NormalizeCurrencyCode converts a symbol or code to its ISO code.
// Returns the original string if not recognized.
func NormalizeCurrencyCode(s string) string {
	if code, ok := SymbolToCode[s]; ok {
		return code
	}
	// Already a code or unknown - return uppercase
	return s
}

// GetCurrencySymbol returns the display symbol for a currency code.
// Returns the code itself if no symbol is defined.
func GetCurrencySymbol(code string) string {
	if sym, ok := CodeToSymbol[code]; ok {
		return sym
	}
	return code
}
