package interpreter

import (
	"fmt"
	"math"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// capacityAt calculates how many units of capacity are needed for a given demand.
// Returns a Quantity with the user-specified unit (e.g., "5 disks", "23 servers").
//
// Examples:
//   - capacityAt(10 TB, 2 TB, "disk") → 5 disks
//   - capacityAt(10000 req/s, 450 req/s, "server") → 23 servers
//   - capacityAt(100 apples, 30, "crate") → 4 crates
func capacityAt(demand, capacity types.Type, unit string) (types.Type, error) {
	return capacityAtWithBuffer(demand, capacity, unit, decimal.Zero)
}

// capacityAtWithBuffer calculates capacity needed with an optional buffer percentage.
// Returns a Quantity with the user-specified unit.
//
// When demand and capacity have different units (e.g., 10 GB vs 2 PB), the function
// normalizes them to the same unit before division.
//
// Examples:
//   - capacityAtWithBuffer(10 TB, 2 TB, "disk", 0.10) → 6 disks  (⌈5.5⌉)
//   - capacityAtWithBuffer(10 GB, 2 PB, "disk", 0.10) → 1 disk   (GB converted to PB)
//   - capacityAtWithBuffer(10000 req/s, 450 req/s, "server", 0.20) → 27 servers
func capacityAtWithBuffer(demand, capacity types.Type, unit string, bufferPercent decimal.Decimal) (types.Type, error) {
	// Extract and normalize values from demand and capacity
	var demandValue, capacityValue decimal.Decimal

	switch d := demand.(type) {
	case *types.Number:
		demandValue = d.Value
		// For numbers, just get capacity value directly
		switch c := capacity.(type) {
		case *types.Number:
			capacityValue = c.Value
		case *types.Quantity:
			capacityValue = c.Value
		case *types.Rate:
			capacityValue = c.Amount.Value
		default:
			return nil, fmt.Errorf("capacity() capacity must be a number, quantity, or rate, got %T", capacity)
		}
	case *types.Quantity:
		// For quantities, convert to capacity's unit if capacity is also a quantity
		switch c := capacity.(type) {
		case *types.Number:
			demandValue = d.Value
			capacityValue = c.Value
		case *types.Quantity:
			// Convert demand to capacity's unit for compatible comparison
			converted, err := convertQuantity(d, c.Unit)
			if err != nil {
				// If conversion fails, fall back to raw values (arbitrary units)
				demandValue = d.Value
				capacityValue = c.Value
			} else {
				demandValue = converted.Value
				capacityValue = c.Value
			}
		case *types.Rate:
			// Quantity vs Rate - use raw amount
			demandValue = d.Value
			capacityValue = c.Amount.Value
		default:
			return nil, fmt.Errorf("capacity() capacity must be a number, quantity, or rate, got %T", capacity)
		}
	case *types.Rate:
		// For rates, normalize both amount units and time units
		switch c := capacity.(type) {
		case *types.Number:
			demandValue = d.Amount.Value
			capacityValue = c.Value
		case *types.Quantity:
			// Special case: capacity might be a throughput unit like Mbps, Gbps
			// These are effectively rates (megabits per second) expressed as quantities
			if isRateUnit(c.Unit) {
				// Convert demand rate to the same throughput unit
				demandValue, capacityValue = normalizeRateToThroughput(d, c)
			} else {
				// Try to convert demand amount to capacity's unit
				converted, err := convertQuantity(d.Amount, c.Unit)
				if err != nil {
					demandValue = d.Amount.Value
					capacityValue = c.Value
				} else {
					demandValue = converted.Value
					capacityValue = c.Value
				}
			}
		case *types.Rate:
			// Both are rates - need to normalize units
			demandValue, capacityValue = normalizeRateValues(d, c)
		default:
			return nil, fmt.Errorf("capacity() capacity must be a number, quantity, or rate, got %T", capacity)
		}
	default:
		return nil, fmt.Errorf("capacity() demand must be a number, quantity, or rate, got %T", demand)
	}

	// Validate capacity is not zero
	if capacityValue.IsZero() {
		return nil, fmt.Errorf("capacity() cannot divide by zero capacity")
	}

	// Validate capacity is positive
	if capacityValue.IsNegative() {
		return nil, fmt.Errorf("capacity() capacity must be positive")
	}

	// Validate buffer percentage (only negative is invalid)
	if bufferPercent.IsNegative() {
		return nil, fmt.Errorf("capacity() buffer percentage cannot be negative")
	}

	// Buffer is a decimal fraction (0.20 = 20%, 1.0 = 100%, 1.2 = 120%)
	bufferMultiplier := decimal.NewFromInt(1).Add(bufferPercent)
	adjustedDemand := demandValue.Mul(bufferMultiplier)

	// Divide: raw_result = adjusted_demand ÷ capacity
	rawResult := adjustedDemand.Div(capacityValue)

	// Apply ceiling
	f, _ := rawResult.Float64()
	result := decimal.NewFromFloat(math.Ceil(f))

	// Return a Quantity with the user-specified unit
	return types.NewQuantity(result, unit), nil
}

// normalizeRateValues normalizes two rates to comparable values.
// Converts demand rate's amount unit to capacity rate's amount unit,
// and normalizes time units if they differ.
func normalizeRateValues(demand, capacity *types.Rate) (demandValue, capacityValue decimal.Decimal) {
	// Try to convert demand's amount to capacity's amount unit
	if demand.Amount.Unit != "" && capacity.Amount.Unit != "" {
		converted, err := convertQuantity(demand.Amount, capacity.Amount.Unit)
		if err == nil {
			demandValue = converted.Value
		} else {
			demandValue = demand.Amount.Value
		}
	} else {
		demandValue = demand.Amount.Value
	}
	capacityValue = capacity.Amount.Value

	// Normalize time units if they differ
	// Common time units: s, sec, second, min, minute, h, hour, day
	if demand.PerUnit != capacity.PerUnit {
		demandSeconds := timeUnitToSeconds(demand.PerUnit)
		capacitySeconds := timeUnitToSeconds(capacity.PerUnit)

		if demandSeconds > 0 && capacitySeconds > 0 {
			// Scale to common time base (per second)
			demandValue = demandValue.Div(decimal.NewFromFloat(demandSeconds))
			capacityValue = capacityValue.Div(decimal.NewFromFloat(capacitySeconds))
		}
	}

	return demandValue, capacityValue
}

// timeUnitToSeconds converts a time unit string to seconds.
// Returns 0 if the unit is not recognized.
func timeUnitToSeconds(unit string) float64 {
	switch unit {
	case "s", "sec", "second", "seconds":
		return 1
	case "min", "minute", "minutes":
		return 60
	case "h", "hr", "hour", "hours":
		return 3600
	case "day", "days":
		return 86400
	case "week", "weeks":
		return 604800
	case "month", "months":
		return 2592000 // 30 days
	case "year", "years":
		return 31536000 // 365 days
	default:
		return 0
	}
}

// isRateUnit checks if a unit is a throughput/rate unit like Mbps, Gbps
// These are compound units that represent data/time (e.g., megabits per second)
func isRateUnit(unit string) bool {
	lower := strings.ToLower(unit)
	switch lower {
	case "bps", "kbps", "mbps", "gbps", "tbps":
		return true
	default:
		return false
	}
}

// normalizeRateToThroughput converts a Rate (like "10 GB/s") to compare with
// a throughput unit Quantity (like "100 Mbps").
//
// Uses the unit registry to convert both values to a common base (bits),
// then normalizes for time units.
//
// Example: 10 GB/s vs 100 Mbps
//   - 10 GB (binary) = 10 × 1024³ × 8 bits = 85,899,345,920 bits
//   - Per second: 85,899,345,920 bps
//   - 100 Mbps = 100 × 10^6 bits/s = 100,000,000 bps
//   - Result: 85,899,345,920 / 100,000,000 ≈ 859
func normalizeRateToThroughput(rate *types.Rate, throughput *types.Quantity) (demandValue, capacityValue decimal.Decimal) {
	// Convert rate's amount to bits (base unit) using the registry
	var amountInBits float64
	if rate.Amount.Unit != "" {
		info, ok := GetUnitInfo(strings.ToLower(rate.Amount.Unit))
		if ok && info.Category == CategoryDataSize {
			val, _ := rate.Amount.Value.Float64()
			amountInBits = info.ToBaseUnit(val)
		} else {
			// Unknown unit - just use raw value
			amountInBits, _ = rate.Amount.Value.Float64()
		}
	} else {
		amountInBits, _ = rate.Amount.Value.Float64()
	}

	// Normalize for time unit (rate is per PerUnit)
	perSecondMultiplier := timeUnitToSeconds(rate.PerUnit)
	if perSecondMultiplier == 0 {
		perSecondMultiplier = 1 // Default to per-second if unknown
	}

	// Amount in bits per second
	bitsPerSecond := amountInBits / perSecondMultiplier

	// Convert throughput to bits per second using the registry
	var throughputBitsPerSecond float64
	throughputInfo, ok := GetUnitInfo(strings.ToLower(throughput.Unit))
	if ok && throughputInfo.Category == CategoryDataSize {
		val, _ := throughput.Value.Float64()
		throughputBitsPerSecond = throughputInfo.ToBaseUnit(val)
	} else {
		// Unknown unit - just use raw value
		throughputBitsPerSecond, _ = throughput.Value.Float64()
	}

	// Return normalized values (both in bits per second)
	demandValue = decimal.NewFromFloat(bitsPerSecond)
	capacityValue = decimal.NewFromFloat(throughputBitsPerSecond)

	return demandValue, capacityValue
}
