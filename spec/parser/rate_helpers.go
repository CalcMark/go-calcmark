package parser

import (
	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/lexer"
)

// parseCapacityValue parses a capacity value for "at ... per UNIT" syntax.
// This handles quantities (2 TB) and slash-rates (450 req/s) but does NOT consume "per".
// The "per" keyword is reserved for the capacity syntax, not rate syntax.
// IMPORTANT: This also does NOT consume "/" followed by a non-time-unit identifier,
// as that "/" should be parsed as part of the capacity syntax (e.g., "2 TB/disk").
func (p *RecursiveDescentParser) parseCapacityValue() (ast.Node, error) {
	// Parse the base expression
	left, err := p.parseExponent()
	if err != nil {
		return nil, err
	}

	// Check for slash-rate syntax: "450 req/s"
	// We need to handle multiply/divide to get the full value
	for p.check(lexer.MULTIPLY) || p.check(lexer.DIVIDE) {
		// Special handling for DIVIDE: check if it's a rate or capacity syntax
		if p.check(lexer.DIVIDE) {
			// Look ahead to see what follows the /
			// Save position to potentially backtrack
			savedPos := p.current

			p.advance() // consume the /

			// Check if next token is an identifier
			if p.check(lexer.IDENTIFIER) {
				nextIdent := string(p.peek().Value)
				if isTimeUnit(nextIdent) {
					// It's a rate (e.g., "450 req/s") - consume the time unit
					p.advance()
					left = &ast.RateLiteral{
						Amount:     left,
						PerUnit:    nextIdent,
						SourceText: "",
						Range:      &ast.Range{},
					}
					continue
				} else {
					// It's capacity syntax (e.g., "2 TB/disk") - backtrack
					// Leave the "/" and identifier for the main parser
					p.current = savedPos
					break
				}
			}

			// Not an identifier after /, treat as division
			right, err := p.parseExponent()
			if err != nil {
				return nil, err
			}
			left = &ast.BinaryOp{
				Operator: "/",
				Left:     left,
				Right:    right,
			}
			continue
		}

		// Handle MULTIPLY
		p.advance() // consume the *
		right, err := p.parseExponent()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryOp{
			Operator: "*",
			Left:     left,
			Right:    right,
		}
	}

	return left, nil
}

// tryParseRateFromDivision attempts to parse a rate from a division operator followed by a time unit.
// Returns (rateNode, true) if successful, (nil, false) otherwise.
// This is a pure function that doesn't modify parser state except for advancing tokens if successful.
func (p *RecursiveDescentParser) tryParseRateFromDivision(left ast.Node) (*ast.RateLiteral, bool) {
	// Check if next token is a time unit identifier (e.g., "s" in "100 MB/s")
	if !p.check(lexer.IDENTIFIER) {
		return nil, false
	}

	timeUnit := string(p.peek().Value)
	if !isTimeUnit(timeUnit) {
		return nil, false
	}

	// Success - consume the time unit and create rate
	p.advance()
	return &ast.RateLiteral{
		Amount:     left,
		PerUnit:    timeUnit,
		SourceText: "",
		Range:      &ast.Range{},
	}, true
}
