package parser

// isAllUppercase checks if a string contains only uppercase letters
// Performance: O(n) where n = string length (typically 3 for currency codes)
func isAllUppercase(s string) bool {
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return len(s) > 0
}

// isCurrency checks if a unit string is a currency code or symbol
func isCurrency(unit string) bool {
	// Common currency symbols
	symbols := map[string]bool{
		"$": true, "€": true, "£": true, "¥": true,
	}

	if symbols[unit] {
		return true
	}

	// ISO 4217 currency codes are 3 uppercase letters
	if len(unit) == 3 {
		return isAllUppercase(unit)
	}

	return false
}
