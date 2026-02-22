package parser

import (
	"strings"

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

// parseNLReadFunction parses: read <quantity> from <identifier>
// Precondition: "read" already consumed, next token is QUANTITY
func (p *RecursiveDescentParser) parseNLReadFunction() (ast.Node, error) {
	// Parse size using parseExponent() — NOT parseExpression() — to avoid
	// consuming FROM as part of a date expression or conversion context.
	size, err := p.parseExponent()
	if err != nil {
		return nil, err
	}
	if !p.match(lexer.FROM) {
		return nil, p.error("expected 'from' after size in 'read <size> from <storage>'")
	}
	if !p.match(lexer.IDENTIFIER) {
		return nil, p.error("expected storage type after 'from' (e.g., ssd, nvme, hdd)")
	}
	storageType := p.previous()
	return &ast.FunctionCall{
		Name:      "read",
		Arguments: []ast.Node{size, &ast.Identifier{Name: string(storageType.Value)}},
	}, nil
}

// parseNLCompressFunction parses: compress <quantity> using <identifier>
// Precondition: "compress" already consumed, next token is QUANTITY
func (p *RecursiveDescentParser) parseNLCompressFunction() (ast.Node, error) {
	size, err := p.parseExponent()
	if err != nil {
		return nil, err
	}
	// "using" is a contextual keyword — it's an IDENTIFIER, not reserved
	if !p.match(lexer.IDENTIFIER) || strings.ToLower(string(p.previous().Value)) != "using" {
		return nil, p.error("expected 'using' after size in 'compress <size> using <algorithm>'")
	}
	if !p.match(lexer.IDENTIFIER) {
		return nil, p.error("expected algorithm after 'using' (e.g., gzip, lz4, zstd)")
	}
	algo := p.previous()
	return &ast.FunctionCall{
		Name:      "compress",
		Arguments: []ast.Node{size, &ast.Identifier{Name: string(algo.Value)}},
	}, nil
}

// parseNLTransferFunction parses: transfer <quantity> across <scope> <network>
// Precondition: "transfer" already consumed, next token is QUANTITY
func (p *RecursiveDescentParser) parseNLTransferFunction() (ast.Node, error) {
	size, err := p.parseExponent()
	if err != nil {
		return nil, err
	}
	// "across" is a contextual keyword — it's an IDENTIFIER, not reserved
	if !p.match(lexer.IDENTIFIER) || strings.ToLower(string(p.previous().Value)) != "across" {
		return nil, p.error("expected 'across' after size in 'transfer <size> across <scope> <network>'")
	}
	if !p.match(lexer.IDENTIFIER) {
		return nil, p.error("expected network scope after 'across' (e.g., local, regional, continental, global)")
	}
	scope := p.previous()
	if !p.match(lexer.IDENTIFIER) {
		return nil, p.error("expected network type (e.g., gigabit, wifi, four_g)")
	}
	network := p.previous()
	return &ast.FunctionCall{
		Name: "transfer_time",
		Arguments: []ast.Node{
			size,
			&ast.Identifier{Name: string(scope.Value)},
			&ast.Identifier{Name: string(network.Value)},
		},
	}, nil
}
