package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// evalUnitConversion evaluates explicit unit conversion: "10 meters in feet"
// Also handles rate-to-rate conversion: "10 m/s in inch/s"
// Also handles currency conversion: "100 USD in EUR" (requires exchange rate in frontmatter)
func (interp *Interpreter) evalUnitConversion(u *ast.UnitConversion) (types.Type, error) {
	// Evaluate the quantity expression
	result, err := interp.evalNode(u.Quantity)
	if err != nil {
		return nil, err
	}

	// Check if this is currency conversion
	if currency, ok := result.(*types.Currency); ok {
		return interp.evalCurrencyConversion(currency, u.TargetUnit)
	}

	// Check if this is a rate-to-rate conversion
	if u.TargetTimeUnit != "" {
		return interp.evalRateUnitConversion(result, u.TargetUnit, u.TargetTimeUnit)
	}

	// Check if source is a rate and target is a time unit
	// This handles "10 MB/day in seconds" -> keep MB, convert day to seconds
	if rate, ok := result.(*types.Rate); ok {
		if types.IsTimeUnit(u.TargetUnit) {
			// Rate with time-only target: keep the quantity unit, convert time
			return interp.evalRateUnitConversion(result, rate.Amount.Unit, u.TargetUnit)
		}
		return nil, fmt.Errorf("rate conversion requires a rate target (e.g., 'MB/s'), got '%s'; use '%s in %s/%s' or '%s per %s'",
			u.TargetUnit, rate.String(), rate.Amount.Unit, u.TargetUnit, rate.String(), u.TargetUnit)
	}

	// Standard quantity conversion
	qty, ok := result.(*types.Quantity)
	if !ok {
		return nil, fmt.Errorf("'in' conversion requires a quantity, got %T", result)
	}

	// Use existing unit conversion logic
	converted, err := convertQuantity(qty, u.TargetUnit)
	if err != nil {
		return nil, err
	}

	return converted, nil
}

// evalCurrencyConversion converts a currency value to another currency.
// Requires an exchange rate to be defined in the frontmatter.
// Example: "100 USD in EUR" with exchange rate USD_EUR: 0.92 → €92.00
func (interp *Interpreter) evalCurrencyConversion(currency *types.Currency, targetCode string) (types.Type, error) {
	// Normalize the target currency code
	normalizedTarget := types.NormalizeCurrencyCode(targetCode)

	// Same currency - no conversion needed
	if currency.Code == normalizedTarget {
		return currency, nil
	}

	// Look up exchange rate
	rate, found := interp.env.GetExchangeRate(currency.Code, normalizedTarget)
	if !found {
		return nil, fmt.Errorf("no exchange rate defined for %s → %s; add to frontmatter: exchange: { %s/%s: <rate> }",
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
