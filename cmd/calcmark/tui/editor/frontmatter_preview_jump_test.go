package editor

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestFrontmatterCursorMovePreviewStability verifies that moving the cursor
// between frontmatter lines does NOT cause the preview pane globals content
// to shift vertically. This is the regression test for the bug where:
//
//  1. Cursor on "globals:" (line 1) → my_var appears at correct preview position
//  2. Cursor moves to "  my_var: 42" (line 2) → my_var jumps up one line in preview
//
// Root cause: the editBuf cursor-line path in renderPreviewPaneAligned fires
// before the frontmatter→globals substitution path, bypassing globalsPanelIdx
// advancement and causing subsequent globals panel lines to shift.
func TestFrontmatterCursorMovePreviewStability(t *testing.T) {
	content := `---
globals:
  my_var: 42
  bolt: 432
---
# Monthly Budget

## Income
salary = $5000
`
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull
	m.globalsExpanded = true

	// Render the preview with cursor on different frontmatter lines.
	// The globals panel content should remain at the same vertical positions
	// regardless of which frontmatter line the cursor is on.

	// Helper: render preview pane and extract the visible lines
	renderPreview := func(cursorLine int, desc string) []string {
		m.cursorLine = cursorLine
		// Simulate navigation: editBuf is always populated after arrow-key movement
		lines := m.GetLines()
		if cursorLine < len(lines) {
			m.editBuf = lines[cursorLine]
		}

		leftWidth, rightWidth := m.GetPaneWidths(m.width)
		const dividerWidth = 1
		leftContentWidth := leftWidth - dividerWidth
		aligned := m.computeAlignedPanes(leftContentWidth, rightWidth, m.GetLineResults())
		preview := m.renderPreviewPaneAligned(rightWidth, m.height-2, aligned)
		previewLines := strings.Split(preview, "\n")

		t.Logf("=== Cursor on line %d (%s) ===", cursorLine, desc)
		for i, line := range previewLines {
			if i < 10 { // Only log first 10 lines for readability
				t.Logf("  preview[%d]: %q", i, truncatePreviewLine(line, 60))
			}
		}
		return previewLines
	}

	// Render with cursor on each frontmatter line (0-indexed):
	//   0: ---
	//   1: globals:
	//   2:   my_var: 42
	//   3:   bolt: 432
	//   4: ---
	previewCursor0 := renderPreview(0, "---")
	previewCursor1 := renderPreview(1, "globals:")
	previewCursor2 := renderPreview(2, "my_var: 42")
	previewCursor3 := renderPreview(3, "bolt: 432")
	previewCursor4 := renderPreview(4, "---")

	// Find the line containing "my_var" and "bolt" in each rendering.
	// These should be at the SAME vertical position regardless of cursor.
	findGlobalLine := func(lines []string, needle string) int {
		for i, line := range lines {
			if strings.Contains(line, needle) {
				return i
			}
		}
		return -1
	}

	myVarPositions := []int{
		findGlobalLine(previewCursor0, "my_var"),
		findGlobalLine(previewCursor1, "my_var"),
		findGlobalLine(previewCursor2, "my_var"),
		findGlobalLine(previewCursor3, "my_var"),
		findGlobalLine(previewCursor4, "my_var"),
	}

	boltPositions := []int{
		findGlobalLine(previewCursor0, "bolt"),
		findGlobalLine(previewCursor1, "bolt"),
		findGlobalLine(previewCursor2, "bolt"),
		findGlobalLine(previewCursor3, "bolt"),
		findGlobalLine(previewCursor4, "bolt"),
	}

	t.Logf("my_var positions: %v", myVarPositions)
	t.Logf("bolt positions: %v", boltPositions)

	// All positions should be the same (no jumping).
	// Check that my_var appears somewhere
	if myVarPositions[0] == -1 {
		t.Fatal("'my_var' not found in preview at all (cursor on line 0)")
	}

	for i := 1; i < len(myVarPositions); i++ {
		if myVarPositions[i] != myVarPositions[0] {
			t.Errorf("my_var position shifted when cursor moved to line %d: was at line %d, now at line %d",
				i, myVarPositions[0], myVarPositions[i])
		}
	}

	for i := 1; i < len(boltPositions); i++ {
		if boltPositions[i] != boltPositions[0] {
			t.Errorf("bolt position shifted when cursor moved to line %d: was at line %d, now at line %d",
				i, boltPositions[0], boltPositions[i])
		}
	}
}

// TestNonFrontmatterCursorMovePreviewStability verifies the general case:
// moving the cursor between adjacent calc lines should not cause the preview
// pane to shift by inserting or removing blank lines.
//
// This catches the related bug where moving from line 15 (insurance) to
// line 16 (fixed_total) adds an extra blank line in the preview before
// the "Savings Target" heading.
func TestNonFrontmatterCursorMovePreviewStability(t *testing.T) {
	content := `---
globals:
  my_var: 42
---
# Monthly Budget

## Income
salary = $5000

## Fixed Expenses
rent = $1500
utilities = $200
insurance = $150
fixed_total = rent + utilities + insurance

## Savings
savings_rate = 20%
savings = salary * savings_rate
`
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 30 // Taller to fit all content
	m.previewMode = PreviewFull
	m.globalsExpanded = true

	// Helper to render preview and find a content line
	renderAndFindLine := func(cursorLine int, desc string) ([]string, int) {
		m.cursorLine = cursorLine
		lines := m.GetLines()
		if cursorLine < len(lines) {
			m.editBuf = lines[cursorLine]
		}

		leftWidth, rightWidth := m.GetPaneWidths(m.width)
		const dividerWidth = 1
		leftContentWidth := leftWidth - dividerWidth
		aligned := m.computeAlignedPanes(leftContentWidth, rightWidth, m.GetLineResults())
		preview := m.renderPreviewPaneAligned(rightWidth, m.height-2, aligned)
		previewLines := strings.Split(preview, "\n")

		t.Logf("=== Cursor on line %d (%s): %d preview lines ===", cursorLine, desc, len(previewLines))
		return previewLines, len(previewLines)
	}

	// Find the source line indices for "insurance" and "fixed_total"
	allLines := m.GetLines()
	insuranceLine := -1
	fixedTotalLine := -1
	for i, line := range allLines {
		if strings.Contains(line, "insurance = ") {
			insuranceLine = i
		}
		if strings.Contains(line, "fixed_total = ") {
			fixedTotalLine = i
		}
	}
	if insuranceLine == -1 || fixedTotalLine == -1 {
		t.Fatalf("Could not find insurance (%d) or fixed_total (%d) lines", insuranceLine, fixedTotalLine)
	}
	t.Logf("insurance at source line %d, fixed_total at source line %d", insuranceLine, fixedTotalLine)

	// Render with cursor on insurance vs fixed_total
	previewInsurance, _ := renderAndFindLine(insuranceLine, "insurance")
	previewFixedTotal, _ := renderAndFindLine(fixedTotalLine, "fixed_total")

	// Find the "Savings" heading in each rendering — it should be at the same position
	findLine := func(lines []string, needle string) int {
		for i, line := range lines {
			if strings.Contains(line, needle) {
				return i
			}
		}
		return -1
	}

	savingsInInsurance := findLine(previewInsurance, "Savings")
	savingsInFixedTotal := findLine(previewFixedTotal, "Savings")

	t.Logf("'Savings' heading position: cursor on insurance=%d, cursor on fixed_total=%d",
		savingsInInsurance, savingsInFixedTotal)

	if savingsInInsurance != savingsInFixedTotal {
		t.Errorf("'Savings' heading shifted position when cursor moved from insurance (line %d, pos %d) to fixed_total (line %d, pos %d)",
			insuranceLine, savingsInInsurance, fixedTotalLine, savingsInFixedTotal)
	}
}

// truncatePreviewLine truncates a string for readable test logs, stripping ANSI codes.
func truncatePreviewLine(s string, maxLen int) string {
	// Strip ANSI escape codes for readability
	cleaned := stripAnsiCodes(s)
	if len(cleaned) > maxLen {
		return cleaned[:maxLen] + "..."
	}
	return cleaned
}

// stripAnsiCodes removes ANSI escape sequences from a string.
func stripAnsiCodes(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
