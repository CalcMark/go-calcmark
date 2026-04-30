package format

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	implDoc "github.com/CalcMark/go-calcmark/v2/impl/document"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

// multibyteSource is the test document used across all format round-trip tests.
// It includes CJK, Latin extended, Cyrillic, BMP emoji (⭐️, ✅), SMP emoji (💰),
// and mixed markdown prose - matching the issue #12 requirements.
const multibyteSource = `# 多字节文档

a = 3
手 = a * 5
café = 100
Москва = 200
💰 = 1000
⭐️ = 手 + 23
✅ = ⭐️ * 2
total = 手 + café + Москва + 💰
`

// multibyteDocWithProse includes interleaved prose blocks for testing document structure.
const multibyteDocWithProse = `# 她鳥足飽經半方結己平向說眼虎

This is plain ascii text.

a = 3
手 = a * 5

就明細今士亮上封訴蝸花果但入東

⭐️ = 手 + 23

Lorem Ipsum คือ เนื้อหาจำลองแบบเรียบๆ

test = ⭐️ * 1K
`

// evalMultibyteDoc is a helper that parses and evaluates a multi-byte document.
func evalMultibyteDoc(t *testing.T, source string) *document.Document {
	t.Helper()
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}
	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	return doc
}

// TestMultibyteCalcMarkRoundTrip verifies that multi-byte CalcMark documents
// survive a format → reparse → re-evaluate → format cycle without data loss.
func TestMultibyteCalcMarkRoundTrip(t *testing.T) {
	// First pass: parse, evaluate, format
	doc1 := evalMultibyteDoc(t, multibyteSource)

	var buf1 bytes.Buffer
	formatter := &CalcMarkFormatter{}
	if err := formatter.Format(&buf1, doc1, Options{Verbose: false}); err != nil {
		t.Fatalf("First format failed: %v", err)
	}
	output1 := buf1.String()

	// Verify multi-byte content is preserved in output
	for _, want := range []string{"手", "café", "Москва", "💰", "⭐️", "✅"} {
		if !strings.Contains(output1, want) {
			t.Errorf("First pass output missing %q", want)
		}
	}

	// Second pass: reparse the formatted output
	doc2 := evalMultibyteDoc(t, output1)

	var buf2 bytes.Buffer
	if err := formatter.Format(&buf2, doc2, Options{Verbose: false}); err != nil {
		t.Fatalf("Second format failed: %v", err)
	}
	output2 := buf2.String()

	// Round-trip: output1 == output2
	if output1 != output2 {
		t.Errorf("Round-trip failed:\nPass 1:\n%s\nPass 2:\n%s", output1, output2)
	}

	// Verify evaluation results match after round-trip
	env1 := implDoc.NewEvaluator()
	env1.Evaluate(doc1)
	env2 := implDoc.NewEvaluator()
	env2.Evaluate(doc2)

	for _, varName := range []string{"a", "手", "café", "Москва", "💰", "⭐️", "✅", "total"} {
		v1, ok1 := env1.GetEnvironment().Get(varName)
		v2, ok2 := env2.GetEnvironment().Get(varName)
		if !ok1 || !ok2 {
			t.Errorf("Variable %q: found in pass1=%v, pass2=%v", varName, ok1, ok2)
			continue
		}
		if v1.String() != v2.String() {
			t.Errorf("Variable %q: pass1=%q, pass2=%q", varName, v1.String(), v2.String())
		}
	}
}

// TestMultibyteCalcMarkRoundTripWithProse verifies round-trip preservation
// of a document mixing multi-byte prose and calculation blocks.
func TestMultibyteCalcMarkRoundTripWithProse(t *testing.T) {
	doc := evalMultibyteDoc(t, multibyteDocWithProse)

	var buf bytes.Buffer
	formatter := &CalcMarkFormatter{}
	if err := formatter.Format(&buf, doc, Options{Verbose: false}); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Verify prose and calculations are preserved
	checks := []string{
		"她鳥足飽經半方結己平向說眼虎",
		"就明細今士亮上封訴蝸花果但入東",
		"Lorem Ipsum คือ",
		"手 = a * 5",
		"⭐️ = 手 + 23",
		"test = ⭐️ * 1K",
	}
	for _, want := range checks {
		if !strings.Contains(output, want) {
			t.Errorf("Output missing %q", want)
		}
	}

	// Reparse and verify block count matches
	doc2, err := document.NewDocument(output)
	if err != nil {
		t.Fatalf("Reparse failed: %v", err)
	}

	b1 := doc.GetBlocks()
	b2 := doc2.GetBlocks()
	if len(b1) != len(b2) {
		t.Errorf("Block count: original=%d, reparsed=%d", len(b1), len(b2))
	}
}

// TestMultibyteJSONFormat verifies that JSON format correctly captures
// multi-byte variable names and their computed values.
func TestMultibyteJSONFormat(t *testing.T) {
	doc := evalMultibyteDoc(t, multibyteSource)

	var buf bytes.Buffer
	formatter := &JSONFormatter{}
	if err := formatter.Format(&buf, doc, Options{}); err != nil {
		t.Fatalf("JSON format failed: %v", err)
	}

	// Must be valid JSON
	if !json.Valid(buf.Bytes()) {
		t.Fatalf("Invalid JSON output:\n%s", buf.String())
	}

	var result JSONDocument
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	// Collect all variables from results
	variables := make(map[string]string)
	for _, block := range result.Blocks {
		for _, r := range block.Results {
			if r.Variable != "" {
				variables[r.Variable] = r.Value
			}
		}
	}

	// Verify multi-byte variable names appear in JSON output
	expectedVars := []string{"a", "手", "café", "Москва", "💰", "⭐️", "✅", "total"}
	for _, v := range expectedVars {
		if _, ok := variables[v]; !ok {
			t.Errorf("Variable %q not found in JSON output", v)
		}
	}

	// Verify the JSON source lines contain multi-byte text
	rawJSON := buf.String()
	for _, want := range []string{"手", "café", "Москва", "💰", "⭐️", "✅"} {
		if !strings.Contains(rawJSON, want) {
			t.Errorf("JSON output missing %q in source lines", want)
		}
	}
}

// TestMultibyteJSONRoundTrip verifies that JSON output can be unmarshalled
// and the multi-byte data is preserved correctly through serialization.
func TestMultibyteJSONRoundTrip(t *testing.T) {
	doc := evalMultibyteDoc(t, multibyteSource)

	var buf bytes.Buffer
	formatter := &JSONFormatter{}
	if err := formatter.Format(&buf, doc, Options{}); err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Unmarshal
	var result JSONDocument
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Re-marshal
	remarshalled, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Re-marshal failed: %v", err)
	}

	// Verify multi-byte content survives JSON round-trip
	remarshalledStr := string(remarshalled)
	for _, want := range []string{"手", "café", "Москва", "💰", "⭐️", "✅"} {
		if !strings.Contains(remarshalledStr, want) {
			t.Errorf("JSON round-trip lost %q", want)
		}
	}
}

// TestMultibyteTextFormat verifies that text format correctly renders
// multi-byte characters in output.
func TestMultibyteTextFormat(t *testing.T) {
	doc := evalMultibyteDoc(t, multibyteSource)

	var buf bytes.Buffer
	formatter := &TextFormatter{}
	if err := formatter.Format(&buf, doc, Options{Verbose: true}); err != nil {
		t.Fatalf("Text format failed: %v", err)
	}

	output := buf.String()

	// Verify multi-byte source lines appear in text output
	for _, want := range []string{"手", "café", "Москва", "💰", "⭐️", "✅"} {
		if !strings.Contains(output, want) {
			t.Errorf("Text output missing %q", want)
		}
	}
}

// TestMultibyteMarkdownFormat verifies that markdown format preserves
// multi-byte content in code fences and prose.
func TestMultibyteMarkdownFormat(t *testing.T) {
	doc := evalMultibyteDoc(t, multibyteDocWithProse)

	var buf bytes.Buffer
	formatter := &MarkdownFormatter{}
	if err := formatter.Format(&buf, doc, Options{}); err != nil {
		t.Fatalf("Markdown format failed: %v", err)
	}

	output := buf.String()

	// Verify multi-byte content in markdown output
	for _, want := range []string{"手", "⭐️", "她鳥足飽經半方結己平向說眼虎", "Lorem Ipsum คือ"} {
		if !strings.Contains(output, want) {
			t.Errorf("Markdown output missing %q", want)
		}
	}

	// Verify calcmark code fence blocks are present
	if !strings.Contains(output, "```calcmark") {
		t.Error("Markdown output should contain calcmark code fences")
	}
}

// TestMultibyteHTMLFormat verifies that HTML format preserves
// multi-byte content and produces valid output.
func TestMultibyteHTMLFormat(t *testing.T) {
	doc := evalMultibyteDoc(t, multibyteDocWithProse)

	var buf bytes.Buffer
	formatter := &HTMLFormatter{}
	if err := formatter.Format(&buf, doc, Options{}); err != nil {
		t.Fatalf("HTML format failed: %v", err)
	}

	output := buf.String()

	// Verify multi-byte content appears in HTML
	for _, want := range []string{"手", "⭐️", "她鳥足飽經半方結己平向說眼虎"} {
		if !strings.Contains(output, want) {
			t.Errorf("HTML output missing %q", want)
		}
	}
}

// TestMultibyteAllFormatsProduceOutput verifies that every registered
// format can handle multi-byte documents without errors.
func TestMultibyteAllFormatsProduceOutput(t *testing.T) {
	doc := evalMultibyteDoc(t, multibyteSource)

	formats := []string{"text", "json", "html", "md", "cm"}
	for _, name := range formats {
		t.Run(name, func(t *testing.T) {
			formatter := GetFormatter(name, "")
			if formatter == nil {
				t.Fatalf("Formatter %q not found", name)
			}

			// Use verbose for text format (non-verbose only shows values)
			opts := Options{}
			if name == "text" {
				opts.Verbose = true
			}

			var buf bytes.Buffer
			if err := formatter.Format(&buf, doc, opts); err != nil {
				t.Fatalf("Format %q failed: %v", name, err)
			}

			output := buf.String()
			if output == "" {
				t.Errorf("Format %q produced empty output", name)
			}

			// Every format should preserve multi-byte identifiers in some form
			if !strings.Contains(output, "手") {
				t.Errorf("Format %q output missing CJK character '手'", name)
			}
		})
	}
}
