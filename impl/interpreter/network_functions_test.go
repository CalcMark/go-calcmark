package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

func TestCalculateRTT(t *testing.T) {
	tests := []struct {
		scope       string
		expectedMs  float64
		expectError bool
		description string
	}{
		{"local", 0.5, false, "Same datacenter"},
		{"regional", 10.0, false, "Same region"},
		{"continental", 50.0, false, "Cross-continent"},
		{"global", 150.0, false, "Global"},
		{"LOCAL", 0.5, false, "Case insensitive"},
		{"  regional  ", 10.0, false, "Whitespace trimming"},
		{"invalid", 0, true, "Invalid scope"},
		{"", 0, true, "Empty scope"},
	}

	for _, tt := range tests {
		t.Run(tt.scope, func(t *testing.T) {
			result, err := calculateRTT(tt.scope)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for scope '%s', got none", tt.scope)
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

func TestCalculateThroughput(t *testing.T) {
	tests := []struct {
		networkType  string
		expectedMBPS float64
		expectError  bool
		description  string
	}{
		{"gigabit", 125.0, false, "1 Gbps"},
		{"ten_gig", 1250.0, false, "10 Gbps"},
		{"hundred_gig", 12500.0, false, "100 Gbps"},
		{"wifi", 12.5, false, "WiFi ~100 Mbps"},
		{"four_g", 2.5, false, "4G ~20 Mbps"},
		{"five_g", 50.0, false, "5G ~400 Mbps"},
		{"GIGABIT", 125.0, false, "Case insensitive"},
		{"  ten_gig  ", 1250.0, false, "Whitespace trimming"},
		{"invalid", 0, true, "Invalid type"},
	}

	for _, tt := range tests {
		t.Run(tt.networkType, func(t *testing.T) {
			result, err := calculateThroughput(tt.networkType)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for type '%s', got none", tt.networkType)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify it's a rate
			if result.PerUnit != "second" {
				t.Errorf("Expected per unit 'second', got '%s'", result.PerUnit)
			}

			gotMBPS := result.Amount.Value.InexactFloat64()
			if gotMBPS != tt.expectedMBPS {
				t.Errorf("Expected %v MB/s, got %v MB/s", tt.expectedMBPS, gotMBPS)
			}

			t.Logf("✓ %s: %v MB/s", tt.description, gotMBPS)
		})
	}
}

func TestCalculateTransferTime(t *testing.T) {
	tests := []struct {
		name        string
		sizeValue   float64
		sizeUnit    string
		scope       string
		networkType string
		expectedMs  float64
		tolerance   float64 // For floating point comparison
		expectError bool
	}{
		{
			name:      "1 KB regional gigabit (RTT dominates)",
			sizeValue: 1, sizeUnit: "kilobyte",
			scope: "regional", networkType: "gigabit",
			expectedMs: 10.0, // ~10ms RTT + negligible transmission
			tolerance:  0.1,
		},
		{
			name:      "1 GB global gigabit",
			sizeValue: 1, sizeUnit: "gigabyte",
			scope: "global", networkType: "gigabit",
			expectedMs: 8342.0, // 150ms RTT + ~8192ms transmission
			tolerance:  50.0,
		},
		{
			name:      "10 MB local ten_gig",
			sizeValue: 10, sizeUnit: "megabyte",
			scope: "local", networkType: "ten_gig",
			expectedMs: 8.5, // 0.5ms RTT + 8ms transmission
			tolerance:  1.0,
		},
		{
			name:      "Invalid size unit",
			sizeValue: 100, sizeUnit: "meters",
			scope: "local", networkType: "gigabit",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := &types.Quantity{
				Value: decimal.NewFromFloat(tt.sizeValue),
				Unit:  tt.sizeUnit,
			}

			result, err := calculateTransferTime(size, tt.scope, tt.networkType)

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

			// Expected is in milliseconds, convert to seconds
			expectedSeconds := tt.expectedMs / 1000.0

			// Check within tolerance (also in seconds)
			toleranceSeconds := tt.tolerance / 1000.0
			diff := resultSeconds - expectedSeconds
			if diff < 0 {
				diff = -diff
			}
			if diff > toleranceSeconds {
				t.Errorf("Expected ~%v seconds, got %v seconds (diff: %v, tolerance: %v)",
					expectedSeconds, resultSeconds, diff, toleranceSeconds)
			}

			t.Logf("✓ %s: %v %s (%v seconds)", tt.name, result.Value, result.Unit, resultSeconds)
		})
	}
}
