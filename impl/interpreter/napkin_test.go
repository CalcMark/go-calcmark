package interpreter

import (
	"fmt"
	"testing"

	"github.com/shopspring/decimal"
)

func TestFormatNapkin(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		precision int
		expected  string
	}{
		// Millions
		{"1.234M", 1234567, 2, "~1.2M"},
		{"47", 47, 2, "~47"},
		{"8734", 8734, 2, "~8700"},
		{"2.347P (as M)", 2347000, 2, "~2.3M"},

		// Thousands
		{"347K", 347234, 2, "~350K"},
		{"12.5K", 12500, 2, "~13K"},

		// Billions
		{"1.5B", 1500000000, 2, "~1.5B"},
		{"5B", 5000000000, 2, "~5B"},

		// Trillions
		{"1.2T", 1234000000000, 2, "~1.2T"},

		// Small numbers (< 1000)
		{"123", 123, 2, "~120"},
		{"9.87", 9.87, 2, "~9.9"},
		{"456", 456, 2, "~460"},

		// Edge cases
		{"Zero", 0, 2, "~0"},
		{"Very small", 0.001, 2, "~0"},
		{"Negative million", -1234567, 2, "~-1.2M"},
		{"Negative thousand", -8734, 2, "~-8700"},

		// Different precisions
		{"1 sig fig", 1234567, 1, "~1M"},
		{"3 sig figs", 1234567, 3, "~1.23M"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := decimal.NewFromFloat(tt.value)
			result := formatNapkin(value, tt.precision)

			if result != tt.expected {
				t.Errorf("formatNapkin(%v, %d) = %q, want %q",
					tt.value, tt.precision, result, tt.expected)
			}

			t.Logf("✓ %s: %v → %s", tt.name, tt.value, result)
		})
	}
}

func TestRoundToSignificantFigures(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		sigFigs  int
		expected string
	}{
		{"1.234 to 2", 1.234567, 2, "1.2"},
		{"8734 to 2", 8734, 2, "8700"},
		{"0.0123 to 2", 0.0123, 2, "0.012"},
		{"123 to 2", 123, 2, "120"},
		{"999 to 2", 999, 2, "1000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := roundToSignificantFigures(tt.value, tt.sigFigs)
			resultStr := fmt.Sprintf("%v", result)

			if resultStr != tt.expected {
				t.Errorf("roundToSignificantFigures(%v, %d) = %v, want %s",
					tt.value, tt.sigFigs, result, tt.expected)
			}

			t.Logf("✓ %s: %v → %v", tt.name, tt.value, result)
		})
	}
}
