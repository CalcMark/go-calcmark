package document_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestDocumentEvaluationStoresResults verifies document evaluation stores LastValue
func TestDocumentEvaluationStoresResults(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectResult bool
	}{
		{"simple conversion", "10 meters in feet", true},
		{"quantity literal", "10 meters", true},
		{"assignment", "x = 10 meters", true},
		{"arithmetic", "5 + 3", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := document.NewDocument(tt.input)
			if err != nil {
				t.Fatalf("NewDocument error: %v", err)
			}

			err = doc.Evaluate()
			if err != nil {
				t.Fatalf("Evaluate error: %v", err)
			}

			blocks := doc.GetBlocks()
			if len(blocks) == 0 {
				t.Fatal("No blocks created")
			}

			calcBlock, ok := blocks[0].Block.(*document.CalcBlock)
			if !ok {
				t.Fatalf("First block is not CalcBlock, got %T", blocks[0].Block)
			}

			lastVal := calcBlock.LastValue()
			if tt.expectResult && lastVal == nil {
				t.Errorf("Expected LastValue but got nil")
			}

			if lastVal != nil {
				t.Logf("✓ LastValue: %v", lastVal)
			} else {
				t.Logf("✗ LastValue is nil")
			}
		})
	}
}
