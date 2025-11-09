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
		if unicode.IsDigit(char) || char == '.' || char == ',' || char == '_' {
			if char != ',' && char != '_' { // Ignore thousands separators
				numStr.WriteRune(char)
			}
			l.advance()
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
		if unicode.IsDigit(char) || char == '.' || char == ',' || char == '_' {
			if char != ',' && char != '_' { // Ignore thousands separators
				numStr.WriteRune(char)
			}
			l.advance()
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
	if strings.ContainsRune("+-*×/=$><! %^", char) {
		return false
	}

	// Digits can't start an identifier but can be within
	if isFirst && unicode.IsDigit(char) {
		return false
	}

	return true
}

// readIdentifier reads an identifier (variable name)
// Identifiers support any Unicode characters including emoji,
// spaces within the name, and international characters
func (l *Lexer) readIdentifier() Token {
	startLine := l.line
	startColumn := l.column
	var identifier strings.Builder
	isFirst := true

	for l.currentChar() != 0 {
		char := l.currentChar()

		// Handle spaces specially - they're allowed within identifiers
		if char == ' ' {
			// Look ahead to see if there's more identifier content
			nextChar := l.peek(1)
			if nextChar != 0 && l.isIdentifierChar(nextChar, false) {
				identifier.WriteRune(char)
				l.advance()
				continue
			} else {
				// Space at the end, stop reading
				break
			}
		}

		// Check if character is valid for identifier
		if !l.isIdentifierChar(char, isFirst) {
			break
		}

		identifier.WriteRune(char)
		l.advance()
		isFirst = false
	}

	// Strip any trailing spaces
	identStr := strings.TrimRight(identifier.String(), " ")

	// Check if identifier is a boolean keyword
	if booleanKeywords[strings.ToLower(identStr)] {
		return Token{
			Type:   BOOLEAN,
			Value:  strings.ToLower(identStr),
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

	return tokens, nil
}

// Tokenize is a convenience function to tokenize text
func Tokenize(text string) ([]Token, error) {
	lexer := NewLexer(text)
	return lexer.Tokenize()
}
