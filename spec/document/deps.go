package document

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// DependencyAnalyzer extracts variable dependencies from CalcBlocks.
type DependencyAnalyzer struct{}

// NewDependencyAnalyzer creates a new dependency analyzer.
func NewDependencyAnalyzer() *DependencyAnalyzer {
	return &DependencyAnalyzer{}
}

// AnalyzeBlock parses a CalcBlock and extracts:
// - Variables defined (from assignments)
// - Variables referenced (from expressions)
func (da *DependencyAnalyzer) AnalyzeBlock(block *CalcBlock) error {
	if block == nil {
		return fmt.Errorf("nil block")
	}

	// Join source lines for parsing
	source := ""
	for _, line := range block.source {
		source += line + "\n"
	}

	// Parse the source
	nodes, err := parser.Parse(source)
	if err != nil {
		block.SetError(err)
		return err
	}

	// Store parsed statements
	block.SetStatements(nodes)

	// Extract defined and referenced variables.
	// Use a map to track defined variables (deduplicates reassignments).
	definedSet := make(map[string]bool)
	definedOrder := []string{} // Preserve first-definition order
	referenced := make(map[string]bool)

	for _, node := range nodes {
		// Find variable definitions (assignments)
		if assignment, ok := node.(*ast.Assignment); ok {
			if !definedSet[assignment.Name] {
				definedSet[assignment.Name] = true
				definedOrder = append(definedOrder, assignment.Name)
			}
		}

		// Find variable references (identifiers)
		extractIdentifiers(node, referenced)
	}

	// Remove self-references (variables defined in this block)
	dependencies := []string{}
	for varName := range referenced {
		if !definedSet[varName] {
			dependencies = append(dependencies, varName)
		}
	}

	block.SetVariables(definedOrder)
	block.SetDependencies(dependencies)

	return nil
}

// extractIdentifiers recursively finds all identifier references in an AST.
func extractIdentifiers(node ast.Node, identifiers map[string]bool) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *ast.Identifier:
		identifiers[n.Name] = true

	case *ast.Expression:
		// Expression is a wrapper - recurse into its nested node
		extractIdentifiers(n.Expr, identifiers)

	case *ast.Assignment:
		// Don't include the assigned variable, but do include RHS
		extractIdentifiers(n.Value, identifiers)

	case *ast.BinaryOp:
		extractIdentifiers(n.Left, identifiers)
		extractIdentifiers(n.Right, identifiers)

	case *ast.UnaryOp:
		extractIdentifiers(n.Operand, identifiers)

	case *ast.ComparisonOp:
		extractIdentifiers(n.Left, identifiers)
		extractIdentifiers(n.Right, identifiers)

	case *ast.FunctionCall:
		for _, arg := range n.Arguments {
			extractIdentifiers(arg, identifiers)
		}

	// Literals don't have identifiers
	case *ast.NumberLiteral,
		*ast.CurrencyLiteral,
		*ast.BooleanLiteral,
		*ast.DateLiteral,
		*ast.TimeLiteral,
		*ast.DurationLiteral,
		*ast.QuantityLiteral:
		// No identifiers in literals

	default:
		// Unknown node type - skip
	}
}
