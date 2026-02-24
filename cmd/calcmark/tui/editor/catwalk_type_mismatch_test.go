package editor

import (
	"fmt"
	"io"
	"strings"
	"testing"

	impldoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/cockroachdb/datadriven"
	"github.com/knz/catwalk"
	"github.com/muesli/termenv"
)

func init() {
	lipgloss.SetColorProfile(termenv.Ascii)
}

// TestEditorCatwalkTypeMismatch tests that type mismatch errors appear on the correct line.
// Bug: Error appears on the line where the variable is defined, not where it's used incorrectly.
func TestEditorCatwalkTypeMismatch(t *testing.T) {
	// Document with type mismatch: cannot multiply number by data rate
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

		catwalk.RunModel(t, path, m,
			catwalk.WithObserver("debug", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).Debug()))
				return err
			}),
			catwalk.WithObserver("results", func(out io.Writer, m tea.Model) error {
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

// TestEditorCatwalkValidValues tests that valid variables show their values even when a later statement has an error.
func TestEditorCatwalkValidValues(t *testing.T) {
	// Document with valid variables a and b, but c has type mismatch error
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

		catwalk.RunModel(t, path, m,
			catwalk.WithObserver("debug", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).Debug()))
				return err
			}),
			catwalk.WithObserver("results", func(out io.Writer, m tea.Model) error {
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
