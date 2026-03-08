package units

import "slices"

// Categories returns the sorted list of unit categories derived from
// StandardUnits and the DataSize conversion category. This is the single
// source of truth — frontmatter validation uses this instead of a
// hardcoded map.
// CategoryCurrency is the category name for currency values ($, €, £, ¥).
// Currency is NOT derived from StandardUnits — it's a standalone type
// that can opt in to scaling via unit_categories: [Currency].
const CategoryCurrency = "Currency"

// CategoryNumber is the category name for plain numeric values (no unit).
// Like Currency, Number is not derived from StandardUnits and must be
// explicitly listed in unit_categories to participate in scaling.
const CategoryNumber = "Number"

// CategoryCustom is the category name for custom units (e.g., "bananas", "eggs").
// Custom units are not in StandardUnits and have no system mapping.
// They must be explicitly listed in unit_categories to participate in scaling.
const CategoryCustom = "Custom"

func Categories() []string {
	seen := make(map[string]bool)
	for _, u := range StandardUnits {
		seen[u.Quantity] = true
	}
	seen[CategoryDataSize] = true
	seen[CategoryCurrency] = true
	seen[CategoryNumber] = true
	seen[CategoryCustom] = true

	cats := make([]string, 0, len(seen))
	for c := range seen {
		cats = append(cats, c)
	}
	slices.Sort(cats)
	return cats
}
