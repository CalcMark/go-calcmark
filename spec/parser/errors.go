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

// newParseError creates a new ParseError
func newParseError(message string, line, column int) *ParseError {
	return &ParseError{
		Message: message,
		Line:    line,
		Column:  column,
	}
}
