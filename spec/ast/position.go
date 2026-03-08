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
