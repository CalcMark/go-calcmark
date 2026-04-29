package interpreter

import (
	"strings"
	"testing"
	"time"

	"github.com/CalcMark/go-calcmark/spec/parser"
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
