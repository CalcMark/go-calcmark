package document

import (
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

// A block with semantic errors still interprets the statements that
// passed semantic checking (go-calcmark#113). Before, the evaluator
// returned as soon as the checker reported an error, so clean lines had
// no value and runtime errors on them were never discovered.

func evaluateSingleBlock(t *testing.T, source string) *document.CalcBlock {
	t.Helper()
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}
	_ = NewEvaluator().Evaluate(doc)
	for _, node := range doc.GetBlocks() {
		if cb, ok := node.Block.(*document.CalcBlock); ok {
			return cb
		}
	}
	t.Fatal("no calc block")
	return nil
}

func diagnosticCodes(cb *document.CalcBlock) []string {
	var codes []string
	for _, d := range cb.Diagnostics() {
		codes = append(codes, d.Code)
	}
	return codes
}

func TestSemanticRecovery_CleanStatementsStillEvaluate(t *testing.T) {
	// Lines 2 and 4 are redefinitions (semantic). Line 1 is a runtime
	// error. Line 3 is fine and must produce a value.
	cb := evaluateSingleBlock(t, "a = 1 / 0\na = 2\nc = 3\nc = 5\n")

	results := cb.Results()
	if len(results) != 4 {
		t.Fatalf("len(Results()) = %d, want 4 (one slot per statement): %v", len(results), results)
	}
	if results[2] == nil {
		t.Errorf("`c = 3` passed semantic checking but produced no value")
	} else if results[2].String() != "3" {
		t.Errorf("`c = 3` = %q, want 3", results[2].String())
	}
	for i, want := range []bool{true, true, false, true} {
		if (results[i] == nil) != want {
			t.Errorf("results[%d] nil = %v, want %v", i, results[i] == nil, want)
		}
	}

	codes := diagnosticCodes(cb)
	wantRuntime := false
	for _, d := range cb.Diagnostics() {
		if d.Code == "eval_error" && d.Line == 1 {
			wantRuntime = true
		}
	}
	if !wantRuntime {
		t.Errorf("`a = 1 / 0` passed semantic checking but its division by zero was never reported; diagnostics = %v", codes)
	}
	redefs := 0
	for _, c := range codes {
		if c == "variable_redefinition" {
			redefs++
		}
	}
	if redefs != 2 {
		t.Errorf("want 2 variable_redefinition diagnostics, got %d: %v", redefs, codes)
	}
}

func TestSemanticRecovery_ErroredStatementsAreSkippedAndMarked(t *testing.T) {
	// `x` is undefined (semantic). `y` is clean. `z` depends on the
	// skipped statement and must cascade rather than evaluate.
	cb := evaluateSingleBlock(t, "w = x + 1\ny = 10\nz = w * 2\n")

	results := cb.Results()
	if len(results) != 3 {
		t.Fatalf("len(Results()) = %d, want 3", len(results))
	}
	if results[0] != nil {
		t.Errorf("semantically-errored `w = x + 1` must not evaluate, got %v", results[0])
	}
	if results[1] == nil || results[1].String() != "10" {
		t.Errorf("`y = 10` = %v, want 10", results[1])
	}
	if results[2] != nil {
		t.Errorf("`z = w * 2` references a skipped statement and must not evaluate, got %v", results[2])
	}
	codes := diagnosticCodes(cb)
	hasUndefined, hasCascade := false, false
	for _, c := range codes {
		switch c {
		case "undefined_variable":
			hasUndefined = true
		case "cascading_error":
			hasCascade = true
		}
	}
	if !hasUndefined || !hasCascade {
		t.Errorf("want undefined_variable + cascading_error, got %v", codes)
	}
}

func TestSemanticRecovery_BlockStillReportsError(t *testing.T) {
	// Partial evaluation is still an error from the caller's point of
	// view: the block keeps a non-nil Error() and Evaluate returns
	// ErrPartialEvaluation, exactly as for runtime-only failures.
	doc, err := document.NewDocument("a = 1\na = 2\nb = 3\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := NewEvaluator().Evaluate(doc); err == nil {
		t.Error("Evaluate returned nil for a block with a semantic error")
	}
	cb := doc.GetBlocks()[0].Block.(*document.CalcBlock)
	if cb.Error() == nil {
		t.Error("block Error() is nil despite a semantic error")
	}
	if r := cb.Results(); len(r) != 3 || r[2] == nil {
		t.Errorf("`b = 3` should have a value alongside the block error; results = %v", r)
	}
}

func TestSemanticRecovery_DownstreamBlockSeesValue(t *testing.T) {
	// The clean value from an errored block is visible to later blocks.
	doc, err := document.NewDocument("a = 1\na = 2\nb = 3\n\n\nc = b + 1\n")
	if err != nil {
		t.Fatal(err)
	}
	_ = NewEvaluator().Evaluate(doc)
	var blocks []*document.CalcBlock
	for _, n := range doc.GetBlocks() {
		if cb, ok := n.Block.(*document.CalcBlock); ok {
			blocks = append(blocks, cb)
		}
	}
	if len(blocks) != 2 {
		t.Fatalf("want 2 calc blocks, got %d", len(blocks))
	}
	if r := blocks[1].Results(); len(r) != 1 || r[0] == nil || r[0].String() != "4" {
		t.Errorf("`c = b + 1` = %v, want 4", r)
	}
}
