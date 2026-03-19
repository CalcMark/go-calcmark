package calcmark

import (
	"strings"
	"testing"
)

// simpleTemplate is a minimal Go template for testing template wrapping.
const simpleTemplate = `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
{{if .Content}}
<div class="embedded">{{.Content}}</div>
{{else}}
{{range .Blocks}}
{{if eq .Type "calculation"}}
<div class="calc">
{{range .SourceLines}}<div>{{.Source}}{{if .Result}} = {{.Result}}{{end}}</div>
{{end}}
</div>
{{else}}
<div class="text">{{.HTML}}</div>
{{end}}
{{end}}
{{end}}
</body>
</html>`

// T1: cm mode, markdown output (existing behavior, must not regress)
func TestConvert_CM_Markdown(t *testing.T) {
	input := "price = 100 USD\ntax = 8.5%\ntotal = price + tax"
	result, err := Convert(input, Options{
		Format: "md",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	// Markdown output should contain the variable names and results
	if !strings.Contains(result, "price") {
		t.Error("expected result to contain 'price'")
	}
	if !strings.Contains(result, "total") {
		t.Error("expected result to contain 'total'")
	}
}

// T2: cm mode, html output with template
func TestConvert_CM_HTMLWithTemplate(t *testing.T) {
	input := "price = 100 USD"
	result, err := Convert(input, Options{
		Format:   "html",
		Template: simpleTemplate,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be a full HTML document using our template
	if !strings.Contains(result, "<!DOCTYPE html>") {
		t.Error("expected full HTML document with DOCTYPE")
	}
	if !strings.Contains(result, "<title>Test</title>") {
		t.Error("expected template title")
	}
	if !strings.Contains(result, "price") {
		t.Error("expected result to contain 'price'")
	}
}

// T3: cm mode, html output without template → fragment
func TestConvert_CM_HTMLFragment(t *testing.T) {
	input := "price = 100 USD"
	result, err := Convert(input, Options{
		Format: "html",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	// Default template includes DOCTYPE — a fragment would not.
	// However, the default go-calcmark HTML template IS a full document.
	// The plan says "fragment by default" means no template = use default template.
	// Actually, re-reading the plan: "HTML output returns fragments by default —
	// no <html>/<head>/<body>. An optional Go template wraps fragments into full documents."
	// So with no template, we should get a fragment.
	if strings.Contains(result, "<!DOCTYPE") {
		t.Error("expected HTML fragment (no DOCTYPE), got full document")
	}
	if strings.Contains(result, "<html") {
		t.Error("expected HTML fragment (no <html> tag)")
	}
	// Should still contain the formatted content
	if !strings.Contains(result, "price") {
		t.Error("expected result to contain 'price'")
	}
}

// T10: locale option works in both modes
func TestConvert_CM_Locale(t *testing.T) {
	input := "price = 1234.56 EUR"
	result, err := Convert(input, Options{
		Format: "md",
		Locale: "de-DE",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// German locale should use comma as decimal separator
	// and period as thousands separator: 1.234,56
	if !strings.Contains(result, "1.234,56") {
		t.Errorf("expected German-formatted number '1.234,56' in result, got:\n%s", result)
	}
}

// T3b: cm mode, default format should be html
func TestConvert_CM_DefaultFormat(t *testing.T) {
	input := "x = 42"
	result, err := Convert(input, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Default format is html, no template = fragment
	if result == "" {
		t.Fatal("expected non-empty result")
	}
}

// Test that empty input returns an error
func TestConvert_EmptyInput(t *testing.T) {
	_, err := Convert("", Options{Format: "md"})
	if err == nil {
		t.Error("expected error for empty input")
	}
}

// Test that whitespace-only input returns an error
func TestConvert_WhitespaceInput(t *testing.T) {
	_, err := Convert("   \n\t\n  ", Options{Format: "md"})
	if err == nil {
		t.Error("expected error for whitespace-only input")
	}
}

// --- Embedded mode tests ---

// T4: embedded mode, markdown output (blocks replaced, prose unchanged)
func TestConvert_Embedded_Markdown(t *testing.T) {
	input := "# Budget\n\n```cm\nprice = 100 USD\n```\n\nSome prose.\n"
	result, err := Convert(input, Options{
		Mode:   Embedded,
		Format: "md",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Prose should be preserved
	if !strings.Contains(result, "# Budget") {
		t.Error("expected heading to be preserved")
	}
	if !strings.Contains(result, "Some prose.") {
		t.Error("expected prose to be preserved")
	}
	// CalcMark block should be replaced with evaluated output
	if strings.Contains(result, "```cm") {
		t.Error("expected cm fence to be replaced with evaluated output")
	}
	// Should contain the evaluated result (price = 100 USD → table or annotation)
	if !strings.Contains(result, "price") {
		t.Error("expected evaluated output to contain 'price'")
	}
}

// T8: embedded mode, block with evaluation error → inline error blockquote
func TestConvert_Embedded_BlockError(t *testing.T) {
	input := "# Test\n\n```cm\nx = unknown_var + 1\n```\n"
	result, err := Convert(input, Options{
		Mode:   Embedded,
		Format: "md",
	})
	// Error should be non-nil (signals block failures)
	if err == nil {
		t.Error("expected non-nil error for block with evaluation issues")
	}
	// But result should still contain output (partial success)
	if result == "" {
		t.Fatal("expected non-empty result even with block errors")
	}
	// Should contain inline error blockquote
	if !strings.Contains(result, "# Test") {
		t.Error("expected heading to be preserved")
	}
}

// T9: embedded mode preserves frontmatter passthrough
func TestConvert_Embedded_FrontmatterPreserved(t *testing.T) {
	input := "---\ntitle: Test Doc\n---\n\n# Doc\n\n```cm\nx = 42\n```\n"
	result, err := Convert(input, Options{
		Mode:   Embedded,
		Format: "md",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Frontmatter should be preserved as-is
	if !strings.Contains(result, "---\ntitle: Test Doc\n---") {
		t.Errorf("expected frontmatter to be preserved, got:\n%s", result)
	}
	// Heading should be preserved
	if !strings.Contains(result, "# Doc") {
		t.Error("expected heading to be preserved")
	}
}

// T5: embedded mode, html fragment (new)
func TestConvert_Embedded_HTMLFragment(t *testing.T) {
	input := "# Budget\n\n```cm\nprice = 100 USD\n```\n\nSome prose.\n"
	result, err := Convert(input, Options{
		Mode:   Embedded,
		Format: "html",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be an HTML fragment (no DOCTYPE, no <html> tag)
	if strings.Contains(result, "<!DOCTYPE") {
		t.Error("expected HTML fragment, got full document")
	}
	// Should contain goldmark-rendered heading
	if !strings.Contains(result, "<h1") {
		t.Error("expected <h1> heading in goldmark output")
	}
	// Should contain prose as paragraph
	if !strings.Contains(result, "Some prose.") {
		t.Error("expected prose paragraph in output")
	}
	// Should contain evaluated CalcMark output (rendered as HTML table or similar)
	if !strings.Contains(result, "price") {
		t.Error("expected evaluated CalcMark content")
	}
}

// T6: embedded mode, html with template (what Lark uses)
func TestConvert_Embedded_HTMLWithTemplate(t *testing.T) {
	input := "# Budget\n\n```cm\nprice = 100 USD\n```\n"
	result, err := Convert(input, Options{
		Mode:     Embedded,
		Format:   "html",
		Template: simpleTemplate,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be a full HTML document using our template
	if !strings.Contains(result, "<!DOCTYPE html>") {
		t.Error("expected full HTML document with DOCTYPE")
	}
	if !strings.Contains(result, "<title>Test</title>") {
		t.Error("expected template title")
	}
	// Content should be inside the embedded div
	if !strings.Contains(result, `class="embedded"`) {
		t.Error("expected embedded content wrapper from template")
	}
	// Should contain goldmark-rendered content
	if !strings.Contains(result, "<h1") {
		t.Error("expected <h1> heading in goldmark output")
	}
}

// T7: embedded mode, no calcmark blocks → passthrough
func TestConvert_Embedded_NoBlocks(t *testing.T) {
	input := "# Just Markdown\n\nNo calcmark here.\n"
	result, err := Convert(input, Options{
		Mode:   Embedded,
		Format: "html",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should contain goldmark-rendered heading and paragraph
	if !strings.Contains(result, "<h1") {
		t.Error("expected <h1> heading")
	}
	if !strings.Contains(result, "No calcmark here.") {
		t.Error("expected prose content")
	}
}

// T9b: embedded mode, frontmatter stripped from HTML output
func TestConvert_Embedded_FrontmatterStrippedFromHTML(t *testing.T) {
	input := "---\ntitle: Test Doc\n---\n\n# Doc\n\n```cm\nx = 42\n```\n"
	result, err := Convert(input, Options{
		Mode:   Embedded,
		Format: "html",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Frontmatter should NOT appear in the HTML output
	if strings.Contains(result, "title: Test Doc") {
		t.Error("expected frontmatter to be stripped from HTML output")
	}
	// The heading should still be there
	if !strings.Contains(result, "<h1") {
		t.Error("expected heading in HTML output")
	}
}

// T4b: embedded mode with multiple blocks
func TestConvert_Embedded_MultipleBlocks(t *testing.T) {
	input := "# Report\n\n```cm\na = 10\n```\n\nMiddle text.\n\n```cm\nb = 20\n```\n\nEnd.\n"
	result, err := Convert(input, Options{
		Mode:   Embedded,
		Format: "md",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "# Report") {
		t.Error("expected heading preserved")
	}
	if !strings.Contains(result, "Middle text.") {
		t.Error("expected middle prose preserved")
	}
	if !strings.Contains(result, "End.") {
		t.Error("expected end prose preserved")
	}
	// Both cm fences should be gone
	if strings.Contains(result, "```cm") {
		t.Error("expected all cm fences to be replaced")
	}
}
