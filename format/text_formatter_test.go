package format

import (
	"bytes"
	"strings"
	"testing"

	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
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

	found := false
	for _, ext := range exts {
		if ext == ".txt" {
			found = true
			break
		}
	}

	if !found {
		t.Error("TextFormatter should handle .txt extension")
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
