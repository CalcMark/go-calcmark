package parser

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/lexer"
	"github.com/CalcMark/go-calcmark/spec/units"
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
	for _, t := range types {
		if p.check(t) {
			p.advance()
			return true
		}
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
	// Try assignment first (identifier '=' expression)
	if p.check(lexer.IDENTIFIER) && p.peekAhead(1).Type == lexer.ASSIGN {
		return p.parseAssignment()
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
	}, nil
}

// parseExpression parses an expression (just delegates to comparison).
// Expression → Comparison
// Note: No depth tracking here since parseUnary and parsePrimary handle it
func (p *RecursiveDescentParser) parseExpression() (ast.Node, error) {
	return p.parseComparison()
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
		}
	}

	return left, nil
}

// parseMultiplicative parses multiplication, division, modulus, and unit conversions.
// Multiplicative → Exponent ( ('*'|'/'|'%') Exponent )* ('in' UNIT)?
func (p *RecursiveDescentParser) parseMultiplicative() (ast.Node, error) {
	left, err := p.parseExponent()
	if err != nil {
		return nil, err
	}

	for p.match(lexer.MULTIPLY, lexer.DIVIDE, lexer.MODULUS) {
		op := p.previous()

		// Special case: Check if DIVIDE might be a rate (e.g., "100 MB/s")
		// Rate syntax: quantity / timeunit (no spaces around /)
		// Division: any expression / any expression
		if op.Type == lexer.DIVIDE {
			// Look ahead to see if next token is a time unit identifier
			if p.check(lexer.IDENTIFIER) {
				nextToken := p.peek()
				timeUnit := string(nextToken.Value)

				// Check if it's a valid time unit
				if isTimeUnit(timeUnit) {
					p.advance() // Consume the time unit
					left = &ast.RateLiteral{
						Amount:     left,
						PerUnit:    timeUnit,
						SourceText: "",
						Range:      &ast.Range{},
					}
					// Break out of multiplication loop to check for OVER/PER conversion
					break
				}
			}
		}

		right, err := p.parseExponent()
		if err != nil {
			return nil, err
		}

		left = &ast.BinaryOp{
			Operator: string(op.Value),
			Left:     left,
			Right:    right,
		}
	}

	// Check for rate with "per" keyword: "5 GB per day"
	// But skip if left is already a RateLiteral (from slash syntax)
	if _, isRate := left.(*ast.RateLiteral); !isRate {
		if p.match(lexer.PER) {
			if !p.match(lexer.IDENTIFIER) {
				return nil, p.error("expected time unit after 'per'")
			}
			timeUnit := string(p.previous().Value)

			// Validate it's a time unit
			if !isTimeUnit(timeUnit) {
				return nil, p.error(fmt.Sprintf("'%s' is not a valid time unit", timeUnit))
			}

			left = &ast.RateLiteral{
				Amount:     left,
				PerUnit:    timeUnit,
				SourceText: "",
				Range:      &ast.Range{},
			}
		}
	}

	// Check for "over" keyword: "100 MB/s over 1 day"
	// Natural syntax for accumulate(rate, time_period)
	if p.match(lexer.OVER) {
		// Parse duration/time period
		duration, err := p.parseExponent()
		if err != nil {
			return nil, err
		}

		// Create function call: accumulate(left, duration)
		return &ast.FunctionCall{
			Name:      "accumulate",
			Arguments: []ast.Node{left, duration},
			Range:     &ast.Range{},
		}, nil
	}

	// Check for "per" after a rate (conversion context)
	// Example: "(100 MB/day) per second" - converts existing rate
	if _, isRate := left.(*ast.RateLiteral); isRate {
		if p.match(lexer.PER) {
			if !p.match(lexer.IDENTIFIER) {
				return nil, p.error("expected time unit after 'per' for rate conversion")
			}
			targetUnit := string(p.previous().Value)

			// Validate it's a time unit
			if !isTimeUnit(targetUnit) {
				return nil, p.error(fmt.Sprintf("'%s' is not a valid time unit for conversion", targetUnit))
			}

			// Create function call: convert_rate(rate, target_unit)
			// Pass target unit as an identifier node
			targetNode := &ast.Identifier{
				Name:  targetUnit,
				Range: &ast.Range{},
			}

			return &ast.FunctionCall{
				Name:      "convert_rate",
				Arguments: []ast.Node{left, targetNode},
				Range:     &ast.Range{},
			}, nil
		}
	}

	// Check for "with" keyword: "10000 req/s with 450 req/s capacity"
	// Natural syntax for requires(load, capacity, buffer?)
	if p.match(lexer.WITH) {
		// Parse capacity expression - use parseMultiplicative() to handle rates
		// (parseExponent() would miss the /s part of slash-rates)
		capacity, err := p.parseMultiplicative()
		if err != nil {
			return nil, err
		}

		// Check for optional "and N%" buffer
		var args []ast.Node
		if p.match(lexer.AND) {
			// Parse buffer percentage - parseExponent() is fine here
			bufferExpr, err := p.parseExponent()
			if err != nil {
				return nil, err
			}
			args = []ast.Node{left, capacity, bufferExpr}
		} else {
			args = []ast.Node{left, capacity}
		}

		// Create function call: requires(load, capacity, buffer?)
		return &ast.FunctionCall{
			Name:      "requires",
			Arguments: args,
			Range:     &ast.Range{},
		}, nil
	}

	// Check for "downtime" keyword: "99.9% downtime per month"
	// Natural syntax for downtime(availability%, time_period)
	if p.match(lexer.DOWNTIME) {
		// Expect "per" keyword
		if !p.match(lexer.PER) {
			return nil, p.error("expected 'per' after 'downtime'")
		}

		// Parse time period (identifier like "month", "year", etc.)
		if !p.match(lexer.IDENTIFIER) {
			return nil, p.error("expected time period after 'downtime per'")
		}
		timePeriod := p.previous()

		// Create function call: downtime(left, time_period_identifier)
		return &ast.FunctionCall{
			Name: "downtime",
			Arguments: []ast.Node{
				left,
				&ast.Identifier{Name: string(timePeriod.Value)},
			},
			Range: &ast.Range{},
		}, nil
	}

	// Check for unit conversion: "10 meters in feet" or "10 feet in nautical miles"
	if p.match(lexer.IN) {
		if !p.match(lexer.IDENTIFIER) {
			return nil, p.error("expected unit name after 'in'")
		}
		targetUnit := p.previous()
		targetUnitName := string(targetUnit.Value)

		// Check for multi-word target unit: "in nautical miles"
		if p.check(lexer.IDENTIFIER) {
			nextWord := string(p.peek().Value)
			if multiWordUnit := units.IsMultiWordUnit(targetUnitName, nextWord); multiWordUnit != "" {
				p.advance() // Consume the second word
				targetUnitName = multiWordUnit
			}
		}

		return &ast.UnitConversion{
			Quantity:   left,
			TargetUnit: targetUnitName,
			Range:      &ast.Range{},
		}, nil
	}

	return left, nil
}

// parseExponent parses exponentiation (right-associative).
// Exponent → Unary ('^' Exponent)?
func (p *RecursiveDescentParser) parseExponent() (ast.Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	if p.match(lexer.EXPONENT) {
		op := p.previous()
		right, err := p.parseExponent() // Right-associative recursion
		if err != nil {
			return nil, err
		}

		return &ast.BinaryOp{
			Operator: string(op.Value),
			Left:     left,
			Right:    right,
		}, nil
	}

	return left, nil
}

// parseUnary parses unary operators.
// Unary → ('+'|'-') Unary | Primary
func (p *RecursiveDescentParser) parseUnary() (ast.Node, error) {
	// Security: track depth for recursive unary (e.g., ---5)
	if p.match(lexer.PLUS, lexer.MINUS) {
		if err := p.enterDepth(); err != nil {
			return nil, err
		}
		defer p.exitDepth()

		op := p.previous()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}

		return &ast.UnaryOp{
			Operator: string(op.Value),
			Operand:  operand,
		}, nil
	}

	return p.parsePrimary()
}

// parsePrimary parses primary expressions (atomic values and higher precedence constructs).
// Primary → NUMBER | BOOLEAN | IDENTIFIER | FUNCTION | CURRENCY | '(' Expression ')' | ...
func (p *RecursiveDescentParser) parsePrimary() (ast.Node, error) {
	// Numbers (with multipliers)
	if p.match(lexer.NUMBER, lexer.NUMBER_K, lexer.NUMBER_M, lexer.NUMBER_B, lexer.NUMBER_T,
		lexer.NUMBER_PERCENT, lexer.NUMBER_SCI) {
		tok := p.previous()

		// Check if followed by an identifier (unit): "12k meters", "1e3 kg", etc.
		if p.check(lexer.IDENTIFIER) {
			unitTok := p.advance()
			unitName := string(unitTok.Value)

			// Check for multi-word units: "1 nautical mile", "5 metric tons"
			// Only consume second word if it forms a valid multi-word unit
			if p.check(lexer.IDENTIFIER) {
				nextWord := string(p.peek().Value)
				// Import units package to check if this is a known multi-word unit
				if multiWordUnit := units.IsMultiWordUnit(unitName, nextWord); multiWordUnit != "" {
					p.advance() // Consume the second word
					unitName = multiWordUnit
				}
			}

			return &ast.QuantityLiteral{
				Value:      string(tok.Value),
				Unit:       unitName,
				SourceText: string(tok.OriginalText) + " " + unitName,
			}, nil
		}

		return &ast.NumberLiteral{
			Value:      string(tok.Value),
			SourceText: string(tok.OriginalText),
		}, nil
	}

	// Booleans
	if p.match(lexer.BOOLEAN) {
		tok := p.previous()
		return &ast.BooleanLiteral{
			Value: string(tok.Value),
		}, nil
	}

	// Prefix currency symbols: $100, €50, £30, ¥1000
	// These combine into QuantityLiteral with mapped unit
	if p.match(lexer.CURRENCY_SYM) {
		currencyTok := p.previous()

		// Must be followed by a number
		if !p.match(lexer.NUMBER, lexer.NUMBER_K, lexer.NUMBER_M, lexer.NUMBER_B, lexer.NUMBER_T,
			lexer.NUMBER_PERCENT, lexer.NUMBER_SCI) {
			return nil, p.error("expected number after currency symbol")
		}

		numberTok := p.previous()

		// Map currency symbol to ISO code
		currencyCode := mapCurrencySymbol(string(currencyTok.Value))

		// Create QuantityLiteral with currency as unit
		// SourceText preserves the original format: "$100"
		return &ast.QuantityLiteral{
			Value:      string(numberTok.Value),
			Unit:       currencyCode,
			SourceText: string(currencyTok.OriginalText) + string(numberTok.OriginalText),
		}, nil
	}

	// Quantity literals: number with unit (5 kg, 10 meters, 100 USD, $50)
	if p.match(lexer.QUANTITY) {
		tok := p.previous()
		// Value format: "number:unit" (e.g., "5:kg", "100:USD", "50:$")
		parts := strings.Split(string(tok.Value), ":")
		if len(parts) != 2 {
			return nil, p.error(fmt.Sprintf("invalid quantity format: %s", tok.Value))
		}

		// Check if it's a currency (unit is a currency code or symbol)
		unit := parts[1]
		if isCurrency(unit) {
			return &ast.CurrencyLiteral{
				Value:  parts[0],
				Symbol: unit,
				Range:  &ast.Range{},
			}, nil
		}

		// Regular quantity (unit of measurement)
		return &ast.QuantityLiteral{
			Value: parts[0],
			Unit:  unit,
			Range: &ast.Range{},
		}, nil
	}

	// Currency symbols followed by number: $100
	if p.match(lexer.CURRENCY_SYM) {
		symbol := p.previous()
		if !p.match(lexer.NUMBER, lexer.NUMBER_K, lexer.NUMBER_M, lexer.NUMBER_B, lexer.NUMBER_T) {
			return nil, p.error("expected number after currency symbol")
		}
		value := p.previous()
		return &ast.CurrencyLiteral{
			Symbol:     string(symbol.Value),
			Value:      string(value.Value),
			SourceText: string(symbol.Value) + string(value.Value),
		}, nil
	}

	// Quantity literals: "100 USD", "10 meters", "5 kg"
	// Lexer tokenizes these as QUANTITY with format "value:unit"
	// Performance: O(1) token check, O(1) string split
	if p.match(lexer.QUANTITY) {
		tok := p.previous()
		parts := strings.Split(string(tok.Value), ":")
		if len(parts) != 2 {
			return nil, p.error("invalid quantity format")
		}

		value := parts[0]
		unit := parts[1]

		// Check if unit is a 3-letter uppercase code (currency)
		// Syntactic check only - semantic validation happens later
		if len(unit) == 3 && isAllUppercase(unit) {
			// Currency code: "100 USD"
			return &ast.CurrencyLiteral{
				Symbol:     unit, // Will be "USD", "EUR", etc.
				Value:      value,
				SourceText: value + " " + unit,
			}, nil
		}

		// Regular quantity: "10 meters", "5 kg"
		// Performance: O(1) lookup in unit map (not shown here, deferred to semantic)
		return &ast.QuantityLiteral{
			Value:      value,
			Unit:       unit,
			SourceText: value + " " + unit,
		}, nil
	}

	// Function calls: avg(...), sqrt(...)
	if p.match(lexer.FUNC_AVG, lexer.FUNC_SQRT) {
		return p.parseFunctionCall()
	}

	// Natural language functions: "average of", "square root of"
	if p.match(lexer.FUNC_AVERAGE_OF, lexer.FUNC_SQUARE_ROOT_OF) {
		return p.parseNaturalLanguageFunction()
	}

	// Date keywords: today, tomorrow, yesterday, this/next/last week/month/year
	if p.match(lexer.DATE_TODAY, lexer.DATE_TOMORROW, lexer.DATE_YESTERDAY,
		lexer.DATE_THIS_WEEK, lexer.DATE_THIS_MONTH, lexer.DATE_THIS_YEAR,
		lexer.DATE_NEXT_WEEK, lexer.DATE_NEXT_MONTH, lexer.DATE_NEXT_YEAR,
		lexer.DATE_LAST_WEEK, lexer.DATE_LAST_MONTH, lexer.DATE_LAST_YEAR) {
		tok := p.previous()
		return &ast.RelativeDateLiteral{
			Keyword:    string(tok.Value),
			SourceText: string(tok.Value),
		}, nil
	}

	// Date literals: "Dec 12", "December 25 2025"
	if p.match(lexer.DATE_LITERAL) {
		tok := p.previous()
		// Value format: "Month:Day:Year" (e.g., "December:25:2025")
		parts := strings.Split(string(tok.Value), ":")

		var year *string
		if len(parts) >= 3 && parts[2] != "" {
			year = &parts[2]
		}

		return &ast.DateLiteral{
			Month:      parts[0],
			Day:        parts[1],
			Year:       year,
			SourceText: string(tok.OriginalText),
		}, nil
	}

	// Duration literals: "2 days", "3 weeks and 4 days"
	if p.match(lexer.DURATION_LITERAL) {
		tok := p.previous()
		// Value format: "value:unit:value:unit:..." (e.g., "2:week:3:day")
		parts := strings.Split(string(tok.Value), ":")

		// For now, use first value/unit pair
		// Semantic analyzer will handle compound durations
		return &ast.DurationLiteral{
			Value:      parts[0],
			Unit:       parts[1],
			SourceText: string(tok.OriginalText),
		}, nil
	}

	// Parenthesized expression
	if p.match(lexer.LPAREN) {
		// Security: track nesting depth for parentheses
		if err := p.enterDepth(); err != nil {
			return nil, err
		}
		defer p.exitDepth()

		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		if _, err := p.consume(lexer.RPAREN, "expected ')' after expression"); err != nil {
			return nil, err
		}

		return expr, nil
	}

	// Identifiers (variables or function calls)
	if p.match(lexer.IDENTIFIER) {
		name := p.previous()

		// Check if it's a function call (identifier followed by '(')
		if p.check(lexer.LPAREN) {
			// This is a function call, parse it
			return p.parseFunctionCall()
		}

		// Otherwise it's just a variable reference
		return &ast.Identifier{Name: string(name.Value)}, nil
	}

	// Number followed by identifier/unit: "100 meters", "5 kg"
	// This is handled by checking after number parsing, but lexer might tokenize differently
	// TODO: Handle quantity literals properly

	// If we get here, we don't know what this is
	current := p.peek()
	return nil, p.errorAt(current, fmt.Sprintf("unexpected token: %s", current.Type))
}

// parseFunctionCall parses a function call.
// FunctionCall → FUNC_NAME '(' ArgumentList ')'
func (p *RecursiveDescentParser) parseFunctionCall() (ast.Node, error) {
	funcName := p.previous() // Already consumed by match()

	if _, err := p.consume(lexer.LPAREN, "expected '(' after function name"); err != nil {
		return nil, err
	}

	// Parse arguments
	var args []ast.Node

	// Empty argument list
	if p.check(lexer.RPAREN) {
		p.advance()
		return &ast.FunctionCall{
			Name:      string(funcName.Value),
			Arguments: args,
		}, nil
	}

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

	if _, err := p.consume(lexer.RPAREN, "expected ')' after arguments"); err != nil {
		return nil, err
	}

	// Validate argument counts based on function
	funcNameStr := string(funcName.Value)
	if funcNameStr == "avg" && len(args) == 0 {
		return nil, p.error("avg() requires at least 1 argument")
	}
	if funcNameStr == "sqrt" {
		if len(args) == 0 {
			return nil, p.error("sqrt() requires exactly 1 argument")
		}
		if len(args) > 1 {
			return nil, p.error("sqrt() requires exactly one argument")
		}
	}

	return &ast.FunctionCall{
		Name:      funcNameStr,
		Arguments: args,
	}, nil
}

// isTimeUnit checks if a string is a valid time unit for rate expressions.
// Valid units: second(s), minute(s), hour(s), day(s), week(s), month(s), year(s), and abbreviations
func isTimeUnit(unit string) bool {
	normalized := strings.ToLower(strings.TrimSpace(unit))

	timeUnits := map[string]bool{
		// Full names and plurals
		"second": true, "seconds": true,
		"minute": true, "minutes": true,
		"hour": true, "hours": true,
		"day": true, "days": true,
		"week": true, "weeks": true,
		"month": true, "months": true,
		"year": true, "years": true,

		// Common abbreviations
		"s": true, "sec": true,
		"m": true, "min": true,
		"h": true, "hr": true,
		"d": true,
		"w": true, "wk": true,
		"mo": true,
		"y":  true, "yr": true,
	}

	return timeUnits[normalized]
}
