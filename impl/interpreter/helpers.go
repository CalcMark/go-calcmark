package interpreter

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// Pure helper functions for parsing CalcMark values.
// These are stateless and easily testable.

// expandNumberLiteral expands number literals with multiplier suffixes.
// Examples: "1k" -> 1000, "1.2M" -> 1200000, "1.5e10" -> 15000000000, "20%" -> 0.20
func expandNumberLiteral(s string) (decimal.Decimal, error) {
	if len(s) == 0 {
		return decimal.Zero, fmt.Errorf("empty number literal")
	}

	// Scientific notation (e.g., 1.2e10)
	if strings.ContainsAny(s, "eE") {
		return decimal.NewFromString(s)
	}

	// Check for percentage (e.g., 20% -> 0.20)
	if strings.HasSuffix(s, "%") {
		baseStr := s[:len(s)-1]
		base, err := decimal.NewFromString(baseStr)
		if err != nil {
			return decimal.Zero, err
		}
		// Convert percentage to decimal: 20% = 0.20
		return base.Div(decimal.NewFromInt(100)), nil
	}

	// Check for multiplier suffix
	lastChar := s[len(s)-1]
	var multiplier decimal.Decimal
	var baseStr string

	switch lastChar {
	case 'k', 'K':
		multiplier = decimal.NewFromInt(1000)
		baseStr = s[:len(s)-1]
	case 'M':
		multiplier = decimal.NewFromInt(1000000)
		baseStr = s[:len(s)-1]
	case 'B':
		multiplier = decimal.NewFromInt(1000000000)
		baseStr = s[:len(s)-1]
	case 'T':
		multiplier = decimal.NewFromInt(1000000000000)
		baseStr = s[:len(s)-1]
	default:
		// No multiplier, parse as-is
		return decimal.NewFromString(s)
	}

	// Parse base and multiply
	base, err := decimal.NewFromString(baseStr)
	if err != nil {
		return decimal.Zero, err
	}

	return base.Mul(multiplier), nil
}

// parseMonth converts a month string to month number (1-12).
// Accepts both 3-letter abbreviations and full month names.
func parseMonth(monthStr string) (int, error) {
	normalized := strings.ToLower(monthStr)

	months := map[string]int{
		"jan": 1, "january": 1,
		"feb": 2, "february": 2,
		"mar": 3, "march": 3,
		"apr": 4, "april": 4,
		"may": 5,
		"jun": 6, "june": 6,
		"jul": 7, "july": 7,
		"aug": 8, "august": 8,
		"sep": 9, "september": 9,
		"oct": 10, "october": 10,
		"nov": 11, "november": 11,
		"dec": 12, "december": 12,
	}

	month, ok := months[normalized]
	if !ok {
		return 0, fmt.Errorf("invalid month: %q", monthStr)
	}

	return month, nil
}

// parseInt parses an integer string.
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// parseBooleanValue converts boolean keywords to bool.
// Accepts: true/false (case-insensitive).
func parseBooleanValue(s string) (bool, error) {
	normalized := strings.ToLower(s)

	switch normalized {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %q", s)
	}
}

// isBooleanKeyword checks if a string is a boolean keyword.
func isBooleanKeyword(s string) bool {
	normalized := strings.ToLower(s)
	switch normalized {
	case "true", "false":
		return true
	default:
		return false
	}
}
