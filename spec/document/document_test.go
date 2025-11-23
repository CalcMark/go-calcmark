package document

import (
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
