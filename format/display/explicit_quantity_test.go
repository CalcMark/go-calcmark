package display

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

func TestFormatExplicitQuantity(t *testing.T) {
	f := DefaultFormatter()

	tests := []struct {
		name     string
		value    decimal.Decimal
		unit     string
		expected string
	}{
		{
			name:     "200 kW in MW shows 0.2 MW",
			value:    decimal.NewFromFloat(0.2),
			unit:     "megawatts",
			expected: "0.2 megawatts",
		},
		{
			name:     "500 kW in MW shows 0.5 MW",
			value:    decimal.NewFromFloat(0.5),
			unit:     "megawatts",
			expected: "0.5 megawatts",
		},
		{
			name:     "1 m in mm shows 1,000 mm",
			value:    decimal.NewFromInt(1000),
			unit:     "millimeters",
			expected: "1,000 millimeters",
		},
		{
			name:     "1 GB in MB shows 1,024 MB",
			value:    decimal.NewFromFloat(1024),
			unit:     "MB",
			expected: "1,024 MB",
		},
		{
			name:     "extreme small value uses scientific notation",
			value:    decimal.NewFromFloat(8.88e-16),
			unit:     "PB",
			expected: "8.88e-16 PB",
		},
		{
			name:     "extreme large value uses scientific notation",
			value:    decimal.NewFromFloat(1.126e15),
			unit:     "bytes",
			expected: "1.126e+15 bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &types.Quantity{
				Value:      tt.value,
				Unit:       tt.unit,
				IsExplicit: true,
			}
			result := f.FormatQuantity(q)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExplicitQuantityRounding(t *testing.T) {
	f := DefaultFormatter()

	tests := []struct {
		name     string
		value    decimal.Decimal
		unit     string
		expected string
	}{
		{
			name:     "value >= 100 rounds to 0 dp",
			value:    decimal.NewFromFloat(156521.739),
			unit:     "millisecond",
			expected: "156,522 millisecond",
		},
		{
			name:     "value 10-99 rounds to 1 dp",
			value:    decimal.NewFromFloat(32.808399),
			unit:     "feet",
			expected: "32.8 feet",
		},
		{
			name:     "value 1-9 rounds to 2 dp",
			value:    decimal.NewFromFloat(3.14159),
			unit:     "meters",
			expected: "3.14 meters",
		},
		{
			name:     "value < 1 rounds to 4 dp",
			value:    decimal.NewFromFloat(0.00027778),
			unit:     "hour",
			expected: "0.0003 hour",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &types.Quantity{
				Value:      tt.value,
				Unit:       tt.unit,
				IsExplicit: true,
			}
			result := f.FormatQuantity(q)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExplicitQuantityPreciseSkipsRounding(t *testing.T) {
	f := DefaultFormatter()

	tests := []struct {
		name     string
		value    decimal.Decimal
		unit     string
		expected string
	}{
		{
			name:     "precise shows full value for large number",
			value:    decimal.NewFromFloat(156521.739),
			unit:     "millisecond",
			expected: "156,521.739 millisecond",
		},
		{
			name:     "precise shows full value for medium number",
			value:    decimal.NewFromFloat(32.808399),
			unit:     "feet",
			expected: "32.808399 feet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &types.Quantity{
				Value:      tt.value,
				Unit:       tt.unit,
				IsExplicit: true,
				IsPrecise:  true,
			}
			result := f.FormatQuantity(q)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExplicitTakesPrecedenceOverNapkin(t *testing.T) {
	// If both flags were ever set, IsExplicit wins (checked first in FormatQuantity).
	// This guards the ordering so a future refactor doesn't silently break it.
	f := DefaultFormatter()

	q := &types.Quantity{
		Value:      decimal.NewFromFloat(0.2),
		Unit:       "megawatts",
		IsExplicit: true,
		IsNapkin:   true,
	}
	result := f.FormatQuantity(q)

	// IsExplicit path does not add "~" prefix — explicit display wins
	if result != "0.2 megawatts" {
		t.Errorf("expected %q, got %q", "0.2 megawatts", result)
	}
}

func TestExplicitFlagDroppedByArithmetic(t *testing.T) {
	// Verify that non-explicit quantities still auto-scale normally
	f := DefaultFormatter()

	q := &types.Quantity{
		Value:      decimal.NewFromFloat(0.2),
		Unit:       "megawatts",
		IsExplicit: false,
	}
	result := f.FormatQuantity(q)

	// Without explicit flag, 0.2 MW should auto-scale to 200 kW
	if result != "200 kW" {
		t.Errorf("non-explicit quantity should auto-scale: expected '200 kW', got %q", result)
	}
}
