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

// TestEditorCatwalkWrapping tests text wrapping alignment between source and preview panes.
func TestEditorCatwalkWrapping(t *testing.T) {
	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		// Only run wrapping tests
		if !strings.HasSuffix(path, "wrapping_alignment") &&
			!strings.HasSuffix(path, "wrapping_calc_lines") {
			return
		}

		// Select document based on test
		var content string
		if strings.HasSuffix(path, "wrapping_alignment") {
			// Document matching the screenshot: calculations and long heading
			content = `a = 2
b = 10 MB


# Testing a reaaeeeeeeeeeeeeeeeeeeeeelly long
c = a * b
`
		} else if strings.HasSuffix(path, "wrapping_calc_lines") {
			// Document with very long variable name in calculation
			content = `very_long_variable_name_that_will_definitely_wrap_in_narrow_pane = 42
result = very_long_variable_name_that_will_definitely_wrap_in_narrow_pane * 2
`
		}

		doc, err := document.NewDocument(content)
		if err != nil {
			t.Fatalf("Failed to create document: %v", err)
		}

		// Evaluate
		eval := impldoc.NewEvaluator()
		evalErr := eval.Evaluate(doc)
		t.Logf("Evaluation error for %s: %v", path, evalErr)

		m := New(doc)
		m.width = 80
		m.height = 24
		m.previewMode = PreviewFull

		catwalk.RunModel(t, path, m,
			catwalk.WithObserver("debug", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).Debug()))
				return err
			}),
			catwalk.WithObserver("lines", func(out io.Writer, m tea.Model) error {
				model := m.(Model)
				// Use DebugLines() to show visual line structure
				_, err := out.Write([]byte(model.DebugLines()))
				return err
			}),
			catwalk.WithObserver("alignment", func(out io.Writer, m tea.Model) error {
				model := m.(Model)
				leftWidth, rightWidth := model.GetPaneWidths(model.width)
				aligned := model.computeAlignedPanes(leftWidth, rightWidth)

				var buf strings.Builder
				buf.WriteString("Source and Preview Alignment:\n")
				buf.WriteString(fmt.Sprintf("Total visual lines: %d\n", len(aligned.sourceLines)))
				buf.WriteString(fmt.Sprintf("Source lines count: %d, Preview lines count: %d\n",
					len(aligned.sourceLines), len(aligned.previewLines)))

				// Show side-by-side alignment
				maxLines := max(len(aligned.sourceLines), len(aligned.previewLines))

				for i := range maxLines {
					var srcContent, prvContent string
					var srcLineNum, prvLineNum int
					var srcWrapped bool

					if i < len(aligned.sourceLines) {
						src := aligned.sourceLines[i]
						srcContent = src.content
						srcLineNum = src.lineNum
						srcWrapped = src.isWrapped
					}

					if i < len(aligned.previewLines) {
						prv := aligned.previewLines[i]
						prvContent = prv.content
						prvLineNum = prv.sourceLineNum
					}

					// Truncate content for readability
					if len(srcContent) > 35 {
						srcContent = srcContent[:35] + "..."
					}
					if len(prvContent) > 35 {
						prvContent = prvContent[:35] + "..."
					}

					buf.WriteString(fmt.Sprintf("[%d] SRC(ln=%d wrap=%v): %-40s | PRV(ln=%d): %q\n",
						i, srcLineNum, srcWrapped, fmt.Sprintf("%q", srcContent),
						prvLineNum, prvContent))
				}

				_, err := out.Write([]byte(buf.String()))
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
