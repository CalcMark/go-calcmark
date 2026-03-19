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
