// Package transform applies document-level post-evaluation transforms
// (scale, convert_to) to CalcMark evaluation results.
package transform

import (
	"strings"

	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/CalcMark/go-calcmark/spec/units"
)

// Apply applies scale and convert_to transforms to an evaluation result.
// Returns a new value (or the original if no transform applies).
// Order: scale first, then convert_to.
func Apply(result types.Type, scale *document.ScaleConfig, convertTo *document.ConvertToConfig) types.Type {
	if result == nil {
		return nil
	}
	if scale == nil && convertTo == nil {
		return result
	}

	switch v := result.(type) {
	case *types.Quantity:
		out := cloneQuantity(v)
		out = applyScaleQuantity(out, scale)
		out = applyConvertToQuantity(out, convertTo)
		return out

	case *types.Rate:
		// Rates are immune to scale but can have their amount's unit converted
		return applyConvertToRate(v, convertTo)

	case *types.Currency:
		// Currency is immune to scale by default. Only scaled when
		// "Currency" is explicitly listed in unit_categories.
		if scale != nil && categoryMatches("Currency", scale.UnitCategories) {
			scaled := v.Value.Mul(scale.Factor)
			return types.NewCurrency(scaled, v.Symbol)
		}
		return result

	case *types.Number:
		// Numbers are immune to scale by default. Only scaled when
		// "Number" is explicitly listed in unit_categories.
		if scale != nil && categoryMatches("Number", scale.UnitCategories) {
			scaled := v.Value.Mul(scale.Factor)
			return types.NewNumber(scaled)
		}
		return result

	default:
		// Duration, Boolean, Date, Time, Percentage — unchanged
		return result
	}
}

// WouldScale reports whether Apply would scale this result given the scale config.
// Used by the TUI to show a visual indicator on scaled values.
// Scaling is always explicit: unit_categories must list the categories to scale.
func WouldScale(result types.Type, scale *document.ScaleConfig) bool {
	if result == nil || scale == nil || len(scale.UnitCategories) == 0 {
		return false
	}
	switch v := result.(type) {
	case *types.Quantity:
		category := units.CategoryForUnit(v.Unit)
		return categoryMatches(category, scale.UnitCategories)
	case *types.Currency:
		return categoryMatches("Currency", scale.UnitCategories)
	case *types.Number:
		return categoryMatches("Number", scale.UnitCategories)
	default:
		return false
	}
}

// applyScaleQuantity multiplies a quantity's value by the scale factor.
// Scaling is explicit: only scales when the quantity's category is listed
// in unit_categories. No unit_categories means no scaling.
func applyScaleQuantity(q *types.Quantity, scale *document.ScaleConfig) *types.Quantity {
	if scale == nil || len(scale.UnitCategories) == 0 {
		return q
	}

	category := units.CategoryForUnit(q.Unit)
	if !categoryMatches(category, scale.UnitCategories) {
		return q
	}

	q.Value = q.Value.Mul(scale.Factor)
	return q
}

// applyConvertToQuantity converts a quantity to the target measurement system.
// Skips explicit quantities (user chose the unit with `in`/`as`).
func applyConvertToQuantity(q *types.Quantity, convertTo *document.ConvertToConfig) *types.Quantity {
	if convertTo == nil {
		return q
	}

	// Explicit `in` overrides convert_to
	if q.IsExplicit {
		return q
	}

	category := units.CategoryForUnit(q.Unit)
	if category == units.CategoryCustom {
		// Custom unit (e.g., "eggs") — no system mapping
		return q
	}

	// Check unit_categories filter
	if len(convertTo.UnitCategories) > 0 {
		if !categoryMatches(category, convertTo.UnitCategories) {
			return q
		}
	}

	// Check if the unit is already in the target system
	currentSystem := units.GetSystemForUnit(q.Unit)
	if isInTargetSystem(currentSystem, convertTo.System) {
		return q
	}

	// Find the target unit
	targetUnit := units.GetDefaultTargetUnit(q.Unit, convertTo.System)
	if targetUnit == "" {
		// No target unit available for this category — pass through
		return q
	}

	// Perform conversion
	converted, err := units.Convert(q.Value, q.Unit, targetUnit)
	if err != nil {
		// Conversion failed — pass through silently
		return q
	}

	q.Value = converted
	q.Unit = targetUnit
	return q
}

// applyConvertToRate converts the Amount's unit of a rate, leaving the time denominator unchanged.
func applyConvertToRate(r *types.Rate, convertTo *document.ConvertToConfig) *types.Rate {
	if convertTo == nil || r == nil || r.Amount == nil {
		return r
	}

	convertedAmount := applyConvertToQuantity(cloneQuantity(r.Amount), convertTo)

	// If nothing changed, return original
	if convertedAmount.Unit == r.Amount.Unit && convertedAmount.Value.Equal(r.Amount.Value) {
		return r
	}

	return &types.Rate{
		Amount:  convertedAmount,
		PerUnit: r.PerUnit,
	}
}

// cloneQuantity creates a shallow copy of a Quantity so transforms don't mutate the original.
func cloneQuantity(q *types.Quantity) *types.Quantity {
	return &types.Quantity{
		Value:      q.Value,
		Unit:       q.Unit,
		IsNapkin:   q.IsNapkin,
		IsExplicit: q.IsExplicit,
		IsPrecise:  q.IsPrecise,
	}
}

// categoryMatches checks if a category is in the allowed list (case-insensitive).
// The special keyword "All" matches every category.
func categoryMatches(category string, allowed []string) bool {
	for _, c := range allowed {
		if strings.EqualFold(c, "All") || strings.EqualFold(category, c) {
			return true
		}
	}
	return false
}

// isInTargetSystem checks if a unit's system matches the convert_to target.
func isInTargetSystem(unitSystem, targetSystem string) bool {
	switch strings.ToLower(targetSystem) {
	case "imperial":
		return unitSystem == "US_Customary" || unitSystem == "Imperial"
	case "si":
		return unitSystem == "SI"
	}
	return false
}

// WouldConvert reports whether Apply would convert this result's unit given
// the convert_to config. Must be called on PRE-transform results (before Apply),
// because Apply changes the unit, making post-transform detection unreliable.
//
// Used by the evaluator to cache per-statement conversion flags.
func WouldConvert(result types.Type, convertTo *document.ConvertToConfig) bool {
	if result == nil || convertTo == nil {
		return false
	}
	switch v := result.(type) {
	case *types.Quantity:
		return wouldConvertQuantity(v, convertTo)
	case *types.Rate:
		if v.Amount == nil {
			return false
		}
		return wouldConvertQuantity(v.Amount, convertTo)
	default:
		return false
	}
}

// wouldConvertQuantity checks whether a quantity would actually be converted.
// Mirrors the bail-out checks in applyConvertToQuantity.
func wouldConvertQuantity(q *types.Quantity, convertTo *document.ConvertToConfig) bool {
	if q.IsExplicit {
		return false
	}
	category := units.CategoryForUnit(q.Unit)
	if category == units.CategoryCustom {
		return false
	}
	if len(convertTo.UnitCategories) > 0 {
		if !categoryMatches(category, convertTo.UnitCategories) {
			return false
		}
	}
	currentSystem := units.GetSystemForUnit(q.Unit)
	if isInTargetSystem(currentSystem, convertTo.System) {
		return false
	}
	targetUnit := units.GetDefaultTargetUnit(q.Unit, convertTo.System)
	if targetUnit == "" {
		return false
	}
	// Confirm conversion would actually succeed
	_, err := units.Convert(q.Value, q.Unit, targetUnit)
	return err == nil
}

// ApplyToResults applies transforms to a slice of results, returning new values.
// This is a convenience for formatters that process result lists.
func ApplyToResults(results []types.Type, scale *document.ScaleConfig, convertTo *document.ConvertToConfig) []types.Type {
	if scale == nil && convertTo == nil {
		return results
	}
	out := make([]types.Type, len(results))
	for i, r := range results {
		out[i] = Apply(r, scale, convertTo)
	}
	return out
}
