package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/lexer"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/CalcMark/go-calcmark/spec/units"
)

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
			Range:    ast.SpanNodes(left, right),
		}, nil
	}

	return left, nil
}

// parseUnary parses unary operators.
// Unary → ('+'|'-'|'not') Unary | Primary
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
			Range:    operand.GetRange(),
		}, nil
	}

	// Handle NOT operator
	if p.match(lexer.NOT) {
		if err := p.enterDepth(); err != nil {
			return nil, err
		}
		defer p.exitDepth()

		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}

		return &ast.UnaryOp{
			Operator: "not",
			Operand:  operand,
			Range:    operand.GetRange(),
		}, nil
	}

	result, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	// Check for "as napkin" or "as <unit>" postfix (higher precedence than unary operators)
	// "as napkin" → NapkinConversion
	// "as seconds" → UnitConversion (for duration conversion like "1 day as seconds")
	// This ensures "-47 as napkin" parses as "napkin(-47)" not "-(napkin(47))"
	if p.check(lexer.IDENTIFIER) && string(p.peek().Value) == "as" {
		p.advance() // consume "as"
		if p.match(lexer.NAPKIN) {
			return &ast.NapkinConversion{
				Expression: result,
				Range:      result.GetRange(),
			}, nil
		}
		if p.match(lexer.PRECISE) {
			return &ast.PreciseConversion{
				Expression: result,
				Range:      result.GetRange(),
			}, nil
		}
		// Check for "as <unit>" for unit/duration conversion: "1 day as seconds"
		if p.check(lexer.IDENTIFIER) {
			targetUnit := string(p.peek().Value)
			// Resolve time unit abbreviations (e.g., "ms" → "millisecond")
			normalizedTimeUnit := types.NormalizeTimeUnit(targetUnit)
			_, isQuantityUnit := units.NormalizeUnitName(targetUnit)
			isDuration := types.IsValidDurationUnit(targetUnit) || types.IsValidDurationUnit(normalizedTimeUnit)
			if isDuration || isQuantityUnit {
				p.advance() // consume the target unit
				// Use the raw form if already valid, normalize only for abbreviations
				resolvedUnit := targetUnit
				if !types.IsValidDurationUnit(targetUnit) && types.IsValidDurationUnit(normalizedTimeUnit) {
					resolvedUnit = normalizedTimeUnit
				}
				node := &ast.UnitConversion{
					Quantity:   result,
					TargetUnit: resolvedUnit,
					Range:      result.GetRange(),
				}
				// Allow chaining: "1 second as hour as precise" or "as napkin"
				if p.check(lexer.IDENTIFIER) && string(p.peek().Value) == "as" {
					p.advance() // consume "as"
					if p.match(lexer.PRECISE) {
						return &ast.PreciseConversion{Expression: node, Range: result.GetRange()}, nil
					}
					if p.match(lexer.NAPKIN) {
						return &ast.NapkinConversion{Expression: node, Range: result.GetRange()}, nil
					}
					return nil, p.error("expected 'napkin' or 'precise' after unit conversion 'as'")
				}
				return node, nil
			}
		}
		return nil, p.error("expected 'napkin', 'precise', or a valid unit after 'as'")
	}

	return result, nil
}

// parsePrimary parses primary expressions (atomic values and higher precedence constructs).
// Primary → NUMBER | BOOLEAN | IDENTIFIER | FUNCTION | CURRENCY | '(' Expression ')' | ...
func (p *RecursiveDescentParser) parsePrimary() (ast.Node, error) {
	// Number literals (with optional unit)
	// Examples: "42", "3.14", "50%", "10 meters", "1k kg"
	//
	// CRITICAL: Must check if identifier is a KNOWN UNIT before consuming it!
	// Otherwise we incorrectly consume identifiers like "downtime" that come after
	// percentages in expressions like "99.9% downtime per month".
	if p.match(lexer.NUMBER, lexer.NUMBER_K, lexer.NUMBER_M, lexer.NUMBER_B, lexer.NUMBER_T,
		lexer.NUMBER_PERCENT, lexer.NUMBER_SCI) {
		tok := p.previous()

		// Check for "PERCENTAGE OF expression" pattern (e.g., "10% of 200")
		if tok.Type == lexer.NUMBER_PERCENT && p.check(lexer.OF) {
			p.advance() // consume "of"
			value, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			return &ast.PercentageOf{
				Percentage: &ast.NumberLiteral{
					Value:      string(tok.Value),
					SourceText: string(tok.OriginalText),
				},
				Value: value,
			}, nil
		}

		// Check if followed by a unit identifier: "10 meters", "50% coverage", etc.
		// Skips NL keywords like "downtime", "over" that have special meaning.
		// Allows arbitrary units for rates ("cars per day", "requests per second").
		if unitName, consumed := p.tryConsumeUnit(); consumed {
			return &ast.QuantityLiteral{
				Value:      string(tok.Value),
				Unit:       unitName,
				SourceText: string(tok.OriginalText) + " " + unitName,
			}, nil
		}

		// Check for mixed number: integer NUMBER followed by FRACTION (e.g., "11 3/8")
		// Only when the number is an integer (no decimal point) and not a multiplier suffix
		if tok.Type == lexer.NUMBER && !strings.Contains(tok.Value, ".") && p.check(lexer.FRACTION) {
			numNode := &ast.NumberLiteral{
				Value:      string(tok.Value),
				SourceText: string(tok.OriginalText),
				Range:      tokenRange(tok),
			}
			// parseFractionLiteral handles optional unit attachment
			fracNode, err := p.parseFractionLiteral()
			if err != nil {
				return nil, err
			}
			return &ast.BinaryOp{
				Operator: "+",
				Left:     numNode,
				Right:    fracNode,
				Range:    ast.SpanNodes(numNode, fracNode),
			}, nil
		}

		// Plain number without unit (or followed by keyword identifier)
		return &ast.NumberLiteral{
			Value:      string(tok.Value),
			SourceText: string(tok.OriginalText),
		}, nil
	}

	// Fraction literals: "1/3", "7/8"
	if p.check(lexer.FRACTION) {
		return p.parseFractionLiteral()
	}

	// Booleans
	if p.match(lexer.BOOLEAN) {
		tok := p.previous()
		return &ast.BooleanLiteral{
			Value: string(tok.Value),
		}, nil
	}

	// Directive references: @scale, @globals.tax_rate
	if p.match(lexer.AT_SIGN) {
		if !p.match(lexer.IDENTIFIER) {
			return nil, p.error("expected directive name after '@' (e.g., @scale, @globals.name)")
		}
		directiveTok := p.previous()
		directive := string(directiveTok.Value)

		var field string
		if directive == "globals" {
			if !p.match(lexer.DOT) {
				return nil, p.error("@globals requires a field name (e.g., @globals.tax_rate)")
			}
			if !p.match(lexer.IDENTIFIER) {
				return nil, p.error("expected field name after '@globals.' (e.g., @globals.tax_rate)")
			}
			field = string(p.previous().Value)

			// Reject nested dots: @globals.a.b
			if p.check(lexer.DOT) {
				return nil, p.error("nested dot access is not supported (e.g., @globals.a.b); only @globals.<name> is valid")
			}
		}

		ref := &ast.DirectiveRef{
			Directive: directive,
			Field:     field,
		}

		// Check for unit annotation: "@scale meters", "@globals.rate kg"
		if unitName, consumed := p.tryConsumeUnit(); consumed {
			return &ast.QuantityLiteral{
				Expr: ref,
				Unit: unitName,
			}, nil
		}

		return ref, nil
	}

	// Prefix currency symbols: $100, €50, £30, ¥1000
	// These create CurrencyLiteral for proper currency handling
	if p.match(lexer.CURRENCY_SYM) {
		currencyTok := p.previous()

		// Must be followed by a number
		if !p.match(lexer.NUMBER, lexer.NUMBER_K, lexer.NUMBER_M, lexer.NUMBER_B, lexer.NUMBER_T,
			lexer.NUMBER_PERCENT, lexer.NUMBER_SCI) {
			return nil, p.error("expected number after currency symbol")
		}

		numberTok := p.previous()

		// Create CurrencyLiteral (preserves the symbol for display)
		return &ast.CurrencyLiteral{
			Symbol:     string(currencyTok.Value),
			Value:      string(numberTok.Value),
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
		tokRange := &ast.Range{
			Start: ast.Position{Line: tok.Line, Column: tok.Column},
			End:   ast.Position{Line: tok.Line, Column: tok.Column + len(tok.Value)},
		}
		if isCurrency(unit) {
			return &ast.CurrencyLiteral{
				Value:  parts[0],
				Symbol: unit,
				Range:  tokRange,
			}, nil
		}

		// Regular quantity (unit of measurement)
		return &ast.QuantityLiteral{
			Value: parts[0],
			Unit:  unit,
			Range: tokRange,
		}, nil
	}

	// Function calls: avg(...), sqrt(...), sum(...), number(...)
	if p.match(lexer.FUNC_AVG, lexer.FUNC_SQRT, lexer.FUNC_SUM, lexer.FUNC_NUMBER) {
		return p.parseFunctionCall()
	}

	// Natural language functions: "average of", "square root of", "sum of"
	if p.match(lexer.FUNC_AVERAGE_OF, lexer.FUNC_SQUARE_ROOT_OF, lexer.FUNC_SUM_OF) {
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
	// Also handles "X from Y" syntax: "2 days from today"
	if p.match(lexer.DURATION_LITERAL) {
		tok := p.previous()
		// Value format: "value:unit:value:unit:..." (e.g., "2:week:3:day")
		parts := strings.Split(string(tok.Value), ":")

		// For now, use first value/unit pair
		// Semantic analyzer will handle compound durations
		durationNode := &ast.DurationLiteral{
			Value:      parts[0],
			Unit:       parts[1],
			SourceText: string(tok.OriginalText),
		}

		// Check for "from" keyword: "2 days from today"
		// This transforms to: baseDate + duration
		if p.match(lexer.FROM) {
			// Parse the base date expression (today, tomorrow, yesterday, or date literal)
			baseDate, err := p.parseFromTarget()
			if err != nil {
				return nil, err
			}

			// Transform "2 days from today" into "today + 2 days"
			return &ast.BinaryOp{
				Operator: "+",
				Left:     baseDate,
				Right:    durationNode,
				Range:    ast.SpanNodes(baseDate, durationNode),
			}, nil
		}

		return durationNode, nil
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
		identName := strings.ToLower(string(name.Value))

		// NL function lookahead: "read <qty|var> from <ident>"
		if identName == "read" && (p.check(lexer.QUANTITY) || p.check(lexer.IDENTIFIER)) {
			if p.peekAhead(1).Type == lexer.FROM {
				return p.parseNLReadFunction()
			}
		}

		// NL function lookahead: "compress <qty|var> using <ident>"
		if identName == "compress" && (p.check(lexer.QUANTITY) || p.check(lexer.IDENTIFIER)) {
			if p.peekAhead(1).Type == lexer.IDENTIFIER && strings.ToLower(string(p.peekAhead(1).Value)) == "using" {
				return p.parseNLCompressFunction()
			}
		}

		// NL function lookahead: "transfer <qty|var> across <ident> <ident>"
		if identName == "transfer" && (p.check(lexer.QUANTITY) || p.check(lexer.IDENTIFIER)) {
			if p.peekAhead(1).Type == lexer.IDENTIFIER && strings.ToLower(string(p.peekAhead(1).Value)) == "across" {
				return p.parseNLTransferFunction()
			}
		}

		// NL growth function lookahead: "compound|grow|depreciate <expr> by ..."
		// If not followed by LPAREN, it's NL syntax.
		if identName == "compound" && !p.check(lexer.LPAREN) {
			return p.parseNLCompoundFunction()
		}
		if identName == "grow" && !p.check(lexer.LPAREN) {
			return p.parseNLGrowFunction()
		}
		if identName == "depreciate" && !p.check(lexer.LPAREN) {
			return p.parseNLDepreciateFunction()
		}

		// Check if it's a function call (identifier followed by '(')
		if p.check(lexer.LPAREN) {
			// This is a function call, parse it
			return p.parseFunctionCall()
		}

		// Otherwise it's just a variable reference
		return &ast.Identifier{
			Name: string(name.Value),
			Range: &ast.Range{
				Start: ast.Position{Line: name.Line, Column: name.Column},
				End:   ast.Position{Line: name.Line, Column: name.Column + len(name.Value)},
			},
		}, nil
	}

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
		funcNameStr := string(funcName.Value)
		if funcNameStr == "sum" {
			return nil, p.error("sum() requires at least 2 arguments")
		}
		if funcNameStr == "number" {
			return nil, p.error("number() requires exactly 1 argument")
		}
		return &ast.FunctionCall{
			Name:      funcNameStr,
			Arguments: args,
			Range: &ast.Range{
				Start: ast.Position{Line: funcName.Line, Column: funcName.Column},
				End:   ast.Position{Line: funcName.Line, Column: funcName.Column + len(funcName.Value)},
			},
		}, nil
	}

	// Parse first argument
	arg, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	arg = p.maybeCompoundModifier(arg)
	args = append(args, arg)

	// Parse remaining arguments
	for p.match(lexer.COMMA) {
		arg, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		arg = p.maybeCompoundModifier(arg)
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
	if funcNameStr == "sum" && len(args) < 2 {
		return nil, p.error("sum() requires at least 2 arguments")
	}
	if funcNameStr == "sqrt" {
		if len(args) == 0 {
			return nil, p.error("sqrt() requires exactly 1 argument")
		}
		if len(args) > 1 {
			return nil, p.error("sqrt() requires exactly one argument")
		}
	}
	if funcNameStr == "number" {
		if len(args) == 0 {
			return nil, p.error("number() requires exactly 1 argument")
		}
		if len(args) > 1 {
			return nil, p.error("number() requires exactly 1 argument")
		}
	}

	return &ast.FunctionCall{
		Name:      funcNameStr,
		Arguments: args,
		Range: &ast.Range{
			Start: ast.Position{Line: funcName.Line, Column: funcName.Column},
			End:   ast.Position{Line: funcName.Line, Column: funcName.Column + len(funcName.Value)},
		},
	}, nil
}

// maybeCompoundModifier checks if an argument is an Identifier "compounded"
// followed by another IDENTIFIER (e.g., "compounded monthly"). If so, combines
// them into a single Identifier with "compounded:" prefix for the evaluator.
// This handles the functional syntax: compound(1000, 5%, 10, compounded monthly)
func (p *RecursiveDescentParser) maybeCompoundModifier(arg ast.Node) ast.Node {
	ident, ok := arg.(*ast.Identifier)
	if !ok || strings.ToLower(ident.Name) != "compounded" {
		return arg
	}
	if !p.check(lexer.IDENTIFIER) {
		return arg
	}
	p.advance()
	period := strings.ToLower(string(p.previous().Value))
	return &ast.Identifier{
		Name:  "compounded:" + period,
		Range: ident.Range,
	}
}

// parseFromTarget parses the target of a "from" expression.
// Valid targets: today, tomorrow, yesterday, or date literals (Dec 25, Dec 25 2025)
func (p *RecursiveDescentParser) parseFromTarget() (ast.Node, error) {
	// Try relative date keywords first
	if p.match(lexer.DATE_TODAY) {
		return &ast.RelativeDateLiteral{
			Keyword:    "today",
			SourceText: string(p.previous().OriginalText),
		}, nil
	}
	if p.match(lexer.DATE_TOMORROW) {
		return &ast.RelativeDateLiteral{
			Keyword:    "tomorrow",
			SourceText: string(p.previous().OriginalText),
		}, nil
	}
	if p.match(lexer.DATE_YESTERDAY) {
		return &ast.RelativeDateLiteral{
			Keyword:    "yesterday",
			SourceText: string(p.previous().OriginalText),
		}, nil
	}

	// Try date literal (Dec 25, December 25 2025)
	if p.match(lexer.DATE_LITERAL) {
		tok := p.previous()
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

	return nil, p.error("expected date (today, tomorrow, yesterday, or date literal) after 'from'")
}

// isNaturalSyntaxKeyword checks if an identifier is a reserved natural syntax keyword.
// These keywords have special meaning in the grammar and should NOT be consumed as unit names.
// Examples: "downtime", "over", "per", "with", "at", "capacity", "as" are used in natural language constructs.
func isNaturalSyntaxKeyword(ident string) bool {
	switch ident {
	case "at":
		return true // Used in capacity planning: "10 TB at 2 TB per disk"
	case "buffer":
		return true // Used in capacity planning: "with 10% buffer"
	case "downtime":
		return true
	case "over":
		return true
	case "per":
		return true
	case "with":
		return true
	case "capacity":
		return true
	case "as":
		return true // Used in "as napkin" conversion syntax
	case "read", "compress", "transfer":
		return true // NL function triggers: "read 100 MB from ssd", etc.
	case "compound", "grow", "depreciate":
		return true // Growth function NL triggers
	case "by", "compounded":
		return true // Contextual keywords used in growth NL syntax
	case "daily", "weekly", "monthly", "quarterly", "yearly":
		return true // Frequency adverbs used in compound() NL syntax
	default:
		return false
	}
}

// isTimeUnit checks if a string is a valid time unit for rate expressions.
// Valid units: second(s), minute(s), hour(s), day(s), week(s), month(s), year(s), and abbreviations.
// Uses types.NormalizeTimeUnit as the source of truth for time unit recognition.
func isTimeUnit(unit string) bool {
	normalized := types.NormalizeTimeUnit(unit)
	// NormalizeTimeUnit returns the input unchanged if not recognized,
	// but returns a canonical form (second, minute, etc.) if recognized.
	// If the output matches one of the canonical forms, it's a time unit.
	switch normalized {
	case "nanosecond", "microsecond", "millisecond", "second", "minute", "hour", "day", "week", "month", "year":
		return true
	default:
		return false
	}
}

// parseFractionLiteral consumes a FRACTION token and returns a FractionLiteral node.
// The token value is "num/denom" (e.g., "1/3"). Parsing to int64 happens here once.
// Optionally checks for a following unit identifier.
func (p *RecursiveDescentParser) parseFractionLiteral() (ast.Node, error) {
	if !p.match(lexer.FRACTION) {
		return nil, p.error("expected fraction literal")
	}
	tok := p.previous()

	parts := strings.SplitN(tok.Value, "/", 2)
	if len(parts) != 2 {
		return nil, p.error("invalid fraction format: " + tok.Value)
	}

	num, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, p.error("fraction numerator too large: " + parts[0])
	}
	denom, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, p.error("fraction denominator too large: " + parts[1])
	}

	fracNode := &ast.FractionLiteral{
		Numerator:   num,
		Denominator: denom,
		SourceText:  tok.OriginalText,
		Range:       tokenRange(tok),
	}

	// Check for unit after fraction: "1/3 cup"
	if p.check(lexer.IDENTIFIER) {
		identTok := p.peek()
		unitName := string(identTok.Value)
		if !isNaturalSyntaxKeyword(unitName) {
			p.advance()
			normalizedUnit, isKnownUnit := units.NormalizeUnitName(unitName)
			if isKnownUnit {
				unitName = normalizedUnit
			}
			fracNode.Unit = unitName
			fracNode.SourceText = tok.OriginalText + " " + unitName
		}
	}

	return fracNode, nil
}
