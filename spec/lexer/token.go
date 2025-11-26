// Package lexer implements the CalcMark lexer/tokenizer
package lexer

import "fmt"

// TokenType represents the type of a token
type TokenType int

const (
	// Literals
	NUMBER   TokenType = iota
	CURRENCY           // DEPRECATED: Use QUANTITY (kept for backward compatibility)
	QUANTITY           // Unified type for numbers with units (currency, measurements, etc.)
	BOOLEAN
	IDENTIFIER

	// Currency (split for parser)
	CURRENCY_SYM  // $
	CURRENCY_CODE // USD

	// Multiplier Numbers (for gocc integration)
	NUMBER_PERCENT // 5%
	NUMBER_K       // 12k
	NUMBER_M       // 1.2M
	NUMBER_B       // 5B
	NUMBER_T       // 2.5T
	NUMBER_SCI     // 1.2e10, 4.5e-7

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
	PER      // "per" - for rate expressions (e.g., "100 MB per second")
	OVER     // "over" - for rate accumulation (e.g., "100 MB/s over 1 day")
	WITH     // "with" - for capacity planning (e.g., "10000 req/s with 450 req/s capacity")
	DOWNTIME // "downtime" - for availability (e.g., "99.9% downtime per month")
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

	// Date keywords
	DATE_TODAY     // "today"
	DATE_TOMORROW  // "tomorrow"
	DATE_YESTERDAY // "yesterday"

	// Relative date keywords
	DATE_THIS_WEEK  // "this week"
	DATE_THIS_MONTH // "this month"
	DATE_THIS_YEAR  // "this year"
	DATE_NEXT_WEEK  // "next week"
	DATE_NEXT_MONTH // "next month"
	DATE_NEXT_YEAR  // "next year"
	DATE_LAST_WEEK  // "last week"
	DATE_LAST_MONTH // "last month"
	DATE_LAST_YEAR  // "last year"

	// Date/Duration literals (combined by lexer)
	DATE_LITERAL     // "Dec 12", "December 25 2025"
	DURATION_LITERAL // "2 days", "3 weeks and 4 days"

	// Special
	NEWLINE
	EOF
)

// String returns the string representation of a TokenType
func (tt TokenType) String() string {
	switch tt {
	case NUMBER:
		return "NUMBER"
	case NUMBER_PERCENT:
		return "NUMBER_PERCENT"
	case NUMBER_K:
		return "NUMBER_K"
	case NUMBER_M:
		return "NUMBER_M"
	case NUMBER_B:
		return "NUMBER_B"
	case NUMBER_T:
		return "NUMBER_T"
	case NUMBER_SCI:
		return "NUMBER_SCI"
	case CURRENCY_SYM:
		return "CURRENCY_SYM"
	case CURRENCY_CODE:
		return "CURRENCY_CODE"
	case CURRENCY:
		return "CURRENCY"
	case QUANTITY:
		return "QUANTITY"
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
	case PER:
		return "PER"
	case OVER:
		return "OVER"
	case WITH:
		return "WITH"
	case DOWNTIME:
		return "DOWNTIME"
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
	case DATE_TODAY:
		return "DATE_TODAY"
	case DATE_TOMORROW:
		return "DATE_TOMORROW"
	case DATE_YESTERDAY:
		return "DATE_YESTERDAY"
	case DATE_THIS_WEEK:
		return "DATE_THIS_WEEK"
	case DATE_THIS_MONTH:
		return "DATE_THIS_MONTH"
	case DATE_THIS_YEAR:
		return "DATE_THIS_YEAR"
	case DATE_NEXT_WEEK:
		return "DATE_NEXT_WEEK"
	case DATE_NEXT_MONTH:
		return "DATE_NEXT_MONTH"
	case DATE_NEXT_YEAR:
		return "DATE_NEXT_YEAR"
	case DATE_LAST_WEEK:
		return "DATE_LAST_WEEK"
	case DATE_LAST_MONTH:
		return "DATE_LAST_MONTH"
	case DATE_LAST_YEAR:
		return "DATE_LAST_YEAR"
	case DATE_LITERAL:
		return "DATE_LITERAL"
	case DURATION_LITERAL:
		return "DURATION_LITERAL"
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
	Type         TokenType
	Value        string // Normalized value (e.g., "1000" with separators stripped)
	OriginalText string // Original text from source (e.g., "1,000")
	Line         int
	Column       int
	StartPos     int // Byte offset in source where token starts
	EndPos       int // Byte offset in source where token ends
}

// String returns a string representation of the token
func (t Token) String() string {
	return fmt.Sprintf("Token(%s, %q, %d:%d)", t.Type, t.Value, t.Line, t.Column)
}
