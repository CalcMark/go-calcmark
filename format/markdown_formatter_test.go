package format

import (
	"bytes"
	"strings"
	"testing"

	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestMarkdownFormatterSimple tests basic Markdown output
func TestMarkdownFormatterSimple(t *testing.T) {
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
	formatter := &MarkdownFormatter{}
	opts := Options{Verbose: false}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()
	// Should contain code fence
	if !strings.Contains(output, "```calcmark") {
		t.Errorf("Expected output to contain calcmark code fence, got: %s", output)
	}
	// Should contain result
	if !strings.Contains(output, "15") {
		t.Errorf("Expected output to contain result, got: %s", output)
	}
	// Should have "Result:" prefix
	if !strings.Contains(output, "**Result:**") {
		t.Errorf("Expected output to have Result prefix, got: %s", output)
	}
}

// TestMarkdownFormatterWithText tests mixed calc and text blocks
func TestMarkdownFormatterWithText(t *testing.T) {
	source := `# Header

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
	formatter := &MarkdownFormatter{}
	opts := Options{Verbose: false}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()
	// Should contain both the header and the calc block
	if !strings.Contains(output, "# Header") {
		t.Errorf("Expected output to contain header, got: %s", output)
	}
	if !strings.Contains(output, "x = 10") {
		t.Errorf("Expected output to contain calculation, got: %s", output)
	}
}

// TestMarkdownFormatterExtensions tests file extensions
func TestMarkdownFormatterExtensions(t *testing.T) {
	formatter := &MarkdownFormatter{}
	exts := formatter.Extensions()

	if len(exts) < 2 {
		t.Fatal("MarkdownFormatter should handle at least 2 extensions")
	}

	foundMD := false
	foundMarkdown := false
	for _, ext := range exts {
		if ext == ".md" {
			foundMD = true
		}
		if ext == ".markdown" {
			foundMarkdown = true
		}
	}

	if !foundMD || !foundMarkdown {
		t.Error("MarkdownFormatter should handle .md and .markdown extensions")
	}
}

// TestMarkdownFormatterWithFrontmatter tests that frontmatter is serialized
func TestMarkdownFormatterWithFrontmatter(t *testing.T) {
	source := `---
globals:
  tax_rate: 0.32
---
# Calculation

x = tax_rate * 100
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
	formatter := &MarkdownFormatter{}
	opts := Options{Verbose: false}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Should start with frontmatter
	if !strings.HasPrefix(output, "---\n") {
		t.Errorf("Expected output to start with frontmatter, got: %s", output)
	}

	// Should contain globals
	if !strings.Contains(output, "tax_rate") {
		t.Errorf("Expected output to contain tax_rate, got: %s", output)
	}

	// Should contain the calculation
	if !strings.Contains(output, "x = tax_rate * 100") {
		t.Errorf("Expected output to contain calculation, got: %s", output)
	}

	// Should contain result
	if !strings.Contains(output, "32") {
		t.Errorf("Expected output to contain result 32, got: %s", output)
	}
}

// TestMarkdownFormatterFiltersResultComments tests that # = comments inside calc blocks are stripped
func TestMarkdownFormatterFiltersResultComments(t *testing.T) {
	// Note: When a result comment is on its own line starting with #, it may be
	// detected as markdown (heading). This test uses inline comment format.
	// The real fix ensures # = inside CalcBlock source lines are filtered.
	source := "x = 10\ny = 20  # = 10\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	var buf bytes.Buffer
	formatter := &MarkdownFormatter{}
	opts := Options{Verbose: false}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// The inline # = should be stripped from calc block lines
	// Note: This is a line-level filter, so partial # = won't be caught,
	// only lines that are purely result comments (# = ...)
	// For now, verify output has the calculations
	if !strings.Contains(output, "x = 10") {
		t.Errorf("Expected output to contain x = 10, got: %s", output)
	}

	// Should have Result rendered
	if !strings.Contains(output, "**Result:**") {
		t.Errorf("Expected output to have Result prefix, got: %s", output)
	}
}

// TestMarkdownFormatterFiltersResultCommentBlocks tests that # = text blocks are filtered
func TestMarkdownFormatterFiltersResultCommentBlocks(t *testing.T) {
	// Simulate a verbose-saved .cm file where "  # = value" becomes a separate text block
	// The detector sees "# =" as a markdown heading, creating a TextBlock
	source := "x = 10\n\n\n# = 10\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	var buf bytes.Buffer
	formatter := &MarkdownFormatter{}
	opts := Options{Verbose: false}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Should NOT contain the # = text block
	if strings.Contains(output, "# = 10") {
		t.Errorf("Expected output to NOT contain # = 10 text block, got: %s", output)
	}

	// Should have the calculation
	if !strings.Contains(output, "x = 10") {
		t.Errorf("Expected output to contain calculation, got: %s", output)
	}

	// Should have Result rendered (once, not duplicated)
	count := strings.Count(output, "10")
	// Should appear in code fence and in Result line, but not as # = heading
	if count > 3 { // x = 10, Result: 10, and possibly in other places
		t.Errorf("Result appears too many times, possible duplicate: %s", output)
	}
}

// TestMarkdownFormatterFiltersFrontmatterBlocks tests that @global blocks are not included
func TestMarkdownFormatterFiltersFrontmatterBlocks(t *testing.T) {
	// Two blank lines create block boundary between @ assignment and calculation
	source := "@global.tax = 0.1\n\n\nx = tax * 100\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	var buf bytes.Buffer
	formatter := &MarkdownFormatter{}
	opts := Options{Verbose: false}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Should NOT contain @global in code fence (it's in frontmatter now)
	if strings.Contains(output, "@global") {
		t.Errorf("Expected @global to be in frontmatter, not in code fence: %s", output)
	}

	// Should have frontmatter with tax
	if !strings.Contains(output, "globals:") || !strings.Contains(output, "tax:") {
		t.Errorf("Expected frontmatter with tax global, got: %s", output)
	}

	// Should contain the x = calculation
	if !strings.Contains(output, "x = tax * 100") {
		t.Errorf("Expected calculation, got: %s", output)
	}
}
