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

// TestEditorCatwalkDiagnosticLine verifies that error diagnostics appear on the
// correct source line (issue #36). A type mismatch error on line 2 must not be
// rendered next to line 1.
func TestEditorCatwalkDiagnosticLine(t *testing.T) {
	content := `b = 12 apples
10 / b
`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := impldoc.NewEvaluator()
	evalErr := eval.Evaluate(doc)
	t.Logf("Evaluation error (expected): %v", evalErr)

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "diagnostic_wrong_line") {
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
