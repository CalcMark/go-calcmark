package editor

import (
	"testing"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config"
	impldoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
)

func init() {
	config.Load()
}

// TestFunctionResultDisplay verifies that function calls display results in preview.
// This is a regression test for the bug where avg(2,4,4) didn't show results.
func TestFunctionResultDisplay(t *testing.T) {
	// Create document with anonymous function call (no variable assignment)
	// Note: CalcMark syntax is "avg(2,4,4)" not "= avg(2,4,4)"
	doc, err := document.NewDocument("avg(2,4,4)\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Evaluate the document
	eval := impldoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate document: %v", err)
	}

	// Create editor model
	m := New(doc)

	// Get line results
	results := m.GetLineResults()

	t.Logf("Got %d results", len(results))
	for i, r := range results {
		t.Logf("Result[%d]: LineNum=%d Source=%q IsCalc=%v VarName=%q Value=%q Error=%q",
			i, r.LineNum, r.Source, r.IsCalc, r.VarName, r.Value, r.Error)
	}

	if len(results) == 0 {
		t.Fatal("Expected at least 1 result, got 0")
	}

	// Find the result for our calculation
	var found bool
	for _, r := range results {
		if r.Source == "avg(2,4,4)" {
			found = true
			if r.Value == "" && r.Error == "" {
				t.Errorf("Expected value for anonymous function call, got empty")
			}
			if r.Value != "" {
				t.Logf("SUCCESS: Function result is %q", r.Value)
			}
			break
		}
	}

	if !found {
		t.Error("Did not find result for 'avg(2,4,4)'")
	}
}

// TestAssignedFunctionResultDisplay tests that assigned function calls work too.
func TestAssignedFunctionResultDisplay(t *testing.T) {
	doc, err := document.NewDocument("result = avg(2,4,4)\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := impldoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate document: %v", err)
	}

	m := New(doc)
	results := m.GetLineResults()

	t.Logf("Got %d results", len(results))
	for i, r := range results {
		t.Logf("Result[%d]: LineNum=%d Source=%q IsCalc=%v VarName=%q Value=%q Error=%q",
			i, r.LineNum, r.Source, r.IsCalc, r.VarName, r.Value, r.Error)
	}

	if len(results) == 0 {
		t.Fatal("Expected at least 1 result, got 0")
	}

	// Find the result
	var found bool
	for _, r := range results {
		if r.VarName == "result" {
			found = true
			if r.Value == "" && r.Error == "" {
				t.Errorf("Expected value for assigned function call, got empty")
			}
			if r.Value != "" {
				t.Logf("SUCCESS: result = %q", r.Value)
			}
			break
		}
	}

	if !found {
		t.Error("Did not find result with VarName 'result'")
	}
}

// TestSqrtFunctionResultDisplay tests that sqrt function displays results.
func TestSqrtFunctionResultDisplay(t *testing.T) {
	doc, err := document.NewDocument("sqrt(16)\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := impldoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate document: %v", err)
	}

	m := New(doc)
	results := m.GetLineResults()

	if len(results) == 0 {
		t.Fatal("Expected at least 1 result, got 0")
	}

	for _, r := range results {
		if r.Source == "sqrt(16)" {
			if r.Value == "" && r.Error == "" {
				t.Errorf("Expected value for sqrt(16), got empty")
			}
			if r.Value != "" {
				t.Logf("SUCCESS: sqrt(16) = %q", r.Value)
			}
			if !r.IsCalc {
				t.Errorf("Expected IsCalc=true for sqrt(16)")
			}
			return
		}
	}

	t.Error("Did not find result for 'sqrt(16)'")
}

// TestAnonymousExpressionResultDisplay tests that bare expressions display results.
func TestAnonymousExpressionResultDisplay(t *testing.T) {
	doc, err := document.NewDocument("2 + 2\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := impldoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate document: %v", err)
	}

	m := New(doc)
	results := m.GetLineResults()

	if len(results) == 0 {
		t.Fatal("Expected at least 1 result, got 0")
	}

	for _, r := range results {
		if r.Source == "2 + 2" {
			if r.Value == "" && r.Error == "" {
				t.Errorf("Expected value for '2 + 2', got empty")
			}
			if r.Value != "" {
				t.Logf("SUCCESS: 2 + 2 = %q", r.Value)
			}
			if !r.IsCalc {
				t.Errorf("Expected IsCalc=true for '2 + 2'")
			}
			return
		}
	}

	t.Error("Did not find result for '2 + 2'")
}
