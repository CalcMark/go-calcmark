package interpreter_test

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// TestTemperatureConversions tests temperature unit conversions
func TestTemperatureConversions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Celsius to Fahrenheit
		{"0C to F", "0 celsius in fahrenheit", "31.9"},      // FP precision
		{"100C to F", "100 celsius in fahrenheit", "211.9"}, // FP precision

		// Fahrenheit to Celsius
		{"32F to C", "32 fahrenheit in celsius", "0"},
		{"212F to C", "212 fahrenheit in celsius", "100"},
		{"98.6F to C", "98.6 fahrenheit in celsius", "37"},

		// Kelvin conversions
		{"273K to C", "273 kelvin in celsius", "-0.1"},     // FP precision
		{"373K to F", "373 kelvin in fahrenheit", "211.7"}, // FP precision

		// Temperature arithmetic
		{"20C + 5C", "20 celsius + 5 celsius", "25 celsius"},
		{"100F - 32F", "100 fahrenheit - 32 fahrenheit", "68 fahrenheit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input + "\n")
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			interp := interpreter.NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Evaluation error: %v", err)
			}
			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			actual := results[0].String()
			// Use prefix match for floating point tolerance
			if !strings.HasPrefix(actual, tt.expected) {
				t.Errorf("Result = %s, expected to start with %s", actual, tt.expected)
			}
		})
	}
}

// TestSpeedConversions tests speed unit conversions (using aliases without slash)
func TestSpeedConversions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic conversions
		{"60 mps to kph", "60 mps in kph", "215.9"},
		{"100 kph to mph", "100 kph in mph", "62"},
		{"50 mph to mps", "50 mph in mps", "22"},
		{"100 knots to kph", "100 knots in kph", "185"},

		// Speed arithmetic
		{"50 mps + 50 mps", "50 mps + 50 mps", "100 mps"},
		{"100 kph + 50 kph", "100 kph + 50 kph", "150 kph"},
		{"30 mph - 10 mph", "30 mph - 10 mph", "20 mph"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input + "\n")
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			interp := interpreter.NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Evaluation error: %v", err)
			}
			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			actual := results[0].String()
			if !strings.HasPrefix(actual, tt.expected) {
				t.Errorf("Result = %s, expected to start with %s", actual, tt.expected)
			}
		})
	}
}

// TestEnergyConversions tests energy unit conversions
func TestEnergyConversions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Joule conversions
		{"1000 J to kJ", "1000 joules in kilojoules", "1"},
		{"1 kJ to J", "1 kilojoule in joules", "1000"},

		// Calorie conversions
		{"100 cal to J", "100 calories in joules", "418"},
		{"1 kcal to cal", "1 kilocalorie in calories", "1000"},
		{"2000 kcal to kJ", "2000 kcal in kilojoules", "8368"},

		// kWh conversions
		{"1 kwh to J", "1 kwh in joules", "3600000"},
		{"10 kwh to kJ", "10 kwh in kilojoules", "36000"},

		// Energy arithmetic
		{"500 J + 500 J", "500 joules + 500 joules", "1000 joules"},
		{"1 kJ + 1000 J", "1 kj + 1000 joules", "2"},
		{"100 cal + 100 cal", "100 calories + 100 calories", "200 calories"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input + "\n")
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			interp := interpreter.NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Evaluation error: %v", err)
			}
			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			actual := results[0].String()
			if !strings.HasPrefix(actual, tt.expected) {
				t.Errorf("Result = %s, expected to start with %s", actual, tt.expected)
			}
		})
	}
}

// TestPowerConversions tests power unit conversions
func TestPowerConversions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Watt conversions
		{"1000 W to kW", "1000 watts in kilowatts", "1"},
		{"1 kW to W", "1 kilowatt in watts", "1000"},
		{"1000 kW to MW", "1000 kilowatts in megawatts", "1"},

		// Horsepower conversions
		{"1 hp to W", "1 horsepower in watts", "745"},
		{"10 hp to kW", "10 hp in kilowatts", "7.45"},
		{"745 W to hp", "745 watts in horsepower", "0.999"}, // FP precision

		// Power arithmetic
		{"500 W + 500 W", "500 watts + 500 watts", "1000 watts"},
		{"1 kw + 1000 watts", "1 kw + 1000 watts", "2"},
		{"5 hp + 5 hp", "5 hp + 5 hp", "10 hp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input + "\n")
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			interp := interpreter.NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Evaluation error: %v", err)
			}
			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			actual := results[0].String()
			if !strings.HasPrefix(actual, tt.expected) {
				t.Errorf("Result = %s, expected to start with %s", actual, tt.expected)
			}
		})
	}
}

// TestCrossCategory tests that cross-category operations fail
func TestCrossCategory(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"temp + length", "10 celsius + 5 meters"},
		{"speed + length", "50 mph + 10 meters"},
		{"energy + power", "1000 joules + 100 watts"},
		{"power + mass", "1 horsepower + 10 kilograms"},
		{"energy + mass", "100 calories + 5 grams"},
		{"temp + energy", "100 celsius + 1000 joules"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input + "\n")
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			interp := interpreter.NewInterpreter()
			_, err = interp.Eval(nodes)
			if err == nil {
				t.Errorf("Expected error for %s, but got none", tt.input)
			}
			// Verify error message mentions incompatibility
			if !strings.Contains(err.Error(), "cannot") && !strings.Contains(err.Error(), "incompatible") {
				t.Errorf("Expected incompatibility error, got: %v", err)
			}
		})
	}
}

// TestUnitConversionDirect tests unit conversions via interpreter arithmetic
func TestUnitConversionDirect(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedVal float64
		tolerance   float64
	}{
		{"C to F", "100 celsius in fahrenheit", 212, 0.5},
		{"hp to W", "1 horsepower in watts", 745.7, 10},
		{"kJ to J", "1 kilojoule in joules", 1000, 1},
		{"kph to mph", "100 kph in mph", 62.137, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input + "\n")
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			interp := interpreter.NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Evaluation error: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			// Extract numeric value from result
			if qty, ok := results[0].(*types.Quantity); ok {
				actual, _ := qty.Value.Float64()
				if actual < tt.expectedVal-tt.tolerance || actual > tt.expectedVal+tt.tolerance {
					t.Errorf("Value = %f, expected ~%f ± %f", actual, tt.expectedVal, tt.tolerance)
				}
			} else {
				t.Errorf("Expected Quantity result, got %T", results[0])
			}
		})
	}
}
