package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/parser"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
)

// Rate × Duration / Rate × Rate arithmetic with time-unit cancellation
// and conversion. The motivating use case (user 2026-04-30):
//
//   cost = $100 / hour
//   work = 3 weeks
//   fee  = cost * work          # → $50,400 (3 weeks → 504 hours, × $100)
//
// Plus the chained form the user actually preferred:
//
//   rate    = $100 / hour
//   density = 40 hours / week
//   weeks   = 3 weeks
//   fee     = rate * density * weeks   # → $12,000
//
// And the simplest case that was also failing pre-change:
//
//   $100 / hour * 1 hour     # → $100  (units already match — no conversion)
//
// All three failed pre-change with "cannot multiply rate (...) and
// duration (...)". Tests are concrete value assertions to lock the
// numeric semantics.

func evalSingleResult(t *testing.T, src string) types.Type {
	t.Helper()
	interp := NewInterpreter()
	nodes, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("no result")
	}
	return results[len(results)-1]
}

func TestRateDuration_CurrencyRateMatchingUnit(t *testing.T) {
	// $100/hour × 1 hour → $100. No conversion needed; just multiply
	// values and drop the cancelled unit. Currency comes back out
	// because the rate's amount was a Currency at construction.
	res := evalSingleResult(t, "$100 / hour * 1 hour\n")
	cur, ok := res.(*types.Currency)
	if !ok {
		t.Fatalf("want *types.Currency, got %T (%v)", res, res)
	}
	if got := cur.Value.String(); got != "100" {
		t.Errorf("value = %s, want 100", got)
	}
}

func TestRateDuration_CurrencyRateMatchingUnitFiveHours(t *testing.T) {
	res := evalSingleResult(t, "$100 / hour * 5 hours\n")
	cur, ok := res.(*types.Currency)
	if !ok {
		t.Fatalf("want *types.Currency, got %T", res)
	}
	if got := cur.Value.String(); got != "500" {
		t.Errorf("value = %s, want 500", got)
	}
}

func TestRateDuration_CurrencyRateConvertedFromWeeks(t *testing.T) {
	// $100/hour × 3 weeks → 3 × 7 × 24 × $100 = $50,400.
	// The duration's unit (weeks) is converted to the rate's PerUnit
	// (hours) before multiplication.
	res := evalSingleResult(t, "$100 / hour * 3 weeks\n")
	cur, ok := res.(*types.Currency)
	if !ok {
		t.Fatalf("want *types.Currency, got %T", res)
	}
	if got := cur.Value.String(); got != "50400" {
		t.Errorf("value = %s, want 50400", got)
	}
}

func TestRateDuration_QuantityRateConstruction(t *testing.T) {
	// `40 hours / week` must construct a valid Rate with a
	// Duration-shaped numerator. Pre-change: rejected with "rate
	// amount must be a number, quantity, or currency, got
	// *types.Duration".
	res := evalSingleResult(t, "40 hours / week\n")
	rate, ok := res.(*types.Rate)
	if !ok {
		t.Fatalf("want *types.Rate, got %T (%v)", res, res)
	}
	if rate.PerUnit != "week" {
		t.Errorf("PerUnit = %q, want 'week'", rate.PerUnit)
	}
	if rate.Amount.Unit != "hour" {
		t.Errorf("Amount.Unit = %q, want 'hour' (canonical singular)",
			rate.Amount.Unit)
	}
	if got := rate.Amount.Value.String(); got != "40" {
		t.Errorf("Amount.Value = %s, want 40", got)
	}
}

func TestRateDuration_QuantityRateTimesDuration(t *testing.T) {
	// 40 hours/week × 3 weeks → 120 hours. Time unit on the rate's
	// numerator stays as the result's unit; the PerUnit cancels with
	// the multiplier's unit.
	res := evalSingleResult(t, "40 hours / week * 3 weeks\n")
	qty, ok := res.(*types.Quantity)
	if !ok {
		t.Fatalf("want *types.Quantity, got %T (%v)", res, res)
	}
	if qty.Unit != "hour" {
		t.Errorf("Unit = %q, want 'hour'", qty.Unit)
	}
	if got := qty.Value.String(); got != "120" {
		t.Errorf("Value = %s, want 120", got)
	}
}

func TestRateDuration_ChainedCancellation(t *testing.T) {
	// $100/hour × 40 hours/week × 3 weeks = $12,000.
	// Left-associative parse: ((rate × density) × weeks).
	//   ($100/hour × 40 hours/week) → cancel hour → $4,000/week
	//   ($4,000/week × 3 weeks)     → cancel week → $12,000 (Currency)
	res := evalSingleResult(t, "$100 / hour * 40 hours / week * 3 weeks\n")
	cur, ok := res.(*types.Currency)
	if !ok {
		t.Fatalf("want *types.Currency, got %T (%v)", res, res)
	}
	if got := cur.Value.String(); got != "12000" {
		t.Errorf("Value = %s, want 12000", got)
	}
}

func TestRateDuration_ChainedIntermediateRate(t *testing.T) {
	// Intermediate: ($100/hour × 40 hours/week) by itself is a Rate
	// of $4000/week — a useful expression in its own right (annual
	// salary at 40 hr/wk * 52 weeks).
	res := evalSingleResult(t, "$100 / hour * 40 hours / week\n")
	rate, ok := res.(*types.Rate)
	if !ok {
		t.Fatalf("want *types.Rate, got %T (%v)", res, res)
	}
	if rate.PerUnit != "week" {
		t.Errorf("PerUnit = %q, want 'week'", rate.PerUnit)
	}
	// $4000/week — Amount.Unit is the currency symbol "$" (the
	// existing convention from rate_eval.go for currency-numerator
	// rates). The numeric value is what we lock here.
	if got := rate.Amount.Value.String(); got != "4000" {
		t.Errorf("Amount.Value = %s, want 4000", got)
	}
}

// Out-of-scope for this round (documented for the follow-up):
//
//   - `3 weeks * $100 / hour` (duration on the left). Operator
//     precedence parses this as `((3 weeks) * $100) / hour` rather
//     than `(3 weeks) * ($100/hour)`, so it requires explicit
//     parentheses around the rate. The natural form is rate-on-left
//     and we lock that one above.
//   - `$100 / hour * 5 kg` should error with a dimensional-mismatch
//     diagnostic. Today the existing Rate × Quantity branch silently
//     coerces the result to `500 kg` — pre-existing behavior we
//     don't tighten in this PR to keep the soak window stable.
