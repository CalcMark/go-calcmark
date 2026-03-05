package interpreter

import (
	"slices"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/identifiers"
)

// TestNetworkMapsMatchCanonicalIdentifiers verifies bidirectional consistency:
// every canonical identifier has a map entry AND every map key is canonical.
func TestNetworkMapsMatchCanonicalIdentifiers(t *testing.T) {
	t.Run("networkLatencies", func(t *testing.T) {
		assertBidirectional(t, "networkLatencies", networkLatencies, identifiers.NetworkScopes)
	})

	t.Run("networkThroughput", func(t *testing.T) {
		assertBidirectional(t, "networkThroughput", networkThroughput, identifiers.NetworkTypes)
	})
}

// TestStorageMapsMatchCanonicalIdentifiers verifies storage maps cover all
// canonical types plus aliases, and contain no extra keys.
func TestStorageMapsMatchCanonicalIdentifiers(t *testing.T) {
	allNames := identifiers.AllStorageNames()

	t.Run("storageThroughput", func(t *testing.T) {
		assertBidirectional(t, "storageThroughput", storageThroughput, allNames)
	})

	t.Run("storageSeekTime", func(t *testing.T) {
		assertBidirectional(t, "storageSeekTime", storageSeekTime, allNames)
	})
}

// TestCompressionMapsMatchCanonicalIdentifiers verifies compression maps
// cover all canonical types and contain no extra keys.
func TestCompressionMapsMatchCanonicalIdentifiers(t *testing.T) {
	assertBidirectional(t, "compressionRatios", compressionRatios, identifiers.CompressionTypes)
}

// assertBidirectional checks that every canonical name has a map entry (forward)
// and every map key is a canonical name (reverse).
func assertBidirectional(t *testing.T, mapName string, m map[string]float64, canonical []string) {
	t.Helper()

	// Forward: every canonical identifier has a map entry
	for _, name := range canonical {
		if _, ok := m[name]; !ok {
			t.Errorf("%s missing canonical identifier: %s", mapName, name)
		}
	}

	// Reverse: every map key is a canonical identifier
	for key := range m {
		if !slices.Contains(canonical, key) {
			t.Errorf("%s has non-canonical key: %s", mapName, key)
		}
	}
}
