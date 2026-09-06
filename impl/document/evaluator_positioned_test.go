package document

import (
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

// Runtime (eval_error) diagnostics carry a column span when the
// interpreter knows which expression failed (go-calcmark#164), so
// editors can underline the bad token exactly as they do for semantic
// diagnostics. Line numbers stay block-relative for Line and
// document-absolute for DocLine/EndLine, matching the semantic path.

func evalErrorDiagnostics(t *testing.T, source string) []document.Diagnostic {
	t.Helper()
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}
	_ = NewEvaluator().Evaluate(doc)
	var out []document.Diagnostic
	for _, node := range doc.GetBlocks() {
		cb, ok := node.Block.(*document.CalcBlock)
		if !ok {
			continue
		}
		for _, d := range cb.Diagnostics() {
			if d.Code == "eval_error" {
				out = append(out, d)
			}
		}
	}
	return out
}

func TestEvalErrorDiagnostic_ArgumentTypeError_HasColumnSpan(t *testing.T) {
	// Text block first so DocLine differs from the block-relative Line.
	source := "# Title\n\nsx_grown = grow(100, 20%, 5)\n"
	diags := evalErrorDiagnostics(t, source)
	if len(diags) != 1 {
		t.Fatalf("got %d eval_error diagnostics, want 1: %+v", len(diags), diags)
	}
	d := diags[0]
	//            1234567890123456789012
	// source:   `sx_grown = grow(100, 20%, 5)` — `20%` starts at column 22.
	if d.Column != 22 {
		t.Errorf("Column = %d, want 22 (start of `20%%`)", d.Column)
	}
	if d.EndColumn <= d.Column {
		t.Errorf("EndColumn = %d, want > Column %d", d.EndColumn, d.Column)
	}
	if d.Line != 1 {
		t.Errorf("Line = %d, want 1 (block-relative)", d.Line)
	}
	if d.DocLine != 3 {
		t.Errorf("DocLine = %d, want 3 (document-absolute)", d.DocLine)
	}
	if d.EndLine != 3 {
		t.Errorf("EndLine = %d, want 3 (document-absolute, like semantic diagnostics)", d.EndLine)
	}
}

func TestEvalErrorDiagnostic_OperatorError_HasColumnSpan(t *testing.T) {
	diags := evalErrorDiagnostics(t, "x = $5 / $2\n")
	if len(diags) != 1 {
		t.Fatalf("got %d eval_error diagnostics, want 1: %+v", len(diags), diags)
	}
	d := diags[0]
	if d.Column != 5 {
		t.Errorf("Column = %d, want 5 (start of `$5 / $2`)", d.Column)
	}
	if d.EndColumn <= d.Column {
		t.Errorf("EndColumn = %d, want > Column %d", d.EndColumn, d.Column)
	}
}

func TestEvalErrorDiagnostic_UnpositionedErrorKeepsStatementStart(t *testing.T) {
	// Division by zero is reported by the operator; whichever range it
	// carries, a diagnostic must never regress to Line 0 / Column 0.
	diags := evalErrorDiagnostics(t, "x = 1 / 0\n")
	if len(diags) != 1 {
		t.Fatalf("got %d eval_error diagnostics, want 1: %+v", len(diags), diags)
	}
	if diags[0].Line != 1 || diags[0].Column < 1 {
		t.Errorf("Line/Column = %d/%d, want 1/>=1", diags[0].Line, diags[0].Column)
	}
}
