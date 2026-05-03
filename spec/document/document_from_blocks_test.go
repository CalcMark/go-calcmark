package document

import (
	"slices"
	"testing"
)

// NewDocumentFromBlocks is the parser-front-end-friendly Document
// constructor: takes pre-built blocks (calc and/or text) plus an
// optional frontmatter, wraps each block in a BlockNode with a
// fresh UUID, and rebuilds the dependency graph. Callers are
// responsible for the segmentation decision; this helper just
// stitches their decision into a real *Document the evaluator can
// consume.
//
// Used by impl/embedded.BuildDocument (Embedded-mode parsing) so
// the same Document construction logic flows through both the
// existing flat-CM NewDocument path and the new fence-aware path.
// Centralizing avoids drift in BlockNode wiring or dependency-
// graph initialization across the two front-ends.

func TestNewDocumentFromBlocks_ZeroBlocks_ReturnsEmptyDocument(t *testing.T) {
	doc := NewDocumentFromBlocks(nil, nil)
	if doc == nil {
		t.Fatal("expected non-nil *Document, got nil")
	}
	if len(doc.GetBlocks()) != 0 {
		t.Errorf("expected 0 blocks, got %d", len(doc.GetBlocks()))
	}
	if doc.GetFrontmatter() != nil {
		t.Errorf("expected nil frontmatter, got %v", doc.GetFrontmatter())
	}
}

func TestNewDocumentFromBlocks_SingleCalcBlock_WrapsWithUUID(t *testing.T) {
	calc := NewCalcBlock([]string{"price = 100"})
	doc := NewDocumentFromBlocks(nil, []Block{calc})
	blocks := doc.GetBlocks()
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Block != calc {
		t.Errorf("expected the same calc block, got different instance")
	}
	if blocks[0].ID == "" {
		t.Error("expected BlockNode.ID to be populated with a UUID")
	}
}

func TestNewDocumentFromBlocks_MixedKinds_PreservesOrder(t *testing.T) {
	text1 := NewTextBlock([]string{"# heading"})
	calc := NewCalcBlock([]string{"price = 100"})
	text2 := NewTextBlock([]string{"prose"})
	doc := NewDocumentFromBlocks(nil, []Block{text1, calc, text2})
	blocks := doc.GetBlocks()
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}
	if blocks[0].Block != text1 || blocks[1].Block != calc || blocks[2].Block != text2 {
		t.Error("expected blocks in input order, got reordered")
	}
}

func TestNewDocumentFromBlocks_AllBlocksHaveUniqueUUIDs(t *testing.T) {
	calc1 := NewCalcBlock([]string{"a = 1"})
	calc2 := NewCalcBlock([]string{"b = 2"})
	text := NewTextBlock([]string{"prose"})
	doc := NewDocumentFromBlocks(nil, []Block{calc1, calc2, text})
	seen := map[string]bool{}
	for _, node := range doc.GetBlocks() {
		if seen[node.ID] {
			t.Errorf("duplicate UUID: %s", node.ID)
		}
		seen[node.ID] = true
	}
}

func TestNewDocumentFromBlocks_GetBlockLookup_WorksByID(t *testing.T) {
	calc := NewCalcBlock([]string{"x = 42"})
	doc := NewDocumentFromBlocks(nil, []Block{calc})
	id := doc.GetBlocks()[0].ID
	node, ok := doc.GetBlock(id)
	if !ok {
		t.Fatalf("GetBlock(%s) returned ok=false", id)
	}
	if node.Block != calc {
		t.Error("GetBlock returned a different block than expected")
	}
}

func TestNewDocumentFromBlocks_FrontmatterStored(t *testing.T) {
	fm := &Frontmatter{}
	doc := NewDocumentFromBlocks(fm, nil)
	if doc.GetFrontmatter() != fm {
		t.Errorf("expected the same frontmatter pointer, got %v", doc.GetFrontmatter())
	}
}

func TestNewDocumentFromBlocks_DependencyGraphRebuilt(t *testing.T) {
	// Cross-block var dependency: calc B references calc A's var.
	// rebuildDependencies wires A → B; transitive-dependents
	// queries should report B when A's var changes.
	calcA := NewCalcBlock([]string{"price = 100"})
	calcB := NewCalcBlock([]string{"tax = price * 0.1"})
	doc := NewDocumentFromBlocks(nil, []Block{calcA, calcB})

	// After construction, asking "who depends on `price`?" should
	// list calc B (by its block ID).
	deps := doc.GetTransitiveDependents([]string{"price"})
	if len(deps) == 0 {
		t.Errorf("expected calc B to be a dependent of `price`, got empty list")
	}
	calcBID := doc.GetBlocks()[1].ID
	if !slices.Contains(deps, calcBID) {
		t.Errorf("expected calc B's ID (%s) in dependents, got %v", calcBID, deps)
	}
}
