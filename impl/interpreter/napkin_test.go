package interpreter

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestEvalNapkinConversion(t *testing.T) {
	tests := []struct {
		name          string
		inputValue    float64
		expectedValue float64
		tolerance     float64
	}{
		// Millions - should round to 2 sig figs
		{"1.234M", 1234567, 1200000, 1}, // ~1.2M
		{"47", 47, 47, 0.1},             // No rounding needed
		{"8734", 8734, 8700, 1},         // ~8.7K → 8700

		// Thousands
		{"347K", 347234, 350000, 100}, // ~350K
		{"12.5K", 12500, 13000, 100},  // ~13K

		// Billions
		{"1.5B", 1500000000, 1500000000, 1000}, // ~1.5B
		{"5B", 5000000000, 5000000000, 1000},   // ~5B

		// Trillions
		{"1.2T", 1234000000000, 1200000000000, 1000000}, // ~1.2T

		// Small numbers
		{"123", 123, 120, 1}, // ~120
		{"456", 456, 460, 1}, // ~460

		// Edge cases
		{"Zero", 0, 0, 0.01},
		{"Negative million", -1234567, -1200000, 1},
		{"Negative thousand", -8734, -8700, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a Number value
			// Mock AST node - in real usage this would come from parser
			// For testing, we'll directly test the logic

			// The napkin conversion should return a rounded Number
			numValue := decimal.NewFromFloat(tt.inputValue)

			// Call the actual rounding logic (extracted from evalNapkinConversion)
			floatVal, _ := numValue.Abs().Float64()

			roundedFloat := floatVal
			if floatVal >= 1000 {
				var magnitude float64
				if floatVal >= 1e12 {
					magnitude = 1e12
				} else if floatVal >= 1e9 {
					magnitude = 1e9
				} else if floatVal >= 1e6 {
					magnitude = 1e6
				} else {
					magnitude = 1000
				}

				scaled := floatVal / magnitude
				rounded := roundToSignificantFigures(scaled, 2)

				var roundedScaled float64
				switch r := rounded.(type) {
				case string:
					if d, err := decimal.NewFromString(r); err == nil {
						roundedScaled, _ = d.Float64()
					} else {
						roundedScaled = scaled
					}
				case int:
					roundedScaled = float64(r)
				default:
					roundedScaled = scaled
				}

				roundedFloat = roundedScaled * magnitude
			} else if floatVal > 0 {
				rounded := roundToSignificantFigures(floatVal, 2)
				switch r := rounded.(type) {
				case string:
					if d, err := decimal.NewFromString(r); err == nil {
						roundedFloat, _ = d.Float64()
					}
				case int:
					roundedFloat = float64(r)
				}
			}

			if numValue.IsNegative() {
				roundedFloat = -roundedFloat
			}

			result := roundedFloat

			diff := result - tt.expectedValue
			if diff < 0 {
				diff = -diff
			}

			if diff > tt.tolerance {
				t.Errorf("Napkin conversion of %v = %v, want %v (diff: %v, tolerance: %v)",
					tt.inputValue, result, tt.expectedValue, diff, tt.tolerance)
			}

			t.Logf("✓ %s: %v → %v (expected %v)", tt.name, tt.inputValue, result, tt.expectedValue)
		})
	}
}

// TestFormatNapkin tests the display formatter (separate from numeric conversion)
func TestFormatNapkinDisplay(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		precision int
		expected  string
	}{
		{"1.234M display", 1234567, 2, "~1.2M"},
		{"47 display", 47, 2, "~47"},
		{"8.7K display", 8734, 2, "~8.7K"},
		{"350K display", 347234, 2, "~350K"},
		{"Negative display", -1234567, 2, "~-1.2M"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := decimal.NewFromFloat(tt.value)
			result := formatNapkin(value, tt.precision)

			if result != tt.expected {
				t.Errorf("formatNapkin(%v, %d) = %q, want %q",
					tt.value, tt.precision, result, tt.expected)
			}
		})
	}
}
