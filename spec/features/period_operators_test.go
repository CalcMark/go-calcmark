package features

import (
	"strings"
	"testing"
)

// U15 — v2.0 period operators registered in the feature catalog.
// LSP completion + help search both pull from the same Registry, so
// these tests pin discoverability for both surfaces.

func TestRegistry_PeriodOperatorDiscoverability(t *testing.T) {
	r := NewRegistry()

	// Each query should match the named entry by exact name.
	cases := []struct {
		query    string
		wantName string
	}{
		{"between", "between"},
		{"length of", "length of"},
		{"days in", "days in"},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			matches := r.Search(tc.query)
			if len(matches) == 0 {
				t.Fatalf("Search(%q) returned 0 matches; v2.0 operator should be discoverable", tc.query)
			}
			found := false
			for _, m := range matches {
				if m.Name == tc.wantName {
					found = true
					break
				}
			}
			if !found {
				names := make([]string, 0, len(matches))
				for _, m := range matches {
					names = append(names, m.Name)
				}
				t.Errorf("Search(%q) didn't return %q; got: %s",
					tc.query, tc.wantName, strings.Join(names, ", "))
			}
		})
	}
}

func TestRegistry_BetweenHasFromToAlias(t *testing.T) {
	r := NewRegistry()
	feat := r.GetByName("between")
	if feat == nil {
		t.Fatal("Registry missing `between` feature")
	}
	if len(feat.Aliases) == 0 {
		t.Fatal("`between` should have at least one alias (from A to B)")
	}
	found := false
	for _, a := range feat.Aliases {
		if strings.Contains(strings.ToLower(a.Name), "from") &&
			strings.Contains(strings.ToLower(a.Name), "to") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("`between` should alias `from A to B`; got aliases: %+v", feat.Aliases)
	}
}

func TestRegistry_LengthOfHasUsefulInsertText(t *testing.T) {
	r := NewRegistry()
	for _, name := range []string{"length of", "days in"} {
		t.Run(name, func(t *testing.T) {
			feat := r.GetByName(name)
			if feat == nil {
				t.Fatalf("Registry missing %q", name)
			}
			if feat.InsertText == "" {
				t.Errorf("%q has no InsertText; LSP completion needs a snippet placeholder", name)
			}
			if !strings.Contains(feat.InsertText, "${1:") {
				t.Errorf("%q InsertText (%q) should include a snippet placeholder ${1:...}",
					name, feat.InsertText)
			}
		})
	}
}
