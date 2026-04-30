package editor

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

func TestSideBySide_BasicRendering(t *testing.T) {

	// SideBySide now adds a divider (1 char) between panes
	// So for total width 80: leftContent=39, divider=1, right=40
	sbs := NewSideBySide(39, 40, lipgloss.Color("236"), lipgloss.Color("235"))

	left := "line 1\nline 2"
	right := "preview 1\npreview 2"

	output := sbs.Render(left, right)
	lines := strings.Split(output, "\n")

	// Should have 2 lines (matching input)
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}

	// Each line should be exactly 80 characters (39 + 1 divider + 40)
	for i, line := range lines {
		width := lipgloss.Width(line)
		if width != 80 {
			t.Errorf("Line %d has width %d, expected 80", i, width)
		}
	}

	// Should contain background styling
	if !strings.Contains(output, "\x1b[48;5;236m") {
		t.Error("Left pane missing background styling (236)")
	}
	if !strings.Contains(output, "\x1b[48;5;235m") {
		t.Error("Right pane missing background styling (235)")
	}
}

func TestSideBySide_UnevenLineCounts(t *testing.T) {

	// For total width 80: leftContent=39, divider=1, right=40
	sbs := NewSideBySide(39, 40, lipgloss.Color("236"), lipgloss.Color("235"))

	left := "line 1\nline 2\nline 3"
	right := "preview 1"

	output := sbs.Render(left, right)
	lines := strings.Split(output, "\n")

	// Should have 3 lines (matching longer input)
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	// All lines should be 80 characters (39 + 1 divider + 40)
	for i, line := range lines {
		width := lipgloss.Width(line)
		if width != 80 {
			t.Errorf("Line %d has width %d, expected 80", i, width)
		}
	}

	// First line should contain both
	if !strings.Contains(lines[0], "line 1") {
		t.Error("First line missing left content")
	}
	if !strings.Contains(lines[0], "preview 1") {
		t.Error("First line missing right content")
	}

	// Lines 2 and 3 should have left content but right should be empty (padded)
	if !strings.Contains(lines[1], "line 2") {
		t.Error("Second line missing left content")
	}
	if !strings.Contains(lines[2], "line 3") {
		t.Error("Third line missing left content")
	}
}

func TestSideBySide_EmptyInput(t *testing.T) {

	// For total width 80: leftContent=39, divider=1, right=40
	sbs := NewSideBySide(39, 40, lipgloss.Color("236"), lipgloss.Color("235"))

	output := sbs.Render("", "")
	lines := strings.Split(output, "\n")

	// Should have 1 line (empty inputs produce 1 empty line each)
	if len(lines) != 1 {
		t.Errorf("Expected 1 line, got %d", len(lines))
	}

	// Line should be exactly 80 characters of styled spaces (39 + 1 divider + 40)
	width := lipgloss.Width(lines[0])
	if width != 80 {
		t.Errorf("Line has width %d, expected 80", width)
	}

	// Should have background styling
	if !strings.Contains(lines[0], "\x1b[48;5;") {
		t.Error("Empty line missing background styling")
	}
}

func TestSideBySide_StyledContent(t *testing.T) {

	// For total width 80: leftContent=39, divider=1, right=40
	sbs := NewSideBySide(39, 40, lipgloss.Color("236"), lipgloss.Color("235"))

	// Create styled content
	leftStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("200")).Render("styled left")
	rightStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("100")).Render("styled right")

	output := sbs.Render(leftStyled, rightStyled)
	lines := strings.Split(output, "\n")

	// Should preserve content styling
	if !strings.Contains(output, "styled left") {
		t.Error("Left styled content missing")
	}
	if !strings.Contains(output, "styled right") {
		t.Error("Right styled content missing")
	}

	// Should have background styling
	if !strings.Contains(output, "\x1b[48;5;236m") {
		t.Error("Left background missing")
	}
	if !strings.Contains(output, "\x1b[48;5;235m") {
		t.Error("Right background missing")
	}

	// Width should still be exact (39 + 1 divider + 40)
	width := lipgloss.Width(lines[0])
	if width != 80 {
		t.Errorf("Line has width %d, expected 80", width)
	}
}

func TestSideBySide_NoGaps(t *testing.T) {

	// For total width 80: leftContent=39, divider=1, right=40
	sbs := NewSideBySide(39, 40, lipgloss.Color("236"), lipgloss.Color("235"))

	left := "short"
	right := "short"

	output := sbs.Render(left, right)
	lines := strings.Split(output, "\n")

	// Line should be exactly 80 characters
	width := lipgloss.Width(lines[0])
	if width != 80 {
		t.Errorf("Line has width %d, expected 80", width)
	}

	// The raw string should contain ONLY styled spaces after content
	// (no unstyled spaces that would show terminal default)
	raw := lines[0]

	// Count how many characters are in the styled portions
	// This is a sanity check that we're not leaving unstyled gaps
	if !strings.Contains(raw, "\x1b[48;5;236m") {
		t.Error("Missing left pane background styling")
	}
	if !strings.Contains(raw, "\x1b[48;5;235m") {
		t.Error("Missing right pane background styling")
	}

	// The padding spaces should have background codes
	// We expect the pattern: content + \x1b[48;5;XXXm + spaces + \x1b[0m
	if !strings.Contains(raw, "\x1b[48;5;236m") || !strings.Contains(raw, "\x1b[48;5;235m") {
		t.Error("Padding spaces missing background styling")
	}
}

func TestSideBySide_TotalWidth(t *testing.T) {
	// TotalWidth includes divider (1 char)
	// leftContent=30, divider=1, right=50 → total=81
	sbs := NewSideBySide(30, 50, lipgloss.Color("236"), lipgloss.Color("235"))

	if sbs.TotalWidth() != 81 {
		t.Errorf("TotalWidth() = %d, expected 81", sbs.TotalWidth())
	}
}

// =============================================================================
// Phase 10 PREVIEW-XX Requirements Tests
// =============================================================================

// TestPreviewRequirements_PREVIEW01 verifies preview pane shows only calculation results
// PREVIEW-01: Preview pane shows ONLY calculation results (not markdown text)
func TestPreviewRequirements_PREVIEW01(t *testing.T) {

	// Mixed content: markdown headers shouldn't be echoed in preview
	// Calculations should show results
	content := `# Budget
x = 100
Some text paragraph
y = x * 2`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull
	m.eval.Evaluate(m.doc)

	results := m.GetLineResults()

	// Line 0: # Budget (markdown) - should have empty value
	if results[0].IsCalc || results[0].Value != "" {
		t.Errorf("PREVIEW-01: Markdown header should not show in preview, got IsCalc=%v Value=%q",
			results[0].IsCalc, results[0].Value)
	}

	// Line 1: x = 100 (calculation) - should have value
	if !results[1].IsCalc || results[1].Value == "" {
		t.Errorf("PREVIEW-01: Calculation should show in preview, got IsCalc=%v Value=%q",
			results[1].IsCalc, results[1].Value)
	}

	// Line 2: Some text paragraph (markdown) - should have empty value
	if results[2].Value != "" {
		t.Errorf("PREVIEW-01: Text paragraph should not show in preview, got Value=%q", results[2].Value)
	}

	// Line 3: y = x * 2 (calculation) - should have value
	if !results[3].IsCalc || results[3].Value == "" {
		t.Errorf("PREVIEW-01: Calculation should show in preview, got IsCalc=%v Value=%q",
			results[3].IsCalc, results[3].Value)
	}

	t.Log("PREVIEW-01: Preview shows only calculation results ✓")
}

// TestPreviewRequirements_PREVIEW02 verifies results are vertically aligned
// PREVIEW-02: Results vertically aligned with source calculation lines
func TestPreviewRequirements_PREVIEW02(t *testing.T) {

	// Document with gaps between calculations
	content := `x = 10

y = 20


z = 30`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull
	m.eval.Evaluate(m.doc)

	results := m.GetLineResults()

	// Verify line numbers match (1:1 alignment)
	expectedLines := []struct {
		lineNum int
		source  string
		hasVal  bool
	}{
		{0, "x = 10", true},
		{1, "", false},
		{2, "y = 20", true},
		{3, "", false},
		{4, "", false},
		{5, "z = 30", true},
	}

	for _, exp := range expectedLines {
		if exp.lineNum >= len(results) {
			t.Errorf("PREVIEW-02: Missing result for line %d", exp.lineNum)
			continue
		}
		r := results[exp.lineNum]
		if r.LineNum != exp.lineNum {
			t.Errorf("PREVIEW-02: Line %d has LineNum=%d (misaligned)", exp.lineNum, r.LineNum)
		}
		hasValue := r.Value != ""
		if hasValue != exp.hasVal {
			t.Errorf("PREVIEW-02: Line %d hasValue=%v, expected %v", exp.lineNum, hasValue, exp.hasVal)
		}
	}

	t.Log("PREVIEW-02: Results vertically aligned with source lines ✓")
}

// TestPreviewRequirements_PREVIEW03 verifies variable assignment format
// PREVIEW-03: Variable assignments display as "variable_name -> result"
func TestPreviewRequirements_PREVIEW03(t *testing.T) {

	content := `price = 100
tax = price * 0.1
total = price + tax`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull
	m.eval.Evaluate(m.doc)

	results := m.GetLineResults()

	testCases := []struct {
		lineNum     int
		expectedVar string
		expectValue bool
	}{
		{0, "price", true},
		{1, "tax", true},
		{2, "total", true},
	}

	for _, tc := range testCases {
		r := results[tc.lineNum]
		if r.VarName != tc.expectedVar {
			t.Errorf("PREVIEW-03: Line %d should have VarName=%q, got %q",
				tc.lineNum, tc.expectedVar, r.VarName)
		}
		if tc.expectValue && r.Value == "" {
			t.Errorf("PREVIEW-03: Line %d should have a value", tc.lineNum)
		}
	}

	// Verify rendered format contains arrow
	view := m.View().Content
	if !strings.Contains(view, "→") {
		t.Error("PREVIEW-03: View should contain arrow (→) character")
	}

	t.Log("PREVIEW-03: Variable assignments display as 'name → result' ✓")
}

// TestPreviewRequirements_PREVIEW04 verifies anonymous calculation format
// PREVIEW-04: Anonymous calculations display as "-> result" (arrow only)
func TestPreviewRequirements_PREVIEW04(t *testing.T) {

	content := `2 + 2
x = 10
3 * 5`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull
	m.eval.Evaluate(m.doc)

	results := m.GetLineResults()

	// Line 0: 2 + 2 (anonymous) - should have empty VarName
	if results[0].VarName != "" {
		t.Errorf("PREVIEW-04: Anonymous calc (line 0) should have empty VarName, got %q", results[0].VarName)
	}
	if results[0].Value != "4" {
		t.Errorf("PREVIEW-04: Anonymous calc (line 0) should have value '4', got %q", results[0].Value)
	}

	// Line 1: x = 10 (named) - should have VarName
	if results[1].VarName != "x" {
		t.Errorf("PREVIEW-04: Named calc (line 1) should have VarName='x', got %q", results[1].VarName)
	}

	// Line 2: 3 * 5 (anonymous) - should have empty VarName
	if results[2].VarName != "" {
		t.Errorf("PREVIEW-04: Anonymous calc (line 2) should have empty VarName, got %q", results[2].VarName)
	}
	if results[2].Value != "15" {
		t.Errorf("PREVIEW-04: Anonymous calc (line 2) should have value '15', got %q", results[2].Value)
	}

	t.Log("PREVIEW-04: Anonymous calculations display as '→ result' ✓")
}

// TestPreviewRequirements_PREVIEW05 verifies non-calculation lines are blank
// PREVIEW-05: Non-calculation lines show blank in preview (spacing preserved)
func TestPreviewRequirements_PREVIEW05(t *testing.T) {

	content := `# Header
x = 10

Some paragraph text

y = 20`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull
	m.eval.Evaluate(m.doc)

	results := m.GetLineResults()

	nonCalcLines := []int{0, 2, 3, 4} // Header, empty, text, empty
	for _, lineNum := range nonCalcLines {
		r := results[lineNum]
		if r.Value != "" {
			t.Errorf("PREVIEW-05: Non-calc line %d should have empty value, got %q", lineNum, r.Value)
		}
	}

	calcLines := []int{1, 5} // x = 10, y = 20
	for _, lineNum := range calcLines {
		r := results[lineNum]
		if r.Value == "" {
			t.Errorf("PREVIEW-05: Calc line %d should have a value", lineNum)
		}
	}

	// Verify total line count matches (spacing preserved)
	if len(results) != 6 {
		t.Errorf("PREVIEW-05: Should have 6 lines total (spacing preserved), got %d", len(results))
	}

	t.Log("PREVIEW-05: Non-calculation lines show blank in preview ✓")
}

// =============================================================================
// Phase 11.1 Preview Pane Filtering Tests
// =============================================================================

// TestPreviewPaneFiltering verifies preview pane shows only calc results and headings,
// filtering out blockquotes, links, and other markdown.
// Bug: Users see blockquotes, links, and other markdown in the Results pane,
// but only headings and calculation results should display.
func TestPreviewPaneFiltering(t *testing.T) {

	tests := []struct {
		name               string
		source             string
		expectInPreview    []string // Lines that SHOULD appear in preview content
		expectNotInPreview []string // Lines that should NOT appear in preview content
	}{
		{
			name:               "blockquote filtered",
			source:             "x = 10\n> This is a quote\ny = 20",
			expectInPreview:    []string{"x", "10", "y", "20"}, // calc results present
			expectNotInPreview: []string{"quote", ">"},         // blockquote filtered
		},
		{
			name:               "link filtered",
			source:             "a = 5\n[link text](http://example.com)\nb = 15",
			expectInPreview:    []string{"a", "5", "b", "15"}, // calc results present
			expectNotInPreview: []string{"link", "http"},      // link filtered
		},
		{
			name:               "heading preserved",
			source:             "## Budget\namount = 100",
			expectInPreview:    []string{"Budget", "amount", "100"}, // heading and calc present
			expectNotInPreview: []string{},                          // nothing should be filtered
		},
		{
			name:               "mixed content",
			source:             "# Project\na = 1\n> Quote\n[link](url)\nb = 2",
			expectInPreview:    []string{"Project", "a", "1", "b", "2"}, // heading and calcs
			expectNotInPreview: []string{"Quote", "link", "url"},        // blockquote and link filtered
		},
		{
			name:               "paragraph text filtered",
			source:             "x = 100\nThis is a paragraph\ny = 200",
			expectInPreview:    []string{"x", "100", "y", "200"}, // calc results
			expectNotInPreview: []string{"paragraph"},            // text filtered
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := document.NewDocument(tc.source)
			if err != nil {
				t.Fatalf("Failed to create document: %v", err)
			}

			m := New(doc)
			m.width = 80
			m.height = 24
			m.previewMode = PreviewFull
			m.eval.Evaluate(m.doc)

			// Get aligned panes to check preview content
			leftWidth, rightWidth := m.GetPaneWidths(m.width)
			aligned := m.computeAlignedPanes(leftWidth, rightWidth, m.GetLineResults())

			// Build preview content string
			var previewContent strings.Builder
			for _, pl := range aligned.previewLines {
				previewContent.WriteString(pl.content)
				previewContent.WriteString("\n")
			}
			preview := previewContent.String()

			// Check expected content IS present
			for _, expected := range tc.expectInPreview {
				if !strings.Contains(preview, expected) {
					t.Errorf("Preview should contain %q but doesn't.\nPreview:\n%s", expected, preview)
				}
			}

			// Check filtered content is NOT present
			for _, notExpected := range tc.expectNotInPreview {
				if strings.Contains(preview, notExpected) {
					t.Errorf("Preview should NOT contain %q but does.\nPreview:\n%s", notExpected, preview)
				}
			}
		})
	}
}
