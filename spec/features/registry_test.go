package features

import (
	"strings"
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
		{CategoryKeyword, 6},     // in, as, of, per, over, as napkin, at
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

func TestNLTriggerKeywords(t *testing.T) {
	r := NewRegistry()
	keywords := r.NLTriggerKeywords()

	// All 6 NL function trigger keywords must be present.
	expected := []string{"compound", "compress", "depreciate", "grow", "read", "transfer"}
	if len(keywords) != len(expected) {
		t.Fatalf("NLTriggerKeywords() returned %d keywords, want %d: got %v", len(keywords), len(expected), keywords)
	}
	for i, want := range expected {
		if keywords[i] != want {
			t.Errorf("NLTriggerKeywords()[%d] = %q, want %q", i, keywords[i], want)
		}
	}
}

func TestParseableAliasesHaveExamples(t *testing.T) {
	r := NewRegistry()

	// Every parseable alias on a function should have a non-empty Example.
	functions := r.ByCategory(CategoryFunction)
	for _, f := range functions {
		for _, alias := range f.Aliases {
			if alias.Parseable && alias.Example == "" {
				t.Errorf("Function %q parseable alias %q has no Example", f.Name, alias.Name)
			}
		}
	}
}

func TestNLExamplesForFunctionsWithoutParseableAliases(t *testing.T) {
	r := NewRegistry()

	// Functions that use NL keywords (over, per, at) should have NLExample set.
	wantNLExample := map[string]string{
		"accumulate":   "100 MB/s over 1 day",
		"convert_rate": "5 MB/s per minute",
		"capacity":     "10 TB at 2 TB per disk",
		"downtime":     "99.9% downtime per month",
	}

	functions := r.ByCategory(CategoryFunction)
	for _, f := range functions {
		if want, ok := wantNLExample[f.Name]; ok {
			if f.NLExample != want {
				t.Errorf("Function %q NLExample = %q, want %q", f.Name, f.NLExample, want)
			}
		}
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
		{"capacity", "requires"},
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

func TestRegistry_FrontmatterCategory(t *testing.T) {
	r := NewRegistry()
	features := r.ByCategory(CategoryFrontmatter)

	if len(features) != 5 {
		t.Errorf("expected 5 frontmatter features, got %d", len(features))
	}

	expected := map[string]bool{
		"exchange":    false,
		"globals":     false,
		"scale":       false,
		"convert_to":  false,
		"measurement": false,
	}
	for _, f := range features {
		if _, ok := expected[f.Name]; !ok {
			t.Errorf("unexpected frontmatter feature %q", f.Name)
		}
		expected[f.Name] = true
	}
	for name, found := range expected {
		if !found {
			t.Errorf("missing frontmatter feature %q", name)
		}
	}

	// scale and convert_to descriptions should contain derived categories
	for _, f := range features {
		if f.Name == "scale" || f.Name == "convert_to" {
			if !strings.Contains(f.Description, "Length") {
				t.Errorf("%q description should contain 'Length'", f.Name)
			}
			if !strings.Contains(f.Description, "DataSize") {
				t.Errorf("%q description should contain 'DataSize'", f.Name)
			}
		}
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

func TestEveryFunctionHasParams(t *testing.T) {
	r := NewRegistry()
	functions := r.ByCategory(CategoryFunction)

	for _, f := range functions {
		if len(f.Params) == 0 {
			t.Errorf("Function %q has no Params — must have parameter specifications", f.Name)
		}
	}
}

func TestGetByName(t *testing.T) {
	r := NewRegistry()

	f := r.GetByName("avg")
	if f == nil {
		t.Fatal("GetByName(avg) returned nil")
	}
	if f.Name != "avg" {
		t.Errorf("GetByName(avg).Name = %q, want 'avg'", f.Name)
	}
	if len(f.Params) == 0 {
		t.Error("avg should have params")
	}
	if len(f.Synonyms) == 0 {
		t.Error("avg should have synonyms (average, mean)")
	}

	notFound := r.GetByName("nonexistent")
	if notFound != nil {
		t.Error("GetByName(nonexistent) should return nil")
	}
}
