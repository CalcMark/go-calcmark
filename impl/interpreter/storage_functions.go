package interpreter

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// Storage throughput constants (consumer-grade, sequential read speeds)
// All values in MB/s (megabytes per second)
var storageThroughput = map[string]float64{
	"ssd":      550.0,  // Consumer SATA SSD: ~550 MB/s
	"sata_ssd": 550.0,  // SATA SSD (explicit): ~550 MB/s
	"nvme":     3500.0, // NVMe Gen3 SSD: ~3500 MB/s
	"pcie_ssd": 7000.0, // PCIe Gen4 SSD: ~7000 MB/s
	"hdd":      150.0,  // 7200 RPM HDD: ~150 MB/s
}

// Storage seek/access latency constants
// All values in milliseconds
var storageSeekTime = map[string]float64{
	"hdd":      10.0, // HDD 7200 RPM: 10 ms average seek
	"sata_ssd": 0.1,  // SATA SSD: 0.1 ms access latency
	"ssd":      0.1,  // Generic SSD: 0.1 ms
	"nvme":     0.01, // NVMe: 0.01 ms access latency
	"pcie_ssd": 0.01, // PCIe Gen4: 0.01 ms
}

// calculateRead returns the time to read data from storage.
// Time Complexity: O(1) - map lookup + arithmetic
//
// Examples:
//   - read(100 GB, ssd) → ~182 seconds
//   - read(1 TB, nvme) → ~293 seconds
//   - read(10 MB, hdd) → ~67 milliseconds
func calculateRead(size *types.Quantity, storageType string) (*types.Duration, error) {
	typeLower := strings.ToLower(strings.TrimSpace(storageType))

	// Look up throughput
	throughputMBPS, exists := storageThroughput[typeLower]
	if !exists {
		return nil, fmt.Errorf(
			"unknown storage type '%s' (valid types: ssd, sata_ssd, nvme, pcie_ssd, hdd)",
			storageType,
		)
	}

	// Convert size to megabytes
	// Note: Reusing the byte conversion helper from network functions
	sizeInMB, err := convertToMegabytes(size)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	// Calculate read time: size / throughput (in seconds)
	readSeconds := sizeInMB / throughputMBPS

	// Create duration
	duration, err := types.NewDuration(
		decimal.NewFromFloat(readSeconds),
		"second",
	)
	if err != nil {
		return nil, err
	}

	// Smart unit selection: < 1s → seconds, < 60s → seconds, >= 60s → minutes
	if readSeconds < 60 {
		return duration, nil // Keep in seconds
	}

	return duration.Convert("minute")
}

// calculateSeek returns the seek/access latency for a storage type.
// Time Complexity: O(1) - map lookup
//
// Examples:
//   - seek(hdd) → 10 millisecond
//   - seek(ssd) → 0.1 millisecond
//   - seek(nvme) → 0.01 millisecond
func calculateSeek(storageType string) (*types.Duration, error) {
	typeLower := strings.ToLower(strings.TrimSpace(storageType))

	latencyMs, exists := storageSeekTime[typeLower]
	if !exists {
		return nil, fmt.Errorf(
			"unknown storage type '%s' (valid types: hdd, sata_ssd, ssd, nvme, pcie_ssd)",
			storageType,
		)
	}

	// Convert milliseconds to seconds for Duration
	latencySeconds := latencyMs / 1000.0

	return types.NewDuration(
		decimal.NewFromFloat(latencySeconds),
		"second",
	)
}
