package format

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestJSONFormatterSimple tests basic JSON output
func TestJSONFormatterSimple(t *testing.T) {
	doc, err := document.NewDocument("x = 10\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	var buf bytes.Buffer
	formatter := &JSONFormatter{}
	opts := Options{Verbose: false}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Parse JSON to verify it's valid
	var result JSONDocument
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	if len(result.Blocks) == 0 {
		t.Fatal("Expected at least one block in output")
	}

	// Check that the output contains the result
	outputStr := buf.String()
	if !strings.Contains(outputStr, "10") {
		t.Errorf("Expected JSON to contain '10', got: %s", outputStr)
	}
}

// TestJSONFormatterStructure tests the JSON structure
func TestJSONFormatterStructure(t *testing.T) {
	doc, err := document.NewDocument("x = 100 USD\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	var buf bytes.Buffer
	formatter := &JSONFormatter{}
	opts := Options{Verbose: false}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Parse and check structure
	var result JSONDocument
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if len(result.Blocks) < 1 {
		t.Fatal("Expected at least one block")
	}

	block := result.Blocks[0]

	// Check required fields
	if block.Type == "" {
		t.Error("JSON block should have 'type' field")
	}

	if block.Source == nil {
		t.Error("JSON block should have 'source' field")
	}
}

// TestJSONFormatterError tests error handling in JSON
func TestJSONFormatterError(t *testing.T) {
	doc, err := document.NewDocument("y = x + 1\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	eval.Evaluate(doc) // Will have error

	var buf bytes.Buffer
	formatter := &JSONFormatter{}
	opts := Options{Verbose: false}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Should still be valid JSON
	var result JSONDocument
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}
}

// TestJSONFormatterWithFrontmatter tests that frontmatter is included in JSON
func TestJSONFormatterWithFrontmatter(t *testing.T) {
	source := `---
exchange:
  USD_EUR: 0.92
globals:
  tax_rate: 0.32
---
x = 10
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
	formatter := &JSONFormatter{}
	opts := Options{Verbose: false}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	var result JSONDocument
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Check frontmatter
	if result.Frontmatter == nil {
		t.Fatal("Expected frontmatter in JSON output")
	}

	if result.Frontmatter.Globals == nil || result.Frontmatter.Globals["tax_rate"] != "0.32" {
		t.Errorf("Expected globals with tax_rate=0.32, got: %v", result.Frontmatter.Globals)
	}

	if result.Frontmatter.Exchange == nil || result.Frontmatter.Exchange["USD_EUR"] != "0.92" {
		t.Errorf("Expected exchange with USD_EUR=0.92, got: %v", result.Frontmatter.Exchange)
	}
}

// TestJSONFormatterExtensions tests file extensions
func TestJSONFormatterExtensions(t *testing.T) {
	formatter := &JSONFormatter{}
	exts := formatter.Extensions()

	if len(exts) == 0 {
		t.Fatal("JSONFormatter should return at least one extension")
	}

	if !slices.Contains(exts, ".json") {
		t.Error("JSONFormatter should handle .json extension")
	}
}
