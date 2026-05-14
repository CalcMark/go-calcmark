package interpreter

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/parser"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/shopspring/decimal"
)

// Rate + Rate / Rate - Rate for time-based rates with compatible
// (cancellable) numerator units. The two rates' PerUnits are both
// time units, so the right-hand rate is converted to the left's
// PerUnit before the values are combined (first-unit-wins).
//
// This is the user-facing fix for the long-standing "cannot add
// rate ($X/year) and rate ($Y/week)" refusal — the message was
// correct in spirit (the values can't be added at the bytes
// level), but with both PerUnits in the same time category we can
// convert one into the other and produce a sensible result.

// evalRate parses a single calcmark source string, evaluates it,
// asserts the last result is a *types.Rate, and returns the rate.
// Fails the test on any parse/eval error.
func evalRate(t *testing.T, source string) *types.Rate {
	t.Helper()
	nodes, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	interp := NewInterpreter()
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("no results returned")
	}
	rate, ok := results[len(results)-1].(*types.Rate)
	if !ok {
		t.Fatalf("expected *types.Rate, got %T (%v)", results[len(results)-1], results[len(results)-1])
	}
	return rate
}

// The headline scenario from the feature request: three currency rates
// at distinct time bases summed via the + operator. First-unit-wins
// fixes the result to the leftmost rate's PerUnit (year).
//
//	home  = $1K / year   → $1000.00 / year
//	car   = $70 / week   → $3650.00 / year   (70 × 31,536,000/604,800)
//	other = $0.5 / day   → $182.50  / year   (0.5 × 31,536,000/86,400)
//	total = $4832.50     / year
func TestRateAddition_CurrencyRatesMixedTimeUnits(t *testing.T) {
	source := "home = $1K / year\n" +
		"car = $70 / week\n" +
		"other = $0.5 / day\n" +
		"total = home + car + other\n"

	rate := evalRate(t, source)

	if rate.PerUnit != "year" {
		t.Errorf("PerUnit = %q, want \"year\" (first-unit-wins)", rate.PerUnit)
	}
	if rate.Amount.Unit != "$" {
		t.Errorf("Amount.Unit = %q, want \"$\"", rate.Amount.Unit)
	}
	if got := rate.Amount.Value.String(); got != "4832.5" {
		t.Errorf("Amount.Value = %s, want 4832.5", got)
	}
}

// Rate + Rate where both PerUnits are already the same time unit
// is the easy case — no conversion required, values just add.
// Locks the no-conversion path.
func TestRateAddition_SameTimeUnit(t *testing.T) {
	source := "a = $100 / hour\nb = $50 / hour\nc = a + b\n"

	rate := evalRate(t, source)

	if rate.PerUnit != "hour" {
		t.Errorf("PerUnit = %q, want \"hour\"", rate.PerUnit)
	}
	if got := rate.Amount.Value.String(); got != "150" {
		t.Errorf("Amount.Value = %s, want 150", got)
	}
}

// Subtraction mirrors addition: time-unit conversion + first-unit-wins.
// $1000/year - $5/week  → $1000/year - $260/year = $740/year.
func TestRateSubtraction_CurrencyRatesMixedTimeUnits(t *testing.T) {
	source := "income = $1000 / year\nspend = $5 / week\nnet = income - spend\n"

	rate := evalRate(t, source)

	if rate.PerUnit != "year" {
		t.Errorf("PerUnit = %q, want \"year\"", rate.PerUnit)
	}
	// $5/week → $/year = 5 × 31,536,000 / 604,800 = 260.714285714…
	// net = 1000 − 260.714285714… = 739.285714285714285714…
	// shopspring/decimal Div rounds at 16 frac digits → 739.2857142857142857.
	if got := rate.Amount.Value.String(); got != "739.2857142857142857" {
		t.Errorf("Amount.Value = %s, want 739.2857142857142857 (1000 - 5×52.142857…)", got)
	}
}

// Data-rate numerators (MB/s vs KB/s) cancel via DataSize category.
// In CalcMark's DataSize registry, KB aliases to KiB and MB to MiB
// (binary, 1024-based). So 1 MB/s + 512 KB/s = 1 MB/s + 0.5 MB/s
// = 1.5 MB/s.
func TestRateAddition_DataSizeCompatibleNumerators(t *testing.T) {
	source := "fast = 1 MB / second\nslow = 512 KB / second\ntotal = fast + slow\n"

	rate := evalRate(t, source)

	if rate.PerUnit != "second" {
		t.Errorf("PerUnit = %q, want \"second\"", rate.PerUnit)
	}
	if rate.Amount.Unit != "MB" {
		t.Errorf("Amount.Unit = %q, want \"MB\" (first-unit-wins)", rate.Amount.Unit)
	}
	if got := rate.Amount.Value.String(); got != "1.5" {
		t.Errorf("Amount.Value = %s, want 1.5", got)
	}
}

// Cross-category numerators (currency vs data) cannot cancel — addition
// must refuse with a clear error. This is the R6-style refusal for
// additive operators: no shared dimension between the two numerators.
func TestRateAddition_IncompatibleNumeratorsRefused(t *testing.T) {
	source := "money = $100 / hour\nbits = 100 MB / hour\nbad = money + bits\n"

	nodes, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	interp := NewInterpreter()
	_, err = interp.Eval(nodes)
	if err == nil {
		t.Fatal("expected error for $/h + MB/h, got nil")
	}
}

// Cross-currency rate addition: EUR/week + JPY/year with a frontmatter
// EUR_JPY exchange rate. The right rate is converted to the left's
// currency before the time-unit conversion runs, so the result is in
// the left's units (EUR/week — first-unit-wins). Without an exchange
// rate, the same expression must refuse (locked in the next test).
//
// Math (with EUR_JPY = 150):
//
//	right = 15000 JPY/year × (1 EUR / 150 JPY)
//	      = 100 EUR/year
//	right scaled to /week = 100 × (604800 / 31536000)
//	                      = 100 / 52.142857… ≈ 1.91780821… EUR/week
//	left + right = 10 EUR/week + 1.91780821… EUR/week
//	             = 11.91780821… EUR/week
func TestRateAddition_CrossCurrencyWithFrontmatterExchange(t *testing.T) {
	source := "salary = €10 / week\nbonus = ¥15000 / year\ntotal = salary + bonus\n"

	nodes, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	interp := NewInterpreter()
	interp.GetEnvironment().SetExchangeRate("JPY", "EUR", decimal.RequireFromString("0.00666666666666666667"))
	// 1/150 = 0.00666… — sets up the same effective rate as EUR_JPY=150 in
	// the reverse direction (the conversion code looks up from→to).

	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	rate, ok := results[len(results)-1].(*types.Rate)
	if !ok {
		t.Fatalf("expected *types.Rate, got %T (%v)", results[len(results)-1], results[len(results)-1])
	}
	if rate.PerUnit != "week" {
		t.Errorf("PerUnit = %q, want \"week\" (first-unit-wins)", rate.PerUnit)
	}
	if rate.Amount.Unit != "€" {
		t.Errorf("Amount.Unit = %q, want \"€\" (first-unit-wins)", rate.Amount.Unit)
	}
	// 100 EUR/year = 100 × 604800 / 31536000 = 100/52.142857… EUR/week.
	// Total = 10 + (100 × 604800 / 31536000) = 11.91780821917808…
	got := rate.Amount.Value.String()
	if !strings.HasPrefix(got, "11.9178082") {
		t.Errorf("Amount.Value = %s, want ~11.91780821… (10 + 100/52.142857…)", got)
	}
}

// Same cross-currency expression without an exchange rate — must
// refuse with a clear error rather than silently picking a rate.
func TestRateAddition_CrossCurrencyWithoutExchangeRefused(t *testing.T) {
	source := "salary = €10 / week\nbonus = ¥15000 / year\ntotal = salary + bonus\n"

	nodes, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	interp := NewInterpreter()
	// No exchange rate configured — interpreter must refuse.

	_, err = interp.Eval(nodes)
	if err == nil {
		t.Fatal("expected error for €/week + ¥/year without exchange rate, got nil")
	}
}

// Two distinct Custom units (bunnies vs owls) have no converter and
// no shared registered category — they're both "Custom" but with
// different unit strings, so the `cancellable` predicate refuses.
// Addition must surface that refusal rather than silently coerce
// (which would produce a meaningless "4 bunnies/year" from "2 bunnies
// + 2 owls"). Locks the contract for arbitrary user-coined units.
func TestRateAddition_DistinctCustomNumeratorsRefused(t *testing.T) {
	source := "rabbits = 2 bunnies / year\nbirds = 2 owls / week\ntotal = rabbits + birds\n"

	nodes, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	interp := NewInterpreter()
	_, err = interp.Eval(nodes)
	if err == nil {
		t.Fatal("expected error for bunnies/year + owls/week (distinct custom units), got nil")
	}
	// The refusal should name the rate operands so the user can see
	// which two units didn't compose.
	msg := err.Error()
	if !strings.Contains(msg, "bunnies") || !strings.Contains(msg, "owls") {
		t.Errorf("error %q should mention both 'bunnies' and 'owls'", msg)
	}
}
