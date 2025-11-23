package interpreter_test

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestUnitConversionAccuracy tests that unit conversions are mathematically correct
func TestUnitConversionAccuracy(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  float64
		tolerance float64 // Allow small floating point errors
	}{
		// Length conversions
		{
			name:      "10 meters + 5 feet",
			input:     "10 meters + 5 feet\n",
			expected:  11.524, // 10 + (5 * 0.3048) = 11.524 meters
			tolerance: 0.001,
		},
		{
			name:      "1 kilometer + 1000 meters",
			input:     "1 kilometer + 1000 meters\n",
			expected:  2.0, // 1 + 1 = 2 km
			tolerance: 0.001,
		},
		{
			name:      "100 inches + 1 foot",
			input:     "100 inches + 1 foot\n",
			expected:  112.0, // 100 + 12 = 112 inches
			tolerance: 0.001,
		},
		{
			name:      "1 mile + 1000 feet",
			input:     "1 mile + 1000 feet\n",
			expected:  1.189394, // 1 + (1000/5280) ≈ 1.189 miles
			tolerance: 0.001,
		},

		// Mass conversions
		{
			name:      "5 kg + 10 lb",
			input:     "5 kg + 10 lb\n",
			expected:  9.5359237, // 5 + (10 * 0.45359237) = 9.5359237 kg
			tolerance: 0.0001,
		},
		{
			name:      "1000 grams + 1 kg",
			input:     "1000 grams + 1 kg\n",
			expected:  2000.0, // 1000 + 1000 = 2000 grams
			tolerance: 0.001,
		},
		{
			name:      "16 ounces + 1 pound",
			input:     "16 ounces + 1 pound\n",
			expected:  32.0, // 16 + 16 = 32 ounces
			tolerance: 0.001,
		},

		// Volume conversions
		{
			name:      "1 liter + 1000 ml",
			input:     "1 liter + 1000 ml\n",
			expected:  2.0, // 1 + 1 = 2 liters
			tolerance: 0.001,
		},
		{
			name:      "1 gallon + 1 liter",
			input:     "1 gallon + 1 liter\n",
			expected:  1.264172, // 1 + (1 / 3.785411784) ≈ 1.264 gallons (first-unit-wins!)
			tolerance: 0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			interp := interpreter.NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			// Get the numeric value from the result
			result := results[0].String()
			var actual float64
			_, err = fmt.Sscanf(result, "%f", &actual)
			if err != nil {
				t.Fatalf("Could not parse result %q as number: %v", result, err)
			}

			diff := math.Abs(actual - tt.expected)
			if diff > tt.tolerance {
				t.Errorf("Result = %f, expected %f (diff: %f, tolerance: %f)",
					actual, tt.expected, diff, tt.tolerance)
			}
		})
	}
}

// TestFirstUnitWins verifies that the result is always in the first operand's unit
func TestFirstUnitWins(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedUnit string
	}{
		{"meters first", "10 meters + 5 feet\n", "meters"},
		{"feet first", "5 feet + 10 meters\n", "feet"},
		{"kg first", "5 kg + 10 lb\n", "kg"},
		{"lb first", "10 lb + 5 kg\n", "lb"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			interp := interpreter.NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			result := results[0].String()
			if !strings.Contains(result, tt.expectedUnit) {
				t.Errorf("Result %q does not contain expected unit %q", result, tt.expectedUnit)
			}
		})
	}
}
