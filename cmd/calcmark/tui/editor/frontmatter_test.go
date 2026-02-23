package editor

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cockroachdb/datadriven"
	"github.com/knz/catwalk"
)

// TestFrontmatterRoundTrip verifies that frontmatter is preserved through edit cycles.
// This is the core regression test for the bug: frontmatter was lost after first edit
// because getDocumentContent() and GetLines() only iterated blocks, not frontmatter.
func TestFrontmatterRoundTrip(t *testing.T) {
	content := `---
globals:
  tax_rate: 10%
---
price = 100
total = price * (1 + tax_rate)
`
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Verify frontmatter lines are included in GetLines()
	lines := m.GetLines()
	if len(lines) == 0 {
		t.Fatal("GetLines() returned empty")
	}

	// First line should be "---"
	if lines[0] != "---" {
		t.Errorf("Expected first line to be '---', got %q", lines[0])
	}

	// Should contain frontmatter content
	foundGlobals := false
	foundTaxRate := false
	foundClosingDash := false
	for _, line := range lines {
		if strings.Contains(line, "globals:") {
			foundGlobals = true
		}
		if strings.Contains(line, "tax_rate") {
			foundTaxRate = true
		}
		if line == "---" {
			foundClosingDash = true
		}
	}
	if !foundGlobals {
		t.Error("GetLines() missing 'globals:' line")
	}
	if !foundTaxRate {
		t.Error("GetLines() missing 'tax_rate' line")
	}
	if !foundClosingDash {
		t.Error("GetLines() missing closing '---' line")
	}

	// Verify getDocumentContent() includes frontmatter
	docContent := m.getDocumentContent()
	if !strings.Contains(docContent, "---") {
		t.Error("getDocumentContent() missing frontmatter delimiters")
	}
	if !strings.Contains(docContent, "tax_rate") {
		t.Error("getDocumentContent() missing frontmatter content")
	}

	// Simulate edit cycle: redetectBlockTypes() should preserve frontmatter
	m.redetectBlockTypes()

	// Verify frontmatter still present after redetect
	linesAfter := m.GetLines()
	if linesAfter[0] != "---" {
		t.Errorf("After redetect, first line should be '---', got %q", linesAfter[0])
	}

	docContentAfter := m.getDocumentContent()
	if !strings.Contains(docContentAfter, "tax_rate") {
		t.Error("After redetect, getDocumentContent() lost frontmatter content")
	}
}

// TestFrontmatterLineCount verifies frontmatterLineCount() returns correct values.
func TestFrontmatterLineCount(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name:     "no frontmatter",
			content:  "x = 10\n",
			expected: 0,
		},
		{
			name: "with globals",
			content: `---
globals:
  my_var: 42
---
x = my_var + 1
`,
			expected: 4, // ---, globals:, my_var: 42, ---
		},
		{
			name: "with exchange",
			content: `---
exchange:
  USD_EUR: 0.92
---
price = 100
`,
			expected: 4, // ---, exchange:, USD_EUR: 0.92, ---
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := document.NewDocument(tt.content)
			if err != nil {
				t.Fatalf("Failed to create document: %v", err)
			}
			m := New(doc)
			got := m.frontmatterLineCount()
			if got != tt.expected {
				t.Errorf("frontmatterLineCount() = %d, want %d", got, tt.expected)
			}
		})
	}
}

// TestGetLineResultsFrontmatterPadding verifies GetLineResults() includes
// non-calc padding entries for frontmatter lines.
func TestGetLineResultsFrontmatterPadding(t *testing.T) {
	content := `---
globals:
  my_var: 42
---
x = my_var + 1
`
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	lines := m.GetLines()
	results := m.GetLineResults()

	// Results count should match lines count
	if len(results) != len(lines) {
		t.Errorf("GetLineResults() returned %d results, but GetLines() has %d lines",
			len(results), len(lines))
	}

	// First 4 results should be non-calc (frontmatter padding)
	fmCount := m.frontmatterLineCount()
	for i := 0; i < fmCount && i < len(results); i++ {
		if results[i].IsCalc {
			t.Errorf("Result[%d] should be non-calc (frontmatter padding), got IsCalc=true", i)
		}
		if results[i].Source != lines[i] {
			t.Errorf("Result[%d].Source = %q, want %q", i, results[i].Source, lines[i])
		}
	}

	// After frontmatter, there should be a calc result for "x = my_var + 1"
	foundCalc := false
	for i := fmCount; i < len(results); i++ {
		if results[i].IsCalc && strings.Contains(results[i].Source, "x = my_var") {
			foundCalc = true
			if results[i].Value == "" {
				t.Error("Expected calc result to have a value for 'x = my_var + 1'")
			}
			break
		}
	}
	if !foundCalc {
		t.Error("No calc result found for 'x = my_var + 1' after frontmatter")
	}
}

// TestInsertFrontmatter verifies the Ctrl+F insert frontmatter command.
func TestInsertFrontmatter(t *testing.T) {
	// Start with no frontmatter
	content := `x = 10
y = 20
`
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Verify no frontmatter initially
	if m.doc.GetFrontmatter() != nil {
		t.Fatal("Document should not have frontmatter initially")
	}
	if m.frontmatterLineCount() != 0 {
		t.Fatal("frontmatterLineCount() should be 0 initially")
	}

	// Insert frontmatter
	result, _ := m.insertFrontmatter()
	m = result.(Model)

	// Verify frontmatter exists
	if m.doc.GetFrontmatter() == nil {
		t.Fatal("Document should have frontmatter after insert")
	}

	// Verify cursor is on the exchange rate value line (line 2, 0-indexed)
	if m.cursorLine != 2 {
		t.Errorf("Cursor should be on line 2 (USD_EUR), got %d", m.cursorLine)
	}

	// Verify frontmatter lines in GetLines()
	lines := m.GetLines()
	if lines[0] != "---" {
		t.Errorf("First line should be '---', got %q", lines[0])
	}

	// Verify exchange rate is in template
	fm := m.doc.GetFrontmatter()
	if _, ok := fm.Exchange["USD_EUR"]; !ok {
		t.Error("Frontmatter should contain USD_EUR exchange rate")
	}

	// Verify globals are also in template
	if _, ok := fm.Globals["my_var"]; !ok {
		t.Error("Frontmatter should contain my_var global")
	}

	// Verify status message
	if m.statusMsg != "Frontmatter inserted" {
		t.Errorf("Status should be 'Frontmatter inserted', got %q", m.statusMsg)
	}

	// Verify globals panel is expanded
	if !m.globalsExpanded {
		t.Error("Globals panel should be expanded after insert")
	}

	// Verify modified flag
	if !m.modified {
		t.Error("Document should be marked modified after insert")
	}
}

// TestInsertFrontmatterAlreadyExists verifies Ctrl+F is a no-op when frontmatter exists.
func TestInsertFrontmatterAlreadyExists(t *testing.T) {
	content := `---
globals:
  my_var: 42
---
x = 10
`
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	linesBefore := len(m.GetLines())

	// Try to insert frontmatter (should be no-op)
	result, _ := m.insertFrontmatter()
	m = result.(Model)

	// Verify no change
	linesAfter := len(m.GetLines())
	if linesAfter != linesBefore {
		t.Errorf("Line count changed: %d -> %d (should be unchanged)", linesBefore, linesAfter)
	}

	if m.statusMsg != "Frontmatter already exists" {
		t.Errorf("Status should be 'Frontmatter already exists', got %q", m.statusMsg)
	}
}

// TestFrontmatterErrorDiagnostics verifies malformed YAML captures an error.
func TestFrontmatterErrorDiagnostics(t *testing.T) {
	// Start with valid frontmatter
	content := `---
globals:
  my_var: 42
---
x = my_var + 1
`
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Initially no error
	if m.frontmatterErr != nil {
		t.Fatal("Should have no frontmatter error initially")
	}

	// Verify globals panel state reports no error
	state := m.GetGlobalsPanelState()
	if state.Error != "" {
		t.Errorf("GlobalsPanelState.Error should be empty, got %q", state.Error)
	}
}

// TestUpdateCurrentLineFrontmatterOffset verifies cursor-to-block mapping
// correctly accounts for frontmatter lines.
func TestUpdateCurrentLineFrontmatterOffset(t *testing.T) {
	content := `---
globals:
  tax_rate: 10%
---
price = 100
total = price * (1 + tax_rate)
`
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	fmCount := m.frontmatterLineCount()
	if fmCount == 0 {
		t.Fatal("Expected non-zero frontmatter line count")
	}

	// Move cursor to a calc line (after frontmatter)
	m.cursorLine = fmCount // First calc line ("price = 100")
	m.editBuf = "price = 200"

	// This should update the first calc line, not crash
	m.updateCurrentLine("price = 200")

	// Verify the line was updated
	lines := m.GetLines()
	if lines[fmCount] != "price = 200" {
		t.Errorf("Expected line %d to be 'price = 200', got %q", fmCount, lines[fmCount])
	}
}

// TestDeleteLineFrontmatter verifies deleting a frontmatter line works.
func TestDeleteLineFrontmatter(t *testing.T) {
	content := `---
globals:
  my_var: 42
---
x = my_var + 1
`
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	linesBefore := len(m.GetLines())

	// Position cursor on a frontmatter line (e.g., "  my_var: 42")
	m.cursorLine = 2

	// Delete the line
	m.deleteLine()

	linesAfter := len(m.GetLines())
	if linesAfter >= linesBefore {
		t.Errorf("Expected fewer lines after delete: before=%d, after=%d", linesBefore, linesAfter)
	}
}

// TestEditorCatwalkFrontmatterInsert tests the Ctrl+F insert frontmatter flow via catwalk.
func TestEditorCatwalkFrontmatterInsert(t *testing.T) {
	// Document without frontmatter
	content := `x = 10
y = 20
z = x + y`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "frontmatter_insert") {
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
				model := m.(Model)
				results := model.GetLineResults()
				var buf strings.Builder
				for _, r := range results {
					buf.WriteString(fmt.Sprintf("Line %d (%s): isCalc=%v value=%s error=%q\n",
						r.LineNum, r.Source, r.IsCalc, r.Value, r.Error))
				}
				_, err := out.Write([]byte(buf.String()))
				return err
			}),
			catwalk.WithObserver("frontmatter", func(out io.Writer, m tea.Model) error {
				model := m.(Model)
				var buf strings.Builder
				fm := model.doc.GetFrontmatter()
				buf.WriteString(fmt.Sprintf("hasFrontmatter=%v\n", fm != nil))
				buf.WriteString(fmt.Sprintf("frontmatterLineCount=%d\n", model.frontmatterLineCount()))
				buf.WriteString(fmt.Sprintf("totalLines=%d\n", model.TotalLines()))
				buf.WriteString(fmt.Sprintf("statusMsg=%q\n", model.statusMsg))
				if fm != nil {
					buf.WriteString(fmt.Sprintf("globalsCount=%d\n", len(fm.Globals)))
				}
				_, err := out.Write([]byte(buf.String()))
				return err
			}),
		)
	})
}

// TestUpdateCurrentLineFrontmatterPersists verifies that editing a frontmatter line
// via updateCurrentLine actually persists the change to the document.
// This is the regression test for: edits to frontmatter lines revert on navigation.
func TestUpdateCurrentLineFrontmatterPersists(t *testing.T) {
	content := `---
globals:
  my_var: 42
---
x = my_var + 1
`
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Position cursor on the "  my_var: 42" line (line 2, 0-indexed)
	m.cursorLine = 2
	m.editBuf = "  growth_rate: 5%"

	// Update the frontmatter line — this was previously a no-op
	m.updateCurrentLine(m.editBuf)

	// Verify the line was persisted to the document
	lines := m.GetLines()
	if lines[2] != "  growth_rate: 5%" {
		t.Errorf("Expected line 2 to be '  growth_rate: 5%%', got %q", lines[2])
	}

	// Verify frontmatter was re-parsed with new variable
	fm := m.doc.GetFrontmatter()
	if fm == nil {
		t.Fatal("Frontmatter should still exist after edit")
	}
	if _, ok := fm.Globals["growth_rate"]; !ok {
		t.Error("Frontmatter should contain 'growth_rate' after edit")
	}
	if _, ok := fm.Globals["my_var"]; ok {
		t.Error("Frontmatter should NOT contain 'my_var' after edit (was renamed)")
	}
}

// TestFrontmatterEditSurvivedNavigation verifies the full user flow:
// insert frontmatter, edit it, navigate away, and confirm the edit persisted.
func TestFrontmatterEditSurvivesNavigation(t *testing.T) {
	content := "x = 10\n"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// 1. Insert frontmatter via Ctrl+F
	result, _ := m.insertFrontmatter()
	m = result.(Model)

	// Cursor should be on line 2 ("  USD_EUR: 0.92" — exchange rate line)
	if m.cursorLine != 2 {
		t.Fatalf("Expected cursor on line 2, got %d", m.cursorLine)
	}

	// 2. Simulate editing: load line, change content (replace exchange rate)
	m.loadCurrentLineIntoEditBuffer()
	m.editBuf = "  EUR_GBP: 0.86"

	// 3. Navigate down (triggers saveCurrentLineAndMoveTo)
	resultDown, _ := m.handleDownKey()
	m = resultDown.(Model)

	// 4. Verify cursor moved down
	if m.cursorLine != 3 {
		t.Errorf("Expected cursor on line 3 after down, got %d", m.cursorLine)
	}

	// 5. Verify the frontmatter edit persisted (not reverted)
	lines := m.GetLines()
	foundEurGbp := false
	for _, line := range lines {
		if strings.Contains(line, "EUR_GBP") {
			foundEurGbp = true
		}
		if strings.Contains(line, "USD_EUR") {
			t.Error("Frontmatter should NOT contain 'USD_EUR' — edit was reverted!")
		}
	}
	if !foundEurGbp {
		t.Error("Frontmatter should contain 'EUR_GBP' after edit + navigation")
	}
}

// TestFrontmatterEnterKeyInsertsLine verifies that pressing Enter while on a
// frontmatter line inserts a new line that persists through navigation.
// This was broken because Serialize() normalized YAML, destroying structural edits.
func TestFrontmatterEnterKeyInsertsLine(t *testing.T) {
	content := `---
globals:
  my_var: 42
---
x = my_var + 1
`
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	linesBefore := m.TotalLines()
	if linesBefore != 5 {
		t.Fatalf("Expected 5 lines initially, got %d", linesBefore)
	}

	// Navigate to the "  my_var: 42" line (line 2) and go to end
	m.cursorLine = 2
	m.loadCurrentLineIntoEditBuffer()
	m.cursorCol = runeLen(m.editBuf) // End of line

	// Press Enter — should insert a new line below
	result, _ := m.handleEnterKey()
	m = result.(Model)

	linesAfter := m.TotalLines()
	if linesAfter != linesBefore+1 {
		t.Errorf("Expected %d lines after Enter, got %d", linesBefore+1, linesAfter)
	}

	// Cursor should be on the new line (line 3)
	if m.cursorLine != 3 {
		t.Errorf("Expected cursor on line 3, got %d", m.cursorLine)
	}

	// Navigate down to exit frontmatter region and verify the line persisted
	result, _ = m.handleDownKey()
	m = result.(Model)
	result, _ = m.handleDownKey()
	m = result.(Model)

	// Total lines should still be linesBefore+1
	if m.TotalLines() != linesBefore+1 {
		t.Errorf("After navigation, expected %d lines, got %d (Enter line was lost!)",
			linesBefore+1, m.TotalLines())
	}

	// Verify the frontmatter structure: line 2 should still be "  my_var: 42"
	lines := m.GetLines()
	if lines[2] != "  my_var: 42" {
		t.Errorf("Line 2 should be '  my_var: 42', got %q", lines[2])
	}
	// Line 3 should be the empty line we inserted
	if lines[3] != "" {
		t.Errorf("Line 3 should be empty (inserted by Enter), got %q", lines[3])
	}
	// Line 4 should be the closing delimiter
	if lines[4] != "---" {
		t.Errorf("Line 4 should be '---', got %q", lines[4])
	}
}

// TestFrontmatterEnterAndTypeNewVariable verifies the full user flow:
// position on my_var line, Enter to create new line, type new variable, navigate away.
func TestFrontmatterEnterAndTypeNewVariable(t *testing.T) {
	content := `---
globals:
  my_var: 42
---
x = my_var + 1
`
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Navigate to "  my_var: 42" (line 2), go to end, press Enter
	m.cursorLine = 2
	m.loadCurrentLineIntoEditBuffer()
	m.cursorCol = runeLen(m.editBuf)

	result, _ := m.handleEnterKey()
	m = result.(Model)

	// Now on line 3 (empty line). Type a new variable.
	m.editBuf = "  growth_rate: 5%"
	m.cursorCol = 17

	// Navigate down to persist the edit
	result, _ = m.handleDownKey()
	m = result.(Model)

	// Verify both variables exist in frontmatter
	fm := m.doc.GetFrontmatter()
	if fm == nil {
		t.Fatal("Frontmatter should exist")
	}
	if _, ok := fm.Globals["my_var"]; !ok {
		t.Error("Frontmatter should contain 'my_var'")
	}
	if _, ok := fm.Globals["growth_rate"]; !ok {
		t.Error("Frontmatter should contain 'growth_rate'")
	}
}

// TestEditorCatwalkFrontmatterEditing tests editing within frontmatter lines via catwalk.
func TestEditorCatwalkFrontmatterEditing(t *testing.T) {
	// Document with frontmatter
	content := `---
globals:
  my_var: 42
---
x = my_var + 1`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "frontmatter_editing") {
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
				model := m.(Model)
				results := model.GetLineResults()
				var buf strings.Builder
				for _, r := range results {
					buf.WriteString(fmt.Sprintf("Line %d (%s): isCalc=%v value=%s error=%q\n",
						r.LineNum, r.Source, r.IsCalc, r.Value, r.Error))
				}
				_, err := out.Write([]byte(buf.String()))
				return err
			}),
			catwalk.WithObserver("frontmatter", func(out io.Writer, m tea.Model) error {
				model := m.(Model)
				var buf strings.Builder
				fm := model.doc.GetFrontmatter()
				buf.WriteString(fmt.Sprintf("hasFrontmatter=%v\n", fm != nil))
				buf.WriteString(fmt.Sprintf("frontmatterLineCount=%d\n", model.frontmatterLineCount()))
				buf.WriteString(fmt.Sprintf("frontmatterErr=%v\n", model.frontmatterErr))
				if fm != nil {
					for _, name := range fm.GlobalKeys() {
						buf.WriteString(fmt.Sprintf("global: %s=%s\n", name, fm.Globals[name]))
					}
				}
				_, err := out.Write([]byte(buf.String()))
				return err
			}),
		)
	})
}

// TestEditorCatwalkFrontmatterGlobalsAlignment tests that the Globals panel renders
// inline with frontmatter lines in the preview pane (not as a fixed header).
func TestEditorCatwalkFrontmatterGlobalsAlignment(t *testing.T) {
	// Document with frontmatter containing a global variable
	content := `---
globals:
  my_var: 42
---
x = my_var + 1`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if !strings.HasSuffix(path, "frontmatter_globals_alignment") {
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
			catwalk.WithObserver("alignment", func(out io.Writer, m tea.Model) error {
				model := m.(Model)
				leftWidth, rightWidth := model.GetPaneWidths(model.width)
				aligned := model.computeAlignedPanes(leftWidth, rightWidth)

				var buf strings.Builder
				buf.WriteString("Source and Preview Alignment:\n")
				buf.WriteString(fmt.Sprintf("Source lines: %d, Preview lines: %d\n",
					len(aligned.sourceLines), len(aligned.previewLines)))

				maxLines := max(len(aligned.sourceLines), len(aligned.previewLines))

				for i := range maxLines {
					var srcContent string
					var srcLineNum int
					var srcIsPadding bool
					var prvIsFrontmatter bool

					if i < len(aligned.sourceLines) {
						src := aligned.sourceLines[i]
						srcContent = src.content
						srcLineNum = src.lineNum
						srcIsPadding = src.isPadding
					}

					var prvContent string
					var prvLineNum int
					if i < len(aligned.previewLines) {
						prv := aligned.previewLines[i]
						prvContent = prv.content
						prvLineNum = prv.sourceLineNum
						prvIsFrontmatter = prv.isFrontmatter
					}

					// Truncate for readability
					if len(srcContent) > 30 {
						srcContent = srcContent[:30] + "..."
					}
					if len(prvContent) > 30 {
						prvContent = prvContent[:30] + "..."
					}

					buf.WriteString(fmt.Sprintf("[%d] SRC(ln=%d pad=%v): %-35s | PRV(ln=%d fm=%v): %q\n",
						i, srcLineNum, srcIsPadding, fmt.Sprintf("%q", srcContent),
						prvLineNum, prvIsFrontmatter, prvContent))
				}

				_, err := out.Write([]byte(buf.String()))
				return err
			}),
		)
	})
}
