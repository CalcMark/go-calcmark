package document

import (
	"strings"

	"github.com/CalcMark/go-calcmark/spec/lexer"
)

// Detector analyzes source text and splits it into blocks.
type Detector struct {
}

// NewDetector creates a new block detector.
func NewDetector() *Detector {
	return &Detector{}
}

// DetectBlocks splits source into blocks using these rules:
// - 2 consecutive empty lines = block boundary
// - 1 empty line = part of current block
// - Calculations vs text determined by parsing each line
//
// Unicode-aware: handles all line terminators (LF, CRLF, CR, U+2028, U+2029).
func (d *Detector) DetectBlocks(source string) ([]Block, error) {
	lines := splitLines(source) // Unicode-aware line splitting
	blocks := []Block{}

	currentBlockLines := []string{}
	currentBlockType := BlockText // Default to text
	emptyLineCount := 0

	for _, line := range lines {
		isEmpty := isEmptyLine(line) // Unicode-aware empty check

		if isEmpty {
			emptyLineCount++

			// 2 consecutive empty lines = block boundary
			if emptyLineCount >= 2 {
				// Flush current block (if not empty)
				if len(currentBlockLines) > 0 && !allEmpty(currentBlockLines) {
					blocks = append(blocks, d.createBlock(currentBlockType, currentBlockLines))
					currentBlockLines = []string{}
				}

				// Reset for next block
				emptyLineCount = 0
				// Next non-empty line determines block type
				continue
			}

			// 1 empty line - include in current block
			currentBlockLines = append(currentBlockLines, line)

		} else {
			// Non-empty line
			emptyLineCount = 0

			// Determine if this line is a calculation
			isCalc, err := d.isCalculation(line)
			if err != nil {
				// Lexer error on calc-like line - propagate immediately
				return nil, err
			}

			// If first line of new block, set type
			if len(currentBlockLines) == 0 {
				currentBlockType = BlockText
				if isCalc {
					currentBlockType = BlockCalculation
				}
			} else {
				// Check if block type changes
				expectedType := BlockText
				if isCalc {
					expectedType = BlockCalculation
				}

				// If type changes, start new block
				if expectedType != currentBlockType {
					// Flush current block
					blocks = append(blocks, d.createBlock(currentBlockType, currentBlockLines))
					currentBlockLines = []string{}
					currentBlockType = expectedType
				}
			}

			currentBlockLines = append(currentBlockLines, line)
		}
	}

	// Flush remaining block (if not empty)
	if len(currentBlockLines) > 0 && !allEmpty(currentBlockLines) {
		blocks = append(blocks, d.createBlock(currentBlockType, currentBlockLines))
	}

	return blocks, nil
}

// allEmpty checks if all lines in a slice are empty.
func allEmpty(lines []string) bool {
	for _, line := range lines {
		if !isEmptyLine(line) {
			return false
		}
	}
	return true
}

// isCalculation checks if a line is a valid calculation.
// The approach: if a line parses successfully as a calculation, it's a calculation.
// If it fails to parse, it's text (markdown).
//
// Returns (true, nil) for valid calculation lines.
// Returns (false, nil) for text lines (including invalid syntax - treated as markdown).
func (d *Detector) isCalculation(line string) (bool, error) {
	trimmed := strings.TrimSpace(line)

	if trimmed == "" {
		return false, nil
	}

	// Explicit markdown patterns are never calculations
	if isMarkdownPattern(trimmed) {
		return false, nil
	}

	// Try to tokenize the line
	lex := lexer.NewLexer(trimmed)
	tokens, err := lex.Tokenize()

	if err != nil {
		// Tokenization failed - this is text, not a calculation
		// Don't propagate the error; just treat it as markdown
		return false, nil
	}

	// Successfully tokenized - check if it looks like a meaningful calculation
	// An empty or trivial token stream isn't a calculation
	if len(tokens) == 0 {
		return false, nil
	}

	// Filter out NEWLINE tokens for analysis
	meaningfulTokens := filterNonNewlineTokens(tokens)
	if len(meaningfulTokens) == 0 {
		return false, nil
	}

	// A valid calculation must have recognizable structure
	return looksLikeCalculation(meaningfulTokens), nil
}

// filterNonNewlineTokens returns tokens excluding NEWLINE.
// Pure function: no side effects.
func filterNonNewlineTokens(tokens []lexer.Token) []lexer.Token {
	result := make([]lexer.Token, 0, len(tokens))
	for _, t := range tokens {
		if t.Type != lexer.NEWLINE {
			result = append(result, t)
		}
	}
	return result
}

// looksLikeCalculation checks if tokens represent a calculation structure.
// Pure function: deterministic, no side effects.
func looksLikeCalculation(tokens []lexer.Token) bool {
	if len(tokens) == 0 {
		return false
	}

	first := tokens[0]

	// Assignment: identifier = ...
	if first.Type == lexer.IDENTIFIER && len(tokens) >= 2 && tokens[1].Type == lexer.ASSIGN {
		return true
	}

	// Expression starting with number (including multiplier suffixes like 5K, 3M)
	if isNumberToken(first.Type) {
		return true
	}

	// Expression starting with quantity (e.g., "10 meters")
	if first.Type == lexer.QUANTITY {
		return true
	}

	// Expression starting with currency symbol or currency token
	if first.Type == lexer.CURRENCY || first.Type == lexer.CURRENCY_SYM {
		return true
	}

	// Expression starting with paren
	if first.Type == lexer.LPAREN {
		return true
	}

	// Boolean literal
	if first.Type == lexer.BOOLEAN {
		return true
	}

	// Unary operators (not, -)
	if first.Type == lexer.NOT || first.Type == lexer.MINUS {
		return true
	}

	// Date literals and keywords
	if isDateToken(first.Type) {
		return true
	}

	// Identifier alone or followed by operator/function call = calculation
	// But multiple consecutive identifiers = prose (like "More text")
	if first.Type == lexer.IDENTIFIER {
		// Single identifier is a variable reference
		if len(tokens) == 1 {
			return true
		}
		// Identifier followed by operator or paren (function call) = calculation
		second := tokens[1]
		if isOperatorToken(second.Type) || second.Type == lexer.LPAREN {
			return true
		}
		// Identifier followed by another identifier = likely prose
		if second.Type == lexer.IDENTIFIER {
			return false
		}
		// Identifier followed by keyword (like "in", "as") = calculation
		if isKeywordToken(second.Type) {
			return true
		}
		return true // Default: treat as calculation
	}

	return false
}

// isNumberToken checks if a token type is a number variant.
// Pure function.
func isNumberToken(t lexer.TokenType) bool {
	switch t {
	case lexer.NUMBER, lexer.NUMBER_PERCENT, lexer.NUMBER_K,
		lexer.NUMBER_M, lexer.NUMBER_B, lexer.NUMBER_T, lexer.NUMBER_SCI:
		return true
	}
	return false
}

// isDateToken checks if a token type is a date-related token.
// Pure function.
func isDateToken(t lexer.TokenType) bool {
	switch t {
	case lexer.DATE_TODAY, lexer.DATE_TOMORROW, lexer.DATE_YESTERDAY,
		lexer.DATE_THIS_WEEK, lexer.DATE_THIS_MONTH, lexer.DATE_THIS_YEAR,
		lexer.DATE_NEXT_WEEK, lexer.DATE_NEXT_MONTH, lexer.DATE_NEXT_YEAR,
		lexer.DATE_LAST_WEEK, lexer.DATE_LAST_MONTH, lexer.DATE_LAST_YEAR,
		lexer.DATE_LITERAL, lexer.DURATION_LITERAL:
		return true
	}
	return false
}

// isOperatorToken checks if a token type is an operator.
// Pure function.
func isOperatorToken(t lexer.TokenType) bool {
	switch t {
	case lexer.PLUS, lexer.MINUS, lexer.MULTIPLY, lexer.DIVIDE,
		lexer.MODULUS, lexer.EXPONENT, lexer.ASSIGN,
		lexer.GREATER_THAN, lexer.LESS_THAN, lexer.GREATER_EQUAL,
		lexer.LESS_EQUAL, lexer.EQUAL, lexer.NOT_EQUAL,
		lexer.AND, lexer.OR:
		return true
	}
	return false
}

// isKeywordToken checks if a token type is a CalcMark keyword.
// Pure function.
func isKeywordToken(t lexer.TokenType) bool {
	switch t {
	case lexer.AS, lexer.FROM, lexer.IN, lexer.OF, lexer.PER, lexer.OVER, lexer.WITH:
		return true
	}
	return false
}

// isMarkdownPattern checks if a line matches common markdown patterns.
// These patterns explicitly indicate the line is NOT a calculation.
func isMarkdownPattern(line string) bool {
	// Headers
	if strings.HasPrefix(line, "#") {
		return true
	}

	// Lists (but *= and -= are calculations)
	if strings.HasPrefix(line, "*") && !strings.HasPrefix(line, "*=") {
		return true
	}
	if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "-=") &&
		len(line) > 1 && line[1] == ' ' {
		return true
	}

	// Blockquotes
	if strings.HasPrefix(line, ">") {
		return true
	}

	// Links
	if strings.HasPrefix(line, "[") && strings.Contains(line, "](") {
		return true
	}

	// Inline bold/italic: **text** or *text* (not surrounded by spaces like " * ")
	// This catches markdown formatting in prose
	if hasInlineMarkdownFormatting(line) {
		return true
	}

	return false
}

// hasInlineMarkdownFormatting detects **bold** and *italic* markdown patterns.
// These are NOT arithmetic operators when immediately adjacent to word characters.
func hasInlineMarkdownFormatting(line string) bool {
	// Look for **text** pattern (bold)
	// The key difference from power operator: ** immediately followed by non-space
	for i := 0; i < len(line)-2; i++ {
		if line[i] == '*' && line[i+1] == '*' {
			// Check if this looks like bold (not power operator)
			// Bold: **word (no space after **)
			// Power: x ** y (spaces around **)
			if i+2 < len(line) && line[i+2] != ' ' && line[i+2] != '*' {
				// Check for closing **
				closeIdx := strings.Index(line[i+2:], "**")
				if closeIdx > 0 {
					return true
				}
			}
		}
	}
	return false
}

// createBlock creates the appropriate block type.
func (d *Detector) createBlock(blockType BlockType, lines []string) Block {
	switch blockType {
	case BlockCalculation:
		return NewCalcBlock(lines)
	case BlockText:
		return NewTextBlock(lines)
	default:
		return NewTextBlock(lines)
	}
}
