package parser

import (
	"strings"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/lexer"
)

// parseNLCompoundFunction parses: compound <principal> by <rate> [per <period> | compounded <freq>] over <duration>
// Precondition: "compound" already consumed, next token is start of principal expression.
func (p *RecursiveDescentParser) parseNLCompoundFunction() (ast.Node, error) {
	keyword := p.previous() // "compound" token — for range tracking

	// Parse principal
	principal, err := p.parseExponent()
	if err != nil {
		return nil, err
	}

	// Expect "by" (contextual identifier)
	if !p.matchIdentValue("by") {
		return nil, p.error("compound: expected 'by' after principal amount")
	}

	// Parse rate
	rate, err := p.parseExponent()
	if err != nil {
		return nil, err
	}

	args := []ast.Node{principal, rate}

	// Check for optional modifiers before OVER: "per <period>" or "compounded <frequency>"
	var modifier *ast.Identifier
	if p.match(lexer.PER) {
		// Mode 2: per-period literal rate — "compound $1000 by 5% per month over 12 months"
		if !p.match(lexer.IDENTIFIER) {
			return nil, p.error("compound: expected period after 'per' (e.g., month, quarter)")
		}
		period := strings.ToLower(string(p.previous().Value))
		modifier = &ast.Identifier{Name: period}
	} else if p.checkIdentValue("compounded") {
		// Mode 3: financial compounding — "compound $1000 by 12% compounded monthly over 10 years"
		p.advance() // consume "compounded"
		if !p.match(lexer.IDENTIFIER) {
			return nil, p.error("compound: expected frequency after 'compounded' (e.g., monthly, quarterly)")
		}
		freq := strings.ToLower(string(p.previous().Value))
		modifier = &ast.Identifier{Name: "compounded:" + freq}
	} else if p.checkFrequencyAdverb() {
		// Bare frequency adverb — "compound $1000 by 5% monthly over 10 years"
		// Equivalent to "compounded monthly".
		freq := strings.ToLower(string(p.peek().Value))
		p.advance()
		modifier = &ast.Identifier{Name: "compounded:" + freq}
	}

	// Expect "over"
	if !p.match(lexer.OVER) {
		return nil, p.error("compound: expected 'over' after rate")
	}

	// Parse duration
	duration, err := p.parseExponent()
	if err != nil {
		return nil, err
	}
	args = append(args, duration)

	if modifier != nil {
		args = append(args, modifier)
	}

	return &ast.FunctionCall{
		Name:      "compound",
		Arguments: args,
		Range:     tokenRange(keyword),
	}, nil
}

// parseNLGrowFunction parses: grow <amount> by <increment> over <duration>
// Precondition: "grow" already consumed, next token is start of amount expression.
func (p *RecursiveDescentParser) parseNLGrowFunction() (ast.Node, error) {
	keyword := p.previous() // "grow" token — for range tracking

	// Parse starting amount
	amount, err := p.parseExponent()
	if err != nil {
		return nil, err
	}

	// Expect "by" (contextual identifier)
	if !p.matchIdentValue("by") {
		return nil, p.error("grow: expected 'by' after starting amount")
	}

	// Parse increment
	increment, err := p.parseExponent()
	if err != nil {
		return nil, err
	}

	// Expect "over"
	if !p.match(lexer.OVER) {
		return nil, p.error("grow: expected 'over' after increment")
	}

	// Parse duration
	duration, err := p.parseExponent()
	if err != nil {
		return nil, err
	}

	return &ast.FunctionCall{
		Name:      "grow",
		Arguments: []ast.Node{amount, increment, duration},
		Range:     tokenRange(keyword),
	}, nil
}

// parseNLDepreciateFunction parses: depreciate <value> by <rate> [compounded <freq>] over <duration> [to <salvage>]
// Precondition: "depreciate" already consumed, next token is start of value expression.
func (p *RecursiveDescentParser) parseNLDepreciateFunction() (ast.Node, error) {
	keyword := p.previous() // "depreciate" token — for range tracking

	// Parse value
	value, err := p.parseExponent()
	if err != nil {
		return nil, err
	}

	// Expect "by" (contextual identifier)
	if !p.matchIdentValue("by") {
		return nil, p.error("depreciate: expected 'by' after value")
	}

	// Parse rate
	rate, err := p.parseExponent()
	if err != nil {
		return nil, err
	}

	args := []ast.Node{value, rate}

	// Check for optional "compounded <freq>" modifier
	var modifier *ast.Identifier
	if p.checkIdentValue("compounded") {
		p.advance() // consume "compounded"
		if !p.match(lexer.IDENTIFIER) {
			return nil, p.error("depreciate: expected frequency after 'compounded' (e.g., monthly, quarterly)")
		}
		freq := strings.ToLower(string(p.previous().Value))
		modifier = &ast.Identifier{Name: "compounded:" + freq}
	}

	// Expect "over"
	if !p.match(lexer.OVER) {
		return nil, p.error("depreciate: expected 'over' after rate")
	}

	// Parse duration
	duration, err := p.parseExponent()
	if err != nil {
		return nil, err
	}
	args = append(args, duration)

	if modifier != nil {
		args = append(args, modifier)
	}

	// Check for optional "to <salvage>" — "to" is contextual, only checked here
	if p.checkIdentValue("to") {
		p.advance() // consume "to"
		salvage, err := p.parseExponent()
		if err != nil {
			return nil, err
		}
		args = append(args, salvage)
	}

	return &ast.FunctionCall{
		Name:      "depreciate",
		Arguments: args,
		Range:     tokenRange(keyword),
	}, nil
}

// frequencyAdverbs are adverbial period forms that indicate financial compounding.
var frequencyAdverbs = map[string]bool{
	"daily":     true,
	"weekly":    true,
	"monthly":   true,
	"quarterly": true,
	"yearly":    true,
}

// checkFrequencyAdverb returns true if the current token is a frequency adverb
// (monthly, quarterly, etc.) without consuming it.
func (p *RecursiveDescentParser) checkFrequencyAdverb() bool {
	if !p.check(lexer.IDENTIFIER) {
		return false
	}
	return frequencyAdverbs[strings.ToLower(string(p.peek().Value))]
}

// matchIdentValue checks if the current token is an IDENTIFIER with the given value
// (case-insensitive) and advances past it. Returns true if matched.
func (p *RecursiveDescentParser) matchIdentValue(value string) bool {
	if p.check(lexer.IDENTIFIER) && strings.ToLower(string(p.peek().Value)) == value {
		p.advance()
		return true
	}
	return false
}

// checkIdentValue checks if the current token is an IDENTIFIER with the given value
// (case-insensitive) without consuming it.
func (p *RecursiveDescentParser) checkIdentValue(value string) bool {
	return p.check(lexer.IDENTIFIER) && strings.ToLower(string(p.peek().Value)) == value
}
