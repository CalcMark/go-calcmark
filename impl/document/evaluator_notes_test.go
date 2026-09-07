package document

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

// Informational notes ("3 days = 72 hours (rate is per hour)") ride the
// block's diagnostics as severity "info", positioned on the statement
// that produced them. They never mark the block or its variables as
// errored: the value is valid, only the assumption is being surfaced.

const notesSource = "# Wages\n\nrate = $20/hour\ndpw = 3 days\nweekly = 8 * rate * dpw\n"

func infoDiagnostics(t *testing.T, doc *document.Document) []document.Diagnostic {
	t.Helper()
	var out []document.Diagnostic
	for _, node := range doc.GetBlocks() {
		cb, ok := node.Block.(*document.CalcBlock)
		if !ok {
			continue
		}
		for _, d := range cb.Diagnostics() {
			if d.Severity == "info" {
				out = append(out, d)
			}
		}
	}
	return out
}

func TestNotes_AttachAsInfoDiagnosticOnTheStatement(t *testing.T) {
	doc, err := document.NewDocument(notesSource)
	if err != nil {
		t.Fatal(err)
	}
	if err := NewEvaluator().Evaluate(doc); err != nil {
		t.Fatalf("a note must not be an evaluation error, got %v", err)
	}
	diags := infoDiagnostics(t, doc)
	if len(diags) != 1 {
		t.Fatalf("want 1 info diagnostic, got %d: %+v", len(diags), diags)
	}
	d := diags[0]
	if d.Code != "unit_conversion" {
		t.Errorf("Code = %q, want unit_conversion", d.Code)
	}
	if !strings.HasPrefix(d.Message, "3 days = 72 hours") {
		t.Errorf("Message = %q", d.Message)
	}
	if d.Line != 3 {
		t.Errorf("Line = %d, want 3 (block-relative: the weekly line)", d.Line)
	}
	if d.DocLine != 5 {
		t.Errorf("DocLine = %d, want 5 (document-absolute)", d.DocLine)
	}
	if d.Column != 1 {
		t.Errorf("Column = %d, want 1 (whole statement)", d.Column)
	}
}

func TestNotes_ValueStillLandsAndNothingIsBlocked(t *testing.T) {
	doc, err := document.NewDocument(notesSource + "total = weekly * 2\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := NewEvaluator().Evaluate(doc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, node := range doc.GetBlocks() {
		cb, ok := node.Block.(*document.CalcBlock)
		if !ok {
			continue
		}
		for i, r := range cb.Results() {
			if r == nil {
				t.Errorf("statement %d has no value; a note must not block evaluation", i)
			}
		}
		for _, d := range cb.Diagnostics() {
			if d.Severity == "error" || d.Code == "cascading_error" {
				t.Errorf("unexpected %s diagnostic: %+v", d.Severity, d)
			}
		}
	}
}

func TestNotes_NoneWhenUnitsMatch(t *testing.T) {
	doc, err := document.NewDocument("rate = $20/hour\nx = rate * 2 hours\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := NewEvaluator().Evaluate(doc); err != nil {
		t.Fatal(err)
	}
	if diags := infoDiagnostics(t, doc); len(diags) != 0 {
		t.Errorf("want no notes, got %+v", diags)
	}
}

func TestNotes_SurviveIncrementalReevaluation(t *testing.T) {
	// EvaluateBlock is the editor's path; notes must be rebuilt, not
	// duplicated, each time the block is re-evaluated.
	doc, err := document.NewDocument(notesSource)
	if err != nil {
		t.Fatal(err)
	}
	blocks := doc.GetBlocks()
	calcID := blocks[len(blocks)-1].ID
	ev := NewEvaluator()
	for range 2 {
		if err := ev.EvaluateBlock(doc, calcID); err != nil {
			t.Fatal(err)
		}
	}
	if diags := infoDiagnostics(t, doc); len(diags) != 1 {
		t.Errorf("want exactly 1 note after re-evaluation, got %d: %+v", len(diags), diags)
	}
}
