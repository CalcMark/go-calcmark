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

// --- Tests below validate the rich JSON output we want ---

// TestJSONFormatterPerStatementResults verifies that each assignment in a calc
// block includes its evaluated result, not just the block's last value.
func TestJSONFormatterPerStatementResults(t *testing.T) {
	source := "x = 10\ny = x * 2\nz = y + 1\n"
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
	if err := formatter.Format(&buf, doc, Options{}); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	var result JSONDocument
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Find the calc block
	var calcBlock *JSONBlock
	for i := range result.Blocks {
		if result.Blocks[i].Type == "calculation" {
			calcBlock = &result.Blocks[i]
			break
		}
	}
	if calcBlock == nil {
		t.Fatal("Expected a calculation block")
	}

	// Each statement must have its own result
	if len(calcBlock.Results) != 3 {
		t.Fatalf("Expected 3 per-statement results, got %d", len(calcBlock.Results))
	}

	expected := []struct {
		source string
		result string
	}{
		{"x = 10", "10"},
		{"y = x * 2", "20"},
		{"z = y + 1", "21"},
	}

	for i, want := range expected {
		if calcBlock.Results[i].Source != want.source {
			t.Errorf("Result[%d].Source = %q, want %q", i, calcBlock.Results[i].Source, want.source)
		}
		if calcBlock.Results[i].Value != want.result {
			t.Errorf("Result[%d].Value = %q, want %q", i, calcBlock.Results[i].Value, want.result)
		}
	}
}

// TestJSONFormatterPerStatementResultsWithBlankLines verifies result alignment
// when calc blocks contain blank line separators (the same bug the markdown
// formatter had).
func TestJSONFormatterPerStatementResultsWithBlankLines(t *testing.T) {
	source := "a = 1\n\nb = 2\n\nc = a + b\n"
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
	if err := formatter.Format(&buf, doc, Options{}); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	var result JSONDocument
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	var calcBlock *JSONBlock
	for i := range result.Blocks {
		if result.Blocks[i].Type == "calculation" {
			calcBlock = &result.Blocks[i]
			break
		}
	}
	if calcBlock == nil {
		t.Fatal("Expected a calculation block")
	}

	if len(calcBlock.Results) != 3 {
		t.Fatalf("Expected 3 results (blank lines excluded), got %d", len(calcBlock.Results))
	}

	expected := []struct {
		source string
		value  string
	}{
		{"a = 1", "1"},
		{"b = 2", "2"},
		{"c = a + b", "3"},
	}
	for i, want := range expected {
		if calcBlock.Results[i].Source != want.source {
			t.Errorf("Result[%d].Source = %q, want %q", i, calcBlock.Results[i].Source, want.source)
		}
		if calcBlock.Results[i].Value != want.value {
			t.Errorf("Result[%d].Value = %q, want %q", i, calcBlock.Results[i].Value, want.value)
		}
	}
}

// TestJSONFormatterVariableResultMapping verifies that each variable can be
// looked up with its name AND result value — essential for programmatic consumers.
func TestJSONFormatterVariableResultMapping(t *testing.T) {
	source := "price = $50\ntax_rate = 0.08\ntax = price * tax_rate\ntotal = price + tax\n"
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
	if err := formatter.Format(&buf, doc, Options{}); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	var result JSONDocument
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	var calcBlock *JSONBlock
	for i := range result.Blocks {
		if result.Blocks[i].Type == "calculation" {
			calcBlock = &result.Blocks[i]
			break
		}
	}
	if calcBlock == nil {
		t.Fatal("Expected a calculation block")
	}

	// Build a variable→value map from the results for easy lookup
	varValues := make(map[string]string)
	for _, r := range calcBlock.Results {
		if r.Variable != "" {
			varValues[r.Variable] = r.Value
		}
	}

	// Each variable should have its evaluated value
	expectedVars := map[string]string{
		"price":    "$50.00",
		"tax_rate": "0.08",
		"tax":      "$4.00",
		"total":    "$54.00",
	}

	for name, wantVal := range expectedVars {
		got, ok := varValues[name]
		if !ok {
			t.Errorf("Variable %q not found in results", name)
			continue
		}
		if got != wantVal {
			t.Errorf("Variable %q = %q, want %q", name, got, wantVal)
		}
	}
}

// TestJSONFormatterDiagnosticsHavePositions verifies that evaluation errors
// include line and column information, not just a message string.
func TestJSONFormatterDiagnosticsHavePositions(t *testing.T) {
	source := "x = 10\ny = unknown_var + 1\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	eval.Evaluate(doc) // expected to fail on y

	var buf bytes.Buffer
	formatter := &JSONFormatter{}
	if err := formatter.Format(&buf, doc, Options{}); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	var result JSONDocument
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Find the calc block with the error
	var calcBlock *JSONBlock
	for i := range result.Blocks {
		if result.Blocks[i].Type == "calculation" {
			calcBlock = &result.Blocks[i]
			break
		}
	}
	if calcBlock == nil {
		t.Fatal("Expected a calculation block")
	}

	// Should have diagnostics with position info
	if len(calcBlock.Diagnostics) == 0 {
		t.Fatal("Expected at least one diagnostic for undefined variable error")
	}

	diag := calcBlock.Diagnostics[0]
	if diag.Message == "" {
		t.Error("Diagnostic should have a message")
	}
	if diag.Severity == "" {
		t.Error("Diagnostic should have a severity")
	}
	// Line should be > 0 (1-indexed from parser)
	if diag.Line == 0 {
		t.Error("Diagnostic should have a line number")
	}
}

// TestJSONFormatterTextBlockRenderedHTML verifies that text blocks include
// rendered HTML so consumers can display rich markdown without re-parsing.
func TestJSONFormatterTextBlockRenderedHTML(t *testing.T) {
	source := "# Budget Overview\n\nThis is a **bold** statement.\n\nx = 10\n"
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
	if err := formatter.Format(&buf, doc, Options{}); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	var result JSONDocument
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Find text block
	var textBlock *JSONBlock
	for i := range result.Blocks {
		if result.Blocks[i].Type == "text" {
			textBlock = &result.Blocks[i]
			break
		}
	}
	if textBlock == nil {
		t.Fatal("Expected a text block")
	}

	// Text blocks should include rendered HTML
	if textBlock.HTML == "" {
		t.Error("Text block should include rendered HTML")
	}

	// HTML should contain rendered markdown elements
	if !strings.Contains(textBlock.HTML, "<h1") {
		t.Errorf("Expected HTML to contain <h1> heading, got: %s", textBlock.HTML)
	}
	if !strings.Contains(textBlock.HTML, "<strong>") || !strings.Contains(textBlock.HTML, "bold") {
		t.Errorf("Expected HTML to contain <strong>bold</strong>, got: %s", textBlock.HTML)
	}
}

// TestJSONFormatterRawValue verifies that raw_value is populated and always ASCII.
func TestJSONFormatterRawValue(t *testing.T) {
	source := "salary = $6500\nbonus = $500\ntotal = salary + bonus\n"
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
	if err := formatter.Format(&buf, doc, Options{}); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	var result JSONDocument
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	var calcBlock *JSONBlock
	for i := range result.Blocks {
		if result.Blocks[i].Type == "calculation" {
			calcBlock = &result.Blocks[i]
			break
		}
	}
	if calcBlock == nil {
		t.Fatal("Expected a calculation block")
	}

	for i, r := range calcBlock.Results {
		// raw_value must be populated for results with values
		if r.Value != "" && r.RawValue == "" {
			t.Errorf("Result[%d] has Value=%q but empty RawValue", i, r.Value)
		}

		// raw_value must be ASCII-only (no locale-specific characters)
		for j, ch := range r.RawValue {
			if ch > 127 {
				t.Errorf("Result[%d].RawValue contains non-ASCII at position %d: %q", i, j, r.RawValue)
				break
			}
		}
	}

	// Verify specific raw values
	if len(calcBlock.Results) >= 3 {
		// total = salary + bonus = $7000
		lastResult := calcBlock.Results[2]
		if lastResult.Value != "$7,000.00" {
			t.Errorf("Value = %q, want %q", lastResult.Value, "$7,000.00")
		}
		// raw_value should be the machine-readable form
		if !strings.Contains(lastResult.RawValue, "7000") {
			t.Errorf("RawValue = %q, should contain '7000'", lastResult.RawValue)
		}
	}
}

// TestJSONFormatterCurrencyResults verifies that currency values in results
// include the display-formatted representation (e.g., "$6,500.00" not "6500").
func TestJSONFormatterCurrencyResults(t *testing.T) {
	source := "salary = $6500\nbonus = $500\ntotal = salary + bonus\n"
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
	if err := formatter.Format(&buf, doc, Options{}); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	var result JSONDocument
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	var calcBlock *JSONBlock
	for i := range result.Blocks {
		if result.Blocks[i].Type == "calculation" {
			calcBlock = &result.Blocks[i]
			break
		}
	}
	if calcBlock == nil {
		t.Fatal("Expected a calculation block")
	}

	if len(calcBlock.Results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(calcBlock.Results))
	}

	// The display value should use the human-readable format (display.Format),
	// matching what the markdown/text formatters show.
	lastResult := calcBlock.Results[2]
	if lastResult.Value != "$7,000.00" {
		t.Errorf("Expected display-formatted '$7,000.00', got %q", lastResult.Value)
	}
}
