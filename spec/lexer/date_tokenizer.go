package lexer

import (
	"strings"
	"unicode"
)

// isPotentialNotationFollow reports whether the rune at pos is a
// digit AND the matched phrase ends with a notation prefix (FQ /
// CQ / FY / CY). This indicates the user intends notation literal
// (FQ1, CY2026) and we should NOT consume the matched phrase
// keyword. Lowercased compare on the trailing two letters of the
// phrase.
func isPotentialNotationFollow(text []rune, pos int, phrase string) bool {
	if pos >= len(text) {
		return false
	}
	if !unicode.IsDigit(text[pos]) {
		return false
	}
	if len(phrase) < 2 {
		return false
	}
	tail := strings.ToLower(phrase[len(phrase)-2:])
	switch tail {
	case "fq", "cq", "fy", "cy":
		return true
	}
	return false
}

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

	// Try two-word phrases (this week, next month, last year). When
	// the phrase ends with a notation prefix (FQ, CQ, FY, CY) and is
	// followed immediately by a digit, do NOT match — `next FQ1`
	// means "FQ1 of next FY" (notation literal), not the bare-phrase
	// `next FQ` keyword. Without this guard the lexer greedily eats
	// `next FQ` and leaves `1` dangling as a NUMBER.
	twoWords := l.peekTwoWords()
	if tokenType, ok := RelativeDateKeywords[strings.ToLower(twoWords)]; ok {
		consumed := len([]rune(twoWords))
		nextPos := startPos + consumed
		// Peek the rune after the matched phrase. Reject the match if
		// it's a digit and the second word is a 2-char notation prefix
		// — covers `next FQ1`, `last CY26`, `this FY2027`, etc.
		if isPotentialNotationFollow(l.text, nextPos, twoWords) {
			// Fall through to single-word match below.
		} else {
			l.pos = startPos + consumed
			return tokenType, true
		}
	}

	// Try three-word phrases (this fiscal quarter, next fiscal year)
	threeWords := l.peekThreeWords()
	if tokenType, ok := ThreeWordDateKeywords[strings.ToLower(threeWords)]; ok {
		// Consume all three words
		l.pos = startPos + len([]rune(threeWords))
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

// peekThreeWords returns the next three words separated by spaces.
// Used for three-word phrases like "this fiscal quarter".
func (l *Lexer) peekThreeWords() string {
	pos := l.pos
	var result []rune
	wordsFound := 0

	for pos < len(l.text) && wordsFound < 3 {
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

// tryReadNotation attempts to read Q1-Q4, FQ1-FQ4, FY26/FY2026, CY26/CY2026 patterns.
// Only matches when the notation is NOT followed by identifier characters (_, letters).
// This prevents "fq1_start" from being tokenized as FQ1 + _start.
func (l *Lexer) tryReadNotation() (TokenType, string, bool) {
	startPos := l.pos
	// peekWord only reads letters; we need letters+digits for notation.
	word := l.peekAlphanumWord()
	upper := strings.ToUpper(word)

	// Check that notation is not followed by identifier chars (prevents fq1_start → FQ1)
	endPos := startPos + len([]rune(word))
	if endPos < len(l.text) && (l.text[endPos] == '_' || unicode.IsLetter(l.text[endPos])) {
		return 0, "", false
	}

	// Q1-Q4: calendar quarter
	if len(upper) == 2 && upper[0] == 'Q' && upper[1] >= '1' && upper[1] <= '4' {
		l.pos = startPos + len([]rune(word))
		return CALENDAR_QUARTER_LITERAL, string(upper[1:]), true
	}

	// FQ1-FQ4: fiscal quarter
	if len(upper) == 3 && upper[:2] == "FQ" && upper[2] >= '1' && upper[2] <= '4' {
		l.pos = startPos + len([]rune(word))
		return FISCAL_QUARTER_LITERAL, string(upper[2:]), true
	}

	// FY + 2-4 digits: fiscal year
	if len(upper) >= 4 && len(upper) <= 6 && upper[:2] == "FY" {
		digits := upper[2:]
		if isAllDigits(digits) {
			l.pos = startPos + len([]rune(word))
			return FISCAL_YEAR_LITERAL, digits, true
		}
	}

	// CY + 2-4 digits: calendar year
	if len(upper) >= 4 && len(upper) <= 6 && upper[:2] == "CY" {
		digits := upper[2:]
		if isAllDigits(digits) {
			l.pos = startPos + len([]rune(word))
			return CALENDAR_YEAR_LITERAL, digits, true
		}
	}

	return 0, "", false
}

func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// peekAlphanumWord returns the next word of letters and digits (no spaces) without advancing.
// Used for notation patterns like Q1, FQ3, FY2026, CY26.
func (l *Lexer) peekAlphanumWord() string {
	var word []rune
	pos := l.pos

	for pos < len(l.text) && (unicode.IsLetter(l.text[pos]) || unicode.IsDigit(l.text[pos])) {
		word = append(word, l.text[pos])
		pos++
	}

	return string(word)
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
	var hasExplicitDay bool

	// Try to read day (number)
	if unicode.IsDigit(l.currentChar()) {
		numStr := l.readNumberString()
		l.skipWhitespace()

		// Check if this number is a year (4 digits) or a day
		if len(numStr) == 4 {
			// It's a year: "January 2026" — no day number scanned.
			year = numStr
			day = "1" // Default to 1st
		} else {
			// It's a day: "January 15"
			day = numStr
			hasExplicitDay = true

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

	// Combine into value: "month:day:year:hasExplicitDay" where
	// the last field is "1" or "0". Parser splits by ":" and reads
	// parts[3] when present; legacy callers reading parts[0..2] are
	// unaffected.
	flag := "0"
	if hasExplicitDay {
		flag = "1"
	}
	value := month + ":" + day
	if year != "" {
		value += ":" + year
	} else {
		value += ":"
	}
	value += ":" + flag

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
		for range 3 {
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
