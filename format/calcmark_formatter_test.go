package format

import (
	"bytes"
	"strings"
	"testing"

	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestCalcMarkFormatterSimple tests basic CalcMark output
func TestCalcMarkFormatterSimple(t *testing.T) {
	source := "x = 10\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	var buf bytes.Buffer
	formatter := &CalcMarkFormatter{}
	opts := Options{Verbose: false}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()
	// Should contain source
	if !strings.Contains(output, "x = 10") {
		t.Errorf("Expected output to contain source, got: %s", output)
	}
}

// TestCalcMarkFormatterVerbose tests verbose mode with comments
func TestCalcMarkFormatterVerbose(t *testing.T) {
	source := "x = 100 + 50\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	var buf bytes.Buffer
	formatter := &CalcMarkFormatter{}
	opts := Options{Verbose: true}

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()
	// Should contain source
	if !strings.Contains(output, "x = 100 + 50") {
		t.Errorf("Expected output to contain source, got: %s", output)
	}
	// In verbose mode, should contain result as comment
	if !strings.Contains(output, "#") || !strings.Contains(output, "150") {
		t.Errorf("Expected verbose output to contain result comment, got: %s", output)
	}
}

// TestCalcMarkFormatterExtensions tests file extensions
func TestCalcMarkFormatterExtensions(t *testing.T) {
	formatter := &CalcMarkFormatter{}
	exts := formatter.Extensions()

	if len(exts) < 2 {
		t.Fatal("CalcMarkFormatter should handle at least 2 extensions")
	}

	foundCM := false
	foundCalcmark := false
	for _, ext := range exts {
		if ext == ".cm" {
			foundCM = true
		}
		if ext == ".calcmark" {
			foundCalcmark = true
		}
	}

	if !foundCM || !foundCalcmark {
		t.Error("CalcMarkFormatter should handle .cm and .calcmark extensions")
	}
}

// TestCalcMarkFormatterRoundTrip tests that input == output for valid documents.
// This is critical for /save command to be lossless.
func TestCalcMarkFormatterRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "single calculation",
			source: "x = 10\n",
		},
		{
			name:   "multiple calculations in one block",
			source: "x = 10\ny = 20\nz = x + y\n",
		},
		{
			name: "two separate blocks",
			source: `alice = 100


bob = alice * 2
`,
		},
		{
			name: "calculation and text blocks",
			source: `x = 42


# Markdown Heading

Some **bold** text here.


y = x * 2
`,
		},
		{
			name: "three blocks with dependencies",
			source: `base = 100


rate = 0.15


total = base * (1 + rate)
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the source
			doc, err := document.NewDocument(tt.source)
			if err != nil {
				t.Fatalf("Failed to create document: %v", err)
			}

			// Evaluate (needed for results, though not used in non-verbose)
			eval := implDoc.NewEvaluator()
			if err := eval.Evaluate(doc); err != nil {
				t.Fatalf("Failed to evaluate: %v", err)
			}

			// Format back to CalcMark (non-verbose for round-trip)
			var buf bytes.Buffer
			formatter := &CalcMarkFormatter{}
			err = formatter.Format(&buf, doc, Options{Verbose: false})
			if err != nil {
				t.Fatalf("Format failed: %v", err)
			}

			output := buf.String()

			// Round-trip test: output should equal input
			if output != tt.source {
				t.Errorf("Round-trip failed:\nInput:\n%q\n\nOutput:\n%q", tt.source, output)
			}
		})
	}
}

// TestCalcMarkFormatterRoundTripReparse tests that formatted output can be reparsed
// and produces the same document structure.
func TestCalcMarkFormatterRoundTripReparse(t *testing.T) {
	source := `x = 100


y = x * 2


z = y + 50
`

	// First parse
	doc1, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Format
	var buf bytes.Buffer
	formatter := &CalcMarkFormatter{}
	err = formatter.Format(&buf, doc1, Options{Verbose: false})
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Reparse the output
	doc2, err := document.NewDocument(buf.String())
	if err != nil {
		t.Fatalf("Failed to reparse formatted output: %v", err)
	}

	// Compare block counts
	blocks1 := doc1.GetBlocks()
	blocks2 := doc2.GetBlocks()

	if len(blocks1) != len(blocks2) {
		t.Errorf("Block count mismatch: original=%d, reparsed=%d", len(blocks1), len(blocks2))
	}

	// Compare block types and source
	for i := range blocks1 {
		if blocks1[i].Block.Type() != blocks2[i].Block.Type() {
			t.Errorf("Block %d type mismatch: %v vs %v",
				i, blocks1[i].Block.Type(), blocks2[i].Block.Type())
		}

		src1 := strings.Join(blocks1[i].Block.Source(), "\n")
		src2 := strings.Join(blocks2[i].Block.Source(), "\n")
		if src1 != src2 {
			t.Errorf("Block %d source mismatch:\n%q\nvs\n%q", i, src1, src2)
		}
	}
}
