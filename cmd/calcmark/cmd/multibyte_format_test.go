package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/format"
)

// TestMultibyteEvalTextFormat verifies that cm eval with text format
// correctly renders multi-byte variable names and values.
func TestMultibyteEvalTextFormat(t *testing.T) {
	input := "a = 3\n手 = a * 5\n⭐️ = 手 + 23\ntest = ⭐️ * 1K\n"
	doc := evalTestDoc(t, input)

	formatter := format.GetFormatter("text", "")
	var buf bytes.Buffer
	if err := formatter.Format(&buf, doc, format.Options{Verbose: true}); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// All multi-byte source lines should appear in verbose text output
	for _, want := range []string{"手", "⭐️", "test"} {
		if !strings.Contains(output, want) {
			t.Errorf("Text output missing %q", want)
		}
	}
}

// TestMultibyteEvalJSONFormat verifies that cm eval with JSON format
// produces valid JSON with multi-byte variable names in results.
func TestMultibyteEvalJSONFormat(t *testing.T) {
	input := "a = 3\n手 = a * 5\ncafé = 100\nМосква = 200\n💰 = 1000\n⭐️ = 手 + 23\n"
	doc := evalTestDoc(t, input)

	formatter := format.GetFormatter("json", "")
	var buf bytes.Buffer
	if err := formatter.Format(&buf, doc, format.Options{}); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if !json.Valid(buf.Bytes()) {
		t.Fatalf("Invalid JSON:\n%s", buf.String())
	}

	var result format.JSONDocument
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Collect variables from all blocks
	variables := make(map[string]bool)
	for _, block := range result.Blocks {
		for _, r := range block.Results {
			if r.Variable != "" {
				variables[r.Variable] = true
			}
		}
	}

	// Verify all multi-byte variable names appear
	expectedVars := []string{"a", "手", "café", "Москва", "💰", "⭐️"}
	for _, v := range expectedVars {
		if !variables[v] {
			t.Errorf("Variable %q not found in JSON results", v)
		}
	}
}

// TestMultibyteConvertCMRoundTrip simulates cm convert --to cm for a
// multi-byte document and verifies lossless round-trip.
func TestMultibyteConvertCMRoundTrip(t *testing.T) {
	input := "a = 3\n手 = a * 5\n⭐️ = 手 + 23\ntest = ⭐️ * 1K\n"
	doc := evalTestDoc(t, input)

	// Format as .cm (convert --to cm)
	formatter := format.GetFormatter("cm", "")
	var buf bytes.Buffer
	if err := formatter.Format(&buf, doc, format.Options{Verbose: false}); err != nil {
		t.Fatalf("CM format failed: %v", err)
	}

	output := buf.String()

	// Verify source is preserved
	if !strings.Contains(output, "手 = a * 5") {
		t.Error("CM output missing '手 = a * 5'")
	}
	if !strings.Contains(output, "⭐️ = 手 + 23") {
		t.Error("CM output missing '⭐️ = 手 + 23'")
	}
	if !strings.Contains(output, "test = ⭐️ * 1K") {
		t.Error("CM output missing 'test = ⭐️ * 1K'")
	}

	// Reparse and re-evaluate to verify values survive round-trip
	doc2 := evalTestDoc(t, output)

	formatter2 := format.GetFormatter("json", "")
	var jsonBuf bytes.Buffer
	if err := formatter2.Format(&jsonBuf, doc2, format.Options{}); err != nil {
		t.Fatalf("JSON format of round-tripped doc failed: %v", err)
	}

	var result format.JSONDocument
	if err := json.Unmarshal(jsonBuf.Bytes(), &result); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	// Find test variable and verify value
	for _, block := range result.Blocks {
		for _, r := range block.Results {
			if r.Variable == "test" && r.Value != "" {
				if !strings.Contains(r.Value, "38") {
					t.Errorf("test value after round-trip: got %q, want value containing 38", r.Value)
				}
				return
			}
		}
	}
	t.Error("Variable 'test' not found in round-tripped JSON output")
}
