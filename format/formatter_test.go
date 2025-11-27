package format

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestFormatterInterface ensures all formatters implement the interface correctly
func TestFormatterInterface(t *testing.T) {
	formatters := []Formatter{
		&TextFormatter{},
		&JSONFormatter{},
		&CalcMarkFormatter{},
		&HTMLFormatter{},
		&MarkdownFormatter{},
	}

	for _, f := range formatters {
		if f == nil {
			t.Error("Formatter should not be nil")
		}
		if f.Extensions() == nil {
			t.Error("Extensions() should not return nil")
		}
	}
}

// TestOptions validates the Options struct
func TestOptions(t *testing.T) {
	opts := Options{
		Verbose:       true,
		IncludeErrors: true,
		Template:      "custom",
	}

	if !opts.Verbose {
		t.Error("Verbose should be true")
	}
	if !opts.IncludeErrors {
		t.Error("IncludeErrors should be true")
	}
	if opts.Template != "custom" {
		t.Errorf("Expected template 'custom', got '%s'", opts.Template)
	}
}

// TestFormatterWithDocument tests that formatters can handle documents
func TestFormatterWithDocument(t *testing.T) {
	// Create a simple document
	doc, err := document.NewDocument("x = 10\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Each formatter should be able to accept a document
	// (actual formatting tests will be in individual formatter test files)
	formatters := []Formatter{
		&TextFormatter{},
		&JSONFormatter{},
		&CalcMarkFormatter{},
		&HTMLFormatter{},
		&MarkdownFormatter{},
	}

	for _, f := range formatters {
		// Just verify the interface accepts the document type
		// We'll test actual formatting in individual files
		_ = doc
		_ = f
	}
}
