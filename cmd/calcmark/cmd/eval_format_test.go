package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/format"
	implDoc "github.com/CalcMark/go-calcmark/v2/impl/document"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

// evalTestDoc is a helper that creates, parses, and evaluates a CalcMark document.
func evalTestDoc(t *testing.T, input string) *document.Document {
	t.Helper()
	doc, err := document.NewDocument(input)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	return doc
}

// TestEvalFormatFlag_TextDefault verifies that the default format ("text") produces
// plain text output containing the computed result.
func TestEvalFormatFlag_TextDefault(t *testing.T) {
	doc := evalTestDoc(t, "x = 42")

	// "text" is the default format for eval
	formatter := format.GetFormatter("text", "")
	var buf bytes.Buffer
	opts := format.Options{}
	if err := formatter.Format(&buf, doc, opts); err != nil {
		t.Fatalf("Format: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	if !strings.Contains(got, "42") {
		t.Errorf("text format output should contain '42', got %q", got)
	}
}

// TestEvalFormatFlag_JSON verifies that --format json produces valid JSON output
// with the expected JSONDocument structure.
func TestEvalFormatFlag_JSON(t *testing.T) {
	doc := evalTestDoc(t, "x = 42")

	// JSON format - same as what --format json would select
	formatter := format.GetFormatter("json", "")
	var buf bytes.Buffer
	opts := format.Options{}
	if err := formatter.Format(&buf, doc, opts); err != nil {
		t.Fatalf("Format: %v", err)
	}

	// Must be valid JSON
	if !json.Valid(buf.Bytes()) {
		t.Fatalf("json format should produce valid JSON, got:\n%s", buf.String())
	}

	// Parse and verify structure
	var result format.JSONDocument
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if len(result.Blocks) == 0 {
		t.Fatal("expected at least one block in JSON output")
	}

	// The calculation block should have type "calculation" with results
	found := false
	for _, block := range result.Blocks {
		if block.Type == "calculation" {
			found = true
			if len(block.Results) == 0 {
				t.Error("calculation block should have results")
			}
		}
	}
	if !found {
		t.Error("expected a 'calculation' block in JSON output")
	}
}

// TestEvalFormatFlag_JSONMultiline verifies JSON output for a multi-line document
// with both text and calculation blocks.
func TestEvalFormatFlag_JSONMultiline(t *testing.T) {
	input := `# Budget

price = $100
tax = 10%
total = price + price * tax
`
	doc := evalTestDoc(t, input)

	formatter := format.GetFormatter("json", "")
	var buf bytes.Buffer
	opts := format.Options{Verbose: true}
	if err := formatter.Format(&buf, doc, opts); err != nil {
		t.Fatalf("Format: %v", err)
	}

	if !json.Valid(buf.Bytes()) {
		t.Fatalf("json format should produce valid JSON, got:\n%s", buf.String())
	}

	var result format.JSONDocument
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// Should have both text and calculation blocks
	hasText := false
	hasCalc := false
	for _, block := range result.Blocks {
		switch block.Type {
		case "text":
			hasText = true
		case "calculation":
			hasCalc = true
		}
	}

	if !hasText {
		t.Error("expected at least one 'text' block for the heading")
	}
	if !hasCalc {
		t.Error("expected at least one 'calculation' block")
	}
}

// TestEvalFormatFlag_TextAndJSONDiffer verifies that text and json formatters
// produce structurally different output for the same document.
func TestEvalFormatFlag_TextAndJSONDiffer(t *testing.T) {
	// Use a multi-line document so text output won't accidentally be valid JSON
	input := `# Test
result = 2 + 2
`
	doc := evalTestDoc(t, input)

	opts := format.Options{}

	// Format as text
	textFormatter := format.GetFormatter("text", "")
	var textBuf bytes.Buffer
	if err := textFormatter.Format(&textBuf, doc, opts); err != nil {
		t.Fatalf("text Format: %v", err)
	}

	// Format as JSON
	jsonFormatter := format.GetFormatter("json", "")
	var jsonBuf bytes.Buffer
	if err := jsonFormatter.Format(&jsonBuf, doc, opts); err != nil {
		t.Fatalf("json Format: %v", err)
	}

	// They must differ
	if textBuf.String() == jsonBuf.String() {
		t.Error("text and json output should differ")
	}

	// JSON output must be valid JSON with blocks array
	var result format.JSONDocument
	if err := json.Unmarshal(jsonBuf.Bytes(), &result); err != nil {
		t.Fatalf("json output should unmarshal as JSONDocument: %v", err)
	}

	if len(result.Blocks) == 0 {
		t.Error("JSON output should have blocks")
	}

	// Text output should be plain text (no "blocks" key)
	if strings.Contains(textBuf.String(), `"blocks"`) {
		t.Error("text output should not contain JSON structure")
	}
}

// TestEvalFormatFlag_JSONVariableTracking verifies that JSON output includes
// variable names and raw values for agent consumption.
func TestEvalFormatFlag_JSONVariableTracking(t *testing.T) {
	doc := evalTestDoc(t, "price = $99.99")

	formatter := format.GetFormatter("json", "")
	var buf bytes.Buffer
	opts := format.Options{}
	if err := formatter.Format(&buf, doc, opts); err != nil {
		t.Fatalf("Format: %v", err)
	}

	var result format.JSONDocument
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// Find the calculation block with the variable
	for _, block := range result.Blocks {
		if block.Type == "calculation" {
			for _, r := range block.Results {
				if r.Variable == "price" {
					// Value should contain the formatted result
					if r.Value == "" {
						t.Error("expected value for variable assignment")
					}
					return
				}
			}
			t.Error("expected result with variable 'price'")
			return
		}
	}
	t.Error("expected a calculation block")
}
