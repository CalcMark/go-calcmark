package document

import (
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
