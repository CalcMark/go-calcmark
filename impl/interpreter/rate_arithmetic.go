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

// rateTimesDuration multiplies a Rate by a Duration. The Duration is
// converted into the rate's PerUnit (e.g. weeks → hours when the
// rate is `$/hour`); once both sides agree on the time unit, that
// dimension cancels and the rate's numerator is the result type.
//
// Errors when:
//   - The rate's PerUnit is not a recognised time unit (so the
//     conversion has nowhere to land — e.g. `100 req/server * 3
//     weeks` doesn't compose).
//   - The Duration's unit can't convert to the PerUnit (defence in
//     depth — Duration values from the parser already use the
//     canonical time-unit set, but a future feature could surface a
//     different unit name).
func rateTimesDuration(r *types.Rate, d *types.Duration) (types.Type, error) {
	// PerUnit must be a time unit for the conversion to be meaningful.
	// `100 req / server * 3 weeks` is a category error; surface it.
	if !types.IsValidDurationUnit(r.PerUnit) {
		return nil, fmt.Errorf(
			"cannot multiply rate (%s) by duration (%s): rate's PerUnit %q is not a time unit",
			r, d, r.PerUnit)
	}
	converted, err := d.Convert(r.PerUnit)
	if err != nil {
		return nil, fmt.Errorf(
			"cannot multiply rate (%s) by duration (%s): %w",
			r, d, err)
	}
	value := r.Amount.Value.Mul(converted.Value)
	return rateNumeratorAsResult(r, value), nil
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
//   left.PerUnit ↔ right.Amount.Unit
//     (a / b) * (b / c) → (a / c)
//     example: ($/hour) * (hours/week) → $/week
//
//   right.PerUnit ↔ left.Amount.Unit
//     (b / c) * (a / b) → (a / c)  — commuted
//     example: (hours/week) * ($/hour) → $/week
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
	if isTimeUnit(left.PerUnit) && isTimeUnit(right.Amount.Unit) {
		// Convert right's Amount value into left.PerUnit so we can
		// multiply against left's Amount cleanly.
		factor, err := convertTimeValue(right.Amount.Value, right.Amount.Unit, left.PerUnit)
		if err != nil {
			return nil, false, fmt.Errorf(
				"cannot cancel %q on (%s) * (%s): %w",
				left.PerUnit, left, right, err)
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
	if isTimeUnit(left.Amount.Unit) && isTimeUnit(right.PerUnit) {
		factor, err := convertTimeValue(left.Amount.Value, left.Amount.Unit, right.PerUnit)
		if err != nil {
			return nil, false, fmt.Errorf(
				"cannot cancel %q on (%s) * (%s): %w",
				right.PerUnit, left, right, err)
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

// rateNumeratorAsResult reconstructs the rate's numerator as a
// concrete result type after the denominator has been cancelled.
// Currency-symbol numerator → Currency; non-empty unit → Quantity;
// empty unit → Number.
func rateNumeratorAsResult(r *types.Rate, value decimal.Decimal) types.Type {
	unit := r.Amount.Unit
	if unit == "" {
		return types.NewNumber(value)
	}
	if _, ok := types.SymbolToCode[unit]; ok {
		return types.NewCurrency(value, unit)
	}
	// ISO codes (USD, EUR, ...) live in the symbol→code map values
	// AND can appear as Amount.Unit when the user wrote "100 USD/day"
	// rather than "$100/day". Reverse-lookup catches them.
	if isCurrencyCode(unit) {
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
// Kept local for now; U3 may switch call sites to `types.IsCurrencyCode`
// (which checks both SymbolToCode and CodeToSymbol) for completeness.
func isCurrencyCode(s string) bool {
	for _, code := range types.SymbolToCode {
		if code == s {
			return true
		}
	}
	return false
}
