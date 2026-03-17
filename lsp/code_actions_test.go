package lsp

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// --- extractSuggestions tests ---

func TestExtractSuggestions(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		want []string
	}{
		{
			"single suggestion",
			`undefined variable "pric" — did you mean "price"?`,
			[]string{"price"},
		},
		{
			"multiple suggestions",
			`undefined variable "pric" — did you mean one of: price, premium, priority?`,
			[]string{"price", "premium", "priority"},
		},
		{
			"no suggestions",
			`undefined variable "unknown_var"`,
			nil,
		},
		{
			"empty message",
			"",
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSuggestions(tt.msg)
			if len(got) != len(tt.want) {
				t.Fatalf("extractSuggestions(%q) = %v, want %v", tt.msg, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("suggestion[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// --- diagnosticCode tests ---

func TestDiagnosticCode(t *testing.T) {
	t.Run("nil code", func(t *testing.T) {
		diag := protocol.Diagnostic{Code: nil}
		if got := diagnosticCode(diag); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("string code", func(t *testing.T) {
		diag := protocol.Diagnostic{
			Code: &protocol.IntegerOrString{Value: "undefined_variable"},
		}
		if got := diagnosticCode(diag); got != "undefined_variable" {
			t.Errorf("got %q, want 'undefined_variable'", got)
		}
	})
}

// --- Integration test: code actions from real evaluation ---

func TestCodeAction_UndefinedVariableWithSuggestion(t *testing.T) {
	s := NewServer()
	snap := s.evaluate("price = 100\ntotal = pric * 2")

	// Find the undefined_variable diagnostic
	var undefinedDiag *protocol.Diagnostic
	for _, d := range snap.Diagnostics {
		if d.Code == "undefined_variable" {
			lspDiag := toLSPDiagnostic(d)
			undefinedDiag = &lspDiag
			break
		}
	}

	if undefinedDiag == nil {
		t.Fatal("expected undefined_variable diagnostic for 'pric'")
	}

	// Verify the diagnostic message contains a suggestion
	suggestions := extractSuggestions(undefinedDiag.Message)
	if len(suggestions) == 0 {
		t.Fatalf("expected suggestion in message: %q", undefinedDiag.Message)
	}
	if suggestions[0] != "price" {
		t.Errorf("expected suggestion 'price', got %q", suggestions[0])
	}

	// Verify the diagnostic range has precise position (not the whole line)
	if undefinedDiag.Range.Start.Character == 0 && undefinedDiag.Range.End.Character == 1000 {
		t.Error("expected precise range from AST, got full-line fallback")
	}
}

func TestCodeAction_NoDiagnosticNoAction(t *testing.T) {
	// No undefined variable → no code actions
	s := NewServer()
	snap := s.evaluate("price = 100\ntotal = price * 2")

	for _, d := range snap.Diagnostics {
		if d.Code == "undefined_variable" {
			t.Fatal("did not expect undefined_variable diagnostic")
		}
	}
}
