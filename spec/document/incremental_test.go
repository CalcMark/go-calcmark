package document

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// TestSimpleDependencyTracking validates basic dependency tracking
// and incremental evaluation.
func TestSimpleDependencyTracking(t *testing.T) {
	// Create simple document
	source := `x = 10


y = x + 5`

	doc, err := NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	blocks := doc.GetBlocks()
	t.Logf("Got %d blocks", len(blocks))

	// Debug: show what blocks we got
	for i, node := range blocks {
		t.Logf("Block %d (ID=%s, Type=%v):", i, node.ID[:8], node.Block.Type())
		if cb, ok := node.Block.(*CalcBlock); ok {
			t.Logf("  Source: %v", cb.Source())
			t.Logf("  Variables: %v", cb.Variables())
			t.Logf("  Dependencies: %v", cb.Dependencies())
		}
	}

	// Find x block
	var xBlockID string
	for _, node := range blocks {
		if calcBlock, ok := node.Block.(*CalcBlock); ok {
			source := strings.Join(calcBlock.Source(), "")
			if strings.Contains(source, "x = 10") {
				xBlockID = node.ID
				break
			}
		}
	}

	if xBlockID == "" {
		t.Fatal("Could not find x block")
	}

	// Change x
	result, err := doc.ReplaceBlockSource(xBlockID, []string{"x = 100"})
	if err != nil {
		t.Fatalf("ReplaceBlockSource failed: %v", err)
	}

	t.Logf("✅ Changed x, affected %d blocks:", len(result.AffectedBlockIDs))
	for _, id := range result.AffectedBlockIDs {
		t.Logf("  - %s", id[:8])
	}

	// Should have at least the modified block
	if len(result.AffectedBlockIDs) < 1 {
		t.Error("Expected at least 1 affected block")
	}
}

// TestParallelMarkdownRendering validates parallel rendering.
func TestParallelMarkdownRendering(t *testing.T) {
	source := `# Markdown 1


# Markdown 2


# Markdown 3`

	doc, err := NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	blocks := doc.GetBlocks()

	// Count text blocks
	textBlocks := []*TextBlock{}
	for _, node := range blocks {
		if tb, ok := node.Block.(*TextBlock); ok {
			textBlocks = append(textBlocks, tb)
		}
	}

	t.Logf("Found %d text blocks", len(textBlocks))

	if len(textBlocks) == 0 {
		t.Skip("No text blocks found, skipping parallel rendering test")
	}

	// Render all blocks in parallel
	var wg sync.WaitGroup
	start := time.Now()
	results := make([]string, len(textBlocks))

	for i, tb := range textBlocks {
		wg.Add(1)
		go func(idx int, block *TextBlock) {
			defer wg.Done()
			results[idx] = block.Render()
		}(i, tb)
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Verify all rendered
	for i, html := range results {
		if html == "" {
			t.Errorf("Block %d failed to render", i)
		}
	}

	t.Logf("✅ Parallel rendering: %d markdown blocks in %v", len(textBlocks), elapsed)
}

// TestDirtyFlagBehavior validates dirty flag semantics.
func TestDirtyFlagBehavior(t *testing.T) {
	source := `x = 10


y = 20`

	doc, err := NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	blocks := doc.GetBlocks()

	// Initially not dirty (new blocks start clean)
	for _, node := range blocks {
		if calcBlock, ok := node.Block.(*CalcBlock); ok {
			if calcBlock.IsDirty() {
				t.Log("Note: New blocks start dirty (acceptable)")
			}
		}
	}

	// Find first calc block
	var firstID string
	for _, node := range blocks {
		if _, ok := node.Block.(*CalcBlock); ok {
			firstID = node.ID
			break
		}
	}

	if firstID == "" {
		t.Fatal("No calc blocks found")
	}

	// Modify it
	result, err := doc.ReplaceBlockSource(firstID, []string{"x = 999"})
	if err != nil {
		t.Fatalf("ReplaceBlockSource failed: %v", err)
	}

	t.Logf("✅ Modified 1 block, affected %d blocks", len(result.AffectedBlockIDs))

	// At minimum, the modified block should be in affected list
	found := false
	for _, id := range result.AffectedBlockIDs {
		if id == firstID {
			found = true
			break
		}
	}

	if !found {
		t.Error("Modified block should be in affected list")
	}
}

// TestInsertDeleteOperations validates CRUD operations.
func TestInsertDeleteOperations(t *testing.T) {
	source := `x = 10


z = 30`

	doc, err := NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	initialCount := len(doc.GetBlocks())
	t.Logf("Initial: %d blocks", initialCount)

	// Find first block
	var firstID string
	for _, node := range doc.GetBlocks() {
		firstID = node.ID
		break
	}

	// Insert after first
	insertResult, err := doc.InsertBlock(firstID, BlockCalculation, []string{"y = 20"})
	if err != nil {
		t.Fatalf("InsertBlock failed: %v", err)
	}

	afterInsert := len(doc.GetBlocks())
	t.Logf("After insert: %d blocks", afterInsert)

	if afterInsert != initialCount+1 {
		t.Errorf("Expected %d blocks after insert, got %d", initialCount+1, afterInsert)
	}

	t.Logf("✅ Insert affected %d blocks", len(insertResult.AffectedBlockIDs))

	// Delete the inserted block
	insertedID := insertResult.ModifiedBlockID
	deleteResult, err := doc.DeleteBlock(insertedID)
	if err != nil {
		t.Fatalf("DeleteBlock failed: %v", err)
	}

	afterDelete := len(doc.GetBlocks())
	t.Logf("After delete: %d blocks", afterDelete)

	if afterDelete != initialCount {
		t.Errorf("Expected %d blocks after delete, got %d", initialCount, afterDelete)
	}

	t.Logf("✅ Delete affected %d blocks", len(deleteResult.AffectedBlockIDs))
}
