package units

import "slices"

// Categories returns the sorted list of unit categories derived from
// StandardUnits and the DataSize conversion category. This is the single
// source of truth — frontmatter validation uses this instead of a
// hardcoded map.
func Categories() []string {
	seen := make(map[string]bool)
	for _, u := range StandardUnits {
		seen[u.Quantity] = true
	}
	seen[CategoryDataSize] = true

	cats := make([]string, 0, len(seen))
	for c := range seen {
		cats = append(cats, c)
	}
	slices.Sort(cats)
	return cats
}
