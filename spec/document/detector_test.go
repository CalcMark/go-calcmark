package document

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/lexer"
)

func TestBlockDetection(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name          string
		source        string
		expectedTypes []BlockType
		expectedCount int
	}{
		{
			name:          "single calc line",
			source:        "x = 10",
			expectedTypes: []BlockType{BlockCalculation},
			expectedCount: 1,
		},
		{
			name:          "two calc lines",
			source:        "x = 10\ny = 20",
			expectedTypes: []BlockType{BlockCalculation},
			expectedCount: 1,
		},
		{
			name:          "calc then text",
			source:        "x = 10\n# Header",
			expectedTypes: []BlockType{BlockCalculation, BlockText},
			expectedCount: 2,
		},
		{
			name:          "calc with 1 empty line (stays in block)",
			source:        "x = 10\n\ny = 20",
			expectedTypes: []BlockType{BlockCalculation},
			expectedCount: 1,
		},
		{
			name:          "calc with 2 empty lines (splits blocks)",
			source:        "x = 10\n\n\ny = 20",
			expectedTypes: []BlockType{BlockCalculation, BlockCalculation},
			expectedCount: 2,
		},
		{
			name:          "text then calc then text",
			source:        "# Header\nx = 10\nMore text",
			expectedTypes: []BlockType{BlockText, BlockCalculation, BlockText},
			expectedCount: 3,
		},
		{
			name:          "ordered list is text not calc",
			source:        "1. First\n2. Second\n3. Third",
			expectedTypes: []BlockType{BlockText},
			expectedCount: 1,
		},
		{
			name:          "ordered list mixed with calc",
			source:        "1. First item\nx = 10\n2. Second item",
			expectedTypes: []BlockType{BlockText, BlockCalculation, BlockText},
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks, err := detector.DetectBlocks(tt.source)
			if err != nil {
				t.Fatalf("DetectBlocks() error = %v", err)
			}

			if len(blocks) != tt.expectedCount {
				t.Errorf("Expected %d blocks, got %d", tt.expectedCount, len(blocks))
			}

			for i, expectedType := range tt.expectedTypes {
				if i >= len(blocks) {
					break
				}
				if blocks[i].Type() != expectedType {
					t.Errorf("Block %d: expected type %v, got %v", i, expectedType, blocks[i].Type())
				}
			}
		})
	}
}

func TestMarkdownWithBoldText(t *testing.T) {
	detector := NewDetector()

	// Test that markdown bold syntax (**text**) is correctly identified as text
	tests := []struct {
		name   string
		source string
		isText bool
	}{
		{
			name:   "bold text in sentence",
			source: "This tests the **two empty line rule** for creating hard block boundaries.",
			isText: true,
		},
		{
			name:   "heading with bold",
			source: "# Block **Boundary** Rules",
			isText: true,
		},
		{
			name:   "sentence with numbers and bold",
			source: "There are **10 ways** to do this.",
			isText: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks, err := detector.DetectBlocks(tt.source)
			if err != nil {
				t.Fatalf("DetectBlocks() error = %v, want nil", err)
			}

			if len(blocks) != 1 {
				t.Fatalf("Expected 1 block, got %d", len(blocks))
			}

			if tt.isText && blocks[0].Type() != BlockText {
				t.Errorf("Expected TextBlock, got %v", blocks[0].Type())
			}
		})
	}
}

func TestPowerOperatorNotConfusedWithMarkdown(t *testing.T) {
	detector := NewDetector()

	// Test that actual power operators are still detected as calculations
	tests := []struct {
		name   string
		source string
		isCalc bool
	}{
		{
			name:   "power operator with spaces",
			source: "2 ** 3",
			isCalc: true,
		},
		{
			name:   "power in expression",
			source: "x = 2 ** 10",
			isCalc: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks, err := detector.DetectBlocks(tt.source)
			if err != nil {
				t.Fatalf("DetectBlocks() error = %v, want nil", err)
			}

			if len(blocks) != 1 {
				t.Fatalf("Expected 1 block, got %d", len(blocks))
			}

			if tt.isCalc && blocks[0].Type() != BlockCalculation {
				t.Errorf("Expected CalcBlock, got %v", blocks[0].Type())
			}
		})
	}
}

func TestMultilineWithMarkdownBold(t *testing.T) {
	detector := NewDetector()

	// Test multiline documents with bold markdown
	source := `# Block Boundary Rules Test

This tests the **two empty line rule** for creating hard block boundaries.

## Same Type, One Empty Line

x = 10

y = 20`

	blocks, err := detector.DetectBlocks(source)
	if err != nil {
		t.Fatalf("DetectBlocks() error = %v, want nil", err)
	}

	t.Logf("Got %d blocks", len(blocks))
	for i, b := range blocks {
		t.Logf("  Block %d: type=%v, lines=%d", i, b.Type(), len(b.Source()))
	}
}

func TestFileContentFromBlockBoundaries(t *testing.T) {
	detector := NewDetector()

	// This is the exact content from block_boundaries.cm
	source := `# Block Boundary Rules Test

This tests the **two empty line rule** for creating hard block boundaries.

## Same Type, One Empty Line

x = 10

y = 20

These two calculations should be in the SAME CalcBlock.

## Same Type, Two Empty Lines

a = 1


b = 2

These should be in DIFFERENT CalcBlocks (two empty lines = hard boundary).

## Different Types Always Split

c = 3

This is text.

d = 4

Even with one empty line, different types create boundaries.

## Text Blocks with Two Empty Lines

First text paragraph.


Second text paragraph.

These should be separate TextBlocks.

## Text Blocks with One Empty Line

First paragraph.

Second paragraph.

These should be in the SAME TextBlock.
`

	blocks, err := detector.DetectBlocks(source)
	if err != nil {
		t.Fatalf("DetectBlocks() error = %v, want nil", err)
	}

	t.Logf("Got %d blocks", len(blocks))
	for i, b := range blocks {
		t.Logf("  Block %d: type=%v, lines=%d", i, b.Type(), len(b.Source()))
		for j, line := range b.Source() {
			t.Logf("    Line %d: %q", j, line)
		}
	}
}

func TestQuantityLiterals(t *testing.T) {
	detector := NewDetector()

	// Test that quantity literals (number + unit) are detected as calculations
	tests := []struct {
		name   string
		source string
		isCalc bool
	}{
		{"number with unit", "10 meters", true},
		{"unit conversion", "10 meters in feet", true},
		{"currency amount", "$100", true},
		{"plain number", "42", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks, err := detector.DetectBlocks(tt.source)
			if err != nil {
				t.Fatalf("DetectBlocks error: %v", err)
			}

			if len(blocks) != 1 {
				t.Fatalf("Expected 1 block, got %d", len(blocks))
			}

			isCalc := blocks[0].Type() == BlockCalculation
			if isCalc != tt.isCalc {
				t.Errorf("Expected isCalc=%v, got %v for %q", tt.isCalc, isCalc, tt.source)
				// Debug: show what the lexer produces
				lex := lexer.NewLexer(tt.source)
				tokens, _ := lex.Tokenize()
				for i, tok := range tokens {
					t.Logf("  Token %d: %v", i, tok)
				}
			}
		})
	}
}

// --- Phase 2: CommonMark detector gap tests ---
// These tests verify that the detector correctly classifies CommonMark constructs.

func TestReferenceStyleLinkDefinition(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name   string
		line   string
		isCalc bool
	}{
		{"simple reference", "[wiki]: https://en.wikipedia.org", false},
		{"reference with title", `[example]: https://example.com "Example"`, false},
		{"short reference", "[1]: http://example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isCalc, err := detector.IsCalculation(tt.line)
			if err != nil {
				t.Fatalf("IsCalculation error: %v", err)
			}
			if isCalc != tt.isCalc {
				t.Errorf("IsCalculation(%q) = %v, want %v", tt.line, isCalc, tt.isCalc)
			}
		})
	}
}

func TestSetextHeadingKeepsSameBlock(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name          string
		source        string
		expectedTypes []BlockType
		expectedCount int
	}{
		{
			name:          "setext H1 with ===",
			source:        "Heading\n=======",
			expectedTypes: []BlockType{BlockText},
			expectedCount: 1,
		},
		{
			name:          "setext H2 with ---",
			source:        "Heading\n-------",
			expectedTypes: []BlockType{BlockText},
			expectedCount: 1,
		},
		{
			name:          "setext heading followed by calc",
			source:        "Heading\n=======\nx = 10",
			expectedTypes: []BlockType{BlockText, BlockCalculation},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks, err := detector.DetectBlocks(tt.source)
			if err != nil {
				t.Fatalf("DetectBlocks error: %v", err)
			}

			if len(blocks) != tt.expectedCount {
				t.Errorf("Expected %d blocks, got %d", tt.expectedCount, len(blocks))
				for i, b := range blocks {
					t.Logf("  Block %d: type=%v lines=%v", i, b.Type(), b.Source())
				}
			}

			for i, expectedType := range tt.expectedTypes {
				if i >= len(blocks) {
					break
				}
				if blocks[i].Type() != expectedType {
					t.Errorf("Block %d: expected type %v, got %v", i, expectedType, blocks[i].Type())
				}
			}
		})
	}
}

func TestHorizontalRuleAfterBlankLine(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name   string
		line   string
		isCalc bool
	}{
		{"triple dash", "---", false},
		{"triple asterisk", "***", false},
		{"triple underscore", "___", false},
		{"spaced dashes", "- - -", false},
		{"spaced asterisks", "* * *", false},
		{"long dashes", "----------", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isCalc, err := detector.IsCalculation(tt.line)
			if err != nil {
				t.Fatalf("IsCalculation error: %v", err)
			}
			if isCalc != tt.isCalc {
				t.Errorf("IsCalculation(%q) = %v, want %v", tt.line, isCalc, tt.isCalc)
			}
		})
	}
}

func TestFencedCodeBlockFence(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name   string
		line   string
		isCalc bool
	}{
		{"backtick fence", "```", false},
		{"tilde fence", "~~~", false},
		{"backtick with lang", "```go", false},
		{"long backtick fence", "````", false},
		{"long tilde fence", "~~~~", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isCalc, err := detector.IsCalculation(tt.line)
			if err != nil {
				t.Fatalf("IsCalculation error: %v", err)
			}
			if isCalc != tt.isCalc {
				t.Errorf("IsCalculation(%q) = %v, want %v", tt.line, isCalc, tt.isCalc)
			}
		})
	}
}

func TestFencedCodeBlockContentsStayInTextBlock(t *testing.T) {
	detector := NewDetector()

	// Lines inside a fenced code block that look like calculations
	// must NOT be classified as CalcBlocks
	source := "```\nx = 10\ny = x * 2\n```"

	blocks, err := detector.DetectBlocks(source)
	if err != nil {
		t.Fatalf("DetectBlocks error: %v", err)
	}

	if len(blocks) != 1 {
		t.Errorf("Expected 1 TextBlock for fenced code block, got %d blocks", len(blocks))
		for i, b := range blocks {
			t.Logf("  Block %d: type=%v lines=%v", i, b.Type(), b.Source())
		}
		return
	}

	if blocks[0].Type() != BlockText {
		t.Errorf("Fenced code block should be TextBlock, got %v", blocks[0].Type())
	}
}

func TestIndentedCodeBlock(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name   string
		line   string
		isCalc bool
	}{
		{"4-space indent with calc syntax", "    x = 10", false},
		{"tab indent with calc syntax", "\tx = 10", false},
		{"8-space indent", "        y = 20", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isCalc, err := detector.IsCalculation(tt.line)
			if err != nil {
				t.Fatalf("IsCalculation error: %v", err)
			}
			if isCalc != tt.isCalc {
				t.Errorf("IsCalculation(%q) = %v, want %v", tt.line, isCalc, tt.isCalc)
			}
		})
	}
}

func TestPlusListMarker(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name   string
		line   string
		isCalc bool
	}{
		{"plus list item", "+ First item", false},
		{"plus with nested", "+ Nested item here", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isCalc, err := detector.IsCalculation(tt.line)
			if err != nil {
				t.Fatalf("IsCalculation error: %v", err)
			}
			if isCalc != tt.isCalc {
				t.Errorf("IsCalculation(%q) = %v, want %v", tt.line, isCalc, tt.isCalc)
			}
		})
	}
}

func TestImageSyntax(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name   string
		line   string
		isCalc bool
	}{
		{"simple image", "![alt text](image.png)", false},
		{"image with title", `![photo](pic.jpg "A photo")`, false},
		{"image at line start", "![diagram](https://example.com/diagram.svg)", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isCalc, err := detector.IsCalculation(tt.line)
			if err != nil {
				t.Fatalf("IsCalculation error: %v", err)
			}
			if isCalc != tt.isCalc {
				t.Errorf("IsCalculation(%q) = %v, want %v", tt.line, isCalc, tt.isCalc)
			}
		})
	}
}

func TestEmptyLineDelimiter(t *testing.T) {
	detector := NewDetector()

	// 2 consecutive empty lines = block boundary
	source := `x = 10


y = 20`

	blocks, err := detector.DetectBlocks(source)
	if err != nil {
		t.Fatalf("DetectBlocks() error = %v", err)
	}

	if len(blocks) != 2 {
		t.Errorf("Expected 2 blocks (split by 2 empty lines), got %d", len(blocks))
	}

	// Both should be calc blocks
	for i, block := range blocks {
		if block.Type() != BlockCalculation {
			t.Errorf("Block %d should be calculation, got %v", i, block.Type())
		}
	}
}

// --- Phase 4e: Regression safety net ---
// Parse all existing .cm files through DetectBlocks and record block type counts.
// This prevents detector changes from silently reclassifying existing documents.

func TestDetectorRegressionAllCMFiles(t *testing.T) {
	detector := NewDetector()

	// Expected block type counts for each .cm file.
	// Format: file path (relative to testdata/) -> {textBlocks, calcBlocks}
	type blockCounts struct {
		text int
		calc int
	}

	// Walk testdata/ to find all .cm files and verify they all parse without error.
	// Then check specific known files for stable block counts.
	testdataDir := "../../testdata"

	// Known files with expected block counts
	knownFiles := map[string]blockCounts{
		"spec/valid/documents/mixed_content.cm":    {text: 4, calc: 3},
		"spec/valid/documents/block_boundaries.cm": {text: 6, calc: 5},
	}

	// First: verify ALL .cm files parse without error
	err := filepath.Walk(testdataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".cm" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("Failed to read %s: %v", path, err)
			return nil
		}

		// Strip frontmatter before DetectBlocks (same as NewDocument does)
		source := string(data)
		_, remaining, fmErr := ParseFrontmatter(source)
		if fmErr != nil {
			// Some files may have intentionally invalid frontmatter
			remaining = source
		}

		_, detectErr := detector.DetectBlocks(remaining)
		if detectErr != nil {
			t.Errorf("DetectBlocks failed for %s: %v", path, detectErr)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Walk error: %v", err)
	}

	// Second: verify known files have stable block type counts
	for relPath, expected := range knownFiles {
		fullPath := filepath.Join(testdataDir, relPath)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			t.Errorf("Failed to read %s: %v", fullPath, err)
			continue
		}

		source := string(data)
		_, remaining, _ := ParseFrontmatter(source)

		blocks, err := detector.DetectBlocks(remaining)
		if err != nil {
			t.Errorf("DetectBlocks failed for %s: %v", relPath, err)
			continue
		}

		textCount, calcCount := 0, 0
		for _, b := range blocks {
			switch b.Type() {
			case BlockText:
				textCount++
			case BlockCalculation:
				calcCount++
			}
		}

		if textCount != expected.text || calcCount != expected.calc {
			t.Errorf("%s: expected %d text + %d calc blocks, got %d text + %d calc",
				relPath, expected.text, expected.calc, textCount, calcCount)
			for i, b := range blocks {
				t.Logf("  Block %d: type=%v lines=%d", i, b.Type(), len(b.Source()))
			}
		}
	}
}
