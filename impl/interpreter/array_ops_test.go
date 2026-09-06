package interpreter

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
	"github.com/CalcMark/go-calcmark/v2/spec/parser"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/shopspring/decimal"
)

// Table columns are Arrays; arithmetic on them is element-wise with
// scalar broadcasting, and aggregates reduce them (go-calcmark#118,
// R7–R12).

func ratesInterpreter(t *testing.T) *Interpreter {
	t.Helper()
	cur := func(v int64) types.Type { return types.NewCurrency(decimal.NewFromInt(v), "$") }
	num := func(v int64) types.Type { return types.NewNumber(decimal.NewFromInt(v)) }
	rate, _ := types.NewArray([]types.Type{cur(250), cur(150)})
	hc, _ := types.NewArray([]types.Type{num(2), num(5)})
	role, _ := types.NewArray([]types.Type{types.NewText("Senior"), types.NewText("Junior")})
	three, _ := types.NewArray([]types.Type{num(1), num(2), num(3)})
	empty, _ := types.NewArray(nil)
	rates, err := types.NewTable("rates", []string{"role", "rate", "hc"}, map[string]*types.Array{"role": role, "rate": rate, "hc": hc})
	if err != nil {
		t.Fatal(err)
	}
	env := NewEnvironment()
	env.Set("rates", rates)
	env.Set("three", three)
	env.Set("empty", empty)
	env.Set("n", num(5))
	return NewInterpreterWithEnv(env)
}

func evalOne(t *testing.T, interp *Interpreter, src string) (types.Type, error) {
	t.Helper()
	nodes, err := parser.Parse(src + "\n")
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		return nil, err
	}
	return results[0], nil
}

func wantArray(t *testing.T, v types.Type, err error, want string) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := v.(*types.Array)
	if !ok {
		t.Fatalf("got %T (%v), want *types.Array", v, v)
	}
	if arr.String() != want {
		t.Errorf("got %s, want %s", arr.String(), want)
	}
}

func wantError(t *testing.T, err error, fragments ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error containing %q", fragments)
	}
	for _, f := range fragments {
		if !strings.Contains(err.Error(), f) {
			t.Errorf("error %q does not mention %q", err.Error(), f)
		}
	}
}

func TestMemberAccess_ReadsColumnArray(t *testing.T) {
	v, err := evalOne(t, ratesInterpreter(t), "x = rates.hc")
	wantArray(t, v, err, "[2, 5]")
}

func TestMemberAccess_Errors(t *testing.T) {
	interp := ratesInterpreter(t)
	_, err := evalOne(t, interp, "x = rates.nope")
	wantError(t, err, "column", "nope", "rates", "role, rate, hc")
	_, err = evalOne(t, interp, "y = n.field")
	wantError(t, err, "n", "not a table")
	_, err = evalOne(t, interp, "z = ghost.col")
	wantError(t, err, "ghost")
	if r := PositionOf(err); r == nil {
		t.Error("member access errors must be positioned")
	}
}

func TestElementWise_ArrayTimesArray(t *testing.T) {
	v, err := evalOne(t, ratesInterpreter(t), "cost = rates.rate * rates.hc")
	wantArray(t, v, err, "[$500.00, $750.00]")
}

func TestElementWise_Broadcast(t *testing.T) {
	interp := ratesInterpreter(t)
	v, err := evalOne(t, interp, "a = rates.hc * 2")
	wantArray(t, v, err, "[4, 10]")
	v, err = evalOne(t, interp, "b = 100 - rates.hc")
	wantArray(t, v, err, "[98, 95]")
	v, err = evalOne(t, interp, "c = -rates.hc")
	wantArray(t, v, err, "[-2, -5]")
}

func TestElementWise_LengthMismatch(t *testing.T) {
	_, err := evalOne(t, ratesInterpreter(t), "x = rates.hc + three")
	wantError(t, err, "2", "3", "length")
}

func TestElementWise_TextColumnRejected(t *testing.T) {
	_, err := evalOne(t, ratesInterpreter(t), "x = rates.role * 2")
	wantError(t, err, "text")
}

func TestAggregates_OverArrays(t *testing.T) {
	interp := ratesInterpreter(t)
	cases := []struct{ src, want string }{
		{"s = sum(rates.hc)", "7"},
		{"t = sum(rates.rate * rates.hc)", "$1250.00"},
		{"a = avg(rates.hc)", "3.5"},
		{"lo = min(rates.rate)", "$150.00"},
		{"hi = max(rates.hc)", "5"},
		{"c = count(rates.role)", "2"},
		{"m = max(three)", "3"},
	}
	for _, c := range cases {
		v, err := evalOne(t, interp, c.src)
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
			continue
		}
		if v.String() != c.want {
			t.Errorf("%s = %s, want %s", c.src, v.String(), c.want)
		}
	}
}

func TestAggregates_ScalarFormsUnchanged(t *testing.T) {
	interp := ratesInterpreter(t)
	cases := []struct{ src, want string }{
		{"s = sum(1, 2, 3)", "6"},
		{"a = avg(10, 20)", "15"},
		{"lo = min(4, 9, 1)", "1"},
		{"hi = max(4, 9, 1)", "9"},
		{"c = count(4, 9, 1)", "3"},
	}
	for _, c := range cases {
		v, err := evalOne(t, interp, c.src)
		if err != nil {
			t.Errorf("%s: %v", c.src, err)
			continue
		}
		if v.String() != c.want {
			t.Errorf("%s = %s, want %s", c.src, v.String(), c.want)
		}
	}
}

func TestAggregates_Errors(t *testing.T) {
	interp := ratesInterpreter(t)
	_, err := evalOne(t, interp, "x = sum(5)")
	wantError(t, err, "sum()", "2 arguments")
	_, err = evalOne(t, interp, "y = sum(rates.hc, 1)")
	wantError(t, err, "array")
	_, err = evalOne(t, interp, "z = min(empty)")
	wantError(t, err, "empty")
	_, err = evalOne(t, interp, "w = sum(rates.role)")
	wantError(t, err, "text")
}

func TestMemberAccess_NodeIsRanged(t *testing.T) {
	nodes, err := parser.Parse("x = rates.rate\n")
	if err != nil {
		t.Fatal(err)
	}
	ma := nodes[0].(*ast.Assignment).Value.(*ast.MemberAccess)
	if ma.GetRange() == nil || ma.GetRange().Start.Column != 5 {
		t.Errorf("range = %v", ma.GetRange())
	}
}
