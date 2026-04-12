package lsp

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// prepareServerDoc loads a source document into a fresh server and returns
// both the server and the URI so tests can drive handlers directly.
func prepareServerDoc(t *testing.T, source string) (*Server, string) {
	t.Helper()
	s := NewServer()
	ds := &documentState{}
	ds.setSource(source)
	ds.setSnapshot(s.evaluate(source))

	uri := "test://acceptance.cm"
	s.mu.Lock()
	s.documents[uri] = ds
	s.mu.Unlock()
	return s, uri
}

func completionAt(t *testing.T, s *Server, uri string, line, col uint32) []protocol.CompletionItem {
	t.Helper()
	r, err := s.textDocumentCompletion(nil, &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: line, Character: col},
		},
	})
	if err != nil {
		t.Fatalf("completion error: %v", err)
	}
	if r == nil {
		return nil
	}
	items, ok := r.([]protocol.CompletionItem)
	if !ok {
		t.Fatalf("completion result is not []CompletionItem: %T", r)
	}
	return items
}

// TestR1_EnumCompletionInsideThroughput — cursor inside throughput(|) returns
// network type enum values as bare identifier items, not functions.
func TestR1_EnumCompletionInsideThroughput(t *testing.T) {
	source := "x = throughput()"
	s, uri := prepareServerDoc(t, source)

	// cursor between the parens: col = len("x = throughput(") = 15
	items := completionAt(t, s, uri, 0, 15)
	if len(items) == 0 {
		t.Fatal("expected enum completion items, got none")
	}

	labels := itemLabels(items)
	for _, want := range []string{"gigabit", "ten_gig", "wifi"} {
		if !slices.Contains(labels, want) {
			t.Errorf("expected %q in completion labels, got %v", want, labels)
		}
	}

	// Should NOT include functions like accumulate
	for _, l := range labels {
		if l == "accumulate" || strings.HasPrefix(l, "accumulate(") {
			t.Errorf("function %q leaked into enum completion context", l)
		}
	}
}

// TestR2_VariableTypeFilteringAccumulate — cursor inside accumulate(|, 1 hour)
// returns rate-typed variables, not duration or number ones.
func TestR2_VariableTypeFilteringAccumulate(t *testing.T) {
	source := "bandwidth = 10 MB/s\ndelay = 1 hour\nplain = 42\nresult = accumulate(, 1 hour)"
	s, uri := prepareServerDoc(t, source)

	// Line 3 is "result = accumulate(, 1 hour)" — cursor just after "("
	// col = len("result = accumulate(") = 20
	items := completionAt(t, s, uri, 3, 20)

	// Positive and negative checks are both narrowed to data.kind == "variable"
	// items so this test cannot pass vacuously from a function or unit label
	// coincidentally matching "bandwidth".
	var bandwidthVar bool
	for _, it := range items {
		d, ok := it.Data.(completionItemData)
		if !ok || d.Kind != "variable" {
			continue
		}
		switch it.Label {
		case "bandwidth":
			bandwidthVar = true
			if d.VariableType != "rate" {
				t.Errorf("bandwidth variable type = %q, want rate", d.VariableType)
			}
		case "delay":
			t.Errorf("duration-typed 'delay' leaked into rate-filtered completions")
		case "plain":
			t.Errorf("number-typed 'plain' leaked into rate-filtered completions")
		}
	}
	if !bandwidthVar {
		t.Errorf("expected rate-typed 'bandwidth' variable in completions")
	}
}

// TestR3_FunctionDataFunctionName — both signature-form and NL-example
// completion items for `grow` carry data.functionName == "grow".
func TestR3_FunctionDataFunctionName(t *testing.T) {
	source := "x = gro"
	s, uri := prepareServerDoc(t, source)

	// cursor at end of line: col = 7
	items := completionAt(t, s, uri, 0, 7)
	growItemCount := 0
	for _, it := range items {
		d, ok := it.Data.(completionItemData)
		if !ok {
			continue
		}
		if d.FunctionName == "grow" {
			growItemCount++
			if d.Kind != "function" {
				t.Errorf("grow item kind=%q, want function", d.Kind)
			}
		}
	}
	if growItemCount == 0 {
		t.Errorf("no completion items with data.functionName == grow; got %d items total", len(items))
	}
}

// TestR3b_NLCompletionDataFunctionName — NL-example completion items for
// functions with synonyms carry the canonical name, not the alias display name.
func TestR3b_NLCompletionDataFunctionName(t *testing.T) {
	source := "x = av"
	s, uri := prepareServerDoc(t, source)

	items := completionAt(t, s, uri, 0, 6)
	nlSeen := false
	for _, it := range items {
		d, ok := it.Data.(completionItemData)
		if !ok {
			continue
		}
		if it.Kind != nil && *it.Kind == protocol.CompletionItemKindSnippet && d.FunctionName == "avg" {
			nlSeen = true
			if len(d.Params) == 0 {
				t.Errorf("NL avg item %q has empty data.params", it.Label)
			}
		}
		// No item should carry the display name as functionName
		if d.FunctionName == "avg (average, mean)" {
			t.Errorf("item %q has display name as functionName: %q", it.Label, d.FunctionName)
		}
	}
	if !nlSeen {
		t.Error("expected NL-example item with data.functionName == avg")
	}
}

// TestNLSignatureHelp_GrowAcceptance — end-to-end signatureHelp for NL-form
// "grow 100 by 20 over 5 months" with cursor on each numeric literal.
func TestNLSignatureHelp_GrowAcceptance(t *testing.T) {
	source := "grow 100 by 20 over 5 months"
	s, uri := prepareServerDoc(t, source)

	cases := []struct {
		name      string
		col       uint32
		wantParam uint32
	}{
		{"cursor on 100", uint32(len([]rune("grow 1"))), 0},
		{"cursor on 20", uint32(len([]rune("grow 100 by 2"))), 1},
		{"cursor on 5", uint32(len([]rune("grow 100 by 20 over 5"))), 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params := &protocol.SignatureHelpParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: tc.col},
				},
			}
			r, err := s.signatureHelpHandle(params)
			if err != nil {
				t.Fatalf("signatureHelp error: %v", err)
			}
			if r == nil {
				t.Fatal("expected non-nil signatureHelp for NL grow")
			}
			help, ok := r.(*lspSignatureHelp)
			if !ok {
				t.Fatalf("result is not *lspSignatureHelp: %T", r)
			}
			if help.ActiveParameter == nil {
				t.Fatal("expected activeParameter to be set")
			}
			if *help.ActiveParameter != protocol.UInteger(tc.wantParam) {
				t.Errorf("activeParameter = %d, want %d", *help.ActiveParameter, tc.wantParam)
			}
			if !strings.Contains(help.Signatures[0].Label, "grow") {
				t.Errorf("signature label = %q, want to contain grow", help.Signatures[0].Label)
			}
		})
	}
}

// TestNLSignatureHelp_CompoundWithAssignment — end-to-end signatureHelp for
// "goal = compound $1000 by 5% monthly over 10 years".
func TestNLSignatureHelp_CompoundWithAssignment(t *testing.T) {
	source := "goal = compound $1000 by 5% monthly over 10 years"
	s, uri := prepareServerDoc(t, source)

	params := &protocol.SignatureHelpParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: uint32(len([]rune("goal = compound $10")))}, // cursor on "1000"
		},
	}
	r, err := s.signatureHelpHandle(params)
	if err != nil {
		t.Fatalf("signatureHelp error: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil signatureHelp for NL compound")
	}
	help, ok := r.(*lspSignatureHelp)
	if !ok {
		t.Fatalf("result is not *lspSignatureHelp: %T", r)
	}
	if help.ActiveParameter == nil || *help.ActiveParameter != 0 {
		var got any
		if help.ActiveParameter != nil {
			got = *help.ActiveParameter
		}
		t.Errorf("activeParameter = %v, want 0", got)
	}
	if !strings.Contains(help.Signatures[0].Label, "compound") {
		t.Errorf("label = %q, want to contain compound", help.Signatures[0].Label)
	}
}

// TestR4_SignatureHelpGrowSecondArg — signatureHelp at grow(100, |, 5)
// returns activeParameter=1 and parameters[1].data.type is set.
func TestR4_SignatureHelpGrowSecondArg(t *testing.T) {
	source := "goal = grow(100, , 5)"
	s, uri := prepareServerDoc(t, source)

	// cursor after "grow(100, " → col = len("goal = grow(100, ") = 17
	params := &protocol.SignatureHelpParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 17},
		},
	}
	r, err := s.signatureHelpHandle(params)
	if err != nil {
		t.Fatalf("signatureHelp error: %v", err)
	}
	help, ok := r.(*lspSignatureHelp)
	if !ok {
		t.Fatalf("result is not *lspSignatureHelp: %T", r)
	}
	if help.ActiveParameter == nil || *help.ActiveParameter != 1 {
		var got any
		if help.ActiveParameter != nil {
			got = *help.ActiveParameter
		}
		t.Errorf("activeParameter = %v, want 1", got)
	}

	// Marshal and check the wire shape
	b, err := json.Marshal(help)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	wireJSON := string(b)
	if !strings.Contains(wireJSON, `"activeParameter":1`) {
		t.Errorf("wire missing activeParameter:1\n%s", wireJSON)
	}
	// grow's increment param is ArgTypeAny in the spec. The wireParamData
	// struct emits Name first (when present), then Type.
	if !strings.Contains(wireJSON, `"name":"increment","type":"any"`) {
		t.Errorf("wire missing parameters[1].name='increment' + type='any'\n%s", wireJSON)
	}
}

// TestR5_HoverOnGrow — hover on grow in "goal = grow(100, 20, 5)" contains
// the signature, description, and at least one example.
func TestR5_HoverOnGrow(t *testing.T) {
	source := "goal = grow(100, 20, 5)"
	content := hoverContent(t, source, 0, 7)
	if content == "" {
		t.Fatal("expected hover on grow")
	}
	for _, want := range []string{"grow(", "**grow**"} {
		if !strings.Contains(content, want) {
			t.Errorf("missing %q in hover:\n%s", want, content)
		}
	}
	if !strings.Contains(content, "Example") {
		t.Errorf("hover missing Example section:\n%s", content)
	}
}

// TestR6_HoverOnPriceVariable — hover on price in "price = 100" contains
// 'number' and '100'.
func TestR6_HoverOnPriceVariable(t *testing.T) {
	source := "price = 100"
	content := hoverContent(t, source, 0, 0)
	if content == "" {
		t.Fatal("expected hover on price")
	}
	if !strings.Contains(content, "number") {
		t.Errorf("hover missing 'number':\n%s", content)
	}
	if !strings.Contains(content, "100") {
		t.Errorf("hover missing '100':\n%s", content)
	}
}
