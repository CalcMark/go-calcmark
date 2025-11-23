package document

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
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
