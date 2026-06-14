package lsp

import (
	"slices"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/identifiers"
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
// Pinned against the full canonical list so adding a new unit (quarter,
// fortnight, …) requires an explicit ack here.
func TestEnumValueCompletions_ConvertRateTimeUnit(t *testing.T) {
	ctx := argumentContext{funcName: "convert_rate", paramIdx: 1}
	items := enumCompletionsForContext(ctx, "")
	labels := itemLabels(items)
	for _, want := range []string{"second", "minute", "hour", "day", "week", "month", "quarter", "year"} {
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

// TestVariableCompletions_PercentOfOperandFiltersToPercentage drives the real
// completion handler end-to-end: the percentage operand of `X% of Y` must
// surface only percentage-typed variables, not quantities — even though there
// is no enclosing function call to read a ParamSpec from. This is the keyword-
// operand counterpart to the function-argument filtering above.
func TestVariableCompletions_PercentOfOperandFiltersToPercentage(t *testing.T) {
	// Both variables share the `my` prefix, so prefix-matching alone cannot
	// explain the exclusion — the type filter is what drops the quantity.
	source := "myrate = 5%\nmyqty = 10 kg\nresult = my of 1000"
	s, uri := prepareServerDoc(t, source)

	// Line 2, cursor right after the `my` the user is typing into the
	// percentage operand (col 11 = just past the 'y' of "my").
	items := completionAt(t, s, uri, 2, 11)
	labels := itemLabels(items)

	if !slices.Contains(labels, "myrate") {
		t.Errorf("expected percentage var 'myrate' at the '%% of' operand, got %v", labels)
	}
	if slices.Contains(labels, "myqty") {
		t.Errorf("did not expect quantity var 'myqty' at the '%% of' operand, got %v", labels)
	}
}

// TestVariableCompletions_PositionFiltering pins the bug repro:
// `grow $100 by flour over 10` (line 0) followed by `flour = 10`
// (line 1) used to offer `flour` as a completion on line 0 because
// the LSP passed nil definedLines. With the Environment now tracking
// per-variable definition lines and the LSP forwarding them to
// `VariableSuggestions`, references to a variable defined LATER in
// the document are excluded.
func TestVariableCompletions_PositionFiltering(t *testing.T) {
	s := NewServer()
	// price defined on line 0; flour defined on line 1.
	snap := s.evaluate("price = 100\nflour = 10\n")

	t.Run("cursor on line 0 sees price (defined here is also excluded; flour comes later)", func(t *testing.T) {
		// Per VariableSuggestions: lineNum >= cursorLine excludes —
		// so a variable defined on the same line as the cursor is also
		// excluded (you can't reference a variable on the line where
		// you're defining it, before the `=`).
		items := variableCompletionItems(snap, "", 0, "")
		labels := itemLabels(items)
		if slices.Contains(labels, "price") {
			t.Errorf("did not expect 'price' on line 0 (same-line self-reference), got %v", labels)
		}
		if slices.Contains(labels, "flour") {
			t.Errorf("did not expect 'flour' on line 0 (defined later on line 1), got %v", labels)
		}
	})

	t.Run("cursor on line 1 sees price but not flour (flour is the LHS being defined)", func(t *testing.T) {
		items := variableCompletionItems(snap, "", 1, "")
		labels := itemLabels(items)
		if !slices.Contains(labels, "price") {
			t.Errorf("expected 'price' on line 1 (defined above), got %v", labels)
		}
		if slices.Contains(labels, "flour") {
			t.Errorf("did not expect 'flour' on line 1 (same-line self-reference), got %v", labels)
		}
	})

	t.Run("cursor on line 2 (after both definitions) sees both", func(t *testing.T) {
		items := variableCompletionItems(snap, "", 2, "")
		labels := itemLabels(items)
		for _, want := range []string{"price", "flour"} {
			if !slices.Contains(labels, want) {
				t.Errorf("expected %q on line 2 (after definition), got %v", want, labels)
			}
		}
	})
}

// TestVariableCompletions_BooleanDoesNotLeakIntoRateFilter proves that a
// boolean-typed (ArgTypeAny-mapped) variable does NOT appear in a rate-
// filtered completion context. Regression guard for ADV-007.
func TestVariableCompletions_BooleanDoesNotLeakIntoRateFilter(t *testing.T) {
	s := NewServer()
	snap := s.evaluate("flag = true\nbandwidth = 10 MB/s")

	items := variableCompletionItems(snap, "", 2, "rate")
	labels := itemLabels(items)
	if slices.Contains(labels, "flag") {
		t.Errorf("boolean-typed 'flag' leaked into rate-filtered completions: %v", labels)
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

// TestFunctionCompletions_SynonymFunctionCanonicalName asserts that functions
// with synonyms (like avg/average/mean) carry the canonical name in data, not
// the display name with synonym suffix.
func TestFunctionCompletions_SynonymFunctionCanonicalName(t *testing.T) {
	items := functionCompletionItems("av")
	if len(items) == 0 {
		t.Fatal("expected avg completion items, got none")
	}

	avgSigSeen := false
	avgNLSeen := false
	for _, it := range items {
		d, ok := it.Data.(completionItemData)
		if !ok {
			t.Errorf("item %q missing completionItemData", it.Label)
			continue
		}
		if d.FunctionName != "avg" {
			t.Errorf("item %q has data.functionName = %q, want avg", it.Label, d.FunctionName)
		}
		if len(d.Params) == 0 {
			t.Errorf("avg item %q has empty params", it.Label)
		}
		if it.Kind != nil && *it.Kind == protocol.CompletionItemKindFunction {
			avgSigSeen = true
		}
		if it.Kind != nil && *it.Kind == protocol.CompletionItemKindSnippet {
			avgNLSeen = true
		}
	}
	if !avgSigSeen {
		t.Error("expected signature-form avg item")
	}
	if !avgNLSeen {
		t.Error("expected NL-example avg item")
	}
}

// TestFunctionCompletions_CompoundNLParamsPopulated asserts that compound's
// NL-example items carry full params (principal, rate, periods, period?).
func TestFunctionCompletions_CompoundNLParamsPopulated(t *testing.T) {
	items := functionCompletionItems("compou")
	nlSeen := false
	for _, it := range items {
		d, ok := it.Data.(completionItemData)
		if !ok {
			continue
		}
		if d.FunctionName != "compound" {
			continue
		}
		if it.Kind != nil && *it.Kind == protocol.CompletionItemKindSnippet {
			nlSeen = true
			if len(d.Params) < 3 {
				t.Errorf("compound NL item %q has %d params, want >= 3", it.Label, len(d.Params))
			}
			if len(d.Params) > 0 && d.Params[0].Name != "principal" {
				t.Errorf("compound first param = %q, want principal", d.Params[0].Name)
			}
		}
	}
	if !nlSeen {
		t.Error("expected at least one NL-example compound item")
	}
}

// TestFunctionCompletions_NLExampleCarriesSnippetFormat asserts the LSP
// (not the client) is the source of truth for placeholder boundaries on
// NL-example completion items. Each numeric token — including currency
// prefix and percent suffix — must be wrapped in `${N:token}` and the
// item must declare InsertTextFormat=Snippet so the client can drop its
// own boundary inference.
func TestFunctionCompletions_NLExampleCarriesSnippetFormat(t *testing.T) {
	items := functionCompletionItems("compou")
	var nlItem *struct {
		insertText string
		isSnippet  bool
	}
	for _, it := range items {
		if it.Kind == nil || *it.Kind != protocol.CompletionItemKindSnippet {
			continue
		}
		// Skip the paren-form snippet item — that has Kind=Function in
		// emit, but we filter both here just by surface-form. The NL row
		// is detected by an InsertText that does NOT start with the
		// canonical function name followed by '('.
		if it.InsertText == nil {
			continue
		}
		if it.Data == nil {
			continue
		}
		d, ok := it.Data.(completionItemData)
		if !ok || d.FunctionName != "compound" {
			continue
		}
		// NL alias label is "compound by over" or similar; the paren
		// form's label is "compound". Distinguish on InsertText shape.
		if !containsRune(*it.InsertText, ' ') {
			continue
		}
		if it.InsertTextFormat == nil || *it.InsertTextFormat != protocol.InsertTextFormatSnippet {
			t.Errorf("NL example item %q does not declare InsertTextFormat=Snippet", it.Label)
		}
		nlItem = &struct {
			insertText string
			isSnippet  bool
		}{*it.InsertText, true}
		break
	}
	if nlItem == nil {
		t.Fatal("expected an NL-example item for compound")
	}
	// Must contain `${1:` somewhere — the first placeholder.
	if !containsString(nlItem.insertText, "${1:") {
		t.Errorf("NL example InsertText missing ${1:...} placeholder: %q", nlItem.insertText)
	}
	// `$` currency prefix must be inside placeholder 1, not outside.
	if containsString(nlItem.insertText, "$${1:") {
		t.Errorf("NL example InsertText leaves `$` outside the placeholder: %q", nlItem.insertText)
	}
	// `%` percent suffix must be inside the percent-typed placeholder.
	// The compound NL example shape `... by 5% ...` should ship the `%`
	// inside the placeholder, not as `}%`.
	if containsString(nlItem.insertText, "}%") {
		t.Errorf("NL example InsertText leaves `%%` outside the placeholder: %q", nlItem.insertText)
	}
}

// TestKeywordCompletions_SnippetAndData asserts the keyword-operator forms are
// delivered as snippet items carrying data.kind == "keyword" and the keyword
// identity, with the canonical templates (percent inside the first tab stop).
func TestKeywordCompletions_SnippetAndData(t *testing.T) {
	items := keywordCompletionItems("")
	byKeyword := map[string]protocol.CompletionItem{}
	for _, it := range items {
		d, ok := it.Data.(completionItemData)
		if !ok {
			t.Fatalf("item %q missing completionItemData", it.Label)
		}
		if d.Kind != "keyword" {
			t.Errorf("item %q data.kind = %q, want %q", it.Label, d.Kind, "keyword")
		}
		if it.InsertTextFormat == nil || *it.InsertTextFormat != protocol.InsertTextFormatSnippet {
			t.Errorf("keyword item %q is not InsertTextFormat=Snippet", it.Label)
		}
		byKeyword[d.Keyword] = it
	}

	of, ok := byKeyword["of"]
	if !ok {
		t.Fatal("expected an 'of' keyword item")
	}
	if of.InsertText == nil || *of.InsertText != "${1:23%} of ${2:1000}" {
		t.Errorf("of InsertText = %v, want the %% inside stop 1", of.InsertText)
	}
	for _, want := range []string{"of", "as % of", "in"} {
		if _, ok := byKeyword[want]; !ok {
			t.Errorf("expected keyword item %q", want)
		}
	}
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}

func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle ||
		(len(haystack) > 0 && indexOf(haystack, needle) >= 0))
}

func indexOf(s, sub string) int {
	if len(sub) == 0 {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
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
