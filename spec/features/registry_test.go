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
		// NL function aliases (ellipsis patterns like "read...from" match by prefix)
		{"read", 1, "read"},
		{"compress", 1, "compress"},
		{"transfer", 1, "transfer_time"},
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
		Name: "meter",
		Aliases: []Alias{
			{Name: "meters", Parseable: true},
			{Name: "metre", Parseable: true},
			{Name: "metres", Parseable: true},
			{Name: "m", Parseable: true},
		},
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

func TestNLAliasesAreParseable(t *testing.T) {
	r := NewRegistry()

	// Functions with parseable NL aliases
	nlFunctions := map[string]string{
		"avg":           "average of",
		"sqrt":          "square root of",
		"read":          "read...from",
		"compress":      "compress...using",
		"transfer_time": "transfer...across",
	}

	for funcName, wantAlias := range nlFunctions {
		t.Run(funcName, func(t *testing.T) {
			results := r.Search(funcName)
			var feature *Feature
			for i := range results {
				if results[i].Name == funcName {
					feature = &results[i]
					break
				}
			}
			if feature == nil {
				t.Fatalf("Search(%q) did not find feature", funcName)
			}

			found := false
			for _, alias := range feature.Aliases {
				if alias.Name == wantAlias {
					found = true
					if !alias.Parseable {
						t.Errorf("Alias %q on %q should be Parseable", wantAlias, funcName)
					}
					break
				}
			}
			if !found {
				t.Errorf("Feature %q missing expected alias %q", funcName, wantAlias)
			}
		})
	}
}

func TestNonParseableAliases(t *testing.T) {
	r := NewRegistry()

	// Functions with search-only aliases (not parseable)
	tests := []struct {
		funcName  string
		aliasName string
	}{
		{"rtt", "round trip time"},
		{"requires", "capacity"},
	}

	for _, tt := range tests {
		t.Run(tt.funcName, func(t *testing.T) {
			results := r.Search(tt.funcName)
			var feature *Feature
			for i := range results {
				if results[i].Name == tt.funcName {
					feature = &results[i]
					break
				}
			}
			if feature == nil {
				t.Fatalf("Search(%q) did not find feature", tt.funcName)
			}

			for _, alias := range feature.Aliases {
				if alias.Name == tt.aliasName {
					if alias.Parseable {
						t.Errorf("Alias %q on %q should NOT be Parseable", tt.aliasName, tt.funcName)
					}
					return
				}
			}
			t.Errorf("Feature %q missing expected alias %q", tt.funcName, tt.aliasName)
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
