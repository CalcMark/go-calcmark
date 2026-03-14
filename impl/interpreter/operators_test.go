package interpreter

import (
	"strings"
	"testing"

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

func qty(val int64, unit string) *types.Quantity {
	return &types.Quantity{Value: decimal.NewFromInt(val), Unit: unit}
}
