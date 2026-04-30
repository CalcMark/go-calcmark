package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/shopspring/decimal"
)

// Rate × Duration / Rate × Rate arithmetic with time-unit
// cancellation. Sits next to the binary-op dispatch in operators.go.
// The dispatch calls into these helpers when it spots a Rate on the
// left and a Duration or Rate on the right; everything else stays in
// operators.go where it always lived.

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

// isTimeUnit reports whether s is a recognised duration unit.
// Convenience wrapper to keep the call sites readable.
func isTimeUnit(s string) bool {
	return types.IsValidDurationUnit(s)
}

// convertTimeValue scales a numeric value from one time unit into
// another. Wraps types.Duration.Convert so callers don't have to
// build a Duration just to do the arithmetic.
func convertTimeValue(value decimal.Decimal, fromUnit, toUnit string) (decimal.Decimal, error) {
	if fromUnit == toUnit {
		return value, nil
	}
	d := &types.Duration{Value: value, Unit: fromUnit}
	converted, err := d.Convert(toUnit)
	if err != nil {
		return decimal.Zero, err
	}
	return converted.Value, nil
}

// isCurrencyCode reports whether s appears as an ISO currency code
// in the SymbolToCode map (e.g. "USD"). Symbols like "$" are matched
// directly by membership; codes are matched via the values side.
func isCurrencyCode(s string) bool {
	for _, code := range types.SymbolToCode {
		if code == s {
			return true
		}
	}
	return false
}
