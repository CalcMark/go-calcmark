package document

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestASTStructure debugs the actual AST structure.
func TestASTStructure(t *testing.T) {
	source := "y = x + 5\n"

	nodes, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(nodes) == 0 {
		t.Fatal("No nodes parsed")
	}

	// Inspect the assignment
	assignment, ok := nodes[0].(*ast.Assignment)
	if !ok {
		t.Fatalf("Expected Assignment, got %T", nodes[0])
	}

	t.Logf("Assignment.Name = %q", assignment.Name)
	t.Logf("Assignment.Value type = %T", assignment.Value)
	t.Logf("Assignment.Value = %v", assignment.Value)

	// Try to traverse manually
	walkAndPrint(t, assignment.Value, "  ")
}

func walkAndPrint(t *testing.T, node ast.Node, indent string) {
	if node == nil {
		return
	}

	t.Logf("%s%T: %v", indent, node, node)

	switch n := node.(type) {
	case *ast.BinaryOp:
		t.Logf("%sLeft:", indent)
		walkAndPrint(t, n.Left, indent+"  ")
		t.Logf("%sRight:", indent)
		walkAndPrint(t, n.Right, indent+"  ")
	case *ast.Identifier:
		t.Logf("%s  -> Found identifier: %q", indent, n.Name)
	}
}
