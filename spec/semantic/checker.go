package semantic

import (
	"sort"
	"strconv"
	"strings"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
	"github.com/CalcMark/go-calcmark/v2/spec/units"
)

// findSimilarNames returns variable names similar to the given name.
// Uses simple heuristics: case-insensitive match, prefix match, or edit distance for typos.
func findSimilarNames(name string, candidates map[string]struct{}) []string {
	if len(candidates) == 0 {
		return nil
	}

	nameLower := strings.ToLower(name)
	var matches []string

	for candidate := range candidates {
		candidateLower := strings.ToLower(candidate)

		// Exact case-insensitive match (different casing) - highest priority
		if candidateLower == nameLower && candidate != name {
			return []string{candidate} // Perfect match, return immediately
		}

		// Skip very short candidates (built-in constants like E, PI) unless name is also short
		if len(candidate) <= 2 && len(name) > 2 {
			continue
		}

		// Prefix match - user typed beginning of variable (at least 2 chars to match)
		if len(nameLower) >= 2 {
			if strings.HasPrefix(candidateLower, nameLower) {
				matches = append(matches, candidate)
				continue
			}
			// Candidate is prefix of what user typed
			if len(candidateLower) >= 2 && strings.HasPrefix(nameLower, candidateLower) {
				matches = append(matches, candidate)
				continue
			}
		}

		// Typo detection for reasonably-sized names (3+ chars)
		// Allow 1 edit for short names, 2 edits for longer names
		if len(name) >= 3 && len(candidate) >= 3 {
			maxDist := 1
			if len(name) >= 5 {
				maxDist = 2
			}
			if abs(len(name)-len(candidate)) <= maxDist && levenshteinClose(nameLower, candidateLower, maxDist) {
				matches = append(matches, candidate)
			}
		}
	}

	// Sort for consistent output
	sort.Strings(matches)

	// Limit suggestions
	if len(matches) > 3 {
		matches = matches[:3]
	}

	return matches
}

// levenshteinClose returns true if edit distance is <= maxDist (quick check, not full algorithm)
func levenshteinClose(a, b string, maxDist int) bool {
	if abs(len(a)-len(b)) > maxDist {
		return false
	}

	// Simple character diff count for similar-length strings
	diffs := 0
	minLen := min(len(a), len(b))

	for i := range minLen {
		if a[i] != b[i] {
			diffs++
			if diffs > maxDist {
				return false
			}
		}
	}

	diffs += abs(len(a) - len(b))
	return diffs <= maxDist
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// FrontmatterInfo provides the frontmatter context needed by the semantic
// checker to validate @directive references. This interface breaks the
// import cycle between spec/semantic and spec/document.
type FrontmatterInfo interface {
	HasScale() bool
	HasGlobals() bool
	HasGlobal(name string) bool
	GlobalKeys() []string
}

// Checker performs semantic validation on AST nodes.
type Checker struct {
	env         *Environment
	diagnostics []Diagnostic
	frontmatter FrontmatterInfo // optional; enables @directive validation
	lineOffset  int             // added to block-relative line numbers to produce document-absolute lines
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

// SetLineOffset sets the number of lines preceding this calc block in the
// document (e.g. frontmatter lines). Line numbers embedded in diagnostic
// messages are adjusted by this offset so they are document-absolute rather
// than block-relative.
func (c *Checker) SetLineOffset(offset int) {
	c.lineOffset = offset
}

// SetFrontmatter provides frontmatter context for @directive validation.
// Without frontmatter, @directive references produce "missing frontmatter" errors.
func (c *Checker) SetFrontmatter(fm FrontmatterInfo) {
	c.frontmatter = fm
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
	case *ast.FractionLiteral:
		c.checkFractionLiteral(n)
	// Literals don't need semantic checking (they're syntactically valid)
	case *ast.NumberLiteral, *ast.CurrencyLiteral, *ast.BooleanLiteral:
		// No semantic checks needed for simple literals
	case *ast.DateLiteral:
		c.checkDateLiteral(n) // USER REQUIREMENT: Validate dates
	case *ast.RelativeDateLiteral:
		// Validated by lexer/parser
	case *ast.EndOfExpr:
		c.checkEndOfStartOfInner(n.Period, "end of", n.Range)
	case *ast.StartOfExpr:
		c.checkEndOfStartOfInner(n.Period, "start of", n.Range)
	case *ast.LengthOfExpr:
		op := "length of"
		if n.AsNumber {
			op = "days in"
		}
		c.checkLengthOfInner(n.Period, op, n.Range)
	case *ast.BetweenExpr:
		c.checkBetweenEndpoint(n.Start, "start", n.Range)
		c.checkBetweenEndpoint(n.End, "end", n.Range)
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
	case *ast.AsPercentOf:
		c.checkAsPercentOf(n)
	case *ast.DirectiveRef:
		c.checkDirectiveRef(n)
	}
}

// checkAssignment validates variable assignments.
func (c *Checker) checkAssignment(a *ast.Assignment) {
	// CalcMark semantic rule: Variables are immutable - can only be defined once
	if existing, ok := c.env.GetInfo(a.Name); ok {
		// Variable already defined - this is a redefinition error
		msg := "cannot reassign '" + a.Name + "' — variables are immutable"
		if existing.Range != nil {
			msg += " (first defined at line " +
				strconv.Itoa(existing.Range.Start.Line+c.lineOffset) + ")"
		}

		c.diagnostics = append(c.diagnostics, Diagnostic{
			Severity: Error,
			Code:     DiagVariableRedefinition,
			Message:  msg,
			Detailed: "CalcMark variables are immutable and can only be assigned once per document. This ensures calculations are reproducible and verifiable. Use a different variable name, or edit the original definition.",
			Range:    a.Range,
		})
		return // Don't record the redefinition
	}

	// Check the value expression
	c.checkExpression(a.Value)

	// Record the variable in the environment with its position
	// We don't know the actual type yet (that's the interpreter's job),
	// but we mark it as defined
	c.env.SetWithRange(a.Name, nil, a.Range)
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
			// Build helpful error message
			msg := `undefined variable "` + id.Name + `"`

			// Find similar variable names to suggest
			allVars := c.env.GetAllVariables()
			varNames := make(map[string]struct{}, len(allVars))
			for name := range allVars {
				varNames[name] = struct{}{}
			}

			similar := findSimilarNames(id.Name, varNames)
			detailed := ""

			if len(similar) == 1 {
				msg += ` — did you mean "` + similar[0] + `"?`
			} else if len(similar) > 1 {
				msg += ` — did you mean one of: ` + strings.Join(similar, ", ") + `?`
			}

			// Add context about what IS defined
			if len(allVars) == 0 {
				detailed = "No variables have been defined yet. Define a variable with: name = value"
			} else if len(allVars) <= 5 {
				// List all defined variables if there are few
				names := make([]string, 0, len(allVars))
				for name := range allVars {
					names = append(names, name)
				}
				sort.Strings(names)
				detailed = "Defined variables: " + strings.Join(names, ", ")
			} else {
				detailed = "Check variable spelling. Variables must be defined before use."
			}

			c.addDiagnostic(Diagnostic{
				Severity: Error,
				Code:     DiagUndefinedVariable,
				Message:  msg,
				Detailed: detailed,
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
	case "compound":
		// compound(principal, rate, periods, modifier?)
		// First 3 arguments are expressions, 4th (optional) is a period/compounded identifier
		for i := 0; i < len(f.Arguments) && i < 3; i++ {
			c.checkExpression(f.Arguments[i])
		}
		// Skip 4th argument (period identifier or compounded:freq modifier)
		return
	case "depreciate":
		// depreciate(value, rate, periods, salvage?)
		// All arguments are expressions
		for _, arg := range f.Arguments {
			c.checkExpression(arg)
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
	// Recurse into expression-based quantities (e.g., "@scale meters")
	if q.Expr != nil {
		c.checkNode(q.Expr)
	}
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

// checkAsPercentOf validates "X as % of Y" expressions.
func (c *Checker) checkAsPercentOf(a *ast.AsPercentOf) {
	if a.Numerator != nil {
		c.checkExpression(a.Numerator)
	}
	if a.Denominator != nil {
		c.checkExpression(a.Denominator)
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

// checkDirectiveRef validates a @directive reference against frontmatter.
func (c *Checker) checkDirectiveRef(d *ast.DirectiveRef) {
	// Only @scale and @globals are valid directives
	switch d.Directive {
	case "scale":
		if c.frontmatter == nil || !c.frontmatter.HasScale() {
			c.diagnostics = append(c.diagnostics, Diagnostic{
				Severity: Error,
				Code:     DiagMissingFrontmatter,
				Message:  "@scale requires 'scale:' in frontmatter",
				Detailed: "Add a scale directive to the document frontmatter, e.g.:\n---\nscale: 3\n---",
				Range:    d.Range,
			})
		}

	case "globals":
		if c.frontmatter == nil || !c.frontmatter.HasGlobals() {
			c.diagnostics = append(c.diagnostics, Diagnostic{
				Severity: Error,
				Code:     DiagMissingFrontmatter,
				Message:  "@globals requires 'globals:' in frontmatter",
				Detailed: "Add globals to the document frontmatter, e.g.:\n---\nglobals:\n  tax_rate: 0.085\n---",
				Range:    d.Range,
			})
			return
		}
		if d.Field != "" && !c.frontmatter.HasGlobal(d.Field) {
			// List defined globals for discoverability
			defined := c.frontmatter.GlobalKeys()
			c.diagnostics = append(c.diagnostics, Diagnostic{
				Severity: Error,
				Code:     DiagUndefinedGlobal,
				Message:  "undefined global '" + d.Field + "'; defined globals: " + strings.Join(defined, ", "),
				Detailed: "Check the spelling of the global name, or add it to the frontmatter globals section.",
				Range:    d.Range,
			})
		}

	default:
		c.diagnostics = append(c.diagnostics, Diagnostic{
			Severity: Error,
			Code:     DiagInvalidDirective,
			Message:  "'" + "@" + d.Directive + "' is not a supported directive; use @scale or @globals.name",
			Detailed: "CalcMark supports two directive references:\n  @scale — the numeric scale factor from frontmatter\n  @globals.name — a named global variable from frontmatter",
			Range:    d.Range,
		})
	}
}

// checkFractionLiteral validates fraction literals.
// Checks: denominator is not zero, unit category is allowed for fraction quantities.
func (c *Checker) checkFractionLiteral(f *ast.FractionLiteral) {
	// Check for zero denominator
	if f.Denominator == 0 {
		c.addDiagnostic(Diagnostic{
			Severity: Warning,
			Code:     DiagDivisionByZero,
			Message:  "Fraction with zero denominator will cause a runtime error",
			Range:    f.Range,
		})
	}

	// If fraction has a unit, check category is allowed
	// Fraction quantities are only allowed for: Length, Mass, Volume, Custom (arbitrary units)
	// NOT allowed for: Duration, Currency, Boolean, Rate
	if f.Unit != "" {
		if quantity, found := units.GetUnitQuantity(f.Unit); found {
			switch quantity {
			case "Length", "Mass", "Volume":
				// Allowed
			default:
				c.addDiagnostic(Diagnostic{
					Severity: Error,
					Code:     DiagTypeMismatch,
					Message:  "Fraction quantities are not supported for " + quantity + " units; use decimal notation instead",
					Range:    f.Range,
				})
			}
		}
		// If unit is not found in standard units, it's a custom/arbitrary unit — allowed
	}
}

// checkDataSizeBaseMixingForCapacity checks if demand and capacity units mix binary and decimal bases.
func (c *Checker) checkDataSizeBaseMixingForCapacity(demand, capacity ast.Node) {
	demandUnit := getNodeUnit(demand)
	capacityUnit := getNodeUnit(capacity)

	c.checkDataSizeBaseMixingForUnits(demandUnit, capacityUnit)
}
