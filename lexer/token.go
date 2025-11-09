// Package lexer implements the CalcMark lexer/tokenizer
package lexer

import "fmt"

// TokenType represents the type of a token
type TokenType int

const (
	// Literals
	NUMBER TokenType = iota
	CURRENCY
	BOOLEAN
	IDENTIFIER

	// Operators
	PLUS
	MINUS
	MULTIPLY
	DIVIDE
	MODULUS
	EXPONENT
	ASSIGN

	// Comparison operators
	GREATER_THAN
	LESS_THAN
	GREATER_EQUAL
	LESS_EQUAL
	EQUAL
	NOT_EQUAL

	// Special
	NEWLINE
	EOF
)

// String returns the string representation of a TokenType
func (tt TokenType) String() string {
	switch tt {
	case NUMBER:
		return "NUMBER"
	case CURRENCY:
		return "CURRENCY"
	case BOOLEAN:
		return "BOOLEAN"
	case IDENTIFIER:
		return "IDENTIFIER"
	case PLUS:
		return "PLUS"
	case MINUS:
		return "MINUS"
	case MULTIPLY:
		return "MULTIPLY"
	case DIVIDE:
		return "DIVIDE"
	case MODULUS:
		return "MODULUS"
	case EXPONENT:
		return "EXPONENT"
	case ASSIGN:
		return "ASSIGN"
	case GREATER_THAN:
		return "GREATER_THAN"
	case LESS_THAN:
		return "LESS_THAN"
	case GREATER_EQUAL:
		return "GREATER_EQUAL"
	case LESS_EQUAL:
		return "LESS_EQUAL"
	case EQUAL:
		return "EQUAL"
	case NOT_EQUAL:
		return "NOT_EQUAL"
	case NEWLINE:
		return "NEWLINE"
	case EOF:
		return "EOF"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", tt)
	}
}

// Token represents a lexical token
type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
}

// String returns a string representation of the token
func (t Token) String() string {
	return fmt.Sprintf("Token(%s, %q, %d:%d)", t.Type, t.Value, t.Line, t.Column)
}
