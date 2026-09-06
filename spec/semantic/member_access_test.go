package semantic

import (
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/parser"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/shopspring/decimal"
)

// The checker validates the table side of `table.column`; columns are
// only known at evaluation time, so column errors are runtime and
// positioned there (go-calcmark#118, R5a).
func TestChecker_MemberAccess(t *testing.T) {
	hc, _ := types.NewArray([]types.Type{types.NewNumber(decimal.NewFromInt(1))})
	rates, _ := types.NewTable("rates", []string{"hc"}, map[string]*types.Array{"hc": hc})

	nodes, err := parser.Parse("cost = rates.hc * 2\ntotal = sum(rates.hc)\n")
	if err != nil {
		t.Fatal(err)
	}
	c := NewChecker()
	c.GetEnvironment().Set("rates", rates)
	if diags := c.Check(nodes); len(diags) != 0 {
		t.Errorf("known table produced diagnostics: %+v", diags)
	}

	nodes, _ = parser.Parse("x = ghost.col\n")
	c = NewChecker()
	diags := c.Check(nodes)
	if len(diags) != 1 || diags[0].Code != DiagUndefinedVariable || diags[0].Range == nil {
		t.Errorf("unknown table should yield one ranged undefined-variable diagnostic, got %+v", diags)
	}
}
