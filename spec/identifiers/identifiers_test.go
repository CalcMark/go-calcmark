package identifiers

import (
	"slices"
	"testing"
)

func TestSlicesAreNonEmpty(t *testing.T) {
	sets := map[string][]string{
		"NetworkScopes":    NetworkScopes,
		"NetworkTypes":     NetworkTypes,
		"StorageTypes":     StorageTypes,
		"CompressionTypes": CompressionTypes,
	}
	for name, s := range sets {
		if len(s) == 0 {
			t.Errorf("%s is empty", name)
		}
	}
}

func TestSlicesHaveNoDuplicates(t *testing.T) {
	sets := map[string][]string{
		"NetworkScopes":    NetworkScopes,
		"NetworkTypes":     NetworkTypes,
		"StorageTypes":     StorageTypes,
		"CompressionTypes": CompressionTypes,
	}
	for name, s := range sets {
		seen := make(map[string]bool)
		for _, v := range s {
			if seen[v] {
				t.Errorf("%s contains duplicate: %s", name, v)
			}
			seen[v] = true
		}
	}
}

func TestStorageAliasesPointToCanonicalNames(t *testing.T) {
	for alias, canonical := range StorageAliases {
		if !slices.Contains(StorageTypes, canonical) {
			t.Errorf("StorageAliases[%q] = %q, but %q is not in StorageTypes", alias, canonical, canonical)
		}
	}
}

func TestStorageAliasesDoNotOverlapWithPrimary(t *testing.T) {
	for alias := range StorageAliases {
		if slices.Contains(StorageTypes, alias) {
			t.Errorf("StorageAliases key %q is also in StorageTypes (should be one or the other)", alias)
		}
	}
}

func TestAllStorageNamesIncludesAliases(t *testing.T) {
	all := AllStorageNames()

	// All primary names present
	for _, name := range StorageTypes {
		if !slices.Contains(all, name) {
			t.Errorf("AllStorageNames() missing primary name: %s", name)
		}
	}

	// All aliases present
	for alias := range StorageAliases {
		if !slices.Contains(all, alias) {
			t.Errorf("AllStorageNames() missing alias: %s", alias)
		}
	}

	// Length matches
	expected := len(StorageTypes) + len(StorageAliases)
	if len(all) != expected {
		t.Errorf("AllStorageNames() length = %d, want %d", len(all), expected)
	}
}

func TestJoinNames(t *testing.T) {
	got := JoinNames([]string{"a", "b", "c"})
	if got != "a, b, c" {
		t.Errorf("JoinNames = %q, want %q", got, "a, b, c")
	}
}

func TestOrderingIsDeterministic(t *testing.T) {
	// Verify slices return the same order on repeated access.
	// This matters because view_footer.go:144 displays Examples[0].
	for i := 0; i < 100; i++ {
		if NetworkScopes[0] != "local" {
			t.Fatal("NetworkScopes[0] is not deterministic")
		}
		if NetworkTypes[0] != "gigabit" {
			t.Fatal("NetworkTypes[0] is not deterministic")
		}
		if StorageTypes[0] != "ssd" {
			t.Fatal("StorageTypes[0] is not deterministic")
		}
		if CompressionTypes[0] != "gzip" {
			t.Fatal("CompressionTypes[0] is not deterministic")
		}
	}
}
