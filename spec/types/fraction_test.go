package types

import (
	"math/big"
	"testing"
)

func TestNewFraction(t *testing.T) {
	tests := []struct {
		name    string
		num     int64
		denom   int64
		wantErr bool
	}{
		{"simple", 1, 3, false},
		{"zero numerator", 0, 3, false},
		{"negative numerator", -1, 3, false},
		{"negative denominator", 1, -3, false},
		{"zero denominator", 1, 0, true},
		{"both negative", -1, -3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := NewFraction(tt.num, tt.denom)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if f == nil {
				t.Fatal("expected non-nil Fraction")
			}
		})
	}
}

func TestFractionSimplification(t *testing.T) {
	tests := []struct {
		name      string
		num       int64
		denom     int64
		wantNum   int64
		wantDenom int64
	}{
		{"already simplified", 1, 3, 1, 3},
		{"2/4 → 1/2", 2, 4, 1, 2},
		{"6/9 → 2/3", 6, 9, 2, 3},
		{"10/5 → 2/1", 10, 5, 2, 1},
		{"0/5 → 0/1", 0, 5, 0, 1},
		{"-2/4 → -1/2", -2, 4, -1, 2},
		{"2/-4 → -1/2", 2, -4, -1, 2},
		{"-2/-4 → 1/2", -2, -4, 1, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := NewFraction(tt.num, tt.denom)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			gotNum := f.Num().Int64()
			gotDenom := f.Denom().Int64()
			if gotNum != tt.wantNum || gotDenom != tt.wantDenom {
				t.Errorf("got %d/%d, want %d/%d", gotNum, gotDenom, tt.wantNum, tt.wantDenom)
			}
		})
	}
}

func TestFractionString(t *testing.T) {
	tests := []struct {
		name     string
		num      int64
		denom    int64
		isNapkin bool
		unit     string
		want     string
	}{
		{"simple fraction", 1, 3, false, "", "1/3"},
		{"simplified", 2, 4, false, "", "1/2"},
		{"mixed number", 7, 3, false, "", "2 1/3"},
		{"integer result", 6, 3, false, "", "2"},
		{"zero", 0, 1, false, "", "0"},
		{"negative fraction", -1, 3, false, "", "-1/3"},
		{"negative mixed number", -7, 3, false, "", "-2 1/3"},
		{"with unit", 1, 3, false, "cup", "1/3 cup"},
		{"napkin prefix", 2, 3, true, "", "~2/3"},
		{"napkin with unit", 1, 2, true, "cup", "~1/2 cup"},
		{"improper negative", -5, 2, false, "", "-2 1/2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := NewFraction(tt.num, tt.denom)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			f.IsNapkin = tt.isNapkin
			f.Unit = tt.unit
			got := f.String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFractionToDecimal(t *testing.T) {
	tests := []struct {
		name  string
		num   int64
		denom int64
		want  string
	}{
		{"1/3", 1, 3, "0.333333333333333"},
		{"1/2", 1, 2, "0.5"},
		{"2/1", 2, 1, "2"},
		{"1/4", 1, 4, "0.25"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := NewFraction(tt.num, tt.denom)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := f.ToDecimal().String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFractionIsProper(t *testing.T) {
	tests := []struct {
		name  string
		num   int64
		denom int64
		want  bool
	}{
		{"proper", 1, 3, true},
		{"improper", 7, 3, false},
		{"equal", 3, 3, false},
		{"negative proper", -1, 3, true},
		{"negative improper", -7, 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := NewFraction(tt.num, tt.denom)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := f.IsProper()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFractionExceedsComputationLimit(t *testing.T) {
	tests := []struct {
		name string
		rat  *big.Rat
		want bool
	}{
		{"small fraction", new(big.Rat).SetFrac64(1, 3), false},
		{"large denominator", new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(10), nil)), true},
		{"large numerator", new(big.Rat).SetFrac(new(big.Int).Exp(big.NewInt(10), big.NewInt(19), nil), big.NewInt(1)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFractionFromRat(tt.rat)
			got := f.ExceedsComputationLimit()
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFractionLargeDenominatorFallsBackToDecimal(t *testing.T) {
	// Denominator > 1000 should display as decimal
	f, err := NewFraction(1, 1001)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := f.String()
	// Should be a decimal representation, not "1/1001"
	if got == "1/1001" {
		t.Errorf("expected decimal fallback for denominator > 1000, got %q", got)
	}
}
