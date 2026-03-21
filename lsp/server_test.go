package lsp

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// --- Position conversion tests ---

func TestToLSPPosition(t *testing.T) {
	tests := []struct {
		name string
		pos  ast.Position
		want protocol.Position
	}{
		{
			name: "standard 1-indexed to 0-indexed",
			pos:  ast.Position{Line: 1, Column: 1},
			want: protocol.Position{Line: 0, Character: 0},
		},
		{
			name: "multi-line multi-column",
			pos:  ast.Position{Line: 5, Column: 10},
			want: protocol.Position{Line: 4, Character: 9},
		},
		{
			name: "zero values clamp to 0",
			pos:  ast.Position{Line: 0, Column: 0},
			want: protocol.Position{Line: 0, Character: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToLSPPosition(tt.pos)
			if got.Line != tt.want.Line || got.Character != tt.want.Character {
				t.Errorf("ToLSPPosition(%+v) = %+v, want %+v", tt.pos, got, tt.want)
			}
		})
	}
}

func TestToLSPRange(t *testing.T) {
	t.Run("nil range returns zero range", func(t *testing.T) {
		got := ToLSPRange(nil)
		if got.Start.Line != 0 || got.End.Line != 0 {
			t.Errorf("ToLSPRange(nil) = %+v, want zero range", got)
		}
	})

	t.Run("valid range converts correctly", func(t *testing.T) {
		r := &ast.Range{
			Start: ast.Position{Line: 1, Column: 5},
			End:   ast.Position{Line: 1, Column: 10},
		}
		got := ToLSPRange(r)
		if got.Start.Line != 0 || got.Start.Character != 4 || got.End.Character != 9 {
			t.Errorf("ToLSPRange(%+v) = %+v", r, got)
		}
	})
}

// --- Evaluate tests ---

func TestEvaluate_SimpleAssignment(t *testing.T) {
	s := NewServer()
	snap := s.evaluate("a = 1 + 1")

	if snap.Document == nil {
		t.Fatal("expected document to be parsed")
	}
	if snap.Evaluator == nil {
		t.Fatal("expected evaluator to be present")
	}
	if len(snap.Diagnostics) != 0 {
		t.Errorf("expected no diagnostics, got %d: %v", len(snap.Diagnostics), snap.Diagnostics)
	}
}

func TestEvaluate_DivisionByZero(t *testing.T) {
	s := NewServer()
	snap := s.evaluate("a = 1 / 0")

	if len(snap.Diagnostics) == 0 {
		t.Fatal("expected at least one diagnostic for division by zero")
	}

	// The diagnostic may come as "division_by_zero" or "eval_error" depending
	// on whether it's caught by the semantic checker or the runtime.
	found := false
	for _, d := range snap.Diagnostics {
		if strings.Contains(strings.ToLower(d.Message), "division by zero") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected diagnostic mentioning division by zero, got: %v", snap.Diagnostics)
	}
}

func TestEvaluate_UndefinedVariable(t *testing.T) {
	s := NewServer()
	snap := s.evaluate("b = unknown_var * 2")

	found := false
	for _, d := range snap.Diagnostics {
		if d.Code == "undefined_variable" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected undefined_variable diagnostic, got: %v", snap.Diagnostics)
	}
}

func TestEvaluate_PanicRecovery(t *testing.T) {
	// The evaluate function should never panic — verify the recover() works.
	// We can't easily trigger a panic from normal input, but we can verify
	// the structure handles edge cases without crashing.
	s := NewServer()

	// Empty input should not panic
	snap := s.evaluate("")
	if snap == nil {
		t.Fatal("expected non-nil snapshot for empty input")
	}
}

func TestEvaluate_DocumentSizeLimit(t *testing.T) {
	// Documents over 1MB should be rejected at the didOpen level.
	// The evaluate function itself doesn't enforce the limit — that's
	// in the didOpen handler. But we verify large docs can be evaluated
	// without hanging (timeout protects us).
	s := NewServer()
	// 1000 lines is large but not over the limit
	lines := make([]string, 1000)
	for i := range lines {
		lines[i] = "x = 1"
	}
	snap := s.evaluate(strings.Join(lines, "\n"))
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}
}

func TestEvaluate_ProducesHTML(t *testing.T) {
	s := NewServer()
	snap := s.evaluate("a = 1 + 1")
	snap.HTML = s.renderHTML(snap)

	if snap.HTML == "" {
		t.Fatal("expected non-empty HTML")
	}
	if !strings.Contains(snap.HTML, "calc-block") {
		t.Error("expected HTML to contain 'calc-block'")
	}
	if !strings.Contains(snap.HTML, "data-source-line") {
		t.Error("expected HTML to contain 'data-source-line'")
	}
}

// --- Diagnostic mapping tests ---

func TestToLSPSeverity(t *testing.T) {
	tests := []struct {
		input string
		want  protocol.DiagnosticSeverity
	}{
		{"Error", protocol.DiagnosticSeverityError},
		{"error", protocol.DiagnosticSeverityError},
		{"Warning", protocol.DiagnosticSeverityWarning},
		{"warning", protocol.DiagnosticSeverityWarning},
		{"Hint", protocol.DiagnosticSeverityHint},
		{"hint", protocol.DiagnosticSeverityHint},
		{"unknown", protocol.DiagnosticSeverityInformation},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toLSPSeverity(tt.input)
			if got != tt.want {
				t.Errorf("toLSPSeverity(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// --- Completion helper tests ---

func TestExtractPrefix(t *testing.T) {
	tests := []struct {
		name     string
		lineText string
		col      int
		want     string
	}{
		{"empty line", "", 0, ""},
		{"start of identifier", "abc = 1", 3, "abc"},
		{"mid identifier", "price = 100", 3, "pri"},
		{"after operator", "a = b", 5, "b"},
		{"after space", "a = ", 4, ""},
		{"col beyond line", "abc", 10, "abc"},
		// Directive prefixes
		{"@s directive prefix", "a = @s", 6, "@s"},
		{"@scale full", "@scale", 6, "@scale"},
		{"@globals.tax_rate", "@globals.tax_rate", 17, "@globals.tax_rate"},
		{"@globals. two-stage", "@globals.", 9, "@globals."},
		{"email@example not directive", "email@example", 13, "example"},
		{"space then @s", "a + @s", 6, "@s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPrefix(tt.lineText, tt.col)
			if got != tt.want {
				t.Errorf("extractPrefix(%q, %d) = %q, want %q", tt.lineText, tt.col, got, tt.want)
			}
		})
	}
}

func TestIsMarkdownLine(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"# Heading", true},
		{"## Sub heading", true},
		{"> Blockquote", true},
		{"- List item", true},
		{"* Bold list", true},
		{"a = 1 + 1", false},
		{"price = 100 USD", false},
		{"avg(1, 2, 3)", false},
		{"", true},
		{"   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got := isMarkdownLine(tt.line)
			if got != tt.want {
				t.Errorf("isMarkdownLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestClassifyCompletionContext(t *testing.T) {
	tests := []struct {
		name string
		line string
		col  int
		want completionContext
	}{
		{"after in space", "100 meters in ", 14, completionContextAfterUnitKeyword},
		{"after in typing", "100 meters in fe", 16, completionContextAfterUnitKeyword},
		{"after as space", "price as ", 9, completionContextAfterUnitKeyword},
		{"after as typing napkin", "price as nap", 12, completionContextAfterUnitKeyword},
		{"general assignment", "a = 1 + 1", 9, completionContextGeneral},
		{"general identifier", "acc", 3, completionContextGeneral},
		{"markdown heading", "# Heading", 9, completionContextMarkdown},
		{"markdown list", "- item", 6, completionContextMarkdown},
		{"empty line", "", 0, completionContextMarkdown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyCompletionContext(tt.line, tt.col)
			if got != tt.want {
				t.Errorf("classifyCompletionContext(%q, %d) = %v, want %v", tt.line, tt.col, got, tt.want)
			}
		})
	}
}

// --- Hover helper tests ---

func TestExtractWordAt(t *testing.T) {
	tests := []struct {
		name string
		line string
		col  int
		want string
	}{
		{"at variable", "price = 100", 2, "price"},
		{"at number boundary", "price = 100", 8, "100"},
		{"at operator", "a = b", 2, ""},
		{"empty line", "", 0, ""},
		{"underscore identifier", "my_var = 1", 3, "my_var"},
		// UTF-8: col is in rune positions, not bytes
		{"unicode identifier", "café = 1", 2, "café"},
		{"after unicode", "café = 1", 7, "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractWordAt(tt.line, tt.col)
			if got != tt.want {
				t.Errorf("extractWordAt(%q, %d) = %q, want %q", tt.line, tt.col, got, tt.want)
			}
		})
	}
}

func TestIsIdentifier(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"abc", true},
		{"_var", true},
		{"my_var", true},
		{"var123", true},
		{"123", false},
		{"", false},
		{"a b", false},
		{"a=b", false},
		// Unicode identifiers (CalcMark supports these)
		{"café", true},
		{"日本語", true},
		{"π", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isIdentifier(tt.input)
			if got != tt.want {
				t.Errorf("isIdentifier(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// --- Integration-level tests: evaluate → diagnostics → completions ---

func TestCompletionItems_FunctionsIncluded(t *testing.T) {
	items := functionCompletionItems("")
	if len(items) == 0 {
		t.Fatal("expected function completion items, got none")
	}

	// avg should be in the list
	found := false
	for _, item := range items {
		if item.Label == "avg" {
			found = true
			if item.Kind == nil || *item.Kind != protocol.CompletionItemKindFunction {
				t.Errorf("expected Function kind for avg, got %v", item.Kind)
			}
			break
		}
	}
	if !found {
		t.Error("expected avg in function completion items")
	}
}

func TestCompletionItems_UnitsIncluded(t *testing.T) {
	items := unitCompletionItems("met")
	found := false
	for _, item := range items {
		if strings.HasPrefix(strings.ToLower(item.Label), "met") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected unit items matching 'met' prefix")
	}
}

func TestCompletionItems_VariablesFromSnapshot(t *testing.T) {
	s := NewServer()
	snap := s.evaluate("price = 100\ntax = 10")

	items := variableCompletionItems(snap, "", 2)
	if len(items) == 0 {
		t.Fatal("expected variable completion items")
	}

	found := false
	for _, item := range items {
		if item.Label == "price" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'price' in variable completion items")
	}
}

func TestCompletionItems_NLFunctionsIncluded(t *testing.T) {
	items := functionCompletionItems("average")
	found := false
	for _, item := range items {
		if strings.Contains(item.Label, "average") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected NL function 'average of' in completions for 'average' prefix")
	}
}

// --- Signature help tests ---

func TestExtractFunctionContext(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		col      int
		wantFunc string
		wantIdx  int
	}{
		{"after open paren", "accumulate(", 11, "accumulate", 0},
		{"first param", "accumulate(5%", 13, "accumulate", 0},
		{"after comma", "accumulate(5%, ", 15, "accumulate", 1},
		{"second param typing", "accumulate(5%, 2 ye", 19, "accumulate", 1},
		{"nested deeper", "avg(1, 2, ", 10, "avg", 2},
		{"no function context", "a = 1 + 1", 9, "", -1},
		{"empty line", "", 0, "", -1},
		{"just identifier", "abc", 3, "", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, idx := extractFunctionContext(tt.line, tt.col)
			if fn != tt.wantFunc {
				t.Errorf("extractFunctionContext(%q, %d) func = %q, want %q", tt.line, tt.col, fn, tt.wantFunc)
			}
			if idx != tt.wantIdx {
				t.Errorf("extractFunctionContext(%q, %d) paramIdx = %d, want %d", tt.line, tt.col, idx, tt.wantIdx)
			}
		})
	}
}

func TestSignatureHelpForFunction(t *testing.T) {
	help := signatureHelpForFunction("avg", 0)
	if help == nil {
		t.Fatal("expected signature help for avg, got nil")
	}
	if len(help.Signatures) == 0 {
		t.Fatal("expected at least one signature")
	}
	if len(help.Signatures[0].Parameters) == 0 {
		t.Fatal("expected parameters in signature")
	}
	if help.ActiveParameter == nil {
		t.Fatal("expected active parameter to be set")
	}
}

func TestSignatureHelpForUnknownFunction(t *testing.T) {
	help := signatureHelpForFunction("not_a_function", 0)
	if help != nil {
		t.Error("expected nil for unknown function")
	}
}

// --- Document symbol tests ---

func TestGetLineText(t *testing.T) {
	source := "line 0\nline 1\nline 2"
	tests := []struct {
		line int
		want string
	}{
		{0, "line 0"},
		{1, "line 1"},
		{2, "line 2"},
		{-1, ""},
		{10, ""},
	}

	for _, tt := range tests {
		got := getLineText(source, tt.line)
		if got != tt.want {
			t.Errorf("getLineText(source, %d) = %q, want %q", tt.line, got, tt.want)
		}
	}
}
