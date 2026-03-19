package calcmark

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/format"
	"github.com/CalcMark/go-calcmark/format/display"
	impldoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/impl/embedded"
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
// It scans the input for cm/calcmark fenced code blocks, evaluates each
// independently, and reassembles the document. For "md" format, the output
// is the reassembled Markdown. For "html" format, the reassembled Markdown
// is converted to HTML via goldmark.
func convertEmbedded(input string, opts Options) (string, error) {
	segments := embedded.Scan(input)

	df := localeFormatter(opts.Locale)

	// Process segments: passthrough text is kept as-is,
	// CalcMark blocks are evaluated and formatted as Markdown.
	var out strings.Builder
	var errCount int
	for _, seg := range segments {
		switch seg.Kind {
		case embedded.Passthrough:
			out.WriteString(seg.Text)
		case embedded.CalcMarkBlock:
			result := evalEmbeddedBlock(seg.Text, seg.OpenLine, df)
			out.WriteString(result)
			if strings.HasPrefix(result, "> **CalcMark Error:**") {
				errCount++
			}
		}
	}

	assembled := out.String()

	// For markdown format, return the assembled markdown directly.
	if opts.Format == "md" {
		if errCount > 0 {
			return assembled, fmt.Errorf("%d CalcMark block(s) had errors", errCount)
		}
		return assembled, nil
	}

	// For HTML format, convert assembled markdown to HTML via goldmark.
	// TODO: implement in phase 1c
	if opts.Format == "html" {
		return "", errors.New("embedded HTML output not yet implemented")
	}

	// Other formats not supported for embedded mode.
	return "", fmt.Errorf("embedded mode does not support format %q (use md or html)", opts.Format)
}

// evalEmbeddedBlock evaluates a single CalcMark block and returns its Markdown output.
// On error, returns an inline error blockquote with the host-file line number.
func evalEmbeddedBlock(source string, openLine int, df display.Formatter) string {
	contentLine := openLine + 1

	doc, err := document.NewDocument(source)
	if err != nil {
		return formatEmbeddedBlockError(err.Error(), contentLine)
	}

	eval := impldoc.NewEvaluator()
	eval.SetDisplayFormatter(df)
	if err := eval.Evaluate(doc); err != nil {
		return formatEmbeddedBlockError(err.Error(), contentLine)
	}

	var buf bytes.Buffer
	formatter := &format.MarkdownFormatter{}
	fmtOpts := format.Options{
		Verbose:             true,
		SuppressFrontmatter: true,
		DisplayFormatter:    eval.GetDisplayFormatter(),
	}
	if err := formatter.Format(&buf, doc, fmtOpts); err != nil {
		return formatEmbeddedBlockError(err.Error(), contentLine)
	}

	return buf.String()
}

// formatEmbeddedBlockError returns an inline error blockquote for a failed CalcMark block.
func formatEmbeddedBlockError(msg string, line int) string {
	return fmt.Sprintf("> **CalcMark Error:** %s (line %d)\n\n", msg, line)
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
