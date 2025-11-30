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

// TestCalcMarkFormatterPreservesSource tests that source is preserved exactly
func TestCalcMarkFormatterPreservesSource(t *testing.T) {
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
	opts := Options{Verbose: true} // Verbose should not affect .cm output

	err = formatter.Format(&buf, doc, opts)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()
	// Should contain source exactly
	if !strings.Contains(output, "x = 100 + 50") {
		t.Errorf("Expected output to contain source, got: %s", output)
	}
	// Should NOT add any result annotations (reproducibility)
	if strings.Contains(output, "→") {
		t.Errorf("CalcMark formatter should not add → results, got: %s", output)
	}
	if strings.Contains(output, "# =") {
		t.Errorf("CalcMark formatter should not add # comments, got: %s", output)
	}
}

// TestCalcMarkFormatterFiltersLegacyComments tests that old # = comments are removed
func TestCalcMarkFormatterFiltersLegacyComments(t *testing.T) {
	// Source with legacy # = comment (from old verbose saves)
	source := "x = 10\n  # = 10\n"
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
	// Should contain the calculation
	if !strings.Contains(output, "x = 10") {
		t.Errorf("Expected output to contain calculation, got: %s", output)
	}
	// Should NOT contain the legacy # = comment
	if strings.Contains(output, "# =") {
		t.Errorf("Legacy # = comment should be filtered out, got: %s", output)
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

// TestCalcMarkFormatterWithFrontmatter tests that frontmatter is serialized.
func TestCalcMarkFormatterWithFrontmatter(t *testing.T) {
	source := `---
exchange:
  USD_EUR: 0.92
globals:
  tax_rate: 0.32
---
price = 100 USD
total = price in EUR
`
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

	// Should contain frontmatter
	if !strings.Contains(output, "---") {
		t.Errorf("Expected output to contain frontmatter delimiters")
	}
	if !strings.Contains(output, "exchange:") {
		t.Errorf("Expected output to contain exchange section")
	}
	if !strings.Contains(output, "USD_EUR") {
		t.Errorf("Expected output to contain exchange rate key")
	}
	if !strings.Contains(output, "globals:") {
		t.Errorf("Expected output to contain globals section")
	}
	if !strings.Contains(output, "tax_rate") {
		t.Errorf("Expected output to contain global variable")
	}
	// Should contain source
	if !strings.Contains(output, "price = 100 USD") {
		t.Errorf("Expected output to contain source, got: %s", output)
	}
}

// TestCalcMarkFormatterFrontmatterRoundTrip tests frontmatter round-trip.
func TestCalcMarkFormatterFrontmatterRoundTrip(t *testing.T) {
	source := `---
exchange:
  USD_EUR: 0.92
globals:
  tax_rate: 0.32
---
x = 10
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

	// Compare frontmatter
	fm1 := doc1.GetFrontmatter()
	fm2 := doc2.GetFrontmatter()

	if fm1 == nil || fm2 == nil {
		t.Fatal("Expected frontmatter in both documents")
	}

	// Compare exchange rates
	if len(fm1.Exchange) != len(fm2.Exchange) {
		t.Errorf("Exchange rate count mismatch: %d vs %d", len(fm1.Exchange), len(fm2.Exchange))
	}

	for key, rate1 := range fm1.Exchange {
		rate2, ok := fm2.Exchange[key]
		if !ok {
			t.Errorf("Missing exchange rate %s in reparsed document", key)
		} else if !rate1.Equal(rate2) {
			t.Errorf("Exchange rate %s mismatch: %v vs %v", key, rate1, rate2)
		}
	}

	// Compare globals
	if len(fm1.Globals) != len(fm2.Globals) {
		t.Errorf("Globals count mismatch: %d vs %d", len(fm1.Globals), len(fm2.Globals))
	}

	for name, val1 := range fm1.Globals {
		val2, ok := fm2.Globals[name]
		if !ok {
			t.Errorf("Missing global %s in reparsed document", name)
		} else if val1 != val2 {
			t.Errorf("Global %s mismatch: %q vs %q", name, val1, val2)
		}
	}
}
