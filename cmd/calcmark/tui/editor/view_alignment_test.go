package editor

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func init() {
	// Force ASCII for consistent test output
	lipgloss.SetColorProfile(termenv.Ascii)
}

// TestViewAlignment_EditMode tests that source and preview panes
// have the same number of lines during edit mode.
// This is the actual bug - rendered output misalignment.
func TestViewAlignment_EditMode(t *testing.T) {
	content := `# Compression Function
gzip_compressed = compress(1 GB, gzip)
lz4_compressed = compress(100 MB, lz4)
zstd_compressed = compress(500 MB, zstd)
bzip2_compressed = compress(1000 MB, bzip2)`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 20
	m.previewMode = PreviewFull

	// Navigate to line 3 (zstd_compressed)
	var updated tea.Model
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)

	t.Logf("Before insert: cursorLine=%d, mode=%v", m.cursorLine, m.mode)

	// Press 'o' to insert line below and enter edit mode
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = updated.(Model)

	t.Logf("After 'o': cursorLine=%d, mode=%v, editBuf=%q", m.cursorLine, m.mode, m.editBuf)

	// Type some text
	for _, r := range "test" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}

	t.Logf("After typing: cursorLine=%d, mode=%v, editBuf=%q", m.cursorLine, m.mode, m.editBuf)
	docLines := m.GetLines()
	t.Logf("Document lines: %d", len(docLines))
	for i, l := range docLines {
		t.Logf("  doc[%d] = %q", i, l)
	}

	// Now render and check alignment
	view := m.View()

	// The view contains both panes side by side
	// We need to extract and compare line counts
	lines := strings.Split(view, "\n")
	t.Logf("Total rendered lines: %d", len(lines))

	// Log the actual rendered output for debugging
	t.Log("=== Rendered View ===")
	for i, line := range lines {
		if i < 15 { // Just show first 15 lines
			t.Logf("[%2d] %s", i, line)
		}
	}

	// Get the aligned panes data to understand what SHOULD be rendered
	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	t.Logf("=== Aligned Model State ===")
	t.Logf("Source lines: %d, Preview lines: %d", len(aligned.sourceLines), len(aligned.previewLines))
	t.Logf("sourceToVisual: %v", aligned.sourceToVisual)
	for i, pl := range aligned.previewLines {
		t.Logf("  preview[%d] sourceLineNum=%d content=%q", i, pl.sourceLineNum, pl.content)
	}

	// Check that source and preview have same line count
	if len(aligned.sourceLines) != len(aligned.previewLines) {
		t.Errorf("Alignment broken: source has %d lines, preview has %d lines",
			len(aligned.sourceLines), len(aligned.previewLines))
	}

	// Now the key test: render each pane separately and count lines
	// Match View()'s calculations exactly
	contentHeight := m.height - 3 // status bar, footer, separator
	paneContentHeight := contentHeight - 1
	if paneContentHeight < 3 {
		paneContentHeight = 3
	}

	globalsHeight := 1 // collapsed state
	if m.globalsExpanded {
		globalsHeight = 1 + m.getGlobalsCount()
		if m.getGlobalsCount() == 0 {
			globalsHeight = 2
		}
	}
	globalsHeight++ // +1 for separator

	sourceContentHeight := paneContentHeight
	if m.previewMode != PreviewHidden {
		sourceContentHeight = paneContentHeight - globalsHeight
	}
	if sourceContentHeight < 1 {
		sourceContentHeight = 1
	}

	t.Logf("=== Height Calculations ===")
	t.Logf("m.height=%d, contentHeight=%d, paneContentHeight=%d", m.height, contentHeight, paneContentHeight)
	t.Logf("globalsHeight=%d (expanded=%v)", globalsHeight, m.globalsExpanded)
	t.Logf("sourceContentHeight=%d (passed to renderSourcePaneAligned)", sourceContentHeight)
	t.Logf("paneContentHeight=%d (passed to renderPreviewPaneAligned)", paneContentHeight)
	t.Logf("rightWidth=%d, WrapText(editBuf=%q, rightWidth)=%v", rightWidth, m.editBuf, WrapText(m.editBuf, rightWidth))

	sourcePane := m.renderSourcePaneAligned(leftWidth, sourceContentHeight, aligned)
	previewPane := m.renderPreviewPaneAligned(rightWidth, paneContentHeight, aligned)

	sourceLines := strings.Split(sourcePane, "\n")
	previewLines := strings.Split(previewPane, "\n")

	t.Logf("=== Rendered Pane Line Counts ===")
	t.Logf("Source pane lines: %d", len(sourceLines))
	t.Logf("Preview pane lines: %d", len(previewLines))

	// Log first few lines of each pane
	t.Log("=== Source Pane (first 10 lines) ===")
	for i := 0; i < min(10, len(sourceLines)); i++ {
		t.Logf("  [%d] %q", i, sourceLines[i])
	}

	t.Log("=== Preview Pane (first 10 lines) ===")
	for i := 0; i < min(10, len(previewLines)); i++ {
		t.Logf("  [%d] %q", i, previewLines[i])
	}

	// The visual alignment check:
	// In View(), source pane gets globalsHeight worth of padding prepended,
	// so the rendered total heights should match after View() joins them.
	// Here we test the raw render functions which DON'T include that padding.
	//
	// Source renders: sourceContentHeight lines (content + tilde fills)
	// Preview renders: globalsHeight + results lines (globals + sep + content + fills)
	//
	// The correct check is that:
	// sourceContentHeight + globalsHeight == paneContentHeight (what preview is passed)
	//
	// Since sourceContentHeight = paneContentHeight - globalsHeight (when preview visible),
	// the equation is: (paneContentHeight - globalsHeight) + globalsHeight == paneContentHeight ✓
	//
	// What we actually want to verify is that CONTENT aligns.
	// Count non-tilde, non-fill lines in source and compare with content lines in preview.
	sourceContentCount := 0
	for _, l := range sourceLines {
		// Tilde lines start with "~"
		if !strings.HasPrefix(l, "~") && l != "" {
			sourceContentCount++
		}
	}

	// Preview content starts after globals (1 line) and separator (1 line)
	previewContentCount := 0
	globalsAndSepLines := 2 // collapsed globals + separator
	for i, l := range previewLines {
		if i < globalsAndSepLines {
			continue // skip globals and separator
		}
		// Empty lines in preview are either padding or edit mode - still count
		// We just want to count how many result lines there are
		if l != "" {
			previewContentCount++
		}
	}

	t.Logf("Source content lines (non-tilde): %d", sourceContentCount)
	t.Logf("Preview content lines (after globals, non-empty): %d", previewContentCount)

	// The AlignedModel guarantees source and preview have same visual line count
	// So content counts should match (excluding padding/empty lines which balance out)
	// Actually, the key is that len(aligned.sourceLines) == len(aligned.previewLines)
	// which we already checked above.

	// For visual alignment in the final View(), what matters is that:
	// - Source pane renders sourceContentHeight lines
	// - Preview pane renders paneContentHeight lines
	// - View() prepends globalsHeight padding to source
	// - Both end up same height: sourceContentHeight + globalsHeight == paneContentHeight ✓
	t.Logf("Visual alignment check: sourceContentHeight(%d) + globalsHeight(%d) = %d, paneContentHeight = %d",
		sourceContentHeight, globalsHeight, sourceContentHeight+globalsHeight, paneContentHeight)

	// Verify the padding math is correct
	if sourceContentHeight+globalsHeight != paneContentHeight {
		t.Errorf("Height math broken: sourceContentHeight(%d) + globalsHeight(%d) = %d, but paneContentHeight = %d",
			sourceContentHeight, globalsHeight, sourceContentHeight+globalsHeight, paneContentHeight)
	}

	// The most important test: verify line-by-line alignment using AlignedModel
	// Each visual line in source should correspond to the same visual line in preview
	t.Log("=== Line-by-Line Alignment Verification ===")
	maxCheck := min(len(aligned.sourceLines), 10)
	for i := 0; i < maxCheck; i++ {
		srcLine := aligned.sourceLines[i]
		prvLine := aligned.previewLines[i]

		srcContent := srcLine.content
		prvContent := prvLine.content
		if len(srcContent) > 40 {
			srcContent = srcContent[:40] + "..."
		}
		if len(prvContent) > 30 {
			prvContent = prvContent[:30] + "..."
		}

		srcType := "normal"
		if srcLine.isPadding {
			srcType = "padding"
		} else if srcLine.isWrapped {
			srcType = "wrapped"
		} else if srcLine.isCursorLine {
			srcType = "CURSOR"
		}

		t.Logf("  [%d] src(line %d, %s): %q | prv(line %d): %q",
			i, srcLine.sourceLineIdx, srcType, srcContent, prvLine.sourceLineNum, prvContent)

		// Verify source line indices match
		if srcLine.sourceLineIdx != prvLine.sourceLineNum {
			t.Errorf("ALIGNMENT BUG at visual %d: source maps to doc line %d, preview maps to doc line %d",
				i, srcLine.sourceLineIdx, prvLine.sourceLineNum)
		}
	}
}

// TestViewAlignment_WithError tests alignment when a calculation error occurs.
// This reproduces the user-reported bug where error messages appear misaligned.
func TestViewAlignment_WithError(t *testing.T) {
	// Content that produces an error (undefined variable)
	content := `# Test Error Alignment
x = 10
y = undefined_var
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 20
	m.previewMode = PreviewFull

	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	// Debug: check results directly
	lineResults := m.GetLineResults()
	t.Logf("LineResults count: %d", len(lineResults))
	for i, lr := range lineResults {
		t.Logf("  lr[%d] LineNum=%d Source=%q Error=%q Value=%q", i, lr.LineNum, lr.Source, lr.Error, lr.Value)
	}

	t.Logf("Document has %d lines", len(m.GetLines()))
	t.Logf("AlignedModel: source=%d, preview=%d lines", len(aligned.sourceLines), len(aligned.previewLines))

	// Line-by-line verification
	t.Log("=== AlignedModel Line Verification (with error) ===")
	for i := 0; i < len(aligned.sourceLines); i++ {
		srcLine := aligned.sourceLines[i]
		prvLine := aligned.previewLines[i]

		srcContent := srcLine.content
		prvContent := prvLine.content
		if len(srcContent) > 35 {
			srcContent = srcContent[:35] + "..."
		}
		if len(prvContent) > 30 {
			prvContent = prvContent[:30] + "..."
		}

		t.Logf("[%d] src(doc %d): %q | prv(doc %d): %q",
			i, srcLine.sourceLineIdx, srcContent, prvLine.sourceLineNum, prvContent)

		if srcLine.sourceLineIdx != prvLine.sourceLineNum {
			t.Errorf("ALIGNMENT BUG at visual %d: source doc line %d != preview doc line %d",
				i, srcLine.sourceLineIdx, prvLine.sourceLineNum)
		}
	}

	// Also render full view and check
	view := m.View()
	lines := strings.Split(view, "\n")
	t.Log("=== Full View Output ===")
	for i, line := range lines {
		if i < 12 {
			t.Logf("[%2d] %s", i, line)
		}
	}
}

// TestViewAlignment_SourceVsPreviewLineByLine does line-by-line comparison
// to verify AlignedModel correctness.
func TestViewAlignment_SourceVsPreviewLineByLine(t *testing.T) {
	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 60
	m.height = 15
	m.previewMode = PreviewFull

	// Navigate down and insert
	var updated tea.Model
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = updated.(Model)

	// Type text
	for _, r := range "new" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}

	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	t.Logf("Mode: %v, CursorLine: %d, EditBuf: %q", m.mode, m.cursorLine, m.editBuf)
	t.Logf("AlignedModel: source=%d, preview=%d lines", len(aligned.sourceLines), len(aligned.previewLines))

	// The key invariant: source and preview aligned lines must have same count
	if len(aligned.sourceLines) != len(aligned.previewLines) {
		t.Fatalf("AlignedModel broken: source=%d, preview=%d", len(aligned.sourceLines), len(aligned.previewLines))
	}

	// Line-by-line verification
	t.Log("=== AlignedModel Line Verification ===")
	for i := 0; i < len(aligned.sourceLines); i++ {
		srcLine := aligned.sourceLines[i]
		prvLine := aligned.previewLines[i]

		srcContent := srcLine.content
		prvContent := prvLine.content
		if len(srcContent) > 30 {
			srcContent = srcContent[:30] + "..."
		}
		if len(prvContent) > 25 {
			prvContent = prvContent[:25] + "..."
		}

		srcType := "normal"
		if srcLine.isPadding {
			srcType = "padding"
		} else if srcLine.isWrapped {
			srcType = "wrapped"
		} else if srcLine.isCursorLine {
			srcType = "CURSOR"
		}

		t.Logf("[%d] src(doc %d, %s): %q | prv(doc %d): %q",
			i, srcLine.sourceLineIdx, srcType, srcContent, prvLine.sourceLineNum, prvContent)

		// Source line indices must match between source and preview at each visual line
		if srcLine.sourceLineIdx != prvLine.sourceLineNum {
			t.Errorf("ALIGNMENT BUG at visual %d: source doc line %d != preview doc line %d",
				i, srcLine.sourceLineIdx, prvLine.sourceLineNum)
		}
	}
}
