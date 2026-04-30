package document

import (
	"slices"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
	"github.com/CalcMark/go-calcmark/v2/spec/parser"
)

// TestDependencyExtraction validates that we correctly extract dependencies.
func TestDependencyExtraction(t *testing.T) {
	source := "y = x + 5\n"

	// Parse manually
	nodes, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	t.Logf("Parsed %d nodes from '%s'", len(nodes), source)
	for i, node := range nodes {
		t.Logf("  Node %d: %T = %v", i, node, node)
	}

	// Now test analyzer
	block := NewCalcBlock([]string{source})
	analyzer := NewDependencyAnalyzer()

	err = analyzer.AnalyzeBlock(block)
	if err != nil {
		t.Fatalf("AnalyzeBlock failed: %v", err)
	}

	t.Logf("Variables defined: %v", block.Variables())
	t.Logf("Dependencies: %v", block.Dependencies())

	// Should define y
	if len(block.Variables()) != 1 || block.Variables()[0] != "y" {
		t.Errorf("Expected Variables: [y], got %v", block.Variables())
	}

	// Should depend on x
	if len(block.Dependencies()) != 1 || block.Dependencies()[0] != "x" {
		t.Errorf("Expected Dependencies: [x], got %v", block.Dependencies())
	}
}

// TestVariableReassignmentNoDuplicates verifies that reassigning a variable
// within the same block does not produce duplicate entries in Variables().
func TestVariableReassignmentNoDuplicates(t *testing.T) {
	tests := []struct {
		name         string
		source       []string
		wantVars     []string
		wantVarCount int
	}{
		{
			name:         "single assignment",
			source:       []string{"x = 5"},
			wantVars:     []string{"x"},
			wantVarCount: 1,
		},
		{
			name:         "reassignment same variable",
			source:       []string{"x = 5", "x = 10"},
			wantVars:     []string{"x"},
			wantVarCount: 1,
		},
		{
			name:         "two different variables",
			source:       []string{"x = 5", "y = 10"},
			wantVars:     []string{"x", "y"},
			wantVarCount: 2,
		},
		{
			name:         "reassignment with other variables",
			source:       []string{"x = 5", "y = 10", "x = 15"},
			wantVars:     []string{"x", "y"},
			wantVarCount: 2,
		},
		{
			name:         "multiple reassignments",
			source:       []string{"x = 1", "x = 2", "x = 3", "x = 4"},
			wantVars:     []string{"x"},
			wantVarCount: 1,
		},
	}

	analyzer := NewDependencyAnalyzer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := NewCalcBlock(tt.source)
			err := analyzer.AnalyzeBlock(block)
			if err != nil {
				t.Fatalf("AnalyzeBlock failed: %v", err)
			}

			vars := block.Variables()

			// Check count
			if len(vars) != tt.wantVarCount {
				t.Errorf("Expected %d variables, got %d: %v", tt.wantVarCount, len(vars), vars)
			}

			// Check all expected variables are present
			for _, wantVar := range tt.wantVars {
				if !slices.Contains(vars, wantVar) {
					t.Errorf("Expected variable %q not found in %v", wantVar, vars)
				}
			}

			// Check no duplicates
			seen := make(map[string]bool)
			for _, v := range vars {
				if seen[v] {
					t.Errorf("Variable %q appears more than once in %v", v, vars)
				}
				seen[v] = true
			}
		})
	}
}

// TestExtractStatementReferences validates per-statement identifier extraction.
// Time complexity: O(n) where n is AST node count — single recursive walk.
func TestExtractStatementReferences(t *testing.T) {
	tests := []struct {
		name string
		src  string // single statement
		want []string
	}{
		{
			name: "simple expression with variable",
			src:  "x + 1\n",
			want: []string{"x"},
		},
		{
			name: "assignment returns only RHS refs",
			src:  "y = x + z\n",
			want: []string{"x", "z"},
		},
		{
			name: "no refs in bare number",
			src:  "42\n",
			want: []string{},
		},
		{
			name: "function call args",
			src:  "avg(a, b)\n",
			want: []string{"a", "b"},
		},
		{
			name: "currency literal has no identifier",
			src:  "1000 EUR - 250 EUR\n",
			want: []string{},
		},
		{
			name: "unit quantity has no identifier",
			src:  "5 kg\n",
			want: []string{},
		},
		{
			name: "nil node returns empty",
			src:  "", // handled as nil below
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node ast.Node
			if tt.src != "" {
				nodes, err := parser.Parse(tt.src)
				if err != nil {
					t.Fatalf("Parse(%q) failed: %v", tt.src, err)
				}
				if len(nodes) != 1 {
					t.Fatalf("expected 1 node, got %d", len(nodes))
				}
				node = nodes[0]
			}

			got := ExtractStatementReferences(node)

			if len(tt.want) == 0 {
				if len(got) != 0 {
					t.Errorf("want empty, got %v", got)
				}
				return
			}

			if !slices.Equal(got, tt.want) {
				t.Errorf("want %v, got %v", tt.want, got)
			}
		})
	}
}
