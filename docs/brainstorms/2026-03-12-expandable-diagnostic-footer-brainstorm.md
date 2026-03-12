---
title: Expandable Diagnostic Footer
topic: TUI diagnostic display
date: 2026-03-12
status: complete
---

# Expandable Diagnostic Footer

## Problem

When the editor encounters errors (frontmatter validation, calc errors), the diagnostic messages are truncated in the preview pane because it's too narrow. For example, `invalid unit category "Weight" in scale.unit_categories; valid categories: All, Area, Currency, Custom, DataSize, Energy, Length, Mass, Number, Power, Speed, Temperature, Volume` gets cut to `⚠ frontmatter: invalid unit category "Weight" in ...`. The user can't see the valid categories list — the most actionable part of the error.

The existing 2-line context footer shows error details when the cursor is on an error line, but 2 lines isn't enough for structured diagnostics with hints, valid options, and (eventually) help links.

## What We're Building

Evolve the existing context footer from a fixed 2-line component to a dynamically-sized component that expands up to 4 lines when the cursor is on an error line.

### Behavior

- **Default:** 2 lines (current behavior, unchanged for non-error lines)
- **On error line:** Expands to up to 4 lines with structured diagnostic content
- **Trigger:** Cursor moves onto a line with an error → footer expands. Cursor moves away → footer shrinks back to 2.
- **Height budget:** The extra 2 lines come from the editor content area. On a 24-row terminal, this means losing 2 rows of source/preview when viewing an error — acceptable trade-off for seeing the full diagnostic.

### Diagnostic Layout (Structured)

```
⚠ Invalid unit category "Weight"
💡 Did you mean "Mass"? Valid: All, Area,
  Currency, Custom, DataSize, Energy,
  Length, Mass, Number, Power, Speed, ...
```

Line 1: Error icon + cleaned error message (no code prefix, no "frontmatter: frontmatter:" duplication).
Lines 2-4: Actionable hint and context (valid options list, suggestions, explanation). Word-wrapped to fill available width.

### What It Covers

- Frontmatter validation errors (invalid categories, malformed YAML)
- Calc errors (undefined variables, type mismatches, division by zero)
- Semantic errors (immutable reassignment, incompatible units)

The structured hints come from the existing `ParseErrorForDisplay()` and `GetHintForDiagnostic()` infrastructure in `components/errors.go`. Frontmatter errors need a new hint mapping.

## Why This Approach

- **Evolves existing component** rather than adding new chrome — simpler layout math, one height variable to adjust instead of managing two footer-like components.
- **4-line cap is predictable** — the layout shift is small and bounded. No content-driven sizing that could surprise the user.
- **Cursor-triggered** matches the existing context footer behavior — no new interaction patterns to learn.
- **Structured layout** (message + hint + context) is more useful than raw word-wrapped error text. Shows the user what to do, not just what went wrong.

## Key Decisions

1. **Evolve the context footer** — not a separate component. The footer already shows errors; this makes it better at it.
2. **Max 4 lines** — enough for error + hint + valid options. Bounded, predictable.
3. **Cursor on error line triggers expansion** — same interaction as current footer, just taller.
4. **Structured content** — error icon + message on line 1, hint + context on lines 2-4. Not raw word-wrap.
5. **Defer calcmark.org help links** — links require URL scheme, error-code-to-page mapping, and OSC8 terminal hyperlink support. Ship the structured diagnostics first.
6. **Fix "frontmatter: frontmatter:" prefix** — the double prefix in error messages should be cleaned up as part of this work.

## Scope

### In scope (v1)
- Dynamic context footer height (2 default, up to 4 on error)
- Recalculate pane heights when footer size changes
- Structured diagnostic rendering (icon + message + hint + context)
- Frontmatter error hints (e.g., "Did you mean 'Mass'?" for "Weight")
- Clean up double "frontmatter:" prefix in error messages
- Existing calc error hints already work via `ParseErrorForDisplay`

### Out of scope (follow-up)
- Clickable help links to calcmark.org (OSC8 hyperlinks)
- Error-code-to-documentation-page URL mapping
- "Did you mean?" fuzzy matching for typos beyond simple cases
- Keyboard shortcut to expand/collapse diagnostic detail

## Resolved Questions

- **Replace or augment context footer?** → Evolve it. One component, simpler.
- **How tall?** → Up to 4 lines, fixed cap.
- **When to show?** → Cursor on error line, same as current footer.
- **Links in v1?** → No, defer to follow-up.
