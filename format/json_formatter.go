package format

import (
	"encoding/json"
	"io"
	"maps"

	"github.com/CalcMark/go-calcmark/v2/format/display"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
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
	Type               string           `json:"type"`
	Source             []string         `json:"source"`
	InterpolatedSource []string         `json:"interpolated_source,omitempty"`
	Results            []JSONResult     `json:"results,omitempty"`
	Error              string           `json:"error,omitempty"`
	Diagnostics        []JSONDiagnostic `json:"diagnostics,omitempty"`
	HTML               string           `json:"html,omitempty"`
}

// JSONResult represents a single evaluated statement's result.
// Value is the locale-formatted display string.
// Type is the CalcMark type name (e.g., "number", "currency", "fraction").
// NumericValue, Unit, DateValue decompose the value for programmatic consumption.
type JSONResult struct {
	Source        string   `json:"source"`
	Value         string   `json:"value,omitempty"`
	Type          string   `json:"type,omitempty"`
	NumericValue  *float64 `json:"numeric_value,omitempty"`
	Numerator     *int64   `json:"numerator,omitempty"`
	Denominator   *int64   `json:"denominator,omitempty"`
	Unit          string   `json:"unit,omitempty"`
	DateValue     string   `json:"date_value,omitempty"`
	IsApproximate bool     `json:"is_approximate,omitempty"`
	IsExplicit    bool     `json:"is_explicit,omitempty"`
	Error         string   `json:"error,omitempty"`
	Variable      string   `json:"variable,omitempty"`
	// Period decomposition (v2.0). Populated only when Type ==
	// "period". Start and End are ISO date strings (YYYY-MM-DD)
	// covering the closed-interval span. PeriodKind is the canonical
	// kind name: calendar_quarter / fiscal_quarter / calendar_year /
	// fiscal_year / calendar_month / named_month / relative_week /
	// relative_month / relative_year / relative_quarter /
	// relative_fiscal_quarter / relative_fiscal_year / custom.
	PeriodStart string `json:"period_start,omitempty"`
	PeriodEnd   string `json:"period_end,omitempty"`
	PeriodKind  string `json:"period_kind,omitempty"`
	// Elements decomposes an array result (Type == "array") one entry
	// per row, each shaped like a scalar result (go-calcmark#118).
	Elements []JSONResult `json:"elements,omitempty"`
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
	case *types.Array:
		jr.Type = "array"
	case *types.Text:
		jr.Type = "text"
	case *types.Number:
		jr.Type = "number"
		f := v.Value.InexactFloat64()
		jr.NumericValue = &f
		if v.IsNapkin {
			jr.IsApproximate = true
		}
	case *types.Currency:
		jr.Type = "currency"
		f := v.Value.InexactFloat64()
		jr.NumericValue = &f
		jr.Unit = v.Code
		if v.IsNapkin {
			jr.IsApproximate = true
		}
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
	case *types.Fraction:
		jr.Type = "fraction"
		f := v.ToDecimal().InexactFloat64()
		jr.NumericValue = &f
		if v.Num().IsInt64() {
			n := v.Num().Int64()
			jr.Numerator = &n
		}
		if v.Denom().IsInt64() {
			d := v.Denom().Int64()
			jr.Denominator = &d
		}
		if v.Unit != "" {
			jr.Unit = v.Unit
		}
		if v.IsNapkin {
			jr.IsApproximate = true
		}
	case *types.Boolean:
		jr.Type = "boolean"
	case *types.Period:
		// v2.0 Period: human-readable label via String() in Value;
		// machine-readable Start/End/Kind for downstream consumers.
		jr.Type = "period"
		if v.Start != nil {
			jr.PeriodStart = v.Start.Time.Format("2006-01-02")
		}
		if v.End != nil {
			jr.PeriodEnd = v.End.Time.Format("2006-01-02")
		}
		jr.PeriodKind = periodKindName(v.Kind)
	}
}

// periodKindName returns the canonical lowercased name for a
// PeriodKind. JSON consumers (cmw, LSP, integrations) match against
// these strings — keep them stable. New kinds added to types.PeriodKind
// must register a name here too; "unknown" is a fallback that signals
// drift between the two enums.
func periodKindName(k types.PeriodKind) string {
	switch k {
	case types.PeriodCalendarQuarter:
		return "calendar_quarter"
	case types.PeriodFiscalQuarter:
		return "fiscal_quarter"
	case types.PeriodCalendarMonth:
		return "calendar_month"
	case types.PeriodCalendarYear:
		return "calendar_year"
	case types.PeriodFiscalYear:
		return "fiscal_year"
	case types.PeriodNamedMonth:
		return "named_month"
	case types.PeriodRelativeWeek:
		return "relative_week"
	case types.PeriodRelativeMonth:
		return "relative_month"
	case types.PeriodRelativeYear:
		return "relative_year"
	case types.PeriodRelativeQuarter:
		return "relative_quarter"
	case types.PeriodRelativeFiscalQuarter:
		return "relative_fiscal_quarter"
	case types.PeriodRelativeFiscalYear:
		return "relative_fiscal_year"
	case types.PeriodCustom:
		return "custom"
	}
	return "unknown"
}

// arrayElements decomposes each array element like a scalar result.
func arrayElements(arr *types.Array, df display.Formatter) []JSONResult {
	out := make([]JSONResult, len(arr.Elements))
	for i, el := range arr.Elements {
		populateResult(&out[i], el)
		out[i].Value = df.Format(el)
	}
	return out
}

// statementError returns the message of the first error-severity
// diagnostic on the given block-relative line. Falls back to the
// block-level error only when no diagnostic is line-attributed (parse
// failures and other block-wide errors), so a statement never inherits
// an error that belongs to another line.
func statementError(block *document.CalcBlock, line int) string {
	attributed := false
	for _, d := range block.Diagnostics() {
		if d.Line > 0 {
			attributed = true
		}
		// Warnings never cost a line its value, so they are not "the
		// reason this result is missing". Errors and cascading hints are.
		if d.Line == line && d.Severity != "warning" {
			return d.Message
		}
	}
	if !attributed && block.Error() != nil {
		return block.Error().Error()
	}
	return ""
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
					if arr, ok := stmt.Result.(*types.Array); ok {
						jr.Elements = arrayElements(arr, df)
					}
				} else if msg := statementError(block, stmt.Line); msg != "" {
					// Per-result error: the diagnostic reported against THIS
					// line, never a neighbor's (go-calcmark#113).
					jr.Error = msg
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
			// Include interpolated source when interpolation has been applied
			if interp := block.InterpolatedSource(); len(interp) > 0 {
				// Only set if different from raw source (interpolation applied)
				raw := block.Source()
				different := len(interp) != len(raw)
				if !different {
					for i := range interp {
						if interp[i] != raw[i] {
							different = true
							break
						}
					}
				}
				if different {
					jb.InterpolatedSource = interp
				}
			}
			jb.HTML = block.Render()
		}

		result.Blocks = append(result.Blocks, jb)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
