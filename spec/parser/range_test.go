package parser

import (
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
)

// Every leaf expression carries a source range so diagnostics can point
// at the exact token (go-calcmark#164). Columns are 1-based; End is
// exclusive, matching the convention Identifier and FunctionCall already
// used.

func assertRange(t *testing.T, label string, n ast.Node, startCol, endCol int) {
	t.Helper()
	r := n.GetRange()
	if r == nil || r.Start.Line == 0 {
		t.Fatalf("%s (%T): no range", label, n)
	}
	if r.Start.Column != startCol || r.End.Column != endCol {
		t.Errorf("%s (%T): columns %d-%d, want %d-%d", label, n, r.Start.Column, r.End.Column, startCol, endCol)
	}
}

func TestParse_FunctionArgumentsHaveRanges(t *testing.T) {
	nodes, err := Parse("x = grow(100, 20%, 5)\n")
	if err != nil {
		t.Fatal(err)
	}
	call := nodes[0].(*ast.Assignment).Value.(*ast.FunctionCall)
	//        1234567890123456789012
	// source x = grow(100, 20%, 5)
	assertRange(t, "100", call.Arguments[0], 10, 13)
	assertRange(t, "20%", call.Arguments[1], 15, 18)
	assertRange(t, "5", call.Arguments[2], 20, 21)
}

func TestParse_LiteralsHaveRanges(t *testing.T) {
	cases := []struct {
		src      string
		startCol int
		endCol   int
	}{
		{"x = $5\n", 5, 7},
		{"x = 1,000\n", 5, 10},
		{"x = 10 meters\n", 5, 14},
		{"x = 2 weeks\n", 5, 12},
		{"x = 100 MB/s\n", 5, 13},
		{"x = true\n", 5, 9},
		{"x = 1/3\n", 5, 8},
		{"x = 2026-01-15\n", 5, 15},
	}
	for _, c := range cases {
		nodes, err := Parse(c.src)
		if err != nil {
			t.Fatalf("Parse(%q): %v", c.src, err)
		}
		value := nodes[0].(*ast.Assignment).Value
		assertRange(t, c.src, value, c.startCol, c.endCol)
	}
}

func TestParse_BinaryOpSpansOperands(t *testing.T) {
	nodes, err := Parse("x = 1 + $5 / $2\n")
	if err != nil {
		t.Fatal(err)
	}
	//        123456789012345
	// source x = 1 + $5 / $2
	outer := nodes[0].(*ast.Assignment).Value.(*ast.BinaryOp)
	assertRange(t, "1 + $5 / $2", outer, 5, 16)
	assertRange(t, "$5 / $2", outer.Right, 9, 16)
}
