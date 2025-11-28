package parser

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
)

// TestCapacityAtSyntax tests the new "X at Y per UNIT" capacity planning syntax.
// This syntax computes: ⌈demand / capacity⌉ and returns a quantity with the specified unit.
//
// Grammar:
//   demand "at" capacity "per" unit ["with" percentage "buffer"]
//
// Where:
//   - demand: quantity or rate (e.g., "10 TB", "10000 req/s")
//   - capacity: quantity or rate per unit (e.g., "2 TB", "450 req/s")
//   - unit: identifier for the result unit (e.g., "disk", "server", "crate")
//   - buffer: optional percentage buffer (e.g., "20%")
//
// Dimensional analysis:
//   10 TB at 2 TB per disk → 10 TB ÷ (2 TB/disk) = 5 disks
//   100 MB/s at 10 MB/s per connection → 100 MB/s ÷ (10 MB/s/connection) = 10 connections

func TestCapacityAtSyntax_Quantities(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		checkFunc   func(*testing.T, []ast.Node)
	}{
		{
			name:        "simple quantity: 10 TB at 2 TB per disk",
			input:       "10 TB at 2 TB per disk\n",
			expectError: false,
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				fc, ok := nodes[0].(*ast.FunctionCall)
				if !ok {
					t.Fatalf("Expected FunctionCall, got %T", nodes[0])
				}
				if fc.Name != "capacity" {
					t.Errorf("Expected function 'capacity', got '%s'", fc.Name)
				}
				// Args: demand, capacity, unit
				if len(fc.Arguments) != 3 {
					t.Errorf("Expected 3 arguments (demand, capacity, unit), got %d", len(fc.Arguments))
				}
			},
		},
		{
			name:        "quantity with buffer: 10 TB at 2 TB per disk with 10% buffer",
			input:       "10 TB at 2 TB per disk with 10% buffer\n",
			expectError: false,
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				fc, ok := nodes[0].(*ast.FunctionCall)
				if !ok {
					t.Fatalf("Expected FunctionCall, got %T", nodes[0])
				}
				if fc.Name != "capacity" {
					t.Errorf("Expected function 'capacity', got '%s'", fc.Name)
				}
				// Args: demand, capacity, unit, buffer
				if len(fc.Arguments) != 4 {
					t.Errorf("Expected 4 arguments (demand, capacity, unit, buffer), got %d", len(fc.Arguments))
				}
			},
		},
		{
			name:        "apples in crates: 100 apples at 30 per crate",
			input:       "100 apples at 30 per crate\n",
			expectError: false,
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				fc, ok := nodes[0].(*ast.FunctionCall)
				if !ok {
					t.Fatalf("Expected FunctionCall, got %T", nodes[0])
				}
				if len(fc.Arguments) != 3 {
					t.Errorf("Expected 3 arguments, got %d", len(fc.Arguments))
				}
			},
		},
		{
			name:        "pure numbers: 100 at 30 per unit",
			input:       "100 at 30 per unit\n",
			expectError: false,
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				fc, ok := nodes[0].(*ast.FunctionCall)
				if !ok {
					t.Fatalf("Expected FunctionCall, got %T", nodes[0])
				}
				if len(fc.Arguments) != 3 {
					t.Errorf("Expected 3 arguments, got %d", len(fc.Arguments))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkFunc != nil && !tt.expectError {
				tt.checkFunc(t, nodes)
			}

			if !tt.expectError {
				t.Logf("✓ Parsed: %s", tt.input)
			}
		})
	}
}

func TestCapacityAtSyntax_Rates(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		checkFunc   func(*testing.T, []ast.Node)
	}{
		{
			name:        "slash-rate: 10000 req/s at 450 req/s per server",
			input:       "10000 req/s at 450 req/s per server\n",
			expectError: false,
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				fc, ok := nodes[0].(*ast.FunctionCall)
				if !ok {
					t.Fatalf("Expected FunctionCall, got %T", nodes[0])
				}
				if fc.Name != "capacity" {
					t.Errorf("Expected function 'capacity', got '%s'", fc.Name)
				}
				if len(fc.Arguments) != 3 {
					t.Errorf("Expected 3 arguments, got %d", len(fc.Arguments))
				}
			},
		},
		{
			name:        "slash-rate with buffer: 10000 req/s at 450 req/s per server with 20% buffer",
			input:       "10000 req/s at 450 req/s per server with 20% buffer\n",
			expectError: false,
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				fc, ok := nodes[0].(*ast.FunctionCall)
				if !ok {
					t.Fatalf("Expected FunctionCall, got %T", nodes[0])
				}
				if len(fc.Arguments) != 4 {
					t.Errorf("Expected 4 arguments (with buffer), got %d", len(fc.Arguments))
				}
			},
		},
		{
			name:        "bandwidth: 100 MB/s at 10 MB/s per connection",
			input:       "100 MB/s at 10 MB/s per connection\n",
			expectError: false,
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				fc, ok := nodes[0].(*ast.FunctionCall)
				if !ok {
					t.Fatalf("Expected FunctionCall, got %T", nodes[0])
				}
				if len(fc.Arguments) != 3 {
					t.Errorf("Expected 3 arguments, got %d", len(fc.Arguments))
				}
			},
		},
		// NOTE: "100 MB per second at 10 MB per second per connection" is intentionally
		// NOT supported because the multiple "per" keywords create ambiguous parsing.
		// Use slash syntax for rates in capacity planning: "100 MB/s at 10 MB/s per connection"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkFunc != nil && !tt.expectError {
				tt.checkFunc(t, nodes)
			}

			if !tt.expectError {
				t.Logf("✓ Parsed: %s", tt.input)
			}
		})
	}
}

func TestCapacityAtSyntax_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		checkFunc   func(*testing.T, []ast.Node)
	}{
		{
			name:        "demand less than capacity: 5 at 10 per unit",
			input:       "5 at 10 per unit\n",
			expectError: false,
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				// Should parse successfully - result is 1 (ceiling of 0.5)
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
			},
		},
		{
			name:        "exact division: 100 at 25 per batch",
			input:       "100 at 25 per batch\n",
			expectError: false,
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				// Should parse successfully - result is exactly 4
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
			},
		},
		{
			name:        "decimal buffer: 10 TB at 2 TB per disk with 0.5% buffer",
			input:       "10 TB at 2 TB per disk with 0.5% buffer\n",
			expectError: false,
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				fc, ok := nodes[0].(*ast.FunctionCall)
				if !ok {
					t.Fatalf("Expected FunctionCall, got %T", nodes[0])
				}
				if len(fc.Arguments) != 4 {
					t.Errorf("Expected 4 arguments, got %d", len(fc.Arguments))
				}
			},
		},
		{
			name:        "large buffer: 100 at 50 per unit with 100% buffer",
			input:       "100 at 50 per unit with 100% buffer\n",
			expectError: false,
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				// 100% buffer means double the demand
				fc, ok := nodes[0].(*ast.FunctionCall)
				if !ok {
					t.Fatalf("Expected FunctionCall, got %T", nodes[0])
				}
				if len(fc.Arguments) != 4 {
					t.Errorf("Expected 4 arguments, got %d", len(fc.Arguments))
				}
			},
		},
		{
			name:        "multi-word unit: 100 GB at 50 GB per storage node",
			input:       "100 GB at 50 GB per storage_node\n",
			expectError: false,
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				// Unit names can be identifiers with underscores
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkFunc != nil && !tt.expectError {
				tt.checkFunc(t, nodes)
			}

			if !tt.expectError {
				t.Logf("✓ Parsed: %s", tt.input)
			}
		})
	}
}

// TestCapacityAtSyntax_Assignment tests that capacity expressions work in assignments
func TestCapacityAtSyntax_Assignment(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		varName string
		numArgs int
	}{
		{
			name:    "simple assignment",
			input:   "disks = 10 TB at 2 TB per disk\n",
			varName: "disks",
			numArgs: 3,
		},
		{
			name:    "assignment with buffer",
			input:   "servers = 10000 req/s at 450 req/s per server with 20% buffer\n",
			varName: "servers",
			numArgs: 4,
		},
		{
			name:    "assignment with underscore variable",
			input:   "storage_units = 100 GB at 50 GB per unit\n",
			varName: "storage_units",
			numArgs: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(nodes) != 1 {
				t.Fatalf("Expected 1 node, got %d", len(nodes))
			}

			assign, ok := nodes[0].(*ast.Assignment)
			if !ok {
				t.Fatalf("Expected Assignment, got %T", nodes[0])
			}

			if assign.Name != tt.varName {
				t.Errorf("Expected variable '%s', got '%s'", tt.varName, assign.Name)
			}

			fc, ok := assign.Value.(*ast.FunctionCall)
			if !ok {
				t.Fatalf("Expected FunctionCall as assignment value, got %T", assign.Value)
			}

			if fc.Name != "capacity" {
				t.Errorf("Expected function 'capacity', got '%s'", fc.Name)
			}

			if len(fc.Arguments) != tt.numArgs {
				t.Errorf("Expected %d arguments, got %d", tt.numArgs, len(fc.Arguments))
			}

			t.Logf("✓ Parsed assignment: %s = capacity(...)", tt.varName)
		})
	}
}

// TestCapacityAtSyntax_ASTStructure verifies the detailed AST structure
func TestCapacityAtSyntax_ASTStructure(t *testing.T) {
	input := "10 TB at 2 TB per disk\n"
	nodes, err := Parse(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	fc, ok := nodes[0].(*ast.FunctionCall)
	if !ok {
		t.Fatalf("Expected FunctionCall, got %T", nodes[0])
	}

	// Verify function name
	if fc.Name != "capacity" {
		t.Errorf("Expected function name 'capacity', got '%s'", fc.Name)
	}

	// Verify argument count
	if len(fc.Arguments) != 3 {
		t.Fatalf("Expected 3 arguments, got %d", len(fc.Arguments))
	}

	// First argument: demand (10 TB) - should be QuantityLiteral
	demand, ok := fc.Arguments[0].(*ast.QuantityLiteral)
	if !ok {
		t.Errorf("Expected demand to be QuantityLiteral, got %T", fc.Arguments[0])
	} else {
		if demand.Unit != "TB" {
			t.Errorf("Expected demand unit 'TB', got '%s'", demand.Unit)
		}
		t.Logf("  Demand: %s %s", demand.Value, demand.Unit)
	}

	// Second argument: capacity (2 TB) - should be QuantityLiteral
	capacity, ok := fc.Arguments[1].(*ast.QuantityLiteral)
	if !ok {
		t.Errorf("Expected capacity to be QuantityLiteral, got %T", fc.Arguments[1])
	} else {
		if capacity.Unit != "TB" {
			t.Errorf("Expected capacity unit 'TB', got '%s'", capacity.Unit)
		}
		t.Logf("  Capacity: %s %s", capacity.Value, capacity.Unit)
	}

	// Third argument: unit (disk) - should be Identifier
	unit, ok := fc.Arguments[2].(*ast.Identifier)
	if !ok {
		t.Errorf("Expected unit to be Identifier, got %T", fc.Arguments[2])
	} else {
		if unit.Name != "disk" {
			t.Errorf("Expected unit name 'disk', got '%s'", unit.Name)
		}
		t.Logf("  Unit: %s", unit.Name)
	}
}

// TestCapacityAtSyntax_ASTStructureWithBuffer verifies AST structure with buffer
func TestCapacityAtSyntax_ASTStructureWithBuffer(t *testing.T) {
	input := "10 TB at 2 TB per disk with 10% buffer\n"
	nodes, err := Parse(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	fc, ok := nodes[0].(*ast.FunctionCall)
	if !ok {
		t.Fatalf("Expected FunctionCall, got %T", nodes[0])
	}

	// Verify 4 arguments (demand, capacity, unit, buffer)
	if len(fc.Arguments) != 4 {
		t.Fatalf("Expected 4 arguments, got %d", len(fc.Arguments))
	}

	// Fourth argument: buffer (10%) - should be NumberLiteral
	buffer, ok := fc.Arguments[3].(*ast.NumberLiteral)
	if !ok {
		t.Errorf("Expected buffer to be NumberLiteral, got %T", fc.Arguments[3])
	} else {
		t.Logf("  Buffer: %s", buffer.Value)
	}
}

// TestCapacityAtSyntax_SlashSyntax tests that "/" can be used instead of "per"
func TestCapacityAtSyntax_SlashSyntax(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		checkFunc   func(*testing.T, []ast.Node)
	}{
		{
			name:        "slash syntax: 10 TB at 2 TB/disk",
			input:       "10 TB at 2 TB/disk\n",
			expectError: false,
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				fc, ok := nodes[0].(*ast.FunctionCall)
				if !ok {
					t.Fatalf("Expected FunctionCall, got %T", nodes[0])
				}
				if fc.Name != "capacity" {
					t.Errorf("Expected function 'capacity', got '%s'", fc.Name)
				}
				if len(fc.Arguments) != 3 {
					t.Errorf("Expected 3 arguments, got %d", len(fc.Arguments))
				}
				// Verify the unit is "disk"
				unit, ok := fc.Arguments[2].(*ast.Identifier)
				if !ok {
					t.Errorf("Expected unit to be Identifier, got %T", fc.Arguments[2])
				} else if unit.Name != "disk" {
					t.Errorf("Expected unit 'disk', got '%s'", unit.Name)
				}
			},
		},
		{
			name:        "slash syntax with buffer: 10 GB/day at 2 GB/disk with 30% buffer",
			input:       "10 GB/day at 2 GB/disk with 30% buffer\n",
			expectError: false,
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				fc, ok := nodes[0].(*ast.FunctionCall)
				if !ok {
					t.Fatalf("Expected FunctionCall, got %T", nodes[0])
				}
				if len(fc.Arguments) != 4 {
					t.Errorf("Expected 4 arguments (with buffer), got %d", len(fc.Arguments))
				}
			},
		},
		{
			name:        "slash syntax pure numbers: 100 at 25/batch",
			input:       "100 at 25/batch\n",
			expectError: false,
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				fc, ok := nodes[0].(*ast.FunctionCall)
				if !ok {
					t.Fatalf("Expected FunctionCall, got %T", nodes[0])
				}
				if len(fc.Arguments) != 3 {
					t.Errorf("Expected 3 arguments, got %d", len(fc.Arguments))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkFunc != nil && !tt.expectError {
				tt.checkFunc(t, nodes)
			}

			if !tt.expectError {
				t.Logf("✓ Parsed: %s", tt.input)
			}
		})
	}
}

func TestCapacityAtSyntax_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "missing per or slash",
			input: "10 TB at 2 TB disk\n",
		},
		{
			name:  "missing unit after per",
			input: "10 TB at 2 TB per\n",
		},
		{
			name:  "missing capacity",
			input: "10 TB at per disk\n",
		},
		{
			name:  "buffer without with keyword",
			input: "10 TB at 2 TB per disk 10%\n",
		},
		{
			name:  "missing buffer keyword after percentage",
			input: "10 TB at 2 TB per disk with 10%\n",
		},
		{
			name:  "buffer keyword without percentage",
			input: "10 TB at 2 TB per disk with buffer\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if err == nil {
				t.Errorf("Expected parse error for %q but succeeded", tt.input)
			} else {
				t.Logf("✓ Correctly rejected: %s (error: %v)", tt.input, err)
			}
		})
	}
}
