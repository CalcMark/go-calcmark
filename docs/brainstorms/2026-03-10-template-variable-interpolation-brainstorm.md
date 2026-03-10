# Template Variable Interpolation (`{{variable}}`)

**Date:** 2026-03-10
**Status:** Ready for planning

## What We're Building

A post-evaluation interpolation pass that replaces `{{variable_name}}` tags in markdown
text (TextBlocks) with the display-formatted final value of that variable from anywhere
in the document. This enables forward references — a summary section at the top of a
document can show results computed 100 lines below.

### User-Facing Behavior

1. In any markdown prose (headings, paragraphs, tables, lists), the user writes
   `` `{{total_cost}}` `` as an inline code tag.
2. After the interpreter evaluates the entire document, a second pass resolves all
   `{{var}}` tags against the final interpreter environment.
3. The resolved value uses the same display formatting as the Preview Pane: locale-aware,
   with K/M/B suffixes, currency symbols, unit labels, and percentage signs.
4. Works everywhere: TUI Preview Pane, `cm convert --to md/html/json`, `--format` flags,
   and `doceval` site generation.
5. If `{{undefined_var}}` references a variable that doesn't exist, the tag is left as-is
   in the output (silent, non-destructive).

### Example

```
## Executive Summary

| Metric | Value |
|--------|-------|
| Total Revenue | `{{total_rev}}` |
| Gross Margin | `{{gross_margin}}` |
| Headcount | `{{team_hc}}` |

---

total_rev = $4.2M
gross_margin = 28%
team_hc = 14 people
```

After evaluation, the table renders with `$4.2M`, `28%`, and `14 people` in the
Value column, even though the calculations appear below the table.

## Why This Approach

- **Document model layer (`InterpolatedSource()`):** A single method on TextBlock that
  resolves `{{var}}` tags against a provided environment. All output consumers (formatters,
  TUI, doceval) call this instead of raw `Source()` when producing output. One
  implementation point — no duplicated logic across formatters.
- **Source immutability preserved:** `TextBlock.source` is never mutated. The original
  `{{var}}` tag round-trips through save/load. Interpolation happens only at render/output
  time on a copy.
- **Forward references via two-pass design:** The interpreter evaluates the entire document
  first (pass 1), then interpolation resolves tags against the final environment (pass 2).
  No dependency graph needed — just a simple string replacement post-evaluation.
- **Display-formatted output:** Uses the same `display.Formatter` as the Preview Pane, so
  `{{total_cost}}` renders as `$1.2M` not `1200000`.

## Key Decisions

1. **Scope:** TextBlocks only — markdown prose, headings, tables, lists. CalcBlocks already
   show results inline.
2. **Forward references:** Yes. The whole point is summary-at-top with calculations below.
3. **Missing variables:** Leave `{{var}}` as-is in output. Silent, non-destructive.
4. **Formatting:** Display-formatted (locale-aware, same as Preview Pane).
5. **Source preservation:** Source stays immutable. Interpolation is render-time only.
6. **Implementation layer:** `InterpolatedSource(env, formatter)` method on TextBlock in
   the document model. All consumers switch from `Source()` to `InterpolatedSource()`.
7. **TUI autocomplete:** Not in v1. Ship interpolation engine first, add `{{` autocomplete
   as a follow-up.
8. **All output paths:** Works with `cm convert`, all `--format` options, TUI Preview Pane,
   and doceval. This is a core language feature, not a TUI feature.

## Open Questions

_None — all resolved during brainstorm._

## Scope Boundaries (YAGNI)

- No format modifiers like `{{var:raw}}` or `{{var:precise}}` in v1.
- No autocomplete on `{{` in the TUI in v1.
- No interpolation inside CalcBlocks.
- No expression evaluation inside tags (e.g. `{{a + b}}` is NOT supported — only variable names).
- No nested interpolation or escaping (`\{{var}}` to suppress).
