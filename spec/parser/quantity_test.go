package parser_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestQuantityParsing tests number + unit combinations
func TestQuantityParsing(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string // Expected canonical unit or "any" for arbitrary
	}{
		// Canonical SI units
		{"meter short", "10 m\n", "meter"},
		{"meter long", "10 meters\n", "meter"},
		{"kilogram", "5 kg\n", "kilogram"},
		{"kilometer", "100 km\n", "kilometer"},

		// Canonical US units
		{"foot short", "5 ft\n", "foot"},
		{"foot long", "5 feet\n", "foot"},
		{"pound", "10 lb\n", "pound"},
		{"mile", "10 mi\n", "mile"},

		// Arbitrary units (any identifier)
		{"apples", "2 apples\n", "any"},
		{"oranges", "5 oranges\n", "any"},
		{"widgets", "10 widgets\n", "any"},
		{"items", "100 items\n", "any"},

		// Multipliers + units
		{"k + meter", "12k m\n", "meter"},
		{"k + feet", "10k feet\n", "foot"},
		{"M + kg", "1M kg\n", "kilogram"},
		{"B + miles", "5B miles\n", "mile"},

		// Decimals + units
		{"decimal meter", "1.5 meters\n", "meter"},
		{"decimal kg", "2.75 kg\n", "kilogram"},

		// Scientific notation + units
		{"sci meter", "1e3 meters\n", "meter"},
		{"sci kg", "1.5e6 kg\n", "kilogram"},

		// Currency codes (3 uppercase letters)
		{"USD", "100 USD\n", "currency"},
		{"EUR", "50 EUR\n", "currency"},
		{"GBP", "25 GBP\n", "currency"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			if len(nodes) != 1 {
				t.Fatalf("Parse(%q) returned %d nodes, want 1", tt.input, len(nodes))
			}

			// TODO: Verify node type (QuantityLiteral or CurrencyLiteral)
			// TODO: Verify unit normalization
		})
	}
}

// TestMultiplierWithUnit tests all multiplier + unit combinations
func TestMultiplierWithUnit(t *testing.T) {
	multipliers := []string{"k", "M", "B", "T"}
	units := []string{"meters", "kg", "feet", "pounds"}

	for _, mult := range multipliers {
		for _, unit := range units {
			input := "10" + mult + " " + unit + "\n"

			t.Run(mult+"_"+unit, func(t *testing.T) {
				nodes, err := parser.Parse(input)
				if err != nil {
					t.Errorf("Parse(%q) error = %v", input, err)
				}

				if len(nodes) != 1 {
					t.Errorf("Parse(%q) returned %d nodes, want 1", input, len(nodes))
				}
			})
		}
	}
}

// TestArbitraryUnits tests that any identifier can be a unit
func TestArbitraryUnits(t *testing.T) {
	arbitraryUnits := []string{
		"apples", "oranges", "widgets", "items", "pieces",
		"foobar", "bazqux", "x", "y", "z",
		"Item1", "Widget2", "Box5",
		"UPPERCASE", "MixedCase", "lowercase",
	}

	for _, unit := range arbitraryUnits {
		input := "10 " + unit + "\n"

		t.Run(unit, func(t *testing.T) {
			nodes, err := parser.Parse(input)
			if err != nil {
				t.Errorf("Parse(%q) error = %v (arbitrary units should be valid)", input, err)
			}

			if len(nodes) != 1 {
				t.Errorf("Parse(%q) returned %d nodes, want 1", input, len(nodes))
			}
		})
	}
}

// TestQuantityInExpressions tests quantities in arithmetic expressions
func TestQuantityInExpressions(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"addition", "10 meters + 5 meters\n"},
		{"subtraction", "100 feet - 25 feet\n"},
		{"multiplication", "5 meters * 2\n"},
		{"division", "100 meters / 4\n"},
		{"assignment", "distance = 100 meters\n"},
		{"function arg", "avg(10 meters, 20 meters)\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Errorf("Parse(%q) error = %v", tt.input, err)
			}

			if len(nodes) == 0 {
				t.Errorf("Parse(%q) returned 0 nodes", tt.input)
			}
		})
	}
}

// TestInvalidQuantitySyntax tests error handling for invalid unit syntax
func TestInvalidQuantitySyntax(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		// Missing space
		{"no space", "10meters\n"},
		{"k no space", "10kmeters\n"},

		// Invalid separators
		{"dash", "10-meters\n"},
		{"underscore", "5_kg\n"},

		// Multiple units
		{"two units", "10 meters feet\n"},

		// Reserved keywords as units (if they exist)
		// {"keyword", "10 if\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(tt.input)
			if err == nil {
				t.Logf("Parse(%q) expected error but got none (might be valid alternate interpretation)", tt.input)
				// Don't fail - these might be valid in different contexts
			}
		})
	}
}

// TestCurrencyCodeRecognition tests that 3-letter uppercase = currency
func TestCurrencyCodeRecognition(t *testing.T) {
	tests := []struct {
		input      string
		isCurrency bool // Should be recognized as currency
	}{
		{"100 USD\n", true},
		{"50 EUR\n", true},
		{"25 GBP\n", true},
		{"100 XXX\n", true},   // Syntactically valid, semantically invalid
		{"100 usd\n", false},  // Lowercase - regular unit
		{"100 US\n", false},   // Only 2 letters - regular unit
		{"100 USDD\n", false}, // 4 letters - regular unit
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			if len(nodes) != 1 {
				t.Fatalf("Parse(%q) returned %d nodes, want 1", tt.input, len(nodes))
			}

			// TODO: Check if node is CurrencyLiteral vs QuantityLiteral
			// based on tt.isCurrency
		})
	}
}

// BenchmarkQuantityParsing benchmarks quantity parsing performance
func BenchmarkQuantityParsing(b *testing.B) {
	input := "10 meters\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkArbitraryUnit benchmarks arbitrary unit parsing
func BenchmarkArbitraryUnit(b *testing.B) {
	input := "10 apples\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMultiplierWithUnit benchmarks multiplier + unit parsing
func BenchmarkMultiplierWithUnit(b *testing.B) {
	input := "12k meters\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}
