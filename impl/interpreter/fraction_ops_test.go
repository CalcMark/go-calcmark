package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/shopspring/decimal"
)

func TestFractionArithmetic(t *testing.T) {
	tests := []struct {
		name     string
		left     types.Type
		right    types.Type
		operator string
		want     string
	}{
		// Fraction + Fraction
		{"1/3 + 1/3", frac(1, 3), frac(1, 3), "+", "2/3"},
		{"1/3 + 1/4", frac(1, 3), frac(1, 4), "+", "7/12"},
		{"1/3 + 1/3 via cascade", frac(2, 3), frac(1, 3), "+", "1"},

		// Fraction - Fraction
		{"2/3 - 1/3", frac(2, 3), frac(1, 3), "-", "1/3"},
		{"1/3 - 1/2", frac(1, 3), frac(1, 2), "-", "-1/6"},

		// Fraction * Fraction
		{"1/3 * 1/3", frac(1, 3), frac(1, 3), "*", "1/9"},
		{"2/3 * 3/4", frac(2, 3), frac(3, 4), "*", "1/2"},

		// Fraction / Fraction
		{"1/3 / 1/3", frac(1, 3), frac(1, 3), "/", "1"},
		{"1/2 / 1/4", frac(1, 2), frac(1, 4), "/", "2"},

		// Fraction + Number
		{"1/3 + 1", frac(1, 3), num(1), "+", "1 1/3"},
		{"1/3 + 2", frac(1, 3), num(2), "+", "2 1/3"},

		// Fraction * Number
		{"1/3 * 3", frac(1, 3), num(3), "*", "1"},
		{"2/3 * 6", frac(2, 3), num(6), "*", "4"},

		// Number + Fraction
		{"1 + 1/3", num(1), frac(1, 3), "+", "1 1/3"},

		// Number * Fraction
		{"3 * 1/3", num(3), frac(1, 3), "*", "1"},

		// Fraction with unit (mixed number desugaring: 12 + 1/2 pints)
		{"12 + 1/2 pints", num(12), fracUnit(1, 2, "pint"), "+", "12 1/2 pint"},
		{"1/2 pints * 3", fracUnit(1, 2, "pint"), num(3), "*", "1 1/2 pint"},
		{"2/3 cup + 1/4 cup", fracUnit(2, 3, "cup"), fracUnit(1, 4, "cup"), "+", "11/12 cup"},

		// Fraction * Currency → Currency (rounded by currency display)
		{"1/3 * $200", frac(1, 3), cur(200, "$"), "*", "$66.67"},

		// Fraction * Duration → Duration
		{"1/2 * 1 hour", frac(1, 2), dur(1, "hour"), "*", "0.5 hour"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evalBinaryOperation(tt.left, tt.right, tt.operator)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := result.String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFractionExponentiation(t *testing.T) {
	tests := []struct {
		name    string
		left    types.Type
		right   types.Type
		want    string
		wantErr bool
	}{
		{"(2/3)^3", frac(2, 3), num(3), "8/27", false},
		{"(1/2)^2", frac(1, 2), num(2), "1/4", false},
		{"(1/3)^0", frac(1, 3), num(0), "1", false},
		// Fractional exponent → decimal
		{"(1/4)^0.5", frac(1, 4), numf(0.5), "0.5", false},
		// Exponent > 100 → decimal fallback (DoS protection)
		{"(1/3)^200", frac(1, 3), num(200), "", false}, // just check no error/hang
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evalBinaryOperation(tt.left, tt.right, "^")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.want != "" {
				got := result.String()
				if got != tt.want {
					t.Errorf("got %q, want %q", got, tt.want)
				}
			}
		})
	}
}

func TestFractionUnary(t *testing.T) {
	f, _ := types.NewFraction(1, 3)
	result, err := evalUnaryOperation(f, "-")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := result.String()
	if got != "-1/3" {
		t.Errorf("got %q, want \"-1/3\"", got)
	}
}

func TestFractionComparison(t *testing.T) {
	tests := []struct {
		name  string
		left  types.Type
		right types.Type
		op    string
		want  bool
	}{
		{"1/3 < 1/2", frac(1, 3), frac(1, 2), "<", true},
		{"1/2 > 1/3", frac(1, 2), frac(1, 3), ">", true},
		{"1/3 == 1/3", frac(1, 3), frac(1, 3), "==", true},
		{"1/3 != 1/2", frac(1, 3), frac(1, 2), "!=", true},
		{"1/3 < 1 (number)", frac(1, 3), num(1), "<", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evalComparison(tt.left, tt.right, tt.op)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			b, ok := result.(*types.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", result)
			}
			if b.Value != tt.want {
				t.Errorf("got %v, want %v", b.Value, tt.want)
			}
		})
	}
}

func TestFractionSqrt(t *testing.T) {
	tests := []struct {
		name       string
		input      types.Type
		wantFrac   bool // true if we expect a Fraction result
		wantString string
	}{
		{"sqrt(1/4) = 1/2", frac(1, 4), true, "1/2"},
		{"sqrt(4/9) = 2/3", frac(4, 9), true, "2/3"},
		{"sqrt(1/3) decimal fallback", frac(1, 3), false, ""}, // just check no error
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evalSqrt([]types.Type{tt.input})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantFrac {
				f, ok := result.(*types.Fraction)
				if !ok {
					t.Fatalf("expected Fraction, got %T (%s)", result, result)
				}
				if f.String() != tt.wantString {
					t.Errorf("got %q, want %q", f.String(), tt.wantString)
				}
			}
		})
	}
}

func TestFractionNumber(t *testing.T) {
	f, _ := types.NewFraction(1, 3)
	result, err := ExtractNumber(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := result.Value.String()
	if got != "0.333333333333333" {
		t.Errorf("got %q, want \"0.333333333333333\"", got)
	}
}

// Helper functions for test readability
func frac(num, denom int64) *types.Fraction {
	f, _ := types.NewFraction(num, denom)
	return f
}

func fracUnit(num, denom int64, unit string) *types.Fraction {
	f, _ := types.NewFraction(num, denom)
	f.Unit = unit
	return f
}

func num(n int64) *types.Number {
	return types.NewNumber(decimal.NewFromInt(n))
}

func numf(f float64) *types.Number {
	return types.NewNumber(decimal.NewFromFloat(f))
}

func cur(val int64, sym string) *types.Currency {
	return types.NewCurrency(decimal.NewFromInt(val), sym)
}

func dur(val int64, unit string) *types.Duration {
	return &types.Duration{Value: decimal.NewFromInt(val), Unit: unit}
}
