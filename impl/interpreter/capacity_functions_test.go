package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

func TestRequiresCapacity(t *testing.T) {
	tests := []struct {
		name        string
		load        types.Type
		capacity    types.Type
		buffer      decimal.Decimal
		expected    string
		expectError bool
	}{
		{
			name:        "10000 req/s ÷ 450 req/s",
			load:        &types.Quantity{Value: decimal.NewFromInt(10000), Unit: "req"},
			capacity:    &types.Quantity{Value: decimal.NewFromInt(450), Unit: "req"},
			buffer:      decimal.Zero,
			expected:    "23", // ⌈10000÷450⌉ = ⌈22.22⌉ = 23
			expectError: false,
		},
		{
			name:        "10 TB ÷ 2 TB",
			load:        &types.Quantity{Value: decimal.NewFromInt(10), Unit: "TB"},
			capacity:    &types.Quantity{Value: decimal.NewFromInt(2), Unit: "TB"},
			buffer:      decimal.Zero,
			expected:    "5", // Exact division
			expectError: false,
		},
		{
			name:        "100 ÷ 30",
			load:        types.NewNumber(decimal.NewFromInt(100)),
			capacity:    types.NewNumber(decimal.NewFromInt(30)),
			buffer:      decimal.Zero,
			expected:    "4", // ⌈3.33⌉ = 4
			expectError: false,
		},
		{
			name:        "load < capacity",
			load:        types.NewNumber(decimal.NewFromInt(5)),
			capacity:    types.NewNumber(decimal.NewFromInt(10)),
			buffer:      decimal.Zero,
			expected:    "1", // ⌈0.5⌉ = 1
			expectError: false,
		},
		{
			name:        "zero capacity",
			load:        types.NewNumber(decimal.NewFromInt(100)),
			capacity:    types.NewNumber(decimal.Zero),
			buffer:      decimal.Zero,
			expectError: true,
		},
		{
			name:        "negative capacity",
			load:        types.NewNumber(decimal.NewFromInt(100)),
			capacity:    types.NewNumber(decimal.NewFromInt(-10)),
			buffer:      decimal.Zero,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result *types.Number
			var err error

			if tt.buffer.IsZero() {
				result, err = requiresCapacityNoBuffer(tt.load, tt.capacity)
			} else {
				result, err = requiresCapacity(tt.load, tt.capacity, tt.buffer)
			}

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Value.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result.Value.String())
			}

			t.Logf("✓ %s = %s", tt.name, result.Value.String())
		})
	}
}

func TestRequiresCapacityWithBuffer(t *testing.T) {
	tests := []struct {
		name     string
		load     decimal.Decimal
		capacity decimal.Decimal
		buffer   decimal.Decimal
		expected string
	}{
		{
			name:     "20% buffer",
			load:     decimal.NewFromInt(10000),
			capacity: decimal.NewFromInt(450),
			buffer:   decimal.NewFromInt(20), // 20%
			expected: "27",                   // ⌈(10000×1.2)÷450⌉ = ⌈26.67⌉ = 27
		},
		{
			name:     "10% buffer",
			load:     decimal.NewFromInt(10),
			capacity: decimal.NewFromInt(2),
			buffer:   decimal.NewFromInt(10), // 10%
			expected: "6",                    // ⌈(10×1.1)÷2⌉ = ⌈5.5⌉ = 6
		},
		{
			name:     "0.1% buffer",
			load:     decimal.NewFromInt(1000),
			capacity: decimal.NewFromInt(100),
			buffer:   decimal.NewFromFloat(0.1), // 0.1%
			expected: "11",                      // ⌈(1000×1.001)÷100⌉ = ⌈10.01⌉ = 11
		},
		{
			name:     "120% buffer",
			load:     decimal.NewFromInt(100),
			capacity: decimal.NewFromInt(50),
			buffer:   decimal.NewFromInt(120), // 120%
			expected: "5",                     // ⌈(100×2.2)÷50⌉ = ⌈4.4⌉ = 5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			load := types.NewNumber(tt.load)
			capacity := types.NewNumber(tt.capacity)

			result, err := requiresCapacity(load, capacity, tt.buffer)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Value.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result.Value.String())
			}

			t.Logf("✓ %s = %s", tt.name, result.Value.String())
		})
	}
}
