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

// Helper to normalize unit names for lookup
func normalized(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
