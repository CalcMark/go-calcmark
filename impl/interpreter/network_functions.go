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
	"gigabit": 125.0,   // 1 Gbps = 125 MB/s
	"10g":     1250.0,  // 10 Gbps = 1.25 GB/s
	"100g":    12500.0, // 100 Gbps = 12.5 GB/s
	"wifi":    12.5,    // ~100 Mbps typical WiFi
	"4g":      2.5,     // ~20 Mbps typical 4G
	"5g":      50.0,    // ~400 Mbps typical 5G
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
// Network types: gigabit, 10g, 100g, wifi, 4g, 5g
// Time Complexity: O(1) - map lookup
//
// Examples:
//   - throughput(gigabit) → 125 MB/s
//   - throughput(10g) → 1250 MB/s
//   - throughput(wifi) → 12.5 MB/s
func calculateThroughput(networkType string) (*types.Rate, error) {
	typeLower := strings.ToLower(strings.TrimSpace(networkType))

	mbps, exists := networkThroughput[typeLower]
	if !exists {
		return nil, fmt.Errorf(
			"unknown network type '%s' (valid types: gigabit, 10g, 100g, wifi, 4g, 5g)",
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
//   - transfer_time(10 MB, local, 10g) → ~8.5 ms (0.5ms RTT + 8ms transmission)
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
	sizeInMB, err := convertQuantityToMegabytes(size)
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

// convertQuantityToMegabytes converts a quantity to megabytes (float64).
// Handles byte-based units: byte, kilobyte, megabyte, gigabyte, etc.
// Time Complexity: O(1) - unit conversion
func convertQuantityToMegabytes(q *types.Quantity) (float64, error) {
	// Normalize unit name
	unitLower := strings.ToLower(q.Unit)

	// Conversion factors to bytes
	bytesPerUnit := map[string]float64{
		"byte":      1,
		"bytes":     1,
		"b":         1,
		"kilobyte":  1024,
		"kilobytes": 1024,
		"kb":        1024,
		"megabyte":  1024 * 1024,
		"megabytes": 1024 * 1024,
		"mb":        1024 * 1024,
		"gigabyte":  1024 * 1024 * 1024,
		"gigabytes": 1024 * 1024 * 1024,
		"gb":        1024 * 1024 * 1024,
		"terabyte":  1024 * 1024 * 1024 * 1024,
		"terabytes": 1024 * 1024 * 1024 * 1024,
		"tb":        1024 * 1024 * 1024 * 1024,
	}

	multiplier, ok := bytesPerUnit[unitLower]
	if !ok {
		return 0, fmt.Errorf("cannot convert '%s' to bytes (expected byte-based unit)", q.Unit)
	}

	// Convert to bytes, then to megabytes
	bytes := q.Value.InexactFloat64() * multiplier
	megabytes := bytes / (1024 * 1024)

	return megabytes, nil
}
