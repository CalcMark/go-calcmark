package format

import (
	_ "embed"
	"fmt"
	gohtml "html"
	"html/template"
	"io"
	"strings"
	"sync"

	"github.com/CalcMark/go-calcmark/v2/format/display"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
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

//go:embed templates/partials.gohtml
var partialsTemplate string

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

// PartialsTemplate returns the embedded shared template partials.
// These define cm-content, cm-frontmatter, cm-blocks, cm-calc-block, and cm-text-block.
// Custom templates are automatically parsed with these partials so they can call them
// via {{template "cm-content" .}} or individual partials for layout control.
func PartialsTemplate() string {
	return partialsTemplate
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
	Type              string
	SourceLines       []TemplateLine // For calc blocks with per-line results
	Error             string
	HasLineDiagnostic bool          // True if any SourceLine has a per-line Error
	Warnings          []string      // Diagnostic warnings (e.g., reserved keyword as variable name)
	HTML              template.HTML // For text blocks
	DocLine           int           // 1-indexed document-absolute start line (for scroll sync)
}

// TemplateLine represents a single source line with its result
type TemplateLine struct {
	Source      string
	Result      string // Formatted result for this line
	Error       string // Error message for this line (empty if successful)
	IsCascading bool   // True if this is a cascading error (hint severity)
	DocLine     int    // 1-indexed document-absolute line number (for scroll sync)
}

// TemplateFrontmatter represents frontmatter for template rendering
type TemplateFrontmatter struct {
	Globals          []TemplateGlobal
	Exchange         []TemplateExchange
	Scale            string // e.g. "2x" or "0.5x [Length, Mass]"
	ConvertTo        string // e.g. "imperial" or "si [Length]"
	FiscalYearStarts string // e.g. "July" or "July 15"
	Extra            []TemplateExtra
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

	// Parse shared partials first, then the page template into the same set.
	// This makes cm-content, cm-frontmatter, cm-blocks, etc. available to
	// both internal and custom templates via {{template "cm-..." .}}.
	tmpl, err := template.New("html").Parse(partialsTemplate)
	if err != nil {
		return fmt.Errorf("partials parse error: %w", err)
	}
	if _, err := tmpl.Parse(templateContent); err != nil {
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

		// Add fiscal year config
		if fm.FiscalYearStarts != nil {
			monthNames := []string{"", "January", "February", "March", "April", "May", "June",
				"July", "August", "September", "October", "November", "December"}
			fc := fm.FiscalYearStarts
			if fc.Month >= 1 && fc.Month <= 12 {
				s := monthNames[fc.Month]
				if fc.Day > 1 {
					s += fmt.Sprintf(" %d", fc.Day)
				}
				tfm.FiscalYearStarts = s
			}
		}

		// Add extra (non-CalcMark) frontmatter fields
		for _, ef := range fm.Extra {
			tfm.Extra = append(tfm.Extra, TemplateExtra{
				Key:   ef.Key,
				Value: formatExtraValue(ef.Value),
			})
		}

		// Only set if there's content
		if len(tfm.Globals) > 0 || len(tfm.Exchange) > 0 || tfm.Scale != "" || tfm.ConvertTo != "" || tfm.FiscalYearStarts != "" || len(tfm.Extra) > 0 {
			data.Frontmatter = tfm
		}
	}

	blocks := doc.GetBlocks()

	// Compute document-absolute line offset: frontmatter lines come before blocks.
	docLine := 1
	if fm := doc.GetFrontmatter(); fm != nil {
		docLine += fm.LineCount()
	}

	// Cross-block markdown render (issue #129). Footnote references
	// and their definitions can live in separate TextBlocks split by
	// CalcBlocks. Per-block goldmark renders see only one side and
	// leave the reference as literal `[^x]`. Render all text blocks
	// as one markdown pass so footnote linkage survives, then map
	// the per-block HTML back into the regular iteration below. The
	// trailing footnotes section is appended to the last text block.
	textBlocks := make([]*document.TextBlock, 0, len(blocks))
	for _, n := range blocks {
		if tb, ok := n.Block.(*document.TextBlock); ok {
			textBlocks = append(textBlocks, tb)
		}
	}
	joinedHTML, joinedTrailing, joinedOK := document.RenderTextBlocksJoined(textBlocks)
	joinedHTMLByBlock := make(map[*document.TextBlock]string, len(textBlocks))
	if joinedOK {
		for i, tb := range textBlocks {
			joinedHTMLByBlock[tb] = joinedHTML[i]
		}
		if joinedTrailing != "" && len(textBlocks) > 0 {
			last := textBlocks[len(textBlocks)-1]
			joinedHTMLByBlock[last] = strings.TrimSpace(joinedHTMLByBlock[last]) + "\n" + joinedTrailing
		}
	}

	for _, node := range blocks {
		tb := TemplateBlock{}
		blockStartLine := docLine

		switch block := node.Block.(type) {
		case *document.CalcBlock:
			tb.Type = "calculation"
			tb.DocLine = blockStartLine

			// Build diagnostic-by-line map for per-line error display
			diagByLine := make(map[int]document.Diagnostic)
			for _, d := range block.Diagnostics() {
				if d.Line > 0 {
					diagByLine[d.Line] = d
				}
			}

			stmts := AlignResults(block)
			lineNum := 0
			for _, stmt := range stmts {
				lineNum++ // count every source line (including blanks) to match diagnostic Line numbers
				if stmt.IsBlank || stmt.IsResultLine {
					continue
				}
				tl := TemplateLine{
					Source:  stmt.Source,
					DocLine: blockStartLine + lineNum - 1,
				}
				if stmt.Result != nil {
					tl.Result = df.Format(stmt.Result)
				} else if d, ok := diagByLine[lineNum]; ok {
					tl.Error = d.Message
					tl.IsCascading = d.Code == "cascading_error"
					tb.HasLineDiagnostic = true
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
			// Prefer the joined-render HTML when the joined pass
			// succeeded (footnote references resolve to definitions
			// in other text blocks). Empty HTML is a valid joined
			// result — a block consisting only of `[^x]: definition`
			// lines contributes nothing inline because goldmark
			// extracts the definition into the trailing footnotes
			// section. Fall back to per-block Render() only when
			// the joined render couldn't split cleanly.
			var renderedHTML string
			joinedHTML, joined := joinedHTMLByBlock[block]
			if joined {
				renderedHTML = joinedHTML
				// Post-process interpolation sentinels the joined
				// renderer left in place (mirrors TextBlock.Render).
				renderedHTML = strings.ReplaceAll(renderedHTML, "\x02", `<span class="cm-interpolated">`)
				renderedHTML = strings.ReplaceAll(renderedHTML, "\x03", `</span>`)
			} else {
				renderedHTML = block.Render()
			}
			if renderedHTML == "" && !joined {
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
			for _, diag := range block.Diagnostics() {
				msg := diag.Message
				if diag.Detailed != "" {
					msg += " — " + diag.Detailed
				}
				tb.Warnings = append(tb.Warnings, msg)
			}
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
