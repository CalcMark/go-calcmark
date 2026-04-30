package editor

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

// TestAlignment_OneToOneMapping verifies that every source line maps to exactly one preview line.
// This is CRITICAL for the split-pane editor UX - users expect source and preview to align vertically.
func TestAlignment_OneToOneMapping(t *testing.T) {
	tests := []struct {
		name            string
		source          string
		wantSourceLines int
	}{
		{
			name:            "simple text",
			source:          "hello world",
			wantSourceLines: 1,
		},
		{
			name:            "two text lines",
			source:          "line 1\nline 2",
			wantSourceLines: 2,
		},
		{
			name:            "heading level 1",
			source:          "# Title",
			wantSourceLines: 1,
		},
		{
			name:            "heading level 2",
			source:          "## Subtitle",
			wantSourceLines: 1,
		},
		{
			name:            "heading followed by text",
			source:          "# Title\nSome text here",
			wantSourceLines: 2,
		},
		{
			name:            "ordered list 3 items",
			source:          "1. First\n2. Second\n3. Third",
			wantSourceLines: 3,
		},
		{
			name:            "unordered list 3 items",
			source:          "- First\n- Second\n- Third",
			wantSourceLines: 3,
		},
		{
			name:            "mixed heading and list",
			source:          "# Title\n\n1. First\n2. Second",
			wantSourceLines: 4,
		},
		{
			name:            "paragraph with empty lines",
			source:          "First para\n\nSecond para",
			wantSourceLines: 3,
		},
		{
			name:            "bold text",
			source:          "This is **bold** text",
			wantSourceLines: 1,
		},
		{
			name:            "italic text",
			source:          "This is *italic* text",
			wantSourceLines: 1,
		},
		{
			name:            "code span",
			source:          "This is `code` text",
			wantSourceLines: 1,
		},
		{
			name:            "link",
			source:          "[text](https://example.com)",
			wantSourceLines: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := document.NewDocument(tt.source)
			if err != nil {
				t.Fatalf("Failed to create document: %v", err)
			}

			m := New(doc)
			m.width = 80
			m.height = 24

			// Get source lines
			sourceLines := m.GetLines()
			if len(sourceLines) != tt.wantSourceLines {
				t.Errorf("Source line count mismatch: got %d, want %d",
					len(sourceLines), tt.wantSourceLines)
			}

			// Get aligned model (use full width for simplicity)
			aligned := m.GetAlignedModel(40, 40)
			if aligned == nil {
				t.Fatal("GetAlignedModel returned nil")
			}

			// CRITICAL: Preview line count MUST match source line count for 1:1 alignment
			previewLineCount := len(aligned.PreviewLines)
			if previewLineCount != len(sourceLines) {
				t.Errorf("ALIGNMENT BUG: Preview has %d lines but source has %d lines",
					previewLineCount, len(sourceLines))
				t.Logf("Source lines: %v", sourceLines)
				t.Logf("Source visual lines: %d", len(aligned.SourceLines))
				t.Logf("Preview visual lines: %d", len(aligned.PreviewLines))
			}
		})
	}
}

// TestAlignment_HeadingRendering specifically tests heading alignment.
// Headings are a common source of alignment bugs because markdown renderers
// often add decorative elements or spacing.
func TestAlignment_HeadingRendering(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"h1", "# Heading 1"},
		{"h2", "## Heading 2"},
		{"h3", "### Heading 3"},
		{"h4", "#### Heading 4"},
		{"h5", "##### Heading 5"},
		{"h6", "###### Heading 6"},
		{"h1 with text below", "# Title\nText below"},
		{"h2 with text below", "## Subtitle\nText below"},
		{"multiple headings", "# H1\n## H2\n### H3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := document.NewDocument(tt.source)
			if err != nil {
				t.Fatalf("Failed to create document: %v", err)
			}

			m := New(doc)
			m.width = 80
			m.height = 24

			sourceLines := m.GetLines()
			aligned := m.GetAlignedModel(40, 40)

			previewLineCount := len(aligned.PreviewLines)

			if previewLineCount != len(sourceLines) {
				t.Errorf("Heading alignment bug: Preview has %d lines but source has %d lines",
					previewLineCount, len(sourceLines))
				t.Logf("Source: %q", tt.source)
				t.Logf("Source lines: %v", sourceLines)
				t.Logf("Preview lines: %d", len(aligned.PreviewLines))
			}
		})
	}
}

// TestAlignment_OrderedLists verifies ordered lists maintain 1:1 alignment
// AND have correct numbering.
func TestAlignment_OrderedLists(t *testing.T) {
	source := "1. First\n2. Second\n3. Third"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	sourceLines := m.GetLines()
	aligned := m.GetAlignedModel(40, 40)

	// Verify line count
	previewLineCount := len(aligned.PreviewLines)

	if previewLineCount != len(sourceLines) {
		t.Errorf("Ordered list alignment: Preview has %d lines but source has %d",
			previewLineCount, len(sourceLines))
	}

	// Verify numbering - extract text from preview lines
	previewText := []string{}
	for _, line := range aligned.PreviewLines {
		previewText = append(previewText, line.Content)
	}

	// Check that preview contains incrementing numbers.
	// Strip ANSI escape codes before checking since styled output may separate
	// digits from punctuation with escape sequences (e.g., "1" + ESC + ".").
	hasOne := false
	hasTwo := false
	hasThree := false
	for _, line := range previewText {
		plain := ansiEscapeRe.ReplaceAllString(line, "")
		if strings.Contains(plain, "1.") || strings.Contains(plain, "1 ") {
			hasOne = true
		}
		if strings.Contains(plain, "2.") || strings.Contains(plain, "2 ") {
			hasTwo = true
		}
		if strings.Contains(plain, "3.") || strings.Contains(plain, "3 ") {
			hasThree = true
		}
	}

	if !hasOne || !hasTwo || !hasThree {
		t.Errorf("Ordered list numbering broken: preview=%v", previewText)
	}
}

// TestAlignment_EmptyLines verifies empty lines are preserved in alignment.
func TestAlignment_EmptyLines(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   int
	}{
		{"single empty", "", 1}, // Empty document has 1 empty line
		{"text with empty after", "hello\n", 1},
		{"text empty text", "first\n\nsecond", 3},
		{"multiple empty", "\n\n\n", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := document.NewDocument(tt.source)
			if err != nil {
				t.Fatalf("Failed to create document: %v", err)
			}

			m := New(doc)
			m.width = 80
			m.height = 24

			sourceLines := m.GetLines()
			if len(sourceLines) != tt.want {
				t.Errorf("Source line count: got %d, want %d", len(sourceLines), tt.want)
			}

			aligned := m.GetAlignedModel(40, 40)
			previewLineCount := len(aligned.PreviewLines)

			if previewLineCount != len(sourceLines) {
				t.Errorf("Empty line alignment: Preview has %d lines but source has %d",
					previewLineCount, len(sourceLines))
			}
		})
	}
}

// TestAlignment_CalcBlocks verifies that calc blocks maintain alignment.
func TestAlignment_CalcBlocks(t *testing.T) {
	source := "x = 10\ny = 20\nz = x + y"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	sourceLines := m.GetLines()
	aligned := m.GetAlignedModel(40, 40)

	previewLineCount := len(aligned.PreviewLines)

	if previewLineCount != len(sourceLines) {
		t.Errorf("Calc block alignment: Preview has %d lines but source has %d",
			previewLineCount, len(sourceLines))
	}
}

// TestAlignment_GlamourHeadingOutput tests what glamour actually renders for headings.
// This helps debug visual alignment issues where the line count is correct but
// the rendering makes it appear misaligned.
func TestAlignment_GlamourHeadingOutput(t *testing.T) {
	source := "# My Title\nText below"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	sourceLines := m.GetLines()
	aligned := m.GetAlignedModel(40, 40)

	t.Logf("Source: %q", source)
	t.Logf("Source lines (%d):", len(sourceLines))
	for i, line := range sourceLines {
		t.Logf("  [%d] %q", i, line)
	}

	t.Logf("Preview lines (%d):", len(aligned.PreviewLines))
	for i, line := range aligned.PreviewLines {
		t.Logf("  [%d] Content=%q, SourceLine=%d",
			i, line.Content, line.SourceLineIdx)
	}

	// Verify 1:1 mapping
	if len(aligned.PreviewLines) != len(sourceLines) {
		t.Errorf("Preview line count (%d) != source line count (%d)",
			len(aligned.PreviewLines), len(sourceLines))
	}

	// Verify each preview line correctly maps to its source line
	for i, previewLine := range aligned.PreviewLines {
		if previewLine.SourceLineIdx != i {
			t.Errorf("Preview line %d maps to source line %d (expected %d)",
				i, previewLine.SourceLineIdx, i)
		}
	}
}

// TestAlignment_OrderedListFollowedByBlankAndText tests the specific case the user reported:
// typing a blank line after an ordered list, then typing more content.
// This is a regression test for alignment issues with this pattern.
func TestAlignment_OrderedListFollowedByBlankAndText(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   int // expected line count
	}{
		{
			name:   "ordered list + blank + heading",
			source: "1. First\n2. Second\n3. Third\n\n# Title",
			want:   5,
		},
		{
			name:   "ordered list + blank + text",
			source: "1. First\n2. Second\n3. Third\n\nSome text here",
			want:   5,
		},
		{
			name:   "ordered list + blank + calc",
			source: "1. First\n2. Second\n3. Third\n\nx = 10",
			want:   5,
		},
		{
			name:   "ordered list + 2 blanks + text",
			source: "1. First\n2. Second\n3. Third\n\n\nText after two blanks",
			want:   6,
		},
		{
			name:   "ordered list + blank + another list",
			source: "1. First\n2. Second\n\n1. New list",
			want:   4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := document.NewDocument(tt.source)
			if err != nil {
				t.Fatalf("Failed to create document: %v", err)
			}

			m := New(doc)
			m.width = 80
			m.height = 24

			sourceLines := m.GetLines()
			if len(sourceLines) != tt.want {
				t.Errorf("Source line count: got %d, want %d", len(sourceLines), tt.want)
				t.Logf("Source lines: %v", sourceLines)
			}

			aligned := m.GetAlignedModel(40, 40)
			previewLineCount := len(aligned.PreviewLines)

			if previewLineCount != len(sourceLines) {
				t.Errorf("ALIGNMENT BUG: Preview has %d lines but source has %d lines",
					previewLineCount, len(sourceLines))
				t.Logf("Source: %q", tt.source)
				t.Logf("Source lines (%d): %v", len(sourceLines), sourceLines)
				t.Logf("Preview lines (%d):", len(aligned.PreviewLines))
				for i, line := range aligned.PreviewLines {
					t.Logf("  [%d] SourceLine=%d Content=%q",
						i, line.SourceLineIdx, line.Content)
				}
			}
		})
	}
}

// TestAlignment_MixedContent verifies complex documents with mixed content types.
func TestAlignment_MixedContent(t *testing.T) {
	source := `# Project Budget

Initial amount:

x = 1000

## Expenses

1. Rent
2. Food
3. Transport

Total spent:

y = 500

## Remaining

z = x - y`

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	sourceLines := m.GetLines()
	aligned := m.GetAlignedModel(40, 40)

	previewLineCount := len(aligned.PreviewLines)

	if previewLineCount != len(sourceLines) {
		t.Errorf("Mixed content alignment: Preview has %d lines but source has %d",
			previewLineCount, len(sourceLines))
		t.Logf("Source (%d lines): %v", len(sourceLines), sourceLines)
		t.Logf("Preview (%d lines)", len(aligned.PreviewLines))
		for i, line := range aligned.PreviewLines {
			t.Logf("  Preview[%d]: %q", i, line.Content)
		}
	}
}
