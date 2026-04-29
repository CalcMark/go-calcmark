package interpreter

import (
	"strings"
	"testing"
	"time"

	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// TestEndOfExpr_VariableBoundDeferredR9 — R9 (variables-as-periods)
// is demoted to a future PR. Until the *types.Period value-type
// plumbing lands, `q = Q1; e = end of q` returns a clear runtime
// error directing users to use the literal directly.
func TestEndOfExpr_VariableBoundDeferredR9(t *testing.T) {
	interp := newTestInterpreterWithClock(time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC))
	src := "q = Q1\ne = end of q"
	nodes, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = interp.Eval(nodes)
	if err == nil {
		t.Fatal("expected runtime error from `e = end of q`; got nil")
	}
	if !strings.Contains(err.Error(), "variable-bound") &&
		!strings.Contains(err.Error(), "Period") &&
		!strings.Contains(err.Error(), "period") {
		t.Errorf("error %q should mention variable-bound or Period", err.Error())
	}
}

// TestEndOfExpr_NumberInnerRuntimeError — `end of 5` is rejected at
// runtime even when the type-checker is bypassed. The interpreter
// guard ensures correctness.
func TestEndOfExpr_NumberInnerRuntimeError(t *testing.T) {
	interp := newTestInterpreterWithClock(time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC))
	nodes, err := parser.Parse("x = end of 5")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := interp.Eval(nodes); err == nil {
		t.Fatal("expected runtime error from `end of 5`; got nil")
	}
}

// TestEndOfExpr_BareMonthName — regression for PR-1b. Pre-PR-1b,
// `end of April` worked through the old string-prefix dispatch. The
// new AST-based dispatch sees `*ast.DateLiteral` (parser produces
// that for bare month names with implicit Day=1, Year=nil) and
// initially rejected it. The fix accepts bare month names (no
// digits in SourceText) and routes them through the existing
// keyword-keyed evalEndOf path.
func TestEndOfExpr_BareMonthName(t *testing.T) {
	cases := []struct {
		name, input    string
		wantMonth, wantDay int
	}{
		{"end of April", "x = end of April", 4, 30},
		{"end of January", "x = end of January", 1, 31},
		{"end of February (non-leap)", "x = end of February", 2, 28},
		{"end of December", "x = end of December", 12, 31},
		{"end of Apr (abbrev)", "x = end of Apr", 4, 30},
		{"start of April", "x = start of April", 4, 1},
		{"start of February", "x = start of February", 2, 1},
	}
	// Use a non-leap year for predictable Feb behavior.
	clock := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(clock)
			nodes, err := parser.Parse(tc.input)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval(%q): %v", tc.input, err)
			}
			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}
			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("expected *types.Date, got %T", results[0])
			}
			if int(date.Time.Month()) != tc.wantMonth || date.Time.Day() != tc.wantDay {
				t.Errorf("got %s, want month=%d day=%d",
					date.Time.Format("2006-01-02"), tc.wantMonth, tc.wantDay)
			}
		})
	}
}
