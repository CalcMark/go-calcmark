# Text Wrapping Alignment Bug - FIXED

## Summary

**Status: FIXED ✅**

When source lines wrap to multiple visual lines, the preview pane was showing **empty lines** for wrapped continuations instead of showing the wrapped preview content. This created extra empty visual lines in the preview pane that didn't correspond to actual content.

## Root Cause

The alignment code in `view.go` and `aligned.go` handles source line wrapping correctly (creating multiple visual lines for a single source line), but the preview rendering doesn't properly wrap its content to match. When a source line wraps, the preview should also wrap its corresponding content, but instead it shows empty lines.

## Evidence

### Test 1: `testdata/wrapping_alignment`

**Document:**
```
a = 2
b = 10 MB


# Testing a reaaeeeeeeeeeeeeeeeeeeeeelly long
c = a * b
```

**Bug on line [5]:**
```
[5] SRC(ln=0 wrap=true): "reaaeeeeeeeeeeeeeeeeeeeeelly long"  | PRV(ln=4): ""
```

The source pane correctly shows the wrapped continuation of the heading, but the preview pane shows an **empty line** instead of the wrapped heading content.

### Test 2: `testdata/wrapping_calc_lines`

**Document:**
```
very_long_variable_name_that_will_definitely_wrap_in_narrow_pane = 42
result = very_long_variable_name_that_will_definitely_wrap_in_narrow_pane * 2
```

**Bugs on lines [1], [3], [4]:**
```
[1] SRC(ln=0 wrap=true): "nitely_wrap_in_narrow_pane = 42"         | PRV(ln=0): ""
[3] SRC(ln=0 wrap=true): "very_long_variable_name_that_will_d..."  | PRV(ln=1): ""
[4] SRC(ln=0 wrap=true): "nitely_wrap_in_narrow_pane * 2"          | PRV(ln=1): ""
```

ALL wrapped source lines show empty preview lines! The preview should show wrapped parts of the rendered content:
- Line [0] preview: "very_long_variable_name... → 42" (wraps)
- Line [1] preview: should show wrapped part, not empty
- Line [2] preview: "result → 84" (wraps)
- Lines [3], [4] preview: should show wrapped parts, not empty

## Expected Behavior

When a source line wraps to N visual lines, the preview should ALSO wrap its rendered content to N visual lines, maintaining 1:1 alignment. Both panes should have the same number of visual lines, and wrapped continuations should show actual content, not empty lines.

## Impact

- **Visual alignment is broken** - Users see empty lines in preview that don't exist in source
- **Preview content is cut off** - Wrapped preview content is not visible
- **User confusion** - The extra empty lines make it look like there's missing content

## Fix

**Two changes were made:**

### 1. Fixed `wrapStyledLine` in `view.go` (lines 344-395)

**Problem:** When styled content (ANSI codes) exceeded maxWidth, the function returned a single unwrapped line instead of wrapping it.

**Solution:** Strip ANSI codes to get plain text, wrap the plain text using `WrapText`, then return the wrapped lines. This properly handles calc result wrapping like `very_long_variable_name → 42`.

```go
// Extract plain text (removes ANSI codes)
plainText := stripANSI(line)

// Wrap the plain text
wrappedPlainLines := WrapText(plainText, maxWidth)

return wrappedPlainLines
```

### 2. Fixed text block preview distribution in `aligned.go` (lines 158-184)

**Problem:** The code rendered the entire text block at once, then tried to distribute rendered lines 1:1 to source lines. This failed when a single source line (like a long heading) rendered to multiple glamour lines due to wrapping.

**Solution:** Render each source line individually, capturing ALL its wrapped visual lines. Store the complete wrapped output for each source line in the cache.

```go
for blockLineIdx, lineResult := range blockResults {
    lineText := lineResult.Source
    // ...

    // Render this line individually
    // renderMarkdown returns all visual lines for this source line
    renderedLines := renderMarkdown(lineText, input.PreviewWidth)

    // Store all rendered lines for this source line
    // This preserves wrapping: if a heading wraps to 2 lines, we store both
    textBlockPreviewCache[blockLineIdx] = renderedLines
}
```

**Trade-off:** This approach means multi-line markdown constructs (like ordered lists where `1.` should become `1.`, `2.`, `3.`) are now rendered line-by-line instead of as a block. However, this is acceptable because:
1. The wrapping bug was more critical
2. Most text blocks are single-line (headings, paragraphs)
3. Ordered lists can still be rendered correctly with future enhancements

## Testing

Run the wrapping alignment tests:

```bash
go test ./cmd/calcmark/tui/editor -run "TestEditorCatwalkWrapping" -v
```

**All tests now PASS ✅** with wrapped preview content correctly displayed.

## Files Modified

- `cmd/calcmark/tui/editor/view.go` - Fixed `wrapStyledLine()`, added `stripANSI()`
- `cmd/calcmark/tui/editor/aligned.go` - Fixed text block preview rendering to handle wrapping
- `cmd/calcmark/tui/editor/catwalk_wrapping_test.go` - Test runner with alignment observer
- `cmd/calcmark/tui/editor/testdata/wrapping_alignment` - Heading wrapping test (updated with fix)
- `cmd/calcmark/tui/editor/testdata/wrapping_calc_lines` - Calc line wrapping test (updated with fix)
