package parser_test

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestSecurityNestingDepthLimit tests that deeply nested expressions are rejected
func TestSecurityNestingDepthLimit(t *testing.T) {
	// Create expression with 150 nested parentheses (exceeds limit of 100)
	deep := strings.Repeat("(", 150) + "1" + strings.Repeat(")", 150) + "\n"

	_, err := parser.Parse(deep)
	if err == nil {
		t.Fatal("expected error for deeply nested expression, got nil")
	}

	if !strings.Contains(err.Error(), "nesting depth exceeds security limit") {
		t.Errorf("expected nesting depth error, got: %v", err)
	}
}

// TestSecurityNestingDepthOK tests that expressions within limit are accepted
func TestSecurityNestingDepthOK(t *testing.T) {
	// Create expression with 50 nested parentheses (within limit of 100)
	ok := strings.Repeat("(", 50) + "1" + strings.Repeat(")", 50) + "\n"

	_, err := parser.Parse(ok)
	if err != nil {
		t.Fatalf("unexpected error for moderately nested expression: %v", err)
	}
}

// TestSecurityTokenCountLimit tests that token bombs are rejected
func TestSecurityTokenCountLimit(t *testing.T) {
	// Create expression with >10,000 tokens
	// Format: x1+x2+x3+...+x15000
	var b strings.Builder
	for i := range 15000 {
		if i > 0 {
			b.WriteString("+")
		}
		b.WriteString("x")
	}
	b.WriteString("\n")

	_, err := parser.Parse(b.String())
	if err == nil {
		t.Fatal("expected error for token bomb, got nil")
	}

	if !strings.Contains(err.Error(), "token count exceeds security limit") {
		t.Errorf("expected token count error, got: %v", err)
	}
}

// TestSecurityTokenCountOK tests that normal token counts are accepted
func TestSecurityTokenCountOK(t *testing.T) {
	// Create expression with ~5000 tokens (within limit)
	var b strings.Builder
	for i := range 2500 {
		if i > 0 {
			b.WriteString("+")
		}
		b.WriteString("1")
	}
	b.WriteString("\n")

	_, err := parser.Parse(b.String())
	if err != nil {
		t.Fatalf("unexpected error for normal token count: %v", err)
	}
}

// TestSecurityUnaryDepth tests that deeply nested unary operators are rejected
func TestSecurityUnaryDepth(t *testing.T) {
	// Create expression with 150 unary minus operators: ------...-----5
	deep := strings.Repeat("-", 150) + "5\n"

	_, err := parser.Parse(deep)
	if err == nil {
		t.Fatal("expected error for deeply nested unary operators, got nil")
	}

	if !strings.Contains(err.Error(), "nesting depth exceeds security limit") {
		t.Errorf("expected nesting depth error, got: %v", err)
	}
}

// TestSecurityCombinedDepth tests that combined nesting is tracked
func TestSecurityCombinedDepth(t *testing.T) {
	// Use deeply nested parentheses within function: avg((((  ... (1) ...))))
	// 120 levels of parentheses should exceed the limit
	var b strings.Builder
	b.WriteString("avg(")
	for range 120 {
		b.WriteString("(")
	}
	b.WriteString("1")
	for range 120 {
		b.WriteString(")")
	}
	b.WriteString(")")
	b.WriteString("\n")

	_, err := parser.Parse(b.String())
	if err == nil {
		t.Fatal("expected error for combined deep nesting, got nil")
	}

	// Should fail on nesting depth
	if !strings.Contains(err.Error(), "depth exceeds") {
		t.Errorf("expected depth error, got: %v", err)
	}
}
