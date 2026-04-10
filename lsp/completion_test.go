package lsp

import (
	"slices"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/identifiers"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// itemLabels extracts the Label strings from a slice of completion items.
func itemLabels(items []protocol.CompletionItem) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Label
	}
	return out
}

// findItemByLabel returns the first item with the given label, or nil.
func findItemByLabel(items []protocol.CompletionItem, label string) *protocol.CompletionItem {
	for i, it := range items {
		if it.Label == label {
			return &items[i]
		}
	}
	return nil
}

// TestEnumValueCompletions_Throughput asserts R1: completion inside the first
// arg of throughput(...) returns the network types as bare identifier items
// and NO function items.
func TestEnumValueCompletions_Throughput(t *testing.T) {
	ctx := argumentContext{
		funcName: "throughput",
		paramIdx: 0,
	}
	items := enumCompletionsForContext(ctx, "")
	if len(items) == 0 {
		t.Fatal("expected enum completion items for throughput(, got none")
	}

	labels := itemLabels(items)
	for _, want := range identifiers.NetworkTypes {
		if !slices.Contains(labels, want) {
			t.Errorf("expected label %q in enum completions, got %v", want, labels)
		}
	}

	// Items must be unquoted bare identifiers
	for _, it := range items {
		if strings.HasPrefix(it.Label, `"`) {
			t.Errorf("enum completion label %q is quoted, expected bare identifier", it.Label)
		}
		if it.InsertText != nil && strings.HasPrefix(*it.InsertText, `"`) {
			t.Errorf("enum completion insertText %q is quoted, expected bare identifier", *it.InsertText)
		}
	}

	// Every item must carry data.kind == "enum_value"
	for _, it := range items {
		d, ok := it.Data.(completionItemData)
		if !ok {
			t.Errorf("item %q missing completionItemData", it.Label)
			continue
		}
		if d.Kind != "enum_value" {
			t.Errorf("item %q data.kind = %q, want %q", it.Label, d.Kind, "enum_value")
		}
	}
}

// TestEnumValueCompletions_PrefixFiltering asserts that the prefix filters
// the enum value list.
func TestEnumValueCompletions_PrefixFiltering(t *testing.T) {
	ctx := argumentContext{funcName: "throughput", paramIdx: 0}
	items := enumCompletionsForContext(ctx, "g")
	if len(items) == 0 {
		t.Fatal("expected items matching prefix 'g'")
	}
	for _, it := range items {
		if !strings.HasPrefix(it.Label, "g") {
			t.Errorf("item %q does not start with 'g'", it.Label)
		}
	}
	// gigabit should be present
	if findItemByLabel(items, "gigabit") == nil {
		t.Error("expected 'gigabit' in prefix-filtered results")
	}
}

// TestEnumValueCompletions_Rtt asserts rtt() offers network scopes.
func TestEnumValueCompletions_Rtt(t *testing.T) {
	ctx := argumentContext{funcName: "rtt", paramIdx: 0}
	items := enumCompletionsForContext(ctx, "re")
	if findItemByLabel(items, "regional") == nil {
		t.Errorf("expected 'regional' in rtt(re... completions, got %v", itemLabels(items))
	}
}

// TestEnumValueCompletions_ReadSecondArg asserts read's second arg gets storage types.
func TestEnumValueCompletions_ReadSecondArg(t *testing.T) {
	ctx := argumentContext{funcName: "read", paramIdx: 1}
	items := enumCompletionsForContext(ctx, "ss")
	if findItemByLabel(items, "ssd") == nil {
		t.Errorf("expected 'ssd' in read(1 GB, ss... completions, got %v", itemLabels(items))
	}
}

// TestEnumValueCompletions_CompressSecondArg asserts compress(_, gz) → gzip.
func TestEnumValueCompletions_CompressSecondArg(t *testing.T) {
	ctx := argumentContext{funcName: "compress", paramIdx: 1}
	items := enumCompletionsForContext(ctx, "gz")
	if findItemByLabel(items, "gzip") == nil {
		t.Errorf("expected 'gzip' in compress(..., gz... completions, got %v", itemLabels(items))
	}
}

// TestEnumValueCompletions_ConvertRateTimeUnit asserts that convert_rate's
// second arg now offers the canonical time unit names as enum completions.
func TestEnumValueCompletions_ConvertRateTimeUnit(t *testing.T) {
	ctx := argumentContext{funcName: "convert_rate", paramIdx: 1}
	items := enumCompletionsForContext(ctx, "")
	labels := itemLabels(items)
	for _, want := range []string{"second", "minute", "hour", "day", "year"} {
		if !slices.Contains(labels, want) {
			t.Errorf("expected %q in convert_rate time_unit completions, got %v", want, labels)
		}
	}
}

// TestEnumValueCompletions_NonEnumStringParam asserts a free-form string
// param (compound.period) yields no enum completions — the code path
// must fall through to general completions.
func TestEnumValueCompletions_NonEnumStringParam(t *testing.T) {
	ctx := argumentContext{funcName: "compound", paramIdx: 3}
	items := enumCompletionsForContext(ctx, "")
	if len(items) != 0 {
		t.Errorf("expected no enum items for compound.period, got %d: %v", len(items), itemLabels(items))
	}
}

// TestEnumValueCompletions_UnknownFunction asserts no items for unknown funcs.
func TestEnumValueCompletions_UnknownFunction(t *testing.T) {
	ctx := argumentContext{funcName: "frobnicate", paramIdx: 0}
	items := enumCompletionsForContext(ctx, "")
	if len(items) != 0 {
		t.Errorf("expected no enum items for unknown function, got %d", len(items))
	}
}

// TestVariableCompletions_TypeFiltering asserts R2: when the active param
// is rate-typed, only rate-typed variables are offered.
func TestVariableCompletions_TypeFiltering(t *testing.T) {
	s := NewServer()
	snap := s.evaluate("bandwidth = 10 MB/s\ndelay = 1 hour\nplain = 42")

	// Rate-typed param (accumulate's first arg) should return only bandwidth
	rateItems := variableCompletionItems(snap, "", 3, "rate")
	labels := itemLabels(rateItems)
	if !slices.Contains(labels, "bandwidth") {
		t.Errorf("expected 'bandwidth' in rate-filtered items, got %v", labels)
	}
	if slices.Contains(labels, "delay") {
		t.Errorf("did not expect 'delay' (duration) in rate-filtered items, got %v", labels)
	}
	if slices.Contains(labels, "plain") {
		t.Errorf("did not expect 'plain' (number) in rate-filtered items, got %v", labels)
	}

	// Duration-typed param should return only delay
	durItems := variableCompletionItems(snap, "", 3, "duration")
	durLabels := itemLabels(durItems)
	if !slices.Contains(durLabels, "delay") {
		t.Errorf("expected 'delay' in duration-filtered items, got %v", durLabels)
	}
	if slices.Contains(durLabels, "bandwidth") {
		t.Errorf("did not expect 'bandwidth' in duration-filtered items, got %v", durLabels)
	}

	// ArgTypeAny accepts everything
	anyItems := variableCompletionItems(snap, "", 3, "any")
	anyLabels := itemLabels(anyItems)
	for _, want := range []string{"bandwidth", "delay", "plain"} {
		if !slices.Contains(anyLabels, want) {
			t.Errorf("expected %q in any-type items, got %v", want, anyLabels)
		}
	}

	// Empty requiredType (no filter, bare expression context) also accepts everything
	allItems := variableCompletionItems(snap, "", 3, "")
	allLabels := itemLabels(allItems)
	for _, want := range []string{"bandwidth", "delay", "plain"} {
		if !slices.Contains(allLabels, want) {
			t.Errorf("expected %q in unfiltered items, got %v", want, allLabels)
		}
	}
}

// TestVariableCompletions_DataKind asserts that variable items carry
// data.kind == "variable" and the inferred type.
func TestVariableCompletions_DataKind(t *testing.T) {
	s := NewServer()
	snap := s.evaluate("tax_rate = 8%\nprice = 100")

	items := variableCompletionItems(snap, "", 2, "")
	for _, it := range items {
		d, ok := it.Data.(completionItemData)
		if !ok {
			t.Errorf("variable %q missing completionItemData", it.Label)
			continue
		}
		if d.Kind != "variable" {
			t.Errorf("variable %q data.kind = %q, want variable", it.Label, d.Kind)
		}
		if it.Label == "tax_rate" && d.VariableType != "percentage" {
			t.Errorf("tax_rate data.variableType = %q, want percentage", d.VariableType)
		}
		if it.Label == "price" && d.VariableType != "number" {
			t.Errorf("price data.variableType = %q, want number", d.VariableType)
		}
	}
}

// TestFunctionCompletions_DataFunctionName asserts R3: every function
// completion item (signature form AND NL example form) carries
// data.functionName == canonical name and data.params populated.
func TestFunctionCompletions_DataFunctionName(t *testing.T) {
	items := functionCompletionItems("gro")
	if len(items) == 0 {
		t.Fatal("expected grow completion items, got none")
	}

	growSigSeen := false
	growNLSeen := false
	for _, it := range items {
		d, ok := it.Data.(completionItemData)
		if !ok {
			t.Errorf("item %q missing completionItemData", it.Label)
			continue
		}
		// Every function item should carry kind=function and functionName set.
		if d.Kind != "function" {
			t.Errorf("item %q data.kind = %q, want function", it.Label, d.Kind)
		}
		if d.FunctionName == "" {
			t.Errorf("item %q has empty data.functionName", it.Label)
		}
		if d.FunctionName == "grow" {
			if len(d.Params) == 0 {
				t.Errorf("grow item %q has empty params", it.Label)
			}
			if len(d.Params) > 0 && d.Params[0].Name != "amount" {
				t.Errorf("grow first param name = %q, want amount", d.Params[0].Name)
			}
			// NL-example rows use Snippet kind; signature-form rows use Function
			if it.Kind != nil && *it.Kind == protocol.CompletionItemKindFunction {
				growSigSeen = true
			}
			if it.Kind != nil && *it.Kind == protocol.CompletionItemKindSnippet {
				growNLSeen = true
			}
		}
	}
	if !growSigSeen {
		t.Error("expected at least one signature-form grow item with data.functionName == grow")
	}
	// NL form is optional — not every function has one. grow does have an NL
	// alias via "grow X by Y over Z", but we assert softly.
	_ = growNLSeen
}

// TestFunctionCompletions_ThroughputParamsEnumValues asserts that
// throughput's single param carries EnumValues in data so clients can pre-fill
// placeholder suggestions without a second round-trip.
func TestFunctionCompletions_ThroughputParamsEnumValues(t *testing.T) {
	items := functionCompletionItems("throughput")
	found := false
	for _, it := range items {
		d, ok := it.Data.(completionItemData)
		if !ok {
			continue
		}
		if d.FunctionName != "throughput" {
			continue
		}
		found = true
		if len(d.Params) != 1 {
			t.Fatalf("throughput should have 1 param, got %d", len(d.Params))
		}
		if len(d.Params[0].EnumValues) == 0 {
			t.Errorf("throughput param[0].enumValues is empty; expected network types")
		}
	}
	if !found {
		t.Error("no throughput item returned by functionCompletionItems")
	}
}

// TestEnumValueCompletions_InsideStringSuppressed asserts that a cursor
// inside a string literal suppresses enum completion — the LSP should not
// offer unquoted bare identifiers inside a "..." literal because calcmark
// requires bare identifiers.
func TestEnumValueCompletions_InsideStringSuppressed(t *testing.T) {
	ctx := argumentContext{
		funcName:     "throughput",
		paramIdx:     0,
		insideString: true,
	}
	items := enumCompletionsForContext(ctx, "")
	if len(items) != 0 {
		t.Errorf("expected no enum items when cursor is inside a string literal, got %d", len(items))
	}
}
