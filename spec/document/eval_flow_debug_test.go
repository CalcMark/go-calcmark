package document_test

// This test lives in package document_test (not document) to break what would
// otherwise be a test-only import cycle: spec/semantic imports spec/document
// (for document.Frontmatter in CheckFrontmatter), and a `package document` test
// importing spec/semantic would close the cycle. The test exercises only
// exported symbols, so the move is safe.

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/semantic"
)

// TestEvalFlowWithTwoBlocks simulates the exact flow that happens when evaluating
// two blocks where the second block redefines a variable from the first.
func TestEvalFlowWithTwoBlocks(t *testing.T) {
	// Create shared environment (like Document does)
	env := interpreter.NewEnvironment()

	t.Log("=== BLOCK 0: a = 3 ===")

	// Parse block 0
	source0 := "a = 3\n"
	nodes0, err := parser.Parse(source0)
	if err != nil {
		t.Fatalf("Parse block 0 failed: %v", err)
	}
	t.Logf("Parsed %d nodes", len(nodes0))

	// Semantic check block 0
	checker0 := semantic.NewChecker()
	// Populate checker with current environment (empty for first block)
	for varName, value := range env.GetAllVariables() {
		checker0.GetEnvironment().Set(varName, value)
	}
	t.Logf("Checker 0 environment has %d variables before check", len(env.GetAllVariables()))

	diags0 := checker0.Check(nodes0)
	t.Logf("Block 0 semantic check: %d diagnostics", len(diags0))
	for _, diag := range diags0 {
		t.Logf("  - %s: %s", diag.Code, diag.Message)
	}

	if len(diags0) > 0 {
		t.Fatal("Block 0 should have no errors")
	}

	// Interpret block 0
	interp0 := interpreter.NewInterpreterWithEnv(env)
	results0, err := interp0.Eval(nodes0)
	if err != nil {
		t.Fatalf("Interpret block 0 failed: %v", err)
	}
	t.Logf("Block 0 produced %d results", len(results0))

	// Check environment after block 0
	t.Logf("Environment after block 0: %d variables", len(env.GetAllVariables()))
	for varName := range env.GetAllVariables() {
		t.Logf("  - %s", varName)
	}

	t.Log("\n=== BLOCK 1: a = 3 (should be redefinition error) ===")

	// Parse block 1
	source1 := "a = 3\n"
	nodes1, err := parser.Parse(source1)
	if err != nil {
		t.Fatalf("Parse block 1 failed: %v", err)
	}
	t.Logf("Parsed %d nodes", len(nodes1))

	// Semantic check block 1
	checker1 := semantic.NewChecker()
	// Populate checker with current environment (should now have 'a')
	t.Logf("Populating checker 1 with %d environment variables:", len(env.GetAllVariables()))
	for varName, value := range env.GetAllVariables() {
		t.Logf("  - Setting %s in checker", varName)
		checker1.GetEnvironment().Set(varName, value)
	}

	diags1 := checker1.Check(nodes1)
	t.Logf("Block 1 semantic check: %d diagnostics", len(diags1))
	for _, diag := range diags1 {
		line := 0
		if diag.Range != nil {
			line = diag.Range.Start.Line
		}
		t.Logf("  - %s: %s (line %d)", diag.Code, diag.Message, line)
	}

	// CRITICAL CHECK: Should have a redefinition diagnostic
	if len(diags1) == 0 {
		t.Error("Block 1 should have a redefinition diagnostic")
	} else {
		hasRedef := false
		for _, diag := range diags1 {
			if diag.Code == "variable_redefinition" {
				hasRedef = true
				t.Logf("✓ Found redefinition diagnostic")
			}
		}
		if !hasRedef {
			t.Error("Diagnostics don't include redefinition error")
		}
	}
}
