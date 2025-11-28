package format

import (
	_ "embed"
	"html/template"
	"io"
	"strings"

	"github.com/CalcMark/go-calcmark/format/display"
	"github.com/CalcMark/go-calcmark/spec/document"
)

//go:embed templates/default.html
var defaultHTMLTemplate string

// HTMLFormatter formats CalcMark documents as HTML.
// Uses an embedded template with modern styling.
type HTMLFormatter struct{}

// Extensions returns the file extensions handled by this formatter.
func (f *HTMLFormatter) Extensions() []string {
	return []string{".html", ".htm"}
}

// TemplateBlock represents a block for template rendering
type TemplateBlock struct {
	Type        string
	SourceLines []TemplateLine // For calc blocks with per-line results
	Error       string
	HTML        template.HTML // For text blocks
}

// TemplateLine represents a single source line with its result
type TemplateLine struct {
	Source string
	Result string // Formatted result for this line
}

// Format writes the document as HTML to the writer.
func (f *HTMLFormatter) Format(w io.Writer, doc *document.Document, opts Options) error {
	tmpl, err := template.New("html").Parse(defaultHTMLTemplate)
	if err != nil {
		return err
	}

	data := struct {
		Blocks []TemplateBlock
	}{}

	blocks := doc.GetBlocks()

	for _, node := range blocks {
		tb := TemplateBlock{}

		switch block := node.Block.(type) {
		case *document.CalcBlock:
			tb.Type = "calculation"

			// Build source lines with inline results
			sourceLines := block.Source()
			results := block.Results()
			tb.SourceLines = make([]TemplateLine, len(sourceLines))

			for i, line := range sourceLines {
				tl := TemplateLine{Source: line}
				// Add result if available for this line
				if i < len(results) && results[i] != nil {
					tl.Result = display.Format(results[i]) // Use display package for human-readable output
				}
				tb.SourceLines[i] = tl
			}

			if block.Error() != nil {
				tb.Error = block.Error().Error()
			}

		case *document.TextBlock:
			tb.Type = "text"
			// Call Render() to actively process markdown to HTML
			html := block.Render()
			if html == "" {
				// Fallback: just show source with line breaks
				html = strings.Join(block.Source(), "<br>")
			}
			tb.HTML = template.HTML(html) // Convert to template.HTML to mark as safe
		}

		data.Blocks = append(data.Blocks, tb)
	}

	return tmpl.Execute(w, data)
}
