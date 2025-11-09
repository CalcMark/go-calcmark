package lexer

import (
	"fmt"
	"strings"
	"unicode"
)

// Boolean keywords
var booleanKeywords = map[string]bool{
	"true":  true,
	"false": true,
	"yes":   true,
	"no":    true,
	"t":     true,
	"f":     true,
	"y":     true,
	"n":     true,
}

// Reserved keywords (Go spec compliant + future control flow)
// See: https://go.dev/ref/spec#Keywords
var reservedKeywords = map[string]TokenType{
	// Logical operators (Go spec)
	"and": AND,
	"or":  OR,
	"not": NOT,

	// Future control flow keywords
	"if":       IF,
	"then":     THEN,
	"else":     ELSE,
	"elif":     ELIF,
	"end":      END,
	"for":      FOR,
	"in":       IN,
	"while":    WHILE,
	"return":   RETURN,
	"break":    BREAK,
	"continue": CONTINUE,
	"let":      LET,
	"const":    CONST,

	// Reserved function names (canonical)
	"avg":  FUNC_AVG,
	"sqrt": FUNC_SQRT,
}

// LexerError represents a lexer error
type LexerError struct {
	Message string
	Line    int
	Column  int
}

func (e *LexerError) Error() string {
	return fmt.Sprintf("%s at %d:%d", e.Message, e.Line, e.Column)
}

// Lexer tokenizes CalcMark expressions
type Lexer struct {
	text   []rune
	pos    int
	line   int
	column int
}

// NewLexer creates a new lexer for the given text
func NewLexer(text string) *Lexer {
	return &Lexer{
		text:   []rune(text),
		pos:    0,
		line:   1,
		column: 1,
	}
}

// currentChar returns the current character or 0 if at end
func (l *Lexer) currentChar() rune {
	if l.pos >= len(l.text) {
		return 0
	}
	return l.text[l.pos]
}

// peek looks ahead at character at given offset
func (l *Lexer) peek(offset int) rune {
	pos := l.pos + offset
	if pos >= len(l.text) {
		return 0
	}
	return l.text[pos]
}

// advance moves to the next character
func (l *Lexer) advance() {
	if l.pos < len(l.text) {
		if l.text[l.pos] == '\n' {
			l.line++
			l.column = 1
		} else {
			l.column++
		}
		l.pos++
	}
}

// skipWhitespace skips whitespace except newlines
func (l *Lexer) skipWhitespace() {
	for l.currentChar() == ' ' || l.currentChar() == '\t' || l.currentChar() == '\r' {
		l.advance()
	}
}

// readNumber reads a number token (supports commas and underscores)
func (l *Lexer) readNumber() Token {
	startLine := l.line
	startColumn := l.column
	var numStr strings.Builder

	for l.currentChar() != 0 {
		char := l.currentChar()
		if unicode.IsDigit(char) || char == '.' {
			numStr.WriteRune(char)
			l.advance()
		} else if (char == ',' || char == '_') && unicode.IsDigit(l.peek(1)) {
			// Only consume comma/underscore if followed by a digit (thousand separator)
			l.advance() // Skip the separator, don't add to numStr
		} else {
			break
		}
	}

	return Token{
		Type:   NUMBER,
		Value:  numStr.String(),
		Line:   startLine,
		Column: startColumn,
	}
}

// readCurrency reads a currency value (e.g., $1,000 or $1000.50)
func (l *Lexer) readCurrency() (Token, error) {
	startLine := l.line
	startColumn := l.column

	// Read the $ symbol
	symbol := l.currentChar()
	l.advance()

	// Read the number part
	var numStr strings.Builder
	for l.currentChar() != 0 {
		char := l.currentChar()
		if unicode.IsDigit(char) || char == '.' {
			numStr.WriteRune(char)
			l.advance()
		} else if (char == ',' || char == '_') && unicode.IsDigit(l.peek(1)) {
			// Only consume comma/underscore if followed by a digit (thousand separator)
			l.advance() // Skip the separator, don't add to numStr
		} else {
			break
		}
	}

	if numStr.Len() == 0 {
		return Token{}, &LexerError{
			Message: "Invalid currency",
			Line:    startLine,
			Column:  startColumn,
		}
	}

	// Store both symbol and value as "symbol:value"
	value := fmt.Sprintf("%c:%s", symbol, numStr.String())

	return Token{
		Type:   CURRENCY,
		Value:  value,
		Line:   startLine,
		Column: startColumn,
	}, nil
}

// isIdentifierChar checks if a character can be part of an identifier
func (l *Lexer) isIdentifierChar(char rune, isFirst bool) bool {
	if char == ' ' || char == '\t' || char == '\r' || char == '\n' {
		return false // Whitespace handled separately
	}

	// Reserved operators and special characters
	if strings.ContainsRune("+-*×/=$><! %^(),", char) {
		return false
	}

	// Digits can't start an identifier but can be within
	if isFirst && unicode.IsDigit(char) {
		return false
	}

	return true
}

// readIdentifier reads an identifier (variable name)
// Identifiers support any Unicode characters including emoji and international characters
// NOTE: Spaces are NOT allowed in identifiers (this allows multi-token function names)
func (l *Lexer) readIdentifier() Token {
	startLine := l.line
	startColumn := l.column
	var identifier strings.Builder
	isFirst := true

	for l.currentChar() != 0 {
		char := l.currentChar()

		// Spaces terminate identifiers (no spaces within identifiers)
		if char == ' ' || char == '\t' || char == '\r' || char == '\n' {
			break
		}

		// Check if character is valid for identifier
		if !l.isIdentifierChar(char, isFirst) {
			break
		}

		identifier.WriteRune(char)
		l.advance()
		isFirst = false
	}

	identStr := identifier.String()
	lowerIdent := strings.ToLower(identStr)

	// Check reserved keywords FIRST (including logical operators and function names)
	if tokenType, isReserved := reservedKeywords[lowerIdent]; isReserved {
		return Token{
			Type:   tokenType,
			Value:  lowerIdent,
			Line:   startLine,
			Column: startColumn,
		}
	}

	// Check if identifier is a boolean keyword
	if booleanKeywords[lowerIdent] {
		return Token{
			Type:   BOOLEAN,
			Value:  lowerIdent,
			Line:   startLine,
			Column: startColumn,
		}
	}

	return Token{
		Type:   IDENTIFIER,
		Value:  identStr,
		Line:   startLine,
		Column: startColumn,
	}
}

// Tokenize tokenizes the entire input
func (l *Lexer) Tokenize() ([]Token, error) {
	var tokens []Token

	for l.currentChar() != 0 {
		l.skipWhitespace()

		if l.currentChar() == 0 {
			break
		}

		char := l.currentChar()

		// Newline
		if char == '\n' {
			tokens = append(tokens, Token{
				Type:   NEWLINE,
				Value:  "\\n",
				Line:   l.line,
				Column: l.column,
			})
			l.advance()
			continue
		}

		// Currency
		if char == '$' {
			token, err := l.readCurrency()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token)
			continue
		}

		// Number
		if unicode.IsDigit(char) {
			tokens = append(tokens, l.readNumber())
			continue
		}

		// Identifier (check before operators)
		if l.isIdentifierChar(char, true) {
			// Check if this might be a single 'x' used as multiply
			// Only treat 'x' as multiply if:
			// 1. It's a single character 'x'
			// 2. Next character is whitespace or digit
			// 3. Previous token was a number
			if char == 'x' || char == 'X' {
				nextChar := l.peek(1)
				if (nextChar == 0 || nextChar == ' ' || nextChar == '\t' ||
					nextChar == '\n' || nextChar == '\r' || unicode.IsDigit(nextChar)) &&
					len(tokens) > 0 && tokens[len(tokens)-1].Type == NUMBER {
					tokens = append(tokens, Token{
						Type:   MULTIPLY,
						Value:  string(char),
						Line:   l.line,
						Column: l.column,
					})
					l.advance()
					continue
				}
			}
			tokens = append(tokens, l.readIdentifier())
			continue
		}

		// Operators
		if char == '+' {
			tokens = append(tokens, Token{
				Type:   PLUS,
				Value:  "+",
				Line:   l.line,
				Column: l.column,
			})
			l.advance()
			continue
		}

		if char == '-' {
			tokens = append(tokens, Token{
				Type:   MINUS,
				Value:  "-",
				Line:   l.line,
				Column: l.column,
			})
			l.advance()
			continue
		}

		if char == '*' || char == '×' {
			// Check for ** (exponent)
			if char == '*' && l.peek(1) == '*' {
				tokens = append(tokens, Token{
					Type:   EXPONENT,
					Value:  "**",
					Line:   l.line,
					Column: l.column,
				})
				l.advance()
				l.advance()
			} else {
				tokens = append(tokens, Token{
					Type:   MULTIPLY,
					Value:  string(char),
					Line:   l.line,
					Column: l.column,
				})
				l.advance()
			}
			continue
		}

		if char == '/' {
			tokens = append(tokens, Token{
				Type:   DIVIDE,
				Value:  "/",
				Line:   l.line,
				Column: l.column,
			})
			l.advance()
			continue
		}

		if char == '%' {
			tokens = append(tokens, Token{
				Type:   MODULUS,
				Value:  "%",
				Line:   l.line,
				Column: l.column,
			})
			l.advance()
			continue
		}

		if char == '^' {
			tokens = append(tokens, Token{
				Type:   EXPONENT,
				Value:  "^",
				Line:   l.line,
				Column: l.column,
			})
			l.advance()
			continue
		}

		// Comparison and assignment operators
		if char == '=' {
			// Check for ==
			if l.peek(1) == '=' {
				tokens = append(tokens, Token{
					Type:   EQUAL,
					Value:  "==",
					Line:   l.line,
					Column: l.column,
				})
				l.advance()
				l.advance()
			} else {
				tokens = append(tokens, Token{
					Type:   ASSIGN,
					Value:  "=",
					Line:   l.line,
					Column: l.column,
				})
				l.advance()
			}
			continue
		}

		if char == '>' {
			// Check for >=
			if l.peek(1) == '=' {
				tokens = append(tokens, Token{
					Type:   GREATER_EQUAL,
					Value:  ">=",
					Line:   l.line,
					Column: l.column,
				})
				l.advance()
				l.advance()
			} else {
				tokens = append(tokens, Token{
					Type:   GREATER_THAN,
					Value:  ">",
					Line:   l.line,
					Column: l.column,
				})
				l.advance()
			}
			continue
		}

		if char == '<' {
			// Check for <=
			if l.peek(1) == '=' {
				tokens = append(tokens, Token{
					Type:   LESS_EQUAL,
					Value:  "<=",
					Line:   l.line,
					Column: l.column,
				})
				l.advance()
				l.advance()
			} else {
				tokens = append(tokens, Token{
					Type:   LESS_THAN,
					Value:  "<",
					Line:   l.line,
					Column: l.column,
				})
				l.advance()
			}
			continue
		}

		if char == '!' {
			// Check for !=
			if l.peek(1) == '=' {
				tokens = append(tokens, Token{
					Type:   NOT_EQUAL,
					Value:  "!=",
					Line:   l.line,
					Column: l.column,
				})
				l.advance()
				l.advance()
				continue
			}
			// Otherwise '!' alone is not a valid token, will fall through to error
		}

		// Parentheses
		if char == '(' {
			tokens = append(tokens, Token{
				Type:   LPAREN,
				Value:  "(",
				Line:   l.line,
				Column: l.column,
			})
			l.advance()
			continue
		}

		if char == ')' {
			tokens = append(tokens, Token{
				Type:   RPAREN,
				Value:  ")",
				Line:   l.line,
				Column: l.column,
			})
			l.advance()
			continue
		}

		// Comma (for function arguments)
		if char == ',' {
			tokens = append(tokens, Token{
				Type:   COMMA,
				Value:  ",",
				Line:   l.line,
				Column: l.column,
			})
			l.advance()
			continue
		}

		// Unknown character
		return nil, &LexerError{
			Message: fmt.Sprintf("Unexpected character '%c'", char),
			Line:    l.line,
			Column:  l.column,
		}
	}

	// Add EOF token
	tokens = append(tokens, Token{
		Type:   EOF,
		Value:  "",
		Line:   l.line,
		Column: l.column,
	})

	// Post-process tokens to combine multi-token function names
	tokens = combineMultiTokenFunctions(tokens)

	return tokens, nil
}

// combineMultiTokenFunctions combines multi-token sequences into single function tokens
// Examples:
//   "average" + "of" → FUNC_AVERAGE_OF
//   "square" + "root" + "of" → FUNC_SQUARE_ROOT_OF
func combineMultiTokenFunctions(tokens []Token) []Token {
	result := make([]Token, 0, len(tokens))
	i := 0

	for i < len(tokens) {
		token := tokens[i]

		// Check for "average of" (case insensitive)
		if token.Type == IDENTIFIER && strings.ToLower(token.Value) == "average" {
			if i+1 < len(tokens) {
				nextToken := tokens[i+1]
				// Check for "of" after "average"
				if nextToken.Type == IDENTIFIER && strings.ToLower(nextToken.Value) == "of" {
					// Combine into FUNC_AVERAGE_OF
					result = append(result, Token{
						Type:   FUNC_AVERAGE_OF,
						Value:  "average of",
						Line:   token.Line,
						Column: token.Column,
					})
					i += 2 // Skip both tokens
					continue
				}
			}
		}

		// Check for "square root of" (case insensitive)
		if token.Type == IDENTIFIER && strings.ToLower(token.Value) == "square" {
			if i+2 < len(tokens) {
				rootToken := tokens[i+1]
				ofToken := tokens[i+2]
				// Check for "root of" after "square"
				if rootToken.Type == IDENTIFIER && strings.ToLower(rootToken.Value) == "root" &&
					ofToken.Type == IDENTIFIER && strings.ToLower(ofToken.Value) == "of" {
					// Combine into FUNC_SQUARE_ROOT_OF
					result = append(result, Token{
						Type:   FUNC_SQUARE_ROOT_OF,
						Value:  "square root of",
						Line:   token.Line,
						Column: token.Column,
					})
					i += 3 // Skip all three tokens
					continue
				}
			}
		}

		// No multi-token match, keep original token
		result = append(result, token)
		i++
	}

	return result
}

// Tokenize is a convenience function to tokenize text
func Tokenize(text string) ([]Token, error) {
	lexer := NewLexer(text)
	return lexer.Tokenize()
}
