package interpreter_test

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/shopspring/decimal"
)

// evalLine is a test helper that parses and evaluates a single expression.
func evalGrowthLine(t *testing.T, input string) string {
	t.Helper()
	nodes, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse(%q) error = %v", input, err)
	}
	interp := interpreter.NewInterpreter()
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval(%q) error = %v", input, err)
	}
	if len(results) == 0 {
		t.Fatalf("Eval(%q) returned no results", input)
	}
	return results[0].String()
}

// evalGrowthLineErr is a test helper that expects an evaluation error.
func evalGrowthLineErr(t *testing.T, input string) string {
	t.Helper()
	nodes, err := parser.Parse(input)
	if err != nil {
		return err.Error()
	}
	interp := interpreter.NewInterpreter()
	_, err = interp.Eval(nodes)
	if err == nil {
		t.Fatalf("Eval(%q) expected error, got nil", input)
	}
	return err.Error()
}

// TestCompoundGrowthMode1 tests basic compound growth: compound(principal, rate, periods)
func TestCompoundGrowthMode1(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "basic number",
			input: "compound(1000, 5%, 10)",
			want:  "1628.89",
		},
		{
			name:  "currency",
			input: "compound($1000, 5%, 10)",
			want:  "$1628.89",
		},
		{
			name:  "quantity",
			input: "compound(500 customers, 20%, 12)",
			want:  "4458.05 customers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evalGrowthLine(t, tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestCompoundGrowthMode3 tests financial compounding: compound(p, r, d, compounded:freq)
func TestCompoundGrowthMode3(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "compounded monthly",
			input: "compound($1000, 5%, 10 years, compounded monthly)",
			want:  "$1647.01",
		},
		{
			name:  "compounded quarterly",
			input: "compound($1000, 5%, 10 years, compounded quarterly)",
			want:  "$1643.62",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evalGrowthLine(t, tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestLinearGrow tests grow(amount, increment, periods)
func TestLinearGrow(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "basic number",
			input: "grow(100, 20, 5)",
			want:  "200",
		},
		{
			name:  "currency",
			input: "grow($500, $100, 36)",
			want:  "$4100.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evalGrowthLine(t, tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// TestDepreciate tests depreciate(value, rate, periods)
func TestDepreciate(t *testing.T) {
	got := evalGrowthLine(t, "depreciate($50000, 15%, 5)")
	if got != "$22185.27" {
		t.Errorf("got %q, want %q", got, "$22185.27")
	}
}

// TestDepreciateWithSalvage tests depreciate with salvage floor
func TestDepreciateWithSalvage(t *testing.T) {
	// After 10 years at 15%, value would be ~$9843.72 (above $5000 salvage)
	got := evalGrowthLine(t, "depreciate($50000, 15%, 10, $5000)")
	if got != "$9843.72" {
		t.Errorf("got %q, want %q", got, "$9843.72")
	}

	// Test where salvage floor IS applied (very long depreciation)
	got2 := evalGrowthLine(t, "depreciate($50000, 15%, 20, $5000)")
	if got2 != "$5000.00" {
		t.Errorf("got %q, want %q", got2, "$5000.00")
	}
}

// TestGrowthErrors tests error cases
func TestGrowthErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "compound rate too high",
			input:   "compound(1000, 150%, 10)",
			wantErr: "rate must be between -100% and 100%",
		},
		{
			name:    "compound rate too low",
			input:   "compound(1000, -150%, 10)",
			wantErr: "rate must be between -100% and 100%",
		},
		{
			name:    "depreciate negative rate",
			input:   "depreciate(1000, -10%, 5)",
			wantErr: "rate must be positive",
		},
		{
			name:    "compound too many periods",
			input:   "compound(1000, 5%, 20000)",
			wantErr: "too many periods",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := evalGrowthLineErr(t, tt.input)
			if !strings.Contains(errStr, tt.wantErr) {
				t.Errorf("error = %q, want error containing %q", errStr, tt.wantErr)
			}
		})
	}
}

// TestCompoundGrowthMath verifies the mathematical correctness of compound growth
func TestCompoundGrowthMath(t *testing.T) {
	one := decimal.NewFromInt(1)

	// compound(1000, 5%, 10) = 1000 * (1.05)^10 = 1628.894627...
	principal := decimal.NewFromInt(1000)
	rate := decimal.NewFromFloat(0.05)
	periods := int32(10)

	expected := principal.Mul(one.Add(rate).Pow(decimal.NewFromInt(int64(periods))))
	expectedRounded := expected.Round(2)

	if !expectedRounded.Equal(decimal.RequireFromString("1628.89")) {
		t.Errorf("math check: 1000*(1.05)^10 = %s, want 1628.89", expectedRounded)
	}
}

// TestCompoundNLParity tests NL syntax produces same results as functional
func TestCompoundNLParity(t *testing.T) {
	funcResult := evalGrowthLine(t, "compound($1000, 5%, 10)")
	nlResult := evalGrowthLine(t, "compound $1000 by 5% over 10")

	if funcResult != nlResult {
		t.Errorf("NL %q != functional %q", nlResult, funcResult)
	}
}
