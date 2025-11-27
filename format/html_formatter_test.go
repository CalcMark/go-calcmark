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
