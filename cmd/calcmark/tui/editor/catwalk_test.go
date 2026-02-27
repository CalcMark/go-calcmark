package editor

import (
	"fmt"
	"io"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/cockroachdb/datadriven"
)

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
		// Skip preview_pane subdirectory (handled by TestEditorCatwalkPreviewPane)
		if strings.HasPrefix(path, "testdata/preview_pane/") {
			return
		}

		// Skip tests that have dedicated test functions with custom documents
		// Tests that require a fresh document (not polluted by previous tests in the walk)
		// are added here and have their own test function below.
		skipTests := []string{
			"edit_variable_no_redef",              // TestEditorCatwalkEditVariable
			"edit_b_value_bug",                    // TestEditorCatwalkEditVariable
			"error_shows_valid_values",            // TestEditorCatwalkValidValues
			"error_wrong_line_type_mismatch",      // TestEditorCatwalkTypeMismatch
			"wrapping_alignment",                  // TestEditorCatwalkWrapping
			"wrapping_calc_lines",                 // TestEditorCatwalkWrapping
			"layout_alignment_at_80",              // TestEditorCatwalkLayoutAlignment
			"viewport_scrolling",                  // TestEditorCatwalkViewportScrolling
			"cursor_navigation",                   // TestEditorCatwalkCursorNavigation
			"word_movement",                       // TestEditorCatwalkWordMovement
			"evaluation_debounce",                 // TestEditorCatwalkEvaluationDebounce
			"dependent_results",                   // TestEditorCatwalkDependentResults
			"insert_at_end",                       // TestEditorCatwalkInsertAtEnd
			"insert_line",                         // TestEditorCatwalkInsertLine
			"scroll_navigation",                   // TestEditorCatwalkScrollNavigation
			"delete_empty_line",                   // TestEditorCatwalkDeleteEmptyLine
			"typing_text",                         // TestEditorCatwalkTypingText
			"text_wrapping_40col",                 // TestEditorCatwalkTextWrapping40Col
			"long_document_scroll",                // TestEditorCatwalkLongDocumentScroll
			"help_toggle",                         // TestEditorCatwalkHelpToggle
			"document_navigation",                 // TestEditorCatwalkDocumentNavigation
			"word_nav_comprehensive",              // TestEditorCatwalkWordNavComprehensive
			"delete_last_char",                    // TestEditorCatwalkDeleteLastChar
			"undo",                                // TestEditorCatwalkUndo
			"clipboard",                           // TestEditorCatwalkClipboard
			"export_flow",                         // TestEditorCatwalkExportFlow
			"help_interactive",                    // TestEditorCatwalkHelpInteractive
			"frontmatter_insert",                  // TestEditorCatwalkFrontmatterInsert
			"frontmatter_editing",                 // TestEditorCatwalkFrontmatterEditing
			"open_unsaved_prompt",                 // TestEditorCatwalkOpenUnsavedPrompt
			"frontmatter_globals_alignment",       // TestEditorCatwalkFrontmatterGlobalsAlignment
			"frontmatter_exchange_alignment",      // TestEditorCatwalkFrontmatterExchangeAlignment
			"frontmatter_both_sections_alignment", // TestEditorCatwalkFrontmatterBothSectionsAlignment
			"frontmatter_empty_alignment",         // TestEditorCatwalkFrontmatterEmptyAlignment
			"frontmatter_insert_undo",             // TestEditorCatwalkFrontmatterInsertUndo
			"selection",                           // TestEditorCatwalkSelection
			"shift_selection",                     // TestEditorCatwalkShiftSelection
			"cmd_shortcuts",                       // TestEditorCatwalkCmdShortcuts
			"share_to_overlay",                    // TestEditorCatwalkShareToOverlay
			"open_from_overlay",                   // TestEditorCatwalkOpenFromOverlay
			"open_from_unsaved",                   // TestEditorCatwalkOpenFromUnsaved
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

		RunModelV2(t, path, m,
			WithObserverV2("debug", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).Debug()))
				return err
			}),
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
			WithObserverV2("alignment", func(out io.Writer, m tea.Model) error {
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
			WithObserverV2("results", func(out io.Writer, m tea.Model) error {
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

		RunModelV2(t, path, m,
			WithObserverV2("debug", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).Debug()))
				return err
			}),
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
			WithObserverV2("results", func(out io.Writer, m tea.Model) error {
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

// compressionDocumentContent is the shared document content for compression tests.
// Each test function creates its own fresh document from this content.
const compressionDocumentContent = `# Compression Function - compress()

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

// TestEditorCatwalkCompressionInsertLine tests insert line behavior with compression document.
// Uses a fresh document to avoid shared mutation pollution from other tests.
func TestEditorCatwalkCompressionInsertLine(t *testing.T) {
	doc, err := document.NewDocument(compressionDocumentContent)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata/compression", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "insert_line") {
			return
		}

		m := New(doc)
		m.width = 80 // Narrower width to test wrapping
		m.height = 24
		m.previewMode = PreviewFull

		RunModelV2(t, path, m,
			WithObserverV2("debug", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).Debug()))
				return err
			}),
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkCompressionTypeNewLine tests typing on newly inserted lines with compression document.
// Uses a fresh document to avoid shared mutation pollution from other tests.
func TestEditorCatwalkCompressionTypeNewLine(t *testing.T) {
	doc, err := document.NewDocument(compressionDocumentContent)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata/compression", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "type_new_line") {
			return
		}

		m := New(doc)
		m.width = 80 // Narrower width to test wrapping
		m.height = 24
		m.previewMode = PreviewFull

		RunModelV2(t, path, m,
			WithObserverV2("debug", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).Debug()))
				return err
			}),
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
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

		RunModelV2(t, path, m,
			WithObserverV2("debug", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).Debug()))
				return err
			}),
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
			WithObserverV2("alignment", func(out io.Writer, m tea.Model) error {
				model := m.(Model)
				leftWidth, rightWidth := model.GetPaneWidths(model.width)
				aligned := model.computeAlignedPanes(leftWidth, rightWidth)

				var buf strings.Builder
				buf.WriteString("Source and Preview Alignment:\n")
				buf.WriteString(fmt.Sprintf("Total visual lines: %d\n", len(aligned.sourceLines)))
				buf.WriteString(fmt.Sprintf("Source lines count: %d, Preview lines count: %d\n",
					len(aligned.sourceLines), len(aligned.previewLines)))

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
			WithObserverV2("results", func(out io.Writer, m tea.Model) error {
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

		RunModelV2(t, path, m,
			WithObserverV2("debug", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).Debug()))
				return err
			}),
			WithObserverV2("scroll", func(out io.Writer, m tea.Model) error {
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
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
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

		RunModelV2(t, path, m,
			WithObserverV2("debug", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).Debug()))
				return err
			}),
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkDocumentNavigation tests Ctrl+Home and Ctrl+End navigation.
// Verifies: NAV-05 (Ctrl+Home -> doc start) and NAV-06 (Ctrl+End -> doc end).
// Uses a fresh document to avoid pollution from other tests.
func TestEditorCatwalkDocumentNavigation(t *testing.T) {
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "document_navigation") {
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
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
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

		RunModelV2(t, path, m,
			WithObserverV2("debug", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).Debug()))
				return err
			}),
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkWordNavComprehensive tests Alt+B/F word navigation (macOS-friendly).
// On macOS terminals, Option+Arrow sends ESC+b/ESC+f escape sequences (Alt+b/Alt+f).
// These are standard readline/emacs bindings for backward-word and forward-word.
// Verifies: NAV-01 (Alt+B word left) and NAV-02 (Alt+F word right).
// Uses a fresh document to avoid pollution from other tests.
func TestEditorCatwalkWordNavComprehensive(t *testing.T) {
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "word_nav_comprehensive") {
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
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
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
						buf.WriteString(fmt.Sprintf("Line %d (%s): value=%s, error=%q\n", r.LineNum, r.Source, r.Value, r.Error))
					}
				}
				_, err := out.Write([]byte(buf.String()))
				return err
			}),
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
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
						buf.WriteString(fmt.Sprintf("Line %d (%s): value=%s, error=%q\n", r.LineNum, r.Source, r.Value, r.Error))
					}
				}
				_, err := out.Write([]byte(buf.String()))
				return err
			}),
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkInsertAtEnd tests inserting text at the end of a document.
// Uses a fresh document to avoid shared mutation pollution from other catwalk tests.
func TestEditorCatwalkInsertAtEnd(t *testing.T) {
	// Simple document with multiple lines to navigate through
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "insert_at_end") {
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
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkInsertLine tests inserting new lines via 'o' key.
// Uses a fresh document to avoid shared mutation pollution from other catwalk tests.
func TestEditorCatwalkInsertLine(t *testing.T) {
	// Simple document with multiple lines
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		// Skip compression subdirectory (handled by dedicated compression test)
		if strings.Contains(path, "compression/") {
			return
		}
		if !strings.HasSuffix(path, "insert_line") {
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
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkScrollNavigation tests scroll behavior after inserting lines.
// Uses a fresh document to avoid shared mutation pollution from other catwalk tests.
func TestEditorCatwalkScrollNavigation(t *testing.T) {
	// Simple document with multiple lines
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "scroll_navigation") {
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
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkDeleteEmptyLine tests DELETE key behavior on empty lines.
// Uses a fresh document to avoid shared mutation pollution from other catwalk tests.
func TestEditorCatwalkDeleteEmptyLine(t *testing.T) {
	// Simple document with multiple lines
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "delete_empty_line") {
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
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
			WithObserverV2("view", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).View().Content))
				return err
			}),
		)
	})
}

// TestEditorCatwalkTypingText tests basic text typing behaviors.
// Verifies: Characters appear correctly, backspace deletes, delete key works, cursor advances.
// Uses a fresh document to avoid shared mutation pollution from other catwalk tests.
func TestEditorCatwalkTypingText(t *testing.T) {
	content := `# Test Document
x = 10
y = 20`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "typing_text") {
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
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkTextWrapping40Col tests text wrapping at narrow width (40 columns).
// Verifies: Long lines wrap correctly, visual line count increases, alignment is correct.
// Uses a fresh document to avoid shared mutation pollution from other catwalk tests.
func TestEditorCatwalkTextWrapping40Col(t *testing.T) {
	// Document with lines that will wrap at 40 columns
	content := `# Wrapping Test at 40 Columns
This is a line that is definitely longer than forty columns.
x = 12345 + 67890 * 2
Short line
Another very long line that will certainly wrap when displayed at narrow width.`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "text_wrapping_40col") {
			return
		}

		m := New(doc)
		m.width = 40 // Narrow width to force wrapping
		m.height = 24
		m.previewMode = PreviewFull

		RunModelV2(t, path, m,
			WithObserverV2("debug", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).Debug()))
				return err
			}),
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
			WithObserverV2("alignment", func(out io.Writer, m tea.Model) error {
				model := m.(Model)
				leftWidth, rightWidth := model.GetPaneWidths(model.width)
				aligned := model.computeAlignedPanes(leftWidth, rightWidth)

				var buf strings.Builder
				buf.WriteString("Source and Preview Alignment:\n")
				buf.WriteString(fmt.Sprintf("Total visual lines: %d\n", len(aligned.sourceLines)))

				for i := 0; i < len(aligned.sourceLines) && i < 15; i++ {
					src := aligned.sourceLines[i]
					buf.WriteString(fmt.Sprintf("[%d] ln=%d wrap=%v: %q\n",
						i, src.lineNum, src.isWrapped, src.content))
				}

				_, err := out.Write([]byte(buf.String()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkLongDocumentScroll tests scrolling through a long document (50+ lines).
// Verifies: Cursor stays visible, scroll margin maintained, Page Up/Down work.
// Uses a fresh document to avoid shared mutation pollution from other catwalk tests.
func TestEditorCatwalkLongDocumentScroll(t *testing.T) {
	// Create a document with 50+ lines
	var lines []string
	lines = append(lines, "# Long Document Scroll Test")
	for i := 2; i <= 55; i++ {
		if i%5 == 0 {
			lines = append(lines, fmt.Sprintf("calc_%d = %d * 2", i, i))
		} else {
			lines = append(lines, fmt.Sprintf("line %d content here", i))
		}
	}
	content := strings.Join(lines, "\n")

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "long_document_scroll") {
			return
		}

		m := New(doc)
		m.width = 80
		m.height = 16 // Small viewport (10 visible lines) to test scrolling
		m.previewMode = PreviewFull

		RunModelV2(t, path, m,
			WithObserverV2("debug", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).Debug()))
				return err
			}),
			WithObserverV2("scroll", func(out io.Writer, m tea.Model) error {
				model := m.(Model)
				var buf strings.Builder
				buf.WriteString(fmt.Sprintf("cursorLine=%d scrollOffset=%d totalLines=%d visibleHeight=%d\n",
					model.cursorLine, model.scrollOffset, model.TotalLines(), model.getVisibleHeight()))
				visibleStart := model.scrollOffset
				visibleEnd := model.scrollOffset + model.getVisibleHeight()
				inView := model.cursorLine >= visibleStart && model.cursorLine < visibleEnd
				buf.WriteString(fmt.Sprintf("cursorInView=%v visibleRange=[%d,%d)\n",
					inView, visibleStart, visibleEnd))
				_, err := out.Write([]byte(buf.String()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkHelpToggle tests F1 help overlay toggle behavior.
// Verifies: F1 opens help, F1 closes help, Esc closes help, editing continues after.
// Uses a fresh document to avoid shared mutation pollution from other catwalk tests.
func TestEditorCatwalkHelpToggle(t *testing.T) {
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "help_toggle") {
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
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkPreviewPane tests Phase 10 preview pane requirements.
// Each test file in testdata/preview_pane uses a specific document to test
// different preview pane behaviors.
func TestEditorCatwalkPreviewPane(t *testing.T) {
	// Define test-specific documents
	testDocs := map[string]string{
		"pane_ratio": `# Header
x = 10
y = 20
z = 30`,
		"vertical_alignment": `# Header

x = 10


y = x + 5


z = y * 2`,
		"anonymous_calc_format": `# Math Examples
2 + 2
total = 100
100 * 2`,
		"results_header": `# Header
x = 10
y = 20
z = 30`,
		"non_calc_lines_blank": `# Budget Calculator

This is some explanatory text.

income = 5000
expenses = 3000

## Summary

savings = income - expenses`,
		"scroll_sync": `# Long Document
a = 1
b = 2
c = 3
d = 4
e = 5
f = 6
g = 7
h = 8
i = 9
j = 10
k = 11
l = 12
m = 13
n = 14`,
		"napkin_tilde": `# Data Transfer
rate = 5 MB/s
time = 1 day
total = accumulate(rate, time) as napkin`,
		"cascading_errors": `a = undefined_var
b = a + 1
c = b * 2`,
		"currency_formatting": `# Budget
price = 100 USD
tax = price * 10%
total = price + tax
large = 1500 USD`,
		"cursor_wrap_alignment": `# Household Budget

total_gross = salary_1 + salary_2

net = total_gross * 0.7`,
	}

	datadriven.Walk(t, "testdata/preview_pane", func(t *testing.T, path string) {
		// Extract test name from path
		testName := strings.TrimPrefix(path, "testdata/preview_pane/")

		docContent, ok := testDocs[testName]
		if !ok {
			t.Fatalf("No test document defined for %s", testName)
		}

		doc, err := document.NewDocument(docContent)
		if err != nil {
			t.Fatalf("Failed to create document for %s: %v", testName, err)
		}

		m := New(doc)
		m.width = 80
		m.height = 24
		m.previewMode = PreviewFull

		// Evaluate to get results
		m.eval.Evaluate(m.doc)

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
					buf.WriteString(fmt.Sprintf("Line %d (%s): value=%s, error=%q\n",
						r.LineNum, r.Source, r.Value, r.Error))
				}
				_, err := out.Write([]byte(buf.String()))
				return err
			}),
			WithObserverV2("alignment", func(out io.Writer, m tea.Model) error {
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
					var srcInfo, prvInfo string
					if i < len(aligned.sourceLines) {
						sl := aligned.sourceLines[i]
						srcContent := sl.content
						if len(srcContent) > 40 {
							srcContent = srcContent[:40]
						}
						srcInfo = fmt.Sprintf("SRC(ln=%d wrap=%v): %q", sl.lineNum, sl.isWrapped, srcContent)
					}
					if i < len(aligned.previewLines) {
						pl := aligned.previewLines[i]
						prvContent := pl.content
						if len(prvContent) > 40 {
							prvContent = prvContent[:40] + "..."
						}
						prvInfo = fmt.Sprintf("PRV(ln=%d): %q", pl.sourceLineNum, prvContent)
					}
					buf.WriteString(fmt.Sprintf("[%d] %-50s | %s\n", i, srcInfo, prvInfo))
				}
				_, err := out.Write([]byte(buf.String()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkDeleteLastChar tests DELETE key removing the last remaining character.
// This is a regression test for the bug: "DELETE sometimes fails to remove single character"
// Verifies: DELETE at position 0 on single-character line removes the character.
// Uses a fresh document with single "a" character.
func TestEditorCatwalkDeleteLastChar(t *testing.T) {
	// Document with single character "a"
	doc, err := document.NewDocument("a")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "delete_last_char") {
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
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkUndo tests undo/redo behavior.
// Verifies: Ctrl+Z undoes, Ctrl+Y redoes, cursor restoration, timer grouping.
// Uses a fresh document to avoid shared mutation pollution from other catwalk tests.
func TestEditorCatwalkUndo(t *testing.T) {
	// Simple document for undo testing
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "/undo") {
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
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkClipboard runs clipboard-specific catwalk tests.
// Uses a fresh document to avoid shared mutation pollution from other catwalk tests.
func TestEditorCatwalkClipboard(t *testing.T) {
	// Simple document for clipboard testing
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "clipboard") {
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
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkExportFlow tests the complete export flow through catwalk.
func TestEditorCatwalkExportFlow(t *testing.T) {
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "export_flow") {
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
		)
	})
}

// TestEditorCatwalkHelpInteractive tests the interactive help overlay through catwalk.
// Verifies: opening help, navigating, executing commands, dismissing.
func TestEditorCatwalkHelpInteractive(t *testing.T) {
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "help_interactive") {
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
		)
	})
}

// TestEditorCatwalkOpenUnsavedPrompt tests Ctrl+O save prompt when there are unsaved changes.
// Reproduces the bug where Ctrl+O skipped the unsaved changes check and didn't reset editBuf.
func TestEditorCatwalkOpenUnsavedPrompt(t *testing.T) {
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "open_unsaved_prompt") {
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
		)
	})
}

// TestEditorCatwalkSelection tests Ctrl+A select-all and navigation clearing selection.
// Uses a fresh document to avoid state accumulation from other tests in the walk.
func TestEditorCatwalkSelection(t *testing.T) {
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "/selection") {
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
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkShiftSelection tests Shift+Arrow text selection.
// Uses a fresh document because the test modifies content (typing replaces selection).
func TestEditorCatwalkShiftSelection(t *testing.T) {
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "/shift_selection") {
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
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkCmdShortcuts tests macOS Cmd (Super) keyboard shortcuts
// for clipboard and undo/redo operations.
func TestEditorCatwalkCmdShortcuts(t *testing.T) {
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "/cmd_shortcuts") {
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
			WithObserverV2("lines", func(out io.Writer, m tea.Model) error {
				_, err := out.Write([]byte(m.(Model).DebugLines()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkShareToOverlay tests the Share To Gist overlay through catwalk.
// Verifies: opening share overlay via command menu, Esc cancels, field navigation.
func TestEditorCatwalkShareToOverlay(t *testing.T) {
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "share_to_overlay") {
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
		)
	})
}

// TestEditorCatwalkOpenFromOverlay tests the Open From Gist overlay through catwalk.
// Verifies: opening open-from overlay via command menu, Esc cancels, text input.
func TestEditorCatwalkOpenFromOverlay(t *testing.T) {
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "open_from_overlay") {
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
		)
	})
}

// TestEditorCatwalkOpenFromUnsaved tests Open From Gist with unsaved changes.
// Verifies: typing creates unsaved state, Open From triggers save prompt,
// cancel returns to editing, discard proceeds to Open From overlay.
func TestEditorCatwalkOpenFromUnsaved(t *testing.T) {
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "open_from_unsaved") {
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
		)
	})
}
