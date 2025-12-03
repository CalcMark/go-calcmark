package document

import (
	"testing"
)

// TestOrderedListFollowedByCalc tests block detection when an ordered list
// is followed by a calculation with one empty line in between.
func TestOrderedListFollowedByCalc(t *testing.T) {
	// This is what the user is typing:
	// Line 0: "1. hi"
	// Line 1: "2. yes"
	// Line 2: "" (empty)
	// Line 3: "a = b"

	source := "1. hi\n2. yes\n\na = b"

	detector := NewDetector()
	blocks, err := detector.DetectBlocks(source)
	if err != nil {
		t.Fatalf("Failed to detect blocks: %v", err)
	}

	t.Logf("Source:\n%s\n", source)
	t.Logf("Detected %d blocks:", len(blocks))
	for i, block := range blocks {
		t.Logf("Block %d: %T", i, block)
		t.Logf("  Source: %v", block.Source())
	}

	// Type changes cause block splits even with only one empty line
	// TextBlock (ordered list) -> CalcBlock (calculation)
	if len(blocks) != 2 {
		t.Errorf("Expected 2 blocks (text then calc), got %d", len(blocks))
	}

	// First block should be a TextBlock (ordered list)
	if _, ok := blocks[0].(*TextBlock); !ok {
		t.Errorf("Block 0: expected TextBlock, got %T", blocks[0])
	}

	// Second block should be a CalcBlock
	if len(blocks) > 1 {
		if _, ok := blocks[1].(*CalcBlock); !ok {
			t.Errorf("Block 1: expected CalcBlock, got %T", blocks[1])
		}
	}
}

// TestOrderedListThenCalcSeparate tests with TWO empty lines (proper separation).
func TestOrderedListThenCalcSeparate(t *testing.T) {
	// With TWO empty lines, they should be separate blocks
	source := "1. hi\n2. yes\n\n\na = b"

	detector := NewDetector()
	blocks, err := detector.DetectBlocks(source)
	if err != nil {
		t.Fatalf("Failed to detect blocks: %v", err)
	}

	t.Logf("Source:\n%s\n", source)
	t.Logf("Detected %d blocks:", len(blocks))
	for i, block := range blocks {
		t.Logf("Block %d: %T", i, block)
		t.Logf("  Source: %v", block.Source())
	}

	// With TWO empty lines, should be 2 blocks
	if len(blocks) != 2 {
		t.Errorf("Expected 2 blocks, got %d", len(blocks))
	}

	// First should be TextBlock (ordered list)
	if _, ok := blocks[0].(*TextBlock); !ok {
		t.Errorf("Block 0: expected TextBlock, got %T", blocks[0])
	}

	// Second should be CalcBlock
	if _, ok := blocks[1].(*CalcBlock); !ok {
		t.Errorf("Block 1: expected CalcBlock, got %T", blocks[1])
	}
}
