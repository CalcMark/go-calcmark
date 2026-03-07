package units

import (
	"slices"
	"testing"
)

func TestCategories_MatchesStandardUnits(t *testing.T) {
	cats := Categories()
	if len(cats) == 0 {
		t.Fatal("Categories() returned empty list")
	}

	// Every Quantity in StandardUnits must appear in Categories()
	for _, u := range StandardUnits {
		if !slices.Contains(cats, u.Quantity) {
			t.Errorf("Quantity %q from StandardUnits not in Categories()", u.Quantity)
		}
	}

	// CategoryDataSize must be included
	if !slices.Contains(cats, CategoryDataSize) {
		t.Errorf("CategoryDataSize (%q) not in Categories()", CategoryDataSize)
	}

	// Spot-check expected categories
	expected := []string{"Length", "Mass", "Volume", "Temperature", "Speed", "Energy", "Power", "Area", "DataSize"}
	for _, exp := range expected {
		if !slices.Contains(cats, exp) {
			t.Errorf("expected category %q not in Categories()", exp)
		}
	}
}

func TestCategories_Sorted(t *testing.T) {
	cats := Categories()
	if !slices.IsSorted(cats) {
		t.Errorf("Categories() not sorted: %v", cats)
	}
}

func TestCategories_NoDuplicates(t *testing.T) {
	cats := Categories()
	seen := make(map[string]bool)
	for _, c := range cats {
		if seen[c] {
			t.Errorf("duplicate category %q in Categories()", c)
		}
		seen[c] = true
	}
}
