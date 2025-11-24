package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// BenchmarkUnitConversion benchmarks the unit conversion performance
func BenchmarkUnitConversion(b *testing.B) {
	qty := &types.Quantity{
		Value: decimal.NewFromInt(100),
		Unit:  "meters",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = convertQuantity(qty, "feet")
	}
}

// BenchmarkUnitConversion_SameUnit benchmarks conversion when units are the same (fast path)
func BenchmarkUnitConversion_SameUnit(b *testing.B) {
	qty := &types.Quantity{
		Value: decimal.NewFromInt(100),
		Unit:  "meters",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = convertQuantity(qty, "meters")
	}
}

// BenchmarkMultipleConversions tests realistic scenario with multiple conversions
func BenchmarkMultipleConversions(b *testing.B) {
	units := []struct {
		from string
		to   string
	}{
		{"meters", "feet"},
		{"kilometers", "miles"},
		{"kilograms", "pounds"},
		{"liters", "gallons"},
		{"grams", "ounces"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, u := range units {
			qty := &types.Quantity{
				Value: decimal.NewFromInt(100),
				Unit:  u.from,
			}
			_, _ = convertQuantity(qty, u.to)
		}
	}
}
