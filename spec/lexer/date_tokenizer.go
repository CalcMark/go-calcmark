package lexer

import (
	"strings"
	"unicode"
)

// tryReadDateKeyword attempts to read a date keyword (today, tomorrow, etc.)
// Returns token type and true if matched, otherwise 0 and false
// Performance: O(1) map lookups
func (l *Lexer) tryReadDateKeyword() (TokenType, bool) {
	startPos := l.pos

	// Try simple keywords first (today, tomorrow, yesterday)
	word := l.peekWord()
	if tokenType, ok := DateKeywords[strings.ToLower(word)]; ok {
		// Consume the word
		l.pos = startPos + len([]rune(word))
		return tokenType, true
	}

	// Try two-word phrases (this week, next month, last year)
	twoWords := l.peekTwoWords()
	if tokenType, ok := RelativeDateKeywords[strings.ToLower(twoWords)]; ok {
		// Consume both words
		l.pos = startPos + len([]rune(twoWords))
		return tokenType, true
	}

	return 0, false
}

// peekWord returns the next word without advancing
func (l *Lexer) peekWord() string {
	var word []rune
	pos := l.pos

	for pos < len(l.text) && (unicode.IsLetter(l.text[pos]) || l.text[pos] == '_') {
		word = append(word, l.text[pos])
		pos++
	}

	return string(word)
}

// peekTwoWords returns the next two words separated by space
func (l *Lexer) peekTwoWords() string {
	pos := l.pos
	var result []rune
	wordsFound := 0

	for pos < len(l.text) && wordsFound < 2 {
		ch := l.text[pos]

		if unicode.IsLetter(ch) || ch == '_' {
			result = append(result, ch)
			pos++
		} else if ch == ' ' && len(result) > 0 {
			result = append(result, ch)
			pos++
			wordsFound++
		} else {
			break
		}
	}

	return strings.TrimSpace(string(result))
}

// tryReadMonthName attempts to read a month name
// Returns canonical month name and true if matched, otherwise empty string and false
// Performance: O(1) map lookup
func (l *Lexer) tryReadMonthName() (string, bool) {
	word := l.peekWord()
	if month, ok := MonthNames[strings.ToLower(word)]; ok {
		return month, true
	}
	return "", false
}

// readDateLiteral reads a date literal: "Dec 12", "December 25 2025"
// Format: MONTH [DAY] [YEAR]
// Performance: O(1) for month lookup, O(k) where k <= 3 components
func (l *Lexer) readDateLiteral() Token {
	startLine := l.line
	startColumn := l.column
	startPos := l.pos

	// Read month name
	month, ok := l.tryReadMonthName()
	if !ok {
		return l.errorToken("expected month name")
	}

	// Consume month word
	monthWord := l.peekWord()
	for i := 0; i < len([]rune(monthWord)); i++ {
		l.advance()
	}

	l.skipWhitespace()

	var day string
	var year string

	// Try to read day (number)
	if unicode.IsDigit(l.currentChar()) {
		numStr := l.readNumberString()
		l.skipWhitespace()

		// Check if this number is a year (4 digits) or a day
		if len(numStr) == 4 {
			// It's a year: "January 2026"
			year = numStr
			day = "1" // Default to 1st
		} else {
			// It's a day: "January 15"
			day = numStr

			// Try to read year (4-digit number)
			if unicode.IsDigit(l.currentChar()) {
				year = l.readNumberString()
			}
		}
	}

	// Default day to 1 if only month (no day or year)
	if day == "" {
		day = "1"
	}

	// Combine into value: "month:day:year"
	value := month + ":" + day
	if year != "" {
		value += ":" + year
	} else {
		value += ":"
	}

	sourceText := string(l.text[startPos:l.pos])

	return Token{
		Type:         DATE_LITERAL,
		Value:        value,
		OriginalText: sourceText,
		Line:         startLine,
		Column:       startColumn,
		StartPos:     startPos,
		EndPos:       l.pos,
	}
}

// readNumberString reads a number and returns it as a string (for date components)
func (l *Lexer) readNumberString() string {
	var num []rune

	for unicode.IsDigit(l.currentChar()) {
		num = append(num, l.currentChar())
		l.advance()
	}

	return string(num)
}

// tryReadTimeUnit attempts to read a time unit (day, week, month, etc.)
// Returns canonical unit name and true if matched, otherwise empty string and false
// Performance: O(1) map lookup
func (l *Lexer) tryReadTimeUnit() (string, bool) {
	word := l.peekWord()
	if unit, ok := TimeUnits[strings.ToLower(word)]; ok {
		return unit, true
	}
	return "", false
}

// readDurationLiteral reads a duration literal: "2 days", "3 weeks and 4 days"
// Format: NUMBER UNIT ["and" NUMBER UNIT]*
// Performance: O(k) where k = number of terms (typically 1-3)
func (l *Lexer) readDurationLiteral() Token {
	startLine := l.line
	startColumn := l.column
	startPos := l.pos

	type term struct {
		value string
		unit  string
	}

	terms := make([]term, 0, 3) // Pre-allocate for typical case

	for {
		// Read number
		if !unicode.IsDigit(l.currentChar()) {
			if len(terms) == 0 {
				return l.errorToken("expected number for duration")
			}
			break
		}

		value := l.readNumberString()
		l.skipWhitespace()

		// Read time unit
		unit, ok := l.tryReadTimeUnit()
		if !ok {
			return l.errorToken("expected time unit (day, week, month, year)")
		}

		// Consume unit word
		unitWord := l.peekWord()
		for i := 0; i < len([]rune(unitWord)); i++ {
			l.advance()
		}

		terms = append(terms, term{value, unit})

		l.skipWhitespace()

		// Check for "and"
		if strings.ToLower(l.peekWord()) != "and" {
			break
		}

		// Consume "and"
		for i := 0; i < 3; i++ {
			l.advance()
		}
		l.skipWhitespace()
	}

	// Combine terms into value: "value:unit:value:unit:..."
	var valueParts []string
	for _, t := range terms {
		valueParts = append(valueParts, t.value, t.unit)
	}
	value := strings.Join(valueParts, ":")

	sourceText := string(l.text[startPos:l.pos])

	return Token{
		Type:         DURATION_LITERAL,
		Value:        value,
		OriginalText: sourceText,
		Line:         startLine,
		Column:       startColumn,
		StartPos:     startPos,
		EndPos:       l.pos,
	}
}

// errorToken creates an error token
func (l *Lexer) errorToken(message string) Token {
	return Token{
		Type:     EOF, // Use EOF to signal error
		Value:    message,
		Line:     l.line,
		Column:   l.column,
		StartPos: l.pos,
		EndPos:   l.pos,
	}
}
