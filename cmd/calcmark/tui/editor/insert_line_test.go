package editor

// insert_line_test.go — Line insertion, Enter key, O key, compression, trailing lines.

import (
	"fmt"
	"slices"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

func TestInsertLineBelow_CursorPosition(t *testing.T) {
	// When pressing 'o' (insert below), cursor should move to the new line
	content := `line0 = 0
line1 = 1
line2 = 2`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 80
	m.height = 24

	// Start on line 1
	m.cursorLine = 1

	// Record state before
	linesBefore := m.TotalLines()
	t.Logf("Before insert: cursorLine=%d, totalLines=%d", m.cursorLine, linesBefore)

	// Insert line below (simulates 'o' key, first part)
	m.insertLineBelow()

	// After insert:
	// - Total lines should increase by 1
	// - Cursor should be on the NEW line (line 2, which is now empty)
	// - The old line 2 content should now be at line 3

	linesAfter := m.TotalLines()
	t.Logf("After insert: cursorLine=%d, totalLines=%d", m.cursorLine, linesAfter)

	if linesAfter != linesBefore+1 {
		t.Errorf("Total lines should increase by 1: was %d, now %d", linesBefore, linesAfter)
	}

	// Cursor should be on line 2 (the newly inserted line)
	expectedCursorLine := 2
	if m.cursorLine != expectedCursorLine {
		t.Errorf("Cursor should be on line %d (new line), but is on %d",
			expectedCursorLine, m.cursorLine)
	}

	// Verify line content
	lines := m.GetLines()
	t.Logf("Lines after insert: %v", lines)

	// The new line should be empty
	if lines[2] != "" {
		t.Errorf("New line at index 2 should be empty, got %q", lines[2])
	}
}

func TestInsertLineBelow_ThenEnterEditMode(t *testing.T) {
	// Full 'o' key simulation: insert + enter edit mode
	content := `x = 10
y = 20`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 80
	m.height = 24

	// Start on line 0
	m.cursorLine = 0
	t.Logf("Initial: cursorLine=%d, totalLines=%d", m.cursorLine, m.TotalLines())

	// Simulate 'o' key: insert below then enter edit mode
	m.insertLineBelow()
	t.Logf("After insertLineBelow: cursorLine=%d, totalLines=%d", m.cursorLine, m.TotalLines())

	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()
	t.Logf("After enterEditMode: cursorLine=%d, mode=%v, editBuf=%q",
		m.cursorLine, m.mode, m.editBuf)

	// Should be in edit mode
	if m.mode != StateDefault {
		t.Fatalf("Should be in edit mode, got %v", m.mode)
	}

	// Cursor should be on line 1 (the new line)
	if m.cursorLine != 1 {
		t.Errorf("Cursor should be on line 1, got %d", m.cursorLine)
	}

	// Edit buffer should be empty (new line)
	if m.editBuf != "" {
		t.Errorf("Edit buffer should be empty for new line, got %q", m.editBuf)
	}
}

func TestInsertLineBelow_AtEndOfDocument(t *testing.T) {
	// Insert at end of document
	content := `x = 10`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 80
	m.height = 24

	// Start on last line
	m.cursorLine = 0
	totalBefore := m.TotalLines()

	m.insertLineBelow()

	// Should have one more line
	if m.TotalLines() != totalBefore+1 {
		t.Errorf("Should have %d lines, got %d", totalBefore+1, m.TotalLines())
	}

	// Cursor should be on new line
	if m.cursorLine != 1 {
		t.Errorf("Cursor should be on line 1, got %d", m.cursorLine)
	}
}

func TestInsertLine_VisualAlignment(t *testing.T) {
	// After inserting a line, visual alignment should be correct
	content := `x = 10
y = 20`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 80
	m.height = 24

	// Insert line below line 0
	m.cursorLine = 0
	m.insertLineBelow()
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()

	// Compute visual alignment
	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	t.Logf("After insert, cursorLine=%d", m.cursorLine)
	t.Logf("Source lines: %d, Preview lines: %d",
		len(aligned.sourceLines), len(aligned.previewLines))
	t.Logf("sourceToVisual: %v", aligned.sourceToVisual)

	for i, sl := range aligned.sourceLines {
		t.Logf("  Source[%d]: srcIdx=%d cursor=%v content=%q",
			i, sl.sourceLineIdx, sl.isCursorLine, sl.content)
	}

	// Critical: counts must match
	if len(aligned.sourceLines) != len(aligned.previewLines) {
		t.Errorf("Alignment broken: source=%d, preview=%d",
			len(aligned.sourceLines), len(aligned.previewLines))
	}

	// The cursor line should be marked correctly
	cursorMarked := false
	for i, sl := range aligned.sourceLines {
		if sl.isCursorLine {
			cursorMarked = true
			if sl.sourceLineIdx != m.cursorLine {
				t.Errorf("Cursor line at visual %d has sourceLineIdx=%d, want %d",
					i, sl.sourceLineIdx, m.cursorLine)
			}
		}
	}

	if !cursorMarked {
		t.Error("No line marked as cursor line")
	}

	// Visual line for cursor should match the mapping
	expectedVisual, ok := aligned.sourceToVisual[m.cursorLine]
	if !ok {
		t.Errorf("No visual mapping for cursor line %d", m.cursorLine)
	} else {
		t.Logf("Cursor line %d maps to visual line %d", m.cursorLine, expectedVisual)
	}
}

func TestInsertLine_ThenTypeAndExit(t *testing.T) {
	// Full workflow: insert, type, exit edit mode
	content := `x = 10`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 80
	m.height = 24

	// Simulate 'o' then typing "- bullet"
	m.cursorLine = 0
	m.insertLineBelow()
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()

	// Type content
	m.editBuf = "- bullet"
	m.cursorCol = len(m.editBuf)

	// Exit edit mode
	m.saveCurrentLine(true)

	// Check final state
	t.Logf("After exit: cursorLine=%d, mode=%v", m.cursorLine, m.mode)

	lines := m.GetLines()
	t.Logf("Final lines: %v", lines)

	// Should have 2 lines
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(lines))
	}

	// Line 0 should be "x = 10"
	if lines[0] != "x = 10" {
		t.Errorf("Line 0 should be 'x = 10', got %q", lines[0])
	}

	// Line 1 should be "- bullet"
	if lines[1] != "- bullet" {
		t.Errorf("Line 1 should be '- bullet', got %q", lines[1])
	}

	// Cursor should still be on line 1
	if m.cursorLine != 1 {
		t.Errorf("Cursor should be on line 1 after exit, got %d", m.cursorLine)
	}

	// Visual alignment should be correct
	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	if len(aligned.sourceLines) != len(aligned.previewLines) {
		t.Errorf("Alignment broken after edit: source=%d, preview=%d",
			len(aligned.sourceLines), len(aligned.previewLines))
	}
}

func TestInsertLine_NavigationAfterInsert(t *testing.T) {
	// Simulate opening compression.cm and pressing 'o' to insert
	// Bug: After insert, navigation with 'j' highlights wrong line

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
	m := New(doc)
	m.width = 120
	m.height = 24
	m.previewMode = PreviewFull

	totalLinesBefore := m.TotalLines()
	t.Logf("Lines before insert: %d", totalLinesBefore)

	// Navigate to line 5 (zstd_compressed)
	m.cursorLine = 5
	cursorBefore := m.cursorLine

	// Get visual state before insert
	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	alignedBefore := m.computeAlignedPanes(leftWidth, rightWidth)
	visualBefore, _ := alignedBefore.sourceToVisual[m.cursorLine]
	t.Logf("Before insert: cursorLine=%d, visual=%d", m.cursorLine, visualBefore)

	// Press 'o' to insert line below
	m.insertLineBelow()

	totalLinesAfter := m.TotalLines()
	t.Logf("Lines after insert: %d", totalLinesAfter)

	// Verify line count increased
	if totalLinesAfter != totalLinesBefore+1 {
		t.Errorf("Line count should increase by 1: was %d, now %d",
			totalLinesBefore, totalLinesAfter)
	}

	// Cursor should be on the NEW line (cursorBefore + 1)
	expectedCursor := cursorBefore + 1
	if m.cursorLine != expectedCursor {
		t.Errorf("Cursor should be on line %d after insert, got %d",
			expectedCursor, m.cursorLine)
	}

	// Get visual state after insert
	alignedAfter := m.computeAlignedPanes(leftWidth, rightWidth)
	visualAfter, ok := alignedAfter.sourceToVisual[m.cursorLine]
	if !ok {
		t.Errorf("No visual mapping for cursor line %d after insert", m.cursorLine)
	}
	t.Logf("After insert: cursorLine=%d, visual=%d", m.cursorLine, visualAfter)

	// KEY TEST: Visual mapping should be consistent
	// The cursor's visual line should be the visual index for that source line
	// Find cursor in source lines
	cursorFoundAtVisual := -1
	for i, sl := range alignedAfter.sourceLines {
		if sl.isCursorLine {
			cursorFoundAtVisual = i
			break
		}
	}

	if cursorFoundAtVisual == -1 {
		t.Error("Cursor line not found in visual structure after insert")
	} else if cursorFoundAtVisual != visualAfter {
		t.Errorf("Cursor highlight mismatch: cursor found at visual %d, but mapping says %d",
			cursorFoundAtVisual, visualAfter)
	}

	// Simulate navigation with 'j' (move down)
	m.cursorLine++
	if m.cursorLine >= m.TotalLines() {
		m.cursorLine = m.TotalLines() - 1
	}

	alignedNav := m.computeAlignedPanes(leftWidth, rightWidth)
	visualNav, ok := alignedNav.sourceToVisual[m.cursorLine]
	if !ok {
		t.Errorf("No visual mapping for cursor line %d after navigation", m.cursorLine)
	}

	// Find where cursor is highlighted
	navCursorVisual := -1
	for i, sl := range alignedNav.sourceLines {
		if sl.isCursorLine {
			navCursorVisual = i
			break
		}
	}

	t.Logf("After j navigation: cursorLine=%d, visual=%d, cursorFound=%d",
		m.cursorLine, visualNav, navCursorVisual)

	if navCursorVisual != visualNav {
		t.Errorf("Navigation bug: cursor at visual %d, mapping says %d",
			navCursorVisual, visualNav)
	}
}

func TestInsertLineAtEnd_ThenType(t *testing.T) {
	// Bug: Insert at end of document, type 'i' to edit, type bullet
	// Results in misalignment

	content := `# Header
x = 10
y = 20`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)
	m.width = 100
	m.height = 24
	m.previewMode = PreviewFull

	// Navigate to last line
	m.cursorLine = m.TotalLines() - 1
	t.Logf("Starting at line %d (last line)", m.cursorLine)

	// Press 'o' to insert below (at end of document)
	m.insertLineBelow()

	// Cursor should be on new empty line at the end
	expectedLine := 3 // 0-indexed: was 3 lines (0,1,2), now 4 (0,1,2,3)
	if m.cursorLine != expectedLine {
		t.Errorf("Cursor should be on line %d, got %d", expectedLine, m.cursorLine)
	}

	// Total lines should be 4 now
	if m.TotalLines() != 4 {
		t.Errorf("Should have 4 lines, got %d", m.TotalLines())
	}

	// The new line should be empty
	lines := m.GetLines()
	if lines[m.cursorLine] != "" {
		t.Errorf("New line should be empty, got %q", lines[m.cursorLine])
	}

	// Press 'i' to enter edit mode
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()
	if m.mode != StateDefault {
		t.Fatalf("Should be in edit mode, got %v", m.mode)
	}

	// Edit buffer should be empty
	if m.editBuf != "" {
		t.Errorf("Edit buffer should be empty, got %q", m.editBuf)
	}

	// Type a bullet "- Test"
	m.editBuf = "- Test"
	m.cursorCol = len(m.editBuf)

	// Get visual state during edit
	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	alignedEdit := m.computeAlignedPanes(leftWidth, rightWidth)

	t.Logf("During edit: %d source lines, %d preview lines",
		len(alignedEdit.sourceLines), len(alignedEdit.previewLines))

	// Source and preview should still be aligned
	if len(alignedEdit.sourceLines) != len(alignedEdit.previewLines) {
		t.Errorf("Alignment broken during edit: source=%d, preview=%d",
			len(alignedEdit.sourceLines), len(alignedEdit.previewLines))
	}

	// The cursor line visual should exist
	_, ok := alignedEdit.sourceToVisual[m.cursorLine]
	if !ok {
		t.Errorf("No visual mapping for cursor line %d during edit", m.cursorLine)
	}

	// Exit edit mode
	m.saveCurrentLine(true)

	// Verify the bullet was saved
	lines = m.GetLines()
	if lines[m.cursorLine] != "- Test" {
		t.Errorf("Bullet should be saved, got %q", lines[m.cursorLine])
	}

	// Visual alignment should be correct after exit
	alignedAfter := m.computeAlignedPanes(leftWidth, rightWidth)
	if len(alignedAfter.sourceLines) != len(alignedAfter.previewLines) {
		t.Errorf("Alignment broken after exit: source=%d, preview=%d",
			len(alignedAfter.sourceLines), len(alignedAfter.previewLines))
	}

	// Verify sourceToVisual has entry for every source line
	for i := 0; i < m.TotalLines(); i++ {
		if _, ok := alignedAfter.sourceToVisual[i]; !ok {
			t.Errorf("Missing sourceToVisual entry for line %d", i)
		}
	}
}

func TestInsertLine_VisualMappingConsistency(t *testing.T) {
	// Test that sourceToVisual stays consistent during insert operations
	// This is the core of the navigation bug

	content := `line0 = 0
line1 = 1
line2 = 2
line3 = 3
line4 = 4`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 100
	m.height = 24
	m.previewMode = PreviewFull

	leftWidth, rightWidth := m.GetPaneWidths(m.width)

	// For each line, insert below and verify mapping stays consistent
	for insertAt := range 3 {
		t.Run(fmt.Sprintf("InsertAt%d", insertAt), func(t *testing.T) {
			// Reset
			doc, _ := document.NewDocument(content)
			m := New(doc)
			m.width = 100
			m.height = 24
			m.previewMode = PreviewFull

			linesBefore := m.TotalLines()

			// Position cursor and insert
			m.cursorLine = insertAt
			m.insertLineBelow()

			linesAfter := m.TotalLines()
			if linesAfter != linesBefore+1 {
				t.Errorf("Line count wrong: was %d, now %d", linesBefore, linesAfter)
			}

			// Get aligned panes
			aligned := m.computeAlignedPanes(leftWidth, rightWidth)

			// CRITICAL: sourceToVisual should have entries for ALL source lines
			for srcLine := 0; srcLine < m.TotalLines(); srcLine++ {
				visualIdx, ok := aligned.sourceToVisual[srcLine]
				if !ok {
					t.Errorf("sourceToVisual missing entry for source line %d", srcLine)
					continue
				}

				// The visual index should be valid
				if visualIdx < 0 || visualIdx >= len(aligned.sourceLines) {
					t.Errorf("sourceToVisual[%d] = %d is out of bounds (0-%d)",
						srcLine, visualIdx, len(aligned.sourceLines)-1)
					continue
				}

				// The source line at that visual index should match
				sl := aligned.sourceLines[visualIdx]
				if sl.sourceLineIdx != srcLine {
					t.Errorf("sourceToVisual[%d] = visual %d, but that visual has sourceLineIdx=%d",
						srcLine, visualIdx, sl.sourceLineIdx)
				}
			}

			// Verify cursor is at correct position
			expectedCursor := insertAt + 1
			if m.cursorLine != expectedCursor {
				t.Errorf("Cursor should be at %d, got %d", expectedCursor, m.cursorLine)
			}

			// Verify cursor highlight is on correct visual line
			cursorVisualIdx, _ := aligned.sourceToVisual[m.cursorLine]
			foundCursorAt := -1
			for i, sl := range aligned.sourceLines {
				if sl.isCursorLine {
					foundCursorAt = i
					break
				}
			}

			if foundCursorAt != cursorVisualIdx {
				t.Errorf("Cursor highlight at visual %d, but mapping says %d",
					foundCursorAt, cursorVisualIdx)
			}
		})
	}
}

func TestEnterKey_InEditMode(t *testing.T) {
	// Test that Enter key works properly in edit mode
	// User reported: "Try to type ENTER - doesn't work"

	content := `# Header
x = 10`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 100
	m.height = 24
	m.previewMode = PreviewFull

	// Navigate to end and insert new line
	m.cursorLine = m.TotalLines() - 1
	m.insertLineBelow()

	// Enter edit mode on new line
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()
	if m.mode != StateDefault {
		t.Fatalf("Should be in edit mode")
	}

	// Type some content
	m.editBuf = "- bullet"
	m.cursorCol = len(m.editBuf)

	// Simulate Enter key - should exit and save
	// (This depends on how Enter is handled in edit mode)
	m.saveCurrentLine(true)

	// Verify saved
	lines := m.GetLines()
	if !slices.Contains(lines, "- bullet") {
		t.Errorf("Bullet not saved. Lines: %v", lines)
	}
}

func TestCompressionFile_InsertLine(t *testing.T) {
	// Exact test case from compression.cm bug report
	// Navigate with j, press o, navigation becomes broken

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

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 80 // Narrower width that might cause wrapping
	m.height = 24
	m.previewMode = PreviewFull

	// SIMULATE PRESSING 'o' KEY: insert line below then enter edit mode

	leftWidth, rightWidth := m.GetPaneWidths(m.width)

	// Get initial aligned state
	alignedInitial := m.computeAlignedPanes(leftWidth, rightWidth)
	t.Logf("Initial: %d source lines, %d source visual lines",
		m.TotalLines(), len(alignedInitial.sourceLines))

	// Log the visual structure to see if there's any wrapping/padding
	for i, sl := range alignedInitial.sourceLines {
		t.Logf("  Visual[%d]: srcIdx=%d, cursor=%v, wrap=%v, pad=%v, lineNum=%d, content=%q",
			i, sl.sourceLineIdx, sl.isCursorLine, sl.isWrapped, sl.isPadding, sl.lineNum, truncate(sl.content, 40))
	}

	// Navigate to line 8 (snappy_compressed, 0-indexed)
	m.cursorLine = 8
	t.Logf("\nCursor at line 8 (snappy_compressed)")

	// Check that cursor highlight is correct BEFORE insert
	alignedBefore := m.computeAlignedPanes(leftWidth, rightWidth)
	cursorVisualBefore := -1
	for i, sl := range alignedBefore.sourceLines {
		if sl.isCursorLine {
			cursorVisualBefore = i
			break
		}
	}

	visualIdxBefore, _ := alignedBefore.sourceToVisual[m.cursorLine]
	t.Logf("Before insert: cursor source=%d, cursor visual=%d, mapping says=%d",
		m.cursorLine, cursorVisualBefore, visualIdxBefore)

	if cursorVisualBefore != visualIdxBefore {
		t.Errorf("BEFORE INSERT: cursor highlight at visual %d, but mapping says %d",
			cursorVisualBefore, visualIdxBefore)
	}

	// Press 'o' to insert line below
	m.insertLineBelow()
	t.Logf("After insert: cursor now at line %d", m.cursorLine)

	// Check cursor highlight AFTER insert
	alignedAfter := m.computeAlignedPanes(leftWidth, rightWidth)
	cursorVisualAfter := -1
	for i, sl := range alignedAfter.sourceLines {
		if sl.isCursorLine {
			cursorVisualAfter = i
			break
		}
	}

	visualIdxAfter, _ := alignedAfter.sourceToVisual[m.cursorLine]
	t.Logf("After insert: cursor source=%d, cursor visual=%d, mapping says=%d",
		m.cursorLine, cursorVisualAfter, visualIdxAfter)

	if cursorVisualAfter != visualIdxAfter {
		t.Errorf("AFTER INSERT: cursor highlight at visual %d, but mapping says %d",
			cursorVisualAfter, visualIdxAfter)
	}

	// Now simulate 'j' navigation (move down)
	m.cursorLine++
	t.Logf("After j: cursor now at line %d", m.cursorLine)

	alignedNav := m.computeAlignedPanes(leftWidth, rightWidth)
	cursorVisualNav := -1
	for i, sl := range alignedNav.sourceLines {
		if sl.isCursorLine {
			cursorVisualNav = i
			break
		}
	}

	visualIdxNav, _ := alignedNav.sourceToVisual[m.cursorLine]
	t.Logf("After navigation: cursor source=%d, cursor visual=%d, mapping says=%d",
		m.cursorLine, cursorVisualNav, visualIdxNav)

	if cursorVisualNav != visualIdxNav {
		t.Errorf("AFTER NAVIGATION: cursor highlight at visual %d, but mapping says %d",
			cursorVisualNav, visualIdxNav)
	}
}

func TestOKey_FullSequence(t *testing.T) {
	// Test the full 'o' key sequence: insert line + enter edit mode
	// This is what causes the bug

	content := `# Header
line1 = 1
line2 = 2
line3 = 3`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	t.Logf("Initial lines: %d", m.TotalLines())
	for i, line := range m.GetLines() {
		t.Logf("  [%d] %q", i, line)
	}

	// Position cursor on line 1 (line1 = 1)
	m.cursorLine = 1
	t.Logf("Cursor at line %d before 'o'", m.cursorLine)

	// Simulate pressing 'o': insertLineBelow + enterEditMode
	m.insertLineBelow()
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()

	t.Logf("After 'o': cursor=%d, mode=%v, editBuf=%q",
		m.cursorLine, m.mode, m.editBuf)

	// Cursor should be on the new line (line 2)
	if m.cursorLine != 2 {
		t.Errorf("Cursor should be at line 2 after 'o', got %d", m.cursorLine)
	}

	// Edit buffer should be empty (new line)
	if m.editBuf != "" {
		t.Errorf("Edit buffer should be empty, got %q", m.editBuf)
	}

	// Verify visual structure
	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	t.Logf("Visual structure:")
	for i, sl := range aligned.sourceLines {
		marker := ""
		if sl.isCursorLine {
			marker = " <-- CURSOR"
		}
		t.Logf("  Visual[%d]: srcIdx=%d, content=%q%s",
			i, sl.sourceLineIdx, truncate(sl.content, 30), marker)
	}

	// Find cursor visual line
	cursorVisual := -1
	for i, sl := range aligned.sourceLines {
		if sl.isCursorLine {
			cursorVisual = i
			break
		}
	}

	visualFromMap, _ := aligned.sourceToVisual[m.cursorLine]
	t.Logf("Cursor: source=%d, visualFound=%d, visualMap=%d",
		m.cursorLine, cursorVisual, visualFromMap)

	if cursorVisual != visualFromMap {
		t.Errorf("Cursor mismatch: found at visual %d, mapping says %d",
			cursorVisual, visualFromMap)
	}

	// Exit edit mode (press Escape)
	m.saveCurrentLine(false)
	t.Logf("After Escape: mode=%v", m.mode)

	// Navigate with 'j'
	m.cursorLine++
	t.Logf("After 'j': cursor=%d", m.cursorLine)

	// Check visual alignment after navigation
	alignedNav := m.computeAlignedPanes(leftWidth, rightWidth)
	cursorVisualNav := -1
	for i, sl := range alignedNav.sourceLines {
		if sl.isCursorLine {
			cursorVisualNav = i
			break
		}
	}

	visualFromMapNav, _ := alignedNav.sourceToVisual[m.cursorLine]
	t.Logf("After nav: cursor source=%d, visualFound=%d, visualMap=%d",
		m.cursorLine, cursorVisualNav, visualFromMapNav)

	if cursorVisualNav != visualFromMapNav {
		t.Errorf("Nav cursor mismatch: found at visual %d, mapping says %d",
			cursorVisualNav, visualFromMapNav)
	}
}

// truncate truncates a string to max length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func TestInsertLine_GetLineResultsConsistency(t *testing.T) {
	// Test that GetLineResults returns correct LineNum values after insert
	// This is critical for cursor highlight to work correctly

	content := `# Header
x = 10
y = 20`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 100
	m.height = 24
	m.previewMode = PreviewFull

	// Get initial results
	resultsBefore := m.GetLineResults()
	t.Logf("Before insert: %d results", len(resultsBefore))
	for i, r := range resultsBefore {
		t.Logf("  [%d] LineNum=%d, Source=%q, BlockID=%s", i, r.LineNum, r.Source, r.BlockID)
	}

	// LineNum should match index
	for i, r := range resultsBefore {
		if r.LineNum != i {
			t.Errorf("Before insert: result[%d].LineNum = %d, want %d", i, r.LineNum, i)
		}
	}

	// Insert line after first line (# Header)
	m.cursorLine = 0
	m.insertLineBelow()

	// Get results after insert
	resultsAfter := m.GetLineResults()
	t.Logf("After insert: %d results (cursor at %d)", len(resultsAfter), m.cursorLine)
	for i, r := range resultsAfter {
		t.Logf("  [%d] LineNum=%d, Source=%q, BlockID=%s", i, r.LineNum, r.Source, r.BlockID)
	}

	// LineNum should still match index after insert
	for i, r := range resultsAfter {
		if r.LineNum != i {
			t.Errorf("After insert: result[%d].LineNum = %d, want %d", i, r.LineNum, i)
		}
	}

	// Should have 4 results now
	if len(resultsAfter) != 4 {
		t.Errorf("Expected 4 results after insert, got %d", len(resultsAfter))
	}

	// Line 1 (the inserted line) should be empty
	if resultsAfter[1].Source != "" {
		t.Errorf("Inserted line should be empty, got %q", resultsAfter[1].Source)
	}
}

func TestWrappedLine_InsertBelow(t *testing.T) {
	// Test inserting below a line that wraps (compression.cm scenario)
	// The wrapped lines should not affect the insert position

	content := `short = 1
this_is_a_very_long_line_that_will_definitely_wrap_when_displayed = 999
another = 2`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 50 // Narrow enough to cause wrapping
	m.height = 24
	m.previewMode = PreviewFull

	leftWidth, rightWidth := m.GetPaneWidths(m.width)

	// Get initial visual state
	alignedBefore := m.computeAlignedPanes(leftWidth, rightWidth)
	t.Logf("Before insert: %d source visual lines", len(alignedBefore.sourceLines))
	for i, sl := range alignedBefore.sourceLines {
		t.Logf("  [%d] srcIdx=%d lineNum=%d wrap=%v content=%q",
			i, sl.sourceLineIdx, sl.lineNum, sl.isWrapped, sl.content)
	}

	// Position on the wrapped line (source line 1)
	m.cursorLine = 1

	// Get visual mapping for the wrapped line
	visualForLine1, _ := alignedBefore.sourceToVisual[1]
	t.Logf("Source line 1 maps to visual %d", visualForLine1)

	// Insert below the wrapped line
	m.insertLineBelow()

	// The NEW line should be source line 2
	if m.cursorLine != 2 {
		t.Errorf("Cursor should be on line 2, got %d", m.cursorLine)
	}

	// Get visual state after insert
	alignedAfter := m.computeAlignedPanes(leftWidth, rightWidth)
	t.Logf("After insert: %d source visual lines", len(alignedAfter.sourceLines))
	for i, sl := range alignedAfter.sourceLines {
		t.Logf("  [%d] srcIdx=%d lineNum=%d wrap=%v padding=%v cursor=%v content=%q",
			i, sl.sourceLineIdx, sl.lineNum, sl.isWrapped, sl.isPadding, sl.isCursorLine, sl.content)
	}

	// Verify mapping consistency
	for srcLine := 0; srcLine < m.TotalLines(); srcLine++ {
		visualIdx, ok := alignedAfter.sourceToVisual[srcLine]
		if !ok {
			t.Errorf("Missing mapping for source line %d", srcLine)
			continue
		}

		// Check the visual line at that index has correct sourceLineIdx
		if visualIdx >= 0 && visualIdx < len(alignedAfter.sourceLines) {
			sl := alignedAfter.sourceLines[visualIdx]
			if sl.sourceLineIdx != srcLine {
				t.Errorf("Mapping inconsistency: sourceToVisual[%d]=%d, but visual[%d].sourceLineIdx=%d",
					srcLine, visualIdx, visualIdx, sl.sourceLineIdx)
			}
		}
	}

	// Find cursor in visual structure
	cursorVisual := -1
	for i, sl := range alignedAfter.sourceLines {
		if sl.isCursorLine {
			cursorVisual = i
			break
		}
	}

	expectedVisual, _ := alignedAfter.sourceToVisual[m.cursorLine]
	if cursorVisual != expectedVisual {
		t.Errorf("Cursor highlight bug: found at visual %d, mapping says %d",
			cursorVisual, expectedVisual)
	}
}

// TestAlignedModelCache tests that the cache works correctly.
func TestTrailingEmptyLinesPreserved(t *testing.T) {
	// Test that pressing Enter multiple times at end of document preserves empty lines
	// This is critical for TUI editors - users need to see and edit trailing lines
	content := `gzip_compressed = compress(1 GB, gzip)
lz4_compressed = compress(100 MB, lz4)`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 100
	m.height = 24
	m.previewMode = PreviewFull

	// Initial state: 2 lines
	if got := len(m.GetLines()); got != 2 {
		t.Fatalf("Initial: expected 2 lines, got %d", got)
	}

	// Move cursor to last line and insert below (like pressing 'o')
	m.cursorLine = 1
	m.insertLineBelow()

	// After first insert: 3 lines
	if got := len(m.GetLines()); got != 3 {
		t.Errorf("After first insert: expected 3 lines, got %d", got)
	}
	if m.cursorLine != 2 {
		t.Errorf("After first insert: cursor should be at 2, got %d", m.cursorLine)
	}

	// Insert another (like pressing Enter again)
	m.insertLineBelow()

	// After second insert: 4 lines (this was the bug - it used to stay at 3)
	lines := m.GetLines()
	if len(lines) != 4 {
		t.Errorf("After second insert: expected 4 lines, got %d", len(lines))
	}
	if m.cursorLine != 3 {
		t.Errorf("After second insert: cursor should be at 3, got %d", m.cursorLine)
	}

	// Verify results match lines
	results := m.GetLineResults()
	if len(lines) != len(results) {
		t.Errorf("Mismatch: %d lines but %d results", len(lines), len(results))
	}

	// Verify aligned model has all lines
	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	aligned := m.computeAlignedModelFresh(leftWidth, rightWidth)

	for lineNum := range len(lines) {
		found := false
		for _, sl := range aligned.SourceLines {
			if sl.SourceLineIdx == lineNum && sl.Kind != AlignedLinePadding {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Source line %d not represented in aligned model", lineNum)
		}
	}
}
