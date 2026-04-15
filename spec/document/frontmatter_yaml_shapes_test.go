package document

// YAML-shape corner-case tests for ParseFrontmatter + KeyRanges.
//
// Each test probes a specific YAML syntactic construct (anchors, tagged
// scalars, flow mappings, block scalars, etc.) and asserts the observed
// behavior. The bar is "no panic, no silent data-loss, KeyRanges is either
// sensibly populated or documented-imperfect for exotic shapes". KeyRanges
// is currently consumed only by the LSP for anchoring diagnostics/hovers;
// imperfect End.Line for multi-line block scalars is acceptable and
// acknowledged in collectKeyRanges / nodeEndPosition in frontmatter.go.

import (
	"testing"
)

// helper: parse and assert no panic / no error / non-nil fm.
func parseOrFatal(t *testing.T, source string) *Frontmatter {
	t.Helper()
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("ParseFrontmatter: unexpected error: %v", err)
	}
	if fm == nil {
		t.Fatal("ParseFrontmatter: unexpected nil frontmatter")
	}
	return fm
}

// TestParseFrontmatter_YAMLShape_AnchorsAndAliases verifies YAML anchors (&x)
// and aliases (*x) are resolved by yaml.v3 and both the anchor- and alias-
// bearing keys appear in Extra + KeyRanges.
func TestParseFrontmatter_YAMLShape_AnchorsAndAliases(t *testing.T) {
	source := "---\ndefault: &d\n  precision: 2\noverrides: *d\n---\n"
	fm := parseOrFatal(t, source)

	if _, ok := fm.KeyRanges["default"]; !ok {
		t.Error("KeyRanges missing 'default' (anchor key)")
	}
	if r, ok := fm.KeyRanges["overrides"]; !ok {
		t.Error("KeyRanges missing 'overrides' (alias key)")
	} else if r.Start.Line != 4 {
		t.Errorf("overrides Start.Line = %d, want 4", r.Start.Line)
	}

	// Both should appear in Extra (neither is a registered CalcMark key).
	haveDefault, haveOverrides := false, false
	for _, e := range fm.Extra {
		switch e.Key {
		case "default":
			haveDefault = true
		case "overrides":
			haveOverrides = true
		}
	}
	if !haveDefault || !haveOverrides {
		t.Errorf("Extra missing keys; haveDefault=%v haveOverrides=%v", haveDefault, haveOverrides)
	}
}

// TestParseFrontmatter_YAMLShape_TaggedScalar verifies !!str and similar
// explicit YAML tags parse without panic; the value surfaces as the tagged
// Go type (string in this case, not time.Time).
func TestParseFrontmatter_YAMLShape_TaggedScalar(t *testing.T) {
	source := "---\ndate: !!str 2026-04-14\n---\n"
	fm := parseOrFatal(t, source)

	if _, ok := fm.KeyRanges["date"]; !ok {
		t.Error("KeyRanges missing 'date'")
	}
	var got any
	for _, e := range fm.Extra {
		if e.Key == "date" {
			got = e.Value
		}
	}
	if s, ok := got.(string); !ok || s != "2026-04-14" {
		t.Errorf("Extra[date] = %T %v, want string \"2026-04-14\" (!!str should force string)", got, got)
	}
}

// TestParseFrontmatter_YAMLShape_FlowMapping verifies inline flow syntax
// { k: v, k2: v2 } parses correctly for a registered key (exchange).
func TestParseFrontmatter_YAMLShape_FlowMapping(t *testing.T) {
	source := "---\nexchange: { USD_EUR: 1.1, USD_GBP: 0.8 }\n---\n"
	fm := parseOrFatal(t, source)

	if len(fm.Exchange) != 2 {
		t.Errorf("Exchange len = %d, want 2; got %v", len(fm.Exchange), fm.Exchange)
	}
	if _, ok := fm.KeyRanges["exchange"]; !ok {
		t.Error("KeyRanges missing 'exchange'")
	}
}

// TestParseFrontmatter_YAMLShape_LiteralBlockScalar verifies `|` literal block
// scalars don't panic and are captured as multi-line strings in Extra.
//
// KNOWN LIMITATION: KeyRanges.End.Line for block scalars points at the KEY
// line, not the last physical line of the folded/literal content. See the
// comment in nodeEndPosition() in frontmatter.go — yaml.v3 does not expose
// block-scalar end positions. Fine for LSP's current use (it only needs the
// key anchor), but flipped when yaml.v3 exposes end or we switch parsers.
func TestParseFrontmatter_YAMLShape_LiteralBlockScalar(t *testing.T) {
	source := "---\ndescription: |\n  Multi\n  line\n---\n"
	fm := parseOrFatal(t, source)

	r, ok := fm.KeyRanges["description"]
	if !ok {
		t.Fatal("KeyRanges missing 'description'")
	}
	if r.Start.Line != 2 {
		t.Errorf("description Start.Line = %d, want 2", r.Start.Line)
	}
	// Imperfect-but-documented: End.Line == 2 (key line), not 4.
	if r.End.Line != 2 {
		t.Errorf("description End.Line = %d, want 2 (known limitation — yaml.v3 "+
			"does not expose block-scalar end; flip this test when fixed)", r.End.Line)
	}

	var val any
	for _, e := range fm.Extra {
		if e.Key == "description" {
			val = e.Value
		}
	}
	if s, ok := val.(string); !ok || s != "Multi\nline" {
		t.Errorf("Extra[description] = %T %q, want string \"Multi\\nline\"", val, val)
	}
}

// TestParseFrontmatter_YAMLShape_FoldedBlockScalar verifies `>` folded block
// scalars behave like literal: captured as a string, KeyRanges has an entry
// (End.Line same limitation as literal block scalars).
func TestParseFrontmatter_YAMLShape_FoldedBlockScalar(t *testing.T) {
	source := "---\nsummary: >\n  Folded\n  into one line\n---\n"
	fm := parseOrFatal(t, source)

	r, ok := fm.KeyRanges["summary"]
	if !ok {
		t.Fatal("KeyRanges missing 'summary'")
	}
	if r.Start.Line != 2 {
		t.Errorf("summary Start.Line = %d, want 2", r.Start.Line)
	}

	var val any
	for _, e := range fm.Extra {
		if e.Key == "summary" {
			val = e.Value
		}
	}
	if s, ok := val.(string); !ok || s != "Folded into one line" {
		t.Errorf("Extra[summary] = %T %q, want folded one-line string", val, val)
	}
}

// TestParseFrontmatter_YAMLShape_EmptyValue verifies `title:` with no value
// yields a nil Extra value and a populated KeyRanges entry. No panic, no
// silent data-loss (the key is preserved even with nil value).
func TestParseFrontmatter_YAMLShape_EmptyValue(t *testing.T) {
	source := "---\ntitle:\n---\n"
	fm := parseOrFatal(t, source)

	r, ok := fm.KeyRanges["title"]
	if !ok {
		t.Fatal("KeyRanges missing 'title'")
	}
	if r.Start.Line != 2 {
		t.Errorf("title Start.Line = %d, want 2", r.Start.Line)
	}

	found := false
	for _, e := range fm.Extra {
		if e.Key == "title" {
			found = true
			if e.Value != nil {
				t.Errorf("Extra[title].Value = %T %v, want nil", e.Value, e.Value)
			}
		}
	}
	if !found {
		t.Error("Extra missing 'title' entry (empty-value key should be preserved)")
	}
}

// TestParseFrontmatter_YAMLShape_ScalarTypes verifies bool, null, and int
// scalars are captured with the correct Go types in Extra.
func TestParseFrontmatter_YAMLShape_ScalarTypes(t *testing.T) {
	source := "---\npublished: false\ndraft: null\npriority: 1\n---\n"
	fm := parseOrFatal(t, source)

	want := map[string]any{
		"published": false,
		"draft":     nil,
		"priority":  1,
	}
	got := map[string]any{}
	for _, e := range fm.Extra {
		got[e.Key] = e.Value
	}
	for k, v := range want {
		gv, ok := got[k]
		if !ok {
			t.Errorf("Extra missing %q", k)
			continue
		}
		if gv != v {
			t.Errorf("Extra[%q] = %T %v, want %T %v", k, gv, gv, v, v)
		}
		if _, rok := fm.KeyRanges[k]; !rok {
			t.Errorf("KeyRanges missing %q", k)
		}
	}
}

// TestParseFrontmatter_YAMLShape_DateScalar verifies an unquoted ISO date
// is parsed by yaml.v3 as time.Time. Worth documenting because it's a
// common footgun — users expecting a string get a time.Time.
func TestParseFrontmatter_YAMLShape_DateScalar(t *testing.T) {
	source := "---\ndate: 2026-04-14\n---\n"
	fm := parseOrFatal(t, source)

	var got any
	for _, e := range fm.Extra {
		if e.Key == "date" {
			got = e.Value
		}
	}
	// yaml.v3 auto-parses ISO dates to time.Time. Document this so users can
	// quote the value if they want a string.
	if got == nil {
		t.Fatal("Extra missing 'date'")
	}
	// We don't strictly assert time.Time here (the exact type is a yaml.v3
	// implementation detail); we just assert it parsed and is not a string.
	if _, isStr := got.(string); isStr {
		t.Errorf("Extra[date] = string (unexpected); yaml.v3 auto-parses ISO dates to time.Time")
	}
	if _, ok := fm.KeyRanges["date"]; !ok {
		t.Error("KeyRanges missing 'date'")
	}
}

// TestParseFrontmatter_YAMLShape_QuotedSpecialChars verifies a quoted string
// containing `:` and `!` parses without splitting on the colon.
func TestParseFrontmatter_YAMLShape_QuotedSpecialChars(t *testing.T) {
	source := "---\ntitle: \"Hello: World!\"\n---\n"
	fm := parseOrFatal(t, source)

	var got any
	for _, e := range fm.Extra {
		if e.Key == "title" {
			got = e.Value
		}
	}
	if s, ok := got.(string); !ok || s != "Hello: World!" {
		t.Errorf("Extra[title] = %T %v, want string \"Hello: World!\"", got, got)
	}
	if _, ok := fm.KeyRanges["title"]; !ok {
		t.Error("KeyRanges missing 'title'")
	}
}

// TestParseFrontmatter_YAMLShape_MultiDocMarkerInBody verifies that a
// `---` line inside the frontmatter body is treated as the closing fence
// (first `---` wins). This is existing behavior: the fence scan stops at
// the first match. We assert "no panic and predictable truncation", not
// "error on embedded ---". If/when we want stricter validation, flip the
// expectation here.
func TestParseFrontmatter_YAMLShape_MultiDocMarkerInBody(t *testing.T) {
	// Layout:
	// 1: ---
	// 2: title: a
	// 3: ---      <- treated as closing fence
	// 4: foo: b
	// 5: ---
	source := "---\ntitle: a\n---\nfoo: b\n---\n"
	fm, rest, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("ParseFrontmatter: unexpected error: %v (expected graceful truncation at first ---)", err)
	}
	if fm == nil {
		t.Fatal("expected non-nil frontmatter")
	}
	// Only 'title' should be captured; 'foo' lives in the body.
	if _, ok := fm.KeyRanges["foo"]; ok {
		t.Error("KeyRanges unexpectedly includes 'foo' — fence parse should stop at first ---")
	}
	if _, ok := fm.KeyRanges["title"]; !ok {
		t.Error("KeyRanges missing 'title'")
	}
	if rest == "" {
		t.Error("expected non-empty remaining body after truncated fence")
	}
}

// TestParseFrontmatter_YAMLShape_InlineComment verifies `#` inline comments
// are stripped by yaml.v3 and do not leak into the captured value.
func TestParseFrontmatter_YAMLShape_InlineComment(t *testing.T) {
	source := "---\ntitle: Hello  # a comment\n---\n"
	fm := parseOrFatal(t, source)

	var got any
	for _, e := range fm.Extra {
		if e.Key == "title" {
			got = e.Value
		}
	}
	if s, ok := got.(string); !ok || s != "Hello" {
		t.Errorf("Extra[title] = %T %v, want string \"Hello\" (comment must be stripped)", got, got)
	}
}
