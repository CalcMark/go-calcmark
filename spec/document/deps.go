package document

import (
	"fmt"
	"slices"
	"strings"

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
	source := strings.Join(block.source, "\n") + "\n"

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

// ExtractStatementReferences returns the sorted set of variable names
// referenced by a single AST statement. For assignments, only RHS
// references are included (the defined variable is excluded by
// extractIdentifiers). Returns nil for a nil node.
// O(n) time where n is the number of AST nodes in the statement.
func ExtractStatementReferences(node ast.Node) []string {
	if node == nil {
		return nil
	}
	refs := make(map[string]bool)
	extractIdentifiers(node, refs)
	if len(refs) == 0 {
		return nil
	}
	result := make([]string, 0, len(refs))
	for name := range refs {
		result = append(result, name)
	}
	slices.Sort(result)
	return result
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

	case *ast.UnitConversion:
		// Recurse into Quantity; TargetUnit is a string, not a node.
		extractIdentifiers(n.Quantity, identifiers)

	case *ast.NapkinConversion:
		extractIdentifiers(n.Expression, identifiers)

	case *ast.PercentageOf:
		extractIdentifiers(n.Percentage, identifiers)
		extractIdentifiers(n.Value, identifiers)

	case *ast.AsPercentOf:
		extractIdentifiers(n.Numerator, identifiers)
		extractIdentifiers(n.Denominator, identifiers)

	case *ast.RateLiteral:
		extractIdentifiers(n.Amount, identifiers)

	// Literals don't have identifiers
	case *ast.NumberLiteral,
		*ast.CurrencyLiteral,
		*ast.BooleanLiteral,
		*ast.DateLiteral,
		*ast.TimeLiteral,
		*ast.DurationLiteral,
		*ast.QuantityLiteral,
		*ast.RelativeDateLiteral:
		// No identifiers in literals

	default:
		// Unknown node type - skip
	}
}
