package lsp

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/identifiers"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// TestSignatureHelp_ActiveParameterClampedForVariadic asserts that activeParameter
// stays within the declared parameters slice for variadic functions (avg, sum)
// even when the cursor is past the declared arg count. Sending an out-of-bounds
// index violates the LSP protocol.
func TestSignatureHelp_ActiveParameterClampedForVariadic(t *testing.T) {
	// avg declares a single variadic "values" param. Cursor at the 3rd arg
	// (paramIdx=2) must clamp to index 0, not 2.
	help := signatureHelpForFunction("avg", 2, false)
	if help == nil {
		t.Fatal("expected help for avg")
	}
	if len(help.Signatures) == 0 {
		t.Fatal("expected at least one signature")
	}
	if help.ActiveParameter == nil {
		t.Fatal("expected activeParameter to be set")
	}
	params := help.Signatures[0].Parameters
	if int(*help.ActiveParameter) >= len(params) {
		t.Errorf("activeParameter %d is out of bounds for parameters length %d",
			*help.ActiveParameter, len(params))
	}
}

// TestSignatureHelp_ActiveParameterClampedForOverApplied asserts that a
// non-variadic function whose user typed too many commas still receives a
// valid (clamped) activeParameter index.
func TestSignatureHelp_ActiveParameterClampedForOverApplied(t *testing.T) {
	// accumulate declares 2 params. paramIdx=5 must clamp to 1 (last index).
	help := signatureHelpForFunction("accumulate", 5, false)
	if help == nil {
		t.Fatal("expected help for accumulate")
	}
	params := help.Signatures[0].Parameters
	if int(*help.ActiveParameter) >= len(params) {
		t.Errorf("activeParameter %d is out of bounds for parameters length %d",
			*help.ActiveParameter, len(params))
	}
}

// TestSignatureHelp_ParameterDataShape asserts R4: signatureHelp returns
// structured `data` on each parameter with type + examples.
func TestSignatureHelp_ParameterDataShape(t *testing.T) {
	help := signatureHelpForFunction("accumulate", 0, false)
	if help == nil {
		t.Fatal("expected signature help for accumulate")
	}
	if len(help.Signatures) == 0 || len(help.Signatures[0].Parameters) != 2 {
		t.Fatalf("expected accumulate to have 2 parameters, got %+v", help.Signatures)
	}

	p0 := help.Signatures[0].Parameters[0]
	d0, ok := p0.Data.(wireParamData)
	if !ok {
		t.Fatalf("parameter[0].Data is not wireParamData, got %T", p0.Data)
	}
	if d0.Type != types.ArgTypeRate {
		t.Errorf("parameter[0].data.type = %q, want rate", d0.Type)
	}
	if len(d0.Examples) == 0 {
		t.Error("parameter[0].data.examples is empty")
	}

	p1 := help.Signatures[0].Parameters[1]
	d1, ok := p1.Data.(wireParamData)
	if !ok {
		t.Fatalf("parameter[1].Data is not wireParamData, got %T", p1.Data)
	}
	if d1.Type != types.ArgTypeDuration {
		t.Errorf("parameter[1].data.type = %q, want duration", d1.Type)
	}
}

// TestSignatureHelp_GrowActiveParam asserts R4 directly: for grow(100, |, 5),
// activeParameter == 1 and parameters[1].data.type matches the spec.
func TestSignatureHelp_GrowActiveParam(t *testing.T) {
	help := signatureHelpForFunction("grow", 1, false)
	if help == nil {
		t.Fatal("expected signature help for grow")
	}
	if help.ActiveParameter == nil || *help.ActiveParameter != 1 {
		var got any
		if help.ActiveParameter != nil {
			got = *help.ActiveParameter
		}
		t.Errorf("activeParameter = %v, want 1", got)
	}
	params := help.Signatures[0].Parameters
	if len(params) < 2 {
		t.Fatalf("expected grow to have >=2 params, got %d", len(params))
	}
	d, ok := params[1].Data.(wireParamData)
	if !ok {
		t.Fatalf("parameters[1].Data is not wireParamData")
	}
	// grow's second param (increment) is ArgTypeAdditive — accepts
	// Number, Quantity, or Currency, but excludes Percentage so the
	// completion dropdown doesn't surface percentage variables.
	if d.Type != types.ArgTypeAdditive {
		t.Errorf("grow.increment.data.type = %q, want additive", d.Type)
	}
}

// TestSignatureHelp_GrowSyntaxMatchesParamSpecNames asserts that the
// signature label uses the same vocabulary as the ParamSpec — no
// drift between the registry's hand-written `Syntax` and the actual
// param names. Drift produced confusing help where the panel header
// said `grow(starting_amount, increment, duration)` while the
// param-info row read `amount` and the description said `periods`.
// User-flagged 2026-04-26.
func TestSignatureHelp_GrowSyntaxMatchesParamSpecNames(t *testing.T) {
	help := signatureHelpForFunction("grow", 0, false)
	if help == nil {
		t.Fatal("expected signature help for grow")
	}
	label := help.Signatures[0].Label
	for _, want := range []string{"amount", "increment", "periods"} {
		if !strings.Contains(label, want) {
			t.Errorf("signature label %q missing param name %q", label, want)
		}
	}
	for _, badName := range []string{"starting_amount", "duration"} {
		if strings.Contains(label, badName) {
			t.Errorf("signature label %q still contains stale param name %q", label, badName)
		}
	}
}

// TestSignatureHelp_NLContextUsesAliasExample asserts that when the
// LSP detects an NL-form call (e.g., `grow 100 by 20 over 5 months`),
// the signature label echoes that natural-language template instead
// of the paren-form Syntax. Without this, the help panel shows
// `grow(amount, increment, periods)` while the user is typing the
// NL alias — confusing because the cursor is in a totally different
// syntactic context. User-flagged 2026-04-26.
func TestSignatureHelp_NLContextUsesAliasExample(t *testing.T) {
	help := signatureHelpForFunction("grow", 0, true /* isNL */)
	if help == nil {
		t.Fatal("expected signature help for grow in NL context")
	}
	label := help.Signatures[0].Label
	if !strings.Contains(label, "by") || !strings.Contains(label, "over") {
		t.Errorf("NL signature label %q missing the alias keywords (by/over)", label)
	}
	if strings.Contains(label, "(") {
		t.Errorf("NL signature label %q should not include parens (paren-form leaked)", label)
	}
}

// TestSignatureHelp_EnumParamCarriesValues asserts that the structured data
// on throughput's single parameter includes the EnumValues list.
func TestSignatureHelp_EnumParamCarriesValues(t *testing.T) {
	help := signatureHelpForFunction("throughput", 0, false)
	if help == nil {
		t.Fatal("expected signature help for throughput")
	}
	p := help.Signatures[0].Parameters[0]
	d, ok := p.Data.(wireParamData)
	if !ok {
		t.Fatalf("parameters[0].Data is not wireParamData")
	}
	if !reflect.DeepEqual(d.EnumValues, identifiers.NetworkTypes) {
		t.Errorf("throughput.data.enumValues = %v, want %v", d.EnumValues, identifiers.NetworkTypes)
	}
}

// TestSignatureHelp_NLFormGrow verifies the full signatureHelp path for
// NL-form "grow 100 by 20 over 5 months" — extractArgumentContext returns
// a valid context and signatureHelpForFunction produces the correct response.
func TestSignatureHelp_NLFormGrow(t *testing.T) {
	cases := []struct {
		name      string
		line      string
		col       int
		wantParam int
	}{
		{
			name:      "cursor on first literal",
			line:      "grow 100 by 20 over 5 months",
			col:       len([]rune("grow 1")),
			wantParam: 0,
		},
		{
			name:      "cursor on second literal",
			line:      "grow 100 by 20 over 5 months",
			col:       len([]rune("grow 100 by 2")),
			wantParam: 1,
		},
		{
			name:      "cursor on third literal",
			line:      "grow 100 by 20 over 5 months",
			col:       len([]rune("grow 100 by 20 over 5")),
			wantParam: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := extractArgumentContext(tc.line, tc.col)
			if ctx.funcName != "grow" {
				t.Fatalf("funcName = %q, want grow", ctx.funcName)
			}
			help := signatureHelpForFunction(ctx.funcName, ctx.paramIdx, ctx.isNL)
			if help == nil {
				t.Fatal("expected non-nil signatureHelp")
			}
			if help.ActiveParameter == nil {
				t.Fatal("expected activeParameter to be set")
			}
			if int(*help.ActiveParameter) != tc.wantParam {
				t.Errorf("activeParameter = %d, want %d", *help.ActiveParameter, tc.wantParam)
			}
			if !strings.Contains(help.Signatures[0].Label, "grow") {
				t.Errorf("signature label = %q, want to contain grow", help.Signatures[0].Label)
			}
		})
	}
}

// TestSignatureHelp_NLFormCompoundWithAssignment verifies NL signatureHelp
// for "goal = compound $1000 by 5% monthly over 10 years" with assignment prefix.
func TestSignatureHelp_NLFormCompoundWithAssignment(t *testing.T) {
	line := "goal = compound $1000 by 5% monthly over 10 years"

	ctx := extractArgumentContext(line, len([]rune("goal = compound $10")))
	if ctx.funcName != "compound" {
		t.Fatalf("funcName = %q, want compound", ctx.funcName)
	}
	if ctx.paramIdx != 0 {
		t.Errorf("paramIdx = %d, want 0", ctx.paramIdx)
	}

	help := signatureHelpForFunction(ctx.funcName, ctx.paramIdx, ctx.isNL)
	if help == nil {
		t.Fatal("expected non-nil signatureHelp for compound NL")
	}
	if !strings.Contains(help.Signatures[0].Label, "compound") {
		t.Errorf("label = %q, want to contain compound", help.Signatures[0].Label)
	}
}

// TestSignatureHelp_JSONShape asserts the wire format: marshaling the result
// yields the expected top-level keys and nested structure.
func TestSignatureHelp_JSONShape(t *testing.T) {
	help := signatureHelpForFunction("accumulate", 1, false)
	if help == nil {
		t.Fatal("expected help")
	}
	b, err := json.Marshal(help)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)

	// Substrings are anchored to their wire-level containers so a
	// regression that mis-nests 'data' or displaces a field can't pass.
	wantSubstrings := []string{
		`"signatures":[`,
		`"activeSignature"`,
		`"activeParameter":1`,
		`"parameters":[`,
		`"label":"rate"`,
		`"label":"duration"`,
		`"data":{`,
		`"type":"rate"`,
		`"type":"duration"`,
		`"examples":[`,
		// Documentation must still be present for backward compat
		`"documentation":{`,
	}
	for _, want := range wantSubstrings {
		if !strings.Contains(s, want) {
			t.Errorf("json output missing %q\nfull: %s", want, s)
		}
	}
}
