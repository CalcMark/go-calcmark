package format

import (
	"testing"

	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
)

func TestAlignResults(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		checks func(t *testing.T, stmts []AlignedStatement)
	}{
		{
			name:  "simple expression",
			input: "2 + 2\n",
			checks: func(t *testing.T, stmts []AlignedStatement) {
				if len(stmts) != 1 {
					t.Fatalf("got %d statements, want 1", len(stmts))
				}
				if stmts[0].Result == nil {
					t.Fatal("expected non-nil result")
				}
				if stmts[0].IsBlank || stmts[0].IsResultLine {
					t.Fatal("should not be blank or result line")
				}
			},
		},
		{
			name:  "assignment extracts variable",
			input: "x = 10\n",
			checks: func(t *testing.T, stmts []AlignedStatement) {
				if len(stmts) != 1 {
					t.Fatalf("got %d statements, want 1", len(stmts))
				}
				if stmts[0].Variable != "x" {
					t.Errorf("Variable = %q, want %q", stmts[0].Variable, "x")
				}
			},
		},
		{
			name:  "blank lines marked correctly",
			input: "x = 10\n\ny = 20\n",
			checks: func(t *testing.T, stmts []AlignedStatement) {
				if len(stmts) != 3 {
					t.Fatalf("got %d statements, want 3", len(stmts))
				}
				if !stmts[1].IsBlank {
					t.Error("middle statement should be blank")
				}
				if stmts[1].Result != nil {
					t.Error("blank line should have nil result")
				}
				// Non-blank lines should have results
				if stmts[0].Result == nil {
					t.Error("first statement should have result")
				}
				if stmts[2].Result == nil {
					t.Error("third statement should have result")
				}
			},
		},
		{
			name:  "does not extract variable from comparison",
			input: "5 == 5\n",
			checks: func(t *testing.T, stmts []AlignedStatement) {
				if len(stmts) != 1 {
					t.Fatalf("got %d statements, want 1", len(stmts))
				}
				if stmts[0].Variable != "" {
					t.Errorf("Variable = %q, want empty (== is comparison)", stmts[0].Variable)
				}
			},
		},
		{
			name:  "expression with no variable extraction on arithmetic",
			input: "2 + 3 * 4\n",
			checks: func(t *testing.T, stmts []AlignedStatement) {
				if len(stmts) != 1 {
					t.Fatalf("got %d statements, want 1", len(stmts))
				}
				if stmts[0].Variable != "" {
					t.Errorf("Variable = %q, want empty (no assignment)", stmts[0].Variable)
				}
				if stmts[0].Result == nil {
					t.Error("expected non-nil result")
				}
			},
		},
		{
			name:  "multiple expressions maintain alignment",
			input: "a = 1\nb = 2\nc = a + b\n",
			checks: func(t *testing.T, stmts []AlignedStatement) {
				if len(stmts) != 3 {
					t.Fatalf("got %d statements, want 3", len(stmts))
				}
				for i, s := range stmts {
					if s.Result == nil {
						t.Errorf("statement %d should have result", i)
					}
				}
				if stmts[0].Variable != "a" {
					t.Errorf("stmts[0].Variable = %q, want %q", stmts[0].Variable, "a")
				}
				if stmts[1].Variable != "b" {
					t.Errorf("stmts[1].Variable = %q, want %q", stmts[1].Variable, "b")
				}
				if stmts[2].Variable != "c" {
					t.Errorf("stmts[2].Variable = %q, want %q", stmts[2].Variable, "c")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := document.NewDocument(tt.input)
			if err != nil {
				t.Fatalf("NewDocument: %v", err)
			}
			eval := implDoc.NewEvaluator()
			if err := eval.Evaluate(doc); err != nil {
				t.Fatalf("Evaluate: %v", err)
			}

			blocks := doc.GetBlocks()
			var calcBlock *document.CalcBlock
			for _, node := range blocks {
				if cb, ok := node.Block.(*document.CalcBlock); ok {
					calcBlock = cb
					break
				}
			}
			if calcBlock == nil {
				t.Fatal("no calc block found")
			}

			stmts := AlignResults(calcBlock)
			tt.checks(t, stmts)
		})
	}
}
