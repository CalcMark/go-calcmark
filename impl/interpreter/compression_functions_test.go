package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

func TestCalculateCompression(t *testing.T) {
	tests := []struct {
		name            string
		sizeValue       float64
		sizeUnit        string
		compressionType string
		expectedValue   float64
		tolerance       float64
		expectError     bool
	}{
		{
			name:      "1 GB gzip",
			sizeValue: 1, sizeUnit: "gigabyte",
			compressionType: "gzip",
			expectedValue:   0.333, // 1/3
			tolerance:       0.01,
		},
		{
			name:      "100 MB lz4",
			sizeValue: 100, sizeUnit: "megabyte",
			compressionType: "lz4",
			expectedValue:   50, // 100/2
			tolerance:       0.01,
		},
		{
			name:      "500 MB zstd",
			sizeValue: 500, sizeUnit: "megabyte",
			compressionType: "zstd",
			expectedValue:   142.857, // 500/3.5
			tolerance:       0.01,
		},
		{
			name:      "1000 MB bzip2",
			sizeValue: 1000, sizeUnit: "megabyte",
			compressionType: "bzip2",
			expectedValue:   250, // 1000/4
			tolerance:       0.01,
		},
		{
			name:      "200 MB none",
			sizeValue: 200, sizeUnit: "megabyte",
			compressionType: "none",
			expectedValue:   200, // 200/1 (no compression)
			tolerance:       0.01,
		},
		{
			name:      "300 MB snappy",
			sizeValue: 300, sizeUnit: "megabyte",
			compressionType: "snappy",
			expectedValue:   120, // 300/2.5
			tolerance:       0.01,
		},
		{
			name:      "Case insensitive - GZIP",
			sizeValue: 10, sizeUnit: "gigabyte",
			compressionType: "GZIP",
			expectedValue:   3.333, // 10/3
			tolerance:       0.01,
		},
		{
			name:      "Whitespace trimming",
			sizeValue: 6, sizeUnit: "gigabyte",
			compressionType: "  lz4  ",
			expectedValue:   3, // 6/2
			tolerance:       0.01,
		},
		{
			name:      "Preserves unit - kilobytes",
			sizeValue: 1000, sizeUnit: "kilobyte",
			compressionType: "gzip",
			expectedValue:   333.333, // 1000/3
			tolerance:       0.01,
		},
		{
			name:      "Invalid compression type",
			sizeValue: 100, sizeUnit: "megabyte",
			compressionType: "zip",
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := &types.Quantity{
				Value: decimal.NewFromFloat(tt.sizeValue),
				Unit:  tt.sizeUnit,
			}

			result, err := calculateCompression(size, tt.compressionType)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify unit is preserved
			if result.Unit != tt.sizeUnit {
				t.Errorf("Expected unit %s, got %s", tt.sizeUnit, result.Unit)
			}

			// Check value
			gotValue, _ := result.Value.Float64()
			diff := gotValue - tt.expectedValue
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerance {
				t.Errorf("Expected ~%v %s, got %v %s (diff: %v, tolerance: %v)",
					tt.expectedValue, tt.sizeUnit, gotValue, result.Unit, diff, tt.tolerance)
			}

			t.Logf("✓ %s: %v %s → %v %s", tt.name,
				tt.sizeValue, tt.sizeUnit, gotValue, result.Unit)
		})
	}
}
