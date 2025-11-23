package document

import (
	"strings"
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

	// Common markdown patterns = NOT calculations
	if strings.HasPrefix(trimmed, "#") { // Headers
		return false
	}
	if strings.HasPrefix(trimmed, "*") && !strings.HasPrefix(trimmed, "*=") { // Lists (but *= is calc)
		return false
	}
	if strings.HasPrefix(trimmed, "-") && !strings.HasPrefix(trimmed, "-=") && len(trimmed) > 1 && trimmed[1] == ' ' { // Lists
		return false
	}
	if strings.HasPrefix(trimmed, ">") { // Blockquotes
		return false
	}
	if strings.HasPrefix(trimmed, "[") && strings.Contains(trimmed, "](") { // Links
		return false
	}

	// Strong calculation indicators
	hasAssignment := strings.Contains(trimmed, "=") && !strings.HasPrefix(trimmed, "=")
	hasArithmetic := strings.ContainsAny(trimmed, "+-*/%")
	hasComparison := strings.Contains(trimmed, "==") || strings.Contains(trimmed, "!=") ||
		strings.Contains(trimmed, ">=") || strings.Contains(trimmed, "<=")
	hasNumber := strings.ContainsAny(trimmed, "0123456789")
	hasCurrency := strings.ContainsAny(trimmed, "$€£¥")
	hasFunctionCall := strings.Contains(trimmed, "(") && strings.Contains(trimmed, ")") && hasNumber

	// Clear calculation patterns
	if hasAssignment { // x = 10
		return true
	}
	if hasArithmetic && hasNumber { // 1 + 2
		return true
	}
	if hasComparison { // x == y
		return true
	}
	if hasCurrency { // $100
		return true
	}
	if hasFunctionCall { // avg(1, 2, 3)
		return true
	}

	// Default to text for ambiguous cases
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
