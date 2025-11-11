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
