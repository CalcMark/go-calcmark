package lsp

import (
	"sort"

	specDoc "github.com/CalcMark/go-calcmark/spec/document"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// buildFrontmatterCompletions returns completion items for CalcMark
// frontmatter context. It returns:
//
//   - nil when the cursor is outside the frontmatter region, on a fence, or
//     at a value position for a non-EnumString / unregistered key
//   - all registered keys (sorted alphabetically by name) when the cursor is
//     at a key position inside the region, including blank lines
//   - the enum values of the key (sorted) when the cursor is at a value
//     position for an EnumString-typed registered key
//
// Filtering decision: prefix filtering is intentionally NOT performed
// server-side. LSP clients filter the returned list client-side as the user
// types; returning the full registry keeps the server stateless and matches
// how the existing calc-block completion surfaces all functions/units.
//
// Complexity per D10: O(|Registry|) ≈ 6 iterations for the key branch;
// O(|EnumValues|) for the value branch.
func buildFrontmatterCompletions(source string, position protocol.Position) []protocol.CompletionItem {
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
	case CursorKey:
		return registeredKeyCompletions()
	case CursorValue:
		if ctx.Key == "" {
			return nil
		}
		key, found := specDoc.LookupKey(ctx.Key)
		if !found {
			return nil
		}
		if key.Type != specDoc.FrontmatterKeyEnumString {
			// Struct / map value completion is deferred (Open Question).
			return nil
		}
		return enumValueCompletions(key)
	}
	return nil
}

// registeredKeyCompletions returns one CompletionItem per entry in the
// frontmatter Registry, sorted by label.
func registeredKeyCompletions() []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0, len(specDoc.Registry))
	kind := protocol.CompletionItemKindProperty
	for _, key := range specDoc.Registry {
		detail := frontmatterTypeLabel(key)
		label := key.Name
		insertText := key.Name
		items = append(items, protocol.CompletionItem{
			Label:      label,
			Kind:       &kind,
			Detail:     &detail,
			InsertText: &insertText,
			Documentation: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: renderFrontmatterKeyHover(key),
			},
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Label < items[j].Label
	})
	return items
}

// enumValueCompletions returns one CompletionItem per enum value on an
// EnumString-typed registered key, sorted by label.
func enumValueCompletions(key specDoc.RegisteredKey) []protocol.CompletionItem {
	if len(key.EnumValues) == 0 {
		return nil
	}
	items := make([]protocol.CompletionItem, 0, len(key.EnumValues))
	kind := protocol.CompletionItemKindEnumMember
	for _, v := range key.EnumValues {
		value := v
		detail := "value of " + key.Name
		items = append(items, protocol.CompletionItem{
			Label:      value,
			Kind:       &kind,
			Detail:     &detail,
			InsertText: &value,
			Documentation: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: key.Doc,
			},
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Label < items[j].Label
	})
	return items
}
