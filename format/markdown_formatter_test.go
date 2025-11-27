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
