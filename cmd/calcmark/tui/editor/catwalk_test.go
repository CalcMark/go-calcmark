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
		skipTests := []string{
			"edit_variable_no_redef",         // TestEditorCatwalkEditVariable
			"edit_b_value_bug",               // TestEditorCatwalkEditVariable
			"error_shows_valid_values",       // TestEditorCatwalkValidValues
			"error_wrong_line_type_mismatch", // TestEditorCatwalkTypeMismatch
			"wrapping_alignment",             // TestEditorCatwalkWrapping
			"wrapping_calc_lines",            // TestEditorCatwalkWrapping
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
