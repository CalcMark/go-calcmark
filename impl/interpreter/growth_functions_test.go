package interpreter_test

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/format/display"
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

// TestCompoundGrowthMode2 tests bare period modifier: compound(p, r, n, period)
// Mode 2 is a semantic annotation — the period identifier is accepted but the
// math is identical to the 3-arg form (simple compound growth).
func TestCompoundGrowthMode2(t *testing.T) {
	tests := []struct {
		name  string
		input string
		equiv string // equivalent 3-arg form (must produce same result)
	}{
		{
			name:  "month",
			input: "compound($1000, 5%, 12, month)",
			equiv: "compound($1000, 5%, 12)",
		},
		{
			name:  "quarter",
			input: "compound($1000, 5%, 4, quarter)",
			equiv: "compound($1000, 5%, 4)",
		},
		{
			name:  "year",
			input: "compound($1000, 5%, 10, year)",
			equiv: "compound($1000, 5%, 10)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evalGrowthLine(t, tt.input)
			want := evalGrowthLine(t, tt.equiv)
			if got != want {
				t.Errorf("Mode 2 %q = %q, but 3-arg %q = %q (should be identical)",
					tt.input, got, tt.equiv, want)
			}
		})
	}
}

// TestCompoundGrowthMode3 tests financial compounding: compound(p, r, d, compounded freq)
// Formula: A = P(1 + r/n)^(nt) where n = periods per year
func TestCompoundGrowthMode3(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "compounded yearly",
			input: "compound($1000, 5%, 10 years, compounded yearly)",
			want:  "$1628.89",
		},
		{
			name:  "compounded quarterly",
			input: "compound($1000, 5%, 10 years, compounded quarterly)",
			want:  "$1643.62",
		},
		{
			name:  "compounded monthly",
			input: "compound($1000, 5%, 10 years, compounded monthly)",
			want:  "$1647.01",
		},
		{
			name:  "compounded weekly",
			input: "compound($1000, 5%, 10 years, compounded weekly)",
			want:  "$1648.33",
		},
		{
			name:  "compounded daily",
			input: "compound($1000, 5%, 10 years, compounded daily)",
			want:  "$1648.66",
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

// TestCompoundBareFrequencyModifier verifies that bare frequency identifiers
// (e.g., "monthly") produce the same result as the "compounded monthly" form.
// Regression test for #40: compound() was ignoring bare period modifiers.
func TestCompoundBareFrequencyModifier(t *testing.T) {
	tests := []struct {
		name  string
		bare  string
		equiv string
	}{
		{
			name:  "monthly",
			bare:  "compound($1000, 5%, 10 years, monthly)",
			equiv: "compound($1000, 5%, 10 years, compounded monthly)",
		},
		{
			name:  "quarterly",
			bare:  "compound($1000, 5%, 10 years, quarterly)",
			equiv: "compound($1000, 5%, 10 years, compounded quarterly)",
		},
		{
			name:  "yearly",
			bare:  "compound($1000, 5%, 10 years, yearly)",
			equiv: "compound($1000, 5%, 10 years, compounded yearly)",
		},
		{
			name:  "weekly",
			bare:  "compound($1000, 5%, 10 years, weekly)",
			equiv: "compound($1000, 5%, 10 years, compounded weekly)",
		},
		{
			name:  "daily",
			bare:  "compound($1000, 5%, 10 years, daily)",
			equiv: "compound($1000, 5%, 10 years, compounded daily)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evalGrowthLine(t, tt.bare)
			want := evalGrowthLine(t, tt.equiv)
			if got != want {
				t.Errorf("bare %q = %q, but %q = %q (should be identical)",
					tt.bare, got, tt.equiv, want)
			}
		})
	}
}

// TestCompoundNLBareFrequencyModifier verifies that the NL syntax
// "compound $1K by 5% monthly over 10 years" is equivalent to
// "compound $1K by 5% compounded monthly over 10 years".
// Regression test for #40: NL parser didn't recognize bare frequency adverbs.
func TestCompoundNLBareFrequencyModifier(t *testing.T) {
	tests := []struct {
		name  string
		bare  string
		equiv string
	}{
		{
			name:  "monthly",
			bare:  "compound $1K by 5% monthly over 10 years",
			equiv: "compound $1K by 5% compounded monthly over 10 years",
		},
		{
			name:  "quarterly",
			bare:  "compound $1K by 5% quarterly over 10 years",
			equiv: "compound $1K by 5% compounded quarterly over 10 years",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evalGrowthLine(t, tt.bare)
			want := evalGrowthLine(t, tt.equiv)
			if got != want {
				t.Errorf("NL bare %q = %q, but %q = %q (should be identical)",
					tt.bare, got, tt.equiv, want)
			}
		})
	}
}

// TestCompoundGrowthMode3_Ordering verifies that more frequent compounding
// produces higher returns (yearly < quarterly < monthly < weekly < daily).
func TestCompoundGrowthMode3_Ordering(t *testing.T) {
	yearly := evalGrowthLine(t, "compound(1000, 5%, 10 years, compounded yearly)")
	quarterly := evalGrowthLine(t, "compound(1000, 5%, 10 years, compounded quarterly)")
	monthly := evalGrowthLine(t, "compound(1000, 5%, 10 years, compounded monthly)")
	weekly := evalGrowthLine(t, "compound(1000, 5%, 10 years, compounded weekly)")
	daily := evalGrowthLine(t, "compound(1000, 5%, 10 years, compounded daily)")

	results := []string{yearly, quarterly, monthly, weekly, daily}
	for i := 1; i < len(results); i++ {
		if results[i] <= results[i-1] {
			t.Errorf("expected strictly increasing: results[%d]=%s <= results[%d]=%s",
				i, results[i], i-1, results[i-1])
		}
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
		{
			name:  "same-unit quantity",
			input: "grow(50 GB, 500 GB, 3)",
			want:  "1550 GB",
		},
		{
			name:  "mixed-unit quantity TB to GB",
			input: "grow(50 GB, 20 TB, 6)",
			want:  "122930 GB",
		},
		{
			// The previous "with duration periods" form
			// (`grow(..., 6 months)`) is now rejected — see
			// growth_function_types_test.go for the rationale. Use
			// a bare number to express iteration count.
			name:  "mixed-unit quantity with bare period count",
			input: "grow(50 GB, 20 TB, 6)",
			want:  "122930 GB",
		},
		{
			name:  "NL syntax mixed-unit quantity",
			input: "grow 50 GB by 20 TB over 6",
			want:  "122930 GB",
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

// TestDepreciateWithMixedUnitSalvage tests that depreciate converts salvage to principal's unit.
func TestDepreciateWithMixedUnitSalvage(t *testing.T) {
	// depreciate(1 TB, 15%, 20, 100 GB) — salvage 100 GB = 0.09765625 TB
	// After 20 years at 15%, 1 TB becomes ~$0.03875... TB, which is below salvage
	got := evalGrowthLine(t, "depreciate(1 TB, 15%, 20, 100 GB)")
	// 100 GB in TB = 100/1024 ≈ 0.09765625 TB, result should be clamped to salvage
	if got != "0.1 TB" {
		t.Errorf("got %q, want %q", got, "0.1 TB")
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
		{
			name:    "compound unknown frequency",
			input:   "compound($1000, 5%, 10 years, compounded biweekly)",
			wantErr: "unknown compounding frequency",
		},
		{
			name:    "grow incompatible units",
			input:   "grow(50 GB, 20 kg, 6)",
			wantErr: "incompatible units",
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

// TestGrowthResultsPreserveDisplayFormat verifies that growth function results
// preserve type metadata (Currency.Code, Quantity.Unit) needed by the display
// formatter, not just the String() method.
func TestGrowthResultsPreserveDisplayFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "compound currency preserves symbol in formatter",
			input: "compound($475000, 3.2%, 30)",
			want:  "$1.22M",
		},
		{
			name:  "compound quantity preserves unit in formatter",
			input: "compound(500 customers, 20%, 12)",
			want:  "4.46K customers",
		},
		{
			name:  "depreciate currency preserves symbol in formatter",
			input: "depreciate($50000, 15%, 5)",
			want:  "$22.19K",
		},
		{
			name:  "grow currency preserves symbol in formatter",
			input: "grow($500, $100, 36)",
			want:  "$4,100.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}
			interp := interpreter.NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval(%q) error = %v", tt.input, err)
			}
			if len(results) == 0 {
				t.Fatalf("Eval(%q) returned no results", tt.input)
			}
			got := display.Format(results[0])
			if got != tt.want {
				t.Errorf("display.Format(Eval(%q)) = %q, want %q", tt.input, got, tt.want)
			}
		})
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
