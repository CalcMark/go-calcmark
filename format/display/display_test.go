package display

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		// Small numbers - no suffix
		{"zero", "0", "0"},
		{"small integer", "42", "42"},
		{"small decimal", "3.14", "3.14"},
		{"negative small", "-99", "-99"},
		{"decimal precision", "0.5", "0.5"},
		{"sub-thousandth", "999", "999"},

		// Thousands (K)
		{"exactly 1K", "1000", "1K"},
		{"1.5K", "1500", "1.5K"},
		{"10K", "10000", "10K"},
		{"100K", "100000", "100K"},
		{"999K", "999000", "999K"},

		// Millions (M)
		{"exactly 1M", "1000000", "1M"},
		{"1.5M", "1500000", "1.5M"},
		{"10M", "10000000", "10M"},
		{"100M", "100000000", "100M"},

		// Billions (B)
		{"exactly 1B", "1000000000", "1B"},
		{"1.5B", "1500000000", "1.5B"},
		{"7.8B", "7800000000", "7.8B"},

		// Trillions (T)
		{"exactly 1T", "1000000000000", "1T"},
		{"1.2T", "1200000000000", "1.2T"},

		// Negative large numbers
		{"negative 100K", "-100000", "-100K"},
		{"negative 1.5M", "-1500000", "-1.5M"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, _ := decimal.NewFromString(tt.value)
			result := FormatNumber(value)
			if result != tt.expected {
				t.Errorf("FormatNumber(%s) = %q, want %q", tt.value, result, tt.expected)
			}
		})
	}
}

func TestFormatQuantity(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		unit     string
		expected string
	}{
		{"100K users", "100000", "users", "100K users"},             // arbitrary unit: uses K/M/B/T
		{"1.5M bytes normalized", "1500000", "bytes", "1.43 MB"},    // known unit: uses unit normalization
		{"small quantity", "42", "items", "42 items"},               // arbitrary unit: stays as-is
		{"decimal quantity normalized", "3.14", "meters", "3.14 m"}, // known unit: uses canonical symbol
		{"large GB normalized", "23400000", "GB", "22.3 PB"},        // the original problem case!
		{"1000 meters to km", "1000", "m", "1 km"},                  // meters → kilometers
		{"5280 feet to miles", "5280", "feet", "1 mi"},              // feet → miles
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, _ := decimal.NewFromString(tt.value)
			q := &types.Quantity{Value: value, Unit: tt.unit}
			result := FormatQuantity(q)
			if result != tt.expected {
				t.Errorf("FormatQuantity(%s %s) = %q, want %q", tt.value, tt.unit, result, tt.expected)
			}
		})
	}
}

func TestFormatRate(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		unit     string
		perUnit  string
		expected string
	}{
		{"100K users/day", "100000", "users", "day", "100K users/day"},         // arbitrary unit
		{"1.5M bytes/s normalized", "1500000", "bytes", "second", "1.43 MB/s"}, // known unit: normalized
		{"small rate", "100", "requests", "minute", "100 requests/min"},        // arbitrary unit
		{"1000 meters/hour", "1000", "m", "hour", "1 km/h"},                    // meters → km
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, _ := decimal.NewFromString(tt.value)
			r := types.NewRate(&types.Quantity{Value: value, Unit: tt.unit}, tt.perUnit)
			result := FormatRate(r)
			if result != tt.expected {
				t.Errorf("FormatRate() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatCurrency(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		symbol   string
		expected string
	}{
		{"small amount", "42.50", "$", "$42.50"},
		{"large amount", "1500000", "$", "$1.5M"},
		{"millions", "10000000", "€", "€10M"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, _ := decimal.NewFromString(tt.value)
			c := types.NewCurrency(value, tt.symbol)
			result := FormatCurrency(c)
			if result != tt.expected {
				t.Errorf("FormatCurrency(%s%s) = %q, want %q", tt.symbol, tt.value, result, tt.expected)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name     string
		value    types.Type
		expected string
	}{
		{
			name:     "number",
			value:    types.NewNumber(decimal.NewFromInt(100000)),
			expected: "100K",
		},
		{
			name:     "quantity",
			value:    types.NewQuantity(decimal.NewFromInt(1500000), "users"),
			expected: "1.5M users",
		},
		{
			name:     "boolean",
			value:    types.NewBoolean(true),
			expected: "true",
		},
		{
			name:     "nil",
			value:    nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Format(tt.value)
			if result != tt.expected {
				t.Errorf("Format() = %q, want %q", result, tt.expected)
			}
		})
	}
}
