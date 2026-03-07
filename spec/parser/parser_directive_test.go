package parser

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
)

func TestDirectiveRef_AtScale(t *testing.T) {
	nodes, err := Parse("x = @scale\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	assign, ok := nodes[0].(*ast.Assignment)
	if !ok {
		t.Fatalf("expected Assignment, got %T", nodes[0])
	}
	ref, ok := assign.Value.(*ast.DirectiveRef)
	if !ok {
		t.Fatalf("expected DirectiveRef, got %T", assign.Value)
	}
	if ref.Directive != "scale" {
		t.Errorf("expected directive 'scale', got %q", ref.Directive)
	}
	if ref.Field != "" {
		t.Errorf("expected empty field, got %q", ref.Field)
	}
}

func TestDirectiveRef_AtGlobalsField(t *testing.T) {
	nodes, err := Parse("x = @globals.tax_rate\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assign := nodes[0].(*ast.Assignment)
	ref, ok := assign.Value.(*ast.DirectiveRef)
	if !ok {
		t.Fatalf("expected DirectiveRef, got %T", assign.Value)
	}
	if ref.Directive != "globals" {
		t.Errorf("expected directive 'globals', got %q", ref.Directive)
	}
	if ref.Field != "tax_rate" {
		t.Errorf("expected field 'tax_rate', got %q", ref.Field)
	}
}

func TestDirectiveRef_AtGlobalsNoDot_Error(t *testing.T) {
	_, err := Parse("x = @globals\n")
	if err == nil {
		t.Fatal("expected error for @globals without field, got nil")
	}
}

func TestDirectiveRef_NestedDots_Error(t *testing.T) {
	_, err := Parse("x = @globals.a.b\n")
	if err == nil {
		t.Fatal("expected error for @globals.a.b (nested dots), got nil")
	}
}

func TestDirectiveRef_UnknownDirective_Parses(t *testing.T) {
	// @exchange should parse — semantic checker will reject
	nodes, err := Parse("x = @exchange\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assign := nodes[0].(*ast.Assignment)
	ref, ok := assign.Value.(*ast.DirectiveRef)
	if !ok {
		t.Fatalf("expected DirectiveRef, got %T", assign.Value)
	}
	if ref.Directive != "exchange" {
		t.Errorf("expected directive 'exchange', got %q", ref.Directive)
	}
}

func TestDirectiveRef_InArithmetic(t *testing.T) {
	// x = 1 + @scale * 2 — verify correct precedence
	nodes, err := Parse("x = 1 + @scale * 2\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assign := nodes[0].(*ast.Assignment)
	// Should be: 1 + (@scale * 2) due to precedence
	binOp, ok := assign.Value.(*ast.BinaryOp)
	if !ok {
		t.Fatalf("expected BinaryOp, got %T", assign.Value)
	}
	if binOp.Operator != "+" {
		t.Errorf("expected '+', got %q", binOp.Operator)
	}
	// Right side should be @scale * 2
	rightBin, ok := binOp.Right.(*ast.BinaryOp)
	if !ok {
		t.Fatalf("expected BinaryOp for right side, got %T", binOp.Right)
	}
	if rightBin.Operator != "*" {
		t.Errorf("expected '*', got %q", rightBin.Operator)
	}
	ref, ok := rightBin.Left.(*ast.DirectiveRef)
	if !ok {
		t.Fatalf("expected DirectiveRef for @scale, got %T", rightBin.Left)
	}
	if ref.Directive != "scale" {
		t.Errorf("expected directive 'scale', got %q", ref.Directive)
	}
}
