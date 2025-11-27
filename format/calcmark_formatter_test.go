package format

import (
	"bytes"
	"strings"
	"testing"

	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestCalcMarkFormatterSimple tests basic CalcMark output
func TestCalcMarkFormatterSimple(t *testing.T) {
	source := "x = 10\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	var buf bytes.Buffer
	formatter := &CalcMarkFormatter{}
	opts := Options{Verbose: false}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()
	// Should contain source
	if !strings.Contains(output, "x = 10") {
		t.Errorf("Expected output to contain source, got: %s", output)
	}
}

// TestCalcMarkFormatterVerbose tests verbose mode with comments
func TestCalcMarkFormatterVerbose(t *testing.T) {
	source := "x = 100 + 50\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	var buf bytes.Buffer
	formatter := &CalcMarkFormatter{}
	opts := Options{Verbose: true}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()
	// Should contain source
	if !strings.Contains(output, "x = 100 + 50") {
		t.Errorf("Expected output to contain source, got: %s", output)
	}
	// In verbose mode, should contain result as comment
	if !strings.Contains(output, "#") || !strings.Contains(output, "150") {
		t.Errorf("Expected verbose output to contain result comment, got: %s", output)
	}
}

// TestCalcMarkFormatterExtensions tests file extensions
func TestCalcMarkFormatterExtensions(t *testing.T) {
	formatter := &CalcMarkFormatter{}
	exts := formatter.Extensions()

	if len(exts) < 2 {
		t.Fatal("CalcMarkFormatter should handle at least 2 extensions")
	}

	foundCM := false
	foundCalcmark := false
	for _, ext := range exts {
		if ext == ".cm" {
			foundCM = true
		}
		if ext == ".calcmark" {
			foundCalcmark = true
		}
	}

	if !foundCM || !foundCalcmark {
		t.Error("CalcMarkFormatter should handle .cm and .calcmark extensions")
	}
}
