package parser

import (
	"fmt"
	"slices"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
	"github.com/CalcMark/go-calcmark/v2/spec/lexer"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/CalcMark/go-calcmark/v2/spec/units"
)

// RecursiveDescentParser implements a recursive descent parser for CalcMark.
// It uses the hand-written lexer and builds an AST directly.
type RecursiveDescentParser struct {
	tokens  []lexer.Token
	current int
	source  string

	// Security: track nesting depth to prevent stack overflow
	depth    int
	maxDepth int
}

// NewRecursiveDescentParser creates a new parser for the given source text.
func NewRecursiveDescentParser(source string) *RecursiveDescentParser {
	lex := lexer.NewLexer(source)
	tokens, err := lex.Tokenize()
	if err != nil {
		// If tokenization fails, create parser with just EOF token
		tokens = []lexer.Token{{Type: lexer.EOF}}
	}

	return &RecursiveDescentParser{
		tokens:   tokens,
		current:  0,
		source:   source,
		depth:    0,
		maxDepth: MaxNestingDepth,
	}
}

// checkTokenLimit validates that token count doesn't exceed security limit
func (p *RecursiveDescentParser) checkTokenLimit() error {
	if len(p.tokens) > MaxTokenCount {
		return &SecurityError{
			Message: fmt.Sprintf("token count exceeds security limit: %d tokens (max %d)", len(p.tokens), MaxTokenCount),
			Limit:   "MaxTokenCount",
			Actual:  len(p.tokens),
		}
	}
	return nil
}

// Parse parses the source and returns an AST.
func (p *RecursiveDescentParser) Parse() ([]ast.Node, error) {
	// Security: check token count limit before parsing
	if err := p.checkTokenLimit(); err != nil {
		return nil, err
	}
	return p.parseProgram()
}

// ============================================================================
// Helper methods for token navigation
// ============================================================================

// peek returns the current token without advancing.
func (p *RecursiveDescentParser) peek() lexer.Token {
	if p.isAtEnd() {
		return p.tokens[len(p.tokens)-1] // EOF
	}
	return p.tokens[p.current]
}

// peekAhead returns the token N positions ahead without advancing.
func (p *RecursiveDescentParser) peekAhead(n int) lexer.Token {
	pos := p.current + n
	if pos >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1] // EOF
	}
	return p.tokens[pos]
}

// tokenRange returns a Range covering a single token's position.
func tokenRange(tok lexer.Token) *ast.Range {
	return &ast.Range{
		Start: ast.Position{Line: tok.Line, Column: tok.Column},
		End:   ast.Position{Line: tok.Line, Column: tok.Column + len(tok.Value)},
	}
}

// rangeOrFallback returns the node's range if it has a valid line number,
// otherwise a range from the fallback token. Use this when creating
// implicit/postfix FunctionCall nodes where the left operand may lack a range.
func rangeOrFallback(node ast.Node, fallback lexer.Token) *ast.Range {
	if r := node.GetRange(); r != nil && r.Start.Line > 0 {
		return r
	}
	return tokenRange(fallback)
}

// previous returns the most recently consumed token.
func (p *RecursiveDescentParser) previous() lexer.Token {
	if p.current == 0 {
		return p.tokens[0]
	}
	return p.tokens[p.current-1]
}

// advance consumes the current token and returns it.
func (p *RecursiveDescentParser) advance() lexer.Token {
	if !p.isAtEnd() {
		p.current++
	}
	return p.previous()
}

// isAtEnd returns true if we've consumed all tokens.
func (p *RecursiveDescentParser) isAtEnd() bool {
	return p.current >= len(p.tokens) || p.tokens[p.current].Type == lexer.EOF
}

// match checks if the current token matches any of the given types.
// If it matches, consumes the token and returns true.
func (p *RecursiveDescentParser) match(types ...lexer.TokenType) bool {
	if slices.ContainsFunc(types, p.check) {
		p.advance()
		return true
	}
	return false
}

// check returns true if the current token is of the given type.
// Does NOT consume the token.
func (p *RecursiveDescentParser) check(t lexer.TokenType) bool {
	if p.isAtEnd() {
		return false
	}
	return p.peek().Type == t
}

// consume checks that the current token is of the given type and consumes it.
// If not, returns an error.
func (p *RecursiveDescentParser) consume(t lexer.TokenType, message string) (lexer.Token, error) {
	if p.check(t) {
		return p.advance(), nil
	}

	current := p.peek()
	return lexer.Token{}, p.errorAt(current, message)
}

// ============================================================================
// Error handling
// ============================================================================

// error creates a parse error at the current position.
func (p *RecursiveDescentParser) error(message string) error {
	return p.errorAt(p.peek(), message)
}

// errorAt creates a parse error at the given token's position.
func (p *RecursiveDescentParser) errorAt(tok lexer.Token, message string) error {
	return &ParseError{
		Message: message,
		Line:    tok.Line,
		Column:  tok.Column,
	}
}

// tryConsumeUnit checks if the next token is a non-keyword IDENTIFIER
// and consumes it as a unit name. Handles multi-word units like
// "nautical mile". Returns the normalized unit name and true if a unit
// was consumed, or ("", false) if no unit follows.
func (p *RecursiveDescentParser) tryConsumeUnit() (string, bool) {
	if !p.check(lexer.IDENTIFIER) {
		return "", false
	}
	unitName := string(p.peek().Value)

	// NL keywords (growing, declining, etc.) are NOT units
	if isNaturalSyntaxKeyword(unitName) {
		return "", false
	}

	p.advance()

	if normalized, ok := units.NormalizeUnitName(unitName); ok {
		unitName = normalized
	}

	// Multi-word units: "nautical mile", "metric ton"
	if p.check(lexer.IDENTIFIER) {
		nextWord := string(p.peek().Value)
		if multiWordUnit := units.IsMultiWordUnit(unitName, nextWord); multiWordUnit != "" {
			p.advance()
			unitName = multiWordUnit
			if normalized, ok := units.NormalizeUnitName(multiWordUnit); ok {
				unitName = normalized
			}
		}
	}

	return unitName, true
}

// enterDepth increments nesting depth and checks security limit
func (p *RecursiveDescentParser) enterDepth() error {
	p.depth++
	if p.depth > p.maxDepth {
		return &SecurityError{
			Message: fmt.Sprintf("expression nesting depth exceeds security limit: %d levels (max %d)", p.depth, p.maxDepth),
			Limit:   "MaxNestingDepth",
			Actual:  p.depth,
		}
	}
	return nil
}

// exitDepth decrements nesting depth
func (p *RecursiveDescentParser) exitDepth() {
	p.depth--
}

// ============================================================================
// Grammar rules (to be implemented)
// ============================================================================

// parseProgram is the top-level grammar rule.
// Program → StatementList
func (p *RecursiveDescentParser) parseProgram() ([]ast.Node, error) {
	var statements []ast.Node

	// Skip leading newlines
	for p.match(lexer.NEWLINE) {
		// consume newlines
	}

	for !p.isAtEnd() {
		// Skip empty lines
		if p.match(lexer.NEWLINE) {
			continue
		}

		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}

		if stmt != nil {
			statements = append(statements, stmt)
		}

		// Expect newline or EOF after statement
		if !p.isAtEnd() && !p.match(lexer.NEWLINE) {
			return nil, p.error("expected newline after statement")
		}
	}

	return statements, nil
}

// parseStatement parses a single statement.
// Statement → Assignment | Expression
func (p *RecursiveDescentParser) parseStatement() (ast.Node, error) {
	// Try assignment (identifier '=' expression)
	if p.check(lexer.IDENTIFIER) && p.peekAhead(1).Type == lexer.ASSIGN {
		return p.parseAssignment()
	}

	// v2.0 migration diagnostic: `between = <expr>`. Pre-v2.0,
	// `between` could be used as a variable name. v2.0 reserves it
	// for the period operator. Surface a clear migration message
	// rather than the generic "unexpected token: BETWEEN".
	if p.check(lexer.BETWEEN) && p.peekAhead(1).Type == lexer.ASSIGN {
		return nil, p.error("'between' is a reserved keyword in v2.0; rename the variable (e.g., 'between_value')")
	}

	// Otherwise, it's an expression
	return p.parseExpression()
}

// parseAssignment parses a variable assignment.
// Assignment → IDENTIFIER '=' Expression
func (p *RecursiveDescentParser) parseAssignment() (ast.Node, error) {
	name := p.advance() // consume identifier

	if _, err := p.consume(lexer.ASSIGN, "expected '=' in assignment"); err != nil {
		return nil, err
	}

	value, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	return &ast.Assignment{
		Name:  string(name.Value),
		Value: value,
		Range: &ast.Range{
			Start: ast.Position{
				Line:   name.Line,
				Column: name.Column,
			},
			End: ast.Position{
				Line:   name.Line,
				Column: name.Column + len(name.Value),
			},
		},
	}, nil
}

// parseExpression parses an expression.
// Expression → Or
// Note: No depth tracking here since parseUnary and parsePrimary handle it
func (p *RecursiveDescentParser) parseExpression() (ast.Node, error) {
	return p.parseOr()
}

// parseOr parses OR expressions.
// Or → And ( 'or' And )*
func (p *RecursiveDescentParser) parseOr() (ast.Node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.match(lexer.OR) {
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}

		left = &ast.BinaryOp{
			Operator: "or",
			Left:     left,
			Right:    right,
			Range:    ast.SpanNodes(left, right),
		}
	}

	return left, nil
}

// parseAnd parses AND expressions.
// And → Comparison ( 'and' Comparison )*
func (p *RecursiveDescentParser) parseAnd() (ast.Node, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}

	for p.match(lexer.AND) {
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}

		left = &ast.BinaryOp{
			Operator: "and",
			Left:     left,
			Right:    right,
			Range:    ast.SpanNodes(left, right),
		}
	}

	return left, nil
}

// parseComparison parses comparison operators.
// Comparison → Additive ( ('=='|'!='|'>'|'<'|'>='|'<=') Additive )*
func (p *RecursiveDescentParser) parseComparison() (ast.Node, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}

	for p.match(lexer.EQUAL, lexer.NOT_EQUAL, lexer.GREATER_THAN, lexer.LESS_THAN, lexer.GREATER_EQUAL, lexer.LESS_EQUAL) {
		op := p.previous()
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}

		left = &ast.ComparisonOp{
			Operator: string(op.Value),
			Left:     left,
			Right:    right,
			Range:    ast.SpanNodes(left, right),
		}
	}

	return left, nil
}

// parseAdditive parses addition and subtraction.
// Additive → Multiplicative ( ('+'|'-') Multiplicative )*
func (p *RecursiveDescentParser) parseAdditive() (ast.Node, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return nil, err
	}

	for p.match(lexer.PLUS, lexer.MINUS) {
		op := p.previous()
		right, err := p.parseMultiplicative()
		if err != nil {
			return nil, err
		}

		left = &ast.BinaryOp{
			Operator: string(op.Value),
			Left:     left,
			Right:    right,
			Range:    ast.SpanNodes(left, right),
		}
	}

	// Check for "as % of", "as napkin", or "as <unit>" keyword
	// Do this at expression level to allow it to apply to entire sub-expressions
	// including those in parentheses
	if p.match(lexer.AS) {
		// "X as % of Y" or "X as a % of Y" — compute X/Y as a percentage
		// Accept optional article "a" between "as" and "%"
		if p.check(lexer.IDENTIFIER) && string(p.peek().Value) == "a" && p.peekAhead(1).Type == lexer.MODULUS {
			p.advance() // consume "a"
		}
		if p.check(lexer.MODULUS) {
			p.advance() // consume "%"
			if !p.match(lexer.OF) {
				return nil, p.error("expected 'of' after 'as %'")
			}
			denominator, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			return &ast.AsPercentOf{
				Numerator:   left,
				Denominator: denominator,
				Range:       left.GetRange(),
			}, nil
		}
		if p.match(lexer.NAPKIN) {
			return &ast.NapkinConversion{
				Expression: left,
				Range:      left.GetRange(),
			}, nil
		}
		if p.match(lexer.PRECISE) {
			return &ast.PreciseConversion{
				Expression: left,
				Range:      left.GetRange(),
			}, nil
		}
		// Accept any identifier or currency code after "as" — validation happens
		// in the evaluator where we have runtime type information. This ensures
		// lines with "as <anything>" stay classified as calculations (not TEXT).
		// Mirrors the "in" keyword pattern at parseMultiplicative which also
		// accepts both IDENTIFIER and CURRENCY_CODE.
		if p.check(lexer.IDENTIFIER) || p.check(lexer.CURRENCY_CODE) {
			targetUnit := string(p.peek().Value)
			p.advance() // always consume the target

			// Resolve known time/unit abbreviations
			normalizedTimeUnit := types.NormalizeTimeUnit(targetUnit)
			resolvedUnit := targetUnit
			if !types.IsValidDurationUnit(targetUnit) && types.IsValidDurationUnit(normalizedTimeUnit) {
				resolvedUnit = normalizedTimeUnit
			}

			node := &ast.UnitConversion{
				Quantity:   left,
				TargetUnit: resolvedUnit,
				Range:      left.GetRange(),
			}
			// Allow chaining: "1 second as hour as precise" or "as napkin"
			if p.match(lexer.AS) {
				if p.match(lexer.PRECISE) {
					return &ast.PreciseConversion{Expression: node, Range: left.GetRange()}, nil
				}
				if p.match(lexer.NAPKIN) {
					return &ast.NapkinConversion{Expression: node, Range: left.GetRange()}, nil
				}
				return nil, p.error("expected 'napkin' or 'precise' after unit conversion 'as'")
			}
			return node, nil
		}
		return nil, p.error("expected 'napkin', 'precise', or a valid unit after 'as'")
	}

	return left, nil
}
