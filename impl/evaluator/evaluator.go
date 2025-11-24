// Package evaluator implements the CalcMark evaluator
package evaluator

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/constants"
	"github.com/CalcMark/go-calcmark/impl/types"
	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/shopspring/decimal"
)

// EvaluationError represents an evaluation error
type EvaluationError struct {
	Message string
	Range   *ast.Range
}

func (e *EvaluationError) Error() string {
	if e.Range != nil {
		return fmt.Sprintf("%s at %s", e.Message, e.Range.Start)
	}
	return e.Message
}

// Context stores variables during evaluation
type Context struct {
	Variables map[string]types.Type
}

// NewContext creates a new context
func NewContext() *Context {
	return &Context{
		Variables: make(map[string]types.Type),
	}
}

// isConstant checks if a name is a reserved constant
func isConstant(name string) bool {
	lower := strings.ToLower(name)
	return lower == "pi" || lower == "e"
}

// Set stores a variable in the context
// Returns error if trying to set a constant
func (c *Context) Set(name string, value types.Type) error {
	if isConstant(name) {
		return &EvaluationError{
			Message: fmt.Sprintf("Cannot assign to constant '%s'", name),
		}
	}
	c.Variables[name] = value
	return nil
}

// Has checks if a variable exists
func (c *Context) Has(name string) bool {
	_, exists := c.Variables[name]
	return exists
}

// Get retrieves a variable from context
// Handles boolean keyword resolution and mathematical constants
func (c *Context) Get(name string) (types.Type, error) {
	// Look up variable first
	if value, exists := c.Variables[name]; exists {
		return value, nil
	}

	lower := strings.ToLower(name)

	// Check if it's a mathematical constant
	if lower == "pi" {
		return types.NewNumber("3.141592653589793")
	}
	if lower == "e" {
		return types.NewNumber("2.718281828459045")
	}

	// Check if it's a boolean keyword
	booleanKeywords := map[string]bool{
		"true": true, "yes": true, "t": true, "y": true,
		"false": false, "no": false, "f": false, "n": false,
	}

	if boolVal, isBoolKeyword := booleanKeywords[lower]; isBoolKeyword {
		b, err := types.NewBoolean(boolVal)
		if err != nil {
			return nil, err
		}
		return b, nil
	}

	return nil, &EvaluationError{
		Message: fmt.Sprintf("Undefined variable '%s'", name),
	}
}

// Evaluator evaluates CalcMark AST nodes
type Evaluator struct {
	Context *Context
}

// NewEvaluator creates a new evaluator
func NewEvaluator(context *Context) *Evaluator {
	if context == nil {
		context = NewContext()
	}
	return &Evaluator{Context: context}
}

// Eval evaluates a list of nodes and returns their results
func (e *Evaluator) Eval(nodes []ast.Node) ([]types.Type, error) {
	var results []types.Type

	for _, node := range nodes {
		result, err := e.EvalNode(node)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

// EvalNode evaluates a single AST node
func (e *Evaluator) EvalNode(node ast.Node) (types.Type, error) {
	switch n := node.(type) {
	case *ast.NumberLiteral:
		return types.NewNumberWithFormat(n.Value, n.SourceText)

	case *ast.QuantityLiteral:
		// Handle QuantityLiteral (includes currencies like $100 → USD)
		// Use NewCurrencyWithFormat which handles value + unit + source format
		return types.NewCurrencyWithFormat(n.Value, n.Unit, n.SourceText)

	case *ast.CurrencyLiteral:
		return types.NewCurrencyWithFormat(n.Value, n.Symbol, n.SourceText)

	case *ast.BooleanLiteral:
		return types.NewBoolean(n.Value)

	case *ast.Identifier:
		return e.Context.Get(n.Name)

	case *ast.UnaryOp:
		return e.evalUnaryOp(n)

	case *ast.BinaryOp:
		return e.evalBinaryOp(n)

	case *ast.ComparisonOp:
		return e.evalComparisonOp(n)

	case *ast.Assignment:
		value, err := e.EvalNode(n.Value)
		if err != nil {
			return nil, err
		}
		err = e.Context.Set(n.Name, value)
		if err != nil {
			return nil, err
		}
		return value, nil

	case *ast.Expression:
		return e.EvalNode(n.Expr)

	case *ast.FunctionCall:
		return e.evalFunctionCall(n)

	default:
		return nil, &EvaluationError{
			Message: fmt.Sprintf("Unknown node type: %T", node),
			Range:   node.GetRange(),
		}
	}
}

// getDecimalValue extracts the decimal value from a type
func getDecimalValue(t types.Type) (decimal.Decimal, error) {
	switch v := t.(type) {
	case *types.Number:
		return v.Value, nil
	case *types.Currency:
		return v.Value, nil
	default:
		return decimal.Zero, fmt.Errorf("cannot get decimal value from %s", t.TypeName())
	}
}

// evalUnaryOp evaluates a unary operation
func (e *Evaluator) evalUnaryOp(node *ast.UnaryOp) (types.Type, error) {
	operand, err := e.EvalNode(node.Operand)
	if err != nil {
		return nil, err
	}

	operandVal, err := getDecimalValue(operand)
	if err != nil {
		return nil, &EvaluationError{
			Message: err.Error(),
			Range:   node.Range,
		}
	}

	var result decimal.Decimal
	switch node.Operator {
	case "-":
		result = operandVal.Neg()
	case "+":
		result = operandVal
	default:
		return nil, &EvaluationError{
			Message: fmt.Sprintf("Unknown unary operator: %s", node.Operator),
			Range:   node.Range,
		}
	}

	// Preserve type (Number vs Currency)
	switch operand.(type) {
	case *types.Currency:
		curr := operand.(*types.Currency)
		return &types.Currency{
			Value:  result,
			Symbol: curr.Symbol,
		}, nil
	default:
		return &types.Number{Value: result}, nil
	}
}

// evalBinaryOp evaluates a binary operation
func (e *Evaluator) evalBinaryOp(node *ast.BinaryOp) (types.Type, error) {
	left, err := e.EvalNode(node.Left)
	if err != nil {
		return nil, err
	}

	right, err := e.EvalNode(node.Right)
	if err != nil {
		return nil, err
	}

	// Extract decimal values
	leftVal, err := getDecimalValue(left)
	if err != nil {
		return nil, &EvaluationError{Message: err.Error(), Range: node.Range}
	}

	rightVal, err := getDecimalValue(right)
	if err != nil {
		return nil, &EvaluationError{Message: err.Error(), Range: node.Range}
	}

	// Compute the result
	result, err := computeBinaryOp(node.Operator, leftVal, rightVal, isPercentageLiteral(right), node.Range)
	if err != nil {
		return nil, err
	}

	// Determine and create the appropriate result type
	resultTypeName, symbol := determineBinaryResultType(left, right)
	return createResultType(resultTypeName, symbol, result), nil
}

// evalComparisonOp evaluates a comparison operation
func (e *Evaluator) evalComparisonOp(node *ast.ComparisonOp) (types.Type, error) {
	left, err := e.EvalNode(node.Left)
	if err != nil {
		return nil, err
	}

	right, err := e.EvalNode(node.Right)
	if err != nil {
		return nil, err
	}

	leftVal, err := getDecimalValue(left)
	if err != nil {
		return nil, &EvaluationError{
			Message: err.Error(),
			Range:   node.Range,
		}
	}

	rightVal, err := getDecimalValue(right)
	if err != nil {
		return nil, &EvaluationError{
			Message: err.Error(),
			Range:   node.Range,
		}
	}

	var result bool
	switch node.Operator {
	case ">":
		result = leftVal.GreaterThan(rightVal)
	case "<":
		result = leftVal.LessThan(rightVal)
	case ">=":
		result = leftVal.GreaterThanOrEqual(rightVal)
	case "<=":
		result = leftVal.LessThanOrEqual(rightVal)
	case "==":
		result = leftVal.Equal(rightVal)
	case "!=":
		result = !leftVal.Equal(rightVal)
	default:
		return nil, &EvaluationError{
			Message: fmt.Sprintf("Unknown comparison operator: %s", node.Operator),
			Range:   node.Range,
		}
	}

	return types.NewBoolean(result)
}

// evalFunctionCall evaluates a function call
func (e *Evaluator) evalFunctionCall(node *ast.FunctionCall) (types.Type, error) {
	impl, err := lookupFunction(node.Name)
	if err != nil {
		return nil, &EvaluationError{Message: err.Error(), Range: node.Range}
	}
	return impl(e, node.Arguments, node.Range)
}

// Evaluate is a convenience function to parse and evaluate text
// It filters out markdown lines before parsing
func Evaluate(text string, context *Context) ([]types.Type, error) {
	if context == nil {
		context = NewContext()
	}

	// Filter out markdown bullet lines (lines starting with "- " or "* ")
	lines := strings.Split(text, constants.Newline)
	var calcmarkLines []string
	for _, line := range lines {
		stripped := strings.TrimLeft(line, constants.Whitespace)
		// Skip markdown bullets: "- " or "* " (first char is dash/asterisk followed by space)
		if len(stripped) >= 2 && (stripped[0] == '-' || stripped[0] == '*') && stripped[1:2] == constants.Space {
			continue
		}
		calcmarkLines = append(calcmarkLines, line)
	}

	filteredText := strings.Join(calcmarkLines, constants.Newline)
	nodes, err := parser.Parse(filteredText)
	if err != nil {
		return nil, err
	}

	evaluator := NewEvaluator(context)
	return evaluator.Eval(nodes)
}

// EvaluationResult represents an evaluation result with its original line number.
// Used by EvaluateDocument to track which line each result came from.
type EvaluationResult struct {
	Value        types.Type // The computed value
	OriginalLine int        // 1-indexed line number in source document
}

// EvaluateDocument evaluates a full CalcMark document (markdown + calculations).
//
// This function properly handles mixed documents by classifying lines first,
// then evaluating only CALCULATION lines while preserving line numbers.
//
// Architecture:
//  1. Split document into lines
//  2. Classify each line (CALCULATION, MARKDOWN, BLANK)
//  3. Evaluate only CALCULATION lines
//  4. Return results with original line numbers preserved
//
// The classifier uses the provided context to determine if identifiers are defined,
// making classification context-aware (e.g., "x" is CALCULATION if x is defined).
//
// Example:
//
//	Input:
//	  # Title
//	  x = 5
//	  Some text
//	  y = x + 10
//
//	Output:
//	  [
//	    {Value: types.Number(5), OriginalLine: 2},
//	    {Value: types.Number(15), OriginalLine: 4}
//	  ]
func EvaluateDocument(text string, context *Context) ([]EvaluationResult, error) {
	// We avoid importing classifier to prevent circular dependencies
	// (classifier depends on evaluator for context-aware classification)
	// Instead, we classify inline by attempting to parse each line

	// Split into lines
	lines := strings.Split(text, constants.Newline)

	var results []EvaluationResult

	// Evaluate each line individually, tracking line numbers
	for lineNum, line := range lines {
		// Try to evaluate - if it works, it's a calculation
		// This is simpler than importing classifier (avoids circular dependency)
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// Skip obvious markdown (headers, lists)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if len(trimmed) >= 2 && (trimmed[0] == '-' || trimmed[0] == '*') && trimmed[1:2] == constants.Space {
			continue
		}

		// Try to parse and evaluate
		nodes, err := parser.Parse(line)
		if err != nil {
			// Not a valid calculation, treat as markdown
			continue
		}

		evaluator := NewEvaluator(context)
		lineResults, err := evaluator.Eval(nodes)
		if err != nil {
			// Evaluation failed, skip this line
			continue
		}

		// Add results with line number
		for _, value := range lineResults {
			results = append(results, EvaluationResult{
				Value:        value,
				OriginalLine: lineNum + 1, // 1-indexed
			})
		}
	}

	return results, nil
}
