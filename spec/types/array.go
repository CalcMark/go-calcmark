package types

import (
	"fmt"
	"strings"
)

// Array is an ordered, homogeneous collection of values — a table column
// or the result of element-wise arithmetic on one. Elements all share one
// type name (see typeName); a column that mixes types is rejected at
// construction so downstream dispatch never has to guess.
//
// Arrays come from named tables only (no literal syntax, see
// docs/brainstorms/2026-04-06-named-tables-and-arrays-requirements.md R6).
type Array struct {
	Elements []Type
	// ElementType is the typeName of every element ("Number", "Currency",
	// …). Empty for an empty array.
	ElementType string
}

// NewArray builds an Array, requiring every element to share one type.
// An empty slice is a valid (empty) array.
func NewArray(elements []Type) (*Array, error) {
	arr := &Array{Elements: elements}
	for i, el := range elements {
		if el == nil {
			return nil, fmt.Errorf("array element %d is nil", i+1)
		}
		name := typeName(el)
		if i == 0 {
			arr.ElementType = name
			continue
		}
		if name != arr.ElementType {
			return nil, fmt.Errorf("mixed types in array: element %d is %s but earlier elements are %s",
				i+1, name, arr.ElementType)
		}
	}
	return arr, nil
}

// Len returns the number of elements.
func (a *Array) Len() int {
	return len(a.Elements)
}

// String renders the elements as `[a, b, c]` using each element's String().
func (a *Array) String() string {
	parts := make([]string, len(a.Elements))
	for i, el := range a.Elements {
		parts[i] = el.String()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
