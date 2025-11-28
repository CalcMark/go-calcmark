package semantic

import (
	"strings"

	"github.com/CalcMark/go-calcmark/spec/ast"
)

// Checker performs semantic validation on AST nodes.
type Checker struct {
	env         *Environment
	diagnostics []Diagnostic
}

// NewChecker creates a new semantic checker with an empty environment.
func NewChecker() *Checker {
	return &Checker{
		env:         NewEnvironment(),
		diagnostics: make([]Diagnostic, 0),
	}
}

// NewCheckerWithEnv creates a new checker with a pre-populated environment.
// Useful for continuing validation with existing variable bindings.
func NewCheckerWithEnv(env *Environment) *Checker {
	return &Checker{
		env:         env,
		diagnostics: make([]Diagnostic, 0),
	}
}

// Check validates a list of AST nodes and returns all diagnostics found.
// This is the main entry point for semantic validation.
func (c *Checker) Check(nodes []ast.Node) []Diagnostic {
	for _, node := range nodes {
		c.checkNode(node)
	}
	return c.diagnostics
}

// checkNode validates a single AST node.
func (c *Checker) checkNode(node ast.Node) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *ast.Assignment:
		c.checkAssignment(n)
	case *ast.Expression:
		c.checkExpression(n.Expr)
	case *ast.BinaryOp:
		c.checkBinaryOp(n)
	case *ast.ComparisonOp:
		c.checkComparisonOp(n)
	case *ast.UnaryOp:
		c.checkUnaryOp(n)
	case *ast.Identifier:
		c.checkIdentifier(n)
	case *ast.FunctionCall:
		c.checkFunctionCall(n)
	// Literals don't need semantic checking (they're syntactically valid)
	case *ast.NumberLiteral, *ast.CurrencyLiteral, *ast.BooleanLiteral:
		// No semantic checks needed for simple literals
	case *ast.DateLiteral:
		c.checkDateLiteral(n) // USER REQUIREMENT: Validate dates
	case *ast.RelativeDateLiteral:
		// Validated by lexer/parser
	case *ast.TimeLiteral, *ast.DurationLiteral:
		// Validated at parse time
	case *ast.QuantityLiteral:
		c.checkQuantityLiteral(n)
	case *ast.RateLiteral:
		c.checkRateLiteral(n)
	case *ast.UnitConversion:
		c.checkUnitConversion(n)
	case *ast.NapkinConversion:
		c.checkNapkinConversion(n)
	case *ast.PercentageOf:
		c.checkPercentageOf(n)
	}
}

// checkAssignment validates variable assignments.
func (c *Checker) checkAssignment(a *ast.Assignment) {
	// Check the value expression
	c.checkExpression(a.Value)

	// Record the variable in the environment
	// We don't know the actual type yet (that's the interpreter's job),
	// but we mark it as defined
	c.env.Set(a.Name, nil)
}

// checkExpression validates an expression node.
func (c *Checker) checkExpression(expr ast.Node) {
	c.checkNode(expr)
}

// checkBinaryOp validates binary operations for type compatibility.
func (c *Checker) checkBinaryOp(b *ast.BinaryOp) {
	// Check both operands first
	c.checkExpression(b.Left)
	c.checkExpression(b.Right)

	// USER REQUIREMENT: Check unit compatibility for addition/subtraction
	if b.Operator == "+" || b.Operator == "-" {
		c.checkUnitCompatibility(b.Left, b.Right)
	}

	// Division by zero check (if right operand is a literal zero)
	if b.Operator == "/" || b.Operator == "%" {
		if numLit, ok := b.Right.(*ast.NumberLiteral); ok {
			if numLit.Value == "0" {
				c.addDiagnostic(Diagnostic{
					Severity: Warning,
					Code:     DiagDivisionByZero,
					Message:  "Division by zero will cause a runtime error",
					Range:    b.Range,
				})
			}
		}
	}

	// Note: Full type compatibility checking requires type inference,
	// which we'll implement in the interpreter. The semantic checker
	// focuses on obvious errors like undefined variables and invalid currency codes.
}

// checkComparisonOp validates comparison operations.
func (c *Checker) checkComparisonOp(comp *ast.ComparisonOp) {
	c.checkExpression(comp.Left)
	c.checkExpression(comp.Right)
}

// checkUnaryOp validates unary operations.
func (c *Checker) checkUnaryOp(u *ast.UnaryOp) {
	c.checkExpression(u.Operand)
}

// checkIdentifier validates identifier references.
func (c *Checker) checkIdentifier(id *ast.Identifier) {
	// Check if variable is defined
	if !c.env.Has(id.Name) {
		// Check if it's a boolean keyword (true, false, yes, no, etc.)
		if !isBooleanKeyword(id.Name) {
			c.addDiagnostic(Diagnostic{
				Severity: Error, // ERROR: undefined variables block evaluation
				Code:     DiagUndefinedVariable,
				Message:  `Undefined variable "` + id.Name + `" - it must be defined before use`,
				Range:    id.Range,
			})
		}
	}
}

// checkFunctionCall validates function calls.
func (c *Checker) checkFunctionCall(f *ast.FunctionCall) {
	// Special case: convert_rate's second argument is a time unit identifier,
	switch f.Name {
	case "rtt", "throughput", "seek":
		// These functions take ONLY identifier arguments
		// Skip all validation
		return
	case "convert_rate", "downtime":
		// First argument is an expression, second is an identifier
		if len(f.Arguments) > 0 {
			c.checkExpression(f.Arguments[0])
		}
		return
	case "transfer_time", "read", "compress":
		// First argument is an expression, rest are identifiers
		if len(f.Arguments) > 0 {
			c.checkExpression(f.Arguments[0])
		}
		return
	case "capacity":
		// capacity(demand, capacity_per_unit, unit_identifier, buffer?)
		// First two arguments are expressions, third is an identifier, fourth (optional) is expression
		if len(f.Arguments) > 0 {
			c.checkExpression(f.Arguments[0]) // demand
		}
		if len(f.Arguments) > 1 {
			c.checkExpression(f.Arguments[1]) // capacity_per_unit
		}
		// Skip third argument (unit identifier)
		if len(f.Arguments) > 3 {
			c.checkExpression(f.Arguments[3]) // buffer percentage
		}
		// Check for base mixing between demand and capacity units
		if len(f.Arguments) >= 2 {
			c.checkDataSizeBaseMixingForCapacity(f.Arguments[0], f.Arguments[1])
		}
		return
	}

	// Check all arguments for other functions
	for _, arg := range f.Arguments {
		c.checkExpression(arg)
	}

	// Function existence is checked by the parser, so we don't need to validate it here
}

// checkQuantityLiteral validates quantity literals.
func (c *Checker) checkQuantityLiteral(q *ast.QuantityLiteral) {
	// Quantity literals are valid - we check compatibility during operations
	// No need to error here
}

// checkRateLiteral validates rate literals (e.g., "100 MB/s").
func (c *Checker) checkRateLiteral(r *ast.RateLiteral) {
	// Check the amount expression
	if r.Amount != nil {
		c.checkExpression(r.Amount)
	}
	// Rate time unit is validated at parse time
}

// checkUnitConversion validates unit conversion expressions (e.g., "10 meters in feet").
func (c *Checker) checkUnitConversion(u *ast.UnitConversion) {
	// Check the quantity expression being converted
	if u.Quantity != nil {
		c.checkExpression(u.Quantity)
	}
	// Target unit validity is checked at runtime by the interpreter
}

// checkNapkinConversion validates napkin conversions (e.g., "1234567 as napkin").
func (c *Checker) checkNapkinConversion(n *ast.NapkinConversion) {
	// Check the expression being formatted
	if n.Expression != nil {
		c.checkExpression(n.Expression)
	}
}

// checkPercentageOf validates percentage-of expressions (e.g., "10% of 200").
func (c *Checker) checkPercentageOf(p *ast.PercentageOf) {
	// Check both the percentage and value expressions
	if p.Percentage != nil {
		c.checkExpression(p.Percentage)
	}
	if p.Value != nil {
		c.checkExpression(p.Value)
	}
}

// addDiagnostic adds a diagnostic to the checker's list.
func (c *Checker) addDiagnostic(d Diagnostic) {
	c.diagnostics = append(c.diagnostics, d)
}

// isBooleanKeyword checks if an identifier is a boolean keyword.
func isBooleanKeyword(name string) bool {
	normalized := strings.ToLower(name)
	switch normalized {
	case "true", "false":
		return true
	default:
		return false
	}
}

// GetEnvironment returns the current environment (for testing/debugging).
func (c *Checker) GetEnvironment() *Environment {
	return c.env
}

// checkDataSizeBaseMixingForCapacity checks if demand and capacity units mix binary and decimal bases.
func (c *Checker) checkDataSizeBaseMixingForCapacity(demand, capacity ast.Node) {
	demandUnit := getNodeUnit(demand)
	capacityUnit := getNodeUnit(capacity)

	c.checkDataSizeBaseMixingForUnits(demandUnit, capacityUnit)
}
