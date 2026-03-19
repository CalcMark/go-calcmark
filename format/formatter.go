package format

import (
	"io"

	"github.com/CalcMark/go-calcmark/format/display"
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

	// DisplayFormatter is the locale-aware formatter for rendering values.
	// When zero-value, formatters fall back to display.Format() (en-US default).
	DisplayFormatter display.Formatter

	// FrontmatterAsCodeFence renders CalcMark frontmatter as a ```yaml code
	// fence instead of raw --- delimiters. Use when embedding CalcMark output
	// inside a Hugo page that has its own YAML frontmatter.
	FrontmatterAsCodeFence bool

	// SuppressFrontmatter skips frontmatter serialization (used by embedded mode).
	SuppressFrontmatter bool
}

// getFormatter returns the DisplayFormatter from Options, or the default en-US formatter.
func (o Options) getFormatter() display.Formatter {
	// Zero-value check: DisplayConfig with empty separators means no formatter was set
	if o.DisplayFormatter.Config().DecimalSep == "" {
		return display.DefaultFormatter()
	}
	return o.DisplayFormatter
}
