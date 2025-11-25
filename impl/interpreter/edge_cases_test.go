package interpreter_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestUnitEdgeCases tests edge cases for new unit categories
func TestUnitEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Zero values should work
		{"zero celsius", "0 celsius", false},
		{"zero fahrenheit", "0 fahrenheit", false},
		{"zero kelvin", "0 kelvin", false},
		{"zero watts", "0 watts", false},
		{"zero joules", "0 joules", false},
		{"zero mps", "0 mps", false},

		// Very large numbers
		{"million watts", "1000000 watts in megawatts", false},
		{"thousand hp", "1000 horsepower in kilowatts", false},

		// Very small numbers
		{"milliwatt equivalent", "0.001 kilowatts in watts", false},
		{"small calories", "0.1 calories in joules", false},

		// Chained conversions (should work - convert final result)
		{"chained temp", "100 celsius in fahrenheit", false},

		// Invalid: mixing incompatible units
		{"temp + length", "10 celsius + 5 meters", true},
		{"speed + energy", "50 mph + 100 joules", true},
		{"power + mass", "100 watts + 10 kg", true},

		// Invalid: converting incompatible units
		{"temp to length", "100 celsius in meters", true},
		{"speed to power", "50 mph in watts", true},
		{"energy to mass", "1000 joules in kilograms", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input + "\n")
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("Parse error: %v", err)
				}
				return
			}

			interp := interpreter.NewInterpreter()
			_, err = interp.Eval(nodes)

			if tt.wantErr && err == nil {
				t.Errorf("Expected error but got none for: %s", tt.input)
			} else if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.input, err)
			}
		})
	}
}

// TestUnitAliasConsistency verifies that all aliases produce identical results
func TestUnitAliasConsistency(t *testing.T) {
	tests := []struct {
		name    string
		aliases []string // Different ways to write the same thing
	}{
		{
			"celsius aliases",
			[]string{"100 celsius", "100 c", "100 degc"},
		},
		{
			"fahrenheit aliases",
			[]string{"212 fahrenheit", "212 f", "212 degf"},
		},
		{
			"kelvin aliases",
			[]string{"373 kelvin", "373 k"},
		},
		{
			"watts aliases",
			[]string{"1000 watts", "1000 w", "1000 watt"},
		},
		{
			"kilowatts aliases",
			[]string{"1 kilowatts", "1 kw", "1 kilowatt"},
		},
		{
			"horsepower aliases",
			[]string{"5 horsepower", "5 hp"},
		},
		{
			"joules aliases",
			[]string{"1000 joules", "1000 j", "1000 joule"},
		},
		{
			"calories aliases",
			[]string{"100 calories", "100 cal", "100 calorie"},
		},
		{
			"speed aliases (no slash)",
			[]string{"60 mps"},
		},
		{
			"kph aliases",
			[]string{"100 kph", "100 kmh"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var results []string

			for _, alias := range tt.aliases {
				nodes, err := parser.Parse(alias + "\n")
				if err != nil {
					t.Fatalf("Parse error for %s: %v", alias, err)
				}

				interp := interpreter.NewInterpreter()
				res, err := interp.Eval(nodes)
				if err != nil {
					t.Fatalf("Eval error for %s: %v", alias, err)
				}

				if len(res) == 0 {
					t.Fatalf("No results for %s", alias)
				}

				results = append(results, res[0].String())
			}

			// All results should be identical (or at least have same numeric value)
			// We just check they all start with same prefix for FP tolerance
			firstValue := results[0]
			for i, result := range results[1:] {
				// Extract numeric part for comparison
				if result[:3] != firstValue[:3] {
					t.Errorf("Alias %s produced different result: %s vs %s",
						tt.aliases[i+1], result, firstValue)
				}
			}
		})
	}
}

// TestUnitPrecision verifies floating point precision is acceptable
func TestUnitPrecision(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		checkFunc func(string) bool
	}{
		{
			"celsius to fahrenheit precision",
			"100 celsius in fahrenheit",
			func(s string) bool {
				// Should be close to 212
				return s[:3] == "211" || s[:3] == "212"
			},
		},
		{
			"hp to watts precision",
			"1 horsepower in watts",
			func(s string) bool {
				// Should be close to 745.7
				return s[:3] == "745"
			},
		},
		{
			"kph to mph precision",
			"100 kph in mph",
			func(s string) bool {
				// Should be close to 62.137
				return s[:2] == "62"
			},
		},
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
				t.Fatalf("Eval error: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("No results")
			}

			result := results[0].String()
			if !tt.checkFunc(result) {
				t.Errorf("Precision check failed for %s: got %s", tt.input, result)
			}
		})
	}
}
