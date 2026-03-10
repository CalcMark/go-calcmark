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
		// REMOVED: variable reassignment is not allowed in CalcMark
		// Variables can only be defined once per document
		{
			name: "chain of dependencies across multiple blocks",
			source: `a = 5


b = a + 10


c = b * 2


d = c + a`,
			wantVarValues:  map[string]string{"a": "5", "b": "15", "c": "30", "d": "35"},
			wantBlockCount: 4,
		},
		// REMOVED: variable reassignment is not allowed in CalcMark
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

	// Variable reassignment is not allowed in CalcMark.
	// Attempting to redefine 'x' should fail during evaluation.
	blocks = doc.GetBlocks()
	lastID = blocks[len(blocks)-1].ID
	result, err = doc.InsertBlock(lastID, document.BlockCalculation, []string{"x = 100"})
	if err != nil {
		t.Fatalf("InsertBlock 2 failed: %v", err)
	}

	// This should fail because x is already defined
	err = eval.EvaluateBlock(doc, result.ModifiedBlockID)
	if err == nil {
		t.Fatal("Expected error for variable redefinition, but got nil")
	}
	if !strings.Contains(err.Error(), "already defined") && !strings.Contains(err.Error(), "redefinition") {
		t.Errorf("Expected redefinition error, got: %v", err)
	}
}

// TestGlobalScopeWithinSingleBlock verifies that multiple assignments
// of the same variable within a block are rejected.
func TestGlobalScopeWithinSingleBlock(t *testing.T) {
	source := `x = 1
x = 2
y = x
x = 3
z = x + y
`

	// Create and evaluate document - should fail due to variable redefinition
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Evaluation should fail
	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if err == nil {
		t.Fatal("Expected error for variable redefinition within single block, got nil")
	}
	if !strings.Contains(err.Error(), "already defined") && !strings.Contains(err.Error(), "redefinition") {
		t.Errorf("Expected redefinition error, got: %v", err)
	}
}

// TestExplicitInDoesNotMutateVariable verifies that using "in" on a variable
// in a later block does not set IsExplicit on the original value, which would
// cause convert_to transforms to skip it.
func TestExplicitInDoesNotMutateVariable(t *testing.T) {
	source := `---
scale: 2
convert_to: si
---
butter = 4 ounces

Text break

butter_oz = butter in ounces`

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	blocks := doc.GetBlocks()

	// Find the calc blocks
	var butterBlock *document.CalcBlock
	var butterOzBlock *document.CalcBlock
	for _, node := range blocks {
		cb, ok := node.Block.(*document.CalcBlock)
		if !ok {
			continue
		}
		src := strings.Join(cb.Source(), " ")
		if strings.Contains(src, "butter = 4") {
			butterBlock = cb
		}
		if strings.Contains(src, "butter_oz") {
			butterOzBlock = cb
		}
	}

	if butterBlock == nil {
		t.Fatal("Could not find butter block")
	}
	if butterOzBlock == nil {
		t.Fatal("Could not find butter_oz block")
	}

	// butter should be scaled (4→8) and converted to SI (ounces→grams ≈ 227g)
	butterResults := butterBlock.Results()
	if len(butterResults) == 0 {
		t.Fatal("butter block has no results")
	}
	butterVal := butterResults[0].String()
	if !strings.Contains(butterVal, "g") || strings.Contains(butterVal, "ounces") {
		t.Errorf("butter should be converted to grams, got %q", butterVal)
	}

	// butter_oz should stay in ounces (explicit "in" overrides convert_to)
	ozResults := butterOzBlock.Results()
	if len(ozResults) == 0 {
		t.Fatal("butter_oz block has no results")
	}
	ozVal := ozResults[0].String()
	if !strings.Contains(ozVal, "ounces") {
		t.Errorf("butter_oz should stay in ounces (explicit in), got %q", ozVal)
	}
}

// TestScaleNumber verifies that Number type scales when explicitly listed
// in unit_categories, and that @scale exemption prevents double-scaling.
func TestScaleNumber(t *testing.T) {
	source := "---\nscale:\n  factor: 2\n  unit_categories: [Number]\n---\na = 10\nb = a * @scale\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	blocks := doc.GetBlocks()
	var calcBlocks []*document.CalcBlock
	for _, node := range blocks {
		if cb, ok := node.Block.(*document.CalcBlock); ok {
			calcBlocks = append(calcBlocks, cb)
		}
	}

	if len(calcBlocks) == 0 {
		t.Fatal("no calc blocks found")
	}

	// The block has two statements: a = 10 and b = a * @scale
	cb := calcBlocks[0]
	results := cb.Results()
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// "a = 10" → raw value 10, scaled by 2 → 20 (Number in unit_categories)
	aVal := results[0].String()
	if aVal != "20" {
		t.Errorf("a: expected 20 (scaled Number), got %q", aVal)
	}

	// "b = a * @scale" → raw value 10 * 2 = 20, but @scale exempt → 20 (no double-scale)
	bVal := results[1].String()
	if bVal != "20" {
		t.Errorf("b: expected 20 (@scale exempt, no double-scaling), got %q", bVal)
	}
}

// TestScaleNumberDefaultImmune verifies that Number is immune to scale
// when Number is NOT listed in unit_categories.
func TestScaleNumberDefaultImmune(t *testing.T) {
	source := "---\nscale: 3\n---\na = 10\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	blocks := doc.GetBlocks()
	for _, node := range blocks {
		if cb, ok := node.Block.(*document.CalcBlock); ok {
			results := cb.Results()
			if len(results) == 0 {
				t.Fatal("no results")
			}
			aVal := results[0].String()
			if aVal != "10" {
				t.Errorf("a: expected 10 (Number immune by default), got %q", aVal)
			}
			return
		}
	}
	t.Fatal("no calc block found")
}

// TestEvaluateInterpolatesTextBlocks tests that Evaluate() resolves {{var}} tags
// in TextBlocks using the final environment.
func TestEvaluateInterpolatesTextBlocks(t *testing.T) {
	source := "## Summary\n\nTotal: {{total}}\n\n\ntotal = $250\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Find the TextBlock and verify interpolation
	for _, node := range doc.GetBlocks() {
		tb, ok := node.Block.(*document.TextBlock)
		if !ok {
			continue
		}
		interp := tb.InterpolatedSource()
		for _, line := range interp {
			if strings.Contains(line, "$250") {
				// Source should still have the raw tag
				for _, raw := range tb.Source() {
					if strings.Contains(raw, "{{total}}") {
						return // Success
					}
				}
				t.Error("Source() should still contain {{total}}")
				return
			}
		}
	}
	t.Error("expected interpolated TextBlock with $250")
}

// TestEvaluateForwardReference tests that {{var}} tags in TextBlocks above
// the CalcBlock that defines the variable still resolve.
func TestEvaluateForwardReference(t *testing.T) {
	source := "Result: {{x}}\n\n\nx = 42\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	for _, node := range doc.GetBlocks() {
		tb, ok := node.Block.(*document.TextBlock)
		if !ok {
			continue
		}
		interp := tb.InterpolatedSource()
		for _, line := range interp {
			if strings.Contains(line, "Result: 42") {
				return // Success — forward reference resolved
			}
		}
	}
	t.Error("expected forward reference {{x}} to resolve to 42")
}

// TestEvaluateInterpolationPreservesSource tests that Source() is never mutated.
func TestEvaluateInterpolationPreservesSource(t *testing.T) {
	source := "Value: {{x}}\n\n\nx = 100\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	for _, node := range doc.GetBlocks() {
		tb, ok := node.Block.(*document.TextBlock)
		if !ok {
			continue
		}
		for _, line := range tb.Source() {
			if strings.Contains(line, "{{x}}") {
				return // Source preserved
			}
		}
	}
	t.Error("Source() should still contain raw {{x}} tag")
}
