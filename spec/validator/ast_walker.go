package validator

import (
	"github.com/CalcMark/go-calcmark/spec/ast"
)

// walkFunc is called for each node during AST traversal
type walkFunc func(ast.Node)

// walkAST performs depth-first traversal of an AST
// This is the core recursive tree walker that all validators use
func walkAST(node ast.Node, fn walkFunc) {
	if node == nil {
		return
	}

	fn(node)

	switch n := node.(type) {
	case *ast.BinaryOp:
		walkAST(n.Left, fn)
		walkAST(n.Right, fn)

	case *ast.ComparisonOp:
		walkAST(n.Left, fn)
		walkAST(n.Right, fn)

	case *ast.Assignment:
		// Only walk the RHS, not the variable being assigned
		walkAST(n.Value, fn)

	case *ast.Expression:
		walkAST(n.Expr, fn)

	case *ast.UnaryOp:
		walkAST(n.Operand, fn)

	case *ast.FunctionCall:
		for _, arg := range n.Arguments {
			walkAST(arg, fn)
		}
	}
}

// collectNodes collects all nodes of a specific type during traversal
func collectNodes[T ast.Node](root ast.Node, predicate func(ast.Node) (T, bool)) []T {
	var collected []T

	walkAST(root, func(node ast.Node) {
		if matched, ok := predicate(node); ok {
			collected = append(collected, matched)
		}
	})

	return collected
}
