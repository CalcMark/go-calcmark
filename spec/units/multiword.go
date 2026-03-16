package units

import "strings"

// IsMultiWordUnit checks if the combination of two words is a known multi-word unit.
// Returns the combined form if found (preserving user's spelling), empty string otherwise.
// Examples: ("nautical", "mile") -> "nautical mile"
//
//	("nautical", "miles") -> "nautical miles"
//	("metric", "ton") -> "metric ton"
func IsMultiWordUnit(word1, word2 string) string {
	// Combine (preserving case for output)
	combined := word1 + " " + word2
	normalizedCombined := normalized(combined)

	// Check if the combined form exists as a map key
	if _, exists := StandardUnits[normalizedCombined]; exists {
		return combined // Return user's input, not canonical
	}

	// Check if the combined form matches any alias
	for _, unit := range StandardUnits {
		for _, alias := range unit.Aliases {
			if normalized(alias) == normalizedCombined {
				return combined // Return user's input
			}
		}
	}

	return ""
}

// IsHyphenatedUnit checks if a word followed by a hyphen and another word
// forms a known hyphenated unit. Returns the combined form if found.
// Examples: ("pound", "force") -> "pound-force"
//
//	("newton", "second") -> "newton-second"
//	("kilowatt", "hour") -> "kilowatt-hour"
func IsHyphenatedUnit(word1, word2 string) string {
	combined := word1 + "-" + word2
	normalizedCombined := normalized(combined)

	// Check if the combined form exists as a map key
	if _, exists := StandardUnits[normalizedCombined]; exists {
		return combined
	}

	// Check if the combined form matches any alias
	for _, unit := range StandardUnits {
		for _, alias := range unit.Aliases {
			if normalized(alias) == normalizedCombined {
				return combined
			}
		}
	}

	// Also check the conversion registry for units not in StandardUnits
	// (e.g., "kilowatt-hour" is registered in conversion.go aliases)
	if _, ok := conversionRegistry[normalizedCombined]; ok {
		return combined
	}

	return ""
}

// IsHyphenatedTripleUnit checks for 3-part hyphenated units.
// Examples: ("pound-force", "second") -> "pound-force-second"
func IsHyphenatedTripleUnit(prefix, word3 string) string {
	combined := prefix + "-" + word3
	normalizedCombined := normalized(combined)

	if _, exists := StandardUnits[normalizedCombined]; exists {
		return combined
	}

	for _, unit := range StandardUnits {
		for _, alias := range unit.Aliases {
			if normalized(alias) == normalizedCombined {
				return combined
			}
		}
	}

	if _, ok := conversionRegistry[normalizedCombined]; ok {
		return combined
	}

	return ""
}

// Helper to normalize unit names for lookup
func normalized(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
