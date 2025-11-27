package parser

import (
	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/lexer"
)

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
