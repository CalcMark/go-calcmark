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

	// Logical operators (Go spec compliant)
	// See: https://go.dev/ref/spec#Logical_operators
	AND // "and"
	OR  // "or"
	NOT // "not"

	// Grouping
	LPAREN
	RPAREN

	// Punctuation
	COMMA // ","

	// Reserved keywords for future control flow
	IF
	THEN
	ELSE
	ELIF
	END
	FOR
	IN
	WHILE
	RETURN
	BREAK
	CONTINUE
	LET
	CONST

	// Reserved function names (cannot be used as variables)
	FUNC_AVG  // "avg" - canonical name
	FUNC_SQRT // "sqrt" - canonical name

	// Multi-token function keywords (aliases)
	FUNC_AVERAGE_OF     // "average of" → maps to "avg"
	FUNC_SQUARE_ROOT_OF // "square root of" → maps to "sqrt"

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
	case AND:
		return "AND"
	case OR:
		return "OR"
	case NOT:
		return "NOT"
	case LPAREN:
		return "LPAREN"
	case RPAREN:
		return "RPAREN"
	case COMMA:
		return "COMMA"
	case IF:
		return "IF"
	case THEN:
		return "THEN"
	case ELSE:
		return "ELSE"
	case ELIF:
		return "ELIF"
	case END:
		return "END"
	case FOR:
		return "FOR"
	case IN:
		return "IN"
	case WHILE:
		return "WHILE"
	case RETURN:
		return "RETURN"
	case BREAK:
		return "BREAK"
	case CONTINUE:
		return "CONTINUE"
	case LET:
		return "LET"
	case CONST:
		return "CONST"
	case FUNC_AVG:
		return "FUNC_AVG"
	case FUNC_SQRT:
		return "FUNC_SQRT"
	case FUNC_AVERAGE_OF:
		return "FUNC_AVERAGE_OF"
	case FUNC_SQUARE_ROOT_OF:
		return "FUNC_SQUARE_ROOT_OF"
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
