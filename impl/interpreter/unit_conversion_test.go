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

// TestRateUnitConversion tests rate-to-rate unit conversion.
// Converts the quantity unit while preserving or changing the time unit.
func TestRateUnitConversion(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    float64
		tolerance   float64
		wantUnit    string // Expected unit in output (e.g., "inch")
		wantTimeAbb string // Expected time abbreviation (e.g., "/s")
	}{
		{
			name:        "meters per second to inches per second",
			input:       "10 m/s in inch/s\n",
			expected:    393.7, // 10 * 39.37 ≈ 393.7
			tolerance:   0.1,
			wantUnit:    "inch",
			wantTimeAbb: "/s",
		},
		{
			name:        "km per hour to miles per hour",
			input:       "100 km/h in mile/h\n",
			expected:    62.137, // 100 / 1.60934 ≈ 62.137
			tolerance:   0.01,
			wantUnit:    "mile",
			wantTimeAbb: "/h",
		},
		{
			name:        "rate conversion with time unit change",
			input:       "60 m/s in m/min\n",
			expected:    3600, // 60 * 60 = 3600 m/min
			tolerance:   0.1,
			wantUnit:    "m",
			wantTimeAbb: "/min",
		},
		{
			name:        "combined quantity and time conversion",
			input:       "1 km/h in m/s\n",
			expected:    0.2778, // 1000m / 3600s ≈ 0.2778
			tolerance:   0.001,
			wantUnit:    "m",
			wantTimeAbb: "/s",
		},
		{
			name:        "feet per second to meters per second",
			input:       "100 feet/s in m/s\n",
			expected:    30.48, // 100 * 0.3048 = 30.48
			tolerance:   0.01,
			wantUnit:    "m",
			wantTimeAbb: "/s",
		},
		{
			name:        "rate with time-only target (10 MB/day in seconds)",
			input:       "10 MB/day in seconds\n",
			expected:    0.0001157, // 10 / 86400 ≈ 0.0001157
			tolerance:   0.0000001,
			wantUnit:    "MB",
			wantTimeAbb: "/s",
		},
		{
			name:        "rate with time-only target (100 GB/month in hours)",
			input:       "100 GB/month in hours\n",
			expected:    0.1388889, // 100 / 720 ≈ 0.139
			tolerance:   0.001,
			wantUnit:    "GB",
			wantTimeAbb: "/h",
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

			result := results[0].String()
			t.Logf("Result: %s", result)

			// Extract numeric value
			var actual float64
			_, err = fmt.Sscanf(result, "%f", &actual)
			if err != nil {
				t.Fatalf("Could not parse result %q as number: %v", result, err)
			}

			diff := math.Abs(actual - tt.expected)
			if diff > tt.tolerance {
				t.Errorf("Value = %f, expected %f (diff: %f, tolerance: %f)",
					actual, tt.expected, diff, tt.tolerance)
			}

			// Check output contains expected unit
			if !strings.Contains(result, tt.wantUnit) {
				t.Errorf("Result %q does not contain expected unit %q", result, tt.wantUnit)
			}

			// Check output contains expected time abbreviation
			if !strings.Contains(result, tt.wantTimeAbb) {
				t.Errorf("Result %q does not contain expected time unit %q", result, tt.wantTimeAbb)
			}
		})
	}
}

// TestRateUnitConversionErrors tests error cases for rate unit conversion.
func TestRateUnitConversionErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErrPart string // Substring expected in error message
	}{
		{
			name:        "quantity conversion on rate target",
			input:       "10 meters in inch/s\n",
			wantErrPart: "requires a rate",
		},
		{
			name:        "incompatible quantity types",
			input:       "10 m/s in kg/s\n",
			wantErrPart: "cannot convert",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			interp := interpreter.NewInterpreter()
			_, err = interp.Eval(nodes)
			if err == nil {
				t.Fatal("Expected error but got none")
			}

			if !strings.Contains(err.Error(), tt.wantErrPart) {
				t.Errorf("Error %q does not contain %q", err.Error(), tt.wantErrPart)
			}
		})
	}
}
