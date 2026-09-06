package format

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/format/display"
	implDoc "github.com/CalcMark/go-calcmark/v2/impl/document"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

// helper creates a document, evaluates it, and formats it as JSON.
func formatJSON(t *testing.T, source string, opts Options) JSONDocument {
	t.Helper()
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	eval.Evaluate(doc)

	var buf bytes.Buffer
	formatter := &JSONFormatter{}
	if err := formatter.Format(&buf, doc, opts); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	var result JSONDocument
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v\nOutput: %s", err, buf.String())
	}
	return result
}

// findCalcBlock returns the first calculation block, or nil.
func findCalcBlock(doc JSONDocument) *JSONBlock {
	for i := range doc.Blocks {
		if doc.Blocks[i].Type == "calculation" {
			return &doc.Blocks[i]
		}
	}
	return nil
}

// TestJSONFormatterSimple tests basic JSON output
func TestJSONFormatterSimple(t *testing.T) {
	result := formatJSON(t, "x = 10\n", Options{})

	if len(result.Blocks) == 0 {
		t.Fatal("Expected at least one block in output")
	}

	block := findCalcBlock(result)
	if block == nil {
		t.Fatal("Expected a calculation block")
	}

	if len(block.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(block.Results))
	}

	r := block.Results[0]
	if r.Type != "number" {
		t.Errorf("Type = %q, want %q", r.Type, "number")
	}
	if r.NumericValue == nil || *r.NumericValue != 10 {
		t.Errorf("NumericValue = %v, want 10", r.NumericValue)
	}
}

// TestJSONFormatterStructure tests the JSON structure for currency
func TestJSONFormatterStructure(t *testing.T) {
	result := formatJSON(t, "x = 100 USD\n", Options{})

	block := findCalcBlock(result)
	if block == nil {
		t.Fatal("Expected a calculation block")
	}

	if block.Type == "" {
		t.Error("JSON block should have 'type' field")
	}
	if block.Source == nil {
		t.Error("JSON block should have 'source' field")
	}
	if len(block.Results) < 1 {
		t.Fatal("Expected at least one result")
	}

	r := block.Results[0]
	if r.Type != "currency" {
		t.Errorf("Type = %q, want %q", r.Type, "currency")
	}
	if r.Unit != "USD" {
		t.Errorf("Unit = %q, want %q", r.Unit, "USD")
	}
}

// TestJSONFormatterError tests error handling in JSON
func TestJSONFormatterError(t *testing.T) {
	result := formatJSON(t, "y = x + 1\n", Options{})

	// Should still be valid JSON (formatJSON already checked this)
	if len(result.Blocks) == 0 {
		t.Fatal("Expected at least one block")
	}
}

// TestJSONFormatterWithFrontmatter tests that frontmatter is included in JSON
func TestJSONFormatterWithFrontmatter(t *testing.T) {
	source := "---\nexchange:\n  USD_EUR: 0.92\nglobals:\n  tax_rate: 0.32\n---\nx = 10\n"
	result := formatJSON(t, source, Options{})

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

// TestJSONFormatterPerStatementResults verifies that each assignment in a calc
// block includes its evaluated result, not just the block's last value.
func TestJSONFormatterPerStatementResults(t *testing.T) {
	result := formatJSON(t, "x = 10\ny = x * 2\nz = y + 1\n", Options{})

	block := findCalcBlock(result)
	if block == nil {
		t.Fatal("Expected a calculation block")
	}

	if len(block.Results) != 3 {
		t.Fatalf("Expected 3 per-statement results, got %d", len(block.Results))
	}

	expected := []struct {
		source string
		value  string
		typ    string
	}{
		{"x = 10", "10", "number"},
		{"y = x * 2", "20", "number"},
		{"z = y + 1", "21", "number"},
	}

	for i, want := range expected {
		r := block.Results[i]
		if r.Source != want.source {
			t.Errorf("Result[%d].Source = %q, want %q", i, r.Source, want.source)
		}
		if r.Value != want.value {
			t.Errorf("Result[%d].Value = %q, want %q", i, r.Value, want.value)
		}
		if r.Type != want.typ {
			t.Errorf("Result[%d].Type = %q, want %q", i, r.Type, want.typ)
		}
	}
}

// TestJSONFormatterPerStatementResultsWithBlankLines verifies result alignment
// when calc blocks contain blank line separators.
func TestJSONFormatterPerStatementResultsWithBlankLines(t *testing.T) {
	result := formatJSON(t, "a = 1\n\nb = 2\n\nc = a + b\n", Options{})

	block := findCalcBlock(result)
	if block == nil {
		t.Fatal("Expected a calculation block")
	}

	if len(block.Results) != 3 {
		t.Fatalf("Expected 3 results (blank lines excluded), got %d", len(block.Results))
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
		if block.Results[i].Source != want.source {
			t.Errorf("Result[%d].Source = %q, want %q", i, block.Results[i].Source, want.source)
		}
		if block.Results[i].Value != want.value {
			t.Errorf("Result[%d].Value = %q, want %q", i, block.Results[i].Value, want.value)
		}
	}
}

// TestJSONFormatterVariableResultMapping verifies that each variable can be
// looked up with its name AND result value.
func TestJSONFormatterVariableResultMapping(t *testing.T) {
	result := formatJSON(t, "price = $50\ntax_rate = 0.08\ntax = price * tax_rate\ntotal = price + tax\n", Options{})

	block := findCalcBlock(result)
	if block == nil {
		t.Fatal("Expected a calculation block")
	}

	// Build variable→result map
	type varInfo struct {
		value string
		typ   string
		unit  string
	}
	varMap := make(map[string]varInfo)
	for _, r := range block.Results {
		if r.Variable != "" {
			varMap[r.Variable] = varInfo{value: r.Value, typ: r.Type, unit: r.Unit}
		}
	}

	expectedVars := map[string]varInfo{
		"price":    {value: "$50.00", typ: "currency", unit: "USD"},
		"tax_rate": {value: "0.08", typ: "number", unit: ""},
		"tax":      {value: "$4.00", typ: "currency", unit: "USD"},
		"total":    {value: "$54.00", typ: "currency", unit: "USD"},
	}

	for name, want := range expectedVars {
		got, ok := varMap[name]
		if !ok {
			t.Errorf("Variable %q not found in results", name)
			continue
		}
		if got.value != want.value {
			t.Errorf("Variable %q value = %q, want %q", name, got.value, want.value)
		}
		if got.typ != want.typ {
			t.Errorf("Variable %q type = %q, want %q", name, got.typ, want.typ)
		}
		if got.unit != want.unit {
			t.Errorf("Variable %q unit = %q, want %q", name, got.unit, want.unit)
		}
	}
}

// TestJSONFormatterDiagnosticsHavePositions verifies that evaluation errors
// include line and column information, not just a message string.
func TestJSONFormatterDiagnosticsHavePositions(t *testing.T) {
	result := formatJSON(t, "x = 10\ny = unknown_var + 1\n", Options{})

	block := findCalcBlock(result)
	if block == nil {
		t.Fatal("Expected a calculation block")
	}

	if len(block.Diagnostics) == 0 {
		t.Fatal("Expected at least one diagnostic for undefined variable error")
	}

	diag := block.Diagnostics[0]
	if diag.Message == "" {
		t.Error("Diagnostic should have a message")
	}
	if diag.Severity == "" {
		t.Error("Diagnostic should have a severity")
	}
	if diag.Line == 0 {
		t.Error("Diagnostic should have a line number")
	}
}

// TestJSONFormatterTextBlockRenderedHTML verifies that text blocks include
// rendered HTML so consumers can display rich markdown without re-parsing.
func TestJSONFormatterTextBlockRenderedHTML(t *testing.T) {
	result := formatJSON(t, "# Budget Overview\n\nThis is a **bold** statement.\n\nx = 10\n", Options{})

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

	if textBlock.HTML == "" {
		t.Error("Text block should include rendered HTML")
	}

	if !strings.Contains(textBlock.HTML, "<h1") {
		t.Errorf("Expected HTML to contain <h1> heading, got: %s", textBlock.HTML)
	}
	if !strings.Contains(textBlock.HTML, "<strong>") || !strings.Contains(textBlock.HTML, "bold") {
		t.Errorf("Expected HTML to contain <strong>bold</strong>, got: %s", textBlock.HTML)
	}
}

// TestJSONFormatterCurrencyResults verifies that currency values in results
// include the display-formatted representation.
func TestJSONFormatterCurrencyResults(t *testing.T) {
	result := formatJSON(t, "salary = $6500\nbonus = $500\ntotal = salary + bonus\n", Options{})

	block := findCalcBlock(result)
	if block == nil {
		t.Fatal("Expected a calculation block")
	}

	if len(block.Results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(block.Results))
	}

	lastResult := block.Results[2]
	if lastResult.Value != "$7,000.00" {
		t.Errorf("Expected display-formatted '$7,000.00', got %q", lastResult.Value)
	}
	if lastResult.Type != "currency" {
		t.Errorf("Type = %q, want %q", lastResult.Type, "currency")
	}
	if lastResult.Unit != "USD" {
		t.Errorf("Unit = %q, want %q", lastResult.Unit, "USD")
	}
	if lastResult.NumericValue == nil || *lastResult.NumericValue != 7000 {
		t.Errorf("NumericValue = %v, want 7000", lastResult.NumericValue)
	}
}

// --- New type-specific tests ---

// TestJSONFormatterNumberType verifies plain number results.
func TestJSONFormatterNumberType(t *testing.T) {
	result := formatJSON(t, "x = 42\n", Options{})

	block := findCalcBlock(result)
	if block == nil {
		t.Fatal("Expected a calculation block")
	}

	r := block.Results[0]
	if r.Type != "number" {
		t.Errorf("Type = %q, want %q", r.Type, "number")
	}
	if r.NumericValue == nil || *r.NumericValue != 42 {
		t.Errorf("NumericValue = %v, want 42", r.NumericValue)
	}
	if r.Unit != "" {
		t.Errorf("Unit = %q, want empty", r.Unit)
	}
	if r.Variable != "x" {
		t.Errorf("Variable = %q, want %q", r.Variable, "x")
	}
}

// TestJSONFormatterQuantityType verifies quantity results with units.
func TestJSONFormatterQuantityType(t *testing.T) {
	result := formatJSON(t, "weight = 5 kg\n", Options{})

	block := findCalcBlock(result)
	if block == nil {
		t.Fatal("Expected a calculation block")
	}

	r := block.Results[0]
	if r.Type != "quantity" {
		t.Errorf("Type = %q, want %q", r.Type, "quantity")
	}
	if r.NumericValue == nil || *r.NumericValue != 5 {
		t.Errorf("NumericValue = %v, want 5", r.NumericValue)
	}
	if r.Unit != "kg" {
		t.Errorf("Unit = %q, want %q", r.Unit, "kg")
	}
}

// TestJSONFormatterRateType verifies rate results with compound units.
func TestJSONFormatterRateType(t *testing.T) {
	result := formatJSON(t, "speed = 100 MB/s\n", Options{})

	block := findCalcBlock(result)
	if block == nil {
		t.Fatal("Expected a calculation block")
	}

	r := block.Results[0]
	if r.Type != "rate" {
		t.Errorf("Type = %q, want %q", r.Type, "rate")
	}
	if r.NumericValue == nil || *r.NumericValue != 100 {
		t.Errorf("NumericValue = %v, want 100", r.NumericValue)
	}
	if r.Unit != "MB/s" {
		t.Errorf("Unit = %q, want %q", r.Unit, "MB/s")
	}
}

// TestJSONFormatterDurationType verifies duration results.
func TestJSONFormatterDurationType(t *testing.T) {
	result := formatJSON(t, "elapsed = 3 hours\n", Options{})

	block := findCalcBlock(result)
	if block == nil {
		t.Fatal("Expected a calculation block")
	}

	r := block.Results[0]
	if r.Type != "duration" {
		t.Errorf("Type = %q, want %q", r.Type, "duration")
	}
	if r.NumericValue == nil || *r.NumericValue != 3 {
		t.Errorf("NumericValue = %v, want 3", r.NumericValue)
	}
	if r.Unit == "" {
		t.Error("Unit should not be empty for duration")
	}
}

// TestJSONFormatterBooleanType verifies boolean results.
func TestJSONFormatterBooleanType(t *testing.T) {
	result := formatJSON(t, "5 > 3\n", Options{})

	block := findCalcBlock(result)
	if block == nil {
		t.Fatal("Expected a calculation block")
	}

	if len(block.Results) < 1 {
		t.Fatal("Expected at least one result")
	}

	r := block.Results[0]
	if r.Type != "boolean" {
		t.Errorf("Type = %q, want %q", r.Type, "boolean")
	}
	if r.NumericValue != nil {
		t.Errorf("NumericValue should be nil for boolean, got %v", *r.NumericValue)
	}
	if r.Unit != "" {
		t.Errorf("Unit should be empty for boolean, got %q", r.Unit)
	}
}

// TestJSONFormatterZeroNumericValue verifies that numeric_value of 0 is explicitly present.
func TestJSONFormatterZeroNumericValue(t *testing.T) {
	result := formatJSON(t, "x = 0\n", Options{})

	block := findCalcBlock(result)
	if block == nil {
		t.Fatal("Expected a calculation block")
	}

	r := block.Results[0]
	if r.Type != "number" {
		t.Errorf("Type = %q, want %q", r.Type, "number")
	}
	if r.NumericValue == nil {
		t.Fatal("NumericValue should not be nil for x = 0")
	}
	if *r.NumericValue != 0 {
		t.Errorf("NumericValue = %v, want 0", *r.NumericValue)
	}

	// Verify the JSON output actually contains "numeric_value": 0
	doc, _ := document.NewDocument("x = 0\n")
	eval := implDoc.NewEvaluator()
	eval.Evaluate(doc)
	var buf bytes.Buffer
	(&JSONFormatter{}).Format(&buf, doc, Options{})
	if !strings.Contains(buf.String(), `"numeric_value": 0`) {
		t.Errorf("JSON output should contain '\"numeric_value\": 0', got: %s", buf.String())
	}
}

// TestJSONFormatterPerResultError verifies per-result error for partial block failure.
// Uses division by zero which passes semantic analysis but fails at runtime,
// allowing the first statement to succeed before the second fails.
func TestJSONFormatterPerResultError(t *testing.T) {
	result := formatJSON(t, "x = 10\ny = x / 0\n", Options{})

	block := findCalcBlock(result)
	if block == nil {
		t.Fatal("Expected a calculation block")
	}

	if len(block.Results) < 2 {
		t.Fatalf("Expected at least 2 results, got %d", len(block.Results))
	}

	// First result should succeed
	first := block.Results[0]
	if first.Type != "number" {
		t.Errorf("First result Type = %q, want %q", first.Type, "number")
	}
	if first.Error != "" {
		t.Errorf("First result should not have error, got %q", first.Error)
	}

	// Second result should have error, no type
	second := block.Results[1]
	if second.Error == "" {
		t.Error("Second result should have error for division by zero")
	}
	if second.Type != "" {
		t.Errorf("Error result should not have Type, got %q", second.Type)
	}
	if second.NumericValue != nil {
		t.Errorf("Error result should not have NumericValue, got %v", *second.NumericValue)
	}
}

// TestJSONFormatterBlockFieldsRemoved verifies output and variables are removed from JSON.
func TestJSONFormatterBlockFieldsRemoved(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\ny = 20\n")
	eval := implDoc.NewEvaluator()
	eval.Evaluate(doc)

	var buf bytes.Buffer
	(&JSONFormatter{}).Format(&buf, doc, Options{})
	jsonStr := buf.String()

	if strings.Contains(jsonStr, `"output"`) {
		t.Error("JSON should not contain 'output' field")
	}
	if strings.Contains(jsonStr, `"variables"`) {
		t.Error("JSON should not contain 'variables' field")
	}
	if strings.Contains(jsonStr, `"raw_value"`) {
		t.Error("JSON should not contain 'raw_value' field")
	}
}

// TestJSONFormatterLocaleNumericValue verifies numeric_value is locale-independent.
func TestJSONFormatterLocaleNumericValue(t *testing.T) {
	// Use de-DE locale (comma decimal, period thousand)
	deCfg, err := display.NewConfig("de-DE")
	if err != nil {
		t.Fatalf("Failed to create de-DE config: %v", err)
	}
	deFmt := display.NewFormatter(deCfg)

	result := formatJSON(t, "price = $1500\n", Options{DisplayFormatter: deFmt})

	block := findCalcBlock(result)
	if block == nil {
		t.Fatal("Expected a calculation block")
	}

	r := block.Results[0]

	// numeric_value must be locale-independent
	if r.NumericValue == nil || *r.NumericValue != 1500 {
		t.Errorf("NumericValue = %v, want 1500 (locale-independent)", r.NumericValue)
	}

	// unit must be locale-independent
	if r.Unit != "USD" {
		t.Errorf("Unit = %q, want %q (locale-independent)", r.Unit, "USD")
	}

	// value should be locale-formatted (German style)
	if !strings.Contains(r.Value, "1.500") && !strings.Contains(r.Value, "1500") {
		t.Errorf("Value = %q, expected German-formatted currency", r.Value)
	}
}

// TestJSONFormatterPercentageType verifies percentage results.
func TestJSONFormatterPercentageType(t *testing.T) {
	result := formatJSON(t, "rate = 20%\nhalf = 50%\n", Options{})

	block := findCalcBlock(result)
	if block == nil {
		t.Fatal("Expected a calculation block")
	}

	if len(block.Results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(block.Results))
	}

	// First result: 20%
	r := block.Results[0]
	if r.Type != "percentage" {
		t.Errorf("Type = %q, want %q", r.Type, "percentage")
	}
	if r.Variable != "rate" {
		t.Errorf("Variable = %q, want %q", r.Variable, "rate")
	}
	if r.NumericValue == nil {
		t.Fatal("NumericValue should not be nil for percentage")
	}
	if *r.NumericValue != 0.2 {
		t.Errorf("NumericValue = %v, want 0.2 (fractional form of 20%%)", *r.NumericValue)
	}
	if r.Value != "20%" {
		t.Errorf("Value = %q, want %q", r.Value, "20%")
	}
	if r.Unit != "" {
		t.Errorf("Unit = %q, want empty for percentage", r.Unit)
	}

	// Second result: 50%
	r2 := block.Results[1]
	if r2.Type != "percentage" {
		t.Errorf("Type = %q, want %q", r2.Type, "percentage")
	}
	if r2.NumericValue == nil || *r2.NumericValue != 0.5 {
		t.Errorf("NumericValue = %v, want 0.5 (fractional form of 50%%)", r2.NumericValue)
	}
	if r2.Value != "50%" {
		t.Errorf("Value = %q, want %q", r2.Value, "50%")
	}
}

// TestJSONFormatterNapkinEstimate verifies is_approximate for napkin quantities.
// Uses a quantity input (kg) because the napkin operator preserves type:
// plain numbers stay Number (no IsNapkin), quantities stay Quantity with IsNapkin=true.
func TestJSONFormatterNapkinEstimate(t *testing.T) {
	result := formatJSON(t, "estimate = 500 kg as napkin\n", Options{})

	block := findCalcBlock(result)
	if block == nil {
		t.Fatal("Expected a calculation block")
	}

	if len(block.Results) < 1 {
		t.Fatal("Expected at least one result")
	}

	r := block.Results[0]
	if r.Type != "quantity" {
		t.Errorf("Type = %q, want %q", r.Type, "quantity")
	}
	if !r.IsApproximate {
		t.Error("IsApproximate should be true for napkin estimate")
	}
	if r.NumericValue == nil {
		t.Fatal("NumericValue should not be nil for napkin estimate")
	}
}

// TestJSONFormatterNapkinNumber verifies is_approximate for napkin numbers.
func TestJSONFormatterNapkinNumber(t *testing.T) {
	result := formatJSON(t, "estimate = 1234567 as napkin\n", Options{})

	block := findCalcBlock(result)
	if block == nil {
		t.Fatal("Expected a calculation block")
	}

	if len(block.Results) < 1 {
		t.Fatal("Expected at least one result")
	}

	r := block.Results[0]
	if r.Type != "number" {
		t.Errorf("Type = %q, want %q", r.Type, "number")
	}
	if !r.IsApproximate {
		t.Error("IsApproximate should be true for napkin number")
	}
	if r.Value != "~1.2M" {
		t.Errorf("Value = %q, want %q", r.Value, "~1.2M")
	}
}

// TestJSONFormatterNapkinCurrency verifies is_approximate for napkin currencies.
func TestJSONFormatterNapkinCurrency(t *testing.T) {
	result := formatJSON(t, "estimate = $1234567 as napkin\n", Options{})

	block := findCalcBlock(result)
	if block == nil {
		t.Fatal("Expected a calculation block")
	}

	if len(block.Results) < 1 {
		t.Fatal("Expected at least one result")
	}

	r := block.Results[0]
	if r.Type != "currency" {
		t.Errorf("Type = %q, want %q", r.Type, "currency")
	}
	if !r.IsApproximate {
		t.Error("IsApproximate should be true for napkin currency")
	}
	if r.Value != "~$1.2M" {
		t.Errorf("Value = %q, want %q", r.Value, "~$1.2M")
	}
}

// TestJSONFormatterFractionASCII verifies that JSON output uses ASCII fractions,
// never Unicode Number Forms.
func TestJSONFormatterFractionASCII(t *testing.T) {
	source := "a = 1/2\nb = 7/3\n"
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
		t.Fatalf("Format error: %v", err)
	}

	output := buf.String()

	// Values must be ASCII fractions
	if !strings.Contains(output, `"1/2"`) {
		t.Errorf("Expected ASCII \"1/2\" in JSON output, got:\n%s", output)
	}
	if !strings.Contains(output, `"2 1/3"`) {
		t.Errorf("Expected ASCII \"2 1/3\" in JSON output, got:\n%s", output)
	}

	// Unicode fractions must NEVER appear
	unicodeFractions := []string{"½", "⅓", "⅔", "¼", "¾"}
	for _, uf := range unicodeFractions {
		if strings.Contains(output, uf) {
			t.Errorf("Unicode fraction %s must not appear in JSON output", uf)
		}
	}
}

// Each result carries its OWN error, never a neighbor's (go-calcmark#113).
// Before, every result in a block that had a semantic error was stamped
// with the block-level error, so `c = 3` reported "cannot reassign 'a'".
func TestJSONFormatterPerResultError_IsTheStatementsOwn(t *testing.T) {
	result := formatJSON(t, "a = 1 / 0\na = 2\nc = 3\nc = 5\n", Options{})

	block := findCalcBlock(result)
	if block == nil {
		t.Fatal("Expected a calculation block")
	}
	if len(block.Results) != 4 {
		t.Fatalf("Expected 4 results, got %d", len(block.Results))
	}
	if !strings.Contains(block.Results[0].Error, "division by zero") {
		t.Errorf("results[0] (`a = 1 / 0`) error = %q, want division by zero", block.Results[0].Error)
	}
	if !strings.Contains(block.Results[1].Error, "cannot reassign 'a'") {
		t.Errorf("results[1] (`a = 2`) error = %q, want cannot reassign 'a'", block.Results[1].Error)
	}
	if block.Results[2].Error != "" || block.Results[2].Value != "3" {
		t.Errorf("results[2] (`c = 3`) = %+v, want value 3 and no error", block.Results[2])
	}
	if !strings.Contains(block.Results[3].Error, "cannot reassign 'c'") {
		t.Errorf("results[3] (`c = 5`) error = %q, want cannot reassign 'c'", block.Results[3].Error)
	}
	if block.Error == "" {
		t.Error("block-level error should still be reported")
	}
}
