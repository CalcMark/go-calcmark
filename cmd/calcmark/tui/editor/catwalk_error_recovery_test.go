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

// TestEditorCatwalkErrorRecoveryCascading verifies TUI display of error recovery:
// root-cause errors show as non-blocked, cascading errors show as blocked (IsBlocked=true).
func TestEditorCatwalkErrorRecoveryCascading(t *testing.T) {
	content := `a = 1 / 0
b = 10
c = a * 2
`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := impldoc.NewEvaluator()
	_ = eval.Evaluate(doc) // ErrPartialEvaluation expected

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "error_recovery_cascading") {
			return
		}

		m := New(doc)
		m.width = 80
		m.height = 24
		m.previewMode = PreviewFull

		RunModelV2(t, path, m,
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
						buf.WriteString(fmt.Sprintf("Line %d (%s): value=%s, error=%s, blocked=%v\n",
							r.LineNum, r.Source, r.Value, errorMsg, r.IsBlocked))
					}
				}
				_, err := out.Write([]byte(buf.String()))
				return err
			}),
		)
	})
}
