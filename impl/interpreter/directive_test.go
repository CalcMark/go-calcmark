package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/shopspring/decimal"
)

// mockDirectiveResolver implements DirectiveResolver for testing.
type mockDirectiveResolver struct {
	scaleFactor *decimal.Decimal
	globals     map[string]types.Type
}

func (m *mockDirectiveResolver) ScaleFactor() (decimal.Decimal, bool) {
	if m.scaleFactor == nil {
		return decimal.Zero, false
	}
	return *m.scaleFactor, true
}

func (m *mockDirectiveResolver) ResolveGlobal(name string) (types.Type, bool, error) {
	if m.globals == nil {
		return nil, false, nil
	}
	v, ok := m.globals[name]
	return v, ok, nil
}

func decPtr(v decimal.Decimal) *decimal.Decimal { return &v }

func TestEvalDirectiveRef_ScaleNumeric(t *testing.T) {
	interp := NewInterpreter()
	interp.SetDirectiveResolver(&mockDirectiveResolver{
		scaleFactor: decPtr(decimal.NewFromInt(3)),
	})

	results, err := interp.Eval([]ast.Node{
		&ast.Expression{Expr: &ast.DirectiveRef{Directive: "scale"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	num, ok := results[0].(*types.Number)
	if !ok {
		t.Fatalf("expected *types.Number, got %T", results[0])
	}
	if !num.Value.Equal(decimal.NewFromInt(3)) {
		t.Errorf("expected 3, got %s", num.Value)
	}
}

func TestEvalDirectiveRef_ScaleMapForm(t *testing.T) {
	interp := NewInterpreter()
	interp.SetDirectiveResolver(&mockDirectiveResolver{
		scaleFactor: decPtr(decimal.NewFromInt(4)),
	})

	results, err := interp.Eval([]ast.Node{
		&ast.Expression{Expr: &ast.DirectiveRef{Directive: "scale"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	num := results[0].(*types.Number)
	if !num.Value.Equal(decimal.NewFromInt(4)) {
		t.Errorf("expected 4, got %s", num.Value)
	}
}

func TestEvalDirectiveRef_GlobalsNumeric(t *testing.T) {
	interp := NewInterpreter()
	interp.SetDirectiveResolver(&mockDirectiveResolver{
		globals: map[string]types.Type{
			"tax_rate": types.NewNumber(decimal.NewFromFloat(0.32)),
		},
	})

	results, err := interp.Eval([]ast.Node{
		&ast.Expression{Expr: &ast.DirectiveRef{Directive: "globals", Field: "tax_rate"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	num, ok := results[0].(*types.Number)
	if !ok {
		t.Fatalf("expected *types.Number, got %T", results[0])
	}
	expected := decimal.NewFromFloat(0.32)
	if !num.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, num.Value)
	}
}

func TestEvalDirectiveRef_GlobalsCurrency(t *testing.T) {
	interp := NewInterpreter()
	interp.SetDirectiveResolver(&mockDirectiveResolver{
		globals: map[string]types.Type{
			"budget": types.NewCurrency(decimal.NewFromInt(5000), "$"),
		},
	})

	results, err := interp.Eval([]ast.Node{
		&ast.Expression{Expr: &ast.DirectiveRef{Directive: "globals", Field: "budget"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cur, ok := results[0].(*types.Currency)
	if !ok {
		t.Fatalf("expected *types.Currency, got %T", results[0])
	}
	if cur.Symbol != "$" {
		t.Errorf("expected symbol $, got %s", cur.Symbol)
	}
	if !cur.Value.Equal(decimal.NewFromInt(5000)) {
		t.Errorf("expected 5000, got %s", cur.Value)
	}
}

func TestEvalDirectiveRef_ArithmeticWithScale(t *testing.T) {
	interp := NewInterpreter()
	interp.SetDirectiveResolver(&mockDirectiveResolver{
		scaleFactor: decPtr(decimal.NewFromInt(2)),
	})

	// x = 100
	// y = x * @scale  (should be 200)
	nodes := []ast.Node{
		&ast.Assignment{
			Name:  "x",
			Value: &ast.NumberLiteral{Value: "100"},
		},
		&ast.Assignment{
			Name: "y",
			Value: &ast.BinaryOp{
				Left:     &ast.Identifier{Name: "x"},
				Operator: "*",
				Right:    &ast.DirectiveRef{Directive: "scale"},
			},
		},
	}

	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	y, ok := results[1].(*types.Number)
	if !ok {
		t.Fatalf("expected *types.Number for y, got %T", results[1])
	}
	if !y.Value.Equal(decimal.NewFromInt(200)) {
		t.Errorf("expected 200, got %s", y.Value)
	}
}

func TestEvalDirectiveRef_ArithmeticWithGlobals(t *testing.T) {
	interp := NewInterpreter()
	interp.SetDirectiveResolver(&mockDirectiveResolver{
		globals: map[string]types.Type{
			"tax_rate": types.NewNumber(decimal.NewFromFloat(0.1)),
		},
	})

	// income = 1000
	// tax = income * @globals.tax_rate  (should be 100)
	nodes := []ast.Node{
		&ast.Assignment{
			Name:  "income",
			Value: &ast.NumberLiteral{Value: "1000"},
		},
		&ast.Assignment{
			Name: "tax",
			Value: &ast.BinaryOp{
				Left:     &ast.Identifier{Name: "income"},
				Operator: "*",
				Right:    &ast.DirectiveRef{Directive: "globals", Field: "tax_rate"},
			},
		},
	}

	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tax, ok := results[1].(*types.Number)
	if !ok {
		t.Fatalf("expected *types.Number for tax, got %T", results[1])
	}
	if !tax.Value.Equal(decimal.NewFromInt(100)) {
		t.Errorf("expected 100, got %s", tax.Value)
	}
}

func TestEvalDirectiveRef_NoFrontmatter(t *testing.T) {
	interp := NewInterpreter()
	// No directive resolver set

	_, err := interp.Eval([]ast.Node{
		&ast.Expression{Expr: &ast.DirectiveRef{Directive: "scale"}},
	})
	if err == nil {
		t.Fatal("expected error for @scale without frontmatter")
	}
}

func TestEvalDirectiveRef_ScaleWithoutScaleConfig(t *testing.T) {
	interp := NewInterpreter()
	interp.SetDirectiveResolver(&mockDirectiveResolver{
		// No scaleFactor set
		globals: map[string]types.Type{
			"x": types.NewNumber(decimal.NewFromInt(1)),
		},
	})

	_, err := interp.Eval([]ast.Node{
		&ast.Expression{Expr: &ast.DirectiveRef{Directive: "scale"}},
	})
	if err == nil {
		t.Fatal("expected error for @scale without scale config")
	}
}

func TestEvalDirectiveRef_UndefinedGlobal(t *testing.T) {
	interp := NewInterpreter()
	interp.SetDirectiveResolver(&mockDirectiveResolver{
		globals: map[string]types.Type{
			"tax_rate": types.NewNumber(decimal.NewFromFloat(0.1)),
		},
	})

	_, err := interp.Eval([]ast.Node{
		&ast.Expression{Expr: &ast.DirectiveRef{Directive: "globals", Field: "nonexistent"}},
	})
	if err == nil {
		t.Fatal("expected error for undefined global")
	}
}
