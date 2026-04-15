package lsp

import (
	"sort"
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// registryKeyNames returns the canonical set of registered CalcMark
// frontmatter key names. Kept inline so the test fails loudly if Registry
// changes unexpectedly.
var registryKeyNames = []string{
	"convert_to",
	"exchange",
	"fiscal_year_starts",
	"globals",
	"measurement",
	"scale",
}

func itemLabelList(items []protocol.CompletionItem) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Label)
	}
	return out
}

// TestFrontmatterCompletion_KeyPosition_EmptyRegion — cursor inside an
// otherwise empty frontmatter returns all registered keys, alphabetically
// sorted, with Property kind and a non-empty Markdown documentation.
func TestFrontmatterCompletion_KeyPosition_EmptyRegion(t *testing.T) {
	src := "---\n\n---\n"
	items := buildFrontmatterCompletions(src, protocol.Position{Line: 1, Character: 0})
	if len(items) != len(registryKeyNames) {
		t.Fatalf("expected %d items, got %d: %v", len(registryKeyNames), len(items), itemLabelList(items))
	}
	labels := itemLabelList(items)
	if !sort.StringsAreSorted(labels) {
		t.Errorf("labels not alphabetically sorted: %v", labels)
	}
	for i, want := range registryKeyNames {
		if labels[i] != want {
			t.Errorf("labels[%d] = %q, want %q (full list %v)", i, labels[i], want, labels)
		}
	}
	// Verify shape of one item
	var cvt *protocol.CompletionItem
	for i := range items {
		if items[i].Label == "convert_to" {
			cvt = &items[i]
			break
		}
	}
	if cvt == nil {
		t.Fatal("convert_to item not found")
	}
	if cvt.Kind == nil || *cvt.Kind != protocol.CompletionItemKindProperty {
		t.Errorf("convert_to kind = %v, want Property", cvt.Kind)
	}
	if cvt.Detail == nil || !strings.Contains(*cvt.Detail, "enum") {
		t.Errorf("convert_to detail missing 'enum': %v", cvt.Detail)
	}
	mc, ok := cvt.Documentation.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("convert_to documentation not MarkupContent: %T", cvt.Documentation)
	}
	if mc.Kind != protocol.MarkupKindMarkdown {
		t.Errorf("convert_to doc kind = %q, want markdown", mc.Kind)
	}
	if !strings.Contains(strings.ToLower(mc.Value), "unit system") {
		t.Errorf("convert_to doc missing expected text: %q", mc.Value)
	}
}

// TestFrontmatterCompletion_KeyPosition_WithExistingKey — cursor on an
// existing top-level key (before the colon) still returns the full registered
// list. Client performs prefix filtering; server returns all candidates.
func TestFrontmatterCompletion_KeyPosition_WithExistingKey(t *testing.T) {
	src := "---\nconvert_to: si\n---\n"
	items := buildFrontmatterCompletions(src, protocol.Position{Line: 1, Character: 3})
	if len(items) != len(registryKeyNames) {
		t.Fatalf("expected %d items (client-side filtering), got %d", len(registryKeyNames), len(items))
	}
}

// TestFrontmatterCompletion_ValuePosition_EnumKey — cursor at value position
// for an EnumString key returns exactly the enum values, alphabetically
// sorted, as EnumMember items.
func TestFrontmatterCompletion_ValuePosition_EnumKey(t *testing.T) {
	src := "---\nconvert_to: \n---\n"
	// col 12 = after "convert_to: "
	items := buildFrontmatterCompletions(src, protocol.Position{Line: 1, Character: 12})
	if len(items) != 2 {
		t.Fatalf("expected 2 enum items, got %d: %v", len(items), itemLabelList(items))
	}
	labels := itemLabelList(items)
	if !sort.StringsAreSorted(labels) {
		t.Errorf("enum labels not sorted: %v", labels)
	}
	want := map[string]bool{"si": true, "imperial": true}
	for _, l := range labels {
		if !want[l] {
			t.Errorf("unexpected enum label %q", l)
		}
	}
	if items[0].Kind == nil || *items[0].Kind != protocol.CompletionItemKindEnumMember {
		t.Errorf("kind = %v, want EnumMember", items[0].Kind)
	}
}

// TestFrontmatterCompletion_ValuePosition_EnumKey_AlreadyTyped — even after
// a value is already present, value-position completion still returns the
// enum values. Client decides whether to show.
func TestFrontmatterCompletion_ValuePosition_EnumKey_AlreadyTyped(t *testing.T) {
	src := "---\nconvert_to: si\n---\n"
	items := buildFrontmatterCompletions(src, protocol.Position{Line: 1, Character: 14})
	if len(items) != 2 {
		t.Fatalf("expected 2 enum items even when value typed, got %d", len(items))
	}
}

// TestFrontmatterCompletion_ValuePosition_NonEnum — cursor at value position
// for a non-EnumString key (e.g., globals → map) returns nil. Struct/map
// completion is out of scope for this unit.
func TestFrontmatterCompletion_ValuePosition_NonEnum(t *testing.T) {
	src := "---\nglobals: \n---\n"
	items := buildFrontmatterCompletions(src, protocol.Position{Line: 1, Character: 9})
	if items != nil {
		t.Errorf("expected nil for non-enum value position, got %v", itemLabelList(items))
	}
}

// TestFrontmatterCompletion_ValuePosition_ExtraKey — cursor at value position
// for an unregistered key returns nil.
func TestFrontmatterCompletion_ValuePosition_ExtraKey(t *testing.T) {
	src := "---\ntitle: \n---\n"
	items := buildFrontmatterCompletions(src, protocol.Position{Line: 1, Character: 7})
	if items != nil {
		t.Errorf("expected nil for extra-key value, got %v", itemLabelList(items))
	}
}

// TestFrontmatterCompletion_Fence — cursor on `---` fence returns nil.
func TestFrontmatterCompletion_Fence(t *testing.T) {
	src := "---\nconvert_to: si\n---\n"
	if items := buildFrontmatterCompletions(src, protocol.Position{Line: 0, Character: 1}); items != nil {
		t.Errorf("expected nil on opening fence, got %v", itemLabelList(items))
	}
	if items := buildFrontmatterCompletions(src, protocol.Position{Line: 2, Character: 1}); items != nil {
		t.Errorf("expected nil on closing fence, got %v", itemLabelList(items))
	}
}

// TestFrontmatterCompletion_BlankLine — blank line inside region → all
// registered keys (author about to type a new key).
func TestFrontmatterCompletion_BlankLine(t *testing.T) {
	src := "---\nconvert_to: si\n\n---\n"
	items := buildFrontmatterCompletions(src, protocol.Position{Line: 2, Character: 0})
	if len(items) != len(registryKeyNames) {
		t.Fatalf("expected %d items on blank line, got %d", len(registryKeyNames), len(items))
	}
}

// TestFrontmatterCompletion_Outside — cursor outside region returns nil
// (handler falls through to calc completion).
func TestFrontmatterCompletion_Outside(t *testing.T) {
	src := "---\nconvert_to: si\n---\nprice = 100\n"
	items := buildFrontmatterCompletions(src, protocol.Position{Line: 3, Character: 2})
	if items != nil {
		t.Errorf("expected nil outside region, got %v", itemLabelList(items))
	}
}

// TestFrontmatterCompletion_NoFrontmatter — source without a frontmatter
// region returns nil.
func TestFrontmatterCompletion_NoFrontmatter(t *testing.T) {
	items := buildFrontmatterCompletions("price = 100\n", protocol.Position{Line: 0, Character: 0})
	if items != nil {
		t.Errorf("expected nil when no frontmatter, got %v", itemLabelList(items))
	}
}
