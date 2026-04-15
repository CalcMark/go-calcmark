package lsp

import (
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestFrontmatterHover_RegisteredKey — cursor on a registered key returns
// hover content containing the key name, type label, and docstring text.
func TestFrontmatterHover_RegisteredKey(t *testing.T) {
	src := "---\nconvert_to: si\n---\nbody\n"
	h := buildFrontmatterHover(src, protocol.Position{Line: 1, Character: 2})
	if h == nil {
		t.Fatal("expected hover for convert_to, got nil")
	}
	mc, ok := h.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("contents not MarkupContent: %T", h.Contents)
	}
	for _, want := range []string{"**convert_to**", "enum", "si", "imperial"} {
		if !strings.Contains(mc.Value, want) {
			t.Errorf("missing %q in hover:\n%s", want, mc.Value)
		}
	}
	if !strings.Contains(strings.ToLower(mc.Value), "unit system") {
		t.Errorf("expected docstring content in hover:\n%s", mc.Value)
	}
}

// TestFrontmatterHover_EnumValue — cursor on an enum value of a registered
// key returns the same key hover (acceptable delegation).
func TestFrontmatterHover_EnumValue(t *testing.T) {
	src := "---\nconvert_to: si\n---\n"
	h := buildFrontmatterHover(src, protocol.Position{Line: 1, Character: 13})
	if h == nil {
		t.Fatal("expected hover on enum value")
	}
	mc := h.Contents.(protocol.MarkupContent)
	if !strings.Contains(mc.Value, "**convert_to**") {
		t.Errorf("missing key in hover:\n%s", mc.Value)
	}
}

// TestFrontmatterHover_ExtraKey — title is not registered, returns nil.
func TestFrontmatterHover_ExtraKey(t *testing.T) {
	src := "---\ntitle: Hello\n---\n"
	if h := buildFrontmatterHover(src, protocol.Position{Line: 1, Character: 2}); h != nil {
		t.Errorf("expected nil for extra key, got %+v", h)
	}
}

// TestFrontmatterHover_Outside — cursor outside frontmatter returns nil
// (handler must fall through to regular hover).
func TestFrontmatterHover_Outside(t *testing.T) {
	src := "---\nconvert_to: si\n---\nprice = 100\n"
	if h := buildFrontmatterHover(src, protocol.Position{Line: 3, Character: 2}); h != nil {
		t.Errorf("expected nil outside frontmatter, got %+v", h)
	}
}

// TestFrontmatterHover_Fence — cursor on `---` fence returns nil.
func TestFrontmatterHover_Fence(t *testing.T) {
	src := "---\nconvert_to: si\n---\n"
	if h := buildFrontmatterHover(src, protocol.Position{Line: 0, Character: 1}); h != nil {
		t.Errorf("expected nil on fence, got %+v", h)
	}
	if h := buildFrontmatterHover(src, protocol.Position{Line: 2, Character: 1}); h != nil {
		t.Errorf("expected nil on closing fence, got %+v", h)
	}
}

// TestFrontmatterHover_Blank — blank line inside frontmatter returns nil.
func TestFrontmatterHover_Blank(t *testing.T) {
	src := "---\n\nconvert_to: si\n---\n"
	if h := buildFrontmatterHover(src, protocol.Position{Line: 1, Character: 0}); h != nil {
		t.Errorf("expected nil on blank line, got %+v", h)
	}
}

// TestFrontmatterHover_IndentedContinuation — cursor on an indented
// continuation line returns the parent key's hover.
func TestFrontmatterHover_IndentedContinuation(t *testing.T) {
	src := "---\nglobals:\n  price: 100\n---\n"
	h := buildFrontmatterHover(src, protocol.Position{Line: 2, Character: 4})
	if h == nil {
		t.Fatal("expected hover on indented globals continuation")
	}
	mc := h.Contents.(protocol.MarkupContent)
	if !strings.Contains(mc.Value, "**globals**") {
		t.Errorf("missing parent key in hover:\n%s", mc.Value)
	}
}

// TestFrontmatterHover_NoFrontmatter — source without `---` returns nil.
func TestFrontmatterHover_NoFrontmatter(t *testing.T) {
	if h := buildFrontmatterHover("price = 100", protocol.Position{Line: 0, Character: 0}); h != nil {
		t.Errorf("expected nil when no frontmatter, got %+v", h)
	}
}
