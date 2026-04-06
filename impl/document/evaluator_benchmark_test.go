package document

import (
	"fmt"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// BenchmarkEvaluate_ValidDocument benchmarks full document evaluation
// with a realistic multi-statement document (no errors).
// This is the hot path — error recovery must not regress it.
func BenchmarkEvaluate_ValidDocument(b *testing.B) {
	// Build a 100-statement document: x1 = 1, x2 = x1 + 1, ..., x100 = x99 + 1
	var lines []string
	lines = append(lines, "x1 = 1")
	for i := 2; i <= 100; i++ {
		lines = append(lines, fmt.Sprintf("x%d = x%d + 1", i, i-1))
	}
	source := strings.Join(lines, "\n") + "\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		b.Fatalf("NewDocument: %v", err)
	}

	b.ResetTimer()
	for b.Loop() {
		eval := NewEvaluator()
		if err := eval.Evaluate(doc); err != nil {
			b.Fatalf("Evaluate: %v", err)
		}
	}
}

// BenchmarkEvaluate_WithErrors benchmarks error recovery overhead:
// a document where half the statements fail.
func BenchmarkEvaluate_WithErrors(b *testing.B) {
	// 50 successful + 50 errored (cascading from first div-by-zero)
	var lines []string
	lines = append(lines, "bad = 1 / 0")
	for i := 1; i <= 49; i++ {
		lines = append(lines, fmt.Sprintf("ok%d = %d", i, i))
	}
	for i := 1; i <= 50; i++ {
		lines = append(lines, fmt.Sprintf("fail%d = bad + %d", i, i))
	}
	source := strings.Join(lines, "\n") + "\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		b.Fatalf("NewDocument: %v", err)
	}

	b.ResetTimer()
	for b.Loop() {
		eval := NewEvaluator()
		eval.Evaluate(doc) // ErrPartialEvaluation expected
	}
}
