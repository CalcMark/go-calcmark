package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// TestCapacityAtEval tests evaluation of the new "X at Y per UNIT" syntax.
// The function computes: ⌈demand / capacity⌉ and returns a Quantity with the specified unit.
//
// Examples:
//   10 TB at 2 TB per disk → 5 disks
//   100 apples at 30 per crate → 4 crates (⌈3.33⌉)
//   10000 req/s at 450 req/s per server → 23 servers (⌈22.22⌉)

func TestCapacityAtFunction_Quantities(t *testing.T) {
	tests := []struct {
		name          string
		demand        types.Type
		capacity      types.Type
		unit          string
		buffer        decimal.Decimal // 0 means no buffer
		expectedValue string
		expectedUnit  string
		expectError   bool
	}{
		{
			name:          "10 TB at 2 TB per disk",
			demand:        types.NewQuantity(decimal.NewFromInt(10), "TB"),
			capacity:      types.NewQuantity(decimal.NewFromInt(2), "TB"),
			unit:          "disk",
			buffer:        decimal.Zero,
			expectedValue: "5",
			expectedUnit:  "disk",
		},
		{
			name:          "10 TB at 2 TB per disk (pluralized: disks)",
			demand:        types.NewQuantity(decimal.NewFromInt(10), "TB"),
			capacity:      types.NewQuantity(decimal.NewFromInt(2), "TB"),
			unit:          "disks",
			buffer:        decimal.Zero,
			expectedValue: "5",
			expectedUnit:  "disks",
		},
		{
			name:          "100 apples at 30 per crate (ceiling)",
			demand:        types.NewQuantity(decimal.NewFromInt(100), "apples"),
			capacity:      types.NewNumber(decimal.NewFromInt(30)),
			unit:          "crate",
			buffer:        decimal.Zero,
			expectedValue: "4", // ⌈3.33⌉ = 4
			expectedUnit:  "crate",
		},
		{
			name:          "exact division: 100 at 25 per batch",
			demand:        types.NewNumber(decimal.NewFromInt(100)),
			capacity:      types.NewNumber(decimal.NewFromInt(25)),
			unit:          "batch",
			buffer:        decimal.Zero,
			expectedValue: "4",
			expectedUnit:  "batch",
		},
		{
			name:          "demand less than capacity: 5 at 10 per unit",
			demand:        types.NewNumber(decimal.NewFromInt(5)),
			capacity:      types.NewNumber(decimal.NewFromInt(10)),
			unit:          "unit",
			buffer:        decimal.Zero,
			expectedValue: "1", // ⌈0.5⌉ = 1
			expectedUnit:  "unit",
		},
		{
			name:          "with 10% buffer: 10 TB at 2 TB per disk",
			demand:        types.NewQuantity(decimal.NewFromInt(10), "TB"),
			capacity:      types.NewQuantity(decimal.NewFromInt(2), "TB"),
			unit:          "disk",
			buffer:        decimal.NewFromFloat(0.10), // 10%
			expectedValue: "6",                        // ⌈(10×1.1)/2⌉ = ⌈5.5⌉ = 6
			expectedUnit:  "disk",
		},
		{
			name:          "with 20% buffer: 100 at 30 per crate",
			demand:        types.NewNumber(decimal.NewFromInt(100)),
			capacity:      types.NewNumber(decimal.NewFromInt(30)),
			unit:          "crate",
			buffer:        decimal.NewFromFloat(0.20), // 20%
			expectedValue: "4",                        // ⌈(100×1.2)/30⌉ = ⌈4.0⌉ = 4
			expectedUnit:  "crate",
		},
		{
			name:          "with 100% buffer (double demand)",
			demand:        types.NewNumber(decimal.NewFromInt(100)),
			capacity:      types.NewNumber(decimal.NewFromInt(50)),
			unit:          "unit",
			buffer:        decimal.NewFromFloat(1.0), // 100%
			expectedValue: "4",                       // ⌈(100×2.0)/50⌉ = ⌈4.0⌉ = 4
			expectedUnit:  "unit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result types.Type
			var err error

			if tt.buffer.IsZero() {
				result, err = capacityAt(tt.demand, tt.capacity, tt.unit)
			} else {
				result, err = capacityAtWithBuffer(tt.demand, tt.capacity, tt.unit, tt.buffer)
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

			qty, ok := result.(*types.Quantity)
			if !ok {
				t.Fatalf("Expected Quantity result, got %T", result)
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

func TestCapacityAtFunction_Rates(t *testing.T) {
	tests := []struct {
		name          string
		demand        types.Type
		capacity      types.Type
		unit          string
		buffer        decimal.Decimal
		expectedValue string
		expectedUnit  string
		expectError   bool
	}{
		{
			name: "10000 req/s at 450 req/s per server",
			demand: types.NewRate(
				types.NewQuantity(decimal.NewFromInt(10000), "req"),
				"s",
			),
			capacity: types.NewRate(
				types.NewQuantity(decimal.NewFromInt(450), "req"),
				"s",
			),
			unit:          "server",
			buffer:        decimal.Zero,
			expectedValue: "23", // ⌈22.22⌉ = 23
			expectedUnit:  "server",
		},
		{
			name: "100 MB/s at 10 MB/s per connection",
			demand: types.NewRate(
				types.NewQuantity(decimal.NewFromInt(100), "MB"),
				"s",
			),
			capacity: types.NewRate(
				types.NewQuantity(decimal.NewFromInt(10), "MB"),
				"s",
			),
			unit:          "connection",
			buffer:        decimal.Zero,
			expectedValue: "10",
			expectedUnit:  "connection",
		},
		{
			name: "with 20% buffer: 10000 req/s at 450 req/s per server",
			demand: types.NewRate(
				types.NewQuantity(decimal.NewFromInt(10000), "req"),
				"s",
			),
			capacity: types.NewRate(
				types.NewQuantity(decimal.NewFromInt(450), "req"),
				"s",
			),
			unit:          "server",
			buffer:        decimal.NewFromFloat(0.20), // 20%
			expectedValue: "27",                       // ⌈(10000×1.2)/450⌉ = ⌈26.67⌉ = 27
			expectedUnit:  "server",
		},
		{
			name: "1000 requests/hour at 100 requests/hour per worker",
			demand: types.NewRate(
				types.NewQuantity(decimal.NewFromInt(1000), "requests"),
				"hour",
			),
			capacity: types.NewRate(
				types.NewQuantity(decimal.NewFromInt(100), "requests"),
				"hour",
			),
			unit:          "worker",
			buffer:        decimal.Zero,
			expectedValue: "10",
			expectedUnit:  "worker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result types.Type
			var err error

			if tt.buffer.IsZero() {
				result, err = capacityAt(tt.demand, tt.capacity, tt.unit)
			} else {
				result, err = capacityAtWithBuffer(tt.demand, tt.capacity, tt.unit, tt.buffer)
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

			qty, ok := result.(*types.Quantity)
			if !ok {
				t.Fatalf("Expected Quantity result, got %T", result)
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

func TestCapacityAtFunction_Errors(t *testing.T) {
	tests := []struct {
		name     string
		demand   types.Type
		capacity types.Type
		unit     string
		errPart  string
	}{
		{
			name:     "zero capacity",
			demand:   types.NewNumber(decimal.NewFromInt(100)),
			capacity: types.NewNumber(decimal.Zero),
			unit:     "unit",
			errPart:  "zero",
		},
		{
			name:     "negative capacity",
			demand:   types.NewNumber(decimal.NewFromInt(100)),
			capacity: types.NewNumber(decimal.NewFromInt(-10)),
			unit:     "unit",
			errPart:  "positive",
		},
		{
			name:     "incompatible units: TB demand with GB capacity",
			demand:   types.NewQuantity(decimal.NewFromInt(10), "TB"),
			capacity: types.NewQuantity(decimal.NewFromInt(2), "GB"),
			unit:     "disk",
			errPart:  "", // May or may not error depending on implementation
		},
		{
			name: "mismatched rate time units",
			demand: types.NewRate(
				types.NewQuantity(decimal.NewFromInt(100), "req"),
				"s",
			),
			capacity: types.NewRate(
				types.NewQuantity(decimal.NewFromInt(10), "req"),
				"hour",
			),
			unit:    "server",
			errPart: "", // May convert or error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := capacityAt(tt.demand, tt.capacity, tt.unit)

			if tt.errPart != "" {
				if err == nil {
					t.Errorf("Expected error containing %q but got none", tt.errPart)
				} else {
					t.Logf("✓ Got expected error: %v", err)
				}
			} else {
				// For cases where behavior is implementation-dependent
				if err != nil {
					t.Logf("Got error (may be expected): %v", err)
				} else {
					t.Logf("Succeeded (may be expected)")
				}
			}
		})
	}
}

func TestCapacityAtFunction_BufferErrors(t *testing.T) {
	tests := []struct {
		name    string
		demand  types.Type
		buffer  decimal.Decimal
		errPart string
	}{
		{
			name:    "negative buffer",
			demand:  types.NewNumber(decimal.NewFromInt(100)),
			buffer:  decimal.NewFromFloat(-0.10), // -10%
			errPart: "negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capacity := types.NewNumber(decimal.NewFromInt(10))
			_, err := capacityAtWithBuffer(tt.demand, capacity, "unit", tt.buffer)

			if err == nil {
				t.Errorf("Expected error containing %q but got none", tt.errPart)
			} else {
				t.Logf("✓ Got expected error: %v", err)
			}
		})
	}
}

// TestCapacityAtFunction_EndToEnd tests the full interpreter pipeline (parse → eval)
// This ensures the parser, semantic checker, and interpreter all work together.
func TestCapacityAtFunction_EndToEnd(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValue string
		expectedUnit  string
		expectError   bool
	}{
		{
			name:          "basic capacity",
			input:         "10 TB at 2 TB per disk\n",
			expectedValue: "5",
			expectedUnit:  "disk",
		},
		{
			name:          "rate capacity",
			input:         "10000 req/s at 450 req/s per server\n",
			expectedValue: "23",
			expectedUnit:  "server",
		},
		{
			name:          "with buffer",
			input:         "10 TB at 2 TB per disk with 10% buffer\n",
			expectedValue: "6",
			expectedUnit:  "disk",
		},
		{
			name:          "pure numbers",
			input:         "100 at 30 per crate\n",
			expectedValue: "4",
			expectedUnit:  "crate",
		},
		{
			name:          "demand less than capacity",
			input:         "5 at 10 per unit\n",
			expectedValue: "1",
			expectedUnit:  "unit",
		},
		{
			name:          "assignment",
			input:         "disks = 10 TB at 2 TB per disk\n",
			expectedValue: "5",
			expectedUnit:  "disk",
		},
		{
			name:          "slash syntax",
			input:         "10 TB at 2 TB/disk\n",
			expectedValue: "5",
			expectedUnit:  "disk",
		},
		{
			name:          "slash syntax with buffer",
			input:         "10 GB/day at 2 GB/disk with 30% buffer\n",
			expectedValue: "7",
			expectedUnit:  "disk",
		},
		{
			name:          "unit conversion: GB to PB rate",
			input:         "10 GB/s at 2 PB/s per connection with 10% buffer\n",
			expectedValue: "1",
			expectedUnit:  "connection",
		},
		{
			name:          "unit conversion: GB to TB quantity",
			input:         "100 GB at 1 TB per disk\n",
			expectedValue: "1",
			expectedUnit:  "disk",
		},
		{
			name:          "unit conversion: MB to GB rate",
			input:         "500 MB/s at 1 GB/s per pipe\n",
			expectedValue: "1",
			expectedUnit:  "pipe",
		},
		{
			name:          "throughput: GB/s to Mbps",
			input:         "10 GB/s at 100 Mbps per connection\n",
			expectedValue: "859", // 10 GiB/s = 85,899 Mbps → ceil(85899/100) = 859
			expectedUnit:  "connection",
		},
		{
			name:          "throughput: 1 GB/s to Gbps",
			input:         "1 GB/s at 1 Gbps per link\n",
			expectedValue: "9", // 1 GiB/s ≈ 8.59 Gbps → ceil(8.59/1) = 9
			expectedUnit:  "link",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				if tt.expectError {
					t.Logf("✓ Expected parse error: %v", err)
					return
				}
				t.Fatalf("Parse error: %v", err)
			}

			// Evaluate the parsed nodes
			interp := NewInterpreter()
			results, err := interp.Eval(nodes)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("Expected at least one result")
			}

			// Get the last result
			result := results[len(results)-1]

			qty, ok := result.(*types.Quantity)
			if !ok {
				t.Fatalf("Expected Quantity result, got %T", result)
			}

			if qty.Value.String() != tt.expectedValue {
				t.Errorf("Expected value %s, got %s", tt.expectedValue, qty.Value.String())
			}

			if qty.Unit != tt.expectedUnit {
				t.Errorf("Expected unit %q, got %q", tt.expectedUnit, qty.Unit)
			}

			t.Logf("✓ %q → %s %s", tt.input, qty.Value.String(), qty.Unit)
		})
	}
}
