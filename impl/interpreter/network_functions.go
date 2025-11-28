package interpreter

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// Network latency constants based on Jeff Dean's numbers (updated for modern networks)
// All values in milliseconds
var networkLatencies = map[string]float64{
	"local":       0.5,   // Same datacenter: 0.5 ms
	"regional":    10.0,  // Same region (within ~500 km): 10 ms
	"continental": 50.0,  // Cross-continent (~3000-5000 km): 50 ms
	"global":      150.0, // Global (~10000+ km): 150 ms (speed of light in fiber)
}

// Network throughput constants
// All values in MB/s (megabytes per second)
var networkThroughput = map[string]float64{
	"gigabit":     125.0,   // 1 Gbps = 125 MB/s
	"ten_gig":     1250.0,  // 10 Gbps = 1.25 GB/s
	"hundred_gig": 12500.0, // 100 Gbps = 12.5 GB/s
	"wifi":        12.5,    // ~100 Mbps typical WiFi
	"four_g":      2.5,     // ~20 Mbps typical 4G
	"five_g":      50.0,    // ~400 Mbps typical 5G
}

// calculateRTT returns network round-trip time as a Duration.
// Scope options: local, regional, continental, global
// Time Complexity: O(1) - map lookup
//
// Examples:
//   - rtt(local) → 0.5 millisecond
//   - rtt(regional) → 10 millisecond
//   - rtt(global) → 150 millisecond
func calculateRTT(scope string) (*types.Duration, error) {
	scopeLower := strings.ToLower(strings.TrimSpace(scope))

	latencyMs, exists := networkLatencies[scopeLower]
	if !exists {
		return nil, fmt.Errorf(
			"unknown network scope '%s' (valid scopes: local, regional, continental, global)",
			scope,
		)
	}

	// Convert milliseconds to seconds for Duration
	latencySeconds := latencyMs / 1000.0

	return types.NewDuration(
		decimal.NewFromFloat(latencySeconds),
		"second",
	)
}

// calculateThroughput returns network bandwidth as a Rate (MB/s).
// Network types: gigabit, ten_gig, hundred_gig, wifi, four_g, five_g
// Time Complexity: O(1) - map lookup
//
// Examples:
//   - throughput(gigabit) → 125 MB/s
//   - throughput(ten_gig) → 1250 MB/s
//   - throughput(wifi) → 12.5 MB/s
func calculateThroughput(networkType string) (*types.Rate, error) {
	typeLower := strings.ToLower(strings.TrimSpace(networkType))

	mbps, exists := networkThroughput[typeLower]
	if !exists {
		return nil, fmt.Errorf(
			"unknown network type '%s' (valid types: gigabit, ten_gig, hundred_gig, wifi, four_g, five_g)",
			networkType,
		)
	}

	// Return as MB/s rate
	return types.NewRate(
		&types.Quantity{
			Value: decimal.NewFromFloat(mbps),
			Unit:  "megabyte",
		},
		"second",
	), nil
}

// calculateTransferTime computes total data transfer time: RTT + (size / bandwidth)
// Time Complexity: O(1) - map lookups + arithmetic + unit conversion
//
// Examples:
//   - transfer_time(1 KB, regional, gigabit) → ~10 ms (RTT dominates)
//   - transfer_time(1 GB, global, gigabit) → ~8.15 seconds (150ms RTT + 8s transmission)
//   - transfer_time(10 MB, local, ten_gig) → ~8.5 ms (0.5ms RTT + 8ms transmission)
func calculateTransferTime(size *types.Quantity, scope string, networkType string) (*types.Duration, error) {
	// Get RTT
	rtt, err := calculateRTT(scope)
	if err != nil {
		return nil, fmt.Errorf("transfer_time: %w", err)
	}

	// Get throughput
	throughput, err := calculateThroughput(networkType)
	if err != nil {
		return nil, fmt.Errorf("transfer_time: %w", err)
	}

	// Convert size to megabytes for calculation
	// Note: Using custom conversion since unit library doesn't include byte units
	sizeInMB, err := convertToMegabytes(size)
	if err != nil {
		return nil, fmt.Errorf("transfer_time: %w", err)
	}

	// Calculate transmission time: size / throughput (in seconds)
	throughputMBPS := throughput.Amount.Value.InexactFloat64() // MB/s
	transmissionSeconds := sizeInMB / throughputMBPS

	// Total = RTT + transmission time (both in seconds)
	rttSeconds := rtt.Value.InexactFloat64()
	totalSeconds := rttSeconds + transmissionSeconds

	// Create duration
	duration, err := types.NewDuration(
		decimal.NewFromFloat(totalSeconds),
		"second",
	)
	if err != nil {
		return nil, err
	}

	// Smart unit selection: < 60s → keep in seconds, >= 60s → convert to minutes
	if totalSeconds < 60 {
		return duration, nil // Keep in seconds
	}

	return duration.Convert("minute")
}

// convertToMegabytes converts byte-based quantities to megabytes.
// The main unit library doesn't include byte units, so we handle them specially here.
// Time Complexity: O(1) - direct conversion factors
func convertToMegabytes(q *types.Quantity) (float64, error) {
	unitLower := strings.ToLower(q.Unit)

	// Conversion: unit → bytes → megabytes
	var bytesPerUnit float64
	switch unitLower {
	case "byte", "bytes", "b":
		bytesPerUnit = 1
	case "kilobyte", "kilobytes", "kb":
		bytesPerUnit = 1024
	case "megabyte", "megabytes", "mb":
		bytesPerUnit = 1024 * 1024
	case "gigabyte", "gigabytes", "gb":
		bytesPerUnit = 1024 * 1024 * 1024
	case "terabyte", "terabytes", "tb":
		bytesPerUnit = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("not a byte-based unit: %s", q.Unit)
	}

	bytes := q.Value.InexactFloat64() * bytesPerUnit
	return bytes / (1024 * 1024), nil
}
