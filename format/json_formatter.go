package format

import (
	"encoding/json"
	"io"
	"maps"

	"github.com/CalcMark/go-calcmark/spec/document"
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
	Globals  map[string]string `json:"globals,omitempty"`
	Exchange map[string]string `json:"exchange,omitempty"`
}

// JSONBlock represents a single block in JSON output
type JSONBlock struct {
	Type        string           `json:"type"`
	Source      []string         `json:"source"`
	Results     []JSONResult     `json:"results,omitempty"`
	Output      string           `json:"output,omitempty"`
	Error       string           `json:"error,omitempty"`
	Diagnostics []JSONDiagnostic `json:"diagnostics,omitempty"`
	Variables   []string         `json:"variables,omitempty"`
	HTML        string           `json:"html,omitempty"`
}

// JSONResult represents a single evaluated statement's result.
type JSONResult struct {
	Source   string `json:"source"`
	Value    string `json:"value"`
	Variable string `json:"variable,omitempty"`
}

// JSONDiagnostic represents an error or warning with position info.
type JSONDiagnostic struct {
	Severity string `json:"severity"`
	Code     string `json:"code,omitempty"`
	Message  string `json:"message"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
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

		if len(fm.Globals) > 0 || len(fm.Exchange) > 0 {
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
			jb.Variables = block.Variables()

			stmts := AlignResults(block)
			for _, stmt := range stmts {
				if stmt.IsBlank || stmt.IsResultLine {
					continue
				}
				jr := JSONResult{Source: stmt.Source}
				if stmt.Result != nil {
					jr.Value = df.Format(stmt.Result)
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
			} else if block.LastValue() != nil {
				jb.Output = block.LastValue().String()
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
