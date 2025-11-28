package features

import (
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}

	all := r.All()
	if len(all) == 0 {
		t.Error("Registry should have features")
	}
	t.Logf("Registry contains %d features", len(all))
}

func TestRegistrySearch(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		query    string
		wantMin  int    // minimum expected matches
		wantName string // at least one match should have this name
	}{
		{"avg", 1, "avg"},
		{"meter", 1, "meter"},
		{"gzip", 1, "gzip"},
		{"today", 1, "today"},
		{"ssd", 1, "ssd"},
		{"gigabit", 1, "gigabit"},
		{"nonexistent", 0, ""},
		{"", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			results := r.Search(tt.query)
			if len(results) < tt.wantMin {
				t.Errorf("Search(%q) got %d results, want at least %d", tt.query, len(results), tt.wantMin)
			}
			if tt.wantName != "" {
				found := false
				for _, f := range results {
					if f.Name == tt.wantName {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Search(%q) should include %q", tt.query, tt.wantName)
				}
			}
		})
	}
}

func TestRegistryByCategory(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		cat     Category
		wantMin int
	}{
		{CategoryFunction, 10},   // We defined 12 functions
		{CategoryUnit, 30},       // Many units from canonical.go
		{CategoryDate, 5},        // today, tomorrow, yesterday, etc.
		{CategoryNetwork, 5},     // local, regional, gigabit, etc.
		{CategoryStorage, 3},     // ssd, nvme, hdd
		{CategoryCompression, 4}, // gzip, lz4, zstd, bzip2
		{CategoryKeyword, 3},     // in, as, of
		{CategoryOperator, 5},    // +, -, *, /, ^
	}

	for _, tt := range tests {
		t.Run(string(tt.cat), func(t *testing.T) {
			results := r.ByCategory(tt.cat)
			if len(results) < tt.wantMin {
				t.Errorf("ByCategory(%q) got %d results, want at least %d", tt.cat, len(results), tt.wantMin)
			}
			// Verify all results have the correct category
			for _, f := range results {
				if f.Category != tt.cat {
					t.Errorf("Feature %q has category %q, want %q", f.Name, f.Category, tt.cat)
				}
			}
		})
	}
}

func TestFeatureMatch(t *testing.T) {
	f := Feature{
		Name:    "meter",
		Aliases: []string{"meters", "metre", "metres", "m"},
	}

	tests := []struct {
		query string
		want  bool
	}{
		{"met", true},
		{"meter", true},
		{"meters", true},
		{"metr", true}, // matches "metres"
		{"m", true},    // matches alias "m"
		{"foot", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := f.Match(tt.query)
			if got != tt.want {
				t.Errorf("Match(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestRegistryCategories(t *testing.T) {
	r := NewRegistry()
	cats := r.Categories()

	if len(cats) < 5 {
		t.Errorf("Expected at least 5 categories, got %d", len(cats))
	}

	// Check that categories are sorted
	for i := 1; i < len(cats); i++ {
		if cats[i] < cats[i-1] {
			t.Error("Categories should be sorted")
		}
	}
}
