package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/CalcMark/go-calcmark/spec/units"
)

// coerceSpeedToRate converts a Speed quantity (e.g., "60 mph") to a Rate (e.g., 60 mi/h).
// Uses the speed decomposition mapping to determine numerator and time units.
func (interp *Interpreter) coerceSpeedToRate(qty *types.Quantity) (*types.Rate, error) {
	numUnit, timeUnit, ok := units.DecomposeSpeedUnit(qty.Unit)
	if !ok {
		return nil, fmt.Errorf("cannot decompose speed unit '%s' into rate components", qty.Unit)
	}

	amount := &types.Quantity{Value: qty.Value, Unit: numUnit}
	return types.NewRate(amount, timeUnit), nil
}

// coerceRateToSpeed converts a Rate (e.g., 60 km/h) to a Speed quantity (e.g., "60 kph").
// First converts the rate to match the target speed unit's decomposition, then wraps as quantity.
func (interp *Interpreter) coerceRateToSpeed(rate *types.Rate, targetSpeedUnit string) (types.Type, error) {
	numUnit, timeUnit, ok := units.DecomposeSpeedUnit(targetSpeedUnit)
	if !ok {
		return nil, fmt.Errorf("cannot decompose target speed unit '%s'", targetSpeedUnit)
	}

	// Convert the rate to match the target's decomposition
	// First convert the quantity part (e.g., km -> mi)
	convertedAmount, err := convertQuantity(rate.Amount, numUnit)
	if err != nil {
		return nil, fmt.Errorf("cannot convert rate for speed bridge: %w", err)
	}

	// Then scale for time unit differences (e.g., /h -> /s)
	normalizedTimeUnit := types.NormalizeTimeUnit(timeUnit)
	if rate.PerUnit != normalizedTimeUnit {
		sourceSeconds, err := types.TimeUnitToSeconds(rate.PerUnit)
		if err != nil {
			return nil, fmt.Errorf("invalid rate time unit '%s': %w", rate.PerUnit, err)
		}
		targetSeconds, err := types.TimeUnitToSeconds(normalizedTimeUnit)
		if err != nil {
			return nil, fmt.Errorf("invalid target time unit '%s': %w", timeUnit, err)
		}
		scaleFactor := targetSeconds.Div(sourceSeconds)
		convertedAmount = &types.Quantity{
			Value: convertedAmount.Value.Mul(scaleFactor),
			Unit:  convertedAmount.Unit,
		}
	}

	out := &types.Quantity{
		Value: convertedAmount.Value,
		Unit:  targetSpeedUnit,
	}
	out.IsExplicit = true
	return out, nil
}

// speedTimesDuration handles Speed × Duration → distance (e.g., "60 mph * 2 hours").
// Decomposes the speed unit, creates a rate, then delegates to accumulateRate.
func speedTimesDuration(speed *types.Quantity, dur *types.Duration) (types.Type, error) {
	numUnit, timeUnit, ok := units.DecomposeSpeedUnit(speed.Unit)
	if !ok {
		return nil, fmt.Errorf("cannot decompose speed unit '%s' for multiplication", speed.Unit)
	}

	amount := &types.Quantity{Value: speed.Value, Unit: numUnit}
	rate := types.NewRate(amount, timeUnit)

	return accumulateRate(rate, dur.Value, dur.Unit)
}
