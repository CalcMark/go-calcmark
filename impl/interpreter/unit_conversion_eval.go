package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/CalcMark/go-calcmark/spec/units"
)

// evalUnitConversion evaluates explicit unit conversion: "10 meters in feet"
// Also handles rate-to-rate conversion: "10 m/s in inch/s"
// Also handles currency conversion: "100 USD in EUR" (requires exchange rate in frontmatter)
func (interp *Interpreter) evalUnitConversion(u *ast.UnitConversion) (types.Type, error) {
	// Resolve the target unit using measurement conventions.
	// This is a pre-interpreter step for the target side of "X in Y".
	targetUnit := interp.resolveUnit(u.TargetUnit)

	// Evaluate the quantity expression
	result, err := interp.evalNode(u.Quantity)
	if err != nil {
		return nil, err
	}

	// Check if this is currency conversion (currency codes are not affected by measurement conventions)
	if currency, ok := result.(*types.Currency); ok {
		_, isQuantityUnit := units.NormalizeUnitName(targetUnit)
		isDuration := types.IsValidDurationUnit(targetUnit)
		if isQuantityUnit || isDuration {
			return nil, fmt.Errorf("cannot convert currency (%s) to unit '%s'; use a currency code like USD, EUR, GBP",
				currency.Code, targetUnit)
		}
		return interp.evalCurrencyConversion(currency, targetUnit)
	}

	// Check if this is a rate-to-rate conversion
	if u.TargetTimeUnit != "" {
		// Bridge: Speed quantity -> Rate (e.g., "60 kph in m/s")
		if qty, ok := result.(*types.Quantity); ok && units.IsSpeedUnit(qty.Unit) {
			coerced, err := interp.coerceSpeedToRate(qty)
			if err != nil {
				return nil, err
			}
			return interp.evalRateUnitConversion(coerced, targetUnit, u.TargetTimeUnit)
		}
		return interp.evalRateUnitConversion(result, targetUnit, u.TargetTimeUnit)
	}

	// Check if source is a rate and target is a time unit
	// This handles "10 MB/day in seconds" -> keep MB, convert day to seconds
	if rate, ok := result.(*types.Rate); ok {
		if types.IsTimeUnit(targetUnit) {
			return interp.evalRateUnitConversion(result, rate.Amount.Unit, targetUnit)
		}
		// Bridge: Rate -> Speed quantity (e.g., "60 km/h in mph")
		if units.IsSpeedUnit(targetUnit) {
			return interp.coerceRateToSpeed(rate, targetUnit)
		}
		return nil, fmt.Errorf("rate conversion requires a rate target (e.g., 'MB/s'), got '%s'; use '%s in %s/%s' or '%s per %s'",
			targetUnit, rate.String(), rate.Amount.Unit, targetUnit, rate.String(), targetUnit)
	}

	// Duration conversion: "1 day in seconds", "2 weeks in hours"
	if duration, ok := result.(*types.Duration); ok {
		converted, err := duration.Convert(targetUnit)
		if err != nil {
			return nil, fmt.Errorf("cannot convert duration: %w", err)
		}
		return converted, nil
	}

	// Fraction with unit: convert to Duration or Quantity before proceeding.
	// This handles "1/2 hour in minutes" and "1/4 cup in ml".
	if frac, ok := result.(*types.Fraction); ok && frac.Unit != "" {
		dec := fractionToNumber(frac)
		if types.IsValidDurationUnit(frac.Unit) {
			dur := &types.Duration{Value: dec.Value, Unit: frac.Unit}
			converted, err := dur.Convert(targetUnit)
			if err != nil {
				return nil, fmt.Errorf("cannot convert duration: %w", err)
			}
			return converted, nil
		}
		// Treat as quantity
		result = &types.Quantity{Value: dec.Value, Unit: frac.Unit}
	}

	// Standard quantity conversion
	qty, ok := result.(*types.Quantity)
	if !ok {
		return nil, fmt.Errorf("'in' conversion requires a quantity or duration, got %T", result)
	}

	// Use existing unit conversion logic
	converted, err := convertQuantity(qty, targetUnit)
	if err != nil {
		return nil, err
	}

	// Clone to avoid mutating the stored variable value, then mark as
	// explicit so convert_to transforms and auto-scaling skip this result.
	out := &types.Quantity{
		Value: converted.Value,
		Unit:  converted.Unit,
	}
	out.IsExplicit = true
	return out, nil
}

// evalCurrencyConversion converts a currency value to another currency.
// Requires an exchange rate to be defined in the frontmatter.
// Example: "100 USD in EUR" with exchange rate USD_EUR: 0.92 → €92.00
func (interp *Interpreter) evalCurrencyConversion(currency *types.Currency, targetCode string) (types.Type, error) {
	// Normalize the target currency code
	normalizedTarget := types.NormalizeCurrencyCode(targetCode)

	// Validate target looks like a currency code (3 uppercase letters)
	if !looksLikeCurrencyCode(normalizedTarget) {
		return nil, fmt.Errorf("'%s' is not a valid currency code; use ISO 4217 codes like USD, EUR, GBP", targetCode)
	}

	// Same currency - no conversion needed
	if currency.Code == normalizedTarget {
		return currency, nil
	}

	// Look up exchange rate
	rate, found := interp.env.GetExchangeRate(currency.Code, normalizedTarget)
	if !found {
		return nil, fmt.Errorf("no exchange rate defined for %s → %s; add to frontmatter: exchange:\n  %s_%s: <rate>",
			currency.Code, normalizedTarget, currency.Code, normalizedTarget)
	}

	// Convert the value
	convertedValue := currency.Value.Mul(rate)

	// Get the display symbol for the target currency
	targetSymbol := types.GetCurrencySymbol(normalizedTarget)

	return types.NewCurrency(convertedValue, targetSymbol), nil
}

// evalRateUnitConversion handles rate-to-rate conversion: "10 m/s in inch/s"
// Rules:
//   - Source must be a Rate
//   - Quantity units must be convertible (e.g., length-to-length)
//   - Time units must both be valid time units
func (interp *Interpreter) evalRateUnitConversion(result types.Type, targetUnit, targetTimeUnit string) (types.Type, error) {
	rate, ok := result.(*types.Rate)
	if !ok {
		return nil, fmt.Errorf("rate unit conversion (e.g., 'm/s in inch/s') requires a rate, got %T", result)
	}

	// Convert the quantity part (e.g., meters to inches)
	convertedAmount, err := convertQuantity(rate.Amount, targetUnit)
	if err != nil {
		return nil, fmt.Errorf("cannot convert rate quantity: %w", err)
	}

	// Normalize the target time unit
	normalizedTimeUnit := types.NormalizeTimeUnit(targetTimeUnit)

	// If source and target time units differ, we need to scale the amount
	sourceTimeUnit := rate.PerUnit
	if sourceTimeUnit != normalizedTimeUnit {
		// Get seconds per source time unit and target time unit
		sourceSeconds, err := types.TimeUnitToSeconds(sourceTimeUnit)
		if err != nil {
			return nil, fmt.Errorf("invalid source time unit '%s': %w", sourceTimeUnit, err)
		}
		targetSeconds, err := types.TimeUnitToSeconds(normalizedTimeUnit)
		if err != nil {
			return nil, fmt.Errorf("invalid target time unit '%s': %w", targetTimeUnit, err)
		}

		// Scale factor: if going from /s to /min, multiply by 60
		// If going from /min to /s, divide by 60
		scaleFactor := targetSeconds.Div(sourceSeconds)
		convertedAmount = &types.Quantity{
			Value: convertedAmount.Value.Mul(scaleFactor),
			Unit:  convertedAmount.Unit,
		}
	}

	return types.NewRate(convertedAmount, normalizedTimeUnit), nil
}

// looksLikeCurrencyCode checks if a string looks like a currency code (3 uppercase ASCII letters).
func looksLikeCurrencyCode(s string) bool {
	if len(s) != 3 {
		return false
	}
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}
