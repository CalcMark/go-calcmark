package lexer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// This test ensures SYNTAX_HIGHLIGHTER_SPEC.json stays synchronized with the actual implementation
// The JSON spec is a first-class deliverable used by TypeScript clients for syntax highlighting

type SyntaxSpec struct {
	Version string `json:"version"`
	Tokens  struct {
		Keywords struct {
			LogicalOperators struct {
				Tokens []string `json:"tokens"`
			} `json:"logicalOperators"`
			ControlFlow struct {
				Tokens []string `json:"tokens"`
			} `json:"controlFlow"`
			Functions struct {
				Canonical map[string]interface{} `json:"canonical"`
			} `json:"functions"`
			MultiTokenFunctions struct {
				Tokens []struct {
					Pattern   string `json:"pattern"`
					Canonical string `json:"canonical"`
				} `json:"tokens"`
			} `json:"multiTokenFunctions"`
		} `json:"keywords"`
		Operators struct {
			Arithmetic struct {
				Tokens []struct {
					Symbol  string   `json:"symbol"`
					Aliases []string `json:"aliases,omitempty"`
				} `json:"tokens"`
			} `json:"arithmetic"`
			Comparison struct {
				Tokens []struct {
					Symbol string `json:"symbol"`
				} `json:"tokens"`
			} `json:"comparison"`
			Assignment struct {
				Tokens []struct {
					Symbol string `json:"symbol"`
				} `json:"tokens"`
			} `json:"assignment"`
		} `json:"operators"`
		Literals struct {
			Boolean struct {
				TrueValues  []string `json:"trueValues"`
				FalseValues []string `json:"falseValues"`
			} `json:"boolean"`
		} `json:"literals"`
	} `json:"tokens"`
	BreakingChanges map[string][]interface{} `json:"breakingChanges"`
}

func TestSyntaxSpecExists(t *testing.T) {
	specPath := filepath.Join("..", "spec", "SYNTAX_HIGHLIGHTER_SPEC.json")
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		t.Fatal("spec/SYNTAX_HIGHLIGHTER_SPEC.json not found - this is a required deliverable")
	}
}

func TestSyntaxSpecValidJSON(t *testing.T) {
	specPath := filepath.Join("..", "spec", "SYNTAX_HIGHLIGHTER_SPEC.json")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("Failed to read spec/SYNTAX_HIGHLIGHTER_SPEC.json: %v", err)
	}

	var spec SyntaxSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("spec/SYNTAX_HIGHLIGHTER_SPEC.json is not valid JSON: %v", err)
	}

	// Check required fields
	if spec.Version == "" {
		t.Error("Missing version field")
	}
}

func TestSyntaxSpecContainsAllReservedKeywords(t *testing.T) {
	specPath := filepath.Join("..", "spec", "SYNTAX_HIGHLIGHTER_SPEC.json")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("Failed to read spec: %v", err)
	}

	var spec SyntaxSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("Failed to parse spec: %v", err)
	}

	// Build list of all keywords from spec
	specKeywords := make(map[string]bool)
	for _, kw := range spec.Tokens.Keywords.LogicalOperators.Tokens {
		specKeywords[strings.ToLower(kw)] = true
	}
	for _, kw := range spec.Tokens.Keywords.ControlFlow.Tokens {
		specKeywords[strings.ToLower(kw)] = true
	}
	for kw := range spec.Tokens.Keywords.Functions.Canonical {
		specKeywords[strings.ToLower(kw)] = true
	}

	// Check that all keywords in reservedKeywords map are in the spec
	for keyword := range reservedKeywords {
		if !specKeywords[keyword] {
			t.Errorf("Keyword '%s' is in lexer but missing from spec/SYNTAX_HIGHLIGHTER_SPEC.json", keyword)
		}
	}

	// Check that all keywords in spec are in reservedKeywords map
	for keyword := range specKeywords {
		if _, exists := reservedKeywords[keyword]; !exists {
			t.Errorf("Keyword '%s' is in spec/SYNTAX_HIGHLIGHTER_SPEC.json but not in lexer", keyword)
		}
	}
}

func TestSyntaxSpecContainsMultiTokenFunctions(t *testing.T) {
	specPath := filepath.Join("..", "spec", "SYNTAX_HIGHLIGHTER_SPEC.json")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("Failed to read spec: %v", err)
	}

	var spec SyntaxSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("Failed to parse spec: %v", err)
	}

	// Verify multi-token functions are documented
	expectedPatterns := map[string]string{
		"average of":      "avg",
		"square root of":  "sqrt",
	}

	foundPatterns := make(map[string]string)
	for _, mtf := range spec.Tokens.Keywords.MultiTokenFunctions.Tokens {
		foundPatterns[mtf.Pattern] = mtf.Canonical
	}

	for pattern, canonical := range expectedPatterns {
		if foundCanonical, exists := foundPatterns[pattern]; !exists {
			t.Errorf("Multi-token function '%s' missing from spec", pattern)
		} else if foundCanonical != canonical {
			t.Errorf("Multi-token function '%s' maps to '%s' in spec, expected '%s'",
				pattern, foundCanonical, canonical)
		}
	}
}

func TestSyntaxSpecContainsBooleanValues(t *testing.T) {
	specPath := filepath.Join("..", "spec", "SYNTAX_HIGHLIGHTER_SPEC.json")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("Failed to read spec: %v", err)
	}

	var spec SyntaxSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("Failed to parse spec: %v", err)
	}

	// Check that all boolean keywords from lexer are in spec
	allSpecBooleans := make(map[string]bool)
	for _, v := range spec.Tokens.Literals.Boolean.TrueValues {
		allSpecBooleans[strings.ToLower(v)] = true
	}
	for _, v := range spec.Tokens.Literals.Boolean.FalseValues {
		allSpecBooleans[strings.ToLower(v)] = true
	}

	for keyword := range booleanKeywords {
		if !allSpecBooleans[keyword] {
			t.Errorf("Boolean keyword '%s' is in lexer but missing from spec/SYNTAX_HIGHLIGHTER_SPEC.json", keyword)
		}
	}
}

func TestSyntaxSpecContainsOperators(t *testing.T) {
	specPath := filepath.Join("..", "spec", "SYNTAX_HIGHLIGHTER_SPEC.json")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("Failed to read spec: %v", err)
	}

	var spec SyntaxSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("Failed to parse spec: %v", err)
	}

	// Verify key operators are documented
	expectedOperators := []string{"+", "-", "*", "/", "%", "^", "=", "==", "!=", "<", ">", "<=", ">="}

	allOperators := make(map[string]bool)
	for _, op := range spec.Tokens.Operators.Arithmetic.Tokens {
		allOperators[op.Symbol] = true
		for _, alias := range op.Aliases {
			allOperators[alias] = true
		}
	}
	for _, op := range spec.Tokens.Operators.Comparison.Tokens {
		allOperators[op.Symbol] = true
	}
	for _, op := range spec.Tokens.Operators.Assignment.Tokens {
		allOperators[op.Symbol] = true
	}

	for _, op := range expectedOperators {
		if !allOperators[op] {
			t.Errorf("Operator '%s' missing from spec/SYNTAX_HIGHLIGHTER_SPEC.json", op)
		}
	}
}

func TestSyntaxSpecHasBreakingChanges(t *testing.T) {
	specPath := filepath.Join("..", "spec", "SYNTAX_HIGHLIGHTER_SPEC.json")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("Failed to read spec: %v", err)
	}

	var spec SyntaxSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("Failed to parse spec: %v", err)
	}

	// Verify v1.0.0 breaking changes are documented
	if spec.BreakingChanges == nil {
		t.Error("breakingChanges section missing from spec")
		return
	}

	if _, exists := spec.BreakingChanges["v1.0.0"]; !exists {
		t.Error("v1.0.0 breaking changes not documented in spec")
	}
}

// TestSyntaxSpecREADMEExists ensures the integration guide exists
func TestSyntaxSpecREADMEExists(t *testing.T) {
	readmePath := filepath.Join("..", "spec", "SYNTAX_HIGHLIGHTER_README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		t.Fatal("spec/SYNTAX_HIGHLIGHTER_README.md not found - this explains how to use the spec")
	}
}
