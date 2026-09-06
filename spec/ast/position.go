// Package ast defines the Abstract Syntax Tree node types for CalcMark
package ast

import "fmt"

// Position represents a position in source text (1-indexed)
type Position struct {
	Line   int
	Column int
}

// String formats the position as "line:column"
func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

// Range represents a range in source text
type Range struct {
	Start Position
	End   Position
}

// String formats the range as "start-end"
func (r Range) String() string {
	return fmt.Sprintf("%s-%s", r.Start, r.End)
}

// SpanNodes returns a Range spanning from the start of left to the end of right.
// Returns the best partial range if one side is nil; returns nil if both are nil.
func SpanNodes(left, right Node) *Range {
	lr := left.GetRange()
	rr := right.GetRange()
	if lr == nil && rr == nil {
		return nil
	}
	if lr == nil {
		return rr
	}
	if rr == nil {
		return lr
	}
	return &Range{Start: lr.Start, End: rr.End}
}

// SetRangeIfMissing gives n the range r when n has no usable range yet.
// A nil Range and a zero-valued Range (`&Range{}`) both count as missing.
// Nodes that already know their position keep it, so callers can apply
// this unconditionally at parser boundaries without clobbering the
// tighter ranges set by inner constructors.
func SetRangeIfMissing(n Node, r *Range) {
	if n == nil || r == nil {
		return
	}
	if existing := n.GetRange(); existing != nil && existing.Start.Line > 0 {
		return
	}
	switch v := n.(type) {
	case *NumberLiteral:
		v.Range = r
	case *CurrencyLiteral:
		v.Range = r
	case *FractionLiteral:
		v.Range = r
	case *QuantityLiteral:
		v.Range = r
	case *UnitConversion:
		v.Range = r
	case *NapkinConversion:
		v.Range = r
	case *PreciseConversion:
		v.Range = r
	case *PercentageOf:
		v.Range = r
	case *AsPercentOf:
		v.Range = r
	case *RateLiteral:
		v.Range = r
	case *DateLiteral:
		v.Range = r
	case *TimeLiteral:
		v.Range = r
	case *RelativeDateLiteral:
		v.Range = r
	case *EndOfExpr:
		v.Range = r
	case *StartOfExpr:
		v.Range = r
	case *BetweenExpr:
		v.Range = r
	case *LengthOfExpr:
		v.Range = r
	case *DurationLiteral:
		v.Range = r
	case *BooleanLiteral:
		v.Range = r
	case *Identifier:
		v.Range = r
	case *UnaryOp:
		v.Range = r
	case *BinaryOp:
		v.Range = r
	case *ComparisonOp:
		v.Range = r
	case *Assignment:
		v.Range = r
	case *Expression:
		v.Range = r
	case *FunctionCall:
		v.Range = r
	case *DirectiveRef:
		v.Range = r
	case *MemberAccess:
		v.Range = r
	}
}
