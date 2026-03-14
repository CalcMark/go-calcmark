package interpreter

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// TestSameTypeMulDivErrors verifies that multiplying or dividing same-type
// values (Currency*Currency, Currency/Currency, Quantity*Quantity, Quantity/Quantity,
// Fraction-with-unit * or / Fraction-with-unit) produces a helpful error
// suggesting number() as a workaround.
func TestSameTypeMulDivErrors(t *testing.T) {
	tests := []struct {
		name        string
		left        types.Type
		right       types.Type
		operator    string
		wantErr     bool
		errContains string // substring the error message must contain
	}{
		// Currency * Currency → error
		{
			name:        "Currency * Currency errors",
			left:        cur(100, "$"),
			right:       cur(50, "$"),
			operator:    "*",
			wantErr:     true,
			errContains: "number()",
		},
		// Currency / Currency → error
		{
			name:        "Currency / Currency errors",
			left:        cur(500, "$"),
			right:       cur(1000, "$"),
			operator:    "/",
			wantErr:     true,
			errContains: "number()",
		},
		// Quantity * Quantity → error with number() hint
		{
			name:        "Quantity * Quantity errors with number() hint",
			left:        qty(10, "kg"),
			right:       qty(5, "kg"),
			operator:    "*",
			wantErr:     true,
			errContains: "number()",
		},
		// Quantity / Quantity → error with number() hint
		{
			name:        "Quantity / Quantity errors with number() hint",
			left:        qty(10, "dogs"),
			right:       qty(5, "dogs"),
			operator:    "/",
			wantErr:     true,
			errContains: "number()",
		},
		// Fraction-with-unit * Fraction-with-unit → error
		{
			name:        "Fraction-with-unit * Fraction-with-unit errors",
			left:        fracUnit(2, 3, "cup"),
			right:       fracUnit(1, 3, "cup"),
			operator:    "*",
			wantErr:     true,
			errContains: "number()",
		},
		// Fraction-with-unit / Fraction-with-unit → error
		{
			name:        "Fraction-with-unit / Fraction-with-unit errors",
			left:        fracUnit(2, 3, "cup"),
			right:       fracUnit(1, 3, "cup"),
			operator:    "/",
			wantErr:     true,
			errContains: "number()",
		},
		// --- These should STILL work (no error) ---
		// Currency + Currency → ok
		{
			name:     "Currency + Currency still works",
			left:     cur(100, "$"),
			right:    cur(50, "$"),
			operator: "+",
			wantErr:  false,
		},
		// Currency - Currency → ok
		{
			name:     "Currency - Currency still works",
			left:     cur(100, "$"),
			right:    cur(50, "$"),
			operator: "-",
			wantErr:  false,
		},
		// Currency * Number → ok
		{
			name:     "Currency * Number still works",
			left:     cur(100, "$"),
			right:    num(5),
			operator: "*",
			wantErr:  false,
		},
		// Currency / Number → ok
		{
			name:     "Currency / Number still works",
			left:     cur(100, "$"),
			right:    num(5),
			operator: "/",
			wantErr:  false,
		},
		// Quantity * Number → ok
		{
			name:     "Quantity * Number still works",
			left:     qty(10, "kg"),
			right:    num(3),
			operator: "*",
			wantErr:  false,
		},
		// Quantity / Number → ok
		{
			name:     "Quantity / Number still works",
			left:     qty(10, "kg"),
			right:    num(2),
			operator: "/",
			wantErr:  false,
		},
		// Quantity + Quantity → ok
		{
			name:     "Quantity + Quantity still works",
			left:     qty(10, "kg"),
			right:    qty(5, "kg"),
			operator: "+",
			wantErr:  false,
		},
		// Fraction-with-unit + Fraction-with-unit → ok
		{
			name:     "Fraction-with-unit + Fraction-with-unit still works",
			left:     fracUnit(2, 3, "cup"),
			right:    fracUnit(1, 4, "cup"),
			operator: "+",
			wantErr:  false,
		},
		// Dimensionless Fraction * Currency → ok
		{
			name:     "Dimensionless Fraction * Currency still works",
			left:     frac(1, 3),
			right:    cur(200, "$"),
			operator: "*",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evalBinaryOperation(tt.left, tt.right, tt.operator)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got result: %v", result)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestAsPercentOf verifies "X as % of Y" produces correct Percentage results
// and proper errors for type mismatches.
func TestAsPercentOf(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantValue   string // expected String() output
		wantErr     bool
		errContains string
	}{
		// Same-type Currency → Percentage
		{"$100 as % of $500", "x = $100 as % of $500\n", "20%", false, ""},
		{"$152400 as % of $780000", "x = $152400 as % of $780000\n", "19.53846153846154%", false, ""},

		// Same-type Number → Percentage
		{"30 as % of 100", "x = 30 as % of 100\n", "30%", false, ""},

		// Same-type Quantity → Percentage
		{"10 kg as % of 50 kg", "x = 10 kg as % of 50 kg\n", "20%", false, ""},

		// Duration cross-unit → Percentage (normalized to seconds)
		{"3 hours as % of 1 day", "x = 3 hours as % of 1 day\n", "12.5%", false, ""},

		// Error: Currency vs Quantity
		{"currency vs quantity", "x = $100 as % of 50 kg\n", "", true, "both values must be the same type"},

		// Error: Different currencies
		{"$ vs €", "x = $100 as % of €50\n", "", true, "both values must be the same type"},

		// Error: Different quantity units
		{"kg vs meters", "x = 10 kg as % of 5 meters\n", "", true, "both values must be the same type"},

		// Division by zero
		{"div by zero", "x = $100 as % of $0\n", "", true, "division by zero"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			interp := NewInterpreter()
			results, err := interp.Eval(nodes)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got results: %v", results)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}
			got := results[0].String()
			if got != tt.wantValue {
				t.Errorf("got %q, want %q", got, tt.wantValue)
			}
		})
	}
}

// TestFractionUnitConversion verifies that fractions with units can be
// converted using the "in" keyword (e.g., "1/2 hour in minutes").
func TestFractionUnitConversion(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValue string
	}{
		{"1/2 hour in minutes", "x = 1/2 hour in minutes\n", "30 minutes"},
		{"1/4 cup in ml", "x = 1/4 cup in ml\n", "60 ml"},
		{"3/4 lb in oz", "x = 3/4 lb in oz\n", "12.000000000000002 oz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			interp := NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}
			got := results[0].String()
			if got != tt.wantValue {
				t.Errorf("got %q, want %q", got, tt.wantValue)
			}
		})
	}
}

func qty(val int64, unit string) *types.Quantity {
	return &types.Quantity{Value: decimal.NewFromInt(val), Unit: unit}
}
