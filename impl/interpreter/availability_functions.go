package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// calculateDowntime converts an availability percentage to downtime duration.
// Examples:
//   - 99.9% availability over 1 month → 43.2 minutes downtime
//   - 99.99% availability over 1 year → 52.56 minutes downtime
func calculateDowntime(availability, timePeriod types.Type) (*types.Duration, error) {
	// Extract availability percentage (stored as decimal fraction: 99.9% = 0.999)
	var availabilityFraction decimal.Decimal

	switch avail := availability.(type) {
	case *types.Number:
		availabilityFraction = avail.Value
	default:
		return nil, fmt.Errorf("downtime() availability must be a percentage, got %T", availability)
	}

	// Validate availability is in valid range (0-1 for decimal fractions)
	if availabilityFraction.LessThan(decimal.Zero) || availabilityFraction.GreaterThan(decimal.NewFromInt(1)) {
		return nil, fmt.Errorf("downtime() availability must be between 0%% and 100%%, got %s", availabilityFraction.Mul(decimal.NewFromInt(100)))
	}

	// Calculate downtime fraction (1 - availability)
	downtimeFraction := decimal.NewFromInt(1).Sub(availabilityFraction)

	// Get time period in seconds
	var periodSeconds decimal.Decimal

	switch period := timePeriod.(type) {
	case *types.Duration:
		// Duration - convert to seconds using the map
		secondsPerUnit, ok := types.DurationToSeconds[period.Unit]
		if !ok {
			return nil, fmt.Errorf("unknown duration unit: %s", period.Unit)
		}
		periodSeconds = period.Value.Mul(decimal.NewFromInt(secondsPerUnit))

	case *types.Quantity:
		// Time quantity (e.g., "1 month", "30 days")
		secondsPerUnit, ok := types.DurationToSeconds[period.Unit]
		if !ok {
			return nil, fmt.Errorf("downtime() time period must be a time unit, got %s", period.Unit)
		}
		periodSeconds = period.Value.Mul(decimal.NewFromInt(secondsPerUnit))

	case *ast.Identifier:
		// Bare identifier like "month", "year", etc.
		secondsPerUnit, ok := types.DurationToSeconds[period.Name]
		if !ok {
			return nil, fmt.Errorf("downtime() time period must be a time unit, got %s", period.Name)
		}
		periodSeconds = decimal.NewFromInt(1).Mul(decimal.NewFromInt(secondsPerUnit))

	default:
		return nil, fmt.Errorf("downtime() time period must be a duration or time unit, got %T", timePeriod)
	}

	// Calculate downtime in seconds
	downtimeSeconds := periodSeconds.Mul(downtimeFraction)

	// Choose appropriate unit based on magnitude
	// < 60s → seconds, < 3600s → minutes, >= 3600s → hours
	unit := "second"
	value := downtimeSeconds

	seconds, _ := downtimeSeconds.Float64()
	if seconds >= 3600 {
		// Convert to hours
		unit = "hour"
		value = downtimeSeconds.Div(decimal.NewFromInt(3600))
	} else if seconds >= 60 {
		// Convert to minutes
		unit = "minute"
		value = downtimeSeconds.Div(decimal.NewFromInt(60))
	}

	return types.NewDuration(value, unit)
}
