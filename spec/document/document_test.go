package document

import (
	"slices"
	"testing"
)

func TestDocumentCreation(t *testing.T) {
	source := `x = 10
y = 20

# Results
The calculation above computes values.


z = x + y`

	doc, err := NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}

	blocks := doc.GetBlocks()
	if len(blocks) == 0 {
		t.Error("Expected blocks, got none")
	}

	// Each block should have a UUID
	for i, block := range blocks {
		if block.ID == "" {
			t.Errorf("Block %d missing UUID", i)
		}
	}
}

func TestBlockReplacement(t *testing.T) {
	source := "x = 10"

	doc, err := NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}

	blocks := doc.GetBlocks()
	if len(blocks) != 1 {
		t.Fatalf("Expected 1 block, got %d", len(blocks))
	}

	blockID := blocks[0].ID

	// Replace source
	result, err := doc.ReplaceBlockSource(blockID, []string{"x = 20"})
	if err != nil {
		t.Fatalf("ReplaceBlockSource() error = %v", err)
	}

	if result.ModifiedBlockID != blockID {
		t.Errorf("Expected modified block %s, got %s", blockID, result.ModifiedBlockID)
	}

	if len(result.AffectedBlockIDs) == 0 {
		t.Error("Expected affected blocks")
	}
}

func TestBlockInsertion(t *testing.T) {
	source := "x = 10"

	doc, err := NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}

	originalBlockID := doc.GetBlocks()[0].ID

	// Insert after first block
	result, err := doc.InsertBlock(originalBlockID, BlockCalculation, []string{"y = 20"})
	if err != nil {
		t.Fatalf("InsertBlock() error = %v", err)
	}

	if len(doc.GetBlocks()) != 2 {
		t.Errorf("Expected 2 blocks after insert, got %d", len(doc.GetBlocks()))
	}

	if result.ModifiedBlockID == "" {
		t.Error("Expected new block ID")
	}
}

func TestBlockDeletion(t *testing.T) {
	source := `x = 10
y = 20`

	doc, err := NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}

	blocks := doc.GetBlocks()
	if len(blocks) == 0 {
		t.Fatal("Expected at least one block")
	}

	blockToDelete := blocks[0].ID

	// Delete first block
	result, err := doc.DeleteBlock(blockToDelete)
	if err != nil {
		t.Fatalf("DeleteBlock() error = %v", err)
	}

	if result.ModifiedBlockID != blockToDelete {
		t.Errorf("Expected deleted block ID %s, got %s", blockToDelete, result.ModifiedBlockID)
	}

	// Verify block is gone
	_, exists := doc.GetBlock(blockToDelete)
	if exists {
		t.Error("Block should have been deleted")
	}
}

// TestInsertBlockAffectsDependents tests that inserting a block that redefines
// a variable marks dependent blocks as affected.
func TestInsertBlockAffectsDependents(t *testing.T) {
	// Create document: alice = 100, then jim = alice * 2 (separate blocks)
	// Two blank lines separate blocks in CalcMark
	source := `alice = 100


jim = alice * 2`

	doc, err := NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}

	blocks := doc.GetBlocks()
	if len(blocks) != 2 {
		t.Fatalf("Expected 2 blocks, got %d", len(blocks))
	}

	// Find the block ID for jim (which depends on alice)
	jimBlockID := blocks[1].ID
	lastBlockID := blocks[len(blocks)-1].ID

	// Verify jim's block has alice as dependency
	jimBlock := blocks[1].Block.(*CalcBlock)
	t.Logf("Jim's dependencies: %v", jimBlock.Dependencies())

	// Now insert a new block that redefines alice
	result, err := doc.InsertBlock(lastBlockID, BlockCalculation, []string{"alice = 200"})
	if err != nil {
		t.Fatalf("InsertBlock() error = %v", err)
	}

	t.Logf("New block ID: %s", result.ModifiedBlockID)
	t.Logf("Affected blocks: %v", result.AffectedBlockIDs)

	// The affected blocks should include:
	// 1. The new block (alice = 200)
	// 2. jim's block because jim depends on alice
	if len(result.AffectedBlockIDs) < 2 {
		t.Errorf("Expected at least 2 affected blocks, got %d", len(result.AffectedBlockIDs))
	}

	// Check that jim's block is in the affected list
	if !slices.Contains(result.AffectedBlockIDs, jimBlockID) {
		t.Errorf("Expected jim's block (%s) to be in affected blocks: %v", jimBlockID, result.AffectedBlockIDs)
	}
}

// TestInsertBlockDependencyChain tests that dependency chains are followed.
func TestInsertBlockDependencyChain(t *testing.T) {
	// Create: x = 1, y = x + 1, z = y + 1 (separate blocks)
	// Chain: x -> y -> z
	source := `x = 1


y = x + 1


z = y + 1`

	doc, err := NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}

	blocks := doc.GetBlocks()
	if len(blocks) != 3 {
		t.Fatalf("Expected 3 blocks, got %d", len(blocks))
	}

	lastBlockID := blocks[len(blocks)-1].ID

	// Log dependencies for debugging
	for i, block := range blocks {
		if cb, ok := block.Block.(*CalcBlock); ok {
			t.Logf("Block %d: vars=%v, deps=%v", i, cb.Variables(), cb.Dependencies())
		}
	}

	// Insert new block that redefines x
	result, err := doc.InsertBlock(lastBlockID, BlockCalculation, []string{"x = 100"})
	if err != nil {
		t.Fatalf("InsertBlock() error = %v", err)
	}

	t.Logf("Affected blocks: %v", result.AffectedBlockIDs)

	// With transitive dependency tracking:
	// - y depends on x (direct)
	// - z depends on y (transitive through y)
	// So ALL 3 blocks (new x, y, z) should be affected
	if len(result.AffectedBlockIDs) < 3 {
		t.Errorf("Expected at least 3 affected blocks (new x + y + z), got %d: %v",
			len(result.AffectedBlockIDs), result.AffectedBlockIDs)
	}

	// Verify y's block is affected
	yBlockID := blocks[1].ID
	if !slices.Contains(result.AffectedBlockIDs, yBlockID) {
		t.Errorf("Expected y's block (%s) in affected blocks", yBlockID)
	}

	// Verify z's block is affected (transitive dependency)
	zBlockID := blocks[2].ID
	if !slices.Contains(result.AffectedBlockIDs, zBlockID) {
		t.Errorf("Expected z's block (%s) in affected blocks (transitive)", zBlockID)
	}
}
