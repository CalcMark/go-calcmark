package calcmark

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/format"
	"github.com/CalcMark/go-calcmark/format/display"
	impldoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// Mode selects the conversion pipeline.
type Mode int

const (
	// CM processes the entire input as a CalcMark document.
	CM Mode = iota
	// Embedded processes a Markdown document with embedded cm/calcmark fenced code blocks.
	Embedded
)

// Options configures the Convert pipeline.
type Options struct {
	Mode     Mode   // CM or Embedded (default: CM)
	Format   string // Output format: "html", "md", "text", "json" (default: "html")
	Template string // Go template content for wrapping HTML output (optional)
	Locale   string // BCP 47 locale for number formatting (default: "en-US")
}

// Convert processes CalcMark input and returns formatted output.
//
// In CM mode, the entire input is parsed as a CalcMark document, evaluated,
// and formatted to the requested output format.
//
// In Embedded mode, the input is scanned for cm/calcmark fenced code blocks.
// Each block is evaluated independently and replaced with its formatted output.
// Surrounding Markdown prose passes through unchanged.
//
// HTML output returns a fragment by default (no <html>/<head>/<body> wrapper).
// When Template is set, the fragment is wrapped using the provided Go template.
func Convert(input string, opts Options) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", errors.New("empty input")
	}

	// Default format is html
	if opts.Format == "" {
		opts.Format = "html"
	}

	switch opts.Mode {
	case CM:
		return convertCM(input, opts)
	case Embedded:
		return convertEmbedded(input, opts)
	default:
		return "", fmt.Errorf("unknown mode: %d", opts.Mode)
	}
}

// convertCM handles the pure CalcMark conversion pipeline.
func convertCM(input string, opts Options) (string, error) {
	// Parse
	doc, err := document.NewDocument(input)
	if err != nil {
		return "", fmt.Errorf("parse error: %w", err)
	}

	// Evaluate
	eval := impldoc.NewEvaluator()
	eval.SetDisplayFormatter(localeFormatter(opts.Locale))
	if err := eval.Evaluate(doc); err != nil {
		return "", fmt.Errorf("evaluation error: %w", err)
	}

	// Format
	formatter := format.GetFormatter(opts.Format, "")

	// For HTML without a template, use the preview (fragment) template.
	// For HTML with a template, use the caller's template.
	fmtOpts := format.Options{
		Verbose:          true,
		DisplayFormatter: eval.GetDisplayFormatter(),
	}
	if opts.Format == "html" {
		if opts.Template != "" {
			fmtOpts.Template = opts.Template
		} else {
			fmtOpts.Template = format.PreviewHTMLTemplate()
		}
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, doc, fmtOpts); err != nil {
		return "", fmt.Errorf("format error: %w", err)
	}

	return buf.String(), nil
}

// convertEmbedded handles the embedded Markdown conversion pipeline.
func convertEmbedded(input string, opts Options) (string, error) {
	return "", errors.New("embedded mode not yet implemented")
}

// localeFormatter builds a display.Formatter from a BCP 47 locale string.
func localeFormatter(locale string) display.Formatter {
	if locale == "" {
		return display.DefaultFormatter()
	}
	cfg, err := display.NewConfig(locale)
	if err != nil {
		return display.DefaultFormatter()
	}
	return display.NewFormatter(cfg)
}
