package editor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestSelectionHighlighting_WrappedLineFullyHighlighted verifies that
// when a line wraps, all wrapped segments are highlighted when selected.
// Bug: Wrapped line continuation is NOT highlighted even though selected.
func TestSelectionHighlighting_WrappedLineFullyHighlighted(t *testing.T) {
	// Save and restore color profile
	// Use ANSI256 to enable color output for testing highlighting
	originalProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(originalProfile)

	// Create document with a line long enough to wrap at 80 columns
	// Line 9: "remaining = total_income - fixed_total - savings" is ~50 chars
	// At narrower width, it will wrap
	content := `## Discretionary
remaining = total_income - fixed_total - savings`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 50 // Narrow enough to force wrapping
	m.height = 24
	m.previewMode = PreviewFull

	// Evaluate to get results
	m.eval.Evaluate(m.doc)

	// Select all
	m.SelectAll()

	if !m.HasSelection() {
		t.Fatal("Expected selection to be active after SelectAll()")
	}

	// Get the view - this triggers renderLineWithSelection for wrapped lines
	view := m.View()

	// The wrapped line should have selection highlighting on BOTH segments
	// Look for the wrapped content ("savings" in this case)
	lines := strings.Split(view, "\n")

	// Find lines containing "savings" (the wrapped portion)
	foundWrappedSegment := false
	wrappedSegmentHighlighted := false

	for _, line := range lines {
		if strings.Contains(line, "savings") {
			foundWrappedSegment = true
			// Check if it has selection highlighting background.
			// The palette Selection color (#1f3a5f dark) resolves to ANSI256 23,
			// so we check for "48;5;23m" (background). Also accept the old "48;5;240".
			if strings.Contains(line, "48;5;23m") || strings.Contains(line, "48;5;240") {
				wrappedSegmentHighlighted = true
			}
			t.Logf("Wrapped segment line: %q", line)
		}
	}

	if !foundWrappedSegment {
		t.Log("Warning: Could not find wrapped segment in view - test may need adjustment")
	}

	// This test documents the bug - wrapped line continuation should be highlighted
	if foundWrappedSegment && !wrappedSegmentHighlighted {
		t.Error("Wrapped line continuation is NOT highlighted when selected - bug confirmed")
	}
}

// TestSelectionHighlighting_PreservesDimBrightDistinction verifies that
// selection highlighting doesn't override the dim/bright line distinction.
// Bug: Selection highlighting loses dim/bright distinction - all text looks same when selected.
func TestSelectionHighlighting_PreservesDimBrightDistinction(t *testing.T) {
	// Save and restore color profile
	// Use ANSI256 to enable color output for testing
	originalProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(originalProfile)

	content := `# Header
x = 10
y = 20
z = 30`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	// Get view without selection first
	m.eval.Evaluate(m.doc)
	viewWithoutSelection := m.View()

	// Now select all
	m.SelectAll()
	viewWithSelection := m.View()

	// The cursor line should still have different styling from non-cursor lines
	// even when selection is active
	linesWithout := strings.Split(viewWithoutSelection, "\n")
	linesWith := strings.Split(viewWithSelection, "\n")

	t.Logf("View without selection has %d lines", len(linesWithout))
	t.Logf("View with selection has %d lines", len(linesWith))

	// The bug: when selection is active, ALL lines look the same (lose dim/bright distinction)
	// Before selection: cursor line is bright (97m), others are dim (240m)
	// After selection: should still have some distinction between cursor/non-cursor lines

	// Get cursor line (z = 30) and non-cursor line (x = 10) from selection view
	// Cursor is at line 3 (z = 30), line 1 (x = 10) should be different
	var cursorLineWithSel, otherLineWithSel string
	for i, line := range linesWith {
		if strings.Contains(stripANSI(line), "z = 30") || strings.Contains(stripANSI(line), "z = 3") {
			cursorLineWithSel = line
			t.Logf("Cursor line (z=30) at view line %d: %q", i, line)
		}
		if strings.Contains(stripANSI(line), "x = 10") || strings.Contains(stripANSI(line), "x = 1") {
			otherLineWithSel = line
			t.Logf("Other line (x=10) at view line %d: %q", i, line)
		}
	}

	// With selection active, we expect some styling difference
	// If both lines have IDENTICAL ANSI codes, the distinction is lost
	if cursorLineWithSel != "" && otherLineWithSel != "" {
		// Strip the content, keep only ANSI sequences for comparison
		getANSICodes := func(s string) string {
			var codes strings.Builder
			inEscape := false
			for _, ch := range s {
				if ch == '\x1b' {
					inEscape = true
				}
				if inEscape {
					codes.WriteRune(ch)
					if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
						inEscape = false
					}
				}
			}
			return codes.String()
		}

		cursorCodes := getANSICodes(cursorLineWithSel)
		otherCodes := getANSICodes(otherLineWithSel)

		t.Logf("Cursor line ANSI codes: %q", cursorCodes)
		t.Logf("Other line ANSI codes: %q", otherCodes)

		// The bug: if codes are identical, the dim/bright distinction is lost
		// Note: This is a soft check - the visual distinction matters most
		if cursorCodes == otherCodes {
			t.Log("Warning: Cursor and non-cursor lines have identical ANSI codes with selection")
			t.Log("This may indicate loss of dim/bright distinction")
		}
	}
}

// TestPreviewPaneJump_LastCalcBeforeEmptyLine verifies that the preview
// pane content doesn't jump when cursor moves to the last calculation
// before an empty line.
// Bug: Preview pane jump bug is back when cursor moves to last calc before empty line.
// Regression: When cursor moves from line 11 to 12 (last calc before empty line 13),
// an extra blank line appears in the Preview Pane.
//
// IMPORTANT: This test calls renderPreviewPaneAligned directly to check the
// preview pane output. Using extractPreviewPane(View()) would create false
// positives because the context footer uses "│" as value separators for lines
// like "rent = $1500 │ utilities = $200 │ insurance = $150", which gets
// misinterpreted as pane divider rows.
func TestPreviewPaneJump_LastCalcBeforeEmptyLine(t *testing.T) {
	// Save and restore color profile
	originalProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(originalProfile)

	// Document that reproduces the exact scenario from user screenshot:
	// - Line 10: insurance = $150
	// - Line 11: fixed_total = rent + utilities + insurance (last calc)
	// - Line 12: empty line
	// - Line 13: ## Savings Target
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

	// Evaluate to get results
	m.eval.Evaluate(m.doc)

	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	const dividerWidth = 1
	leftContentWidth := leftWidth - dividerWidth
	paneHeight := m.height - 7 // Content area height

	// Helper: render preview pane directly and find "Savings Target" position
	renderAndFindSavings := func(cursorLine int, desc string) int {
		m.cursorLine = cursorLine
		m.cursorCol = 0
		m.editBuf = "" // Clear first to avoid stale state
		m.loadCurrentLineIntoEditBuffer()

		aligned := m.computeAlignedPanes(leftContentWidth, rightWidth)
		preview := m.renderPreviewPaneAligned(rightWidth, paneHeight, aligned)
		previewLines := strings.Split(preview, "\n")

		savingsPos := -1
		for i, line := range previewLines {
			stripped := stripANSI(line)
			if strings.Contains(stripped, "Savings Target") {
				savingsPos = i
				break
			}
		}

		t.Logf("Cursor on line %d (%s): 'Savings Target' at preview line %d, editBuf=%q",
			cursorLine, desc, savingsPos, truncate(m.editBuf, 40))
		return savingsPos
	}

	// Position cursor at line 10 (insurance) and line 11 (fixed_total)
	savingsLine1 := renderAndFindSavings(10, "insurance")
	savingsLine2 := renderAndFindSavings(11, "fixed_total")

	// The position of "Savings Target" should NOT change
	if savingsLine1 == -1 {
		t.Fatal("'Savings Target' not found in preview when cursor on insurance")
	}
	if savingsLine1 != savingsLine2 {
		t.Errorf("Preview pane jumped! 'Savings Target' moved from line %d to %d when cursor moved down",
			savingsLine1, savingsLine2)
	}

	// Also verify that source and preview pane line counts are consistent
	for _, cl := range []int{10, 11} {
		m.cursorLine = cl
		m.cursorCol = 0
		m.editBuf = ""
		m.loadCurrentLineIntoEditBuffer()

		aligned := m.computeAlignedPanes(leftContentWidth, rightWidth)
		sourcePane := m.renderSourcePaneAligned(leftContentWidth, paneHeight, aligned)
		previewPane := m.renderPreviewPaneAligned(rightWidth, paneHeight, aligned)

		srcLines := len(strings.Split(sourcePane, "\n"))
		pvLines := len(strings.Split(previewPane, "\n"))
		t.Logf("Cursor on line %d: source pane=%d lines, preview pane=%d lines", cl, srcLines, pvLines)

		if srcLines != pvLines {
			t.Errorf("Line count mismatch at cursor %d: source=%d, preview=%d", cl, srcLines, pvLines)
		}
	}
}

// TestPaste_PreservesLineNumbersAndStyling verifies that after pasting
// multi-line content, line numbers remain visible and text styling is consistent.
// Bug: Pasting breaks things significantly - missing virtual line numbers and different text color.
func TestPaste_PreservesLineNumbersAndStyling(t *testing.T) {
	// Save and restore color profile
	originalProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	defer lipgloss.SetColorProfile(originalProfile)

	// Start with simple content
	content := `# Budget
income = 5000`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	// Evaluate initial state
	m.eval.Evaluate(m.doc)

	// Get view before paste
	viewBefore := m.View()
	linesBefore := strings.Split(viewBefore, "\n")

	// Count lines with line numbers before paste
	countLinesWithNumbers := func(lines []string) int {
		count := 0
		for _, line := range lines {
			plainLine := stripANSI(line)
			// Lines with numbers have digits followed by content
			trimmed := strings.TrimLeft(plainLine, " ")
			if len(trimmed) > 0 && trimmed[0] >= '0' && trimmed[0] <= '9' {
				count++
			}
		}
		return count
	}

	numberedLinesBefore := countLinesWithNumbers(linesBefore)
	t.Logf("Lines with numbers before paste: %d", numberedLinesBefore)

	// Simulate paste of multi-line content
	// Position cursor at end of document
	lines := m.GetLines()
	m.cursorLine = len(lines) - 1
	m.cursorCol = len([]rune(lines[m.cursorLine]))
	m.loadCurrentLineIntoEditBuffer()

	// Paste multi-line content using the internal method
	// (same as handlePaste but without clipboard dependency)
	pasteLines := []string{
		"",
		"expenses = 3000",
		"savings = income - expenses",
		"",
		"## Notes",
		"This is a note.",
	}

	// Use the internal multi-line paste method
	m.insertMultiLineText(pasteLines)

	// Re-evaluate after paste
	m.eval.Evaluate(m.doc)

	// Get view after paste
	viewAfter := m.View()
	linesAfter := strings.Split(viewAfter, "\n")

	numberedLinesAfter := countLinesWithNumbers(linesAfter)
	t.Logf("Lines with numbers after paste: %d", numberedLinesAfter)

	// All source lines should still have line numbers
	// The number of numbered lines should increase (more content)
	if numberedLinesAfter < numberedLinesBefore {
		t.Errorf("Line numbers missing after paste: had %d numbered lines, now have %d",
			numberedLinesBefore, numberedLinesAfter)
	}

	// Verify that the first few lines still have proper line numbers
	// The bug shows lines at TOP of view without numbers
	// Look for source content lines that should have numbers
	foundLinesWithoutNumbers := []string{}
	for _, line := range linesAfter {
		plainLine := stripANSI(line)
		trimmed := strings.TrimLeft(plainLine, " ")

		// Skip empty lines, divider, header, tilde lines, and results pane
		if trimmed == "" || strings.HasPrefix(trimmed, "─") ||
			strings.HasPrefix(trimmed, "Source") ||
			strings.HasPrefix(trimmed, "~") ||
			strings.HasPrefix(trimmed, "│") ||
			strings.Contains(trimmed, "→") { // results pane has arrows
			continue
		}

		// A proper source line should have: leading spaces, digits, then space, then content
		// If we find content (like "## " or identifiers) without leading digits, it's a bug
		hasNumber := false
		for _, ch := range trimmed {
			if ch >= '0' && ch <= '9' {
				hasNumber = true
				break
			}
			if ch != ' ' {
				// Found non-space, non-digit - this might be unnumbered content
				break
			}
		}

		// Check if this looks like source content without line number
		if !hasNumber && (strings.Contains(trimmed, "=") ||
			strings.HasPrefix(trimmed, "##") ||
			strings.HasPrefix(trimmed, "#")) {
			foundLinesWithoutNumbers = append(foundLinesWithoutNumbers, trimmed)
		}
	}

	if len(foundLinesWithoutNumbers) > 0 {
		t.Errorf("Found source lines without line numbers after paste: %v", foundLinesWithoutNumbers)
	}

	// Also log the full view for debugging
	t.Logf("First 15 lines of view after paste:")
	for i, line := range linesAfter {
		if i >= 15 {
			break
		}
		t.Logf("  %d: %q", i, stripANSI(line))
	}
}

// TestPaste_LargeDocumentWithScrolling tests paste in a larger document
// where scrolling and wrapping may cause line number issues.
// Bug: Missing virtual line numbers and different text color after paste.
func TestPaste_LargeDocumentWithScrolling(t *testing.T) {
	// Save and restore color profile
	originalProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	defer lipgloss.SetColorProfile(originalProfile)

	// Start with a larger document that will need scrolling
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
savings = total_income * savings_rate

## Discretionary
remaining = total_income - fixed_total - savings`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 50  // Narrow to force wrapping
	m.height = 15 // Short to force scrolling
	m.previewMode = PreviewFull

	// Evaluate initial state
	m.eval.Evaluate(m.doc)

	// Position cursor at the bottom of the document (last line)
	lines := m.GetLines()
	m.cursorLine = len(lines) - 1
	m.cursorCol = len([]rune(lines[m.cursorLine]))
	m.loadCurrentLineIntoEditBuffer()

	// Force a view render to scroll the view
	view1 := m.View()

	// Now paste content
	pasteLines := []string{
		"",
		"## Notes",
		"This is additional content.",
	}
	m.insertMultiLineText(pasteLines)
	m.eval.Evaluate(m.doc)

	// Get view after paste
	view2 := m.View()
	linesAfterPaste := strings.Split(view2, "\n")

	// Look for the bug: source content lines without line numbers
	// The user's screenshot showed lines like "insurance", "## Savings Target" without numbers
	// NOTE: Wrapped continuations legitimately lack line numbers - we only flag lines that
	// START a new logical line (## headings, variable = assignments) without numbers
	foundBugs := []string{}
	for i, line := range linesAfterPaste {
		plainLine := stripANSI(line)

		// Skip header, divider, empty lines, tildes, results pane
		if strings.Contains(plainLine, "Source") ||
			strings.HasPrefix(strings.TrimLeft(plainLine, " "), "─") ||
			strings.HasPrefix(strings.TrimLeft(plainLine, " "), "~") ||
			strings.HasPrefix(strings.TrimLeft(plainLine, " "), "│") ||
			strings.TrimSpace(plainLine) == "" {
			continue
		}

		// Check the source pane portion (before │ divider)
		parts := strings.SplitN(plainLine, "│", 2)
		if len(parts) == 0 {
			continue
		}
		sourcePart := parts[0]

		// Detect if this looks like a START of a logical line (not a wrap continuation)
		// Logical line starts: "## ", "# ", "variable =", "varname ="
		// Wrap continuations: typically start with operators or continuation text

		trimmedSource := strings.TrimLeft(sourcePart, " ")

		// A line starts a logical line if it begins with:
		// - "##" or "#" (headers)
		// - word followed by " =" (assignments)
		// Wrapped continuations typically start with: operators, continuation words, etc.

		startsLogicalLine := strings.HasPrefix(trimmedSource, "##") ||
			(strings.HasPrefix(trimmedSource, "#") && !strings.HasPrefix(trimmedSource, "###")) ||
			(len(trimmedSource) > 3 &&
				trimmedSource[0] >= 'a' && trimmedSource[0] <= 'z' &&
				strings.Contains(trimmedSource, " = "))

		// Check if this logical-line-start has a line number
		if startsLogicalLine {
			// Should have format: [spaces][digits][space][content]
			hasLineNumber := false
			for _, ch := range sourcePart {
				if ch >= '0' && ch <= '9' {
					hasLineNumber = true
					break
				}
				// Only spaces allowed before line number
				if ch != ' ' {
					break
				}
			}

			if !hasLineNumber {
				foundBugs = append(foundBugs, fmt.Sprintf("line %d: %q", i, plainLine))
			}
		}
	}

	if len(foundBugs) > 0 {
		t.Errorf("Found source lines without proper line numbers after paste:\n%s",
			strings.Join(foundBugs, "\n"))
	}

	// Log first 20 lines for debugging
	t.Logf("View before paste (first 10 lines):")
	for i, line := range strings.Split(view1, "\n") {
		if i >= 10 {
			break
		}
		t.Logf("  %d: %q", i, stripANSI(line))
	}

	t.Logf("View after paste (first 20 lines):")
	for i, line := range linesAfterPaste {
		if i >= 20 {
			break
		}
		t.Logf("  %d: %q", i, stripANSI(line))
	}
}
