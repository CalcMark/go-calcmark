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

// TestGlobalVariableScope verifies that all variables have global scope.
// Variables defined in one block are accessible in all subsequent blocks,
// and reassignment in any block updates the single global binding.
func TestGlobalVariableScope(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		wantVarValues  map[string]string // variable name -> expected value string
		wantBlockCount int
	}{
		{
			name: "variable referenced across blocks",
			source: `x = 10


y = x + 5`,
			wantVarValues:  map[string]string{"x": "10", "y": "15"},
			wantBlockCount: 2,
		},
		{
			name: "variable reassigned in later block",
			source: `x = 10


x = 20`,
			wantVarValues:  map[string]string{"x": "20"},
			wantBlockCount: 2,
		},
		{
			name: "reassigned variable used by dependent",
			source: `x = 10


y = x * 2


x = 100`,
			wantVarValues:  map[string]string{"x": "100", "y": "20"},
			wantBlockCount: 3,
		},
		{
			name: "chain of dependencies across multiple blocks",
			source: `a = 5


b = a + 10


c = b * 2


d = c + a`,
			wantVarValues:  map[string]string{"a": "5", "b": "15", "c": "30", "d": "35"},
			wantBlockCount: 4,
		},
		{
			name: "variable reassigned multiple times",
			source: `x = 1


x = 2


x = 3


y = x`,
			wantVarValues:  map[string]string{"x": "3", "y": "3"},
			wantBlockCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := document.NewDocument(tt.source)
			if err != nil {
				t.Fatalf("NewDocument failed: %v", err)
			}

			eval := NewEvaluator()
			err = eval.Evaluate(doc)
			if err != nil {
				t.Fatalf("Evaluate failed: %v", err)
			}

			// Check block count
			blocks := doc.GetBlocks()
			calcBlockCount := 0
			for _, node := range blocks {
				if _, ok := node.Block.(*document.CalcBlock); ok {
					calcBlockCount++
				}
			}
			if calcBlockCount != tt.wantBlockCount {
				t.Errorf("Expected %d calc blocks, got %d", tt.wantBlockCount, calcBlockCount)
			}

			// Check variable values in the environment
			env := eval.GetEnvironment()
			for varName, wantValue := range tt.wantVarValues {
				val, ok := env.Get(varName)
				if !ok {
					t.Errorf("Variable %q not found in environment", varName)
					continue
				}
				gotValue := val.String()
				if gotValue != wantValue {
					t.Errorf("Variable %q: expected %q, got %q", varName, wantValue, gotValue)
				}
			}
		})
	}
}

// TestGlobalScopeEnvironmentPersistence verifies that the evaluator's environment
// persists across multiple EvaluateBlock calls (simulating REPL usage).
func TestGlobalScopeEnvironmentPersistence(t *testing.T) {
	// Start with initial document
	doc, err := document.NewDocument("x = 10\n")
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if err != nil {
		t.Fatalf("Initial Evaluate failed: %v", err)
	}

	// Check initial value
	env := eval.GetEnvironment()
	if val, ok := env.Get("x"); !ok || val.String() != "10" {
		t.Errorf("Initial x: expected '10', got %v (ok=%v)", val, ok)
	}

	// Add a new block that uses x
	blocks := doc.GetBlocks()
	lastID := blocks[len(blocks)-1].ID
	result, err := doc.InsertBlock(lastID, document.BlockCalculation, []string{"y = x + 5"})
	if err != nil {
		t.Fatalf("InsertBlock failed: %v", err)
	}

	err = eval.EvaluateBlock(doc, result.ModifiedBlockID)
	if err != nil {
		t.Fatalf("EvaluateBlock failed: %v", err)
	}

	// Re-get environment after EvaluateBlock (it may be replaced)
	env = eval.GetEnvironment()

	// Check that y was computed using x's value
	if val, ok := env.Get("y"); !ok || val.String() != "15" {
		t.Errorf("y = x + 5: expected '15', got %v (ok=%v)", val, ok)
	}

	// Add another block that modifies x
	blocks = doc.GetBlocks()
	lastID = blocks[len(blocks)-1].ID
	result, err = doc.InsertBlock(lastID, document.BlockCalculation, []string{"x = 100"})
	if err != nil {
		t.Fatalf("InsertBlock 2 failed: %v", err)
	}

	err = eval.EvaluateBlock(doc, result.ModifiedBlockID)
	if err != nil {
		t.Fatalf("EvaluateBlock 2 failed: %v", err)
	}

	// Re-get environment after EvaluateBlock
	env = eval.GetEnvironment()

	// Check that x was updated to new value
	if val, ok := env.Get("x"); !ok || val.String() != "100" {
		t.Errorf("x = 100: expected '100', got %v (ok=%v)", val, ok)
	}

	// With reactive semantics, y should now be 105 (x + 5 = 100 + 5)
	// because EvaluateBlock re-evaluates all blocks with final variable values
	if val, ok := env.Get("y"); !ok || val.String() != "105" {
		t.Errorf("y after x change: expected '105' (reactive), got %v (ok=%v)", val, ok)
	}
}

// TestGlobalScopeWithinSingleBlock verifies that multiple assignments
// within the same block work correctly with global scope.
func TestGlobalScopeWithinSingleBlock(t *testing.T) {
	source := `x = 1
x = 2
y = x
x = 3
z = x + y
`

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	env := eval.GetEnvironment()

	// Final values after all statements execute
	tests := map[string]string{
		"x": "3", // x = 3 was last assignment
		"y": "2", // y = x when x was 2
		"z": "5", // z = 3 + 2 = 5
	}

	for varName, wantValue := range tests {
		val, ok := env.Get(varName)
		if !ok {
			t.Errorf("Variable %q not found", varName)
			continue
		}
		if val.String() != wantValue {
			t.Errorf("Variable %q: expected %q, got %q", varName, wantValue, val.String())
		}
	}
}
