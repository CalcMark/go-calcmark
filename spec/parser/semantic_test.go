package parser

import (
	"testing"
)

// TestVariableRedefinitionParsesSuccessfully tests that the parser successfully
// parses code with variable redefinition (semantic validation happens later).
// CalcMark semantic rule: A variable can only be defined once in a document.
// This is enforced at the semantic/document level, not during parsing.
func TestVariableRedefinitionParsesSuccessfully(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "Simple redefinition",
			source: "a = 2\na = 3",
		},
		{
			name:   "Redefinition with calculation",
			source: "x = 10\ny = x * 2\nx = 20",
		},
		{
			name:   "Redefinition in same block",
			source: "total = 100\ntotal = 200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parser should successfully parse the AST
			// Semantic validation (redefinition check) happens at document level
			nodes, err := Parse(tt.source)

			if err != nil {
				t.Errorf("Expected successful parse, got error: %v", err)
				t.Logf("Source: %q", tt.source)
			}

			if len(nodes) == 0 {
				t.Error("Expected non-empty AST")
			}
		})
	}
}

// TestVariableDefinitionAllowed tests that single variable definitions are allowed.
func TestVariableDefinitionAllowed(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "Single definition",
			source: "a = 2",
		},
		{
			name:   "Multiple different variables",
			source: "a = 2\nb = 3\nc = 4",
		},
		{
			name:   "Variable used in expression",
			source: "x = 10\ny = x * 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.source)
			if err != nil {
				t.Errorf("Expected no error for valid definitions, got: %v", err)
			}
		})
	}
}
