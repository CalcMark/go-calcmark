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
	help := signatureHelpForFunction("avg", 2)
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
	help := signatureHelpForFunction("accumulate", 5)
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
	help := signatureHelpForFunction("accumulate", 0)
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
	help := signatureHelpForFunction("grow", 1)
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
	// grow's second param (increment) is ArgTypeAny in the spec (see D1 in plan)
	if d.Type != types.ArgTypeAny {
		t.Errorf("grow.increment.data.type = %q, want any", d.Type)
	}
}

// TestSignatureHelp_EnumParamCarriesValues asserts that the structured data
// on throughput's single parameter includes the EnumValues list.
func TestSignatureHelp_EnumParamCarriesValues(t *testing.T) {
	help := signatureHelpForFunction("throughput", 0)
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

// TestSignatureHelp_JSONShape asserts the wire format: marshaling the result
// yields the expected top-level keys and nested structure.
func TestSignatureHelp_JSONShape(t *testing.T) {
	help := signatureHelpForFunction("accumulate", 1)
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
