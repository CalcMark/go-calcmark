package lsp

import (
	"fmt"
	"strings"

	specDoc "github.com/CalcMark/go-calcmark/v2/spec/document"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// buildFrontmatterHover returns an *lspHover for registered CalcMark
// frontmatter keys when the cursor sits on a key or its value. Returns nil
// when the cursor is outside the frontmatter region, on a fence, on a blank
// line, or on an unregistered (Extra) key — the caller should fall through to
// regular hover dispatch in those cases.
//
// Content is Markdown formatted as:
//
//	**<name>** *(<type-label>)*
//
//	<docstring>
//
// For EnumString keys the type label includes the accepted values, e.g.
// `enum: si | imperial`.
//
// Complexity: O(|frontmatter source|) for region detection, O(|Registry|) for
// lookup — per D10.
func buildFrontmatterHover(source string, position protocol.Position) *lspHover {
	region, ok := DetectRegion(source)
	if !ok {
		return nil
	}
	ctx := ClassifyCursor(region, position)
	if !ctx.InRegion {
		return nil
	}
	switch ctx.Position {
	case CursorFence, CursorOutside:
		return nil
	}
	if ctx.Key == "" {
		return nil
	}
	key, ok := specDoc.LookupKey(ctx.Key)
	if !ok {
		return nil
	}
	return &lspHover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: renderFrontmatterKeyHover(key),
		},
	}
}

// renderFrontmatterKeyHover formats the Markdown body for a registered key.
func renderFrontmatterKeyHover(key specDoc.RegisteredKey) string {
	return fmt.Sprintf("**%s** *(%s)*\n\n%s", key.Name, frontmatterTypeLabel(key), key.Doc)
}

// frontmatterTypeLabel renders a human-readable type label for hover display.
// EnumString keys inline the accepted values for at-a-glance learning.
func frontmatterTypeLabel(key specDoc.RegisteredKey) string {
	switch key.Type {
	case specDoc.FrontmatterKeyEnumString:
		if len(key.EnumValues) > 0 {
			return "enum: " + strings.Join(key.EnumValues, " | ")
		}
		return "enum"
	case specDoc.FrontmatterKeyMapStringDecimal:
		return "map<string, decimal>"
	case specDoc.FrontmatterKeyMapStringString:
		return "map<string, expression>"
	case specDoc.FrontmatterKeyStruct:
		return "struct"
	default:
		return key.Type.String()
	}
}
