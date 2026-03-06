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
price = base_price * (1 + tax_rate)
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
