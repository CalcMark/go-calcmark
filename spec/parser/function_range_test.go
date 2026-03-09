package parser

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
)

// TestAllFunctionCallsHaveRange verifies that every FunctionCall node produced
// by the parser has a non-nil Range with a positive line number. Without this,
// eval errors cannot be mapped back to their source line in the TUI.
//
// This test covers ALL function syntaxes: functional f(args), NL forms, and
// implicit functions (e.g., "99.9% downtime per month"). Any new function or
// NL syntax that omits Range will fail here before it can cause a diagnostic
// misalignment bug.
func TestAllFunctionCallsHaveRange(t *testing.T) {
	// Every callable function in both functional and NL syntax.
	// Each expression must parse without error.
	expressions := []struct {
		name string
		expr string
	}{
		// Math — functional
		{"avg functional", "avg(1, 2, 3)"},
		{"sum functional", "sum(1, 2, 3)"},
		{"sqrt functional", "sqrt(16)"},

		// Math — NL
		{"average of NL", "average of 1, 2, 3"},
		{"sum of NL", "sum of 1, 2, 3"},
		{"square root of NL", "square root of 16"},

		// Growth — functional
		{"compound functional", "compound($1000, 5%, 10)"},
		{"compound with modifier", "compound($1000, 5%, 10 years, monthly)"},
		{"grow functional", "grow($0, $200, 60)"},
		{"depreciate functional", "depreciate($35000, 20%, 5)"},
		{"depreciate with salvage", "depreciate($35000, 20%, 5, $5000)"},

		// Growth — NL
		{"compound NL", "compound $1000 by 5% over 10"},
		{"compound NL with frequency", "compound $1000 by 5% monthly over 10 years"},
		{"compound NL compounded", "compound $1000 by 5% compounded monthly over 10 years"},
		{"grow NL", "grow $0 by $200 over 60"},
		{"depreciate NL", "depreciate $35000 by 20% over 5"},
		{"depreciate functional with salvage", "depreciate($35000, 20%, 5, $5000)"},

		// Network/Storage — functional
		{"rtt functional", "rtt(regional)"},
		{"throughput functional", "throughput(gigabit)"},
		{"seek functional", "seek(hdd)"},

		// Network/Storage — NL
		{"read NL", "read 100 MB from ssd"},
		{"compress NL", "compress 1 GB using gzip"},
		{"transfer NL", "transfer 1 GB across regional gigabit"},

		// Capacity — NL/implicit
		{"capacity NL", "10 TB at 2 TB per disk"},

		// Rate functions — implicit
		{"downtime implicit", "99.9% downtime per month"},
		{"accumulate implicit", "100 req/s over 1 hour"},
	}

	for _, tt := range expressions {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.expr)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.expr, err)
			}

			// Walk all AST nodes and check every FunctionCall
			var funcCalls []*ast.FunctionCall
			for _, node := range nodes {
				walkAST(node, func(n ast.Node) {
					if fc, ok := n.(*ast.FunctionCall); ok {
						funcCalls = append(funcCalls, fc)
					}
				})
			}

			if len(funcCalls) == 0 {
				t.Fatalf("Parse(%q) produced no FunctionCall nodes", tt.expr)
			}

			for _, fc := range funcCalls {
				if fc.Range == nil {
					t.Errorf("FunctionCall %q has nil Range — eval errors will misalign in TUI", fc.Name)
				} else if fc.Range.Start.Line <= 0 {
					t.Errorf("FunctionCall %q has Range.Start.Line=%d (want > 0)", fc.Name, fc.Range.Start.Line)
				}
			}
		})
	}
}

// walkAST recursively visits all nodes in the AST tree.
func walkAST(node ast.Node, visit func(ast.Node)) {
	if node == nil {
		return
	}
	visit(node)

	switch n := node.(type) {
	case *ast.Assignment:
		walkAST(n.Value, visit)
	case *ast.BinaryOp:
		walkAST(n.Left, visit)
		walkAST(n.Right, visit)
	case *ast.UnaryOp:
		walkAST(n.Operand, visit)
	case *ast.FunctionCall:
		for _, arg := range n.Arguments {
			walkAST(arg, visit)
		}
	case *ast.Expression:
		walkAST(n.Expr, visit)
	case *ast.ComparisonOp:
		walkAST(n.Left, visit)
		walkAST(n.Right, visit)
	case *ast.UnitConversion:
		walkAST(n.Quantity, visit)
	case *ast.NapkinConversion:
		walkAST(n.Expression, visit)
	case *ast.PreciseConversion:
		walkAST(n.Expression, visit)
	case *ast.PercentageOf:
		walkAST(n.Percentage, visit)
		walkAST(n.Value, visit)
	}
}
