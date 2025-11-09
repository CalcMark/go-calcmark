// Package parser implements the CalcMark parser
package parser

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/ast"
	"github.com/CalcMark/go-calcmark/lexer"
)

// ParseError represents a parse error
type ParseError struct {
	Message string
	Line    int
	Column  int
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%s at %d:%d", e.Message, e.Line, e.Column)
}

// tokenToRange converts a token to a Range
func tokenToRange(token lexer.Token) *ast.Range {
	start := ast.Position{Line: token.Line, Column: token.Column}
	end := ast.Position{Line: token.Line, Column: token.Column + len(token.Value)}
	return &ast.Range{Start: start, End: end}
}

// Parser parses CalcMark tokens into an AST
type Parser struct {
	tokens []lexer.Token
	pos    int
}

// NewParser creates a new parser from tokens
func NewParser(tokens []lexer.Token) *Parser {
	return &Parser{
		tokens: tokens,
		pos:    0,
	}
}

// currentToken returns the current token
func (p *Parser) currentToken() lexer.Token {
	if p.pos >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1] // Return EOF
	}
	return p.tokens[p.pos]
}

// peek looks ahead at a token
func (p *Parser) peek(offset int) lexer.Token {
	pos := p.pos + offset
	if pos >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1] // Return EOF
	}
	return p.tokens[pos]
}

// advance moves to next token and returns current
func (p *Parser) advance() lexer.Token {
	token := p.currentToken()
	if p.pos < len(p.tokens)-1 {
		p.pos++
	}
	return token
}

// expect expects a specific token type and advances
func (p *Parser) expect(tokenType lexer.TokenType) (lexer.Token, error) {
	token := p.currentToken()
	if token.Type != tokenType {
		return token, &ParseError{
			Message: fmt.Sprintf("Expected %s, got %s", tokenType, token.Type),
			Line:    token.Line,
			Column:  token.Column,
		}
	}
	return p.advance(), nil
}

// ParseStatement parses a single statement (assignment or expression)
func (p *Parser) ParseStatement() (ast.Node, error) {
	// Skip newlines
	for p.currentToken().Type == lexer.NEWLINE {
		p.advance()
	}

	// Check for EOF
	if p.currentToken().Type == lexer.EOF {
		return nil, nil
	}

	// Check if this is an assignment (identifier = ... or boolean = ...)
	// Allow boolean keywords as variable names when followed by =
	current := p.currentToken()
	next := p.peek(1)

	var result ast.Node
	var err error

	if (current.Type == lexer.IDENTIFIER || current.Type == lexer.BOOLEAN) &&
		next.Type == lexer.ASSIGN {
		result, err = p.parseAssignment()
		if err != nil {
			return nil, err
		}
	} else {
		// Otherwise, it's a standalone expression
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		result = &ast.Expression{Expr: expr}
	}

	// After parsing a statement, verify no trailing tokens before newline/EOF
	currentTok := p.currentToken()
	if currentTok.Type != lexer.NEWLINE && currentTok.Type != lexer.EOF {
		return nil, &ParseError{
			Message: fmt.Sprintf("Unexpected token after statement: %s", currentTok.Type),
			Line:    currentTok.Line,
			Column:  currentTok.Column,
		}
	}

	return result, nil
}

// parseAssignment parses an assignment statement
func (p *Parser) parseAssignment() (*ast.Assignment, error) {
	nameToken := p.currentToken()
	if nameToken.Type != lexer.IDENTIFIER && nameToken.Type != lexer.BOOLEAN {
		return nil, &ParseError{
			Message: fmt.Sprintf("Expected identifier, got %s", nameToken.Type),
			Line:    nameToken.Line,
			Column:  nameToken.Column,
		}
	}
	p.advance()

	_, err := p.expect(lexer.ASSIGN)
	if err != nil {
		return nil, err
	}

	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	return &ast.Assignment{
		Name:  nameToken.Value,
		Value: value,
		Range: tokenToRange(nameToken),
	}, nil
}

// parseExpression parses an expression (handles operator precedence)
func (p *Parser) parseExpression() (ast.Node, error) {
	return p.parseComparison()
}

// parseComparison parses comparison operations (lowest precedence)
func (p *Parser) parseComparison() (ast.Node, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}

	comparisonOps := map[lexer.TokenType]bool{
		lexer.GREATER_THAN:   true,
		lexer.LESS_THAN:      true,
		lexer.GREATER_EQUAL:  true,
		lexer.LESS_EQUAL:     true,
		lexer.EQUAL:          true,
		lexer.NOT_EQUAL:      true,
	}

	for comparisonOps[p.currentToken().Type] {
		opToken := p.advance()
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		left = &ast.ComparisonOp{
			Operator: opToken.Value,
			Left:     left,
			Right:    right,
			Range:    tokenToRange(opToken),
		}
	}

	return left, nil
}

// parseAdditive parses addition and subtraction (lower precedence)
func (p *Parser) parseAdditive() (ast.Node, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return nil, err
	}

	for p.currentToken().Type == lexer.PLUS || p.currentToken().Type == lexer.MINUS {
		opToken := p.advance()
		right, err := p.parseMultiplicative()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryOp{
			Operator: opToken.Value,
			Left:     left,
			Right:    right,
			Range:    tokenToRange(opToken),
		}
	}

	return left, nil
}

// parseMultiplicative parses multiplication, division, and modulus (higher precedence)
func (p *Parser) parseMultiplicative() (ast.Node, error) {
	left, err := p.parseExponent()
	if err != nil {
		return nil, err
	}

	for {
		tt := p.currentToken().Type
		if tt != lexer.MULTIPLY && tt != lexer.DIVIDE && tt != lexer.MODULUS {
			break
		}
		opToken := p.advance()
		right, err := p.parseExponent()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryOp{
			Operator: opToken.Value,
			Left:     left,
			Right:    right,
			Range:    tokenToRange(opToken),
		}
	}

	return left, nil
}

// parseExponent parses exponentiation (highest precedence, right-associative)
func (p *Parser) parseExponent() (ast.Node, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	// Right-associative: 2^3^2 = 2^(3^2)
	if p.currentToken().Type == lexer.EXPONENT {
		opToken := p.advance()
		right, err := p.parseExponent() // Recursive for right-associativity
		if err != nil {
			return nil, err
		}
		return &ast.BinaryOp{
			Operator: opToken.Value,
			Left:     left,
			Right:    right,
			Range:    tokenToRange(opToken),
		}, nil
	}

	return left, nil
}

// parsePrimary parses primary expressions (literals, identifiers, unary ops)
func (p *Parser) parsePrimary() (ast.Node, error) {
	token := p.currentToken()

	// Handle unary operators (- and +)
	if token.Type == lexer.MINUS || token.Type == lexer.PLUS {
		opToken := p.advance()
		operand, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryOp{
			Operator: opToken.Value,
			Operand:  operand,
			Range:    tokenToRange(opToken),
		}, nil
	}

	if token.Type == lexer.NUMBER {
		p.advance()
		return &ast.NumberLiteral{
			Value: token.Value,
			Range: tokenToRange(token),
		}, nil
	}

	if token.Type == lexer.CURRENCY {
		p.advance()
		// Token value is "symbol:value"
		parts := strings.SplitN(token.Value, ":", 2)
		symbol := parts[0]
		value := parts[1]
		return &ast.CurrencyLiteral{
			Value:  value,
			Symbol: symbol,
			Range:  tokenToRange(token),
		}, nil
	}

	if token.Type == lexer.BOOLEAN || token.Type == lexer.IDENTIFIER {
		// Treat both BOOLEAN and IDENTIFIER tokens as identifiers
		// The evaluator will determine if a boolean keyword should be
		// treated as a literal or a variable reference based on context
		p.advance()
		return &ast.Identifier{
			Name:  token.Value,
			Range: tokenToRange(token),
		}, nil
	}

	// Handle parenthesized expressions
	if token.Type == lexer.LPAREN {
		p.advance() // consume '('
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		// Expect closing ')'
		if p.currentToken().Type != lexer.RPAREN {
			return nil, &ParseError{
				Message: fmt.Sprintf("Expected ')', got %s", p.currentToken().Type),
				Line:    p.currentToken().Line,
				Column:  p.currentToken().Column,
			}
		}
		p.advance() // consume ')'
		return expr, nil
	}

	return nil, &ParseError{
		Message: fmt.Sprintf("Unexpected token %s", token.Type),
		Line:    token.Line,
		Column:  token.Column,
	}
}

// Parse parses all statements
func (p *Parser) Parse() ([]ast.Node, error) {
	var statements []ast.Node

	for p.currentToken().Type != lexer.EOF {
		stmt, err := p.ParseStatement()
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			statements = append(statements, stmt)
		}

		// Skip trailing newlines
		for p.currentToken().Type == lexer.NEWLINE {
			p.advance()
		}
	}

	return statements, nil
}

// Parse is a convenience function to parse text
func Parse(text string) ([]ast.Node, error) {
	tokens, err := lexer.Tokenize(text)
	if err != nil {
		return nil, err
	}
	parser := NewParser(tokens)
	return parser.Parse()
}
