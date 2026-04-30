package parser

import (
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
)

// unwrapExpr extracts the inner node from an Expression wrapper if present.
func unwrapExpr(node ast.Node) ast.Node {
	if expr, ok := node.(*ast.Expression); ok {
		return expr.Expr
	}
	return node
}

func TestParseFraction(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		check   func(t *testing.T, nodes []ast.Node)
		wantErr bool
	}{
		{
			name:  "simple fraction",
			input: "1/3\n",
			check: func(t *testing.T, nodes []ast.Node) {
				inner := unwrapExpr(nodes[0])
				frac, ok := inner.(*ast.FractionLiteral)
				if !ok {
					t.Fatalf("expected FractionLiteral, got %T", inner)
				}
				if frac.Numerator != 1 || frac.Denominator != 3 {
					t.Errorf("got %d/%d, want 1/3", frac.Numerator, frac.Denominator)
				}
			},
		},
		{
			name:  "division with spaces unchanged",
			input: "1 / 3\n",
			check: func(t *testing.T, nodes []ast.Node) {
				inner := unwrapExpr(nodes[0])
				binop, ok := inner.(*ast.BinaryOp)
				if !ok {
					t.Fatalf("expected BinaryOp, got %T", inner)
				}
				if binop.Operator != "/" {
					t.Errorf("expected / operator, got %s", binop.Operator)
				}
			},
		},
		{
			name:  "mixed number",
			input: "11 3/8\n",
			check: func(t *testing.T, nodes []ast.Node) {
				inner := unwrapExpr(nodes[0])
				binop, ok := inner.(*ast.BinaryOp)
				if !ok {
					t.Fatalf("expected BinaryOp for mixed number, got %T", inner)
				}
				if binop.Operator != "+" {
					t.Errorf("expected + operator for mixed number, got %s", binop.Operator)
				}
				numLit, ok := binop.Left.(*ast.NumberLiteral)
				if !ok {
					t.Fatalf("expected NumberLiteral left, got %T", binop.Left)
				}
				if numLit.Value != "11" {
					t.Errorf("expected integer part 11, got %s", numLit.Value)
				}
				fracLit, ok := binop.Right.(*ast.FractionLiteral)
				if !ok {
					t.Fatalf("expected FractionLiteral right, got %T", binop.Right)
				}
				if fracLit.Numerator != 3 || fracLit.Denominator != 8 {
					t.Errorf("expected fraction 3/8, got %d/%d", fracLit.Numerator, fracLit.Denominator)
				}
			},
		},
		{
			name:  "fraction with unit",
			input: "1/3 cup\n",
			check: func(t *testing.T, nodes []ast.Node) {
				inner := unwrapExpr(nodes[0])
				frac, ok := inner.(*ast.FractionLiteral)
				if !ok {
					t.Fatalf("expected FractionLiteral, got %T", inner)
				}
				if frac.Numerator != 1 || frac.Denominator != 3 {
					t.Errorf("got %d/%d, want 1/3", frac.Numerator, frac.Denominator)
				}
				if frac.Unit != "cup" {
					t.Errorf("got unit %q, want \"cup\"", frac.Unit)
				}
			},
		},
		{
			name:  "mixed number with unit",
			input: "11 3/8 inch\n",
			check: func(t *testing.T, nodes []ast.Node) {
				inner := unwrapExpr(nodes[0])
				binop, ok := inner.(*ast.BinaryOp)
				if !ok {
					t.Fatalf("expected BinaryOp for mixed number, got %T", inner)
				}
				fracLit, ok := binop.Right.(*ast.FractionLiteral)
				if !ok {
					t.Fatalf("expected FractionLiteral right, got %T", binop.Right)
				}
				if fracLit.Unit != "inch" {
					t.Errorf("got unit %q, want \"inch\"", fracLit.Unit)
				}
			},
		},
		{
			name:  "negative fraction via unary",
			input: "-1/3\n",
			check: func(t *testing.T, nodes []ast.Node) {
				inner := unwrapExpr(nodes[0])
				unary, ok := inner.(*ast.UnaryOp)
				if !ok {
					t.Fatalf("expected UnaryOp, got %T", inner)
				}
				if unary.Operator != "-" {
					t.Errorf("expected - operator, got %s", unary.Operator)
				}
				_, ok = unary.Operand.(*ast.FractionLiteral)
				if !ok {
					t.Fatalf("expected FractionLiteral operand, got %T", unary.Operand)
				}
			},
		},
		{
			name:  "fraction in assignment",
			input: "a = 1/3\n",
			check: func(t *testing.T, nodes []ast.Node) {
				assign, ok := nodes[0].(*ast.Assignment)
				if !ok {
					t.Fatalf("expected Assignment, got %T", nodes[0])
				}
				frac, ok := assign.Value.(*ast.FractionLiteral)
				if !ok {
					t.Fatalf("expected FractionLiteral, got %T", assign.Value)
				}
				if frac.Numerator != 1 || frac.Denominator != 3 {
					t.Errorf("got %d/%d, want 1/3", frac.Numerator, frac.Denominator)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(nodes) != 1 {
				t.Fatalf("expected 1 node, got %d", len(nodes))
			}
			tt.check(t, nodes)
		})
	}
}
