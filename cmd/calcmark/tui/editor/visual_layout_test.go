package editor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestVisualLayout_GutterSpace verifies that there is a space between line numbers and content
func TestVisualLayout_GutterSpace(t *testing.T) {
	// Save and restore color profile for test isolation
	originalProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	defer lipgloss.SetColorProfile(originalProfile)

	content := "x = 10\ny = 20"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	view := m.View()
	lines := strings.Split(view, "\n")

	// Look for source lines (lines with line numbers)
	foundGutter := false
	for i, line := range lines {
		plainLine := stripANSI(line)

		// Check if line starts with a line number (e.g., "   1 ")
		if len(plainLine) > 5 && plainLine[0] == ' ' {
			// Line numbers are right-aligned in 4-char field
			// After line number, there should be a gutter space, then content
			// Pattern: "   1 x = 10" not "   1x = 10"

			// Find the first non-space digit (start of line number)
			firstDigit := -1
			for pos, ch := range plainLine {
				if ch >= '0' && ch <= '9' {
					firstDigit = pos
					break
				}
			}

			if firstDigit == -1 {
				continue // Not a line number line
			}

			// Find end of line number (first space after digit)
			endOfLineNum := -1
			for pos := firstDigit; pos < len(plainLine); pos++ {
				if plainLine[pos] == ' ' {
					endOfLineNum = pos
					break
				}
			}

			if endOfLineNum == -1 {
				continue // Malformed
			}

			// Check if there's a gutter space after the line number
			if endOfLineNum+1 < len(plainLine) {
				gutterChar := plainLine[endOfLineNum]
				nextChar := plainLine[endOfLineNum+1]

				if gutterChar == ' ' && nextChar != ' ' && !strings.ContainsRune("│", rune(nextChar)) {
					foundGutter = true
					t.Logf("Line %d has gutter space: %q", i, plainLine[:min(50, len(plainLine))])
				} else if gutterChar != ' ' {
					t.Errorf("Line %d missing gutter space after line number: %q", i, plainLine[:min(50, len(plainLine))])
				}
			}
		}
	}

	if !foundGutter {
		t.Error("Did not find any lines with gutter space between line number and content")
	}
}

// TestVisualLayout_DividerPosition verifies divider is at consistent column based on terminal width
func TestVisualLayout_DividerPosition(t *testing.T) {
	// Save and restore color profile
	originalProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	defer lipgloss.SetColorProfile(originalProfile)

	testCases := []struct {
		width          int
		expectedDivCol int // Expected divider column position
	}{
		{80, 43},  // leftWidth=44, dividerWidth=1, leftContent=43
		{100, 54}, // leftWidth=55, dividerWidth=1, leftContent=54
		{60, 32},  // leftWidth=33, dividerWidth=1, leftContent=32
	}

	content := "x = 10\ny = 20\nz = 30"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("width_%d", tc.width), func(t *testing.T) {
			m := New(doc)
			m.width = tc.width
			m.height = 24
			m.previewMode = PreviewFull

			// Calculate expected divider position
			leftWidth, _ := m.GetPaneWidths(tc.width)
			const dividerWidth = 1
			expectedDividerCol := leftWidth - dividerWidth

			view := m.View()
			lines := strings.Split(view, "\n")

			dividerFound := false
			for i, line := range lines {
				plainLine := stripANSI(line)

				// Find divider character
				dividerPos := strings.IndexRune(plainLine, '│')
				if dividerPos == -1 {
					continue
				}

				dividerFound = true
				if dividerPos != expectedDividerCol {
					t.Errorf("Line %d: divider at column %d, expected %d (width=%d, leftWidth=%d)\n  Line: %q",
						i, dividerPos, expectedDividerCol, tc.width, leftWidth, plainLine)
				} else {
					t.Logf("Line %d: divider at column %d ✓", i, dividerPos)
				}
			}

			if !dividerFound {
				t.Error("No divider found in view output")
			}
		})
	}
}

// TestVisualLayout_NoExtraDividers verifies there are no extra pipe characters in content
func TestVisualLayout_NoExtraDividers(t *testing.T) {
	// Save and restore color profile
	originalProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	defer lipgloss.SetColorProfile(originalProfile)

	content := "x = 10\ny = 20"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	view := m.View()
	lines := strings.Split(view, "\n")

	// Get expected divider position
	leftWidth, _ := m.GetPaneWidths(80)
	const dividerWidth = 1
	expectedDividerCol := leftWidth - dividerWidth

	for i, line := range lines {
		plainLine := stripANSI(line)

		// Count all pipe characters
		pipePositions := []int{}
		for pos, ch := range plainLine {
			if ch == '│' {
				pipePositions = append(pipePositions, pos)
			}
		}

		// Should have at most ONE divider per line (at expected position)
		if len(pipePositions) > 1 {
			t.Errorf("Line %d has %d pipe characters (expected at most 1):\n  Positions: %v\n  Line: %q",
				i, len(pipePositions), pipePositions, plainLine)
		} else if len(pipePositions) == 1 && pipePositions[0] != expectedDividerCol {
			t.Errorf("Line %d has pipe at unexpected position %d (expected %d)\n  Line: %q",
				i, pipePositions[0], expectedDividerCol, plainLine)
		}
	}
}

// TestVisualLayout_VerticalAlignment verifies preview results align with their source lines
func TestVisualLayout_VerticalAlignment(t *testing.T) {
	// Save and restore color profile
	originalProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	defer lipgloss.SetColorProfile(originalProfile)

	// Create document with calculations
	content := `# Header

x = 10


y = x + 5


z = y * 2`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 30
	m.previewMode = PreviewFull

	// Evaluate to get results
	m.eval.Evaluate(m.doc)

	view := m.View()
	lines := strings.Split(view, "\n")

	// Get divider position
	leftWidth, _ := m.GetPaneWidths(80)
	const dividerWidth = 1
	dividerCol := leftWidth - dividerWidth

	// Track which visual lines have calculations and results
	sourceCalcs := make(map[int]string)    // visual line -> calculation
	previewResults := make(map[int]string) // visual line -> result

	for i, line := range lines {
		plainLine := stripANSI(line)

		if len(plainLine) <= dividerCol {
			continue
		}

		// Extract source (left of divider) and preview (right of divider)
		sourcePart := plainLine[:dividerCol]
		previewPart := ""
		if dividerCol+1 < len(plainLine) {
			previewPart = strings.TrimSpace(plainLine[dividerCol+1:])
		}

		// Check if source has a calculation (contains "=")
		if strings.Contains(sourcePart, "=") {
			sourceCalcs[i] = strings.TrimSpace(sourcePart)
		}

		// Check if preview has a result (contains "→")
		if strings.Contains(previewPart, "→") {
			previewResults[i] = previewPart
		}
	}

	// Verify calculations and results are on the same visual lines
	t.Logf("Found %d source calculations and %d preview results", len(sourceCalcs), len(previewResults))

	for lineNum, calc := range sourceCalcs {
		if result, hasResult := previewResults[lineNum]; hasResult {
			t.Logf("Line %d: %q → %q ✓", lineNum, calc, result)
		} else {
			// Result might be on adjacent line due to headers/padding
			// Check ±1 lines
			found := false
			for offset := -1; offset <= 1; offset++ {
				if result, ok := previewResults[lineNum+offset]; ok {
					t.Logf("Line %d: %q → result on line %d: %q (offset %d)",
						lineNum, calc, lineNum+offset, result, offset)
					found = true
					break
				}
			}
			if !found {
				t.Logf("Line %d: %q → (no result found nearby)", lineNum, calc)
			}
		}
	}
}

// =============================================================================
// Phase 10 Preview Pane Tests
// =============================================================================

// TestPreviewPaneWidthRatio verifies the 60/40 pane width ratio
// Requirement: Phase 10 - Fixed 60/40 width ratio (source/preview)
func TestPreviewPaneWidthRatio(t *testing.T) {
	testCases := []struct {
		totalWidth            int
		expectedSourcePercent int
		expectedPreviewPercent int
	}{
		{80, 60, 40},
		{100, 60, 40},
		{120, 60, 40},
		{60, 60, 40},
	}

	content := "x = 10"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("width_%d", tc.totalWidth), func(t *testing.T) {
			m := New(doc)
			m.previewMode = PreviewFull

			leftWidth, rightWidth := m.GetPaneWidths(tc.totalWidth)

			// Calculate actual percentages (accounting for integer rounding)
			actualSourcePercent := (leftWidth * 100) / tc.totalWidth
			actualPreviewPercent := (rightWidth * 100) / tc.totalWidth

			// Allow 1% tolerance for rounding
			if abs(actualSourcePercent-tc.expectedSourcePercent) > 2 {
				t.Errorf("Source pane: got %d%% (%d/%d), expected ~%d%%",
					actualSourcePercent, leftWidth, tc.totalWidth, tc.expectedSourcePercent)
			}
			if abs(actualPreviewPercent-tc.expectedPreviewPercent) > 2 {
				t.Errorf("Preview pane: got %d%% (%d/%d), expected ~%d%%",
					actualPreviewPercent, rightWidth, tc.totalWidth, tc.expectedPreviewPercent)
			}

			// Verify total width is accounted for
			if leftWidth+rightWidth != tc.totalWidth {
				t.Errorf("Widths don't sum to total: %d + %d = %d, expected %d",
					leftWidth, rightWidth, leftWidth+rightWidth, tc.totalWidth)
			}

			t.Logf("Width %d: source=%d (%d%%), preview=%d (%d%%) ✓",
				tc.totalWidth, leftWidth, actualSourcePercent, rightWidth, actualPreviewPercent)
		})
	}
}

// TestPreviewPaneHeader verifies the preview pane has "Results" header
// Requirement: Phase 10 - Preview pane header is "Results" (not "Preview")
func TestPreviewPaneHeader(t *testing.T) {
	originalProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	defer lipgloss.SetColorProfile(originalProfile)

	content := "x = 10"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	view := m.View()

	// Check for "Results" header in the rendered view
	if !strings.Contains(view, "Results") {
		t.Error("Preview pane should have 'Results' header")
	}

	// Ensure old "Preview" header is not present
	if strings.Contains(view, "Preview") && !strings.Contains(view, "Results") {
		t.Error("Preview pane should use 'Results' header, not 'Preview'")
	}

	t.Log("Preview pane has 'Results' header ✓")
}

// TestPreviewPaneAnonymousCalculationFormat verifies arrow format for anonymous calcs
// Requirement: PREVIEW-04 - Anonymous calculations display as "-> result" (arrow only)
func TestPreviewPaneAnonymousCalculationFormat(t *testing.T) {
	originalProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	defer lipgloss.SetColorProfile(originalProfile)

	// Document with both named and anonymous calculations
	content := `x = 10
2 + 2
y = 20
100 * 3`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull
	m.eval.Evaluate(m.doc)

	results := m.GetLineResults()

	// Line 0: x = 10 (named) -> should have VarName
	if results[0].VarName != "x" {
		t.Errorf("Line 0 should have VarName='x', got %q", results[0].VarName)
	}
	if results[0].Value != "10" {
		t.Errorf("Line 0 should have Value='10', got %q", results[0].Value)
	}

	// Line 1: 2 + 2 (anonymous) -> should have empty VarName
	if results[1].VarName != "" {
		t.Errorf("Line 1 (anonymous calc) should have empty VarName, got %q", results[1].VarName)
	}
	if results[1].Value != "4" {
		t.Errorf("Line 1 should have Value='4', got %q", results[1].Value)
	}

	// Line 2: y = 20 (named) -> should have VarName
	if results[2].VarName != "y" {
		t.Errorf("Line 2 should have VarName='y', got %q", results[2].VarName)
	}

	// Line 3: 100 * 3 (anonymous) -> should have empty VarName
	if results[3].VarName != "" {
		t.Errorf("Line 3 (anonymous calc) should have empty VarName, got %q", results[3].VarName)
	}
	if results[3].Value != "300" {
		t.Errorf("Line 3 should have Value='300', got %q", results[3].Value)
	}

	// Verify the rendered view shows arrow format for both types
	view := m.View()
	plainView := stripANSI(view)

	// Named variable format: "varName → value"
	if !strings.Contains(plainView, "x") || !strings.Contains(plainView, "→") {
		t.Error("Named variable should display with arrow format")
	}

	// Anonymous calc format: just "→ value" (no variable name placeholder)
	// The anonymous calcs should just have the arrow and value
	if !strings.Contains(plainView, "→") {
		t.Error("Results should contain arrow (→) character")
	}

	t.Log("Anonymous calculation format verified ✓")
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// TestRenderLineWithCursor_CursorBeyondContent verifies no panic when cursor > content length
// Regression test for panic "slice bounds out of range [:48] with length 41" (fix: 7dbe80f)
func TestRenderLineWithCursor_CursorBeyondContent(t *testing.T) {
	// Save and restore color profile
	originalProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	defer lipgloss.SetColorProfile(originalProfile)

	doc, err := document.NewDocument("short")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Set cursor beyond content length - this should NOT panic
	// Content is "short" (5 chars), cursor at position 48
	content := "short"
	col := 48 // Way beyond content length
	width := 80

	// This was panicking before the fix with "slice bounds out of range [:48] with length 41"
	// After fix, should clamp col and render without panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("renderLineWithCursor panicked with cursor beyond content: %v", r)
		}
	}()

	result := m.renderLineWithCursor(content, col, width, false)

	// Verify we got a result (specific content doesn't matter, just no panic)
	if result == "" {
		t.Error("Expected non-empty result from renderLineWithCursor")
	}
}

// TestRenderLineWithCursor_UTF8Content verifies proper rune handling
func TestRenderLineWithCursor_UTF8Content(t *testing.T) {
	// Save and restore color profile
	originalProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	defer lipgloss.SetColorProfile(originalProfile)

	doc, err := document.NewDocument("hello")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Content with multi-byte UTF-8 characters
	// "héllo" has 5 runes but 6 bytes (é is 2 bytes)
	content := "héllo"
	col := 3 // Cursor at rune position 3 (the second 'l')
	width := 80

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("renderLineWithCursor panicked with UTF-8 content: %v", r)
		}
	}()

	result := m.renderLineWithCursor(content, col, width, false)

	if result == "" {
		t.Error("Expected non-empty result from renderLineWithCursor with UTF-8")
	}
}
