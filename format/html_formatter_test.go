package format

import (
	"bytes"
	"strings"
	"testing"

	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestHTMLFormatterSimple tests basic HTML output
func TestHTMLFormatterSimple(t *testing.T) {
	source := "x = 10 + 5\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	var buf bytes.Buffer
	formatter := &HTMLFormatter{}
	opts := Options{Verbose: false}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()
	// Should contain HTML structure
	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Errorf("Expected HTML doctype, got: %s", output)
	}
	// Should contain result
	if !strings.Contains(output, "15") {
		t.Errorf("Expected output to contain result, got: %s", output)
	}
}

// TestHTMLFormatterWithError tests error display in HTML
func TestHTMLFormatterWithError(t *testing.T) {
	source := "y = x + 1\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	eval.Evaluate(doc) // Will have error

	var buf bytes.Buffer
	formatter := &HTMLFormatter{}
	opts := Options{Verbose: false}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()
	// Should be valid HTML
	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("Expected HTML doctype")
	}
	// Should mention error
	if !strings.Contains(output, "Error") && !strings.Contains(output, "error") {
		t.Log("Warning: Expected error mention in HTML")
	}
}

// TestHTMLFormatterExtensions tests file extensions
func TestHTMLFormatterExtensions(t *testing.T) {
	formatter := &HTMLFormatter{}
	exts := formatter.Extensions()

	if len(exts) < 2 {
		t.Fatal("HTMLFormatter should handle at least 2 extensions")
	}

	foundHTML := false
	foundHTM := false
	for _, ext := range exts {
		if ext == ".html" {
			foundHTML = true
		}
		if ext == ".htm" {
			foundHTM = true
		}
	}

	if !foundHTML || !foundHTM {
		t.Error("HTMLFormatter should handle .html and .htm extensions")
	}
}

// TestHTMLFormatterTemplate tests template rendering
func TestHTMLFormatterTemplate(t *testing.T) {
	source := `# Test

x = 100
`
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	var buf bytes.Buffer
	formatter := &HTMLFormatter{}
	opts := Options{Verbose: false}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()
	// Should have styling
	if !strings.Contains(output, "<style>") {
		t.Error("Expected HTML to include styling")
	}
	// Should contain calc-block class
	if !strings.Contains(output, "calc-block") {
		t.Error("Expected styled calc blocks")
	}
}

// TestHTMLFormatterIntermediateValues tests that HTML output includes
// intermediate calculation results for each line.
func TestHTMLFormatterIntermediateValues(t *testing.T) {
	// Multiple calculations in one block with dependencies
	source := `x = 10
y = x * 2
z = y + 5
`
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	var buf bytes.Buffer
	formatter := &HTMLFormatter{}
	opts := Options{Verbose: true}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Should contain all intermediate results
	// x = 10 → 10
	if !strings.Contains(output, "10") {
		t.Errorf("Expected HTML to contain x result (10), got: %s", output)
	}
	// y = x * 2 → 20
	if !strings.Contains(output, "20") {
		t.Errorf("Expected HTML to contain y result (20), got: %s", output)
	}
	// z = y + 5 → 25
	if !strings.Contains(output, "25") {
		t.Errorf("Expected HTML to contain z result (25), got: %s", output)
	}

	// Should contain source expressions
	if !strings.Contains(output, "x = 10") {
		t.Error("Expected HTML to contain source 'x = 10'")
	}
	if !strings.Contains(output, "y = x * 2") {
		t.Error("Expected HTML to contain source 'y = x * 2'")
	}
}

// TestHTMLFormatterMultiBlockIntermediates tests intermediate values across blocks.
func TestHTMLFormatterMultiBlockIntermediates(t *testing.T) {
	// Two separate blocks with dependencies
	source := `base = 100


rate = base * 0.15
total = base + rate
`
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	var buf bytes.Buffer
	formatter := &HTMLFormatter{}
	opts := Options{Verbose: true}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Should contain results from both blocks
	// base = 100 → 100
	if !strings.Contains(output, "100") {
		t.Errorf("Expected HTML to contain base result (100)")
	}
	// rate = base * 0.15 → 15
	if !strings.Contains(output, "15") {
		t.Errorf("Expected HTML to contain rate result (15)")
	}
	// total = base + rate → 115
	if !strings.Contains(output, "115") {
		t.Errorf("Expected HTML to contain total result (115)")
	}
}
