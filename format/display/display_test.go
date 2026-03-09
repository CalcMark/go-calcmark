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
		{"fractional lb rounded", "1.1941138678655463", "lb", "1.19 lb"}, // unit conversion result: round to 2dp
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

func TestNapkinFormat(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		unit     string
		isNapkin bool
		expected string
	}{
		// Napkin estimates should show tilde prefix
		{
			name:     "napkin estimate shows tilde",
			value:    "400",
			unit:     "GB",
			isNapkin: true,
			expected: "~400 GB",
		},
		{
			name:     "exact quantity no tilde",
			value:    "400",
			unit:     "GB",
			isNapkin: false,
			expected: "400 GB",
		},
		{
			name:     "napkin with large normalized value",
			value:    "22.3",
			unit:     "PB",
			isNapkin: true,
			expected: "~22.3 PB",
		},
		{
			name:     "napkin with arbitrary unit",
			value:    "100000",
			unit:     "users",
			isNapkin: true,
			expected: "~100K users",
		},
		{
			name:     "napkin with small value",
			value:    "42",
			unit:     "items",
			isNapkin: true,
			expected: "~42 items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, _ := decimal.NewFromString(tt.value)
			q := &types.Quantity{Value: value, Unit: tt.unit, IsNapkin: tt.isNapkin}
			result := FormatQuantity(q)
			if result != tt.expected {
				t.Errorf("FormatQuantity(%s %s, isNapkin=%v) = %q, want %q",
					tt.value, tt.unit, tt.isNapkin, result, tt.expected)
			}
		})
	}
}

func TestThousandSeparators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// addThousandSeparators function tests
		{"three digits", "999", "999"},
		{"four digits", "1000", "1,000"},
		{"five digits", "12345", "12,345"},
		{"six digits", "123456", "123,456"},
		{"seven digits", "1234567", "1,234,567"},
		{"exact 1500", "1500", "1,500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addThousandSeparators(tt.input)
			if result != tt.expected {
				t.Errorf("addThousandSeparators(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestUnifiedCurrencyFormat(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		symbol   string
		expected string
	}{
		// Small values - 2 decimal places
		{"small dollar", "42.50", "$", "$42.50"},
		{"small cents", "0.99", "$", "$0.99"},
		{"small whole number", "100", "$", "$100.00"},

		// Mid-range - thousand separators (1000-9999)
		{"mid dollar", "1500", "$", "$1,500.00"},
		{"mid dollar decimal", "1500.50", "$", "$1,500.50"},
		{"exactly 1000", "1000", "$", "$1,000.00"},
		{"upper mid-range", "9999", "$", "$9,999.00"},

		// Large values - K/M/B suffixes (10000+)
		{"large dollar K", "15000", "$", "$15K"},
		{"large dollar M", "1500000", "$", "$1.5M"},
		{"large dollar B", "1500000000", "$", "$1.5B"},

		// Code to symbol conversion (all known codes map to symbols)
		{"USD to dollar", "100", "USD", "$100.00"},
		{"EUR to euro", "100", "EUR", "€100.00"},
		{"GBP to pound", "100", "GBP", "£100.00"},
		{"JPY to yen", "100", "JPY", "¥100"},

		// Edge cases
		{"zero", "0", "$", "$0.00"},
		{"negative small", "-50.00", "$", "-$50.00"},
		{"negative mid", "-1500", "$", "-$1,500.00"},
		{"negative large", "-15000", "$", "-$15K"},

		// ISO code currencies — space between code and amount
		{"ISO code small", "42.50", "CNY", "CNY 42.50"},
		{"ISO code mid-range", "1500", "CNY", "CNY 1,500.00"},
		{"ISO code large K", "15000", "CNY", "CNY 15K"},
		{"ISO code large M", "1500000", "CNY", "CNY 1.5M"},
		{"ISO code zero", "0", "CNY", "CNY 0.00"},
		{"ISO code negative small", "-50.00", "CNY", "-CNY 50.00"},
		{"ISO code negative large", "-15000", "CNY", "-CNY 15K"},
		{"VND zero-decimal", "5000", "VND", "VND 5,000"},
		{"KRW zero-decimal", "5000", "KRW", "KRW 5,000"},

		// Regression: symbol currencies must NOT gain a space
		{"JPY symbol unaffected", "5000", "JPY", "¥5,000"},
		{"EUR symbol unaffected", "100", "EUR", "€100.00"},
		{"GBP symbol unaffected", "100", "GBP", "£100.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, _ := decimal.NewFromString(tt.value)
			c := types.NewCurrency(value, tt.symbol)
			got := FormatCurrency(c)
			if got != tt.expected {
				t.Errorf("FormatCurrency(%s%s) = %q, want %q", tt.symbol, tt.value, got, tt.expected)
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

// TestDefaultFormatterMatchesPackageFunctions verifies that the Formatter struct
// with DefaultConfig produces identical output to the package-level free functions.
func TestDefaultFormatterMatchesPackageFunctions(t *testing.T) {
	f := DefaultFormatter()

	tests := []struct {
		name  string
		value types.Type
	}{
		{"number 100K", types.NewNumber(decimal.NewFromInt(100000))},
		{"number 1.5M", types.NewNumber(decimal.NewFromInt(1500000))},
		{"number small", types.NewNumber(decimal.NewFromFloat(42.50))},
		{"number zero", types.NewNumber(decimal.NewFromInt(0))},
		{"number negative", types.NewNumber(decimal.NewFromInt(-5000))},
		{"currency small", types.NewCurrency(decimal.NewFromFloat(42.50), "$")},
		{"currency mid", types.NewCurrency(decimal.NewFromInt(1500), "$")},
		{"currency large", types.NewCurrency(decimal.NewFromInt(1500000), "$")},
		{"currency JPY", types.NewCurrency(decimal.NewFromInt(100), "JPY")},
		{"quantity known", types.NewQuantity(decimal.NewFromInt(1000), "m")},
		{"quantity arbitrary", types.NewQuantity(decimal.NewFromInt(100000), "users")},
		{"boolean true", types.NewBoolean(true)},
		{"nil", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkgResult := Format(tt.value)
			fmtResult := f.Format(tt.value)
			if pkgResult != fmtResult {
				t.Errorf("mismatch: Format()=%q, Formatter.Format()=%q", pkgResult, fmtResult)
			}
		})
	}
}

func TestDisplayConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     DisplayConfig
		wantErr bool
	}{
		{"valid en-US", DefaultConfig(), false},
		{"valid de-DE", DisplayConfig{DecimalSep: ",", ThousandSep: "."}, false},
		{"empty decimal", DisplayConfig{DecimalSep: "", ThousandSep: ","}, true},
		{"empty thousand", DisplayConfig{DecimalSep: ".", ThousandSep: ""}, true},
		{"same separators", DisplayConfig{DecimalSep: ".", ThousandSep: "."}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
