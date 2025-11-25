package document

import (
	"strings"

	"github.com/CalcMark/go-calcmark/spec/units"
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
			isCalc := d.isCalculation(line)

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

// isCalculation checks if a line is a calculation.
// Uses heuristics since the parser is currently too permissive.
// Default: when ambiguous, treat as text (markdown).
func (d *Detector) isCalculation(line string) bool {
	trimmed := strings.TrimSpace(line)

	if trimmed == "" {
		return false
	}

	// First, check for markdown patterns (explicit text indicators)
	if isMarkdownPattern(trimmed) {
		return false
	}

	// Then, check for calculation patterns
	return isCalculationPattern(trimmed)
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

	return false
}

// isCalculationPattern checks if a line matches calculation patterns.
// Returns true if the line appears to be a calculation based on heuristics.
func isCalculationPattern(line string) bool {
	// Check for strong calculation indicators
	if hasAssignmentOperator(line) {
		return true
	}

	if hasArithmeticExpression(line) {
		return true
	}

	if hasComparisonOperator(line) {
		return true
	}

	if hasCurrencySymbol(line) {
		return true
	}

	if hasFunctionCall(line) {
		return true
	}

	if hasUnitExpression(line) {
		return true
	}

	return false
}

// hasAssignmentOperator checks if line contains an assignment operator.
// Examples: "x = 10", "price = 100 USD"
func hasAssignmentOperator(line string) bool {
	return strings.Contains(line, "=") && !strings.HasPrefix(line, "=")
}

// hasArithmeticExpression checks if line contains arithmetic with numbers.
// Examples: "1 + 2", "5 * 3", "10 / 2"
func hasArithmeticExpression(line string) bool {
	hasArithmetic := strings.ContainsAny(line, "+-*/%")
	hasNumber := strings.ContainsAny(line, "0123456789")
	return hasArithmetic && hasNumber
}

// hasComparisonOperator checks if line contains comparison operators.
// Examples: "x == y", "a != b", "value >= 10"
func hasComparisonOperator(line string) bool {
	return strings.Contains(line, "==") ||
		strings.Contains(line, "!=") ||
		strings.Contains(line, ">=") ||
		strings.Contains(line, "<=")
}

// hasCurrencySymbol checks if line contains currency symbols.
// Examples: "$100", "€50", "£25"
func hasCurrencySymbol(line string) bool {
	return strings.ContainsAny(line, "$€£¥")
}

// hasFunctionCall checks if line appears to be a function call with numbers.
// Examples: "avg(1, 2, 3)", "sqrt(16)"
func hasFunctionCall(line string) bool {
	hasParens := strings.Contains(line, "(") && strings.Contains(line, ")")
	hasNumber := strings.ContainsAny(line, "0123456789")
	return hasParens && hasNumber
}

// hasUnitExpression checks if line contains unit expressions.
// Examples: "10 meters", "5 kg", "100 celsius in fahrenheit"
func hasUnitExpression(line string) bool {
	hasNumber := strings.ContainsAny(line, "0123456789")
	if !hasNumber {
		return false
	}

	// Check for unit conversion keyword
	if strings.Contains(line, " in ") {
		return true
	}

	// Check for known units using canonical unit registry
	// Extract potential unit words from the line
	words := strings.Fields(line)
	for _, word := range words {
		// Clean up punctuation
		cleaned := strings.TrimRight(word, ".,;:!?")
		lower := strings.ToLower(cleaned)

		// Check if this word is a known unit
		if _, exists := units.StandardUnits[lower]; exists {
			return true
		}

		// Also check against all aliases
		for _, unitMapping := range units.StandardUnits {
			for _, alias := range unitMapping.Aliases {
				if lower == strings.ToLower(alias) {
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
