package format

import (
	"encoding/json"
	"io"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// JSONFormatter formats CalcMark documents as JSON.
// Useful for programmatic consumption and integration with other tools.
type JSONFormatter struct{}

// Extensions returns the file extensions handled by this formatter.
func (f *JSONFormatter) Extensions() []string {
	return []string{".json"}
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
	blocks := doc.GetBlocks()
	result := make([]JSONBlock, 0, len(blocks))

	for _, node := range blocks {
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
				// Use Type.String() for consistent output
				jb.Output = block.LastValue().String()
			}

		case *document.TextBlock:
			jb.Type = "text"
		}

		result = append(result, jb)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
