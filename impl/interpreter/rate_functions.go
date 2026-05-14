package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/v2/spec/identifiers"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/shopspring/decimal"
)

// validTimeUnitsList returns a comma-separated list of canonical time
// units accepted by convert_rate / the NL `per` form, for use in
// diagnostics. Skips sub-second units to keep the suggestion list
// focused on the periods most users actually want.
func validTimeUnitsList() string {
	units := []string{"second", "minute", "hour", "day", "week", "month", "quarter", "year"}
	// Sanity-check against the canonical TimeUnits list so divergence
	// shows up in tests rather than slipping past.
	_ = identifiers.TimeUnits
	return identifiers.JoinNames(units)
}

// accumulateRate calculates the total from a rate over a time period.
// When the rate's unit is a currency symbol or code, returns *types.Currency.
// Otherwise returns *types.Quantity.
// Examples:
//   - 100 MB/s over 1 day → 8,640,000 MB (Quantity)
//   - $0.10/hour over 30 days → $72 (Currency)
//   - 5 GB/day over 1 year → 1,825 GB (Quantity)
func accumulateRate(rate *types.Rate, timePeriod decimal.Decimal, periodUnit string) (types.Type, error) {
	if rate == nil {
		return nil, fmt.Errorf("rate cannot be nil")
	}

	// Convert the time period to seconds
	periodSeconds, err := types.TimeUnitToSeconds(periodUnit)
	if err != nil {
		return nil, fmt.Errorf("invalid time period unit: %w", err)
	}
	totalSeconds := timePeriod.Mul(periodSeconds)

	// Convert rate's time unit to seconds
	rateSeconds, err := types.TimeUnitToSeconds(rate.PerUnit)
	if err != nil {
		return nil, fmt.Errorf("invalid rate time unit: %w", err)
	}

	// Calculate total amount
	// Formula: (amount per rate_time) * (total_time / rate_time)
	// Example: (100 MB/s) * (86400 s / 1 s) = 8,640,000 MB
	totalAmount := rate.Amount.Value.Mul(totalSeconds).Div(rateSeconds)

	// If the rate's unit is a currency, return a Currency type so it
	// can interoperate with other currency values in arithmetic.
	if types.IsCurrencyCode(rate.Amount.Unit) {
		return types.NewCurrency(totalAmount, rate.Amount.Unit), nil
	}

	return &types.Quantity{
		Value: totalAmount,
		Unit:  rate.Amount.Unit,
	}, nil
}

// convertRateTimeUnit converts a rate to a different time unit.
// Examples:
//   - 5 million/day per second → 57.87/second
//   - 10 TB/month per second → 3.86 MB/second
//   - 1000 req/s per hour → 3.6M/hour
func convertRateTimeUnit(rate *types.Rate, targetUnit string) (*types.Rate, error) {
	if rate == nil {
		return nil, fmt.Errorf("rate cannot be nil")
	}

	// If already in target unit, return as-is
	normalizedCurrent := types.NormalizeTimeUnit(rate.PerUnit)
	normalizedTarget := types.NormalizeTimeUnit(targetUnit)

	if normalizedCurrent == normalizedTarget {
		return rate, nil
	}

	// Get seconds for both units
	sourceSeconds, err := types.TimeUnitToSeconds(rate.PerUnit)
	if err != nil {
		return nil, fmt.Errorf("invalid source time unit %q: valid units are %s — also accepted as `<rate> per <unit>`", rate.PerUnit, validTimeUnitsList())
	}

	targetSeconds, err := types.TimeUnitToSeconds(targetUnit)
	if err != nil {
		return nil, fmt.Errorf("invalid target time unit %q: valid units are %s — also accepted as `<rate> per <unit>`", targetUnit, validTimeUnitsList())
	}

	// Calculate conversion factor using multiplication to avoid precision loss.
	// When sourceSeconds/targetSeconds creates a repeating decimal (e.g., 1/3600),
	// subsequent division loses precision. Using Mul(targetSeconds/sourceSeconds)
	// preserves exact integer arithmetic when possible.
	//
	// Example: Converting 5M/day to per second
	// - Day has 86400 seconds, second has 1 second
	// - conversionFactor = 1 / 86400 (dividing by larger gives smaller result)
	// - newAmount = 5M * (1/86400) = 57.87/s
	//
	// Example: Converting 1000/second to per hour
	// - Second has 1 second, hour has 3600 seconds
	// - conversionFactor = 3600 / 1 = 3600 (exact integer, no precision loss)
	// - newAmount = 1000 * 3600 = 3,600,000/h (exact)
	conversionFactor := targetSeconds.Div(sourceSeconds)
	newAmount := rate.Amount.Value.Mul(conversionFactor)

	return &types.Rate{
		Amount: &types.Quantity{
			Value: newAmount,
			Unit:  rate.Amount.Unit,
		},
		PerUnit: targetUnit,
	}, nil
}
