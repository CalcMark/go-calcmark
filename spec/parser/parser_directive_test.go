package parser

import (
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
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

// TestDirectiveRef_SelfReferentialExpression tests @scale ^ @scale and
// (@scale + 1) * @scale — multiple references in one expression.
func TestDirectiveRef_SelfReferentialExpression(t *testing.T) {
	t.Run("@scale ^ @scale", func(t *testing.T) {
		nodes, err := Parse("x = @scale ^ @scale\n")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assign := nodes[0].(*ast.Assignment)
		binOp, ok := assign.Value.(*ast.BinaryOp)
		if !ok {
			t.Fatalf("expected BinaryOp, got %T", assign.Value)
		}
		if binOp.Operator != "^" {
			t.Errorf("expected '^', got %q", binOp.Operator)
		}
		leftRef, ok := binOp.Left.(*ast.DirectiveRef)
		if !ok {
			t.Fatalf("expected DirectiveRef on left, got %T", binOp.Left)
		}
		rightRef, ok := binOp.Right.(*ast.DirectiveRef)
		if !ok {
			t.Fatalf("expected DirectiveRef on right, got %T", binOp.Right)
		}
		if leftRef.Directive != "scale" || rightRef.Directive != "scale" {
			t.Errorf("expected both directives to be 'scale', got %q and %q",
				leftRef.Directive, rightRef.Directive)
		}
	})

	t.Run("(@scale + 1) * @scale", func(t *testing.T) {
		nodes, err := Parse("x = (@scale + 1) * @scale\n")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assign := nodes[0].(*ast.Assignment)
		binOp, ok := assign.Value.(*ast.BinaryOp)
		if !ok {
			t.Fatalf("expected BinaryOp (*), got %T", assign.Value)
		}
		if binOp.Operator != "*" {
			t.Errorf("expected '*', got %q", binOp.Operator)
		}
		// Right side should be a DirectiveRef
		rightRef, ok := binOp.Right.(*ast.DirectiveRef)
		if !ok {
			t.Fatalf("expected DirectiveRef on right, got %T", binOp.Right)
		}
		if rightRef.Directive != "scale" {
			t.Errorf("expected directive 'scale', got %q", rightRef.Directive)
		}
	})
}

// TestDirectiveRef_WithUnit tests that @scale meters parses as a QuantityLiteral
// with Expr=DirectiveRef and Unit=meter.
func TestDirectiveRef_WithUnit(t *testing.T) {
	nodes, err := Parse("x = @scale meters\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assign := nodes[0].(*ast.Assignment)
	qty, ok := assign.Value.(*ast.QuantityLiteral)
	if !ok {
		t.Fatalf("expected QuantityLiteral, got %T", assign.Value)
	}
	if qty.Expr == nil {
		t.Fatal("expected Expr to be non-nil (expression-based quantity)")
	}
	ref, ok := qty.Expr.(*ast.DirectiveRef)
	if !ok {
		t.Fatalf("expected DirectiveRef in Expr, got %T", qty.Expr)
	}
	if ref.Directive != "scale" {
		t.Errorf("expected directive 'scale', got %q", ref.Directive)
	}
	if qty.Unit != "meter" {
		t.Errorf("expected unit 'meter', got %q", qty.Unit)
	}
}

// TestDirectiveRef_WithRate tests that @scale meters / second parses as a rate.
func TestDirectiveRef_WithRate(t *testing.T) {
	nodes, err := Parse("x = @scale meters / second\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assign := nodes[0].(*ast.Assignment)
	rate, ok := assign.Value.(*ast.RateLiteral)
	if !ok {
		t.Fatalf("expected RateLiteral, got %T", assign.Value)
	}
	// The amount should be a QuantityLiteral with Expr=DirectiveRef
	qty, ok := rate.Amount.(*ast.QuantityLiteral)
	if !ok {
		t.Fatalf("expected QuantityLiteral in rate.Amount, got %T", rate.Amount)
	}
	if qty.Expr == nil {
		t.Fatal("expected Expr to be non-nil")
	}
	ref, ok := qty.Expr.(*ast.DirectiveRef)
	if !ok {
		t.Fatalf("expected DirectiveRef in qty.Expr, got %T", qty.Expr)
	}
	if ref.Directive != "scale" {
		t.Errorf("expected directive 'scale', got %q", ref.Directive)
	}
}

// TestDirectiveRef_GlobalsWithUnit tests that @globals.rate kg parses correctly.
func TestDirectiveRef_GlobalsWithUnit(t *testing.T) {
	nodes, err := Parse("x = @globals.rate kg\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assign := nodes[0].(*ast.Assignment)
	qty, ok := assign.Value.(*ast.QuantityLiteral)
	if !ok {
		t.Fatalf("expected QuantityLiteral, got %T", assign.Value)
	}
	if qty.Expr == nil {
		t.Fatal("expected Expr to be non-nil")
	}
	ref, ok := qty.Expr.(*ast.DirectiveRef)
	if !ok {
		t.Fatalf("expected DirectiveRef in Expr, got %T", qty.Expr)
	}
	if ref.Directive != "globals" || ref.Field != "rate" {
		t.Errorf("expected globals.rate, got %s.%s", ref.Directive, ref.Field)
	}
	if qty.Unit != "kilogram" {
		t.Errorf("expected unit 'kilogram', got %q", qty.Unit)
	}
}

// TestDirectiveRef_UnitLookaheadSkipsNLKeywords tests that natural syntax
// keywords (grow, at, per, etc.) are NOT consumed as units after a directive.
func TestDirectiveRef_UnitLookaheadSkipsNLKeywords(t *testing.T) {
	keywords := []string{"at", "per", "with", "over", "grow", "as", "compound"}
	for _, kw := range keywords {
		t.Run(kw+" not consumed as unit", func(t *testing.T) {
			// "@scale <keyword>" — the keyword should NOT be consumed as a unit.
			// This may fail to parse fully (because the keyword expects more tokens),
			// but it should NOT produce a QuantityLiteral with Unit=keyword.
			nodes, err := Parse("x = @scale " + kw + "\n")
			if err != nil {
				// Parse error is acceptable — the keyword triggers NL syntax
				// that expects more tokens. The key thing is it didn't silently
				// consume the keyword as a unit.
				return
			}
			assign := nodes[0].(*ast.Assignment)
			if qty, ok := assign.Value.(*ast.QuantityLiteral); ok {
				if qty.Unit == kw {
					t.Errorf("keyword %q was incorrectly consumed as a unit after @scale", kw)
				}
			}
		})
	}
}

// TestDirectiveRef_ContainsScaleRef_InQuantityLiteral tests that
// ContainsScaleRef correctly detects @scale inside a QuantityLiteral.
func TestDirectiveRef_ContainsScaleRef_InQuantityLiteral(t *testing.T) {
	nodes, err := Parse("x = @scale meters\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assign := nodes[0].(*ast.Assignment)
	if !ast.ContainsScaleRef(assign.Value) {
		t.Error("ContainsScaleRef should return true for @scale meters (QuantityLiteral with DirectiveRef)")
	}
}

// TestDirectiveRef_ContainsScaleRef_InRate tests that ContainsScaleRef
// correctly detects @scale inside a RateLiteral wrapping a QuantityLiteral.
func TestDirectiveRef_ContainsScaleRef_InRate(t *testing.T) {
	nodes, err := Parse("x = @scale meters / second\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assign := nodes[0].(*ast.Assignment)
	if !ast.ContainsScaleRef(assign.Value) {
		t.Error("ContainsScaleRef should return true for @scale meters / second")
	}
}
