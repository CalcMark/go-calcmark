package parser

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
)

func TestParse_MemberAccess(t *testing.T) {
	nodes, err := Parse("x = sales.q1\n")
	if err != nil {
		t.Fatal(err)
	}
	ma, ok := nodes[0].(*ast.Assignment).Value.(*ast.MemberAccess)
	if !ok {
		t.Fatalf("value = %T, want *ast.MemberAccess", nodes[0].(*ast.Assignment).Value)
	}
	obj, ok := ma.Object.(*ast.Identifier)
	if !ok || obj.Name != "sales" || ma.Field != "q1" {
		t.Errorf("MemberAccess = %s", ma)
	}
	if r := ma.GetRange(); r == nil || r.Start.Column != 5 || r.End.Column != 13 {
		t.Errorf("range = %v, want columns 5-13 spanning `sales.q1`", r)
	}
}

func TestParse_MemberAccessInsideExpressionsAndCalls(t *testing.T) {
	for _, src := range []string{
		"cost = rates.rate * rates.hc\n",
		"total = sum(rates.rate * rates.hc)\n",
		"n = count(rates.role)\n",
		"lo = min(rates.rate)\n",
		"hi = max(rates.rate, 10)\n",
	} {
		if _, err := Parse(src); err != nil {
			t.Errorf("Parse(%q): %v", src, err)
		}
	}
}

func TestParse_SumAcceptsSingleArgument(t *testing.T) {
	// `sum(array)` is legal now; arity for scalars is checked by the
	// interpreter, where the argument type is known.
	if _, err := Parse("total = sum(costs)\n"); err != nil {
		t.Errorf("sum(single) must parse: %v", err)
	}
	if _, err := Parse("total = sum()\n"); err == nil {
		t.Error("sum() with no arguments must still be rejected")
	}
}

func TestParse_NestedMemberAccessRejected(t *testing.T) {
	_, err := Parse("x = a.b.c\n")
	if err == nil || !strings.Contains(err.Error(), "nested") {
		t.Errorf("want nested-access error, got %v", err)
	}
}

func TestParse_DirectiveRefUnchanged(t *testing.T) {
	nodes, err := Parse("x = @globals.tax_rate\n")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := nodes[0].(*ast.Assignment).Value.(*ast.DirectiveRef); !ok {
		t.Errorf("@globals.field must still parse as DirectiveRef, got %T", nodes[0].(*ast.Assignment).Value)
	}
}
