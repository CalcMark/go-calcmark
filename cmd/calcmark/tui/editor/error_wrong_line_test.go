package editor

import (
	"strings"
	"testing"

	impldoc "github.com/CalcMark/go-calcmark/v2/impl/document"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

// TestErrorAppearsOnWrongLine tests if an error about variable 'b'
// incorrectly appears on the line with variable 'a'.
func TestErrorAppearsOnWrongLine(t *testing.T) {
	// Create a document with actual redefinition to see if line numbers work
	source := `a = 3

b = 5

b = 6`

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := impldoc.NewEvaluator()
	evalErr := eval.Evaluate(doc)
	// This SHOULD have an error (b is redefined)
	if evalErr == nil {
		t.Error("Expected redefinition error, got nil")
	} else {
		t.Logf("Got expected error: %v", evalErr)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	results := m.GetLineResults()
	t.Logf("Got %d line results:", len(results))
	for i, r := range results {
		if r.Error != "" {
			t.Logf("  Line %d: %q -> ERROR: %s", i, r.Source, r.Error)
		} else if r.IsCalc {
			t.Logf("  Line %d: %q -> %s", i, r.Source, r.Value)
		} else {
			t.Logf("  Line %d: %q (empty)", i, r.Source)
		}
	}

	// The error should be on line 4 (b = 6), NOT line 0 (a = 3)
	if results[0].Error != "" {
		if strings.Contains(results[0].Error, "b") || strings.Contains(results[0].Error, "redefinition") {
			t.Errorf("ERROR APPEARS ON WRONG LINE: Line 0 (a = 3) has error about 'b': %s", results[0].Error)
		} else {
			t.Errorf("Line 0 (a = 3) should have no error, got: %s", results[0].Error)
		}
	}

	// Line 4 should have the redefinition error
	if len(results) > 4 {
		if results[4].Error == "" {
			t.Error("Line 4 (b = 6) should have redefinition error, but has none")
		} else if !strings.Contains(results[4].Error, "b") && !strings.Contains(results[4].Error, "redefinition") {
			t.Errorf("Line 4 error doesn't mention 'b' or 'redefinition': %s", results[4].Error)
		} else {
			t.Logf("✓ Error correctly appears on line 4")
		}
	}
}

// TestSingleVariableNoError verifies that a single assignment doesn't cause errors.
func TestSingleVariableNoError(t *testing.T) {
	source := `a = 3

b = 6`

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	eval := impldoc.NewEvaluator()
	evalErr := eval.Evaluate(doc)
	if evalErr != nil {
		t.Errorf("Document should have no errors, got: %v", evalErr)
	}

	m := New(doc)
	results := m.GetLineResults()

	// Check NO line has errors
	for i, r := range results {
		if r.Error != "" {
			t.Errorf("Line %d should have no error, got: %s", i, r.Error)
		}
	}
}
