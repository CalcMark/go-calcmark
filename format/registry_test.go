package format

import (
	"bytes"
	"io"
	"slices"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestGetFormatterExplicit tests explicit format selection
func TestGetFormatterExplicit(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"text", ".txt"},
		{"json", ".json"},
		{"cm", ".cm"},
		{"html", ".html"},
		{"md", ".md"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			f := GetFormatter(tt.format, "")
			if f == nil {
				t.Fatal("GetFormatter returned nil")
			}

			exts := f.Extensions()
			if !slices.Contains(exts, tt.expected) {
				t.Errorf("Expected formatter to handle %s, got extensions: %v", tt.expected, exts)
			}
		})
	}
}

// TestGetFormatterByExtension tests auto-detection from filename
func TestGetFormatterByExtension(t *testing.T) {
	tests := []struct {
		filename     string
		expectedExt  string
		expectedType string
	}{
		{"output.txt", ".txt", "text"},
		{"result.json", ".json", "json"},
		{"calc.cm", ".cm", "calcmark"},
		{"calc.calcmark", ".calcmark", "calcmark"},
		{"page.html", ".html", "html"},
		{"page.htm", ".htm", "html"},
		{"doc.md", ".md", "markdown"},
		{"doc.markdown", ".markdown", "markdown"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			f := GetFormatter("", tt.filename)
			if f == nil {
				t.Fatal("GetFormatter returned nil")
			}

			exts := f.Extensions()
			if !slices.Contains(exts, tt.expectedExt) {
				t.Errorf("Expected formatter to handle %s, got extensions: %v", tt.expectedExt, exts)
			}
		})
	}
}

// TestGetFormatterExplicitOverridesExtension tests that explicit format takes precedence
func TestGetFormatterExplicitOverridesExtension(t *testing.T) {
	// Request JSON format even though filename is .txt
	f := GetFormatter("json", "output.txt")
	if f == nil {
		t.Fatal("GetFormatter returned nil")
	}

	exts := f.Extensions()
	if !slices.Contains(exts, ".json") {
		t.Error("Explicit format should override filename extension")
	}
}

// TestGetFormatterDefaultToText tests default fallback to text formatter
func TestGetFormatterDefaultToText(t *testing.T) {
	// No format specified, unknown extension
	f := GetFormatter("", "output.xyz")
	if f == nil {
		t.Fatal("GetFormatter returned nil")
	}

	exts := f.Extensions()
	if !slices.Contains(exts, ".txt") {
		t.Error("Should default to text formatter for unknown extensions")
	}
}

// TestGetFormatterUnknownFormat tests handling of unknown format
func TestGetFormatterUnknownFormat(t *testing.T) {
	// Unknown format should fall back to default
	f := GetFormatter("unknown", "")
	if f == nil {
		t.Fatal("GetFormatter should not return nil for unknown format")
	}

	// Should default to text
	exts := f.Extensions()
	if !slices.Contains(exts, ".txt") {
		t.Error("Unknown format should default to text formatter")
	}
}

// TestRegisterCustomFormatter tests registering a custom formatter
func TestRegisterCustomFormatter(t *testing.T) {
	// Create a custom formatter
	custom := &customTestFormatter{}

	// Register it
	RegisterFormatter("custom", custom)

	// Retrieve it
	f := GetFormatter("custom", "")
	if f == nil {
		t.Fatal("Failed to retrieve custom formatter")
	}

	if _, ok := f.(*customTestFormatter); !ok {
		t.Error("Retrieved formatter is not the custom formatter")
	}
}

// customTestFormatter is a test formatter for registration tests
type customTestFormatter struct{}

func (f *customTestFormatter) Format(w io.Writer, doc *document.Document, opts Options) error {
	w.Write([]byte("custom"))
	return nil
}

func (f *customTestFormatter) Extensions() []string {
	return []string{".custom"}
}

// TestRegistryIsolation ensures formatters don't interfere with each other
func TestRegistryIsolation(t *testing.T) {
	// Get multiple formatters
	text := GetFormatter("text", "")
	json := GetFormatter("json", "")

	// Create test document
	doc, err := document.NewDocument("x = 10\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Format with both (even though output is empty for stubs)
	var buf1, buf2 bytes.Buffer
	text.Format(&buf1, doc, Options{})
	json.Format(&buf2, doc, Options{})

	// Just verify they don't panic
	// Actual output validation will be in individual formatter tests
}
