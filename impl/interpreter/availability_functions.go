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

	// Convert time period to a Duration
	var periodDuration *types.Duration

	switch period := timePeriod.(type) {
	case *types.Duration:
		periodDuration = period

	case *types.Quantity:
		// Time quantity (e.g., "1 month", "30 days") - treat as duration
		d, err := types.NewDuration(period.Value, period.Unit)
		if err != nil {
			return nil, fmt.Errorf("downtime() time period: %w", err)
		}
		periodDuration = d

	case *ast.Identifier:
		// Bare identifier like "month", "year" - treat as 1 unit
		d, err := types.NewDuration(decimal.NewFromInt(1), period.Name)
		if err != nil {
			return nil, fmt.Errorf("downtime() time period must be a time unit, got %s", period.Name)
		}
		periodDuration = d

	default:
		return nil, fmt.Errorf("downtime() time period must be a duration or time unit, got %T", timePeriod)
	}

	// Calculate downtime in seconds using existing Duration.ToSeconds()
	periodSeconds := periodDuration.ToSeconds()
	downtimeSeconds := periodSeconds.Mul(downtimeFraction)

	// Create duration in seconds first
	downtimeDuration, err := types.NewDuration(downtimeSeconds, "second")
	if err != nil {
		return nil, err
	}

	// Choose appropriate unit and convert using existing Duration.Convert()
	// < 60s → seconds, < 3600s → minutes, >= 3600s → hours
	seconds, _ := downtimeSeconds.Float64()
	targetUnit := "second"
	if seconds >= 3600 {
		targetUnit = "hour"
	} else if seconds >= 60 {
		targetUnit = "minute"
	}

	return downtimeDuration.Convert(targetUnit)
}
