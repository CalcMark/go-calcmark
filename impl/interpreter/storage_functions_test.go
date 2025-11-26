package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

func TestCalculateRead(t *testing.T) {
	tests := []struct {
		name        string
		sizeValue   float64
		sizeUnit    string
		storageType string
		expectedSec float64
		tolerance   float64
		expectError bool
	}{
		{
			name:      "100 MB on SSD",
			sizeValue: 100, sizeUnit: "megabyte",
			storageType: "ssd",
			expectedSec: 0.182, // 100 MB / 550 MB/s ≈ 0.182s
			tolerance:   0.001,
		},
		{
			name:      "1 GB on NVMe",
			sizeValue: 1, sizeUnit: "gigabyte",
			storageType: "nvme",
			expectedSec: 0.293, // 1024 MB / 3500 MB/s ≈ 0.293s
			tolerance:   0.001,
		},
		{
			name:      "10 MB on HDD",
			sizeValue: 10, sizeUnit: "megabyte",
			storageType: "hdd",
			expectedSec: 0.067, // 10 MB / 150 MB/s ≈ 0.067s
			tolerance:   0.001,
		},
		{
			name:      "500 GB on PCIe SSD",
			sizeValue: 500, sizeUnit: "gigabyte",
			storageType: "pcie_ssd",
			expectedSec: 73.143, // 512000 MB / 7000 MB/s ≈ 73.14s
			tolerance:   1.0,
		},
		{
			name:      "Case insensitive - SSD",
			sizeValue: 1, sizeUnit: "megabyte",
			storageType: "SSD",
			expectedSec: 0.00182, // 1 MB / 550 MB/s
			tolerance:   0.00001,
		},
		{
			name:      "Whitespace trimming",
			sizeValue: 1, sizeUnit: "megabyte",
			storageType: "  nvme  ",
			expectedSec: 0.000286, // 1 MB / 3500 MB/s
			tolerance:   0.000001,
		},
		{
			name:      "Invalid storage type",
			sizeValue: 1, sizeUnit: "megabyte",
			storageType: "tape",
			expectError: true,
		},
		{
			name:      "Invalid size unit",
			sizeValue: 100, sizeUnit: "meters",
			storageType: "ssd",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := &types.Quantity{
				Value: decimal.NewFromFloat(tt.sizeValue),
				Unit:  tt.sizeUnit,
			}

			result, err := calculateRead(size, tt.storageType)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Convert result to seconds for comparison
			var resultSeconds float64
			switch result.Unit {
			case "second":
				resultSeconds, _ = result.Value.Float64()
			case "minute":
				min, _ := result.Value.Float64()
				resultSeconds = min * 60
			case "hour":
				hr, _ := result.Value.Float64()
				resultSeconds = hr * 3600
			default:
				t.Fatalf("Unexpected unit: %s", result.Unit)
			}

			diff := resultSeconds - tt.expectedSec
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerance {
				t.Errorf("Expected ~%v seconds, got %v seconds (diff: %v, tolerance: %v)",
					tt.expectedSec, resultSeconds, diff, tt.tolerance)
			}

			t.Logf("✓ %s: %v %s (%v seconds)", tt.name, result.Value, result.Unit, resultSeconds)
		})
	}
}

func TestCalculateSeek(t *testing.T) {
	tests := []struct {
		storageType string
		expectedMs  float64
		expectError bool
		description string
	}{
		{"hdd", 10.0, false, "HDD 7200 RPM"},
		{"ssd", 0.1, false, "SATA SSD"},
		{"sata_ssd", 0.1, false, "SATA SSD explicit"},
		{"nvme", 0.01, false, "NVMe"},
		{"pcie_ssd", 0.01, false, "PCIe Gen4"},
		{"HDD", 10.0, false, "Case insensitive"},
		{"  ssd  ", 0.1, false, "Whitespace trimming"},
		{"tape", 0, true, "Invalid storage type"},
		{"", 0, true, "Empty storage type"},
	}

	for _, tt := range tests {
		t.Run(tt.storageType, func(t *testing.T) {
			result, err := calculateSeek(tt.storageType)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for type '%s', got none", tt.storageType)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Unit != "second" {
				t.Errorf("Expected unit 'second', got '%s'", result.Unit)
			}

			gotSeconds, _ := result.Value.Float64()
			expectedSeconds := tt.expectedMs / 1000.0
			if gotSeconds != expectedSeconds {
				t.Errorf("Expected %v seconds, got %v seconds", expectedSeconds, gotSeconds)
			}

			t.Logf("✓ %s: %v seconds (%v ms)", tt.description, gotSeconds, tt.expectedMs)
		})
	}
}
