package format

import (
	"encoding/json"
	"io"
	"maps"

	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// JSONFormatter formats CalcMark documents as JSON.
// Useful for programmatic consumption and integration with other tools.
type JSONFormatter struct{}

// Extensions returns the file extensions handled by this formatter.
func (f *JSONFormatter) Extensions() []string {
	return []string{".json"}
}

// JSONDocument represents the full document in JSON output
type JSONDocument struct {
	Frontmatter *JSONFrontmatter `json:"frontmatter,omitempty"`
	Blocks      []JSONBlock      `json:"blocks"`
}

// JSONFrontmatter represents frontmatter in JSON output
type JSONFrontmatter struct {
	Globals   map[string]string `json:"globals,omitempty"`
	Exchange  map[string]string `json:"exchange,omitempty"`
	Scale     *JSONScale        `json:"scale,omitempty"`
	ConvertTo *JSONConvertTo    `json:"convert_to,omitempty"`
}

// JSONScale represents scale config in JSON output
type JSONScale struct {
	Factor         float64  `json:"factor"`
	UnitCategories []string `json:"unit_categories,omitempty"`
}

// JSONConvertTo represents convert_to config in JSON output
type JSONConvertTo struct {
	System         string   `json:"system"`
	UnitCategories []string `json:"unit_categories,omitempty"`
}

// JSONBlock represents a single block in JSON output.
type JSONBlock struct {
	Type        string           `json:"type"`
	Source      []string         `json:"source"`
	Results     []JSONResult     `json:"results,omitempty"`
	Error       string           `json:"error,omitempty"`
	Diagnostics []JSONDiagnostic `json:"diagnostics,omitempty"`
	HTML        string           `json:"html,omitempty"`
}

// JSONResult represents a single evaluated statement's result.
// Value is the locale-formatted display string.
// Type is the CalcMark type name (e.g., "number", "currency").
// NumericValue, Unit, DateValue decompose the value for programmatic consumption.
type JSONResult struct {
	Source        string   `json:"source"`
	Value         string   `json:"value,omitempty"`
	Type          string   `json:"type,omitempty"`
	NumericValue  *float64 `json:"numeric_value,omitempty"`
	Unit          string   `json:"unit,omitempty"`
	DateValue     string   `json:"date_value,omitempty"`
	IsApproximate bool     `json:"is_approximate,omitempty"`
	IsExplicit    bool     `json:"is_explicit,omitempty"`
	Error         string   `json:"error,omitempty"`
	Variable      string   `json:"variable,omitempty"`
}

// JSONDiagnostic represents an error or warning with position info.
type JSONDiagnostic struct {
	Severity string `json:"severity"`
	Code     string `json:"code,omitempty"`
	Message  string `json:"message"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
}

// populateResult fills type-specific decomposition fields on a JSONResult.
func populateResult(jr *JSONResult, result types.Type) {
	switch v := result.(type) {
	case *types.Number:
		jr.Type = "number"
		f := v.Value.InexactFloat64()
		jr.NumericValue = &f
	case *types.Currency:
		jr.Type = "currency"
		f := v.Value.InexactFloat64()
		jr.NumericValue = &f
		jr.Unit = v.Code
	case *types.Quantity:
		jr.Type = "quantity"
		f := v.Value.InexactFloat64()
		jr.NumericValue = &f
		jr.Unit = v.Unit
		if v.IsNapkin {
			jr.IsApproximate = true
		}
		if v.IsExplicit {
			jr.IsExplicit = true
		}
	case *types.Rate:
		jr.Type = "rate"
		f := v.Amount.Value.InexactFloat64()
		jr.NumericValue = &f
		jr.Unit = v.CompoundUnit()
	case *types.Duration:
		jr.Type = "duration"
		f := v.Value.InexactFloat64()
		jr.NumericValue = &f
		jr.Unit = v.Unit
	case *types.Date:
		jr.Type = "date"
		jr.DateValue = v.Time.Format("2006-01-02")
	case *types.Time:
		jr.Type = "time"
	case *types.Percentage:
		jr.Type = "percentage"
		f := v.Value.InexactFloat64()
		jr.NumericValue = &f
	case *types.Boolean:
		jr.Type = "boolean"
	}
}

// Format writes the document as JSON to the writer.
func (f *JSONFormatter) Format(w io.Writer, doc *document.Document, opts Options) error {
	result := JSONDocument{
		Blocks: make([]JSONBlock, 0),
	}

	df := opts.getFormatter()

	// Add frontmatter if present
	if fm := doc.GetFrontmatter(); fm != nil {
		jfm := &JSONFrontmatter{}

		if len(fm.Globals) > 0 {
			jfm.Globals = make(map[string]string)
			maps.Copy(jfm.Globals, fm.Globals)
		}

		if len(fm.Exchange) > 0 {
			jfm.Exchange = make(map[string]string)
			for _, key := range fm.ExchangeKeys() {
				jfm.Exchange[key] = fm.Exchange[key].String()
			}
		}

		if fm.Scale != nil {
			f, _ := fm.Scale.Factor.Float64()
			jfm.Scale = &JSONScale{
				Factor:         f,
				UnitCategories: fm.Scale.UnitCategories,
			}
		}

		if fm.ConvertTo != nil {
			jfm.ConvertTo = &JSONConvertTo{
				System:         fm.ConvertTo.System,
				UnitCategories: fm.ConvertTo.UnitCategories,
			}
		}

		if len(fm.Globals) > 0 || len(fm.Exchange) > 0 || fm.Scale != nil || fm.ConvertTo != nil {
			result.Frontmatter = jfm
		}
	}

	// Add blocks
	for _, node := range doc.GetBlocks() {
		jb := JSONBlock{
			Source: node.Block.Source(),
		}

		switch block := node.Block.(type) {
		case *document.CalcBlock:
			jb.Type = "calculation"

			stmts := AlignResults(block)
			for _, stmt := range stmts {
				if stmt.IsBlank || stmt.IsResultLine {
					continue
				}
				jr := JSONResult{Source: stmt.Source}
				if stmt.Result != nil {
					populateResult(&jr, stmt.Result)
					jr.Value = df.Format(stmt.Result)
				} else if block.Error() != nil {
					// Per-result error: statement was reached but evaluation failed
					jr.Error = block.Error().Error()
				}
				jr.Variable = stmt.Variable
				jb.Results = append(jb.Results, jr)
			}

			// Add diagnostics with position info
			for _, diag := range block.Diagnostics() {
				jb.Diagnostics = append(jb.Diagnostics, JSONDiagnostic{
					Severity: diag.Severity,
					Code:     diag.Code,
					Message:  diag.Message,
					Line:     diag.Line,
					Column:   diag.Column,
				})
			}

			if block.Error() != nil {
				jb.Error = block.Error().Error()
			}

		case *document.TextBlock:
			jb.Type = "text"
			jb.HTML = block.Render()
		}

		result.Blocks = append(result.Blocks, jb)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
