package format

import (
	"io"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// Formatter formats CalcMark documents for output.
// All formatters must implement this interface.
type Formatter interface {
	// Format writes the formatted document to the writer
	Format(w io.Writer, doc *document.Document, opts Options) error

	// Extensions returns file extensions this formatter handles
	Extensions() []string
}

// Options controls formatter behavior
type Options struct {
	Verbose       bool   // Show calculation steps, types, units
	IncludeErrors bool   // Include error details
	Template      string // For template-based formatters (future use)
}
