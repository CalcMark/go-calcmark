package lexer

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/CalcMark/go-calcmark/spec/units"
)

// BooleanKeywords defines the valid boolean keyword values (case-insensitive).
// This map is exported for grammar introspection.
var BooleanKeywords = map[string]bool{
	"true":  true,
	"false": true,
}

// ReservedKeywords defines reserved keywords (Go spec compliant + future control flow).
// This map is exported for grammar introspection.
// See: https://go.dev/ref/spec#Keywords
var ReservedKeywords = map[string]TokenType{
	// Logical operators (Go spec)
	"and": AND,
	"or":  OR,
	"not": NOT,

	// Future control flow keywords
	"if":     IF,
	"then":   THEN,
	"else":   ELSE,
	"elif":   ELIF,
	"end":    END,
	"for":    FOR,
	"in":     IN,
	"as":     AS,     // Conversion: "1234567 as napkin"
	"napkin": NAPKIN, // Human-readable formatting: "1234567 as napkin"
	"per":    PER,    // Rate expressions: "100 MB per second"
	"over":   OVER,   // Rate accumulation: "100 MB/s over 1 day"
	"with":   WITH,   // Capacity planning: "10000 req/s with 450 req/s capacity"
	// NOTE: "downtime" is NOT a reserved keyword - checked contextually in parser
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

// isValidThousandsSeparator checks if comma/underscore at current position is a valid thousands separator
// Returns true if followed by exactly 3 digits (and then non-digit or another separator)
func (l *Lexer) isValidThousandsSeparator(separatorChar rune) bool {
	if separatorChar != ',' && separatorChar != '_' {
		return false
	}

	// Must be followed by exactly 3 digits
	for i := 1; i <= 3; i++ {
		if !unicode.IsDigit(l.peek(i)) {
			return false
		}
	}

	// 4th character must not be a digit (unless it's another separator)
	fourthChar := l.peek(4)
	if unicode.IsDigit(fourthChar) {
		return false
	}

	return true
}

// readNumber reads a number token (supports commas and underscores as thousands separators)
// Handles multipliers (k, M, B, T) and percentages (%) and scientific notation (e, E)
func (l *Lexer) readNumber() Token {
	startLine := l.line
	startColumn := l.column
	startPos := l.pos
	var numStr strings.Builder

	// Read integer part
	for l.currentChar() != 0 {
		char := l.currentChar()
		if unicode.IsDigit(char) {
			numStr.WriteRune(char)
			l.advance()
		} else if (char == ',' || char == '_') && l.isValidThousandsSeparator(char) {
			// Keep separators in the value for now, or strip them?
			// BNF expects: _digit { _digit | '_' }
			// So we should keep underscores if we want to match BNF exactly?
			// The BNF `number` token allows underscores.
			// But `readNumber` usually strips them for the Value.
			// For gocc integration, we want the Value to match what the parser expects.
			// The parser's `expandNumberLiteral` handles the string.
			// Let's keep the raw text in OriginalText, and maybe normalized text in Value?
			// Actually, for the Scanner interface, we use Value as Lit.
			// So Value should probably be the raw text or close to it.
			// Let's include the separator in the string builder if it's an underscore.
			// If it's a comma, we might need to be careful as BNF doesn't support commas in numbers yet (we removed it).
			// Wait, we removed comma support from BNF!
			// So we should NOT consume commas here if we want to match BNF.
			// But the hand-written lexer supports commas.
			// If we consume commas, the parser will receive a token with commas.
			// If the BNF definition doesn't allow commas, the parser might fail validation?
			// No, the parser receives the token from the scanner. It doesn't re-validate the characters against the regex
			// UNLESS we are using the lexer generated by gocc.
			// But we are using -no_lexer!
			// So the parser just sees a token of type NUMBER.
			// It doesn't care what characters are in it, as long as the semantic action handles it.
			// So we CAN support commas in the hand-written lexer!
			// But we should probably strip them for the Value if the semantic action expects clean numbers.
			// The semantic action `expandNumberLiteral` likely uses `strconv.ParseFloat`.
			// `strconv` does NOT like commas.
			// So we should probably strip commas in the Value.
			l.advance() // Skip separator
		} else if char == '.' {
			// Decimal point - handled in next loop
			break
		} else {
			break
		}
	}

	// Read decimal part
	if l.currentChar() == '.' {
		numStr.WriteRune('.')
		l.advance()
		for l.currentChar() != 0 {
			char := l.currentChar()
			if unicode.IsDigit(char) {
				numStr.WriteRune(char)
				l.advance()
			} else {
				break
			}
		}
	}

	value := numStr.String()

	// Check for scientific notation (e/E)
	// Must be followed by optional sign and digits
	if l.currentChar() == 'e' || l.currentChar() == 'E' {
		// Peek ahead to ensure it's a valid scientific notation
		// e10, e+10, e-10
		isSci := false
		p1 := l.peek(1)
		if unicode.IsDigit(p1) {
			isSci = true
		} else if (p1 == '+' || p1 == '-') && unicode.IsDigit(l.peek(2)) {
			isSci = true
		}

		if isSci {
			numStr.WriteRune(l.currentChar())
			l.advance() // consume e/E

			if l.currentChar() == '+' || l.currentChar() == '-' {
				numStr.WriteRune(l.currentChar())
				l.advance()
			}

			for unicode.IsDigit(l.currentChar()) {
				numStr.WriteRune(l.currentChar())
				l.advance()
			}

			return Token{
				Type:         NUMBER_SCI,
				Value:        numStr.String(),
				OriginalText: string(l.text[startPos:l.pos]),
				Line:         startLine,
				Column:       startColumn,
				StartPos:     startPos,
				EndPos:       l.pos,
			}
		}
	}

	// Check for multipliers (no space)
	if l.pos < len(l.text) {
		char := l.currentChar()
		var tokenType TokenType = NUMBER // Default

		switch char {
		case '%':
			tokenType = NUMBER_PERCENT
			l.advance()
			value += "%"
		case 'k', 'K':
			tokenType = NUMBER_K
			l.advance()
			value += string(char)
		case 'M':
			tokenType = NUMBER_M
			l.advance()
			value += "M"
		case 'B':
			tokenType = NUMBER_B
			l.advance()
			value += "B"
		case 'T':
			tokenType = NUMBER_T
			l.advance()
			value += "T"
		}

		if tokenType != NUMBER {
			return Token{
				Type:         tokenType,
				Value:        value,
				OriginalText: string(l.text[startPos:l.pos]),
				Line:         startLine,
				Column:       startColumn,
				StartPos:     startPos,
				EndPos:       l.pos,
			}
		}
	}

	// Check for unit after number (kg, meters, USD, apples, widgets, etc.)
	if l.currentChar() == ' ' {
		savedPos := l.pos
		l.advance() // Skip space

		// Try to read a unit identifier
		if l.isIdentifierChar(l.currentChar(), true) {
			unit := l.readIdentifier()
			unitStr := string(unit.Value)

			// Don't treat reserved keywords as units (e.g., "5 and" should not be a quantity)
			if _, isReserved := ReservedKeywords[strings.ToLower(unitStr)]; isReserved {
				// This is a reserved keyword, not a unit - backtrack
				l.pos = savedPos
			} else if BooleanKeywords[strings.ToLower(unitStr)] {
				// Boolean keyword, not a unit - backtrack
				l.pos = savedPos
			} else {
				// Check for multi-word units: "1 nautical mile", "5 metric tons", "10 square meters"
				// Look ahead for a second identifier that might form a multi-word unit
				if l.currentChar() == ' ' {
					savedPos2 := l.pos
					l.advance() // Skip second space

					if l.isIdentifierChar(l.currentChar(), true) {
						secondUnit := l.readIdentifier()
						secondWord := string(secondUnit.Value)

						// Check if second word is also a reserved keyword
						if _, isReserved := ReservedKeywords[strings.ToLower(secondWord)]; isReserved {
							// Second word is reserved, backtrack
							l.pos = savedPos2
						} else if BooleanKeywords[strings.ToLower(secondWord)] {
							// Second word is boolean, backtrack
							l.pos = savedPos2
						} else {
							// Use canonical unit registry to check for multi-word units
							if combined := units.IsMultiWordUnit(unitStr, secondWord); combined != "" {
								// This is a valid multi-word unit, keep both words
								unitStr = combined
								isMultiWord := true

								// Check for third word (e.g., "meters per second", "kilometers per hour")
								if l.currentChar() == ' ' {
									savedPos3 := l.pos
									l.advance() // Skip third space

									if l.isIdentifierChar(l.currentChar(), true) {
										thirdUnit := l.readIdentifier()
										thirdWord := string(thirdUnit.Value)

										// Check if third word is also a reserved keyword
										if _, isReserved := ReservedKeywords[strings.ToLower(thirdWord)]; isReserved {
											// Third word is reserved, backtrack
											l.pos = savedPos3
										} else if BooleanKeywords[strings.ToLower(thirdWord)] {
											// Third word is boolean, backtrack
											l.pos = savedPos3
										} else {
											// Check if this forms a valid 3-word unit
											if combined3 := units.IsMultiWordUnit(unitStr, thirdWord); combined3 != "" {
												// Valid 3-word unit
												unitStr = combined3
											} else {
												// Not a 3-word unit, backtrack third word
												l.pos = savedPos3
											}
										}
									} else {
										l.pos = savedPos3
									}
								}

								_ = isMultiWord // Mark as used
							} else {
								// Not a multi-word unit, backtrack
								l.pos = savedPos2
							}
						}
					} else {
						// No second identifier, backtrack
						l.pos = savedPos2
					}
				}

				// Accept ANY identifier as a unit (for arbitrary units like "apples", "widgets")
				// This includes known units (kg, meters) and arbitrary units
				// Semantic validation will determine if units are compatible

				// Create QUANTITY token with "value:unit" format
				quantityValue := fmt.Sprintf("%s:%s", value, unitStr)
				return Token{
					Type:         QUANTITY,
					Value:        quantityValue,
					OriginalText: string(l.text[startPos:l.pos]),
					Line:         startLine,
					Column:       startColumn,
					StartPos:     startPos,
					EndPos:       l.pos,
				}
			}
		} else {
			l.pos = savedPos
		}
	}

	return Token{
		Type:         NUMBER,
		Value:        value,
		OriginalText: string(l.text[startPos:l.pos]),
		Line:         startLine,
		Column:       startColumn,
		StartPos:     startPos,
		EndPos:       l.pos,
	}
}

// readCurrency reads a currency symbol token (e.g., $)
// The number part will be read by the next call to Scan()
func (l *Lexer) readCurrency() (Token, error) {
	startLine := l.line
	startColumn := l.column
	startPos := l.pos

	// Read the symbol
	// symbol := l.currentChar() // Unused variable
	l.advance()

	return Token{
		Type:         CURRENCY_SYM,
		Value:        string(l.text[startPos:l.pos]),
		OriginalText: string(l.text[startPos:l.pos]),
		Line:         startLine,
		Column:       startColumn,
		StartPos:     startPos,
		EndPos:       l.pos,
	}, nil
}

// readCurrencyCodeQuantity reads a currency code token (e.g., USD)
// The number part will be read by the next call to Scan()
// Format: GBP100, USD1000, EUR50.25
// Currency code MUST be uppercase, followed immediately by digits (no space).
func (l *Lexer) readCurrencyCodeQuantity() Token {
	startLine := l.line
	startColumn := l.column
	startPos := l.pos

	// We already know the first 3 chars are uppercase letters from scanIdentifier check
	// But we need to verify and consume them

	// Read 3 uppercase letters
	var codeStr strings.Builder
	for i := 0; i < 3; i++ {
		codeStr.WriteRune(l.currentChar())
		l.advance()
	}
	code := codeStr.String()

	// Verify it's a valid currency code
	// If not valid, treat as identifier (this shouldn't happen since we pre-validate)
	if !isValidCurrencyCode(code) {
		return Token{
			Type:     IDENTIFIER,
			Value:    string(l.text[startPos:l.pos]),
			Line:     startLine,
			Column:   startColumn,
			StartPos: startPos,
			EndPos:   l.pos,
		}
	}

	// Now read the number part
	if !unicode.IsDigit(l.currentChar()) {
		// No number after currency code - treat as identifier
		return Token{
			Type:     IDENTIFIER,
			Value:    code,
			Line:     startLine,
			Column:   startColumn,
			StartPos: startPos,
			EndPos:   l.pos,
		}
	}

	// Read number (similar to readNumber but don't create a separate token)
	var numStr strings.Builder
	hasDecimal := false
	lastWasSeparator := false

	for l.currentChar() != 0 {
		char := l.currentChar()

		if unicode.IsDigit(char) {
			numStr.WriteRune(char)
			l.advance()
			lastWasSeparator = false
		} else if char == '.' && !hasDecimal {
			numStr.WriteRune(char)
			l.advance()
			hasDecimal = true
			lastWasSeparator = false
		} else if (char == ',' || char == '_') && !lastWasSeparator {
			// Thousands separator - validate later, skip for now
			l.advance()
			lastWasSeparator = true
		} else {
			break
		}
	}

	// Store as "value:unit" format (e.g., "100:GBP")
	value := fmt.Sprintf("%s:%s", numStr.String(), code)

	return Token{
		Type:         QUANTITY,
		Value:        value,
		OriginalText: string(l.text[startPos:l.pos]),
		Line:         startLine,
		Column:       startColumn,
		StartPos:     startPos,
		EndPos:       l.pos,
	}
}

// isIdentifierChar checks if a character can be part of an identifier.
// Follows Unicode-aware rules per ENCODING_SPEC.md:
// - Start: Letter (category L), underscore, or emoji
// - Continue: Start + Digit (category Nd) + Combining marks (category M)
func (l *Lexer) isIdentifierChar(char rune, isFirst bool) bool {
	// Whitespace terminates identifiers
	if char == ' ' || char == '\t' || char == '\r' || char == '\n' {
		return false
	}

	// Reserved operators and special characters
	if strings.ContainsRune("+-*×/=$><! %^(),", char) {
		return false
	}

	if isFirst {
		// Identifier start: Letter, underscore, or emoji
		return unicode.IsLetter(char) || char == '_' || isEmoji(char)
	}

	// Identifier continue: Start chars + Digit + Combining marks
	return unicode.IsLetter(char) ||
		unicode.IsDigit(char) ||
		unicode.IsMark(char) || // Combining marks (category M)
		char == '_' ||
		isEmoji(char)
}

// EmojiRange represents a Unicode emoji range
type EmojiRange struct {
	Start rune
	End   rune
	Name  string
}

// EmojiRanges defines the Unicode emoji ranges supported for identifiers.
// This list is exported for grammar introspection.
var EmojiRanges = []EmojiRange{
	{Start: 0x1F600, End: 0x1F64F, Name: "Emoticons"},
	{Start: 0x1F300, End: 0x1F5FF, Name: "Miscellaneous Symbols and Pictographs"},
	{Start: 0x1F680, End: 0x1F6FF, Name: "Transport and Map Symbols"},
	{Start: 0x1F900, End: 0x1F9FF, Name: "Supplemental Symbols and Pictographs"},
	{Start: 0x1FA00, End: 0x1FA6F, Name: "Symbols and Pictographs Extended-A"},
}

// isEmoji checks if a rune is an emoji character.
// Covers common emoji ranges sufficient for typical use cases.
// Does not handle: emoji modifiers, ZWJ sequences, or all Unicode emoji blocks.
// This is intentionally simple - extend EmojiRanges if more coverage is needed.
func isEmoji(r rune) bool {
	for _, emojiRange := range EmojiRanges {
		if r >= emojiRange.Start && r <= emojiRange.End {
			return true
		}
	}
	return false
}

// isValidCurrencyCode checks if a string matches the currency code PATTERN.
// Currency codes are syntactically defined as exactly 3 uppercase letters (A-Z).
// Semantic validation against ISO 4217 happens in the semantic checker, not the lexer.
// This allows users to write any 3-letter uppercase code, with validation deferred.
func isValidCurrencyCode(s string) bool {
	// Must be exactly 3 characters
	if len(s) != 3 {
		return false
	}

	// Must be all uppercase letters (A-Z)
	for _, r := range s {
		if !unicode.IsUpper(r) || !unicode.IsLetter(r) {
			return false
		}
	}

	return true
}

// readIdentifier reads an identifier (variable name)
// Identifiers support any Unicode characters including emoji and international characters
// NOTE: Spaces are NOT allowed in identifiers (this allows multi-token function names)
// SPECIAL: Checks for currency code prefix (3 uppercase letters) before reading full identifier
func (l *Lexer) readIdentifier() Token {
	startLine := l.line
	startColumn := l.column
	startPos := l.pos

	// SPECIAL CASE: Check if this might be a prefix currency code (3 uppercase letters + digit)
	// We need to check this BEFORE reading the full identifier because currency codes
	// followed by digits should be parsed as quantities, not identifiers
	if unicode.IsUpper(l.currentChar()) && unicode.IsLetter(l.currentChar()) {
		// Peek ahead to check for pattern: XXX123 (3 letters + digit)
		if unicode.IsUpper(l.peek(1)) && unicode.IsLetter(l.peek(1)) &&
			unicode.IsUpper(l.peek(2)) && unicode.IsLetter(l.peek(2)) &&
			unicode.IsDigit(l.peek(3)) {
			// Could be currency code - verify it's valid
			potentialCode := string([]rune{l.currentChar(), l.peek(1), l.peek(2)})
			if isValidCurrencyCode(potentialCode) {
				// It's a valid currency code followed by digits - parse as quantity
				return l.readCurrencyCodeQuantity()
			}
		}
	}

	// Normal identifier parsing
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
	endPos := l.pos

	// Check reserved keywords FIRST (including logical operators and function names)
	if tokenType, isReserved := ReservedKeywords[lowerIdent]; isReserved {
		return Token{
			Type:     tokenType,
			Value:    lowerIdent,
			Line:     startLine,
			Column:   startColumn,
			StartPos: startPos,
			EndPos:   endPos,
		}
	}

	// Check if identifier is a boolean keyword
	if BooleanKeywords[lowerIdent] {
		return Token{
			Type:     BOOLEAN,
			Value:    lowerIdent,
			Line:     startLine,
			Column:   startColumn,
			StartPos: startPos,
			EndPos:   endPos,
		}
	}

	// Check if standalone identifier is a currency code (3 uppercase letters)
	// Return as CURRENCY_CODE token - semantic validation happens later
	if isValidCurrencyCode(identStr) {
		return Token{
			Type:     CURRENCY_CODE,
			Value:    identStr,
			Line:     startLine,
			Column:   startColumn,
			StartPos: startPos,
			EndPos:   endPos,
		}
	}

	return Token{
		Type:     IDENTIFIER,
		Value:    identStr,
		Line:     startLine,
		Column:   startColumn,
		StartPos: startPos,
		EndPos:   endPos,
	}
}

// makeToken creates a token with position information
// Call this BEFORE advancing past the token
func (l *Lexer) makeToken(tokenType TokenType, value string, length int) Token {
	return Token{
		Type:     tokenType,
		Value:    value,
		Line:     l.line,
		Column:   l.column,
		StartPos: l.pos,
		EndPos:   l.pos + length,
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
			tokens = append(tokens, l.makeToken(NEWLINE, "\\n", 1))
			l.advance()
			continue
		}

		// Currency - support multiple currency symbols
		if char == '$' || char == '€' || char == '£' || char == '¥' {
			token, err := l.readCurrency()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token)
			continue
		}

		// Number
		if unicode.IsDigit(char) {
			// Check if this starts a duration: NUMBER + UNIT
			// Look ahead to see if followed by time unit
			savedPos := l.pos
			_ = l.readNumberString() // Read but don't use yet
			l.skipWhitespace()

			if _, ok := l.tryReadTimeUnit(); ok {
				// This is a duration literal
				l.pos = savedPos // Reset to start
				tokens = append(tokens, l.readDurationLiteral())
				continue
			}

			// Not a duration, just a regular number
			l.pos = savedPos
			tokens = append(tokens, l.readNumber())
			continue
		}

		// Identifier or date/duration keywords (check before operators)
		if l.isIdentifierChar(char, true) {
			// Try date keywords first (today, this week, etc.)
			startPos := l.pos
			if tokenType, ok := l.tryReadDateKeyword(); ok {
				endPos := l.pos
				keywordText := string(l.text[startPos:endPos])
				tokens = append(tokens, Token{
					Type:         tokenType,
					Value:        keywordText, // Store actual keyword text, not token type
					OriginalText: keywordText,
					Line:         l.line,
					Column:       l.column,
					StartPos:     startPos,
					EndPos:       endPos,
				})
				continue
			}

			// Try month names (for date literals)
			if _, ok := l.tryReadMonthName(); ok {
				// This is a date literal
				tokens = append(tokens, l.readDateLiteral())
				continue
			}

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
					tokens = append(tokens, l.makeToken(MULTIPLY, string(char), 1))
					l.advance()
					continue
				}
			}
			token := l.readIdentifier()

			// Error if IDENTIFIER token (not boolean or reserved keyword) is immediately followed by % (no whitespace)
			// Booleans and reserved keywords can be followed by % (it becomes modulus operator)
			if token.Type == IDENTIFIER && l.currentChar() == '%' {
				return nil, &LexerError{
					Message: fmt.Sprintf("Invalid syntax: '%%' cannot follow identifier '%s'", token.Value),
					Line:    l.line,
					Column:  l.column,
				}
			}

			tokens = append(tokens, token)
			continue
		}

		// Operators
		if char == '+' {
			tokens = append(tokens, l.makeToken(PLUS, "+", 1))
			l.advance()
			continue
		}

		if char == '-' {
			tokens = append(tokens, l.makeToken(MINUS, "-", 1))
			l.advance()
			continue
		}

		if char == '*' || char == '×' {
			// Check for ** (exponent)
			if char == '*' && l.peek(1) == '*' {
				tokens = append(tokens, l.makeToken(EXPONENT, "**", 2))
				l.advance()
				l.advance()
			} else {
				tokens = append(tokens, l.makeToken(MULTIPLY, string(char), 1))
				l.advance()
			}
			continue
		}

		if char == '/' {
			tokens = append(tokens, l.makeToken(DIVIDE, "/", 1))
			l.advance()
			continue
		}

		if char == '%' {
			tokens = append(tokens, l.makeToken(MODULUS, "%", 1))
			l.advance()
			continue
		}

		if char == '^' {
			tokens = append(tokens, l.makeToken(EXPONENT, "^", 1))
			l.advance()
			continue
		}

		// Comparison and assignment operators
		if char == '=' {
			// Check for ==
			if l.peek(1) == '=' {
				tokens = append(tokens, l.makeToken(EQUAL, "==", 2))
				l.advance()
				l.advance()
			} else {
				tokens = append(tokens, l.makeToken(ASSIGN, "=", 1))
				l.advance()
			}
			continue
		}

		if char == '>' {
			// Check for >=
			if l.peek(1) == '=' {
				tokens = append(tokens, l.makeToken(GREATER_EQUAL, ">=", 2))
				l.advance()
				l.advance()
			} else {
				tokens = append(tokens, l.makeToken(GREATER_THAN, ">", 1))
				l.advance()
			}
			continue
		}

		if char == '<' {
			// Check for <=
			if l.peek(1) == '=' {
				tokens = append(tokens, l.makeToken(LESS_EQUAL, "<=", 2))
				l.advance()
				l.advance()
			} else {
				tokens = append(tokens, l.makeToken(LESS_THAN, "<", 1))
				l.advance()
			}
			continue
		}

		if char == '!' {
			// Check for !=
			if l.peek(1) == '=' {
				tokens = append(tokens, l.makeToken(NOT_EQUAL, "!=", 2))
				l.advance()
				l.advance()
				continue
			}
			// Otherwise '!' alone is not a valid token, will fall through to error
		}

		// Parentheses
		if char == '(' {
			tokens = append(tokens, l.makeToken(LPAREN, "(", 1))
			l.advance()
			continue
		}

		if char == ')' {
			tokens = append(tokens, l.makeToken(RPAREN, ")", 1))
			l.advance()
			continue
		}

		// Comma (for function arguments)
		if char == ',' {
			tokens = append(tokens, l.makeToken(COMMA, ",", 1))
			l.advance()
			continue
		}

		// Octothorpe - not allowed mid-line in calculations
		if char == '#' {
			return nil, &LexerError{
				Message: "Inline octothorpe (#) is not supported in a calculation but is supported at the start of a line as a Markdown heading",
				Line:    l.line,
				Column:  l.column,
			}
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
//
//	"average" + "of" → FUNC_AVERAGE_OF
//	"square" + "root" + "of" → FUNC_SQUARE_ROOT_OF
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
					// Reconstruct original text from source tokens
					originalText := token.Value + " " + nextToken.Value
					result = append(result, Token{
						Type:         FUNC_AVERAGE_OF,
						Value:        "average of",
						OriginalText: originalText,
						Line:         token.Line,
						Column:       token.Column,
						StartPos:     token.StartPos,
						EndPos:       nextToken.EndPos,
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
					// Reconstruct original text from source tokens
					originalText := token.Value + " " + rootToken.Value + " " + ofToken.Value
					result = append(result, Token{
						Type:         FUNC_SQUARE_ROOT_OF,
						Value:        "square root of",
						OriginalText: originalText,
						Line:         token.Line,
						Column:       token.Column,
						StartPos:     token.StartPos,
						EndPos:       ofToken.EndPos,
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

// TokenizeOld scans the input string and returns a slice of tokens.
// Deprecated: Use Tokenize() from adapter.go instead.
func TokenizeOld(input string) ([]Token, error) {
	l := NewLexer(input)
	return l.Tokenize()
}
