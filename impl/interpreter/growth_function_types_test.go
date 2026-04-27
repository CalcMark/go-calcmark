package interpreter_test

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// Intent of each growth function's parameter types — captured here so
// the runtime contract is documented next to the tests that pin it.
//
// === grow(amount, increment, periods) ===
// Linear additive growth: amount + (increment × periods).
//   amount     Additive — Number, Quantity, or Currency. The starting
//              value. Percentage is meaningless here ("start at 5%"
//              has no semantic) and silently coercing it to a decimal
//              produces nonsense.
//   increment  Additive matching amount's type. Added once per
//              period. Must be the same family as amount so addition
//              is well-defined. Percentage is rejected for the same
//              reason as amount.
//   periods    Number — a count of iterations. NOT a duration.
//              `5 months` was previously accepted (treated as 5),
//              but the unit is decorative and misleading: the
//              runtime is counting ITERATIONS, not time. Reject it
//              so users write what they mean.
//
// === compound(principal, rate, periods, period?) ===
// Compound multiplicative growth: principal × (1 + rate)^periods.
//   principal  Additive — same family as grow.amount.
//   rate       Percentage — the per-period growth rate. A bare
//              number is also accepted (5 == 5%) for legacy reasons
//              but this is fragile; clients should write `5%`.
//   periods    Number — iteration count, same rules as grow.
//   period     Optional String — compounding frequency identifier
//              (`monthly`, `quarterly`, `yearly`, `daily`). Not a
//              type-validated runtime arg; parser-level concern.
//
// === depreciate(value, rate, periods, salvage?) ===
// Declining-balance depreciation: value × (1 - rate)^periods, floored
// at salvage when supplied.
//   value     Additive — starting value.
//   rate      Percentage — per-period decline rate.
//   periods   Number — iteration count.
//   salvage   Optional Additive matching value's type.
//
// Tests below pin the "should reject" half of each contract. The
// happy-path tests live in growth_functions_test.go.

func evalErr(t *testing.T, input string) string {
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

// ----------------------------------------------------------------
// grow — type rejections
// ----------------------------------------------------------------

func TestGrow_RejectsPercentageAsIncrement(t *testing.T) {
	got := evalErr(t, "grow($100, 5%, 5)")
	if !strings.Contains(strings.ToLower(got), "increment") {
		t.Errorf("expected error to mention 'increment'; got %q", got)
	}
}

func TestGrow_RejectsPercentageAsAmount(t *testing.T) {
	got := evalErr(t, "grow(5%, 10, 5)")
	if !strings.Contains(strings.ToLower(got), "amount") {
		t.Errorf("expected error to mention 'amount'; got %q", got)
	}
}

func TestGrow_RejectsDurationAsPeriods(t *testing.T) {
	got := evalErr(t, "grow($100, $20, 5 months)")
	lower := strings.ToLower(got)
	if !strings.Contains(lower, "period") {
		t.Errorf("expected error to mention 'period'; got %q", got)
	}
}

func TestGrow_RejectsDurationAsPeriodsViaNLForm(t *testing.T) {
	// Same constraint over the NL alias.
	got := evalErr(t, "grow $100 by $20 over 5 hours")
	lower := strings.ToLower(got)
	if !strings.Contains(lower, "period") {
		t.Errorf("expected error to mention 'period'; got %q", got)
	}
}

func TestGrow_RejectsRateAsIncrement(t *testing.T) {
	got := evalErr(t, "grow(100, 10 MB/s, 5)")
	if !strings.Contains(strings.ToLower(got), "increment") {
		t.Errorf("expected error to mention 'increment'; got %q", got)
	}
}

// ----------------------------------------------------------------
// compound — type rejections
// ----------------------------------------------------------------

func TestCompound_RejectsDurationAsPeriods(t *testing.T) {
	got := evalErr(t, "compound($1000, 5%, 10 years)")
	lower := strings.ToLower(got)
	if !strings.Contains(lower, "period") {
		t.Errorf("expected error to mention 'period'; got %q", got)
	}
}

func TestCompound_RejectsCurrencyAsRate(t *testing.T) {
	// $20 isn't a percentage. Compound's middle param must be a rate.
	got := evalErr(t, "compound($1000, $20, 10)")
	if !strings.Contains(strings.ToLower(got), "rate") {
		t.Errorf("expected error to mention 'rate'; got %q", got)
	}
}

// ----------------------------------------------------------------
// depreciate — type rejections
// ----------------------------------------------------------------

func TestDepreciate_RejectsDurationAsPeriods(t *testing.T) {
	got := evalErr(t, "depreciate($50000, 15%, 5 years)")
	lower := strings.ToLower(got)
	if !strings.Contains(lower, "period") {
		t.Errorf("expected error to mention 'period'; got %q", got)
	}
}

func TestDepreciate_RejectsCurrencyAsRate(t *testing.T) {
	got := evalErr(t, "depreciate($50000, $200, 5)")
	if !strings.Contains(strings.ToLower(got), "rate") {
		t.Errorf("expected error to mention 'rate'; got %q", got)
	}
}

func TestDepreciate_RejectsPercentageAsValue(t *testing.T) {
	got := evalErr(t, "depreciate(20%, 15%, 5)")
	if !strings.Contains(strings.ToLower(got), "value") {
		t.Errorf("expected error to mention 'value'; got %q", got)
	}
}
