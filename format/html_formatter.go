package format

import (
	_ "embed"
	"fmt"
	gohtml "html"
	"html/template"
	"io"
	"strings"
	"sync"

	"github.com/CalcMark/go-calcmark/format/display"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/microcosm-cc/bluemonday"
)

var (
	sanitizerOnce sync.Once
	sanitizerInst *bluemonday.Policy
)

// htmlSanitizer returns a cached bluemonday UGC policy for sanitizing
// markdown HTML output. Thread-safe via sync.Once.
func htmlSanitizer() *bluemonday.Policy {
	sanitizerOnce.Do(func() {
		sanitizerInst = bluemonday.UGCPolicy()
	})
	return sanitizerInst
}

//go:embed templates/calcmark.css
var styleCSS string

//go:embed templates/default.gohtml
var defaultHTMLTemplate string

//go:embed templates/preview.gohtml
var previewHTMLTemplate string

// StyleCSS returns the shared CalcMark CSS styles.
// Used by the default HTML template, the watch page, and any consumer
// that needs CalcMark-styled HTML output.
func StyleCSS() string {
	return styleCSS
}

// DefaultHTMLTemplate returns the embedded default HTML template.
// Use this to inspect the template data model or as a starting point for custom templates.
func DefaultHTMLTemplate() string {
	return defaultHTMLTemplate
}

// PreviewHTMLTemplate returns a content-only HTML fragment template for
// editor webview previews. No <html>/<head>/<body> wrapper — the editor
// provides its own shell with styles and scripts.
func PreviewHTMLTemplate() string {
	return previewHTMLTemplate
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
	DocLine     int           // 1-indexed document-absolute start line (for scroll sync)
}

// TemplateLine represents a single source line with its result
type TemplateLine struct {
	Source  string
	Result  string // Formatted result for this line
	DocLine int    // 1-indexed document-absolute line number (for scroll sync)
}

// TemplateFrontmatter represents frontmatter for template rendering
type TemplateFrontmatter struct {
	Globals   []TemplateGlobal
	Exchange  []TemplateExchange
	Scale     string // e.g. "2x" or "0.5x [Length, Mass]"
	ConvertTo string // e.g. "imperial" or "si [Length]"
	Extra     []TemplateExtra
}

// TemplateExtra represents a non-CalcMark frontmatter field
type TemplateExtra struct {
	Key   string
	Value string // formatted for display
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
		Style       template.CSS
		Frontmatter *TemplateFrontmatter
		Blocks      []TemplateBlock
		Content     template.HTML // Populated by embedded mode only (via calcmark.Convert).
	}{
		Style: template.CSS(styleCSS),
	}

	df := opts.getFormatter()
	// HTML output should use Unicode fractions (½, ⅓, ¾) for readability.
	cfg := df.Config()
	if !cfg.UnicodeFractions {
		cfg.UnicodeFractions = true
		df = display.NewFormatter(cfg)
	}

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

		// Add extra (non-CalcMark) frontmatter fields
		for _, ef := range fm.Extra {
			tfm.Extra = append(tfm.Extra, TemplateExtra{
				Key:   ef.Key,
				Value: formatExtraValue(ef.Value),
			})
		}

		// Only set if there's content
		if len(tfm.Globals) > 0 || len(tfm.Exchange) > 0 || tfm.Scale != "" || tfm.ConvertTo != "" || len(tfm.Extra) > 0 {
			data.Frontmatter = tfm
		}
	}

	blocks := doc.GetBlocks()

	// Compute document-absolute line offset: frontmatter lines come before blocks.
	docLine := 1
	if fm := doc.GetFrontmatter(); fm != nil {
		docLine += fm.LineCount()
	}

	for _, node := range blocks {
		tb := TemplateBlock{}
		blockStartLine := docLine

		switch block := node.Block.(type) {
		case *document.CalcBlock:
			tb.Type = "calculation"
			tb.DocLine = blockStartLine

			stmts := AlignResults(block)
			lineIdx := 0
			for _, stmt := range stmts {
				lineIdx++ // every aligned statement is a source line
				if stmt.IsBlank || stmt.IsResultLine {
					continue
				}
				tl := TemplateLine{
					Source:  stmt.Source,
					DocLine: blockStartLine + lineIdx - 1,
				}
				if stmt.Result != nil {
					tl.Result = df.Format(stmt.Result)
				}
				tb.SourceLines = append(tb.SourceLines, tl)
			}

			if block.Error() != nil {
				tb.Error = block.Error().Error()
			}
			docLine += len(block.Source())

		case *document.TextBlock:
			tb.Type = "text"
			tb.DocLine = blockStartLine
			// Call Render() to actively process markdown to HTML
			renderedHTML := block.Render()
			if renderedHTML == "" {
				// Fallback: escape each source line to prevent XSS, then join with <br>
				src := block.InterpolatedSource()
				escaped := make([]string, len(src))
				for i, line := range src {
					escaped[i] = gohtml.EscapeString(line)
				}
				renderedHTML = strings.Join(escaped, "<br>")
			}
			// Defense-in-depth: sanitize goldmark output with bluemonday.
			// UGCPolicy allows safe HTML tags (headings, lists, links, code, emphasis)
			// while stripping dangerous content (script, event handlers, data URIs).
			renderedHTML = htmlSanitizer().Sanitize(renderedHTML)
			tb.HTML = template.HTML(renderedHTML)
			docLine += len(block.Source())
		}

		data.Blocks = append(data.Blocks, tb)
	}

	return tmpl.Execute(w, data)
}

// formatExtraValue converts an arbitrary YAML value to a display string.
func formatExtraValue(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case []any:
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = fmt.Sprintf("%v", item)
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}
