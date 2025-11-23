package parser

// TokenType represents the type of a lexer token for the parser.
// This maps from the hand-written lexer's TokenType to parser-friendly names.
type TokenType int

const (
	// Special tokens
	TOKEN_EOF TokenType = iota
	TOKEN_NEWLINE
	TOKEN_EMPTY

	// Literals
	TOKEN_NUMBER
	TOKEN_IDENTIFIER
	TOKEN_CURRENCY_SYMBOL
	TOKEN_CURRENCY_CODE
	TOKEN_UNIT_NAME

	// Operators
	TOKEN_ASSIGN   // =
	TOKEN_PLUS     // +
	TOKEN_MINUS    // -
	TOKEN_MULTIPLY // *
	TOKEN_DIVIDE   // /
	TOKEN_MODULUS  // %
	TOKEN_CARET    // ^

	// Comparison operators
	TOKEN_EQ  // ==
	TOKEN_NEQ // !=
	TOKEN_GT  // >
	TOKEN_LT  // <
	TOKEN_GTE // >=
	TOKEN_LTE // <=

	// Delimiters
	TOKEN_LPAREN // (
	TOKEN_RPAREN // )
	TOKEN_COMMA  // ,
	TOKEN_COLON  // :

	// Date/Time keywords
	TOKEN_TODAY
	TOKEN_TOMORROW
	TOKEN_YESTERDAY
	TOKEN_NOW

	// Month names
	TOKEN_JANUARY
	TOKEN_FEBRUARY
	TOKEN_MARCH
	TOKEN_APRIL
	TOKEN_MAY
	TOKEN_JUNE
	TOKEN_JULY
	TOKEN_AUGUST
	TOKEN_SEPTEMBER
	TOKEN_OCTOBER
	TOKEN_NOVEMBER
	TOKEN_DECEMBER

	// Time units (singular and plural supported by lexer)
	TOKEN_SECOND
	TOKEN_MINUTE
	TOKEN_HOUR
	TOKEN_DAY
	TOKEN_WEEK
	TOKEN_MONTH
	TOKEN_YEAR

	// Boolean keywords
	TOKEN_TRUE
	TOKEN_FALSE
	TOKEN_YES
	TOKEN_NO

	// Function keywords
	TOKEN_AVG
	TOKEN_SQRT
	TOKEN_CONVERT

	// Multi-word function keywords
	TOKEN_AVERAGE // "average" part of "average of"
	TOKEN_OF      // "of" connector
	TOKEN_SQUARE  // "square" part of "square root"
	TOKEN_ROOT    // "root" part of "square root"

	// Connectors
	TOKEN_AND    // and (for compound durations)
	TOKEN_FROM   // from (for natural language dates)
	TOKEN_BEFORE // before
	TOKEN_AFTER  // after
	TOKEN_TO     // to (for convert)
	TOKEN_IN     // in (for convert context)

	// Time/Period keywords
	TOKEN_AM
	TOKEN_PM
	TOKEN_UTC
)

var tokenNames = map[TokenType]string{
	TOKEN_EOF:             "EOF",
	TOKEN_NEWLINE:         "NEWLINE",
	TOKEN_EMPTY:           "EMPTY",
	TOKEN_NUMBER:          "NUMBER",
	TOKEN_IDENTIFIER:      "IDENTIFIER",
	TOKEN_CURRENCY_SYMBOL: "CURRENCY_SYMBOL",
	TOKEN_CURRENCY_CODE:   "CURRENCY_CODE",
	TOKEN_UNIT_NAME:       "UNIT_NAME",
	TOKEN_ASSIGN:          "=",
	TOKEN_PLUS:            "+",
	TOKEN_MINUS:           "-",
	TOKEN_MULTIPLY:        "*",
	TOKEN_DIVIDE:          "/",
	TOKEN_MODULUS:         "%",
	TOKEN_CARET:           "^",
	TOKEN_EQ:              "==",
	TOKEN_NEQ:             "!=",
	TOKEN_GT:              ">",
	TOKEN_LT:              "<",
	TOKEN_GTE:             ">=",
	TOKEN_LTE:             "<=",
	TOKEN_LPAREN:          "(",
	TOKEN_RPAREN:          ")",
	TOKEN_COMMA:           ",",
	TOKEN_COLON:           ":",
	TOKEN_TODAY:           "today",
	TOKEN_TOMORROW:        "tomorrow",
	TOKEN_YESTERDAY:       "yesterday",
	TOKEN_NOW:             "now",
	TOKEN_JANUARY:         "January",
	TOKEN_FEBRUARY:        "February",
	TOKEN_MARCH:           "March",
	TOKEN_APRIL:           "April",
	TOKEN_MAY:             "May",
	TOKEN_JUNE:            "June",
	TOKEN_JULY:            "July",
	TOKEN_AUGUST:          "August",
	TOKEN_SEPTEMBER:       "September",
	TOKEN_OCTOBER:         "October",
	TOKEN_NOVEMBER:        "November",
	TOKEN_DECEMBER:        "December",
	TOKEN_SECOND:          "second",
	TOKEN_MINUTE:          "minute",
	TOKEN_HOUR:            "hour",
	TOKEN_DAY:             "day",
	TOKEN_WEEK:            "week",
	TOKEN_MONTH:           "month",
	TOKEN_YEAR:            "year",
	TOKEN_TRUE:            "true",
	TOKEN_FALSE:           "false",
	TOKEN_YES:             "yes",
	TOKEN_NO:              "no",
	TOKEN_AVG:             "avg",
	TOKEN_SQRT:            "sqrt",
	TOKEN_CONVERT:         "convert",
	TOKEN_AVERAGE:         "average",
	TOKEN_OF:              "of",
	TOKEN_SQUARE:          "square",
	TOKEN_ROOT:            "root",
	TOKEN_AND:             "and",
	TOKEN_FROM:            "from",
	TOKEN_BEFORE:          "before",
	TOKEN_AFTER:           "after",
	TOKEN_TO:              "to",
	TOKEN_IN:              "in",
	TOKEN_AM:              "AM",
	TOKEN_PM:              "PM",
	TOKEN_UTC:             "UTC",
}

// String returns a human-readable name for the token type.
func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return "UNKNOWN"
}
