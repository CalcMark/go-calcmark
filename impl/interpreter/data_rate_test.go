package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// TestDataRateUnitConversions tests that all data rate unit variations work correctly.
// This includes:
// - Binary byte units: KiB, MiB, GiB, TiB (1024-based)
// - Decimal byte units aliased to binary: KB, MB, GB, TB
// - Bit units: Kbit, Mbit, Gbit, Mbps, Gbps (1000-based)
// - Rate syntax: MB/s, MiB/s, Mbps
func TestDataRateUnitConversions(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValue string
		expectedUnit  string
	}{
		// ========== BASIC DATA SIZE ADDITION ==========
		{
			name:          "GB + MB (binary)",
			input:         "1 GB + 1 MB\n",
			expectedValue: "1.0009765625",
			expectedUnit:  "GB",
		},
		{
			name:          "GiB + MiB (explicit binary)",
			input:         "1 GiB + 1 MiB\n",
			expectedValue: "1.0009765625",
			expectedUnit:  "GiB",
		},

		// ========== CAPACITY WITH RATE UNITS ==========
		// Rate-to-Rate conversions (same time unit, different data units)
		{
			name:          "GB/s at MB/s (binary byte rates)",
			input:         "1 GB/s at 100 MB/s per connection\n",
			expectedValue: "11", // 1024 MB / 100 MB = 10.24 → ceil = 11
			expectedUnit:  "connection",
		},
		{
			name:          "GiB/s at MiB/s (explicit binary)",
			input:         "1 GiB/s at 100 MiB/s per pipe\n",
			expectedValue: "11", // Same as above
			expectedUnit:  "pipe",
		},

		// Rate-to-Throughput conversions (GB/s vs Mbps)
		{
			name:          "1 GB/s at 100 Mbps (byte rate vs bit rate)",
			input:         "1 GB/s at 100 Mbps per connection\n",
			expectedValue: "86", // 1 GiB/s = 8,589,934,592 bps / 100M = 85.9 → 86
			expectedUnit:  "connection",
		},
		{
			name:          "10 GB/s at 1 Gbps",
			input:         "10 GB/s at 1 Gbps per link\n",
			expectedValue: "86", // 10 GiB/s ≈ 85.9 Gbps → 86
			expectedUnit:  "link",
		},
		{
			name:          "1 GB/s at 1 Gbps",
			input:         "1 GB/s at 1 Gbps per link\n",
			expectedValue: "9", // 1 GiB/s ≈ 8.59 Gbps → 9
			expectedUnit:  "link",
		},

		// ========== THROUGHPUT UNITS AS QUANTITIES ==========
		{
			name:          "Mbps arithmetic",
			input:         "100 Mbps * 10\n",
			expectedValue: "1000",
			expectedUnit:  "Mbps",
		},
		{
			name:          "Gbps addition",
			input:         "1 Gbps + 1 Gbps\n",
			expectedValue: "2",
			expectedUnit:  "Gbps",
		},

		// ========== MIXED UNIT CAPACITY CALCULATIONS ==========
		{
			name:          "TB at GB per disk",
			input:         "10 TB at 2 TB per disk\n",
			expectedValue: "5",
			expectedUnit:  "disk",
		},
		{
			name:          "TB at GB per disk (conversion)",
			input:         "1 TB at 100 GB per disk\n",
			expectedValue: "11", // 1024 GB / 100 GB = 10.24 → 11
			expectedUnit:  "disk",
		},
		{
			name:          "PB at TB per server",
			input:         "1 PB at 1 TB per server\n",
			expectedValue: "1024",
			expectedUnit:  "server",
		},

		// ========== RATE CAPACITY WITH BUFFER ==========
		{
			name:          "GB/s at Mbps with buffer",
			input:         "1 GB/s at 100 Mbps per connection with 10% buffer\n",
			expectedValue: "95", // 86 * 1.1 = 94.6 → 95
			expectedUnit:  "connection",
		},
		{
			name:          "TB/day at GB/day with buffer",
			input:         "1 TB/day at 100 GB/day per worker with 20% buffer\n",
			expectedValue: "13", // (1024/100) * 1.2 = 12.29 → 13
			expectedUnit:  "worker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			interp := NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("Expected at least one result")
			}

			result := results[len(results)-1]

			switch v := result.(type) {
			case *types.Quantity:
				if v.Value.String() != tt.expectedValue {
					t.Errorf("Expected value %s, got %s", tt.expectedValue, v.Value.String())
				}
				if v.Unit != tt.expectedUnit {
					t.Errorf("Expected unit %q, got %q", tt.expectedUnit, v.Unit)
				}
				t.Logf("✓ %s = %s %s", tt.name, v.Value.String(), v.Unit)
			case *types.Number:
				if v.Value.String() != tt.expectedValue {
					t.Errorf("Expected value %s, got %s", tt.expectedValue, v.Value.String())
				}
				t.Logf("✓ %s = %s", tt.name, v.Value.String())
			default:
				t.Fatalf("Unexpected result type: %T", result)
			}
		})
	}
}

// TestDataSizeUnitRegistry verifies the unit registry has correct conversions.
func TestDataSizeUnitRegistry(t *testing.T) {
	tests := []struct {
		name   string
		unit   string
		value  float64
		toBits float64 // Expected value in bits (base unit)
	}{
		// Bytes
		{"byte", "byte", 1, 8},
		{"bytes", "bytes", 10, 80},

		// Binary byte prefixes (1024-based)
		{"KiB", "kib", 1, 8 * 1024},
		{"MiB", "mib", 1, 8 * 1024 * 1024},
		{"GiB", "gib", 1, 8 * 1024 * 1024 * 1024},
		{"TiB", "tib", 1, 8 * 1024 * 1024 * 1024 * 1024},

		// KB/MB/GB aliased to binary
		{"KB", "kb", 1, 8 * 1024},
		{"MB", "mb", 1, 8 * 1024 * 1024},
		{"GB", "gb", 1, 8 * 1024 * 1024 * 1024},
		{"TB", "tb", 1, 8 * 1024 * 1024 * 1024 * 1024},

		// Bit prefixes (1000-based)
		{"bit", "bit", 1, 1},
		{"Kbit", "kbit", 1, 1000},
		{"Mbit", "mbit", 1, 1000000},
		{"Gbit", "gbit", 1, 1000000000},

		// Throughput aliases
		{"Kbps", "kbps", 1, 1000},
		{"Mbps", "mbps", 1, 1000000},
		{"Gbps", "gbps", 1, 1000000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, ok := GetUnitInfo(tt.unit)
			if !ok {
				t.Fatalf("Unit %q not found in registry", tt.unit)
			}

			if info.Category != CategoryDataSize {
				t.Errorf("Expected category %q, got %q", CategoryDataSize, info.Category)
			}

			gotBits := info.ToBaseUnit(tt.value)
			if gotBits != tt.toBits {
				t.Errorf("ToBaseUnit(%v %s) = %v bits, want %v bits",
					tt.value, tt.name, gotBits, tt.toBits)
			}

			// Verify round-trip
			roundTrip := info.FromBaseUnit(gotBits)
			if roundTrip != tt.value {
				t.Errorf("Round-trip failed: %v -> %v bits -> %v",
					tt.value, gotBits, roundTrip)
			}

			t.Logf("✓ %s: %v → %v bits", tt.name, tt.value, gotBits)
		})
	}
}

// TestByteBitConversions verifies byte-to-bit conversions work correctly.
func TestByteBitConversions(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValue string
		expectedUnit  string
	}{
		// These test quantity arithmetic with byte/bit units
		{
			name:          "bytes to bits addition",
			input:         "1 byte + 8 bits\n",
			expectedValue: "2",
			expectedUnit:  "byte",
		},
		{
			name:          "MB to Mbit",
			input:         "1 MB + 1 Mbit\n",
			expectedValue: "1.11920928955078125", // 1 MiB (8388608 bits) + 1 Mbit (1000000 bits) = 9388608 bits = ~1.119 MiB
			expectedUnit:  "MB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			interp := NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("Expected at least one result")
			}

			result := results[len(results)-1]
			qty, ok := result.(*types.Quantity)
			if !ok {
				t.Fatalf("Expected Quantity, got %T", result)
			}

			if qty.Value.String() != tt.expectedValue {
				t.Errorf("Expected value %s, got %s", tt.expectedValue, qty.Value.String())
			}
			if qty.Unit != tt.expectedUnit {
				t.Errorf("Expected unit %q, got %q", tt.expectedUnit, qty.Unit)
			}

			t.Logf("✓ %s = %s %s", tt.name, qty.Value.String(), qty.Unit)
		})
	}
}
