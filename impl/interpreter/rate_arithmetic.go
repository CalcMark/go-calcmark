package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/CalcMark/go-calcmark/v2/spec/units"
	"github.com/shopspring/decimal"
)

// Rate × Duration / Rate × Rate / Rate × Quantity arithmetic with
// same-dimension unit cancellation across every category CalcMark
// supports. Sits next to the binary-op dispatch in operators.go.
//
// The dispatch calls into these helpers when it spots a Rate on the
// left and a Duration / Quantity / Rate on the right; everything else
// stays in operators.go where it always lived.
//
// The cancellation predicate is "shared category" (with exact-string
// equality as the fallback for Custom units that have no converter).
// Time units live in spec/types/duration.go and use Duration.Convert;
// every other category lives in spec/units/conversion.go. The
// categoryOf + convertWithinCategory helpers below paper over that
// split so the cancellation engine itself doesn't have to care.

// tryRateCancellation attempts to cancel the rate's PerUnit against
// the unit on a scalar operand (a Duration's Unit, a Quantity's Unit).
// Used by `Rate × Duration` and `Rate × Quantity` dispatch.
//
// Returns:
//   - (result, true, nil) when cancellation succeeded — the rate's
//     numerator is reconstructed via `rateNumeratorAsResult`.
//   - (nil, false, nil) when the cancellation predicate did NOT match,
//     OR matched but the conversion failed. In both cases the caller
//     should fall through to R6 refusal at the dispatch level rather
//     than surface the convert-error string. (Plan G3 resolution:
//     leak no internal "cannot convert X to Y" messages as user-
//     facing errors; R6's contract takes over.)
//
// The predicate: cancellation applies when (a) `categoryOf(r.PerUnit)`
// equals `categoryOf(oppUnit)` and both are non-empty, AND (b) for
// `Custom` units, the strings are equal (since Custom has no
// converter).
func tryRateCancellation(r *types.Rate, oppValue decimal.Decimal, oppUnit string) (types.Type, bool, error) {
	rateCat := categoryOf(r.PerUnit)
	oppCat := categoryOf(oppUnit)
	if rateCat == "" || oppCat == "" {
		return nil, false, nil
	}
	if rateCat != oppCat {
		return nil, false, nil
	}
	// Custom units cancel only when literally the same string. Distinct
	// Custom names (cake vs box) fall through to refusal.
	if rateCat == "Custom" && r.PerUnit != oppUnit {
		return nil, false, nil
	}
	converted, err := convertWithinCategory(oppValue, oppUnit, r.PerUnit)
	if err != nil {
		// Predicate matched but conversion math failed. Treat as
		// "no cancellation" so dispatch routes to R6's friendlier
		// refusal message rather than leaking the converter's
		// internal error string.
		return nil, false, nil
	}
	value := r.Amount.Value.Mul(converted)
	return rateNumeratorAsResult(r, value), true, nil
}

// rateTimesDuration is a thin wrapper around `tryRateCancellation` for
// the Rate × Duration call site in operators.go. Returns the same
// 3-tuple shape as `tryRateRateCancellation` so the dispatch can
// uniformly check `cancelled` and fall through to R6 on false.
func rateTimesDuration(r *types.Rate, d *types.Duration) (types.Type, bool, error) {
	return tryRateCancellation(r, d.Value, d.Unit)
}

// rateTimesQuantity is a thin wrapper for the Rate × Quantity call
// site. Same shape as `rateTimesDuration`. This replaces the
// long-standing `operators.go:367-371` silent-coercion rule that
// ignored the rate's `PerUnit` entirely.
func rateTimesQuantity(r *types.Rate, q *types.Quantity) (types.Type, bool, error) {
	return tryRateCancellation(r, q.Value, q.Unit)
}

// tryRateRateCancellation handles `Rate × Rate` when one rate's
// numerator unit cancels the other rate's denominator unit. Returns
// (result, true, nil) on a successful cancellation, (nil, false, nil)
// when no cancellation applies (caller falls through to other rules),
// or (nil, false, err) when a cancellation was detected but the unit
// arithmetic failed (e.g. minute↔hour conversion of an unknown unit).
//
// The two cancellable shapes:
//
//	left.PerUnit ↔ right.Amount.Unit
//	  (a / b) * (b / c) → (a / c)
//	  example: ($/hour) * (hours/week) → $/week
//
//	right.PerUnit ↔ left.Amount.Unit
//	  (b / c) * (a / b) → (a / c)  — commuted
//	  example: (hours/week) * ($/hour) → $/week
//
// In each case the cancelled time unit is converted on the
// quantity-side rate (rightSide for the first, leftSide for the
// second) so a `... * 60 minutes / hour` left-hand factor still
// cancels a `... / hour` right-hand denominator.
//
// Result kind:
//   - The remaining numerator is the surviving Amount (currency or
//     quantity).
//   - The remaining denominator is the surviving PerUnit. If the
//     surviving PerUnit is empty (which doesn't happen today — Rate
//     literals always carry a PerUnit), the result reduces to the
//     Amount type itself.
func tryRateRateCancellation(left, right *types.Rate) (types.Type, bool, error) {
	// Shape 1: left.PerUnit ↔ right.Amount.Unit
	if cancellable(left.PerUnit, right.Amount.Unit) {
		factor, err := convertWithinCategory(right.Amount.Value, right.Amount.Unit, left.PerUnit)
		if err != nil {
			// Predicate matched but conversion failed — treat as
			// no-cancellation so dispatch routes to R6 refusal.
			// (Same plan G3 rule as `tryRateCancellation`.)
			return nil, false, nil
		}
		newAmount := left.Amount.Value.Mul(factor)
		return &types.Rate{
			Amount: &types.Quantity{
				Value: newAmount,
				Unit:  left.Amount.Unit,
			},
			PerUnit: right.PerUnit,
		}, true, nil
	}

	// Shape 2: left.Amount.Unit ↔ right.PerUnit (the commuted case).
	if cancellable(right.PerUnit, left.Amount.Unit) {
		factor, err := convertWithinCategory(left.Amount.Value, left.Amount.Unit, right.PerUnit)
		if err != nil {
			return nil, false, nil
		}
		newAmount := factor.Mul(right.Amount.Value)
		return &types.Rate{
			Amount: &types.Quantity{
				Value: newAmount,
				Unit:  right.Amount.Unit,
			},
			PerUnit: left.PerUnit,
		}, true, nil
	}

	return nil, false, nil
}

// tryRateRateAddition handles `Rate + Rate` and `Rate - Rate` when
// both rates are time-based (PerUnit ∈ time) and their numerator
// units share a category (currency-vs-currency, or any pair the
// `cancellable` predicate accepts). First-unit-wins: the right rate
// is converted to the left's PerUnit and the left's numerator unit
// before the values combine, and the result inherits the left's
// units.
//
// Returns:
//   - (result, true, nil) on a successful addition/subtraction.
//   - (nil, false, nil) when the predicate doesn't match — caller
//     falls through to the generic unsupported-operation error so
//     the user sees a familiar refusal message rather than an
//     opaque conversion failure.
//   - (nil, false, err) is intentionally unused today; reserved
//     for future "matched but failed" signalling if the dispatch
//     ever needs to surface a specific reason.
//
// The cancellation predicate is the same `cancellable` helper used
// by the multiplicative path — so currency-vs-currency works only
// when the two currency symbols match (predicate falls back to
// exact-string on Custom-category units, and currency symbols
// behave as Custom from the units registry's perspective). This
// preserves the long-standing "cannot mix currencies" contract.
func tryRateRateAddition(left, right *types.Rate, operator string) (types.Type, bool, error) {
	if operator != "+" && operator != "-" {
		return nil, false, nil
	}
	// Both PerUnits must be time units. Anything else (e.g. a Rate
	// whose denominator is "request" — not a time at all) falls
	// through to the existing refusal path.
	if categoryOf(left.PerUnit) != "time" || categoryOf(right.PerUnit) != "time" {
		return nil, false, nil
	}
	// Numerator units must be combinable. The `cancellable`
	// predicate handles same-category-non-Custom (e.g. MB/KB share
	// DataSize), same-string-Custom (e.g. $/$ — currency symbols
	// behave as Custom), and rejects cross-category pairs.
	if !cancellable(left.Amount.Unit, right.Amount.Unit) {
		return nil, false, nil
	}

	// Step 1: scale the right rate's value into the left's PerUnit.
	// `$70/week → $/year`: compute (70 × seconds-per-year) / seconds-per-week
	// — multiply first, divide last — so integer-ratio cases like
	// week↔day, year↔day, hour↔second land exactly rather than
	// drifting by a ULP through an intermediate decimal-division
	// rounding. Routing through `convertWithinCategory(1, left, right)`
	// would do the division first and lose precision.
	leftSec, err := types.TimeUnitToSeconds(left.PerUnit)
	if err != nil {
		return nil, false, nil
	}
	rightSec, err := types.TimeUnitToSeconds(right.PerUnit)
	if err != nil {
		return nil, false, nil
	}
	if rightSec.IsZero() {
		return nil, false, nil
	}
	rightValue := right.Amount.Value.Mul(leftSec).Div(rightSec)

	// Step 2: convert the right rate's numerator to the left's
	// numerator unit (e.g. KB → MB). Same-unit short-circuits in
	// convertWithinCategory.
	rightValue, err = convertWithinCategory(rightValue, right.Amount.Unit, left.Amount.Unit)
	if err != nil {
		return nil, false, nil
	}

	// Step 3: combine.
	var result decimal.Decimal
	switch operator {
	case "+":
		result = left.Amount.Value.Add(rightValue)
	case "-":
		result = left.Amount.Value.Sub(rightValue)
	}

	return &types.Rate{
		Amount: &types.Quantity{
			Value: result,
			Unit:  left.Amount.Unit,
		},
		PerUnit: left.PerUnit,
	}, true, nil
}

// cancellable reports whether two units can cancel against each other:
// they share a non-empty category, and for `Custom` (which has no
// converter) the strings are equal.
func cancellable(a, b string) bool {
	catA := categoryOf(a)
	catB := categoryOf(b)
	if catA == "" || catB == "" || catA != catB {
		return false
	}
	if catA == "Custom" && a != b {
		return false
	}
	return true
}

// rateNumeratorAsResult reconstructs the rate's numerator as a
// concrete result type after the denominator has been cancelled.
// Currency-symbol numerator → Currency; non-empty unit → Quantity;
// empty unit → Number.
func rateNumeratorAsResult(r *types.Rate, value decimal.Decimal) types.Type {
	unit := r.Amount.Unit
	if unit == "" {
		return types.NewNumber(value)
	}
	// Currency-symbol numerator (e.g. "$100/hour") → Currency.
	// `types.SymbolToCode` keys are the user-typed symbols ("$",
	// "€", "£", "¥"); membership means the unit string is a
	// recognised currency symbol.
	if _, ok := types.SymbolToCode[unit]; ok {
		// `NewCurrency` accepts either symbol or code and computes
		// the missing field, so passing the symbol back round-trips
		// to a Currency with both Symbol and Code populated.
		return types.NewCurrency(value, unit)
	}
	// ISO-code numerator (e.g. "100 USD/day" — the user wrote the
	// code, not the symbol). `types.IsCurrencyCode` checks both
	// directions of the code↔symbol maps, so it covers any
	// recognised code regardless of which side of the map it lives
	// in. R7 contract: preserve the user's typed form — pass the
	// code through verbatim and let NewCurrency canonicalise.
	if types.IsCurrencyCode(unit) {
		return types.NewCurrency(value, unit)
	}
	return &types.Quantity{Value: value, Unit: unit}
}

// categoryOf returns the unit category for `unit`, or "" when the
// unit is unrecognised.
//
// Returns:
//   - "time" — for any time unit recognised by `types.IsValidDurationUnit`
//     (second, minute, hour, day, week, month, year, etc.).
//     Time units live separately from `units.conversionRegistry`, so
//     they get the first-pass branch here.
//   - The result of `units.CategoryForUnit(unit)` for everything else.
//     That's the canonical name from the units registry: "Length",
//     "Mass", "DataSize", "Speed", "Volume", etc., or "Custom" for
//     unrecognised units (cakes, boxes, anything user-coined).
//   - "" — when `unit` is the empty string (a unitless rate's
//     numerator). Empty-vs-empty equality is intentionally NOT a
//     cancellation match; the cancellation predicate in
//     `tryRateCancellation` rejects empty categories.
//
// The `time` short-circuit is load-bearing: time units appear in
// `units.CategoryForUnit` as `Custom` (not registered in
// `conversionRegistry`), and we don't want them to cancel against
// other Custom units like `box`. Calling `categoryOf` always
// dispatches `time` correctly first.
func categoryOf(unit string) string {
	if unit == "" {
		return ""
	}
	if types.IsValidDurationUnit(unit) {
		return "time"
	}
	return units.CategoryForUnit(unit)
}

// convertWithinCategory scales `value` from `fromUnit` to `toUnit`
// when both units share a unit category and the category supports
// conversion. Returns the converted value, or an error when the two
// units don't share a category, when the category has no converter,
// or when the underlying converter rejects the input.
//
// Dispatch:
//   - Identical units → return the value unchanged.
//   - Both time units → `types.Duration.Convert` does the math.
//   - Both in a `units.conversionRegistry` category → `units.Convert`
//     does the math (same category check is built into that call,
//     but we pre-check here so the error message names both units).
//   - Both `Custom` (or any category not in conversionRegistry) →
//     only the identical-string case above succeeds; otherwise
//     return an error. Custom units have no converter, so cancellation
//     between them only works on exact-string matches like
//     `box × box`.
//
// The error message names both units explicitly so the R6 refusal
// path in operators.go can wrap it without losing information.
func convertWithinCategory(value decimal.Decimal, fromUnit, toUnit string) (decimal.Decimal, error) {
	if fromUnit == toUnit {
		return value, nil
	}
	fromCat := categoryOf(fromUnit)
	toCat := categoryOf(toUnit)
	if fromCat == "" || toCat == "" {
		return decimal.Zero, fmt.Errorf(
			"cannot convert %q to %q: unrecognised unit", fromUnit, toUnit)
	}
	if fromCat != toCat {
		return decimal.Zero, fmt.Errorf(
			"cannot convert %q to %q: different categories (%s vs %s)",
			fromUnit, toUnit, fromCat, toCat)
	}
	if fromCat == "time" {
		d := &types.Duration{Value: value, Unit: fromUnit}
		converted, err := d.Convert(toUnit)
		if err != nil {
			return decimal.Zero, err
		}
		return converted.Value, nil
	}
	// Custom units have no converter. The fromUnit == toUnit case is
	// handled at the top of the function; reaching here means two
	// distinct Custom unit names. Refuse rather than silently coerce.
	if fromCat == "Custom" {
		return decimal.Zero, fmt.Errorf(
			"cannot convert %q to %q: %q has no converter (custom unit)",
			fromUnit, toUnit, fromUnit)
	}
	return units.Convert(value, fromUnit, toUnit)
}

// isTimeUnit reports whether s is a recognised duration unit.
// Retained as a tiny wrapper around `categoryOf` for call-site
// readability; new code should prefer `categoryOf(s) == "time"` so
// the predicate is uniform with non-time-domain checks.
func isTimeUnit(s string) bool {
	return categoryOf(s) == "time"
}

// convertTimeValue is a backwards-compatible alias for
// `convertWithinCategory` retained while U2 is in flight. Once the
// cancellation engine is fully generalised, callers will be moved
// onto `convertWithinCategory` directly and this shim removed.
func convertTimeValue(value decimal.Decimal, fromUnit, toUnit string) (decimal.Decimal, error) {
	return convertWithinCategory(value, fromUnit, toUnit)
}

// isCurrencyCode reports whether s appears as an ISO currency code
// in the SymbolToCode map (e.g. "USD"). Symbols like "$" are matched
// directly by membership; codes are matched via the values side.
//
// Retained as a thin local helper. The result-construction path
// in `rateNumeratorAsResult` uses `types.IsCurrencyCode` directly
// for completeness; this local helper is preserved as a back-compat
// shim for any historical callers.
func isCurrencyCode(s string) bool {
	for _, code := range types.SymbolToCode {
		if code == s {
			return true
		}
	}
	return false
}

// coerceSpeedQuantityToRate widens a Speed-shaped Quantity into the
// Rate it represents (e.g. `60 mph` → `Rate{60 mi, hour}`). Returns
// (nil, false) when the quantity's unit isn't a recognised Speed
// unit. Used by operators.go's left-side widening to make AE4
// (`60 mph × 2 hours → 120 miles`) flow through the cancellation
// engine.
//
// Free-function variant of `Interpreter.coerceSpeedToRate` —
// intentionally has no Interpreter receiver since the binary-op
// dispatch in `evalBinaryOperation` is also a free function.
func coerceSpeedQuantityToRate(qty *types.Quantity) (*types.Rate, bool) {
	numUnit, timeUnit, ok := units.DecomposeSpeedUnit(qty.Unit)
	if !ok {
		return nil, false
	}
	return types.NewRate(&types.Quantity{Value: qty.Value, Unit: numUnit}, timeUnit), true
}

// rateMismatchError builds the R6 refusal message for `Rate × X`
// where the cancellation engine couldn't find a shared dimension.
// The message names both operands AND the categorical reason the
// multiplication doesn't compose, and includes the word "cancel" so
// substring-matching tests have a stable anchor.
//
// Two distinct templates so the message reads naturally for the
// two main cases:
//
//   - Rate × Currency or Rate × Rate-with-same-numerator-currency →
//     "cannot multiply two currencies" form.
//   - Everything else → "cannot multiply rate (X) by Y — Z is a
//     <category> unit; the rate's denominator is <unit>, a <category>
//     unit. No shared dimension to cancel."
//
// The exact wording is *not* part of the contract — tests verify
// substrings (`cancel`, both operand displays, both unit names).
// Future tuning of the wording is welcome as long as those substrings
// survive.
func rateMismatchError(left *types.Rate, right types.Type) error {
	// Special case: Currency × Currency (when right collapses to a
	// Currency-numerator rate or is a bare Currency operand). Plan
	// G10: same R6 contract, slightly different template since
	// "no shared dimension" doesn't quite apply when both sides ARE
	// the same dimension (currency).
	if _, ok := right.(*types.Currency); ok {
		return fmt.Errorf(
			"cannot multiply rate (%s) by currency (%s): multiplying two currencies has no meaningful unit. "+
				"To accumulate a rate over time, multiply by a duration that cancels the rate's denominator instead.",
			left, formatTypeForError(right))
	}

	// Pull units off the right operand for the message body.
	var rightUnit string
	switch r := right.(type) {
	case *types.Duration:
		rightUnit = r.Unit
	case *types.Quantity:
		rightUnit = r.Unit
	case *types.Rate:
		rightUnit = r.PerUnit
	}

	leftCat := categoryOf(left.PerUnit)
	rightCat := categoryOf(rightUnit)

	// Build a category description that names both sides clearly.
	// Keep it short: the substring contract requires the unit names
	// and the word "cancel"; richer prose can come later.
	leftCatPhrase := categoryPhrase(left.PerUnit, leftCat)
	rightCatPhrase := categoryPhrase(rightUnit, rightCat)

	return fmt.Errorf(
		"cannot multiply rate (%s) by %s: %s, but %s. No shared dimension to cancel.",
		left, formatTypeForError(right), leftCatPhrase, rightCatPhrase)
}

// categoryPhrase produces a short natural phrase describing a unit
// and its category, for use in rateMismatchError.
//
//	categoryPhrase("hour", "time")    → `the rate's denominator is "hour" (a time unit)`
//	categoryPhrase("kg", "Mass")      → `"kg" is a Mass unit`
//	categoryPhrase("box", "Custom")   → `"box" is a custom unit`
//	categoryPhrase("", "")            → `the operand has no unit`
func categoryPhrase(unit, category string) string {
	if unit == "" {
		return "the operand has no unit"
	}
	if category == "" {
		return fmt.Sprintf("%q is an unrecognised unit", unit)
	}
	if category == "Custom" {
		return fmt.Sprintf("%q is a custom unit", unit)
	}
	return fmt.Sprintf("%q is a %s unit", unit, category)
}
