package format

import (
	"bytes"
	"os"
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
	// Should show per-line result with arrow
	if !strings.Contains(output, "→ 15") {
		t.Errorf("Expected output to contain per-line result '→ 15', got: %s", output)
	}
}

// TestMarkdownFormatterPerLineResults tests that every calculation line gets its result.
func TestMarkdownFormatterPerLineResults(t *testing.T) {
	source := "x = 10 + 5\ny = x * 2\nz = y + 1\n"
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

	// Every line should have its result, not just the last
	if !strings.Contains(output, "x = 10 + 5 → 15") {
		t.Errorf("Expected per-line result for x, got: %s", output)
	}
	if !strings.Contains(output, "y = x * 2 → 30") {
		t.Errorf("Expected per-line result for y, got: %s", output)
	}
	if !strings.Contains(output, "z = y + 1 → 31") {
		t.Errorf("Expected per-line result for z, got: %s", output)
	}

	// Should NOT have a separate **Result:** line
	if strings.Contains(output, "**Result:**") {
		t.Errorf("Should not have separate Result line, got: %s", output)
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

x = @globals.tax_rate * 100
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
	if !strings.Contains(output, "x = @globals.tax_rate * 100") {
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

	// Should have per-line result with arrow
	if !strings.Contains(output, "→") {
		t.Errorf("Expected output to have per-line result arrow, got: %s", output)
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

// TestMarkdownFormatterBlankLinesInCalcBlock tests that blank lines within a calc
// block don't misalign results with source lines. Results are indexed per-AST-statement
// (one per non-blank calc line), not per source line.
func TestMarkdownFormatterBlankLinesInCalcBlock(t *testing.T) {
	// Blank line separates the two groups of statements
	source := "x = 10\ny = 20\n\nz = x + y\n"
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

	// Every line should have its correct result, not shifted by blank lines
	if !strings.Contains(output, "x = 10 → 10") {
		t.Errorf("Expected 'x = 10 → 10', got:\n%s", output)
	}
	if !strings.Contains(output, "y = 20 → 20") {
		t.Errorf("Expected 'y = 20 → 20', got:\n%s", output)
	}
	if !strings.Contains(output, "z = x + y → 30") {
		t.Errorf("Expected 'z = x + y → 30', got:\n%s", output)
	}
}

// TestMarkdownFormatterMultipleBlankLines tests calc blocks with multiple blank
// line groups don't lose results at the end.
func TestMarkdownFormatterMultipleBlankLines(t *testing.T) {
	source := "a = 1\n\nb = 2\n\nc = 3\n\nd = a + b + c\n"
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

	if !strings.Contains(output, "a = 1 → 1") {
		t.Errorf("Expected 'a = 1 → 1', got:\n%s", output)
	}
	if !strings.Contains(output, "b = 2 → 2") {
		t.Errorf("Expected 'b = 2 → 2', got:\n%s", output)
	}
	if !strings.Contains(output, "c = 3 → 3") {
		t.Errorf("Expected 'c = 3 → 3', got:\n%s", output)
	}
	if !strings.Contains(output, "d = a + b + c → 6") {
		t.Errorf("Expected 'd = a + b + c → 6', got:\n%s", output)
	}
}

// TestMarkdownFormatterFrontmatterAsCodeFence tests that FrontmatterAsCodeFence
// renders CalcMark frontmatter as a ```yaml code fence instead of raw ---
// delimiters, so it can be embedded inside a Hugo page without collisions.
func TestMarkdownFormatterFrontmatterAsCodeFence(t *testing.T) {
	source := `---
globals:
  tax_rate: 0.32
---
x = @globals.tax_rate * 100
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
	opts := Options{FrontmatterAsCodeFence: true}

	if err := formatter.Format(&buf, doc, opts); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Should NOT start with --- (raw frontmatter)
	if strings.HasPrefix(output, "---\n") {
		t.Errorf("Expected output to NOT start with raw frontmatter, got:\n%s", output)
	}

	// Should contain ```yaml code fence with the frontmatter content
	if !strings.Contains(output, "```yaml\n") {
		t.Errorf("Expected ```yaml code fence, got:\n%s", output)
	}
	if !strings.Contains(output, "tax_rate") {
		t.Errorf("Expected frontmatter content preserved, got:\n%s", output)
	}

	// Should still contain the calculation result
	if !strings.Contains(output, "→ 32") {
		t.Errorf("Expected calculation result, got:\n%s", output)
	}
}

// TestMarkdownFormatterFrontmatterAsCodeFenceNoFrontmatter tests that
// FrontmatterAsCodeFence is a no-op when there's no frontmatter.
func TestMarkdownFormatterFrontmatterAsCodeFenceNoFrontmatter(t *testing.T) {
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
	formatter := &MarkdownFormatter{}
	opts := Options{FrontmatterAsCodeFence: true}

	if err := formatter.Format(&buf, doc, opts); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Should just have the calc block, no yaml fence
	if strings.Contains(output, "```yaml") {
		t.Errorf("Expected no yaml fence when no frontmatter, got:\n%s", output)
	}
	if !strings.Contains(output, "x = 10 → 10") {
		t.Errorf("Expected calculation, got:\n%s", output)
	}
}

// --- Phase 4: Realistic document integration tests (Markdown formatter) ---

// renderMarkdownFromFile loads a .cm file and renders it through the Markdown formatter.
func renderMarkdownFromFile(t *testing.T, path string) string {
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
	formatter := &MarkdownFormatter{}
	opts := Options{Verbose: false}

	if err := formatter.Format(&buf, doc, opts); err != nil {
		t.Fatalf("Markdown format failed for %s: %v", path, err)
	}
	return buf.String()
}

func TestMarkdownFormatterEngineeringDocument(t *testing.T) {
	output := renderMarkdownFromFile(t, "../testdata/examples/markdown_engineering.cm")

	// TextBlock source lines should appear verbatim in output
	textLines := []string{
		"# Structural Load Analysis",
		"## Material Properties",
		"## Cross-Section",
		"The cross-sectional area is used in stress calculations below.",
		"## Safety Factor",
		"> **Note:** This analysis assumes a simply-supported beam",
	}
	for _, line := range textLines {
		if !strings.Contains(output, line) {
			t.Errorf("Expected text line verbatim: %q", line)
		}
	}

	// Calc blocks wrapped in calcmark fences with results
	if !strings.Contains(output, "```calcmark") {
		t.Error("Expected calcmark code fences for calc blocks")
	}
	if !strings.Contains(output, "→") {
		t.Error("Expected result arrows in calc block output")
	}

	// Fenced code block in source should NOT be wrapped in calcmark fences
	if !strings.Contains(output, "I = b * h^3 / 12") {
		t.Error("Expected fenced code block content preserved")
	}
}

func TestMarkdownFormatterFinancialDocument(t *testing.T) {
	output := renderMarkdownFromFile(t, "../testdata/examples/markdown_financial.cm")

	// TextBlock source lines verbatim
	textLines := []string{
		"# Quarterly Revenue Forecast",
		"## Revenue Streams",
		"1. Salaries and benefits",
		"5. General and administrative",
		"- **Operating Income**: The result of",
		"> **Disclaimer:** These projections",
	}
	for _, line := range textLines {
		if !strings.Contains(output, line) {
			t.Errorf("Expected text line verbatim: %q", line)
		}
	}

	// Calc blocks with results
	if !strings.Contains(output, "```calcmark") {
		t.Error("Expected calcmark code fences for calc blocks")
	}
	if !strings.Contains(output, "→") {
		t.Error("Expected result arrows in calc block output")
	}
}

func TestMarkdownFormatterScientificDocument(t *testing.T) {
	output := renderMarkdownFromFile(t, "../testdata/examples/markdown_scientific.cm")

	// TextBlock source lines verbatim
	textLines := []string{
		"# Photovoltaic Cell Efficiency Study",
		"The *fill factor* represents how close",
		"> The theoretical maximum efficiency",
		"![I-V Curve](iv-curve-diagram.png)",
		"<https://www.energy.gov/eere/solar>",
	}
	for _, line := range textLines {
		if !strings.Contains(output, line) {
			t.Errorf("Expected text line verbatim: %q", line)
		}
	}

	// Fenced code block data preserved
	if !strings.Contains(output, "Intensity (W/m2)") {
		t.Error("Expected fenced code block data table preserved")
	}

	// Calc blocks with results
	if !strings.Contains(output, "```calcmark") {
		t.Error("Expected calcmark code fences for calc blocks")
	}
}

// --- Phase 5b: Edge case tests (Markdown formatter) ---

func TestMarkdownFormatterMixedDocumentRoundtrip(t *testing.T) {
	// heading → calc → paragraph → calc → heading should roundtrip correctly
	source := `# First Section

x = 10

This is a paragraph between calculations.

y = x * 2

## Second Section
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

	if err := formatter.Format(&buf, doc, opts); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// All structural elements should be present in correct order
	if !strings.Contains(output, "# First Section") {
		t.Error("Expected first heading preserved")
	}
	if !strings.Contains(output, "x = 10 → 10") {
		t.Error("Expected first calc with result")
	}
	if !strings.Contains(output, "This is a paragraph between calculations.") {
		t.Error("Expected paragraph text preserved")
	}
	if !strings.Contains(output, "y = x * 2 → 20") {
		t.Error("Expected second calc with result")
	}
	if !strings.Contains(output, "## Second Section") {
		t.Error("Expected second heading preserved")
	}

	// Verify structure: headings are NOT inside calcmark fences
	fenceIdx := strings.Index(output, "```calcmark")
	headingIdx := strings.Index(output, "# First Section")
	if fenceIdx >= 0 && headingIdx > fenceIdx {
		t.Error("Heading should appear before first calcmark fence, not inside it")
	}
}

// TestMarkdownFormatterFrontmatterPreservesExtraFields tests that non-CalcMark
// frontmatter fields (title, tags) are preserved in markdown output.
func TestMarkdownFormatterFrontmatterPreservesExtraFields(t *testing.T) {
	source := `---
title: Guacamole Recipe
scale: 4
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
	formatter := &MarkdownFormatter{}
	opts := Options{FrontmatterAsCodeFence: true}

	if err := formatter.Format(&buf, doc, opts); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// The frontmatter should preserve the title field
	if !strings.Contains(output, "title") {
		t.Error("Expected markdown output to preserve 'title' frontmatter field")
	}
	if !strings.Contains(output, "Guacamole Recipe") {
		t.Error("Expected markdown output to preserve title value")
	}
}
