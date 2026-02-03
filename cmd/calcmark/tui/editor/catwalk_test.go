package editor

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cockroachdb/datadriven"
	"github.com/knz/catwalk"
	"github.com/muesli/termenv"
)

func init() {
	// Force ASCII color profile for consistent test output across environments
	lipgloss.SetColorProfile(termenv.Ascii)
}

// TestEditorCatwalk runs data-driven tests for the editor model.
// Test files are in testdata/ directory.
// Run with -rewrite flag to regenerate expected output:
//
//	go test ./cmd/calcmark/tui/editor/... -args -rewrite
func TestEditorCatwalk(t *testing.T) {
	// Create a simple test document
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		// Skip compression subdirectory (handled by separate test)
		if strings.HasPrefix(path, "testdata/compression/") {
			return
		}

		// Skip tests that have dedicated test functions with custom documents
		// Tests that require a fresh document (not polluted by previous tests in the walk)
		// are added here and have their own test function below.
		skipTests := []string{
			"edit_variable_no_redef",         // TestEditorCatwalkEditVariable
			"edit_b_value_bug",               // TestEditorCatwalkEditVariable
			"error_shows_valid_values",       // TestEditorCatwalkValidValues
			"error_wrong_line_type_mismatch", // TestEditorCatwalkTypeMismatch
			"wrapping_alignment",             // TestEditorCatwalkWrapping
			"wrapping_calc_lines",            // TestEditorCatwalkWrapping
			"layout_alignment_at_80",         // TestEditorCatwalkLayoutAlignment
			"viewport_scrolling",             // TestEditorCatwalkViewportScrolling
			"cursor_navigation",              // TestEditorCatwalkCursorNavigation
			"word_movement",                  // TestEditorCatwalkWordMovement
			"evaluation_debounce",            // TestEditorCatwalkEvaluationDebounce
			"dependent_results",              // TestEditorCatwalkDependentResults
		}
		for _, skip := range skipTests {
			if strings.HasSuffix(path, skip) {
				return
			}
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
			catwalk.WithObserver("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
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
				maxLines := len(aligned.sourceLines)
				if len(aligned.previewLines) > maxLines {
					maxLines = len(aligned.previewLines)
				}

				for i := 0; i < maxLines; i++ {
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
				// Custom observer to check line results for errors
				model := m.(Model)
				results := model.GetLineResults()
				var buf strings.Builder
				for _, r := range results {
					if r.IsCalc || r.Error != "" {
						buf.WriteString(fmt.Sprintf("Line %d (%s): value=%s, error=%q\n", r.LineNum, r.Source, r.Value, r.Error))
					}
				}
				_, err := out.Write([]byte(buf.String()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkEditVariable tests editing variable values (regression test for false redefinition errors).
// This reproduces the user's bug: editing "b = 5" to "b = 6" showed error on "a = 3".
func TestEditorCatwalkEditVariable(t *testing.T) {
	// User's exact scenario: two variables separated by empty lines, then markdown
	content := `a = 3

b = 5

# Hello`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		// Only run this test on the specific test file
		if !strings.HasSuffix(path, "edit_variable_no_redef") {
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
			catwalk.WithObserver("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
			catwalk.WithObserver("results", func(out io.Writer, m tea.Model) error {
				// Custom observer to check line results for errors
				model := m.(Model)
				results := model.GetLineResults()
				var buf strings.Builder
				for _, r := range results {
					if r.IsCalc || r.Error != "" {
						buf.WriteString(fmt.Sprintf("Line %d (%s): value=%s, error=%q\n", r.LineNum, r.Source, r.Value, r.Error))
					}
				}
				_, err := out.Write([]byte(buf.String()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkCompression runs tests with compression.cm-like content
// that causes wrapping at narrow widths.
func TestEditorCatwalkCompression(t *testing.T) {
	content := `# Compression Function - compress()

# Compressed size estimates for different compression types
gzip_compressed = compress(1 GB, gzip)
lz4_compressed = compress(100 MB, lz4)
zstd_compressed = compress(500 MB, zstd)
bzip2_compressed = compress(1000 MB, bzip2)
snappy_compressed = compress(300 MB, snappy)
no_compression = compress(200 MB, none)

# Use in calculations
storage_savings = 10 GB - compress(10 GB, gzip)
compressed_transfer = transfer_time(compress(1 GB, lz4), global, gigabit)`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata/compression", func(t *testing.T, path string) {
		m := New(doc)
		m.width = 80 // Narrower width to test wrapping
		m.height = 24
		m.previewMode = PreviewFull

		catwalk.RunModel(t, path, m,
			catwalk.WithObserver("debug", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).Debug()))
				return err
			}),
			catwalk.WithObserver("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkLayoutAlignment tests source/preview alignment at default width.
// Uses a fresh document to avoid shared mutation from other catwalk tests
// that modify the document via key sequences (insert_line, scroll_navigation, etc.).
func TestEditorCatwalkLayoutAlignment(t *testing.T) {
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "layout_alignment_at_80") {
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
			catwalk.WithObserver("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
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

				maxLines := len(aligned.sourceLines)
				if len(aligned.previewLines) > maxLines {
					maxLines = len(aligned.previewLines)
				}

				for i := 0; i < maxLines; i++ {
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
						buf.WriteString(fmt.Sprintf("Line %d (%s): value=%s, error=%q\n", r.LineNum, r.Source, r.Value, r.Error))
					}
				}
				_, err := out.Write([]byte(buf.String()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkViewportScrolling tests viewport scrolling with scroll margin.
// This uses a 20+ line document and a viewport of 10 lines to test:
// - Cursor staying visible after navigation
// - Scroll margin keeping cursor N lines from viewport edge
// - Page Up/Down scrolling behavior
func TestEditorCatwalkViewportScrolling(t *testing.T) {
	// Create a document with 20+ lines to enable scrolling tests
	content := `# Viewport Scrolling Test
line 2
line 3
line 4
line 5
line 6
line 7
line 8
line 9
line 10
line 11
line 12
line 13
line 14
line 15
line 16
line 17
line 18
line 19
line 20
line 21
line 22`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "viewport_scrolling") {
			return
		}

		m := New(doc)
		m.width = 80
		m.height = 16 // Small viewport to test scrolling (visibleHeight = 16-6 = 10 lines)
		m.previewMode = PreviewFull

		catwalk.RunModel(t, path, m,
			catwalk.WithObserver("debug", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).Debug()))
				return err
			}),
			catwalk.WithObserver("scroll", func(out io.Writer, m tea.Model) error {
				// Custom observer focused on scroll state
				model := m.(Model)
				var buf strings.Builder
				buf.WriteString(fmt.Sprintf("cursorLine=%d scrollOffset=%d totalLines=%d visibleHeight=%d\n",
					model.cursorLine, model.scrollOffset, model.TotalLines(), model.getVisibleHeight()))
				// Check if cursor is within visible range with margin
				visibleStart := model.scrollOffset
				visibleEnd := model.scrollOffset + model.getVisibleHeight()
				inView := model.cursorLine >= visibleStart && model.cursorLine < visibleEnd
				hasTopMargin := model.cursorLine >= model.scrollOffset+scrollMargin || model.cursorLine < scrollMargin
				hasBottomMargin := model.cursorLine < model.scrollOffset+model.getVisibleHeight()-scrollMargin ||
					model.cursorLine >= model.TotalLines()-scrollMargin
				buf.WriteString(fmt.Sprintf("cursorInView=%v hasTopMargin=%v hasBottomMargin=%v\n",
					inView, hasTopMargin, hasBottomMargin))
				_, err := out.Write([]byte(buf.String()))
				return err
			}),
			catwalk.WithObserver("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkCursorNavigation tests cursor navigation behaviors.
// Uses a fresh document to avoid pollution from other tests.
func TestEditorCatwalkCursorNavigation(t *testing.T) {
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "cursor_navigation") {
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
			catwalk.WithObserver("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkWordMovement tests Ctrl+Arrow word movement behaviors.
// Uses a fresh document to avoid pollution from other tests.
func TestEditorCatwalkWordMovement(t *testing.T) {
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "word_movement") {
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
			catwalk.WithObserver("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkEvaluationDebounce tests the evaluation pipeline and debounce behavior.
// Verifies that:
// - Calculations are evaluated and show results
// - Non-calculation lines show blank in results
// - Typing new calculations triggers evaluation
func TestEditorCatwalkEvaluationDebounce(t *testing.T) {
	// Document with calculations and dependencies
	content := `rate = 10%
principal = 1000
interest = principal * rate

`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "evaluation_debounce") {
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
						buf.WriteString(fmt.Sprintf("Line %d (%s): value=%s, error=%q\n", r.LineNum, r.Source, r.Value, r.Error))
					}
				}
				_, err := out.Write([]byte(buf.String()))
				return err
			}),
			catwalk.WithObserver("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkDependentResults tests that dependent variables update when source changes.
// Verifies that:
// - Changing a variable updates all dependent lines
// - Multiple dependents update correctly
// - Error states are handled (undefined variable shows error, fix shows result)
func TestEditorCatwalkDependentResults(t *testing.T) {
	// Document with tax/price/total calculation chain
	content := `tax = 10%
price = 100
total = price * (1 + tax)

`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "dependent_results") {
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
						buf.WriteString(fmt.Sprintf("Line %d (%s): value=%s, error=%q\n", r.LineNum, r.Source, r.Value, r.Error))
					}
				}
				_, err := out.Write([]byte(buf.String()))
				return err
			}),
			catwalk.WithObserver("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}
