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
	Type      string   `json:"type"`
	Source    []string `json:"source"`
	Output    string   `json:"output,omitempty"`
	Error     string   `json:"error,omitempty"`
	Variables []string `json:"variables,omitempty"`
}

// Format writes the document as JSON to the writer.
func (f *JSONFormatter) Format(w io.Writer, doc *document.Document, opts Options) error {
	result := JSONDocument{
		Blocks: make([]JSONBlock, 0),
	}

	// Add frontmatter if present
	if fm := doc.GetFrontmatter(); fm != nil {
		jfm := &JSONFrontmatter{}

		if len(fm.Globals) > 0 {
			jfm.Globals = make(map[string]string)
			maps.Copy(jfm.Globals, fm.Globals)
		}

		if len(fm.Exchange) > 0 {
			jfm.Exchange = make(map[string]string)
			for key, rate := range fm.Exchange {
				jfm.Exchange[key] = rate.String()
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

			if block.Error() != nil {
				jb.Error = block.Error().Error()
			} else if block.LastValue() != nil {
				jb.Output = block.LastValue().String()
			}

		case *document.TextBlock:
			jb.Type = "text"
		}

		result.Blocks = append(result.Blocks, jb)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
