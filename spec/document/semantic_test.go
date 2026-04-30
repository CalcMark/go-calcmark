package document_test

import (
	"testing"

	impldoc "github.com/CalcMark/go-calcmark/v2/impl/document"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

// TestVariableRedefinitionRejectedInDocument tests that the document
// rejects variable redefinition through NewDocument.
// CalcMark semantic rule: A variable can only be defined once in a document.
func TestVariableRedefinitionRejectedInDocument(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "Simple redefinition",
			source: "a = 2\na = 3\n",
		},
		{
			name:   "Redefinition with calculation",
			source: "x = 10\ny = x * 2\nx = 20\n",
		},
		{
			name:   "Redefinition across blocks",
			source: "total = 100\n\ntotal = 200\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create document and evaluate - should return error for redefinition
			doc, err := document.NewDocument(tt.source)
			if err != nil {
				t.Fatalf("Failed to create document: %v", err)
			}

			// Evaluate the document - this is where semantic errors are caught
			eval := impldoc.NewEvaluator()
			err = eval.Evaluate(doc)
			if err == nil {
				t.Errorf("Expected error for variable redefinition, got nil")
				t.Logf("Source: %q", tt.source)
			}
		})
	}
}

// TestVariableDefinitionAllowedInDocument tests that single variable definitions work.
func TestVariableDefinitionAllowedInDocument(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "Single definition",
			source: "a = 2\n",
		},
		{
			name:   "Multiple different variables",
			source: "a = 2\nb = 3\nc = 4\n",
		},
		{
			name:   "Variable used in expression",
			source: "x = 10\ny = x * 2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := document.NewDocument(tt.source)
			if err != nil {
				t.Errorf("Failed to create document: %v", err)
				return
			}

			// Evaluate to ensure it works without errors
			eval := impldoc.NewEvaluator()
			err = eval.Evaluate(doc)
			if err != nil {
				t.Errorf("Expected no error for valid definitions, got: %v", err)
			}
		})
	}
}
