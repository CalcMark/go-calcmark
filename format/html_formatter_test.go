package format

import (
	"bytes"
	"os"
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

// TestHTMLFormatterFractionWithCustomUnit tests that fractions with custom units
// render using Unicode symbols in HTML output (e.g., "1/2 tomato" → "½ tomato").
func TestHTMLFormatterFractionWithCustomUnit(t *testing.T) {
	source := "half = 1/2 tomato\n"
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
	opts := Options{}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "½ tomato") {
		t.Errorf("Expected HTML to contain '½ tomato', got: %s", output)
	}
	if strings.Contains(output, "0.5 tomato") {
		t.Errorf("HTML should not contain '0.5 tomato' (decimal fallback)")
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

// TestHTMLFormatterWithTextBlockWarning tests that text block diagnostics
// (e.g., reserved keyword used as variable name) render as warnings in HTML.
func TestHTMLFormatterWithTextBlockWarning(t *testing.T) {
	source := "start = Apr 26\nend = Apr 27\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	eval.Evaluate(doc)

	var buf bytes.Buffer
	formatter := &HTMLFormatter{}
	err = formatter.Format(&buf, doc, Options{})
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "calc-warning") {
		t.Error("Expected calc-warning class in HTML output")
	}
	if !strings.Contains(output, "reserved keyword") {
		t.Error("Expected 'reserved keyword' in warning message")
	}
	if !strings.Contains(output, "end_val") {
		t.Error("Expected suggested alternative variable name in warning")
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

// TestHTMLFormatterBrFallbackEscapesHTML tests that the <br> fallback path
// in the HTML formatter escapes HTML entities to prevent XSS.
func TestHTMLFormatterBrFallbackEscapesHTML(t *testing.T) {
	// Create a text block that would trigger the <br> fallback
	// (Render() returns empty string)
	// We test via the full formatter pipeline with a document containing
	// HTML that should be escaped if it somehow reaches the fallback path.
	source := "# Normal heading\n\n<script>alert('xss')</script>\n"
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

	// The <script> tag must not appear unescaped in the output
	if strings.Contains(output, "<script>") {
		t.Error("Raw <script> tag must not appear in HTML output")
	}
}

// TestHTMLFormatterWithFrontmatter tests that frontmatter is rendered in HTML.
func TestHTMLFormatterWithFrontmatter(t *testing.T) {
	source := `---
exchange:
  USD_EUR: 0.92
globals:
  tax_rate: 0.32
  base_price: 100
---
price = @globals.base_price * (1 + @globals.tax_rate)
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

	// Should contain frontmatter section
	if !strings.Contains(output, "frontmatter") {
		t.Errorf("Expected HTML to contain frontmatter section")
	}

	// Should contain globals with @ prefix styling
	if !strings.Contains(output, "Globals") {
		t.Errorf("Expected HTML to contain Globals heading")
	}
	if !strings.Contains(output, "tax_rate") {
		t.Errorf("Expected HTML to contain tax_rate global")
	}
	if !strings.Contains(output, "base_price") {
		t.Errorf("Expected HTML to contain base_price global")
	}

	// Should contain exchange rates
	if !strings.Contains(output, "Exchange") {
		t.Errorf("Expected HTML to contain Exchange heading")
	}
	if !strings.Contains(output, "USD") && !strings.Contains(output, "EUR") {
		t.Errorf("Expected HTML to contain currency codes")
	}
	if !strings.Contains(output, "0.92") {
		t.Errorf("Expected HTML to contain exchange rate value")
	}

	// Should use dl/dt/dd structure
	if !strings.Contains(output, "<dl>") {
		t.Errorf("Expected HTML to use definition list")
	}
	if !strings.Contains(output, "<dt>") {
		t.Errorf("Expected HTML to use definition terms")
	}
	if !strings.Contains(output, "<dd>") {
		t.Errorf("Expected HTML to use definition descriptions")
	}
}

// TestHTMLFormatterScaleAndConvertTo tests that scale and convert_to
// frontmatter directives are rendered in the HTML output.
func TestHTMLFormatterScaleAndConvertTo(t *testing.T) {
	source := `---
scale:
  factor: 4
  unit_categories: [Mass, Volume]
convert_to: si
---
flour = 2 cups
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
	if err := formatter.Format(&buf, doc, Options{}); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Scale") {
		t.Error("Expected HTML to contain Scale heading")
	}
	if !strings.Contains(output, "4x") {
		t.Error("Expected HTML to contain scale factor '4x'")
	}
	if !strings.Contains(output, "Convert To") {
		t.Error("Expected HTML to contain Convert To heading")
	}
	if !strings.Contains(output, "si") {
		t.Error("Expected HTML to contain convert_to value 'si'")
	}
}

// TestHTMLFormatterFiscalFrontmatter tests that fiscal_year_starts is rendered in HTML.
func TestHTMLFormatterFiscalFrontmatter(t *testing.T) {
	source := `---
fiscal_year_starts: July 15
---
budget = $1,200,000
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
	err = formatter.Format(&buf, doc, Options{})
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Fiscal Year") {
		t.Error("Expected HTML to contain 'Fiscal Year' heading")
	}
	if !strings.Contains(output, "July 15") {
		t.Error("Expected HTML to contain 'July 15' value")
	}
}

// TestHTMLFormatterExtraFrontmatter tests that non-CalcMark frontmatter
// fields (title, tags, etc.) are rendered in the HTML output.
func TestHTMLFormatterExtraFrontmatter(t *testing.T) {
	source := `---
title: Guacamole Recipe
author: Chef
globals:
  servings: 4
---
avocados = 3
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
	if err := formatter.Format(&buf, doc, Options{}); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Extra fields should appear in the frontmatter section
	if !strings.Contains(output, "title") {
		t.Error("Expected HTML to contain 'title' extra field")
	}
	if !strings.Contains(output, "Guacamole Recipe") {
		t.Error("Expected HTML to contain extra field value 'Guacamole Recipe'")
	}
	if !strings.Contains(output, "author") {
		t.Error("Expected HTML to contain 'author' extra field")
	}
	if !strings.Contains(output, "Chef") {
		t.Error("Expected HTML to contain extra field value 'Chef'")
	}
	// CalcMark fields should still render
	if !strings.Contains(output, "Globals") {
		t.Error("Expected HTML to contain Globals heading")
	}
}

// --- Phase 4: Realistic document integration tests (HTML formatter) ---

// renderHTMLFromFile loads a .cm file and renders it through the HTML formatter.
func renderHTMLFromFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", path, err)
	}

	doc, err := document.NewDocument(string(data))
	if err != nil {
		t.Fatalf("Failed to create document from %s: %v", path, err)
	}

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate %s: %v", path, err)
	}

	var buf bytes.Buffer
	formatter := &HTMLFormatter{}
	opts := Options{Verbose: true}

	if err := formatter.Format(&buf, doc, opts); err != nil {
		t.Fatalf("HTML format failed for %s: %v", path, err)
	}
	return buf.String()
}

func TestHTMLFormatterEngineeringDocument(t *testing.T) {
	output := renderHTMLFromFile(t, "../testdata/examples/markdown_engineering.cm")

	// Structural HTML tags from CommonMark features used in this document
	expectedTags := []struct {
		tag     string
		feature string
	}{
		{"<h1", "ATX heading H1"},
		{"<h2", "ATX heading H2"},
		{"<h3", "ATX heading H3"},
		{"<hr", "horizontal rule"},
		{"<code>", "inline code"},
		{"<pre>", "fenced code block"},
		{"<strong>", "bold text"},
		{"<blockquote", "blockquote"},
	}

	for _, tc := range expectedTags {
		if !strings.Contains(output, tc.tag) {
			t.Errorf("Expected HTML tag %q for %s feature", tc.tag, tc.feature)
		}
	}

	// Security: no raw HTML passthrough from source content
	if strings.Contains(output, "<script") {
		t.Error("No <script> tags should pass through to HTML output")
	}

	// Calc results present
	if !strings.Contains(output, "calc-block") {
		t.Error("Expected calc-block sections in HTML output")
	}
}

func TestHTMLFormatterFinancialDocument(t *testing.T) {
	output := renderHTMLFromFile(t, "../testdata/examples/markdown_financial.cm")

	// Structural HTML tags
	for _, tag := range []string{"<h1", "<h2", "<h3", "<ol", "<ul", "<li", "<strong>", "<code>", "<blockquote"} {
		if !strings.Contains(output, tag) {
			t.Errorf("Expected HTML tag %q in financial document output", tag)
		}
	}

	// Nested blockquote
	if !strings.Contains(output, "<blockquote") {
		t.Error("Expected blockquote in financial document")
	}

	// Security
	if strings.Contains(output, "<script") {
		t.Error("No <script> tags should pass through to HTML output")
	}

	// Calc results present
	if !strings.Contains(output, "calc-block") {
		t.Error("Expected calc-block sections in HTML output")
	}
}

// --- Phase 5b: Edge case tests (HTML formatter) ---

func TestHTMLFormatterFencedCodeBlockNotExecuted(t *testing.T) {
	// CalcMark expressions inside fenced code blocks must NOT be executed
	source := "# Demo\n\n```\nx = 10\ny = x * 2\n```\n"
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

	if err := formatter.Format(&buf, doc, opts); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Should contain code block content as text, not as a calc-block
	if !strings.Contains(output, "x = 10") {
		t.Error("Expected fenced code block content in output")
	}

	// Should NOT have a calculation div — the calc-like content is inside a code fence
	// Note: "calc-block" appears in CSS styles, so check for the actual div element
	if strings.Contains(output, `class="calc-block"`) {
		t.Error("Fenced code block content should NOT produce calc-block div sections")
	}
}

func TestHTMLFormatterIndentedCodeBlockNotExecuted(t *testing.T) {
	// 4-space indented code that looks like CalcMark must NOT be executed
	source := "# Indented Code\n\n    x = 10\n    y = x * 2\n"
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

	if err := formatter.Format(&buf, doc, opts); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Should NOT have a calculation div — indented code is text, not calculation
	if strings.Contains(output, `class="calc-block"`) {
		t.Error("Indented code block content should NOT produce calc-block div sections")
	}
}

func TestHTMLFormatterScientificDocument(t *testing.T) {
	output := renderHTMLFromFile(t, "../testdata/examples/markdown_scientific.cm")

	// Structural HTML tags
	for _, tag := range []string{"<h1", "<h2", "<h3", "<em>", "<strong>", "<pre>", "<code>", "<blockquote", "<a href="} {
		if !strings.Contains(output, tag) {
			t.Errorf("Expected HTML tag %q in scientific document output", tag)
		}
	}

	// Image tag
	if !strings.Contains(output, "<img") {
		t.Error("Expected <img> tag for image syntax")
	}

	// Autolink
	if !strings.Contains(output, "energy.gov") {
		t.Error("Expected autolink URL in output")
	}

	// Security
	if strings.Contains(output, "<script") {
		t.Error("No <script> tags should pass through to HTML output")
	}

	// Calc results present
	if !strings.Contains(output, "calc-block") {
		t.Error("Expected calc-block sections in HTML output")
	}
}

// TestHTMLFormatterDataSourceLineAttributes verifies that data-source-line attributes
// are present on calc-line and text-block elements for scroll sync.
func TestHTMLFormatterDataSourceLineAttributes(t *testing.T) {
	source := "# Budget\n\nx = 10\ny = x * 2\n"
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
	err = formatter.Format(&buf, doc, Options{})
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Text block (heading) should have data-source-line
	if !strings.Contains(output, `data-source-line="1"`) {
		t.Errorf("Expected text block with data-source-line=\"1\", output:\n%s", output)
	}

	// Calc lines should have data-source-line attributes
	// "x = 10" is on line 3, "y = x * 2" is on line 4
	if !strings.Contains(output, `data-source-line="3"`) {
		t.Errorf("Expected calc line with data-source-line=\"3\", output:\n%s", output)
	}
	if !strings.Contains(output, `data-source-line="4"`) {
		t.Errorf("Expected calc line with data-source-line=\"4\", output:\n%s", output)
	}
}

// TestHTMLFormatterDataSourceLineWithFrontmatter verifies that data-source-line
// accounts for frontmatter lines.
func TestHTMLFormatterDataSourceLineWithFrontmatter(t *testing.T) {
	source := "---\nglobals:\n  tax: 10%\n---\n\nprice = 100\n"
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
	err = formatter.Format(&buf, doc, Options{})
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Frontmatter is 4 lines (--- through ---), plus empty line = 5
	// "price = 100" should be on line 6
	if !strings.Contains(output, `data-source-line="6"`) {
		t.Errorf("Expected calc line with data-source-line=\"6\" after frontmatter, output:\n%s", output)
	}
}

// TestHTMLFormatter_ErrorAfterBlankLine verifies that an error on a line
// after a blank line within a calc block shows an inline error in the HTML.
// Diagnostic Line numbers count all source lines (including blanks), so the
// formatter's line counter must also count blanks to match diagnostics correctly.
func TestHTMLFormatter_ErrorAfterBlankLine(t *testing.T) {
	// Blank line between b and a means they're in the same block but
	// the diagnostic for a=1/0 is on line 3 (counting the blank).
	source := "b = $23\n\na = 1 / 0\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	eval := implDoc.NewEvaluator()
	_ = eval.Evaluate(doc) // ErrPartialEvaluation expected

	var buf bytes.Buffer
	formatter := &HTMLFormatter{}
	opts := Options{Verbose: true}

	if err := formatter.Format(&buf, doc, opts); err != nil {
		t.Fatalf("Format: %v", err)
	}

	output := buf.String()

	// b = $23 should have a result
	if !strings.Contains(output, "calc-inline-result") {
		t.Error("expected b = $23 to have calc-inline-result")
	}

	// a = 1 / 0 should have a diagnostic below the source line
	if !strings.Contains(output, "calc-line-diagnostic") {
		t.Errorf("expected a = 1 / 0 to have calc-line-diagnostic, but HTML output was:\n%s", output)
	}
	if !strings.Contains(output, "division by zero") {
		t.Error("expected 'division by zero' in diagnostic")
	}
}

// TestHTMLFormatter_SemanticErrorShowsOnCorrectLine verifies that when a
// semantic error (like variable redefinition) aborts a block, the diagnostic
// appears on the correct line AND lines without per-line diagnostics still
// show the block-level error so no line appears silently blank.
func TestHTMLFormatter_SemanticErrorShowsOnCorrectLine(t *testing.T) {
	source := "a = 1 / 0\na = 2\nc = 3\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	eval := implDoc.NewEvaluator()
	_ = eval.Evaluate(doc) // ErrPartialEvaluation expected

	var buf bytes.Buffer
	formatter := &HTMLFormatter{}
	opts := Options{Verbose: true}

	if err := formatter.Format(&buf, doc, opts); err != nil {
		t.Fatalf("Format: %v", err)
	}

	output := buf.String()

	// The redefinition error should appear as a per-line diagnostic (below line 2)
	if !strings.Contains(output, "calc-line-diagnostic") {
		t.Errorf("expected per-line diagnostic for redefinition, but HTML output was:\n%s", output)
	}
	if !strings.Contains(output, "immutable") {
		t.Errorf("expected 'immutable' in diagnostic message, output:\n%s", output)
	}

	// Block-level error div should NOT be shown when per-line diagnostics cover it
	if strings.Contains(output, `<div class="calc-error">`) {
		t.Error("block-level error div should be suppressed when per-line diagnostics exist")
	}
}

// TestHTMLFormatterCustomTemplateWithPartials verifies that a custom template
// can call shared partials (cm-content, cm-frontmatter, cm-blocks).
func TestHTMLFormatterCustomTemplateWithPartials(t *testing.T) {
	source := "x = 42\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	customTemplate := `<!DOCTYPE html>
<html><head><title>Custom</title><style>{{.Style}}</style></head>
<body class="custom">{{template "cm-content" .}}</body></html>`

	var buf bytes.Buffer
	formatter := &HTMLFormatter{}
	opts := Options{Template: customTemplate}

	if err := formatter.Format(&buf, doc, opts); err != nil {
		t.Fatalf("Format with custom template failed: %v", err)
	}

	output := buf.String()

	// Custom wrapper is present
	if !strings.Contains(output, `<body class="custom">`) {
		t.Error("expected custom body class")
	}
	// Partials rendered the calc block
	if !strings.Contains(output, "calc-block") {
		t.Error("expected calc-block from cm-content partial")
	}
	if !strings.Contains(output, "42") {
		t.Error("expected result '42' from cm-content partial")
	}
	// No ZgotmplZ escaping
	if strings.Contains(output, "ZgotmplZ") {
		t.Error("ZgotmplZ found — template.CSS safety issue")
	}
}

// TestHTMLFormatterCustomTemplateIndividualPartials verifies that a custom template
// can call cm-frontmatter and cm-blocks individually for layout control.
func TestHTMLFormatterCustomTemplateIndividualPartials(t *testing.T) {
	source := "x = 42\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	// Custom template calls individual partials in a custom order
	customTemplate := `<div id="blocks">{{template "cm-blocks" .}}</div>
<div id="fm">{{template "cm-frontmatter" .}}</div>`

	var buf bytes.Buffer
	formatter := &HTMLFormatter{}
	opts := Options{Template: customTemplate}

	if err := formatter.Format(&buf, doc, opts); err != nil {
		t.Fatalf("Format with individual partials failed: %v", err)
	}

	output := buf.String()
	blocksIdx := strings.Index(output, `<div id="blocks">`)
	fmIdx := strings.Index(output, `<div id="fm">`)

	if blocksIdx < 0 || fmIdx < 0 {
		t.Fatalf("expected both wrapper divs in output:\n%s", output)
	}
	// Blocks should come before frontmatter (custom order)
	if blocksIdx > fmIdx {
		t.Error("expected blocks div before frontmatter div (custom partial order)")
	}
	if !strings.Contains(output, "calc-block") {
		t.Error("expected calc-block from cm-blocks partial")
	}
}

// TestHTMLFormatterLegacyCustomTemplateStillWorks verifies that an existing custom
// template that doesn't use partials (defines its own rendering) still works.
func TestHTMLFormatterLegacyCustomTemplateStillWorks(t *testing.T) {
	source := "x = 42\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	// Legacy template: does its own rendering, ignores partials
	legacyTemplate := `<html><body>
{{range .Blocks}}{{if eq .Type "calculation"}}
<pre>{{range .SourceLines}}{{.Source}} = {{.Result}}
{{end}}</pre>
{{end}}{{end}}
</body></html>`

	var buf bytes.Buffer
	formatter := &HTMLFormatter{}
	opts := Options{Template: legacyTemplate}

	if err := formatter.Format(&buf, doc, opts); err != nil {
		t.Fatalf("Format with legacy template failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "42") {
		t.Errorf("expected result '42' in legacy template output:\n%s", output)
	}
}

// TestHTMLFormatterLarkThemeValidation exercises the new template architecture
// with a Lark-style custom template: own page shell, custom accent, custom script,
// and CSS variable overrides for light/dark theming — all in <15 lines of custom code.
func TestHTMLFormatterLarkThemeValidation(t *testing.T) {
	source := "budget = 1000\nrent = budget * 0.3\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	// Lark-style custom template: 12 lines of custom code.
	// Own page shell, custom script, purple accent via CSS variable override.
	larkTemplate := `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Lark Playground</title>
<style>{{.Style}}
:root { --cm-accent: #7c3aed; --cm-font-sans: 'Inter', sans-serif; }
.dark { --cm-accent: #a78bfa; --cm-bg: #1e1e2e; --cm-text: #cdd6f4; --cm-text-code: #cdd6f4; --cm-bg-subtle: #313244; --cm-border: #45475a; }
</style>
<script src="/lark.js" defer></script>
</head>
<body><main>{{template "cm-content" .}}</main></body>
</html>`

	var buf bytes.Buffer
	formatter := &HTMLFormatter{}
	opts := Options{Template: larkTemplate}

	if err := formatter.Format(&buf, doc, opts); err != nil {
		t.Fatalf("Lark template format failed: %v", err)
	}

	output := buf.String()

	// Custom page shell
	if !strings.Contains(output, "<title>Lark Playground</title>") {
		t.Error("expected Lark title")
	}
	if !strings.Contains(output, `<script src="/lark.js"`) {
		t.Error("expected Lark script tag")
	}

	// Purple accent override
	if !strings.Contains(output, "--cm-accent: #7c3aed") {
		t.Error("expected purple accent override in light theme")
	}

	// Dark theme variables
	if !strings.Contains(output, ".dark {") {
		t.Error("expected dark theme class with variable overrides")
	}
	if !strings.Contains(output, "--cm-accent: #a78bfa") {
		t.Error("expected lighter purple in dark theme")
	}

	// Partials rendered the content correctly
	if !strings.Contains(output, "calc-block") {
		t.Error("expected calc-block from partials")
	}
	if !strings.Contains(output, "300") {
		t.Error("expected computed result (1000 * 0.3 = 300)")
	}

	// No ZgotmplZ
	if strings.Contains(output, "ZgotmplZ") {
		t.Error("ZgotmplZ escaping detected")
	}

	// Template is concise — count non-empty lines in the custom template
	lines := 0
	for line := range strings.SplitSeq(larkTemplate, "\n") {
		if strings.TrimSpace(line) != "" {
			lines++
		}
	}
	if lines > 15 {
		t.Errorf("Lark template should be <15 lines of custom code, got %d", lines)
	}
}

// TestHTMLFormatterPartialsAccessor verifies the PartialsTemplate accessor works.
func TestHTMLFormatterPartialsAccessor(t *testing.T) {
	p := PartialsTemplate()
	if p == "" {
		t.Fatal("PartialsTemplate() returned empty string")
	}
	if !strings.Contains(p, "cm-content") {
		t.Error("PartialsTemplate() should contain cm-content definition")
	}
	if !strings.Contains(p, "cm-frontmatter") {
		t.Error("PartialsTemplate() should contain cm-frontmatter definition")
	}
	if !strings.Contains(p, "cm-blocks") {
		t.Error("PartialsTemplate() should contain cm-blocks definition")
	}
}
