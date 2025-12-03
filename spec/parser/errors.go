package parser

import "fmt"

// ParseError represents a parsing error with position information
type ParseError struct {
	Message string
	Line    int
	Column  int
}

func (e *ParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("parse error at line %d, column %d: %s", e.Line, e.Column, e.Message)
	}
	return fmt.Sprintf("parse error: %s", e.Message)
}

// SemanticError represents a semantic validation error
type SemanticError struct {
	Message  string
	VarName  string
	Line     int
	Column   int
	FirstDef *struct {
		Line   int
		Column int
	}
}

func (e *SemanticError) Error() string {
	if e.FirstDef != nil && e.Line > 0 {
		return fmt.Sprintf("semantic error at line %d, column %d: variable %q already defined at line %d, column %d",
			e.Line, e.Column, e.VarName, e.FirstDef.Line, e.FirstDef.Column)
	}
	if e.Line > 0 {
		return fmt.Sprintf("semantic error at line %d, column %d: %s", e.Line, e.Column, e.Message)
	}
	return fmt.Sprintf("semantic error: %s", e.Message)
}
