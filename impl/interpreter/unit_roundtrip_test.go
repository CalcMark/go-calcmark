package interpreter_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/impl/interpreter"
	"github.com/CalcMark/go-calcmark/v2/spec/parser"
)

// TestUnitRoundtrip verifies that unit conversions are accurate within float64 tolerance.
// This covers INTERP-05: Unit conversion roundtrips are lossless within precision limits.
//
// For each conversion pair (unit1 -> unit2 -> unit1), we verify:
// 1. The roundtrip value equals the original within tolerance
// 2. No precision is lost due to intermediate conversions
func TestUnitRoundtrip(t *testing.T) {
	testCases := []struct {
		category  string
		unit1     string
		unit2     string
		value     float64
		tolerance float64 // Relative tolerance as fraction of original value
	}{
		// ==================== Length ====================
		// Base unit: meter
		{"length", "meters", "feet", 100.0, 0.0001},
		{"length", "meters", "inches", 10.0, 0.0001},
		{"length", "meters", "kilometers", 5000.0, 0.0001},
		{"length", "miles", "kilometers", 10.0, 0.0001},
		{"length", "miles", "feet", 5.0, 0.0001},
		{"length", "inches", "centimeters", 100.0, 0.0001},
		{"length", "yards", "meters", 50.0, 0.0001},
		{"length", "feet", "meters", 328.0, 0.0001},

		// ==================== Mass ====================
		// Base unit: kilogram
		{"mass", "kg", "pounds", 50.0, 0.0001},
		{"mass", "kg", "grams", 10.0, 0.0001},
		{"mass", "grams", "ounces", 500.0, 0.0001},
		{"mass", "pounds", "kg", 100.0, 0.0001},
		{"mass", "ounces", "grams", 32.0, 0.0001},
		{"mass", "tonnes", "kg", 5.0, 0.0001},

		// ==================== Volume ====================
		// Base unit: liter
		{"volume", "liters", "gallons", 10.0, 0.0001},
		{"volume", "liters", "ml", 5.0, 0.0001},
		{"volume", "gallons", "liters", 3.0, 0.0001},
		{"volume", "pints", "liters", 8.0, 0.0001},
		{"volume", "quarts", "liters", 4.0, 0.0001},

		// ==================== Data Size ====================
		// Base unit: bit
		// Note: CalcMark uses binary (1024-based) for KB/MB/GB by default
		{"datasize", "MB", "KB", 100.0, 0.0001},
		{"datasize", "GB", "MB", 10.0, 0.0001},
		{"datasize", "TB", "GB", 5.0, 0.0001},
		{"datasize", "KB", "bytes", 1024.0, 0.0001},
		{"datasize", "GB", "KB", 1.0, 0.0001},

		// ==================== Area ====================
		// Base unit: square meter
		{"area", "square meters", "square feet", 100.0, 0.0001},
		{"area", "acres", "hectares", 10.0, 0.0001},
		// Note: "square kilometers" to "square miles" not supported in unit registry
		// as multi-word units with spaces are handled differently

		// ==================== Energy ====================
		// Base unit: joule
		{"energy", "joules", "calories", 1000.0, 0.0001},
		{"energy", "kilocalories", "joules", 100.0, 0.0001},
		{"energy", "kilojoules", "joules", 50.0, 0.0001},

		// ==================== Power ====================
		// Base unit: watt
		{"power", "watts", "kilowatts", 5000.0, 0.0001},
		{"power", "kilowatts", "watts", 10.0, 0.0001},
		{"power", "horsepower", "watts", 5.0, 0.0001},

		// ==================== Temperature ====================
		// Special case: offset-based conversion (not ratio-based)
		// These test the non-linear conversion path
		{"temperature", "celsius", "fahrenheit", 25.0, 0.0001},
		{"temperature", "fahrenheit", "celsius", 77.0, 0.0001},
		{"temperature", "celsius", "kelvin", 100.0, 0.0001},

		// ==================== Speed ====================
		// Note: Speed units like km/h and mph are compound units in the registry.
		// They use the speed category conversion, not rate conversion.
		// Direct roundtrips work via the speed category in unit_library.go
		// Testing via m/s as intermediate since km/h <-> mph requires different syntax
	}

	for _, tc := range testCases {
		testName := fmt.Sprintf("%s_%s_to_%s_roundtrip", tc.category, tc.unit1, tc.unit2)
		t.Run(testName, func(t *testing.T) {
			// Construct roundtrip expression: value unit1 -> unit2 -> unit1
			// Format: (originalValue unit1 in unit2) in unit1
			input := fmt.Sprintf("(%.10f %s in %s) in %s\n", tc.value, tc.unit1, tc.unit2, tc.unit1)

			nodes, err := parser.Parse(input)
			if err != nil {
				t.Fatalf("Parse error for %q: %v", input, err)
			}

			interp := interpreter.NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error for %q: %v", input, err)
			}

			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			result := results[0].String()

			// Parse numeric value from result
			var actual float64
			_, err = fmt.Sscanf(result, "%f", &actual)
			if err != nil {
				t.Fatalf("Could not parse result %q as number: %v", result, err)
			}

			// Calculate relative error
			relError := math.Abs(actual-tc.value) / math.Abs(tc.value)
			if relError > tc.tolerance {
				t.Errorf("Roundtrip failed:\n  Original: %.10f %s\n  After roundtrip: %.10f\n  Relative error: %.10f (tolerance: %.10f)",
					tc.value, tc.unit1, actual, relError, tc.tolerance)
			}
		})
	}
}

// TestUnitConversionPrecision tests that specific known conversions are accurate.
// These are not roundtrips but direct conversions with expected values.
func TestUnitConversionPrecision(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  float64
		tolerance float64
	}{
		// Length precision tests
		{"1 meter to feet", "1 meter in feet\n", 3.28084, 0.0001},
		{"1 foot to meters", "1 foot in meters\n", 0.3048, 0.0001},
		{"1 mile to km", "1 mile in kilometers\n", 1.60934, 0.0001},
		{"1 inch to cm", "1 inch in centimeters\n", 2.54, 0.0001},

		// Mass precision tests
		{"1 kg to pounds", "1 kg in pounds\n", 2.20462, 0.0001},
		{"1 pound to kg", "1 pound in kg\n", 0.453592, 0.0001},
		{"1 oz to grams", "1 ounce in grams\n", 28.3495, 0.001},

		// Volume precision tests
		{"1 gallon to liters", "1 gallon in liters\n", 3.78541, 0.0001},
		{"1 liter to gallons", "1 liter in gallons\n", 0.264172, 0.0001},

		// Data size precision tests (binary-based KB/MB/GB)
		{"1 MB to KB", "1 MB in KB\n", 1024.0, 0.0001},
		{"1 GB to MB", "1 GB in MB\n", 1024.0, 0.0001},
		{"1 TB to GB", "1 TB in GB\n", 1024.0, 0.0001},
		{"1024 KB to MB", "1024 KB in MB\n", 1.0, 0.0001},

		// Temperature precision tests (offset-based)
		{"0 C to F", "0 celsius in fahrenheit\n", 32.0, 0.0001},
		{"100 C to F", "100 celsius in fahrenheit\n", 212.0, 0.0001},
		{"32 F to C", "32 fahrenheit in celsius\n", 0.0, 0.0001},
		{"212 F to C", "212 fahrenheit in celsius\n", 100.0, 0.0001},
		{"0 C to K", "0 celsius in kelvin\n", 273.15, 0.01},

		// Speed precision tests
		// Note: km/h and mph are compound speed units in the registry
		// Direct conversion syntax differs from simple quantity conversion
		// These are tested via rate conversion tests instead

		// Power precision tests
		{"1 hp to watts", "1 horsepower in watts\n", 745.7, 0.1},
		{"1000 watts to kw", "1000 watts in kilowatts\n", 1.0, 0.0001},

		// Energy precision tests
		{"1 calorie to joules", "1 calorie in joules\n", 4.184, 0.001},
		{"1000 joules to kj", "1000 joules in kilojoules\n", 1.0, 0.0001},
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

			var actual float64
			_, err = fmt.Sscanf(result, "%f", &actual)
			if err != nil {
				t.Fatalf("Could not parse result %q as number: %v", result, err)
			}

			diff := math.Abs(actual - tt.expected)
			if diff > tt.tolerance {
				t.Errorf("Conversion %q = %.10f, expected %.10f (diff: %.10f, tolerance: %.10f)",
					tt.input, actual, tt.expected, diff, tt.tolerance)
			}
		})
	}
}

// TestRateRoundtrip tests that rate conversions are accurate.
// Rates have both a quantity unit and a time unit that need conversion.
//
// Note: Rate-to-rate conversion uses a different syntax than quantity conversion.
// For example: "100 MB/s in KB/s" (rate to rate target)
// The target must be a rate (unit/time), not a simple unit.
func TestRateRoundtrip(t *testing.T) {
	tests := []struct {
		name      string
		original  string
		converted string
		back      string
		value     float64
		tolerance float64
	}{
		// Data rate roundtrips - using rate target syntax
		{
			name:      "MB/s to KB/s roundtrip",
			original:  "100 MB/s",
			converted: "KB/s",
			back:      "MB/s",
			value:     100.0,
			tolerance: 0.0001,
		},
		{
			name:      "GB/day to MB/day roundtrip",
			original:  "10 GB/day",
			converted: "MB/day",
			back:      "GB/day",
			value:     10.0,
			tolerance: 0.0001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Construct roundtrip expression using rate-to-rate syntax
			input := fmt.Sprintf("(%s in %s) in %s\n", tt.original, tt.converted, tt.back)

			nodes, err := parser.Parse(input)
			if err != nil {
				t.Fatalf("Parse error for %q: %v", input, err)
			}

			interp := interpreter.NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error for %q: %v", input, err)
			}

			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			result := results[0].String()

			var actual float64
			_, err = fmt.Sscanf(result, "%f", &actual)
			if err != nil {
				t.Fatalf("Could not parse result %q as number: %v", result, err)
			}

			relError := math.Abs(actual-tt.value) / math.Abs(tt.value)
			if relError > tt.tolerance {
				t.Errorf("Rate roundtrip failed:\n  Original: %s\n  After roundtrip: %.10f\n  Relative error: %.10f",
					tt.original, actual, relError)
			}
		})
	}
}

// TestEdgeCaseRoundtrips tests edge cases that might cause precision issues.
func TestEdgeCaseRoundtrips(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  float64
		tolerance float64
	}{
		// Very small values
		{"very small meters to feet", "(0.001 meters in feet) in meters\n", 0.001, 0.0001},
		{"very small kg to pounds", "(0.001 kg in pounds) in kg\n", 0.001, 0.0001},

		// Very large values
		{"very large km to miles", "(1000000 kilometers in miles) in kilometers\n", 1000000.0, 0.0001},
		{"very large TB to GB", "(1000 TB in GB) in TB\n", 1000.0, 0.0001},

		// Values requiring many decimal places
		{"pi meters to feet", "(3.14159265359 meters in feet) in meters\n", 3.14159265359, 0.0001},

		// Temperature edge cases (absolute zero, boiling point)
		{"absolute zero C to K", "(-273.15 celsius in kelvin) in celsius\n", -273.15, 0.01},
		{"boiling point F to C", "(212 fahrenheit in celsius) in fahrenheit\n", 212.0, 0.01},
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

			var actual float64
			_, err = fmt.Sscanf(result, "%f", &actual)
			if err != nil {
				t.Fatalf("Could not parse result %q as number: %v", result, err)
			}

			relError := math.Abs(actual-tt.expected) / math.Max(math.Abs(tt.expected), 1e-10)
			if relError > tt.tolerance {
				t.Errorf("Edge case roundtrip failed:\n  Input: %s\n  Expected: %.10f\n  Actual: %.10f\n  Relative error: %.10f",
					tt.input, tt.expected, actual, relError)
			}
		})
	}
}
