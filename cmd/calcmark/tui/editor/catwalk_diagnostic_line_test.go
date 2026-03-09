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

// diagnosticLineTest defines a catwalk test with pre-evaluated document content.
type diagnosticLineTest struct {
	name    string
	content string
	file    string // testdata filename suffix
}

var diagnosticLineTests = []diagnosticLineTest{
	{
		name: "type mismatch",
		content: `b = 12 apples
10 / b
`,
		file: "diagnostic_wrong_line",
	},
	{
		name: "compound NL eval error",
		content: `compound $1000 by 5% over 10 years
compound $1000 by 5% monthly over 10 ye
`,
		file: "diagnostic_wrong_line_compound",
	},
}

// TestEditorCatwalkDiagnosticLine verifies that error diagnostics appear on the
// correct source line. Errors must render next to their own line, not a neighbor.
func TestEditorCatwalkDiagnosticLine(t *testing.T) {
	for _, tt := range diagnosticLineTests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := document.NewDocument(tt.content)
			if err != nil {
				t.Fatalf("Failed to create document: %v", err)
			}

			eval := impldoc.NewEvaluator()
			evalErr := eval.Evaluate(doc)
			t.Logf("Evaluation error (expected): %v", evalErr)

			file := tt.file
			datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
				if !strings.HasSuffix(path, file) {
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
		})
	}
}
