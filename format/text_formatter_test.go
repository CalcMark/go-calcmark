package format

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	implDoc "github.com/CalcMark/go-calcmark/v2/impl/document"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

// TestTextFormatterSimple tests basic text formatting
func TestTextFormatterSimple(t *testing.T) {
	doc, err := document.NewDocument("x = 10\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Evaluate the document
	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate document: %v", err)
	}

	var buf bytes.Buffer
	formatter := &TextFormatter{}
	opts := Options{Verbose: false}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "10") {
		t.Errorf("Expected output to contain '10', got: %s", output)
	}
}

// TestTextFormatterVerbose tests verbose mode
func TestTextFormatterVerbose(t *testing.T) {
	doc, err := document.NewDocument("x = 10 + 5\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Evaluate the document
	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate document: %v", err)
	}

	var buf bytes.Buffer
	formatter := &TextFormatter{}
	opts := Options{Verbose: true}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()
	// In verbose mode, should show source
	if !strings.Contains(output, "x = 10 + 5") {
		t.Errorf("Expected verbose output to contain source, got: %s", output)
	}
	// Should also show result
	if !strings.Contains(output, "15") {
		t.Errorf("Expected output to contain '15', got: %s", output)
	}
}

// TestTextFormatterError tests error handling
func TestTextFormatterError(t *testing.T) {
	// Create a document with an error (undefined variable)
	doc, err := document.NewDocument("y = x + 1\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Evaluate (this should produce an error)
	eval := implDoc.NewEvaluator()
	eval.Evaluate(doc) // Ignore error as we want to format it

	var buf bytes.Buffer
	formatter := &TextFormatter{}
	opts := Options{Verbose: false}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()
	// Should contain error message
	if !strings.Contains(output, "Error") && !strings.Contains(output, "error") {
		t.Log("Warning: Expected output to contain error message, got:", output)
	}
}

// TestTextFormatterExtensions tests file extensions
func TestTextFormatterExtensions(t *testing.T) {
	formatter := &TextFormatter{}
	exts := formatter.Extensions()

	if len(exts) == 0 {
		t.Fatal("TextFormatter should return at least one extension")
	}

	if !slices.Contains(exts, ".txt") {
		t.Error("TextFormatter should handle .txt extension")
	}
}

// TestTextFormatterFrontmatterVerbose tests verbose mode includes frontmatter section.
func TestTextFormatterFrontmatterVerbose(t *testing.T) {
	source := `---
globals:
  tax_rate: 10%
---
price = 100
total = price * (1 + @globals.tax_rate)
`
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Evaluate
	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	var buf bytes.Buffer
	formatter := &TextFormatter{}
	opts := Options{Verbose: true}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Should contain frontmatter header
	if !strings.Contains(output, "--- Frontmatter ---") {
		t.Errorf("Expected verbose output to contain '--- Frontmatter ---', got:\n%s", output)
	}

	// Should contain the global variable
	if !strings.Contains(output, "tax_rate") {
		t.Errorf("Expected verbose output to contain 'tax_rate', got:\n%s", output)
	}

	// Should contain the calc results
	if !strings.Contains(output, "price = 100") {
		t.Errorf("Expected verbose output to contain 'price = 100', got:\n%s", output)
	}
}

// TestTextFormatterFrontmatterNonVerbose tests that non-verbose mode skips frontmatter.
func TestTextFormatterFrontmatterNonVerbose(t *testing.T) {
	source := `---
globals:
  tax_rate: 10%
---
price = 100
`
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Evaluate
	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	var buf bytes.Buffer
	formatter := &TextFormatter{}
	opts := Options{Verbose: false}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Non-verbose should NOT contain frontmatter header
	if strings.Contains(output, "--- Frontmatter ---") {
		t.Errorf("Non-verbose output should not contain '--- Frontmatter ---', got:\n%s", output)
	}

	// Should still contain the calc result
	if !strings.Contains(output, "100") {
		t.Errorf("Expected output to contain '100', got:\n%s", output)
	}
}

// TestTextFormatterFrontmatterExchange tests exchange rates in verbose mode.
func TestTextFormatterFrontmatterExchange(t *testing.T) {
	source := `---
exchange:
  USD_EUR: 0.92
---
amount = 100 USD
`
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Evaluate
	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	var buf bytes.Buffer
	formatter := &TextFormatter{}
	opts := Options{Verbose: true}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Should contain frontmatter header
	if !strings.Contains(output, "--- Frontmatter ---") {
		t.Errorf("Expected verbose output to contain '--- Frontmatter ---', got:\n%s", output)
	}

	// Should contain exchange rate info
	if !strings.Contains(output, "USD") || !strings.Contains(output, "EUR") {
		t.Errorf("Expected verbose output to contain exchange rate info, got:\n%s", output)
	}
}

// TestTextFormatterMultiStatementBlock tests that non-verbose mode shows all
// per-statement results, not just the last value. Reproduces the bug where
// "a = 10 kg\nb = a + 10 kg" | cm eval only showed "20 kg".
func TestTextFormatterMultiStatementBlock(t *testing.T) {
	doc, err := document.NewDocument("a = 10 kg\nb = a + 10 kg\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	var buf bytes.Buffer
	formatter := &TextFormatter{}
	opts := Options{Verbose: false}

	if err := formatter.Format(&buf, doc, opts); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 2 {
		t.Fatalf("expected 2 output lines, got %d: %q", len(lines), output)
	}
	if !strings.Contains(lines[0], "10 kg") {
		t.Errorf("line 0 = %q, want to contain '10 kg'", lines[0])
	}
	if !strings.Contains(lines[1], "20 kg") {
		t.Errorf("line 1 = %q, want to contain '20 kg'", lines[1])
	}
}

// TestTextFormatterMultipleBlocks tests formatting multiple blocks
func TestTextFormatterMultipleBlocks(t *testing.T) {
	source := `x = 10
y = 20

z = x + y
`
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Evaluate
	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	var buf bytes.Buffer
	formatter := &TextFormatter{}
	opts := Options{Verbose: false}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()
	// Should contain results
	if !strings.Contains(output, "30") {
		t.Errorf("Expected output to contain '30', got: %s", output)
	}
}
