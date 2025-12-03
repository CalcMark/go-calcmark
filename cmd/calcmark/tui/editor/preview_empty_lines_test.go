package editor

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestPreviewWithEmptyLineAfterHeading tests that empty lines between blocks
// are preserved in the preview pane (GetLineResults).
func TestPreviewWithEmptyLineAfterHeading(t *testing.T) {
	// This is the exact case from the user's screenshot:
	// # Test
	// <empty>
	// <empty>
	// 1. Boop
	// 1. Bop
	source := "# Test\n\n\n1. Boop\n1. Bop"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	results := m.GetLineResults()

	// Log what we got
	t.Logf("Got %d line results:", len(results))
	for i, r := range results {
		t.Logf("  [%d] %q (IsCalc=%v)", i, r.Source, r.IsCalc)
	}

	// Expected structure:
	// Line 0: "# Test"
	// Line 1: "" (empty)
	// Line 2: "" (empty)
	// Line 3: "1. Boop"
	// Line 4: "1. Bop"

	if len(results) < 5 {
		t.Fatalf("Expected at least 5 line results, got %d", len(results))
	}

	// Check that lines 1 and 2 are empty
	if results[1].Source != "" {
		t.Errorf("Line 1 should be empty, got %q", results[1].Source)
	}
	if results[2].Source != "" {
		t.Errorf("Line 2 should be empty, got %q", results[2].Source)
	}

	// Check that the ordered list appears on line 3
	if results[3].Source != "1. Boop" {
		t.Errorf("Line 3 should be '1. Boop', got %q", results[3].Source)
	}
}

// TestPreviewWithEmptyLineBeforeCalculation tests empty lines before a calculation.
func TestPreviewWithEmptyLineBeforeCalculation(t *testing.T) {
	source := "# Heading\n\n\na = 10"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	results := m.GetLineResults()

	t.Logf("Got %d line results:", len(results))
	for i, r := range results {
		t.Logf("  [%d] %q (IsCalc=%v)", i, r.Source, r.IsCalc)
	}

	// Expected:
	// Line 0: "# Heading"
	// Line 1: ""
	// Line 2: ""
	// Line 3: "a = 10"

	if len(results) < 4 {
		t.Fatalf("Expected at least 4 line results, got %d", len(results))
	}

	if results[1].Source != "" || results[2].Source != "" {
		t.Errorf("Lines 1 and 2 should be empty, got %q and %q",
			results[1].Source, results[2].Source)
	}

	if results[3].Source != "a = 10" {
		t.Errorf("Line 3 should be 'a = 10', got %q", results[3].Source)
	}
}

// TestBlockSourceContainsEmptyLines verifies that the document blocks
// themselves contain the empty separator lines.
func TestBlockSourceContainsEmptyLines(t *testing.T) {
	source := "# Test\n\n\n1. Boop"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	blocks := doc.GetBlocks()
	t.Logf("Document has %d blocks", len(blocks))

	for i, node := range blocks {
		switch b := node.Block.(type) {
		case *document.TextBlock:
			t.Logf("Block %d (Text): %d lines", i, len(b.Source()))
			for j, line := range b.Source() {
				t.Logf("  [%d] %q", j, line)
			}
		case *document.CalcBlock:
			t.Logf("Block %d (Calc): %d lines", i, len(b.Source()))
			for j, line := range b.Source() {
				t.Logf("  [%d] %q", j, line)
			}
		}
	}

	// First block should have 3 lines: heading + 2 empties
	if len(blocks) < 1 {
		t.Fatal("Expected at least 1 block")
	}

	firstBlock, ok := blocks[0].Block.(*document.TextBlock)
	if !ok {
		t.Fatalf("First block should be TextBlock, got %T", blocks[0].Block)
	}

	if len(firstBlock.Source()) != 3 {
		t.Errorf("First block should have 3 lines (heading + 2 empties), got %d",
			len(firstBlock.Source()))
	}

	// Check the empty lines are actually there
	if firstBlock.Source()[1] != "" || firstBlock.Source()[2] != "" {
		t.Errorf("Lines 1 and 2 of first block should be empty, got %q and %q",
			firstBlock.Source()[1], firstBlock.Source()[2])
	}
}
