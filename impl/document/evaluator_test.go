package document

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestDocumentEvaluation tests the full evaluation pipeline.
func TestDocumentEvaluation(t *testing.T) {
	source := `x = 10


y = x + 5


z = y * 2`

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	// Evaluate all blocks
	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	blocks := doc.GetBlocks()
	if len(blocks) != 3 {
		t.Fatalf("Expected 3 blocks, got %d", len(blocks))
	}

	// Check each block was evaluated
	for _, node := range blocks {
		if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
			if calcBlock.LastValue() == nil {
				t.Errorf("Block %s has nil LastValue", node.ID[:8])
			}
			if calcBlock.IsDirty() {
				t.Errorf("Block %s still dirty after evaluation", node.ID[:8])
			}
			if calcBlock.Error() != nil {
				t.Errorf("Block %s has error: %v", node.ID[:8], calcBlock.Error())
			}

			t.Logf("Block %s: %v = %v",
				node.ID[:8],
				strings.Join(calcBlock.Source(), ""),
				calcBlock.LastValue())
		}
	}
}

// TestIncrementalEvaluation tests re-evaluation after block changes.
func TestIncrementalEvaluation(t *testing.T) {
	source := `x = 10


y = x + 5`

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	// Initial evaluation
	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if err != nil {
		t.Fatalf("Initial Evaluate failed: %v", err)
	}

	blocks := doc.GetBlocks()
	var xBlockID string
	for _, node := range blocks {
		if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
			if strings.Contains(strings.Join(calcBlock.Source(), ""), "x = 10") {
				xBlockID = node.ID
				break
			}
		}
	}

	if xBlockID == "" {
		t.Fatal("Could not find x block")
	}

	// Change x to 100
	_, err = doc.ReplaceBlockSource(xBlockID, []string{"x = 100"})
	if err != nil {
		t.Fatalf("ReplaceBlockSource failed: %v", err)
	}

	// Re-evaluate starting from changed block
	err = eval.EvaluateBlock(doc, xBlockID)
	if err != nil {
		t.Fatalf("EvaluateBlock failed: %v", err)
	}

	// Check that evaluation succeeded
	node, _ := doc.GetBlock(xBlockID)
	calcBlock := node.Block.(*document.CalcBlock)

	if calcBlock.Error() != nil {
		t.Errorf("Block has error after re-evaluation: %v", calcBlock.Error())
	}

	t.Logf("✅ Incremental evaluation: x changed, re-evaluated successfully")
}

// TestEvaluationError tests error handling during evaluation.
func TestEvaluationError(t *testing.T) {
	// Undefined variable should cause error
	source := `result = undefined_var + 10
`

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	// Evaluation should fail
	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if err == nil {
		t.Fatal("Expected evaluation error for undefined variable, got nil")
	}

	t.Logf("✅ Error handling: %v", err)

	// Block should have the error stored
	blocks := doc.GetBlocks()
	if len(blocks) > 0 {
		if calcBlock, ok := blocks[0].Block.(*document.CalcBlock); ok {
			if calcBlock.Error() == nil {
				t.Error("Expected block to have error stored")
			}
		}
	}
}

// TestMixedBlocksEvaluation tests documents with both calc and text blocks.
func TestMixedBlocksEvaluation(t *testing.T) {
	source := `x = 10


# This is markdown text


y = x + 5`

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Count block types
	calcCount := 0
	textCount := 0

	for _, node := range doc.GetBlocks() {
		switch node.Block.Type() {
		case document.BlockCalculation:
			calcCount++
			calcBlock := node.Block.(*document.CalcBlock)
			if calcBlock.LastValue() == nil {
				t.Error("CalcBlock should have LastValue after evaluation")
			}
		case document.BlockText:
			textCount++
			// TextBlocks don't get evaluated
		}
	}

	if calcCount != 2 {
		t.Errorf("Expected 2 calc blocks, got %d", calcCount)
	}
	if textCount != 1 {
		t.Errorf("Expected 1 text block, got %d", textCount)
	}

	t.Logf("✅ Mixed blocks: %d calc, %d text", calcCount, textCount)
}
