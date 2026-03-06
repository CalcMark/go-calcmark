package editor

import (
	"fmt"
	"io"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	impldoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/cockroachdb/datadriven"
)

// TestEditorCatwalkTypeMismatch tests Number * Rate produces a scaled Rate.
func TestEditorCatwalkTypeMismatch(t *testing.T) {
	// Document with number * rate: 5 * 10 MB/s = 50 MB/s
	content := `b = 5
a = 10 MB/s
c = b * a
`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Evaluate to trigger type checking
	eval := impldoc.NewEvaluator()
	evalErr := eval.Evaluate(doc)
	t.Logf("Evaluation error (expected for type mismatch): %v", evalErr)

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "error_wrong_line_type_mismatch") {
			return
		}

		m := New(doc)
		m.width = 80
		m.height = 24
		m.previewMode = PreviewFull

		RunModelV2(t, path, m,
			WithObserverV2("debug", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).Debug()))
				return err
			}),
			WithObserverV2("results", func(out io.Writer, m tea.Model) error {
				model := m.(Model)
				results := model.GetLineResults()
				var buf strings.Builder
				for _, r := range results {
					if r.IsCalc || r.Error != "" {
						errorMsg := r.Error
						if errorMsg == "" {
							errorMsg = `""`
						} else {
							// Extract just the error type and message, not the full diagnostic
							errorMsg = fmt.Sprintf("%q", errorMsg)
						}
						buf.WriteString(fmt.Sprintf("Line %d (%s): value=%s, error=%s\n",
							r.LineNum, r.Source, r.Value, errorMsg))
					}
				}
				_, err := out.Write([]byte(buf.String()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkValidValues tests that Number * Rate shows correct values for all lines.
func TestEditorCatwalkValidValues(t *testing.T) {
	// Document with number * rate: 3 * 10 MB/s = 30 MB/s
	content := `a = 3
b = 10 MB/s
c = a * b
`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Evaluate to trigger type checking
	eval := impldoc.NewEvaluator()
	evalErr := eval.Evaluate(doc)
	t.Logf("Evaluation error (expected for type mismatch): %v", evalErr)

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "error_shows_valid_values") {
			return
		}

		m := New(doc)
		m.width = 80
		m.height = 24
		m.previewMode = PreviewFull

		RunModelV2(t, path, m,
			WithObserverV2("debug", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).Debug()))
				return err
			}),
			WithObserverV2("results", func(out io.Writer, m tea.Model) error {
				model := m.(Model)
				results := model.GetLineResults()
				var buf strings.Builder
				for _, r := range results {
					if r.IsCalc || r.Error != "" {
						errorMsg := r.Error
						if errorMsg == "" {
							errorMsg = `""`
						} else {
							// Extract just the error type and message, not the full diagnostic
							errorMsg = fmt.Sprintf("%q", errorMsg)
						}
						buf.WriteString(fmt.Sprintf("Line %d (%s): value=%s, error=%s\n",
							r.LineNum, r.Source, r.Value, errorMsg))
					}
				}
				_, err := out.Write([]byte(buf.String()))
				return err
			}),
		)
	})
}
