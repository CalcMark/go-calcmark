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

func Categories() []string {
	seen := make(map[string]bool)
	for _, u := range StandardUnits {
		seen[u.Quantity] = true
	}
	seen[CategoryDataSize] = true
	seen[CategoryCurrency] = true

	cats := make([]string, 0, len(seen))
	for c := range seen {
		cats = append(cats, c)
	}
	slices.Sort(cats)
	return cats
}
