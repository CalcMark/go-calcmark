package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/parser"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/shopspring/decimal"
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

// AE3 (engine-level) — same-dimension Custom-unit cancellation. The
// language-level form `100 cakes / box * 5 boxes` doesn't parse
// today because `box` isn't recognised as a rate denominator at
// the lexer/parser level (it's treated as a variable lookup and
// fails). The cancellation engine itself is correct — this test
// constructs the Rate programmatically and verifies the engine
// handles same-string Custom cancellation. Parser-level support
// for arbitrary Custom-unit rate denominators is tracked as a
// follow-up.
func TestRateDuration_QuantityCustomCancellation_Engine(t *testing.T) {
	rate := &types.Rate{
		Amount:  &types.Quantity{Value: decimal.NewFromInt(100), Unit: "cake"},
		PerUnit: "box",
	}
	qty := &types.Quantity{Value: decimal.NewFromInt(5), Unit: "box"}
	result, cancelled, err := rateTimesQuantity(rate, qty)
	if err != nil {
		t.Fatalf("rateTimesQuantity errored: %v", err)
	}
	if !cancelled {
		t.Fatal("expected cancellation, got false")
	}
	resQty, ok := result.(*types.Quantity)
	if !ok {
		t.Fatalf("want *types.Quantity, got %T", result)
	}
	if resQty.Unit != "cake" {
		t.Errorf("Unit = %q, want 'cake'", resQty.Unit)
	}
	if got := resQty.Value.String(); got != "500" {
		t.Errorf("Value = %s, want 500", got)
	}
}

// AE3 (engine-level, refusal) — distinct Custom units do NOT cancel.
// `cake / box * 5 cake` would naively look like cancellation since
// both sides have Custom-category units, but `cake != box` so the
// engine refuses (returns cancelled=false). Dispatch then routes
// to U4's R6 refusal at the operator level.
func TestRateDuration_DistinctCustomDoesNotCancel(t *testing.T) {
	rate := &types.Rate{
		Amount:  &types.Quantity{Value: decimal.NewFromInt(100), Unit: "cake"},
		PerUnit: "box",
	}
	qty := &types.Quantity{Value: decimal.NewFromInt(5), Unit: "cake"}
	_, cancelled, err := rateTimesQuantity(rate, qty)
	if err != nil {
		t.Fatalf("rateTimesQuantity errored unexpectedly: %v", err)
	}
	if cancelled {
		t.Fatal("expected no cancellation between cake/box and cake (distinct Custom units)")
	}
}

// AE6 — Rate × Number stays as scaling. Confirms R5 — bare numbers
// don't get treated as implicit durations in the rate's denominator.
func TestRateDuration_NumberScalingPreserved(t *testing.T) {
	res := evalSingleResult(t, "$100 / hour * 2\n")
	rate, ok := res.(*types.Rate)
	if !ok {
		t.Fatalf("want *types.Rate (scaled), got %T (%v)", res, res)
	}
	if rate.PerUnit != "hour" {
		t.Errorf("PerUnit = %q, want 'hour'", rate.PerUnit)
	}
	if got := rate.Amount.Value.String(); got != "200" {
		t.Errorf("Amount.Value = %s, want 200", got)
	}
}

// Out-of-scope for v2.1 (documented for the follow-up):
//
//   - `3 weeks * $100 / hour` (duration on the left). Operator
//     precedence parses this as `((3 weeks) * $100) / hour` rather
//     than `(3 weeks) * ($100/hour)`, so it requires explicit
//     parentheses around the rate. The natural form is rate-on-left
//     and we lock that one above.
//   - `$100 / hour * 5 kg` should error with a dimensional-mismatch
//     diagnostic — locked by the U4 refusal-contract tests in this
//     same PR (see TestRateDuration_RefusalForCrossCategory_*).
