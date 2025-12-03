package document

import (
	"strings"
	"testing"
)

// TestRedefinitionAcrossBlocks tests that redefinition is detected even when
// the variable is defined in a different block.
func TestRedefinitionAcrossBlocks(t *testing.T) {
	// Two blocks separated by 2 empty lines
	source := "a = 3\n\n\na = 3"

	doc, err := NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	if doc == nil {
		t.Fatal("Document is nil")
	}

	// Evaluate the document - this is when semantic checking happens
	evalErr := doc.Evaluate()
	if evalErr != nil {
		t.Logf("Evaluate error (expected for redefinition): %v", evalErr)
		// Error is expected for redefinition, but we want to check the diagnostics
	}

	blocks := doc.GetBlocks()
	t.Logf("Document has %d blocks", len(blocks))

	// Block 0 should have no diagnostics (first definition is valid)
	if cb0, ok := blocks[0].Block.(*CalcBlock); ok {
		diags0 := cb0.Diagnostics()
		t.Logf("Block 0 has %d diagnostics", len(diags0))
		for i, diag := range diags0 {
			t.Logf("  [%d] Line=%d, Code=%s, Message=%s", i, diag.Line, diag.Code, diag.Message)
		}
		if len(diags0) > 0 {
			t.Error("Block 0 should have no diagnostics (first definition is valid)")
		}
	}

	// Block 1 should have a redefinition diagnostic
	if len(blocks) > 1 {
		if cb1, ok := blocks[1].Block.(*CalcBlock); ok {
			diags1 := cb1.Diagnostics()
			t.Logf("Block 1 has %d diagnostics", len(diags1))
			for i, diag := range diags1 {
				t.Logf("  [%d] Line=%d, Code=%s, Message=%s", i, diag.Line, diag.Code, diag.Message)
			}

			if len(diags1) == 0 {
				t.Error("Block 1 should have a redefinition diagnostic")
			} else {
				// Check that the diagnostic is for redefinition
				hasRedef := false
				for _, diag := range diags1 {
					if diag.Code == "variable_redefinition" ||
						strings.Contains(diag.Message, "already defined") {
						hasRedef = true
						// The diagnostic should be on line 1 of the block (the only line)
						if diag.Line != 1 {
							t.Errorf("Redefinition diagnostic should be on line 1 of block, got line %d", diag.Line)
						}
					}
				}
				if !hasRedef {
					t.Error("Block 1 diagnostics don't include redefinition error")
				}
			}
		}
	} else {
		t.Error("Expected 2 blocks (separated by empty lines), got 1")
	}
}

// TestRedefinitionInSameBlockWithEmptyLine tests redefinition in the same block
// when there's an empty line between assignments (but not 2 empties, so still same block).
func TestRedefinitionInSameBlockWithEmptyLine(t *testing.T) {
	// One empty line - stays in same block
	source := "a = 2\n\nb = a * 2\n\na = 3"

	doc, err := NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	if doc == nil {
		t.Fatal("Document is nil")
	}

	// Evaluate to trigger semantic checking
	evalErr := doc.Evaluate()
	if evalErr != nil {
		t.Logf("Evaluate error (expected): %v", evalErr)
	}

	blocks := doc.GetBlocks()
	t.Logf("Document has %d blocks", len(blocks))

	// Should be 1 block (single empty lines don't split blocks)
	if len(blocks) != 1 {
		t.Errorf("Expected 1 block, got %d", len(blocks))
	}

	if cb, ok := blocks[0].Block.(*CalcBlock); ok {
		diags := cb.Diagnostics()
		t.Logf("Block has %d diagnostics", len(diags))
		for i, diag := range diags {
			t.Logf("  [%d] Line=%d, Code=%s, Message=%s", i, diag.Line, diag.Code, diag.Message)
		}

		// Should have a redefinition diagnostic
		if len(diags) == 0 {
			t.Error("Expected redefinition diagnostic")
		} else {
			// The diagnostic should be on line 5 (the second "a = 3", which is the 5th line in the source)
			// But within the block, it's relative to the block's source
			// Block source is: [a = 2, , b = a * 2, , a = 3]
			// So it's line 5 (1-indexed)
			found := false
			for _, diag := range diags {
				if diag.Code == "variable_redefinition" {
					found = true
					if diag.Line != 5 {
						t.Errorf("Redefinition diagnostic should be on line 5 of block, got line %d", diag.Line)
					}
				}
			}
			if !found {
				t.Error("No redefinition diagnostic found")
			}
		}
	}
}
