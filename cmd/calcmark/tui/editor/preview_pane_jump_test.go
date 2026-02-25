package editor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestPreviewPaneStability_CursorOnWrappingCalcLine verifies that the preview
// pane does not shift content when the cursor moves to a calc line whose result
// wraps to multiple visual lines in the aligned model.
//
// Root cause: renderPreviewPaneAligned computes editLineCount by wrapping editBuf
// at the preview pane width, but the pre-computed aligned model may have a different
// number of visual lines for that source line (preComputedCursorLineCount). The
// editBuf cursor-line path outputs editLineCount lines but skips ALL pre-computed
// lines via `continue`, creating a mismatch that shifts subsequent content.
//
// The bug manifests as: moving the cursor from one calc line to an adjacent one
// inserts or removes a blank line in the preview, causing headings and results
// below to "jump" by one line.
func TestPreviewPaneStability_CursorOnWrappingCalcLine(t *testing.T) {
	// This document reproduces the exact scenario from the user's bug report.
	// "fixed_total = rent + utilities + insurance" wraps differently in the
	// aligned model vs the editBuf, causing preview pane content to jump.
	content := `# Budget
## Income
salary = $5000
side_hustle = $800
total_income = salary + side_hustle

## Fixed Expenses
rent = $1500
utilities = $200
insurance = $150
fixed_total = rent + utilities + insurance

## Savings Target
savings_rate = 0.20
savings = total_income * savings_rate`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	// Find the source line indices we care about
	allLines := m.GetLines()
	insuranceLine := -1
	fixedTotalLine := -1
	savingsRateLine := -1
	for i, line := range allLines {
		if strings.HasPrefix(line, "insurance") {
			insuranceLine = i
		}
		if strings.HasPrefix(line, "fixed_total") {
			fixedTotalLine = i
		}
		if strings.HasPrefix(line, "savings_rate") {
			savingsRateLine = i
		}
	}
	if insuranceLine == -1 || fixedTotalLine == -1 || savingsRateLine == -1 {
		t.Fatalf("Could not find key lines: insurance=%d, fixed_total=%d, savings_rate=%d",
			insuranceLine, fixedTotalLine, savingsRateLine)
	}
	t.Logf("Source lines: insurance=%d, fixed_total=%d, savings_rate=%d",
		insuranceLine, fixedTotalLine, savingsRateLine)

	// Helper: navigate to a target line using handleDownKey (real user flow),
	// render View(), and extract preview pane lines.
	navigateAndRender := func(targetLine int, desc string) []string {
		// Reset to top and navigate down to target (simulates real arrow-key usage)
		m.cursorLine = 0
		m.cursorCol = 0
		m.editBuf = ""
		m.loadCurrentLineIntoEditBuffer()

		for m.cursorLine < targetLine {
			result, _ := m.handleDownKey()
			m = result.(Model)
		}

		view := m.View().Content
		lines := strings.Split(view, "\n")
		var previewLines []string
		for _, line := range lines {
			parts := strings.SplitN(line, "│", 2)
			if len(parts) >= 2 {
				previewLines = append(previewLines, stripAnsiCodes(parts[1]))
			}
		}

		t.Logf("=== Cursor on line %d (%s), editBuf=%q (len=%d) ===",
			m.cursorLine, desc, m.editBuf, len(m.editBuf))
		for i, line := range previewLines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" || (i > 0 && i < len(previewLines)-1) {
				t.Logf("  preview[%2d]: %q", i, truncatePreviewLine(line, 50))
			}
		}
		return previewLines
	}

	// Navigate to each target line and render
	previewOnInsurance := navigateAndRender(insuranceLine, "insurance")
	previewOnFixedTotal := navigateAndRender(fixedTotalLine, "fixed_total")
	previewOnSavingsRate := navigateAndRender(savingsRateLine, "savings_rate")

	// Find "Savings Target" heading position in each rendering
	findLine := func(lines []string, needle string) int {
		for i, line := range lines {
			if strings.Contains(line, needle) {
				return i
			}
		}
		return -1
	}

	savingsOnInsurance := findLine(previewOnInsurance, "Savings Target")
	savingsOnFixedTotal := findLine(previewOnFixedTotal, "Savings Target")
	savingsOnSavingsRate := findLine(previewOnSavingsRate, "Savings Target")

	t.Logf("'Savings Target' positions: insurance=%d, fixed_total=%d, savings_rate=%d",
		savingsOnInsurance, savingsOnFixedTotal, savingsOnSavingsRate)

	// The heading must NOT move regardless of cursor position
	if savingsOnInsurance == -1 {
		t.Fatal("'Savings Target' not found in preview when cursor on insurance")
	}
	if savingsOnFixedTotal != savingsOnInsurance {
		t.Errorf("Preview jumped: 'Savings Target' at line %d (cursor on insurance) vs line %d (cursor on fixed_total)",
			savingsOnInsurance, savingsOnFixedTotal)
	}
	if savingsOnSavingsRate != savingsOnInsurance {
		t.Errorf("Preview jumped: 'Savings Target' at line %d (cursor on insurance) vs line %d (cursor on savings_rate)",
			savingsOnInsurance, savingsOnSavingsRate)
	}
}

// TestPreviewPaneStability_EditLineCountVsPreComputed verifies the core invariant:
// the number of visual lines the preview pane outputs for the cursor's source line
// must match the number of pre-computed aligned lines, ensuring no content shift.
//
// This test directly checks the line count mismatch that causes the bug, independent
// of the visual output.
func TestPreviewPaneStability_EditLineCountVsPreComputed(t *testing.T) {
	content := `# Budget

## Fixed Expenses
rent = $1500
utilities = $200
insurance = $150
fixed_total = rent + utilities + insurance

## Savings
savings_rate = 0.20`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	allLines := m.GetLines()
	for cursorLine := range len(allLines) {
		m.cursorLine = cursorLine
		m.cursorCol = 0
		m.editBuf = "" // Clear first to avoid stale state
		m.loadCurrentLineIntoEditBuffer()

		if m.editBuf == "" {
			continue // Skip lines with empty editBuf (won't trigger the code path)
		}

		leftWidth, rightWidth := m.GetPaneWidths(m.width)
		const dividerWidth = 1
		leftContentWidth := leftWidth - dividerWidth
		aligned := m.computeAlignedPanes(leftContentWidth, rightWidth)

		// Count pre-computed visual lines for this source line in the aligned model
		preComputedCount := 0
		for _, pl := range aligned.previewLines {
			if pl.sourceLineNum == cursorLine {
				preComputedCount++
			}
		}

		// Count how many lines renderPreviewPaneAligned would output for this cursor line.
		// This mirrors the editLineCount computation in the rendering code.
		editLines := wrapTextForTest(m.editBuf, rightWidth)
		editLineCount := len(editLines)
		if editLineCount == 0 {
			editLineCount = 1
		}

		if editLineCount != preComputedCount {
			t.Errorf("Line %d (%q): editLineCount=%d but preComputedCount=%d — preview will shift by %d lines",
				cursorLine, truncatePreviewLine(allLines[cursorLine], 30),
				editLineCount, preComputedCount, editLineCount-preComputedCount)
		}
	}
}

// TestPreviewPaneStability_SourceAndPreviewLineCountMatch verifies that the
// source pane and preview pane render the same total number of content lines
// regardless of cursor position. A mismatch means they will scroll out of sync.
//
// Tests at multiple widths because the wrapping behavior changes with width.
//
// IMPORTANT: This test calls renderSourcePaneAligned and renderPreviewPaneAligned
// directly rather than parsing View() output. View() includes a context footer
// that uses "│" as value separators, which would create false positives if we
// counted lines containing "│" as pane rows.
func TestPreviewPaneStability_SourceAndPreviewLineCountMatch(t *testing.T) {
	content := `# Budget

## Fixed Expenses
rent = $1500
utilities = $200
insurance = $150
fixed_total = rent + utilities + insurance

## Savings
savings_rate = 0.20
savings = $5000 * savings_rate`

	// Test at multiple widths to catch wrapping-dependent bugs.
	// The source line "fixed_total = rent + utilities + insurance" (42 chars)
	// wraps differently at different widths.
	widths := []int{60, 70, 80, 100, 120}

	for _, width := range widths {
		t.Run(fmt.Sprintf("width_%d", width), func(t *testing.T) {
			doc, err := document.NewDocument(content)
			if err != nil {
				t.Fatalf("Failed to create document: %v", err)
			}

			m := New(doc)
			m.width = width
			m.height = 24
			m.previewMode = PreviewFull

			allLines := m.GetLines()

			leftWidth, rightWidth := m.GetPaneWidths(m.width)
			const dividerWidth = 1
			leftContentWidth := leftWidth - dividerWidth
			paneHeight := m.height - 7 // Approximate content height

			var referenceSourceCount, referencePreviewCount int
			for cursorLine := range len(allLines) {
				m.cursorLine = cursorLine
				m.cursorCol = 0
				m.editBuf = ""
				m.loadCurrentLineIntoEditBuffer()

				aligned := m.computeAlignedPanes(leftContentWidth, rightWidth)
				sourcePane := m.renderSourcePaneAligned(leftContentWidth, paneHeight, aligned)
				previewPane := m.renderPreviewPaneAligned(rightWidth, paneHeight, aligned)

				sourceLineCount := len(strings.Split(sourcePane, "\n"))
				previewLineCount := len(strings.Split(previewPane, "\n"))

				if cursorLine == 0 {
					referenceSourceCount = sourceLineCount
					referencePreviewCount = previewLineCount
				}

				// Source pane line count must be stable across cursor positions
				if sourceLineCount != referenceSourceCount {
					t.Errorf("width=%d line %d (%q): source pane lines changed from %d to %d",
						width, cursorLine, truncatePreviewLine(allLines[cursorLine], 30),
						referenceSourceCount, sourceLineCount)
				}

				// Preview pane line count must be stable across cursor positions
				if previewLineCount != referencePreviewCount {
					t.Errorf("width=%d line %d (%q): preview pane lines changed from %d to %d",
						width, cursorLine, truncatePreviewLine(allLines[cursorLine], 30),
						referencePreviewCount, previewLineCount)
				}

				// Source and preview must have equal line counts (1:1 alignment invariant)
				if sourceLineCount != previewLineCount {
					t.Errorf("width=%d line %d (%q): source has %d lines but preview has %d",
						width, cursorLine, truncatePreviewLine(allLines[cursorLine], 30),
						sourceLineCount, previewLineCount)
				}
			}
		})
	}
}

// TestPreviewPaneStability_AdjacentLineMovement verifies that moving the cursor
// one line at a time (simulating arrow-key navigation) does not cause any
// preview pane content to shift. Tests every adjacent pair of lines.
func TestPreviewPaneStability_AdjacentLineMovement(t *testing.T) {
	content := `---
globals:
  my_var: 42
---
# Budget

## Income
salary = $5000

## Fixed Expenses
rent = $1500
utilities = $200
insurance = $150
fixed_total = rent + utilities + insurance

## Savings
savings_rate = 0.20
savings = salary * savings_rate`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 30 // Tall enough to see all content
	m.previewMode = PreviewFull
	m.globalsExpanded = true

	allLines := m.GetLines()

	// Find stable anchor points in the preview — headings that should never move
	anchors := []string{"Budget", "Income", "Fixed Expenses", "Savings"}

	type anchorPosition struct {
		name string
		line int
	}

	// Helper: render and find anchor positions
	findAnchors := func(cursorLine int) []anchorPosition {
		m.cursorLine = cursorLine
		m.cursorCol = 0
		if cursorLine < len(allLines) {
			m.editBuf = allLines[cursorLine]
		}

		leftWidth, rightWidth := m.GetPaneWidths(m.width)
		const dividerWidth = 1
		leftContentWidth := leftWidth - dividerWidth
		aligned := m.computeAlignedPanes(leftContentWidth, rightWidth)
		preview := m.renderPreviewPaneAligned(rightWidth, m.height-2, aligned)
		previewLines := strings.Split(preview, "\n")

		var positions []anchorPosition
		for _, anchor := range anchors {
			pos := -1
			for i, line := range previewLines {
				if strings.Contains(stripAnsiCodes(line), anchor) {
					pos = i
					break
				}
			}
			positions = append(positions, anchorPosition{name: anchor, line: pos})
		}
		return positions
	}

	// Check every adjacent pair of cursor lines
	prevPositions := findAnchors(0)
	for cursorLine := 1; cursorLine < len(allLines); cursorLine++ {
		currPositions := findAnchors(cursorLine)

		for i, curr := range currPositions {
			prev := prevPositions[i]
			if prev.line == -1 || curr.line == -1 {
				continue // Anchor not visible in this scroll window
			}
			if curr.line != prev.line {
				t.Errorf("Cursor %d→%d: '%s' jumped from preview line %d to %d",
					cursorLine-1, cursorLine, curr.name, prev.line, curr.line)
			}
		}
		prevPositions = currPositions
	}
}

// wrapTextForTest wraps text at the given width, mirroring geometry.WrapText behavior.
func wrapTextForTest(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	if len(text) <= width {
		return []string{text}
	}
	var lines []string
	for len(text) > width {
		lines = append(lines, text[:width])
		text = text[width:]
	}
	if len(text) > 0 {
		lines = append(lines, text)
	}
	return lines
}
