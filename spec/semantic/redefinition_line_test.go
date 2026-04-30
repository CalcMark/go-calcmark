package semantic

import (
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/parser"
)

// TestRedefinitionDiagnosticLineNumber verifies that when a variable is redefined,
// the diagnostic points to the SECOND assignment (the redefinition), not the first.
func TestRedefinitionDiagnosticLineNumber(t *testing.T) {
	testCases := []struct {
		name           string
		source         string
		expectedLine   int // 1-indexed line number where error should appear
		expectedColumn int // 1-indexed column number
	}{
		{
			name: "Simple redefinition - same block",
			source: `a = 1
a = 2`,
			expectedLine:   2, // Line 2 is the redefinition
			expectedColumn: 1, // Column 1 (start of 'a')
		},
		{
			name: "Redefinition after other variable",
			source: `a = 1
b = 2
a = 3`,
			expectedLine:   3, // Line 3 is the redefinition
			expectedColumn: 1,
		},
		{
			name: "Redefinition with expressions",
			source: `x = 10
y = x + 5
x = 20`,
			expectedLine:   3, // Line 3 is the redefinition
			expectedColumn: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse the source
			nodes, err := parser.Parse(tc.source + "\n")
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			t.Logf("Parsed %d nodes", len(nodes))

			// Run semantic checker
			checker := NewChecker()
			diags := checker.Check(nodes)

			t.Logf("Semantic checker found %d diagnostics", len(diags))
			for i, diag := range diags {
				line := 0
				col := 0
				if diag.Range != nil {
					line = diag.Range.Start.Line
					col = diag.Range.Start.Column
				}
				t.Logf("  [%d] %s: %s (line %d, col %d)", i, diag.Code, diag.Message, line, col)
			}

			// Should have exactly one diagnostic (the redefinition)
			if len(diags) != 1 {
				t.Fatalf("Expected 1 diagnostic, got %d", len(diags))
			}

			diag := diags[0]

			// Check diagnostic code
			if diag.Code != DiagVariableRedefinition {
				t.Errorf("Expected diagnostic code %q, got %q", DiagVariableRedefinition, diag.Code)
			}

			// Check diagnostic range
			if diag.Range == nil {
				t.Fatal("Diagnostic Range is nil")
			}

			// CRITICAL CHECK: The line should be the REDEFINITION line, not the first definition
			if diag.Range.Start.Line != tc.expectedLine {
				t.Errorf("Diagnostic should be on line %d (redefinition), got line %d",
					tc.expectedLine, diag.Range.Start.Line)
			}

			if diag.Range.Start.Column != tc.expectedColumn {
				t.Errorf("Diagnostic should be at column %d, got column %d",
					tc.expectedColumn, diag.Range.Start.Column)
			}
		})
	}
}

// TestMultipleRedefinitions tests that multiple redefinitions are caught.
func TestMultipleRedefinitions(t *testing.T) {
	source := `a = 1
b = 2
a = 3
b = 4`

	nodes, err := parser.Parse(source + "\n")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	checker := NewChecker()
	diags := checker.Check(nodes)

	// Should have 2 redefinition errors (a and b)
	if len(diags) != 2 {
		t.Fatalf("Expected 2 diagnostics, got %d", len(diags))
	}

	// First redefinition should be on line 3 (a = 3)
	if diags[0].Range.Start.Line != 3 {
		t.Errorf("First redefinition should be on line 3, got line %d", diags[0].Range.Start.Line)
	}

	// Second redefinition should be on line 4 (b = 4)
	if diags[1].Range.Start.Line != 4 {
		t.Errorf("Second redefinition should be on line 4, got line %d", diags[1].Range.Start.Line)
	}

	t.Logf("✓ Both redefinition errors correctly point to redefinition lines")
}
