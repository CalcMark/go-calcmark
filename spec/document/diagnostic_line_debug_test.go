package document_test

import (
	"strings"
	"testing"

	impldoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestDiagnosticLineNumberDebug helps debug where the error line number comes from.
func TestDiagnosticLineNumberDebug(t *testing.T) {
	// Single block with redefinition (one empty line between statements)
	source := "a = 3\n\nb = a * 2\n\na = 3"

	// Detect blocks
	detector := document.NewDetector()
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
	doc, docErr := document.NewDocument(source)
	if docErr != nil {
		t.Fatalf("NewDocument failed: %v", docErr)
	}

	// Evaluate - this is when redefinition check happens
	eval := impldoc.NewEvaluator()
	evalErr := eval.Evaluate(doc)
	t.Logf("\nEvaluate error: %v", evalErr)
	if evalErr == nil {
		t.Fatal("Expected error for redefinition")
	}

	// If doc was created (even partially), check diagnostics
	if doc != nil {
		docBlocks := doc.GetBlocks()
		t.Logf("\nDocument has %d blocks", len(docBlocks))
		for i, node := range docBlocks {
			if cb, ok := node.Block.(*document.CalcBlock); ok {
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

	detector := document.NewDetector()
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

	doc, docErr := document.NewDocument(source)
	if docErr != nil {
		t.Fatalf("NewDocument failed: %v", docErr)
	}

	// Evaluate - redefinition check happens here
	eval := impldoc.NewEvaluator()
	evalErr := eval.Evaluate(doc)
	t.Logf("\nEvaluate error: %v", evalErr)
	if evalErr == nil {
		t.Fatal("Expected error for redefinition")
	}

	if doc != nil {
		docBlocks := doc.GetBlocks()
		t.Logf("\nDocument has %d blocks", len(docBlocks))
		for i, node := range docBlocks {
			if cb, ok := node.Block.(*document.CalcBlock); ok {
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
