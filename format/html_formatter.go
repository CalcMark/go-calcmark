package format

import (
	_ "embed"
	"fmt"
	gohtml "html"
	"html/template"
	"io"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/document"
)

//go:embed templates/default.html
var defaultHTMLTemplate string

// DefaultHTMLTemplate returns the embedded default HTML template.
// Use this to inspect the template data model or as a starting point for custom templates.
func DefaultHTMLTemplate() string {
	return defaultHTMLTemplate
}

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

// TemplateFrontmatter represents frontmatter for template rendering
type TemplateFrontmatter struct {
	Globals   []TemplateGlobal
	Exchange  []TemplateExchange
	Scale     string // e.g. "2x" or "0.5x [Length, Mass]"
	ConvertTo string // e.g. "imperial" or "si [Length]"
}

// TemplateGlobal represents a global variable for template rendering
type TemplateGlobal struct {
	Name  string
	Value string
}

// TemplateExchange represents an exchange rate for template rendering
type TemplateExchange struct {
	From string
	To   string
	Rate string
}

// Format writes the document as HTML to the writer.
func (f *HTMLFormatter) Format(w io.Writer, doc *document.Document, opts Options) error {
	// Use custom template if provided, otherwise use default
	templateContent := defaultHTMLTemplate
	if opts.Template != "" {
		templateContent = opts.Template
	}

	tmpl, err := template.New("html").Parse(templateContent)
	if err != nil {
		return err
	}

	data := struct {
		Frontmatter *TemplateFrontmatter
		Blocks      []TemplateBlock
	}{}

	df := opts.getFormatter()

	// Build frontmatter data if present
	if fm := doc.GetFrontmatter(); fm != nil {
		tfm := &TemplateFrontmatter{}

		// Add globals
		for _, name := range fm.GlobalKeys() {
			value := fm.Globals[name]
			tfm.Globals = append(tfm.Globals, TemplateGlobal{
				Name:  name,
				Value: value,
			})
		}

		// Add exchange rates
		for _, key := range fm.ExchangeKeys() {
			rate := fm.Exchange[key]
			from, to, err := document.ParseExchangeRateKey(key)
			if err == nil {
				tfm.Exchange = append(tfm.Exchange, TemplateExchange{
					From: from,
					To:   to,
					Rate: rate.String(),
				})
			}
		}

		// Add scale config
		if fm.Scale != nil {
			f, _ := fm.Scale.Factor.Float64()
			s := fmt.Sprintf("%gx", f)
			if len(fm.Scale.UnitCategories) > 0 {
				s += " [" + strings.Join(fm.Scale.UnitCategories, ", ") + "]"
			}
			tfm.Scale = s
		}

		// Add convert_to config
		if fm.ConvertTo != nil {
			s := fm.ConvertTo.System
			if len(fm.ConvertTo.UnitCategories) > 0 {
				s += " [" + strings.Join(fm.ConvertTo.UnitCategories, ", ") + "]"
			}
			tfm.ConvertTo = s
		}

		// Only set if there's content
		if len(tfm.Globals) > 0 || len(tfm.Exchange) > 0 || tfm.Scale != "" || tfm.ConvertTo != "" {
			data.Frontmatter = tfm
		}
	}

	blocks := doc.GetBlocks()

	for _, node := range blocks {
		tb := TemplateBlock{}

		switch block := node.Block.(type) {
		case *document.CalcBlock:
			tb.Type = "calculation"

			stmts := AlignResults(block)
			for _, stmt := range stmts {
				if stmt.IsBlank || stmt.IsResultLine {
					continue
				}
				tl := TemplateLine{Source: stmt.Source}
				if stmt.Result != nil {
					tl.Result = df.Format(stmt.Result)
				}
				tb.SourceLines = append(tb.SourceLines, tl)
			}

			if block.Error() != nil {
				tb.Error = block.Error().Error()
			}

		case *document.TextBlock:
			tb.Type = "text"
			// Call Render() to actively process markdown to HTML
			renderedHTML := block.Render()
			if renderedHTML == "" {
				// Fallback: escape each source line to prevent XSS, then join with <br>
				escaped := make([]string, len(block.Source()))
				for i, line := range block.Source() {
					escaped[i] = gohtml.EscapeString(line)
				}
				renderedHTML = strings.Join(escaped, "<br>")
			}
			tb.HTML = template.HTML(renderedHTML) // Convert to template.HTML to mark as safe
		}

		data.Blocks = append(data.Blocks, tb)
	}

	return tmpl.Execute(w, data)
}
