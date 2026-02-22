package parser

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
)

// TestNLReadFunction tests "read <quantity> from <storage>" syntax.
func TestNLReadFunction(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		checkFunc   func(*testing.T, []ast.Node)
	}{
		{
			name:  "basic read from ssd",
			input: "read 100 MB from ssd\n",
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				t.Helper()
				fc := expectFunctionCall(t, nodes, "read", 2)
				if fc == nil {
					return
				}
				expectQuantityArg(t, fc.Arguments[0], "100", "MB")
				expectIdentifierArg(t, fc.Arguments[1], "ssd")
			},
		},
		{
			name:  "read from nvme",
			input: "read 1 GB from nvme\n",
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				t.Helper()
				fc := expectFunctionCall(t, nodes, "read", 2)
				if fc == nil {
					return
				}
				expectQuantityArg(t, fc.Arguments[0], "1", "GB")
				expectIdentifierArg(t, fc.Arguments[1], "nvme")
			},
		},
		{
			name:  "read from hdd",
			input: "read 500 MB from hdd\n",
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				t.Helper()
				fc := expectFunctionCall(t, nodes, "read", 2)
				if fc == nil {
					return
				}
				expectIdentifierArg(t, fc.Arguments[1], "hdd")
			},
		},
		{
			name:  "case insensitive - mixed case",
			input: "Read 100 MB From ssd\n",
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				t.Helper()
				fc := expectFunctionCall(t, nodes, "read", 2)
				if fc == nil {
					return
				}
				expectIdentifierArg(t, fc.Arguments[1], "ssd")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, nodes)
			}
		})
	}
}

// TestNLCompressFunction tests "compress <quantity> using <algorithm>" syntax.
func TestNLCompressFunction(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		checkFunc   func(*testing.T, []ast.Node)
	}{
		{
			name:  "basic compress using gzip",
			input: "compress 1 GB using gzip\n",
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				t.Helper()
				fc := expectFunctionCall(t, nodes, "compress", 2)
				if fc == nil {
					return
				}
				expectQuantityArg(t, fc.Arguments[0], "1", "GB")
				expectIdentifierArg(t, fc.Arguments[1], "gzip")
			},
		},
		{
			name:  "compress using lz4",
			input: "compress 500 MB using lz4\n",
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				t.Helper()
				fc := expectFunctionCall(t, nodes, "compress", 2)
				if fc == nil {
					return
				}
				expectIdentifierArg(t, fc.Arguments[1], "lz4")
			},
		},
		{
			name:  "case insensitive - upper case",
			input: "COMPRESS 1 GB USING GZIP\n",
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				t.Helper()
				fc := expectFunctionCall(t, nodes, "compress", 2)
				if fc == nil {
					return
				}
				expectIdentifierArg(t, fc.Arguments[1], "GZIP")
			},
		},
		{
			name:        "missing algorithm after using",
			input:       "compress 1 GB using\n",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, nodes)
			}
		})
	}
}

// TestNLTransferFunction tests "transfer <quantity> across <scope> <network>" syntax.
func TestNLTransferFunction(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		checkFunc   func(*testing.T, []ast.Node)
	}{
		{
			name:  "basic transfer across regional gigabit",
			input: "transfer 1 GB across regional gigabit\n",
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				t.Helper()
				fc := expectFunctionCall(t, nodes, "transfer_time", 3)
				if fc == nil {
					return
				}
				expectQuantityArg(t, fc.Arguments[0], "1", "GB")
				expectIdentifierArg(t, fc.Arguments[1], "regional")
				expectIdentifierArg(t, fc.Arguments[2], "gigabit")
			},
		},
		{
			name:  "transfer across global wifi",
			input: "transfer 500 MB across global wifi\n",
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				t.Helper()
				fc := expectFunctionCall(t, nodes, "transfer_time", 3)
				if fc == nil {
					return
				}
				expectIdentifierArg(t, fc.Arguments[1], "global")
				expectIdentifierArg(t, fc.Arguments[2], "wifi")
			},
		},
		{
			name:  "case insensitive",
			input: "Transfer 1 GB Across Regional Gigabit\n",
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				t.Helper()
				fc := expectFunctionCall(t, nodes, "transfer_time", 3)
				if fc == nil {
					return
				}
			},
		},
		{
			name:        "missing network type after scope",
			input:       "transfer 1 GB across regional\n",
			expectError: true,
		},
		{
			name:        "missing scope and network after across",
			input:       "transfer 1 GB across\n",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, nodes)
			}
		})
	}
}

// TestNLFunctionBackwardsCompatibility ensures parenthesized syntax and
// variable usage of "read", "compress", "transfer", "using", "across" still work.
func TestNLFunctionBackwardsCompatibility(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		checkFunc   func(*testing.T, []ast.Node)
	}{
		{
			name:  "read() parenthesized still works",
			input: "read(100 MB, ssd)\n",
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				t.Helper()
				fc := expectFunctionCall(t, nodes, "read", 2)
				if fc == nil {
					return
				}
			},
		},
		{
			name:  "compress() parenthesized still works",
			input: "compress(1 GB, gzip)\n",
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				t.Helper()
				fc := expectFunctionCall(t, nodes, "compress", 2)
				if fc == nil {
					return
				}
			},
		},
		{
			name:  "transfer_time() parenthesized still works",
			input: "transfer_time(1 GB, regional, gigabit)\n",
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				t.Helper()
				fc := expectFunctionCall(t, nodes, "transfer_time", 3)
				if fc == nil {
					return
				}
			},
		},
		{
			name:  "using as variable assignment",
			input: "using = 5\n",
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				t.Helper()
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				assign, ok := nodes[0].(*ast.Assignment)
				if !ok {
					t.Fatalf("Expected Assignment, got %T", nodes[0])
				}
				if assign.Name != "using" {
					t.Errorf("Expected variable name 'using', got %q", assign.Name)
				}
			},
		},
		{
			name:  "across as variable assignment",
			input: "across = 10\n",
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				t.Helper()
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				assign, ok := nodes[0].(*ast.Assignment)
				if !ok {
					t.Fatalf("Expected Assignment, got %T", nodes[0])
				}
				if assign.Name != "across" {
					t.Errorf("Expected variable name 'across', got %q", assign.Name)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, nodes)
			}
		})
	}
}

// TestNLFunctionOperatorComposition tests NL functions in expression context.
func TestNLFunctionOperatorComposition(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		checkFunc   func(*testing.T, []ast.Node)
	}{
		{
			name:  "compress multiplied by 3",
			input: "compress 1 GB using gzip * 3\n",
			checkFunc: func(t *testing.T, nodes []ast.Node) {
				t.Helper()
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				binOp, ok := nodes[0].(*ast.BinaryOp)
				if !ok {
					t.Fatalf("Expected BinaryOp at top level, got %T", nodes[0])
				}
				if binOp.Operator != "*" {
					t.Errorf("Expected '*' operator, got %q", binOp.Operator)
				}
				fc, ok := binOp.Left.(*ast.FunctionCall)
				if !ok {
					t.Fatalf("Expected FunctionCall on left of *, got %T", binOp.Left)
				}
				if fc.Name != "compress" {
					t.Errorf("Expected 'compress' function, got %q", fc.Name)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, nodes)
			}
		})
	}
}

// --- Test helpers ---

func expectFunctionCall(t *testing.T, nodes []ast.Node, name string, argCount int) *ast.FunctionCall {
	t.Helper()
	if len(nodes) == 0 {
		t.Fatal("Expected at least 1 node, got 0")
		return nil
	}
	fc, ok := nodes[0].(*ast.FunctionCall)
	if !ok {
		t.Fatalf("Expected FunctionCall, got %T", nodes[0])
		return nil
	}
	if fc.Name != name {
		t.Errorf("Expected function name %q, got %q", name, fc.Name)
	}
	if len(fc.Arguments) != argCount {
		t.Errorf("Expected %d arguments, got %d", argCount, len(fc.Arguments))
		return nil
	}
	return fc
}

func expectQuantityArg(t *testing.T, node ast.Node, value, unit string) {
	t.Helper()
	qty, ok := node.(*ast.QuantityLiteral)
	if !ok {
		t.Errorf("Expected QuantityLiteral, got %T", node)
		return
	}
	if qty.Unit != unit {
		t.Errorf("Expected unit %q, got %q", unit, qty.Unit)
	}
}

func expectIdentifierArg(t *testing.T, node ast.Node, name string) {
	t.Helper()
	ident, ok := node.(*ast.Identifier)
	if !ok {
		t.Errorf("Expected Identifier, got %T", node)
		return
	}
	if ident.Name != name {
		t.Errorf("Expected identifier %q, got %q", name, ident.Name)
	}
}
