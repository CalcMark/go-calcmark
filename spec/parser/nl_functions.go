package parser

import (
	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/lexer"
)

// parseNaturalLanguageFunction parses natural language function syntax.
// NaturalLanguageFunction → "average of" ArgumentList | "square root of" Expression
func (p *RecursiveDescentParser) parseNaturalLanguageFunction() (ast.Node, error) {
	funcToken := p.previous() // FUNC_AVERAGE_OF or FUNC_SQUARE_ROOT_OF

	// Map to canonical function name
	var funcName string

	switch funcToken.Type {
	case lexer.FUNC_AVERAGE_OF:
		funcName = "avg"
	case lexer.FUNC_SQUARE_ROOT_OF:
		funcName = "sqrt"
	default:
		return nil, p.error("unexpected natural language function")
	}

	// For "square root of", just parse one expression
	if funcName == "sqrt" {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		return &ast.FunctionCall{
			Name:      funcName,
			Arguments: []ast.Node{expr},
		}, nil
	}

	// For "average of", parse comma-separated list (no parentheses!)
	var args []ast.Node

	// Parse first argument
	arg, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	args = append(args, arg)

	// Parse remaining arguments
	for p.match(lexer.COMMA) {
		arg, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}

	return &ast.FunctionCall{
		Name:      funcName,
		Arguments: args,
	}, nil
}
