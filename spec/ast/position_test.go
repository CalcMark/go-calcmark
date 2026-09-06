package ast

import "testing"

// Every expression node type must accept a range from SetRangeIfMissing;
// a type left out of the switch would silently keep nil and lose its
// diagnostic position (go-calcmark#164).
func TestSetRangeIfMissing_CoversEveryNodeType(t *testing.T) {
	r := &Range{Start: Position{Line: 1, Column: 3}, End: Position{Line: 1, Column: 7}}
	nodes := []Node{
		&NumberLiteral{}, &CurrencyLiteral{}, &FractionLiteral{}, &QuantityLiteral{},
		&UnitConversion{}, &NapkinConversion{}, &PreciseConversion{}, &PercentageOf{},
		&AsPercentOf{}, &RateLiteral{}, &DateLiteral{}, &TimeLiteral{},
		&RelativeDateLiteral{}, &EndOfExpr{}, &StartOfExpr{}, &BetweenExpr{},
		&LengthOfExpr{}, &DurationLiteral{}, &BooleanLiteral{}, &Identifier{},
		&UnaryOp{}, &BinaryOp{}, &ComparisonOp{}, &Assignment{}, &Expression{},
		&FunctionCall{}, &DirectiveRef{},
	}
	for _, n := range nodes {
		SetRangeIfMissing(n, r)
		if got := n.GetRange(); got == nil || *got != *r {
			t.Errorf("%T: range = %v, want %v", n, got, r)
		}
	}
}

func TestSetRangeIfMissing_KeepsExistingRange(t *testing.T) {
	have := &Range{Start: Position{Line: 2, Column: 5}, End: Position{Line: 2, Column: 6}}
	n := &Identifier{Name: "x", Range: have}
	SetRangeIfMissing(n, &Range{Start: Position{Line: 1, Column: 1}, End: Position{Line: 1, Column: 9}})
	if n.Range != have {
		t.Errorf("existing range was replaced: %v", n.Range)
	}
}

func TestSetRangeIfMissing_TreatsZeroRangeAsMissing(t *testing.T) {
	n := &NumberLiteral{Range: &Range{}}
	r := &Range{Start: Position{Line: 1, Column: 1}, End: Position{Line: 1, Column: 3}}
	SetRangeIfMissing(n, r)
	if *n.Range != *r {
		t.Errorf("zero-valued range was not replaced: %v", n.Range)
	}
}
