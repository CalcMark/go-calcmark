package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// accumulateRate calculates the total quantity from a rate over a time period.
// Examples:
//   - 100 MB/s over 1 day → 8.64 TB
//   - $0.10/hour over 30 days → $72
//   - 5 GB/day over 1 year → 1.825 TB
func accumulateRate(rate *types.Rate, timePeriod decimal.Decimal, periodUnit string) (*types.Quantity, error) {
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
		return nil, fmt.Errorf("invalid source time unit: %w", err)
	}

	targetSeconds, err := types.TimeUnitToSeconds(targetUnit)
	if err != nil {
		return nil, fmt.Errorf("invalid target time unit: %w", err)
	}

	// Calculate conversion ratio
	// Example: Converting 5M/day to per second
	// - Day has 86400 seconds
	// - Want amount per 1 second instead of per 86400 seconds
	// - So divide by ratio: 5M / 86400 = 57.87
	//
	// Example: Converting 1000/second to per hour
	// - Second has 1 second, hour has 3600 seconds
	// - Want amount per 3600 seconds instead of per 1 second
	// - So multiply by ratio: 1000 * 3600 = 3,600,000
	ratio := sourceSeconds.Div(targetSeconds)
	newAmount := rate.Amount.Value.Div(ratio)

	return &types.Rate{
		Amount: &types.Quantity{
			Value: newAmount,
			Unit:  rate.Amount.Unit,
		},
		PerUnit: targetUnit,
	}, nil
}
