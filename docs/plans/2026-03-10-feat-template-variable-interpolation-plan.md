---
title: "feat: Template variable interpolation ({{variable}})"
type: feat
status: completed
date: 2026-03-10
brainstorm: docs/brainstorms/2026-03-10-template-variable-interpolation-brainstorm.md
---

# feat: Template Variable Interpolation (`{{variable}}`)

## Overview

Add a post-evaluation interpolation pass that replaces `{{variable_name}}` tags in markdown
TextBlocks with the display-formatted final value of that variable. This enables forward
references — a summary table at the top of a document can show results computed 100 lines
below.

## Problem Statement / Motivation

CalcMark documents often follow the pattern: executive summary at the top, detailed
calculations below. Today there is no way to reference a computed value in prose without
manually copying it. The `{{variable}}` tag solves this by automatically resolving forward
references after the entire document is evaluated.

## Proposed Solution

### Phase 1: Spec — TextBlock Interpolation Storage

Add an `interpolatedSource` field to `TextBlock` in the spec layer. This stores the
post-interpolation version of the source lines without mutating the original `source`.

**Files changed:**
- `spec/document/block.go`

```go
// TextBlock additions
type TextBlock struct {
    source             []string // Raw source lines (immutable)
    interpolatedSource []string // Post-interpolation lines (nil = no interpolation)
    html               string
    dirty              bool
}

// InterpolatedSource returns interpolated lines if set, otherwise raw source.
// All output consumers should call this instead of Source() for display.
func (tb *TextBlock) InterpolatedSource() []string {
    if tb.interpolatedSource != nil {
        return tb.interpolatedSource
    }
    return tb.source
}

// SetInterpolatedSource stores the post-interpolation lines.
func (tb *TextBlock) SetInterpolatedSource(lines []string) {
    tb.interpolatedSource = lines
}

// ClearInterpolatedSource resets interpolation (e.g., when source changes).
func (tb *TextBlock) ClearInterpolatedSource() {
    tb.interpolatedSource = nil
}
```

**Design note:** `InterpolatedSource()` falls back to `Source()` when no interpolation has
been applied. This means callers that switch to `InterpolatedSource()` will work correctly
even before the interpolation pass runs — a safe, incremental migration.

### Phase 2: Impl — Interpolation Engine

Add a post-evaluation interpolation pass in the evaluator. This runs after all CalcBlocks
are evaluated, so the environment contains final variable values.

**Files changed:**
- `impl/document/evaluator.go`

```go
// interpolateTextBlocks resolves {{var}} tags in all TextBlocks against
// the final environment. Called after full evaluation completes.
func (e *Evaluator) interpolateTextBlocks(doc *document.Document, df display.Formatter) {
    env := e.env.GetAllVariables()
    for _, node := range doc.GetBlocks() {
        tb, ok := node.Block.(*document.TextBlock)
        if !ok {
            continue
        }
        interpolated := make([]string, len(tb.Source()))
        changed := false
        for i, line := range tb.Source() {
            resolved := interpolateLine(line, env, df)
            interpolated[i] = resolved
            if resolved != line {
                changed = true
            }
        }
        if changed {
            tb.SetInterpolatedSource(interpolated)
            tb.SetDirty(true) // Force HTML re-render
        }
    }
}

// interpolateLine replaces all {{var}} tags in a single line.
// Unresolved tags are left as-is.
func interpolateLine(line string, env map[string]types.Type, df display.Formatter) string {
    // Regex: \{\{(\w+)\}\}
    // For each match, look up variable in env, format with df.Format()
    // If not found, leave the {{var}} tag unchanged
}
```

**Integration points in `Evaluate()`:**
- After `applyTransforms(doc, nil)` at line 97, call `e.interpolateTextBlocks(doc, df)`
- The `display.Formatter` parameter needs to be passed into `Evaluate()` or configured on
  the `Evaluator` struct. Preferred: add `SetDisplayFormatter(df display.Formatter)` to
  `Evaluator`, matching the existing `SetDirectiveResolver` pattern.

**Integration points in `EvaluateBlock()`:**
- After `applyTransforms(doc, nil)` at line 243, call `e.interpolateTextBlocks(doc, df)`
- This ensures TUI reactive evaluation also updates interpolation.

**Regex pattern:** `\{\{\s*(\w+)\s*\}\}` — matches `{{name}}` and `{{ name }}` where name
is `[a-zA-Z0-9_]+`. Whitespace inside braces is trimmed. Empty tags `{{}}` and expression
tags `{{a + b}}` are intentionally not matched.

### Phase 3: Consumer Migration — Switch to `InterpolatedSource()`

Switch all output-facing consumers from `Source()` to `InterpolatedSource()`. Editing
consumers keep using raw `Source()`.

**Switch to `InterpolatedSource()` (output paths):**

| File | Line | Context |
|------|------|---------|
| `format/markdown_formatter.go` | 87 | TextBlock line output |
| `format/text_formatter.go` | 90 | TextBlock line output |
| `format/html_formatter.go` | 169-170 | TextBlock escaped source |
| `format/json_formatter.go` | 177 | TextBlock source in JSON |
| `cmd/calcmark/tui/editor/block_render.go` | 87 | TextBlock Preview Pane display |
| `spec/document/markdown.go` | 18 | `Render()` HTML generation |
| `cmd/doceval/main.go` | 377 | doceval TextBlock line output |

**Keep raw `Source()` (editing/internal):**

| File | Context |
|------|---------|
| `cmd/calcmark/tui/editor/editing.go` | Source editing in TUI |
| `cmd/calcmark/tui/editor/model.go` | Block source management |
| `format/calcmark_formatter.go` | Round-trip `.cm` serialization (must preserve `{{var}}` tags) |
| `format/align.go` | CalcBlock result alignment (CalcBlock only, not TextBlock) |
| `impl/document/evaluator.go:117` | `checkTextBlockForLikelyCalculations` (inspects raw source) |

**Spec markdown rendering (`Render()`):**
`TextBlock.Render()` in `spec/document/markdown.go:18` currently calls `tb.SourceText()`.
This needs a new method:

```go
// InterpolatedSourceText returns interpolated source as a single string.
func (tb *TextBlock) InterpolatedSourceText() string {
    return strings.Join(tb.InterpolatedSource(), "\n")
}
```

Then `Render()` switches from `tb.SourceText()` to `tb.InterpolatedSourceText()`.
The HTML cache (`tb.html`) must be invalidated when interpolated source changes — handled
by `SetDirty(true)` in Phase 2.

### Phase 4: CLI Integration

Ensure all CLI commands pass a `display.Formatter` to the evaluator so interpolation
uses display-formatted values.

**Files changed:**
- `cmd/calcmark/cmd/convert.go` — set display formatter on evaluator before calling `Evaluate()`
- `eval.go` (top-level `Eval()` API) — set display formatter on evaluator
- `cmd/calcmark/tui/editor/model.go` — set display formatter on evaluator (TUI already has one)

**The formatter source:** Each CLI command already creates a `display.Formatter` for output.
The same formatter instance should be passed to the evaluator for interpolation consistency.

### Phase 5: Testing

#### Unit Tests

**`spec/document/block_test.go`:**
- `TestInterpolatedSourceFallback` — returns `Source()` when no interpolation set
- `TestInterpolatedSourceAfterSet` — returns interpolated lines after `SetInterpolatedSource()`
- `TestClearInterpolatedSource` — reverts to `Source()` after clear

**`impl/document/interpolation_test.go` (new file):**
- `TestInterpolateLine` — basic `{{var}}` replacement
- `TestInterpolateLineMultipleTags` — `{{a}} and {{b}}` in one line
- `TestInterpolateLineMissingVar` — `{{unknown}}` left as-is
- `TestInterpolateLineNoTags` — plain text unchanged
- `TestInterpolateLineDisplayFormatted` — currency, percentage, unit values
- `TestInterpolateLineAdjacentTags` — `{{a}}{{b}}` without spaces
- `TestInterpolateLineInTable` — `| {{var}} |` in markdown table cell
- `TestInterpolateLineInHeading` — `# Summary: {{total}}`
- `TestInterpolateLineWhitespace` — `{{ var }}` with spaces resolves correctly
- `TestInterpolateLinePartialBraces` — `{var}`, `{{}}` not matched

#### Integration Tests

**`impl/document/evaluator_test.go`:**
- `TestEvaluateInterpolatesTextBlocks` — full document with `{{var}}` in TextBlock
- `TestEvaluateForwardReference` — TextBlock with `{{var}}` before CalcBlock that defines var
- `TestEvaluateInterpolationPreservesSource` — raw `Source()` unchanged after interpolation

#### Golden File Tests

Add a new golden test file:

**`testdata/interpolation.cm`:**
```
## Summary

Total: `{{total}}`
Per item: `{{per_item}}`


items = 10
price = $25
total = items * price
per_item = total / items
```

Expected output (text format): Summary table shows `$250` and `$25`.

**`testdata/interpolation-missing.cm`:**
```
Result: `{{undefined_var}}`


x = 42
```

Expected: `{{undefined_var}}` left as-is in output.

### Phase 6: Documentation

- Update site worked example pages that would benefit from `{{var}}` interpolation
  (services-pl executive summary is the prime candidate)
- Add a "Template Variables" section to the CalcMark language reference
- Update `cm --help` or feature registry to mention interpolation

## Technical Considerations

### Transformed Values (Scale / Convert_to)

`applyTransforms()` modifies CalcBlock results (scale, convert_to) but does NOT modify the
interpreter environment. Since interpolated values should match what CalcBlocks display,
the interpolation pass must apply the same transforms before formatting.

**Decision:** Interpolated values reflect scale/convert_to transforms. If a document has
`scale: 1000` and `widgets = 5`, both the CalcBlock result and `{{widgets}}` show `5,000`.

**Implementation:** In `interpolateTextBlocks()`, after looking up a variable in the
environment, apply `transform.Apply(value, fm.Scale, fm.ConvertTo)` using the document's
frontmatter before passing to the display formatter. The `transform` package is already
imported by `impl/document/evaluator.go`.

### TUI Surgical Evaluation (EvaluateAffectedBlocks)

`EvaluateAffectedBlocks()` only re-evaluates specific CalcBlocks. After a surgical update,
the interpolation pass must still re-run on ALL TextBlocks to pick up changed variable
values. The cost is O(T * V) where T is total TextBlock text and V is variable count —
negligible for typical documents.

**Integration:** Add `e.interpolateTextBlocks(doc, df)` after `applyTransforms(doc, blockIDs)`
in `EvaluateAffectedBlocks()` at line 276.

### JSON Formatter Backwards Compatibility

The JSON formatter currently outputs `"source": node.Block.Source()`. To preserve backwards
compatibility for programmatic consumers, add a new `"interpolated_source"` field alongside
`"source"` when interpolation has been applied. This lets consumers detect which values were
templated.

### Display Formatter Lifecycle

The evaluator currently doesn't hold a reference to a `display.Formatter`. Two options:

1. **Add `SetDisplayFormatter()` to Evaluator** (preferred) — matches `SetDirectiveResolver`
   pattern. The evaluator stores it and uses it during interpolation. If none set, create a
   default formatter.
2. **Pass formatter as parameter to `Evaluate()`** — changes the API signature, breaking
   all callers.

Option 1 is preferred: non-breaking, follows existing patterns.

### Markdown Rendering Cache Invalidation

`TextBlock.Render()` caches HTML in `tb.html` and only re-renders when `tb.dirty` is true.
When `SetInterpolatedSource()` is called, the block must be marked dirty so `Render()`
regenerates HTML from the interpolated source. Phase 2 handles this by calling
`tb.SetDirty(true)` after setting interpolated source.

### CalcMark Formatter (Round-Trip Preservation)

`format/calcmark_formatter.go` serializes documents back to `.cm` format. It MUST use raw
`Source()`, not `InterpolatedSource()`, to preserve the `{{var}}` tags in the saved file.
This is the one formatter that intentionally does NOT resolve interpolation.

### Regex vs. Manual Parsing

A compiled `regexp.Regexp` is the simplest approach for `{{var}}` matching. The regex is
compiled once as a package-level `var` and reused. Performance is not a concern — TextBlocks
are typically short prose paragraphs.

### Spec/Impl Boundary

The `interpolatedSource` field and getter/setter live in `spec/document/block.go` (spec layer).
The interpolation logic (regex, environment lookup, formatting) lives in
`impl/document/evaluator.go` (impl layer). This respects the one-way dependency rule:
impl imports spec, never the reverse.

The `display.Formatter` import is already available in the impl layer. No new cross-layer
dependencies are introduced.

## Acceptance Criteria

- [x] `TextBlock` has `InterpolatedSource()`, `SetInterpolatedSource()`, `ClearInterpolatedSource()` methods
- [x] `InterpolatedSource()` falls back to `Source()` when no interpolation is set
- [x] Post-evaluation pass resolves `{{var}}` tags against final environment
- [x] Forward references work (TextBlock above CalcBlock that defines the variable)
- [x] Missing variables leave `{{var}}` unchanged in output
- [x] Resolved values use display formatting (currency symbols, units, percentages, K/M/B)
- [x] `cm convert --to md` shows interpolated values in TextBlocks
- [x] `cm convert --to html` shows interpolated values in TextBlocks
- [x] `cm convert --to json` shows interpolated values in TextBlocks
- [x] `cm --format text` shows interpolated values
- [x] TUI Preview Pane shows interpolated values
- [x] TUI editing shows raw `{{var}}` tags (not resolved)
- [x] `.cm` file save preserves raw `{{var}}` tags (CalcMark formatter uses `Source()`)
- [x] doceval resolves `{{var}}` tags in Hugo full-document output
- [x] `EvaluateAffectedBlocks()` re-runs interpolation on all TextBlocks
- [x] Interpolated values reflect scale/convert_to transforms (match CalcBlock display)
- [x] JSON output includes both `source` (raw) and `interpolated_source` (resolved) fields
- [x] Frontmatter globals (e.g., `@globals.rate`) accessible via `{{rate}}`
- [x] `TextBlock.Render()` produces HTML from interpolated source
- [x] Multiple `{{var}}` tags on one line all resolve
- [x] Adjacent `{{a}}{{b}}` tags resolve correctly
- [x] `{{ var }}` with whitespace resolves (whitespace trimmed)
- [x] Partial braces (`{var}`, `{{}}`) are NOT matched
- [x] Raw `Source()` is never mutated by interpolation
- [x] All existing tests pass (`task test`)
- [x] `task quality` passes
- [x] Golden test for interpolation with forward references
- [x] Golden test for missing variable (left as-is)

## Dependencies & Risks

**Dependencies:**
- Phase 2 depends on Phase 1 (TextBlock field must exist before evaluator can set it)
- Phase 3 depends on Phase 2 (consumers need interpolated source to be populated)
- Phase 4 depends on Phase 2 (CLI must configure formatter on evaluator)

**Risks:**
- **Accidental interpolation in code examples:** If a CalcMark document has prose describing
  `{{var}}` syntax (like docs), those tags would be interpolated. Mitigated by: documentation
  uses backtick-escaped `` `{{var}}` `` which is still plain text in TextBlocks. Future:
  consider escaping mechanism if this becomes a problem.
- **Performance in large documents:** Regex replacement on every TextBlock line. Not a
  concern — TextBlocks are prose paragraphs, not megabytes of data.
- **Existing documents with literal `{{`:** If a CalcMark document discusses Go templates
  or Mustache syntax using `{{range}}`, the interpolation pass would attempt to resolve it.
  Mitigated by: missing variables are left as-is. Only breaks if the text between `{{` and
  `}}` coincidentally matches a variable name in the same document. No escaping mechanism
  in v1 (YAGNI) — document as known limitation.
- **Doceval progressive mode:** Progressive mode evaluates code blocks within a Hugo page,
  not a standalone `.cm` file. TextBlocks in progressive mode are Hugo markdown, not part
  of the CalcMark document. Interpolation only works with `calcmark_source` full-document
  mode. Document this limitation.

## References & Research

### Internal References
- `spec/document/block.go:189-234` — TextBlock struct (interpolatedSource field goes here)
- `impl/document/evaluator.go:71-99` — `Evaluate()` method (interpolation pass after line 97)
- `impl/document/evaluator.go:108-112` — `GetEnvironment()` (provides variable values)
- `impl/interpreter/environment.go:73-74` — `GetAllVariables()` returns `map[string]types.Type`
- `format/markdown_formatter.go:82-91` — TextBlock output (switch to `InterpolatedSource()`)
- `format/text_formatter.go:90` — TextBlock output (switch to `InterpolatedSource()`)
- `format/html_formatter.go:169-170` — TextBlock output (switch to `InterpolatedSource()`)
- `format/json_formatter.go:177` — TextBlock source in JSON (switch to `InterpolatedSource()`)
- `cmd/calcmark/tui/editor/block_render.go:87` — TUI Preview Pane display
- `spec/document/markdown.go:13-22` — `Render()` HTML generation
- `cmd/doceval/main.go:377` — doceval TextBlock line output

### Brainstorm
- `docs/brainstorms/2026-03-10-template-variable-interpolation-brainstorm.md`

### Learnings Applied
- Display formatter must be consistent across all code paths (from locale formatting bypass solution)
- Use content-based mapping, not index-based (from result mapping pitfalls solution)
- Post-evaluation layers should never mutate source (from two-phase build patterns)
