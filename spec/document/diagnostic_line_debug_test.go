package document

import (
	"strings"
	"testing"
)

// TestDiagnosticLineNumberDebug helps debug where the error line number comes from.
func TestDiagnosticLineNumberDebug(t *testing.T) {
	// Single block with redefinition (one empty line between statements)
	source := "a = 3\n\nb = a * 2\n\na = 3"

	// Detect blocks
	detector := NewDetector()
	blocks, err := detector.DetectBlocks(source)
	if err != nil {
		t.Fatalf("Failed to detect blocks: %v", err)
	}

	t.Logf("Source:\n%s", source)
	t.Logf("\nDetected %d blocks:", len(blocks))
	for i, block := range blocks {
		t.Logf("Block %d: %T", i, block)
		t.Logf("  Source lines: %v", block.Source())
	}

	// Create document and evaluate
	doc, docErr := NewDocument(source)
	if docErr != nil {
		t.Fatalf("NewDocument failed: %v", docErr)
	}

	// Evaluate - this is when redefinition check happens
	evalErr := doc.Evaluate()
	t.Logf("\nEvaluate error: %v", evalErr)
	if evalErr == nil {
		t.Fatal("Expected error for redefinition")
	}

	// If doc was created (even partially), check diagnostics
	if doc != nil {
		blocks := doc.GetBlocks()
		t.Logf("\nDocument has %d blocks", len(blocks))
		for i, node := range blocks {
			if cb, ok := node.Block.(*CalcBlock); ok {
				diags := cb.Diagnostics()
				t.Logf("Block %d diagnostics (%d):", i, len(diags))
				for j, diag := range diags {
					t.Logf("  [%d] Line=%d, Col=%d, Code=%s, Message=%s",
						j, diag.Line, diag.Column, diag.Code, diag.Message)
				}
			}
		}
	}
}

// TestDiagnosticWithSeparateBlocks tests redefinition across separate blocks.
func TestDiagnosticWithSeparateBlocks(t *testing.T) {
	// Two blocks separated by 2 empty lines
	source := "a = 3\n\n\na = 3"

	detector := NewDetector()
	blocks, err := detector.DetectBlocks(source)
	if err != nil {
		t.Fatalf("Failed to detect blocks: %v", err)
	}

	t.Logf("Source:\n%s", source)
	t.Logf("\nDetected %d blocks:", len(blocks))
	for i, block := range blocks {
		t.Logf("Block %d: %T", i, block)
		t.Logf("  Source lines: %v", block.Source())
	}

	// Count actual lines in source for comparison
	lines := strings.Split(source, "\n")
	t.Logf("\nSource has %d lines total:", len(lines))
	for i, line := range lines {
		t.Logf("  Line %d: %q", i, line)
	}

	doc, docErr := NewDocument(source)
	if docErr != nil {
		t.Fatalf("NewDocument failed: %v", docErr)
	}

	// Evaluate - redefinition check happens here
	evalErr := doc.Evaluate()
	t.Logf("\nEvaluate error: %v", evalErr)
	if evalErr == nil {
		t.Fatal("Expected error for redefinition")
	}

	if doc != nil {
		blocks := doc.GetBlocks()
		t.Logf("\nDocument has %d blocks", len(blocks))
		for i, node := range blocks {
			if cb, ok := node.Block.(*CalcBlock); ok {
				diags := cb.Diagnostics()
				t.Logf("Block %d diagnostics (%d):", i, len(diags))
				for j, diag := range diags {
					t.Logf("  [%d] Line=%d (block-relative), Code=%s, Message=%s",
						j, diag.Line, diag.Code, diag.Message)
				}

				// Also show source lines for context
				t.Logf("Block %d source:", i)
				for idx, srcLine := range cb.Source() {
					t.Logf("  [%d] %q", idx+1, srcLine) // 1-indexed to match diagnostic
				}
			}
		}
	}
}
