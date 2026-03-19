package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/CalcMark/go-calcmark/format"
	"github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/impl/embedded"
	specDoc "github.com/CalcMark/go-calcmark/spec/document"
)

// runConvertEmbedded processes a standard Markdown file with embedded
// cm/calcmark fenced code blocks. Each block is evaluated independently
// and replaced with its Markdown output. All other content passes through
// byte-for-byte unchanged.
func runConvertEmbedded(filename string) error {
	// Validate flag interactions.
	if convertFormat != "" && convertFormat != "md" {
		return fmt.Errorf("--embedded only supports Markdown output; --to=%s is not valid with --embedded", convertFormat)
	}
	if convertTemplate != "" {
		return fmt.Errorf("--template is not valid with --embedded")
	}

	// Validate file path (accepts .md/.markdown).
	if err := validateReadFilePathEmbedded(filename); err != nil {
		return fmt.Errorf("invalid file: %w", err)
	}

	// Read input file.
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	// Security: Reject binary/non-text content.
	if err := validateFileContent(content); err != nil {
		return fmt.Errorf("invalid file: %w", err)
	}

	// Scan for cm/calcmark blocks.
	segments := embedded.Scan(string(content))

	// Determine output destination.
	var out *os.File
	if convertOutput != "" {
		out, err = os.Create(convertOutput)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer out.Close()
	} else {
		out = os.Stdout
	}

	// Process segments: passthrough text is written directly,
	// CalcMark blocks are evaluated and formatted.
	var errCount int
	for _, seg := range segments {
		switch seg.Kind {
		case embedded.Passthrough:
			fmt.Fprint(out, seg.Text)

		case embedded.CalcMarkBlock:
			result := evalBlock(seg.Text, seg.OpenLine)
			fmt.Fprint(out, result)
			if strings.HasPrefix(result, "> **CalcMark Error:**") {
				errCount++
			}
		}
	}

	if errCount > 0 {
		return fmt.Errorf("%d CalcMark block(s) had errors", errCount)
	}
	return nil
}

// evalBlock evaluates a single CalcMark block and returns its Markdown output.
// On error, returns an inline error blockquote with the host-file line number.
func evalBlock(source string, openLine int) string {
	// The content starts on the line after the opening fence.
	contentLine := openLine + 1

	doc, err := specDoc.NewDocument(source)
	if err != nil {
		return formatBlockError(err.Error(), contentLine)
	}

	eval := document.NewEvaluator()
	eval.SetDisplayFormatter(localeFormatter())
	if err := eval.Evaluate(doc); err != nil {
		return formatBlockError(err.Error(), contentLine)
	}

	// Check for block-level diagnostics (evaluation errors within statements).
	for _, node := range doc.GetBlocks() {
		if cb, ok := node.Block.(*specDoc.CalcBlock); ok {
			if cb.Error() != nil {
				// Continue — the formatter will include the error annotation.
				// We don't treat per-statement errors as fatal here; the
				// markdown formatter renders them inline.
			}
		}
	}

	var buf bytes.Buffer
	formatter := &format.MarkdownFormatter{}
	opts := format.Options{
		Verbose:             true,
		SuppressFrontmatter: true,
		DisplayFormatter:    eval.GetDisplayFormatter(),
	}
	if err := formatter.Format(&buf, doc, opts); err != nil {
		return formatBlockError(err.Error(), contentLine)
	}

	return buf.String()
}

// formatBlockError returns an inline error blockquote for a failed CalcMark block.
func formatBlockError(msg string, line int) string {
	return fmt.Sprintf("> **CalcMark Error:** %s (line %d)\n\n", msg, line)
}
