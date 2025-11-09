// Package evaluator implements the CalcMark evaluator
package evaluator

import (
	"fmt"
	"math"
	"strings"

	"github.com/CalcMark/go-calcmark/ast"
	"github.com/CalcMark/go-calcmark/constants"
	"github.com/CalcMark/go-calcmark/parser"
	"github.com/CalcMark/go-calcmark/types"
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

// Set stores a variable in the context
func (c *Context) Set(name string, value types.Type) {
	c.Variables[name] = value
}

// Has checks if a variable exists
func (c *Context) Has(name string) bool {
	_, exists := c.Variables[name]
	return exists
}

// Get retrieves a variable from context
// Handles boolean keyword resolution
func (c *Context) Get(name string) (types.Type, error) {
	// Look up variable first
	if value, exists := c.Variables[name]; exists {
		return value, nil
	}

	// Check if it's a boolean keyword
	lower := strings.ToLower(name)
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
		return types.NewNumber(n.Value)

	case *ast.CurrencyLiteral:
		return types.NewCurrency(n.Value, n.Symbol)

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
		e.Context.Set(n.Name, value)
		return value, nil

	case *ast.Expression:
		return e.EvalNode(n.Expr)

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

	// Get decimal values
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

	// Perform operation
	var result decimal.Decimal
	switch node.Operator {
	case "+":
		result = leftVal.Add(rightVal)
	case "-":
		result = leftVal.Sub(rightVal)
	case "*", "×":
		result = leftVal.Mul(rightVal)
	case "/", "÷":
		if rightVal.IsZero() {
			return nil, &EvaluationError{
				Message: "Division by zero",
				Range:   node.Range,
			}
		}
		result = leftVal.Div(rightVal)
	case "%":
		if rightVal.IsZero() {
			return nil, &EvaluationError{
				Message: "Division by zero",
				Range:   node.Range,
			}
		}
		result = leftVal.Mod(rightVal)
	case "**", "^":
		// For exponentiation, we need to convert to float64
		base, _ := leftVal.Float64()
		exp, _ := rightVal.Float64()
		// Simple integer exponentiation
		if rightVal.IsInteger() {
			expInt := rightVal.IntPart()
			if expInt >= 0 {
				result = leftVal.Pow(decimal.NewFromInt(expInt))
			} else {
				// For negative exponents, use 1 / x^|exp|
				posExp := decimal.NewFromInt(-expInt)
				result = decimal.NewFromInt(1).Div(leftVal.Pow(posExp))
			}
		} else {
			// For non-integer exponents, use math.Pow
			floatResult := math.Pow(base, exp)
			result = decimal.NewFromFloat(floatResult)
		}
	default:
		return nil, &EvaluationError{
			Message: fmt.Sprintf("Unknown operator: %s", node.Operator),
			Range:   node.Range,
		}
	}

	// Type coercion: if either operand is Currency, result is Currency
	leftCurrency, leftIsCurrency := left.(*types.Currency)
	rightCurrency, rightIsCurrency := right.(*types.Currency)

	if leftIsCurrency {
		return &types.Currency{Value: result, Symbol: leftCurrency.Symbol}, nil
	}
	if rightIsCurrency {
		return &types.Currency{Value: result, Symbol: rightCurrency.Symbol}, nil
	}

	return &types.Number{Value: result}, nil
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
