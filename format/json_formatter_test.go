package format

import (
	"bytes"
	"encoding/json"
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
	var result []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	if len(result) == 0 {
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
	var result []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if len(result) < 1 {
		t.Fatal("Expected at least one block")
	}

	block := result[0]

	// Check required fields
	if _, ok := block["type"]; !ok {
		t.Error("JSON block should have 'type' field")
	}

	if _, ok := block["source"]; !ok {
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
	var result []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}
}

// TestJSONFormatterExtensions tests file extensions
func TestJSONFormatterExtensions(t *testing.T) {
	formatter := &JSONFormatter{}
	exts := formatter.Extensions()

	if len(exts) == 0 {
		t.Fatal("JSONFormatter should return at least one extension")
	}

	found := false
	for _, ext := range exts {
		if ext == ".json" {
			found = true
			break
		}
	}

	if !found {
		t.Error("JSONFormatter should handle .json extension")
	}
}
