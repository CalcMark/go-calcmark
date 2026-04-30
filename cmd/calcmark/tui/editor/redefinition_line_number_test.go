package editor

import (
	"strings"
	"testing"

	impldoc "github.com/CalcMark/go-calcmark/v2/impl/document"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

// TestRedefinitionErrorAppearsOnCorrectLine tests that redefinition errors
// appear on the REDEFINITION line (the second assignment), not the first.
// This is a regression test for the bug shown in the screenshot where
// the error appeared on line 3 instead of line 4.
func TestRedefinitionErrorAppearsOnCorrectLine(t *testing.T) {
	testCases := []struct {
		name              string
		source            string
		firstDefLine      int // 0-indexed line number of first definition
		redefLine         int // 0-indexed line number of redefinition (should have error)
		expectedErrorLine int // The line where we expect to see the error in LineResults
	}{
		{
			name: "Same block redefinition",
			source: `a = 1
b = 2
a = 3`,
			firstDefLine:      0, // Line 0: a = 1
			redefLine:         2, // Line 2: a = 3 (REDEF - error should be HERE)
			expectedErrorLine: 2,
		},
		{
			name: "Redefinition with empty lines (user's screenshot case)",
			source: `a = 3


a = 3`,
			firstDefLine:      0, // Line 0: a = 3
			redefLine:         3, // Line 3: a = 3 (REDEF - error should be HERE)
			expectedErrorLine: 3, // NOT line 0!
		},
		{
			name: "Redefinition with empty line in between",
			source: `a = 2

b = a * 2

a = 3`,
			firstDefLine:      0, // Line 0: a = 2
			redefLine:         4, // Line 4: a = 3 (REDEF - error should be HERE)
			expectedErrorLine: 4,
		},
		{
			name: "Multiple variables then redefinition",
			source: `a = 1
b = 2
c = 3
a = 4`,
			firstDefLine:      0, // Line 0: a = 1
			redefLine:         3, // Line 3: a = 4 (REDEF - error should be HERE)
			expectedErrorLine: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing: %s", tc.name)
			t.Logf("First definition on line %d, redefinition on line %d", tc.firstDefLine, tc.redefLine)

			// Parse the document - this will succeed at parsing
			doc, err := document.NewDocument(tc.source)
			if err != nil {
				t.Fatalf("NewDocument failed: %v", err)
			}

			if doc == nil {
				t.Fatal("Document is nil")
			}

			// Evaluate the document - this is when semantic checking happens and diagnostics are generated
			eval := impldoc.NewEvaluator()
			evalErr := eval.Evaluate(doc)
			if evalErr != nil {
				t.Logf("Evaluate returned error (expected for redefinition): %v", evalErr)
			}

			// Create the TUI model
			m := New(doc)
			m.width = 80
			m.height = 24

			// Get line results
			results := m.GetLineResults()

			t.Logf("Got %d line results:", len(results))
			for i, r := range results {
				hasError := ""
				if r.Error != "" {
					hasError = " [ERROR: " + r.Error + "]"
				}
				t.Logf("  Line %d: %q%s", i, r.Source, hasError)
			}

			// CRITICAL CHECK: The error should appear on the REDEF line, not the first def line
			if tc.expectedErrorLine >= len(results) {
				t.Fatalf("Expected error line %d is out of bounds (have %d results)",
					tc.expectedErrorLine, len(results))
			}

			// Check that the redefinition line has an error
			if results[tc.expectedErrorLine].Error == "" {
				t.Errorf("Expected error on line %d (redefinition), but no error found", tc.expectedErrorLine)
			} else if !strings.Contains(results[tc.expectedErrorLine].Error, "redefinition") &&
				!strings.Contains(results[tc.expectedErrorLine].Error, "already defined") {
				t.Errorf("Error on line %d doesn't mention redefinition: %q",
					tc.expectedErrorLine, results[tc.expectedErrorLine].Error)
			} else {
				t.Logf("✓ Error correctly appears on line %d: %q",
					tc.expectedErrorLine, results[tc.expectedErrorLine].Error)
			}

			// Check that the first definition line does NOT have an error
			if tc.firstDefLine < len(results) {
				if results[tc.firstDefLine].Error != "" {
					t.Errorf("Line %d (first definition) should NOT have error, but has: %q",
						tc.firstDefLine, results[tc.firstDefLine].Error)
				} else {
					t.Logf("✓ First definition line %d correctly has no error", tc.firstDefLine)
				}
			}
		})
	}
}
