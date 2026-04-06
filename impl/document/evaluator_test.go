package document

import (
	"errors"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
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
	if !errors.Is(err, ErrPartialEvaluation) {
		t.Errorf("Expected ErrPartialEvaluation, got: %v", err)
	}
	// The redefinition error should be on the block's diagnostics
	redefNode, _ := doc.GetBlock(result.ModifiedBlockID)
	redefBlock := redefNode.Block.(*document.CalcBlock)
	if redefBlock.Error() == nil {
		t.Error("Expected block to have error for variable redefinition")
	} else if !strings.Contains(redefBlock.Error().Error(), "already defined") && !strings.Contains(redefBlock.Error().Error(), "redefinition") {
		t.Errorf("Expected redefinition error on block, got: %v", redefBlock.Error())
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

	// Evaluation should return ErrPartialEvaluation (continues past errors)
	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if err == nil {
		t.Fatal("Expected error for variable redefinition within single block, got nil")
	}
	if !errors.Is(err, ErrPartialEvaluation) {
		t.Errorf("Expected ErrPartialEvaluation, got: %v", err)
	}
	// The redefinition error should be on the block's error
	blocks := doc.GetBlocks()
	if len(blocks) > 0 {
		if cb, ok := blocks[0].Block.(*document.CalcBlock); ok {
			if cb.Error() == nil {
				t.Error("Expected block to have redefinition error")
			} else if !strings.Contains(cb.Error().Error(), "already defined") && !strings.Contains(cb.Error().Error(), "redefinition") {
				t.Errorf("Expected redefinition error on block, got: %v", cb.Error())
			}
		}
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
			if strings.Contains(line, "42") && !strings.Contains(line, "{{") {
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

// --- Statement-level error recovery tests (Unit 3) ---

// TestErrorRecovery_DivByZeroThenSuccess tests that a block continues past
// a division-by-zero error and evaluates subsequent independent statements.
func TestErrorRecovery_DivByZeroThenSuccess(t *testing.T) {
	// a = 1/0 fails, b = 10 should succeed
	source := "a = 1 / 0\nb = 10\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	// Should return an error (first error)
	if err == nil {
		t.Fatal("Expected error from division by zero, got nil")
	}

	blocks := doc.GetBlocks()
	if len(blocks) == 0 {
		t.Fatal("Expected at least 1 block")
	}

	cb := blocks[0].Block.(*document.CalcBlock)

	// Results must have exactly len(statements) entries
	results := cb.Results()
	stmts := cb.Statements()
	if len(results) != len(stmts) {
		t.Fatalf("len(results)=%d != len(statements)=%d", len(results), len(stmts))
	}
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// First result is nil (failed)
	if results[0] != nil {
		t.Errorf("Expected nil for failed statement, got %v", results[0])
	}

	// Second result is 10 (succeeded)
	if results[1] == nil {
		t.Fatal("Expected non-nil result for b = 10")
	}
	if results[1].String() != "10" {
		t.Errorf("Expected b = 10, got %q", results[1].String())
	}

	// Block should have error set (first error) and stay dirty
	if cb.Error() == nil {
		t.Error("Expected block.Error() to be non-nil")
	}
	if !cb.IsDirty() {
		t.Error("Expected block to stay dirty when statements have errors")
	}

	// Check diagnostics
	diags := cb.Diagnostics()
	if len(diags) == 0 {
		t.Fatal("Expected at least 1 diagnostic")
	}
	found := false
	for _, d := range diags {
		if d.Code == "eval_error" && d.Severity == "error" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected eval_error diagnostic with error severity")
	}

	// b should be in the environment
	env := eval.GetEnvironment()
	if val, ok := env.Get("b"); !ok || val.String() != "10" {
		t.Errorf("Expected b=10 in environment, got %v (ok=%v)", val, ok)
	}

	// a should be marked as errored in the environment
	if _, errored := env.GetError("a"); !errored {
		t.Error("Expected variable 'a' to be marked as errored")
	}
}

// TestErrorRecovery_CascadingError tests that referencing an errored variable
// produces a cascading_error diagnostic with hint severity.
func TestErrorRecovery_CascadingError(t *testing.T) {
	// a = 1/0 fails, c = a * 2 references errored "a" -> cascading error
	source := "a = 1 / 0\nc = a * 2\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	blocks := doc.GetBlocks()
	cb := blocks[0].Block.(*document.CalcBlock)

	results := cb.Results()
	stmts := cb.Statements()
	if len(results) != len(stmts) {
		t.Fatalf("len(results)=%d != len(statements)=%d", len(results), len(stmts))
	}

	// Both results should be nil (both failed)
	if results[0] != nil {
		t.Errorf("Expected nil for a = 1/0, got %v", results[0])
	}
	if results[1] != nil {
		t.Errorf("Expected nil for c = a*2, got %v", results[1])
	}

	// Check diagnostics
	diags := cb.Diagnostics()
	if len(diags) < 2 {
		t.Fatalf("Expected at least 2 diagnostics, got %d", len(diags))
	}

	// First diagnostic should be eval_error for a = 1/0
	foundEvalError := false
	foundCascading := false
	for _, d := range diags {
		if d.Code == "eval_error" && d.Severity == "error" {
			foundEvalError = true
		}
		if d.Code == diagCodeCascadingError && d.Severity == "hint" {
			foundCascading = true
			if d.Detailed != "a" {
				t.Errorf("Expected cascading_error Detailed='a', got %q", d.Detailed)
			}
		}
	}
	if !foundEvalError {
		t.Error("Expected eval_error diagnostic")
	}
	if !foundCascading {
		t.Error("Expected cascading_error diagnostic with hint severity")
	}

	// Both a and c should be errored
	env := eval.GetEnvironment()
	if _, errored := env.GetError("a"); !errored {
		t.Error("Expected 'a' to be errored")
	}
	if _, errored := env.GetError("c"); !errored {
		t.Error("Expected 'c' to be errored")
	}

	// The error for c should be a CascadingError
	cErr, _ := env.GetError("c")
	var cascErr *interpreter.CascadingError
	if !errors.As(cErr, &cascErr) {
		t.Errorf("Expected CascadingError for 'c', got %T: %v", cErr, cErr)
	} else if cascErr.VarName != "a" {
		t.Errorf("Expected cascading error VarName='a', got %q", cascErr.VarName)
	}
}

// TestErrorRecovery_AllSuccess tests that a block with all successful statements
// behaves exactly as before (no regressions).
func TestErrorRecovery_AllSuccess(t *testing.T) {
	source := "a = 5\nb = a + 10\nc = b * 2\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	blocks := doc.GetBlocks()
	cb := blocks[0].Block.(*document.CalcBlock)

	results := cb.Results()
	stmts := cb.Statements()
	if len(results) != len(stmts) {
		t.Fatalf("len(results)=%d != len(statements)=%d", len(results), len(stmts))
	}
	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	// Verify values
	expected := []string{"5", "15", "30"}
	for i, exp := range expected {
		if results[i] == nil {
			t.Errorf("Result %d is nil, expected %s", i, exp)
			continue
		}
		if results[i].String() != exp {
			t.Errorf("Result %d: expected %s, got %s", i, exp, results[i].String())
		}
	}

	// Block should be clean
	if cb.IsDirty() {
		t.Error("Expected block to be clean after successful evaluation")
	}
	if cb.Error() != nil {
		t.Errorf("Expected no error, got: %v", cb.Error())
	}
	if len(cb.Diagnostics()) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(cb.Diagnostics()))
	}
}

// TestErrorRecovery_AllStatementsFail tests a block where every statement fails.
func TestErrorRecovery_AllStatementsFail(t *testing.T) {
	// a = 1/0, b = 1/0 — both fail independently
	source := "a = 1 / 0\nb = 1 / 0\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	blocks := doc.GetBlocks()
	cb := blocks[0].Block.(*document.CalcBlock)

	results := cb.Results()
	stmts := cb.Statements()
	if len(results) != len(stmts) {
		t.Fatalf("len(results)=%d != len(statements)=%d", len(results), len(stmts))
	}

	// All results should be nil
	for i, r := range results {
		if r != nil {
			t.Errorf("Expected nil for result %d, got %v", i, r)
		}
	}

	// Block should stay dirty
	if !cb.IsDirty() {
		t.Error("Expected block to stay dirty when all statements fail")
	}

	// LastValue should be nil (no successful results)
	if cb.LastValue() != nil {
		t.Errorf("Expected nil LastValue, got %v", cb.LastValue())
	}

	// Should have 2 diagnostics
	diags := cb.Diagnostics()
	if len(diags) != 2 {
		t.Fatalf("Expected 2 diagnostics, got %d", len(diags))
	}
}

// TestErrorRecovery_MultipleErroredVarsCascade tests that a statement
// referencing multiple errored variables produces one cascading error.
func TestErrorRecovery_MultipleErroredVarsCascade(t *testing.T) {
	// a = 1/0, b = 1/0, c = a + b — c depends on two errored vars
	source := "a = 1 / 0\nb = 1 / 0\nc = a + b\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	blocks := doc.GetBlocks()
	cb := blocks[0].Block.(*document.CalcBlock)

	results := cb.Results()
	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	// All nil
	for i, r := range results {
		if r != nil {
			t.Errorf("Expected nil for result %d, got %v", i, r)
		}
	}

	// c should have a cascading_error diagnostic
	diags := cb.Diagnostics()
	foundCascading := false
	for _, d := range diags {
		if d.Code == diagCodeCascadingError {
			foundCascading = true
			// Should name one of the root-cause variables
			if d.Detailed != "a" && d.Detailed != "b" {
				t.Errorf("Expected Detailed to be 'a' or 'b', got %q", d.Detailed)
			}
		}
	}
	if !foundCascading {
		t.Error("Expected cascading_error diagnostic for c = a + b")
	}
}

// TestErrorRecovery_SuccessThenFailInSameBlock tests that a successful
// statement's value persists in the environment even when a later statement fails.
func TestErrorRecovery_SuccessThenFailInSameBlock(t *testing.T) {
	source := "y = 15\nz = 1 / 0\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if err == nil {
		t.Fatal("Expected error from z = 1/0, got nil")
	}

	// y should be in the environment (succeeded)
	env := eval.GetEnvironment()
	if val, ok := env.Get("y"); !ok || val.String() != "15" {
		t.Errorf("Expected y=15 in environment, got %v (ok=%v)", val, ok)
	}

	// z should be errored
	if _, errored := env.GetError("z"); !errored {
		t.Error("Expected 'z' to be errored")
	}
}

// TestErrorRecovery_DiagnosticLineNumbers tests that diagnostics have
// correct Line and DocLine values.
func TestErrorRecovery_DiagnosticLineNumbers(t *testing.T) {
	// Two blocks: first succeeds, second has error on its 2nd statement
	source := "x = 10\n\n\ny = 5\nz = 1 / 0\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Find the block with the error
	for _, node := range doc.GetBlocks() {
		cb, ok := node.Block.(*document.CalcBlock)
		if !ok || cb.Error() == nil {
			continue
		}

		diags := cb.Diagnostics()
		if len(diags) == 0 {
			t.Fatal("Expected diagnostics on errored block")
		}

		// The error diagnostic should have line info
		for _, d := range diags {
			if d.Code == "eval_error" {
				if d.Line == 0 {
					t.Error("Expected non-zero Line on eval_error diagnostic")
				}
				// DocLine should be greater than Line (block-relative) due to preceding blocks
				if d.DocLine == 0 {
					t.Error("Expected non-zero DocLine on eval_error diagnostic")
				}
				return
			}
		}
		t.Error("Expected eval_error diagnostic in errored block")
	}
}

// TestErrorRecovery_ResultsSliceInvariant verifies that
// len(block.Results()) == len(block.Statements()) always holds.
func TestErrorRecovery_ResultsSliceInvariant(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"all success", "a = 1\nb = 2\nc = 3\n"},
		{"first fails", "a = 1 / 0\nb = 10\n"},
		{"last fails", "a = 10\nb = 1 / 0\n"},
		{"middle fails", "a = 10\nb = 1 / 0\nc = 20\n"},
		{"all fail", "a = 1 / 0\nb = 1 / 0\n"},
		{"cascading", "a = 1 / 0\nb = a + 1\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := document.NewDocument(tt.source)
			if err != nil {
				t.Fatalf("NewDocument failed: %v", err)
			}

			eval := NewEvaluator()
			_ = eval.Evaluate(doc) // may or may not error

			for _, node := range doc.GetBlocks() {
				cb, ok := node.Block.(*document.CalcBlock)
				if !ok {
					continue
				}
				results := cb.Results()
				stmts := cb.Statements()
				if len(results) != len(stmts) {
					t.Errorf("Invariant violated: len(results)=%d != len(statements)=%d",
						len(results), len(stmts))
				}
			}
		})
	}
}

// TestErrorRecovery_BlockWithErrorsIsDirty verifies that blocks with
// any error stay dirty and have block.Error() set.
func TestErrorRecovery_BlockWithErrorsIsDirty(t *testing.T) {
	source := "a = 10\nb = 1 / 0\nc = 20\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	_ = eval.Evaluate(doc)

	blocks := doc.GetBlocks()
	cb := blocks[0].Block.(*document.CalcBlock)

	if cb.Error() == nil {
		t.Error("Expected block.Error() to be non-nil")
	}
	if !cb.IsDirty() {
		t.Error("Expected block.IsDirty() == true")
	}

	// But successful results should still be present
	results := cb.Results()
	if results[0] == nil || results[0].String() != "10" {
		t.Errorf("Expected a=10, got %v", results[0])
	}
	if results[1] != nil {
		t.Errorf("Expected nil for b=1/0, got %v", results[1])
	}
	if results[2] == nil || results[2].String() != "20" {
		t.Errorf("Expected c=20, got %v", results[2])
	}
}

// --- Block-level error recovery tests (Unit 4) ---

// TestBlockRecovery_EvaluateContinuesPastErrorBlock tests that Evaluate()
// continues evaluating blocks after a block has errors, and returns ErrPartialEvaluation.
func TestBlockRecovery_EvaluateContinuesPastErrorBlock(t *testing.T) {
	// Block 1: has error (undefined var)
	// Block 2: independent, should succeed
	// Block 3: independent, should succeed
	source := "a = unknown_var + 1\n\n\nb = 10\n\n\nc = 20\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)

	// Should return ErrPartialEvaluation, not nil
	if !errors.Is(err, ErrPartialEvaluation) {
		t.Fatalf("Expected ErrPartialEvaluation, got: %v", err)
	}

	// Verify block 1 has error
	blocks := doc.GetBlocks()
	calcBlocks := make([]*document.CalcBlock, 0)
	for _, node := range blocks {
		if cb, ok := node.Block.(*document.CalcBlock); ok {
			calcBlocks = append(calcBlocks, cb)
		}
	}
	if len(calcBlocks) != 3 {
		t.Fatalf("Expected 3 calc blocks, got %d", len(calcBlocks))
	}

	if calcBlocks[0].Error() == nil {
		t.Error("Block 1 should have an error")
	}

	// Block 2 should succeed
	if calcBlocks[1].Error() != nil {
		t.Errorf("Block 2 should not have error, got: %v", calcBlocks[1].Error())
	}
	if calcBlocks[1].LastValue() == nil || calcBlocks[1].LastValue().String() != "10" {
		t.Errorf("Block 2: expected b=10, got %v", calcBlocks[1].LastValue())
	}

	// Block 3 should succeed
	if calcBlocks[2].Error() != nil {
		t.Errorf("Block 3 should not have error, got: %v", calcBlocks[2].Error())
	}
	if calcBlocks[2].LastValue() == nil || calcBlocks[2].LastValue().String() != "20" {
		t.Errorf("Block 3: expected c=20, got %v", calcBlocks[2].LastValue())
	}
}

// TestBlockRecovery_NoErrorsReturnsNil tests that Evaluate() returns nil when all blocks succeed.
func TestBlockRecovery_NoErrorsReturnsNil(t *testing.T) {
	source := "a = 10\n\n\nb = a + 5\n\n\nc = b * 2\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if err != nil {
		t.Fatalf("Expected nil error for valid document, got: %v", err)
	}
}

// TestBlockRecovery_CascadingAcrossBlocks tests that a block referencing an errored
// variable from a prior block gets cascading_error diagnostics.
func TestBlockRecovery_CascadingAcrossBlocks(t *testing.T) {
	// Block 1: a = 1/0 (div by zero error)
	// Block 2: b = a * 2 (cascading error)
	source := "a = 1 / 0\n\n\nb = a * 2\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if !errors.Is(err, ErrPartialEvaluation) {
		t.Fatalf("Expected ErrPartialEvaluation, got: %v", err)
	}

	// Block 1 should have eval_error
	blocks := doc.GetBlocks()
	calcBlocks := make([]*document.CalcBlock, 0)
	for _, node := range blocks {
		if cb, ok := node.Block.(*document.CalcBlock); ok {
			calcBlocks = append(calcBlocks, cb)
		}
	}
	if len(calcBlocks) != 2 {
		t.Fatalf("Expected 2 calc blocks, got %d", len(calcBlocks))
	}

	if calcBlocks[0].Error() == nil {
		t.Error("Block 1 should have error")
	}

	// Block 2 should also have an error (cascading)
	if calcBlocks[1].Error() == nil {
		t.Error("Block 2 should have cascading error")
	}

	// Block 2's diagnostic should be cascading_error with hint severity
	diags := calcBlocks[1].Diagnostics()
	foundCascading := false
	for _, d := range diags {
		if d.Code == "cascading_error" && d.Severity == "hint" {
			foundCascading = true
		}
	}
	if !foundCascading {
		t.Errorf("Expected cascading_error hint diagnostic on block 2, got: %v", diags)
	}
}

// TestBlockRecovery_AllBlocksHaveErrors tests that ErrPartialEvaluation is returned
// and all blocks have diagnostics when every block fails.
func TestBlockRecovery_AllBlocksHaveErrors(t *testing.T) {
	source := "a = unknown1\n\n\nb = unknown2\n\n\nc = unknown3\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if !errors.Is(err, ErrPartialEvaluation) {
		t.Fatalf("Expected ErrPartialEvaluation, got: %v", err)
	}

	// Every block should have errors
	for _, node := range doc.GetBlocks() {
		if cb, ok := node.Block.(*document.CalcBlock); ok {
			if cb.Error() == nil {
				t.Errorf("Block %s should have error", node.ID[:8])
			}
		}
	}
}

// TestBlockRecovery_EvaluateBlockPass1Recovery tests that EvaluateBlock pass 1
// continues past parse/eval errors in one block to evaluate other blocks.
func TestBlockRecovery_EvaluateBlockPass1Recovery(t *testing.T) {
	source := "a = 1 / 0\n\n\nb = 10\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	// Initial evaluation to set things up
	err = eval.Evaluate(doc)
	if !errors.Is(err, ErrPartialEvaluation) {
		t.Fatalf("Expected ErrPartialEvaluation, got: %v", err)
	}

	// Get block IDs
	blocks := doc.GetBlocks()
	var bBlockID string
	for _, node := range blocks {
		if cb, ok := node.Block.(*document.CalcBlock); ok {
			if strings.Contains(strings.Join(cb.Source(), ""), "b = 10") {
				bBlockID = node.ID
			}
		}
	}

	if bBlockID == "" {
		t.Fatal("Could not find b block")
	}

	// Re-evaluate via EvaluateBlock — should still succeed for b even though a failed
	err = eval.EvaluateBlock(doc, bBlockID)
	if !errors.Is(err, ErrPartialEvaluation) {
		t.Fatalf("Expected ErrPartialEvaluation from EvaluateBlock, got: %v", err)
	}

	// Block b should have a result
	bNode, _ := doc.GetBlock(bBlockID)
	bBlock := bNode.Block.(*document.CalcBlock)
	if bBlock.Error() != nil {
		t.Errorf("Block b should not have error, got: %v", bBlock.Error())
	}
	if bBlock.LastValue() == nil || bBlock.LastValue().String() != "10" {
		t.Errorf("Expected b=10, got %v", bBlock.LastValue())
	}
}

// TestBlockRecovery_EvaluateAffectedBlocksRecovery tests that EvaluateAffectedBlocks
// continues past a block error and evaluates the remaining blocks.
func TestBlockRecovery_EvaluateAffectedBlocksRecovery(t *testing.T) {
	source := "a = 10\n\n\nb = 20\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if err != nil {
		t.Fatalf("Initial Evaluate failed: %v", err)
	}

	// Get block IDs
	blocks := doc.GetBlocks()
	var aBlockID, bBlockID string
	for _, node := range blocks {
		if cb, ok := node.Block.(*document.CalcBlock); ok {
			src := strings.Join(cb.Source(), "")
			if strings.Contains(src, "a = 10") {
				aBlockID = node.ID
			}
			if strings.Contains(src, "b = 20") {
				bBlockID = node.ID
			}
		}
	}

	// Change block a to have an error
	_, err = doc.ReplaceBlockSource(aBlockID, []string{"a = unknown_var"})
	if err != nil {
		t.Fatalf("ReplaceBlockSource failed: %v", err)
	}

	// EvaluateAffectedBlocks for both blocks
	err = eval.EvaluateAffectedBlocks(doc, []string{aBlockID, bBlockID})
	if !errors.Is(err, ErrPartialEvaluation) {
		t.Fatalf("Expected ErrPartialEvaluation, got: %v", err)
	}

	// Block a should have error
	aNode, _ := doc.GetBlock(aBlockID)
	aBlock := aNode.Block.(*document.CalcBlock)
	if aBlock.Error() == nil {
		t.Error("Block a should have error after introducing unknown_var")
	}

	// Block b should still succeed
	bNode, _ := doc.GetBlock(bBlockID)
	bBlock := bNode.Block.(*document.CalcBlock)
	if bBlock.Error() != nil {
		t.Errorf("Block b should not have error, got: %v", bBlock.Error())
	}
}

// TestBlockRecovery_TransformsSkipErroredBlocks tests that applyTransforms
// skips errored blocks but transforms successful ones.
func TestBlockRecovery_TransformsSkipErroredBlocks(t *testing.T) {
	// scale factor 2 with unit_categories including Weight so grams get scaled
	source := "---\nscale:\n  factor: 2\n  unit_categories: [Mass]\n---\na = unknown_var\n\n\nb = 10 grams\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if !errors.Is(err, ErrPartialEvaluation) {
		t.Fatalf("Expected ErrPartialEvaluation, got: %v", err)
	}

	// Find the blocks
	blocks := doc.GetBlocks()
	var aBlock, bBlock *document.CalcBlock
	for _, node := range blocks {
		if cb, ok := node.Block.(*document.CalcBlock); ok {
			src := strings.Join(cb.Source(), "")
			if strings.Contains(src, "unknown_var") {
				aBlock = cb
			}
			if strings.Contains(src, "b = 10 grams") {
				bBlock = cb
			}
		}
	}

	if bBlock == nil {
		t.Fatal("Could not find b block")
	}

	// Errored block should NOT have been transformed (applyTransforms skips errored blocks)
	if aBlock != nil && aBlock.Error() == nil {
		t.Error("Block a should have error")
	}

	// b = 10 grams, scaled by 2 → 20 grams
	results := bBlock.Results()
	if len(results) == 0 {
		t.Fatal("b block has no results")
	}
	val := results[0].String()
	if !strings.Contains(val, "20") {
		t.Errorf("Expected b to be scaled to 20 (grams), got %q", val)
	}
}

// TestBlockRecovery_InterpolationResolveSuccessLeavesErrored tests that
// interpolation resolves {{var}} for successful vars but leaves unresolved
// for errored vars.
func TestBlockRecovery_InterpolationResolveSuccessLeavesErrored(t *testing.T) {
	// Block 1: a = unknown (error)
	// Block 2: b = 42 (success)
	// TextBlock: references both
	source := "a = unknown_var\n\n\nb = 42\n\n\nResult a: {{a}}, Result b: {{b}}\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if !errors.Is(err, ErrPartialEvaluation) {
		t.Fatalf("Expected ErrPartialEvaluation, got: %v", err)
	}

	// Find the TextBlock
	for _, node := range doc.GetBlocks() {
		tb, ok := node.Block.(*document.TextBlock)
		if !ok {
			continue
		}
		interp := tb.InterpolatedSource()
		for _, line := range interp {
			// b should be resolved to "42", a should remain as {{a}}
			if strings.Contains(line, "Result b: 42") {
				if strings.Contains(line, "{{a}}") {
					return // Success
				}
				t.Errorf("Expected {{a}} to remain unresolved, got: %s", line)
				return
			}
		}
	}
	t.Error("expected interpolated text block with b=42")
}

// TestBlockRecovery_SemanticErrorMarksVarsErrored tests that when a block has a semantic
// error, the defined variables are marked as errored so downstream blocks get cascading errors.
func TestBlockRecovery_SemanticErrorMarksVarsErrored(t *testing.T) {
	// Block 1: x = 1, x = 2 (redefinition semantic error within single block)
	// Block 2: y = x + 1 (should get cascading error since x is errored)
	// Note: the semantic checker catches x = 1, x = 2 as redefinition
	source := "x = 1\nx = 2\n\n\ny = x + 1\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if !errors.Is(err, ErrPartialEvaluation) {
		t.Fatalf("Expected ErrPartialEvaluation, got: %v", err)
	}

	// Block 1 should have error (redefinition)
	blocks := doc.GetBlocks()
	calcBlocks := make([]*document.CalcBlock, 0)
	for _, node := range blocks {
		if cb, ok := node.Block.(*document.CalcBlock); ok {
			calcBlocks = append(calcBlocks, cb)
		}
	}
	if len(calcBlocks) < 2 {
		t.Fatalf("Expected at least 2 calc blocks, got %d", len(calcBlocks))
	}

	if calcBlocks[0].Error() == nil {
		t.Error("Block 1 should have redefinition error")
	}

	// Block 2 should have cascading error (x is errored)
	if calcBlocks[1].Error() == nil {
		t.Error("Block 2 should have cascading error from errored x")
	}
	diags := calcBlocks[1].Diagnostics()
	foundCascading := false
	for _, d := range diags {
		if d.Code == "cascading_error" {
			foundCascading = true
		}
	}
	if !foundCascading {
		t.Errorf("Expected cascading_error diagnostic on block 2, got: %v", diags)
	}
}

// --- Golden file integration tests for error recovery ---

// TestGolden_MultiErrorRecovery tests the multi_error_recovery.cm golden file.
// Block 1: a=1/0 (error), b=10 (success). Block 2: c=a*2 (cascading). Block 3: d=42, e=50 (success).
func TestGolden_MultiErrorRecovery(t *testing.T) {
	// Use headings to create separate blocks
	source := "a = 1 / 0\nb = 10\n\n# Block 2\n\nc = a * 2\n\n# Block 3\n\nd = 42\ne = d + 8"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)

	// Should return ErrPartialEvaluation
	if !errors.Is(err, ErrPartialEvaluation) {
		t.Fatalf("expected ErrPartialEvaluation, got: %v", err)
	}

	// Find CalcBlocks only
	var calcBlocks []*document.CalcBlock
	for _, node := range doc.GetBlocks() {
		if cb, ok := node.Block.(*document.CalcBlock); ok {
			calcBlocks = append(calcBlocks, cb)
		}
	}
	if len(calcBlocks) != 3 {
		t.Fatalf("expected 3 calc blocks, got %d", len(calcBlocks))
	}

	// Block 1: mixed success/error
	if calcBlocks[0].Error() == nil {
		t.Error("block 1 should have an error")
	}
	results1 := calcBlocks[0].Results()
	if len(results1) != 2 {
		t.Fatalf("block 1: expected 2 results, got %d", len(results1))
	}
	if results1[0] != nil {
		t.Error("block 1: result[0] should be nil (a = 1/0)")
	}
	if results1[1] == nil {
		t.Error("block 1: result[1] should be non-nil (b = 10)")
	} else if results1[1].String() != "10" {
		t.Errorf("block 1: result[1] = %s, want 10", results1[1].String())
	}

	// Block 2: cascading error
	if calcBlocks[1].Error() == nil {
		t.Error("block 2 should have an error (cascading)")
	}
	diags2 := calcBlocks[1].Diagnostics()
	hasCascading := false
	for _, d := range diags2 {
		if d.Code == "cascading_error" && d.Severity == "hint" {
			hasCascading = true
		}
	}
	if !hasCascading {
		t.Error("block 2 should have a cascading_error diagnostic with hint severity")
	}

	// Block 3: fully successful
	if calcBlocks[2].Error() != nil {
		t.Errorf("block 3 should have no error, got: %v", calcBlocks[2].Error())
	}
	results3 := calcBlocks[2].Results()
	if len(results3) != 2 {
		t.Fatalf("block 3: expected 2 results, got %d", len(results3))
	}
	if results3[0] == nil || results3[0].String() != "42" {
		t.Errorf("block 3: result[0] = %v, want 42", results3[0])
	}
	if results3[1] == nil || results3[1].String() != "50" {
		t.Errorf("block 3: result[1] = %v, want 50", results3[1])
	}
}

// TestGolden_CascadingErrors tests multi-level cascading across blocks.
// Block 1: x=1/0 (root), y=x+10 (cascade), z=y*2 (cascade).
// Block 2: ok=100 (success). Block 3: result=z+ok (cascade).
func TestGolden_CascadingErrors(t *testing.T) {
	// Use headings to create separate blocks
	source := "x = 1 / 0\ny = x + 10\nz = y * 2\n\n# ok block\n\nok = 100\n\n# result block\n\nresult = z + ok"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if !errors.Is(err, ErrPartialEvaluation) {
		t.Fatalf("expected ErrPartialEvaluation, got: %v", err)
	}

	// Find CalcBlocks only
	var calcBlocks []*document.CalcBlock
	for _, node := range doc.GetBlocks() {
		if cb, ok := node.Block.(*document.CalcBlock); ok {
			calcBlocks = append(calcBlocks, cb)
		}
	}
	if len(calcBlocks) != 3 {
		t.Fatalf("expected 3 calc blocks, got %d", len(calcBlocks))
	}

	// Block 1: x errors, y and z cascade
	diags := calcBlocks[0].Diagnostics()
	rootCauseCount := 0
	cascadingCount := 0
	for _, d := range diags {
		if d.Code == "eval_error" {
			rootCauseCount++
		}
		if d.Code == "cascading_error" && d.Severity == "hint" {
			cascadingCount++
		}
	}
	if rootCauseCount != 1 {
		t.Errorf("block 1: expected 1 root-cause error, got %d", rootCauseCount)
	}
	if cascadingCount != 2 {
		t.Errorf("block 1: expected 2 cascading errors (y, z), got %d", cascadingCount)
	}

	// Block 2: ok=100 succeeds
	if calcBlocks[1].Error() != nil {
		t.Errorf("block 2 (ok=100) should succeed, got: %v", calcBlocks[1].Error())
	}
	if calcBlocks[1].LastValue() == nil || calcBlocks[1].LastValue().String() != "100" {
		t.Errorf("block 2: LastValue = %v, want 100", calcBlocks[1].LastValue())
	}

	// Block 3: result=z+ok cascades
	if calcBlocks[2].Error() == nil {
		t.Error("block 3 (result=z+ok) should have cascading error")
	}
	hasCascading := false
	for _, d := range calcBlocks[2].Diagnostics() {
		if d.Code == "cascading_error" {
			hasCascading = true
		}
	}
	if !hasCascading {
		t.Error("block 3 should have cascading_error diagnostic")
	}
}

// TestGolden_MixedSuccessAndErrors tests partial results with div-by-zero mid-document.
func TestGolden_MixedSuccessAndErrors(t *testing.T) {
	source := "price = 25.00\ntax_rate = 0\ntax = price / tax_rate\nsubtotal = price + tax\nshipping = 5.99\ntotal = subtotal + shipping\ndiscount = 10\nfinal = total - discount"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	eval := NewEvaluator()
	err = eval.Evaluate(doc)
	if !errors.Is(err, ErrPartialEvaluation) {
		t.Fatalf("expected ErrPartialEvaluation, got: %v", err)
	}

	blocks := doc.GetBlocks()
	block := blocks[0].Block.(*document.CalcBlock)
	results := block.Results()

	// 8 statements total
	if len(results) != 8 {
		t.Fatalf("expected 8 results, got %d", len(results))
	}

	// price=25.00 succeeds
	if results[0] == nil {
		t.Error("price should succeed")
	}
	// tax_rate=0 succeeds
	if results[1] == nil {
		t.Error("tax_rate should succeed")
	}
	// tax = price / tax_rate fails (div by zero)
	if results[2] != nil {
		t.Error("tax should be nil (division by zero)")
	}
	// subtotal cascades
	if results[3] != nil {
		t.Error("subtotal should be nil (cascading)")
	}
	// shipping=5.99 succeeds
	if results[4] == nil {
		t.Error("shipping should succeed")
	}
	// total cascades
	if results[5] != nil {
		t.Error("total should be nil (cascading)")
	}
	// discount=10 succeeds
	if results[6] == nil {
		t.Error("discount should succeed")
	}
	// final cascades
	if results[7] != nil {
		t.Error("final should be nil (cascading)")
	}

	// Verify diagnostics
	diags := block.Diagnostics()
	rootCount := 0
	cascadeCount := 0
	for _, d := range diags {
		if d.Code == "eval_error" {
			rootCount++
		}
		if d.Code == "cascading_error" {
			cascadeCount++
		}
	}
	if rootCount != 1 {
		t.Errorf("expected 1 root-cause error (tax), got %d", rootCount)
	}
	if cascadeCount != 3 {
		t.Errorf("expected 3 cascading errors (subtotal, total, final), got %d", cascadeCount)
	}
}

// TestMultipleSemanticErrors verifies that ALL semantic errors in a block
// are reported, not just the first one. e.g., "a = 1; a = 2; c = 3; c = 5"
// should report redefinition errors for both 'a' (line 2) and 'c' (line 4).
func TestMultipleSemanticErrors(t *testing.T) {
	source := "a = 1 / 0\na = 2\nc = 3\nc = 5\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	eval := NewEvaluator()
	_ = eval.Evaluate(doc)

	blocks := doc.GetBlocks()
	block := blocks[0].Block.(*document.CalcBlock)

	diags := block.Diagnostics()

	// Should have at least 2 redefinition diagnostics: one for 'a' and one for 'c'
	redefCount := 0
	var redefVars []string
	for _, d := range diags {
		if d.Code == "variable_redefinition" {
			redefCount++
			redefVars = append(redefVars, d.Message)
		}
	}
	if redefCount < 2 {
		t.Errorf("expected at least 2 redefinition diagnostics (for 'a' and 'c'), got %d: %v", redefCount, redefVars)
	}
}
