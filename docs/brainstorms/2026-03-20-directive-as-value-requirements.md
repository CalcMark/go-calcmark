---
date: 2026-03-20
topic: directive-as-value
---

# Directives as First-Class Values in Expressions

## Problem Frame

`@scale` and `@globals.x` directives work in simple arithmetic expressions (`a = @scale * 3`) but fail in other valid expression contexts. Specifically, `@scale meters / second` is not detected as a calculation at all — the line falls through to plain text. Directives should behave as numbers anywhere a literal number or variable is accepted.

Additionally, when a user explicitly references `@scale` in an expression, the auto-scaling behavior should not double-apply to the result — the user is already incorporating the factor manually. The TUI also lacks autocomplete support for directives.

## Requirements

- R1. **Detector recognition**: The calculation detector must recognize `@directive` tokens the same way it recognizes identifiers and numbers. Lines like `b = @scale meters / second` or `c = @globals.rate * total` must be detected as calculations, not text.
- R2. **Value in any expression position**: `@scale` and `@globals.x` must be usable anywhere a number literal or variable reference is valid — as operands, function arguments, unit-annotated values (`@scale m/s`), and in natural-language functions (`growing at @scale for 3 years`).
- R3. **Scaling opt-out on explicit use**: When an expression directly references `@scale`, auto-scaling must NOT be applied to that expression's result. The rationale is that the user is already incorporating the scale factor intentionally. Downstream references to the resulting variable are unaffected — they scale normally.
- R4. **Autocomplete in TUI**: The TUI suggestion engine must offer `@scale` and `@globals.x` completions when typing `@` in an expression position. Globals completions should be sourced from the frontmatter.

## Success Criteria

- `echo '---\nscale:\n  factor: 3\n  unit_categories: [All]\n---\nb = @scale meters / second' | cm --format json` produces a calculation result of `3 m/s` (not text)
- `a = @scale * 3` with factor 3 produces `9` (not `27` — no double-scaling)
- `a = @scale * 3; b = a + 1` — `b` is auto-scaled normally
- Typing `@` in the TUI shows `@scale` and `@globals.` completions when frontmatter defines them

## Scope Boundaries

- No new directive types (only `@scale` and `@globals` are in scope)
- No changes to frontmatter schema
- No changes to `@scale` auto-scaling logic for expressions that don't explicitly reference `@scale`

## Key Decisions

- **Explicit @scale opts out of auto-scaling**: Using `@scale` directly signals the user is handling scaling themselves. This avoids surprising double-application. The opt-out is expression-local — downstream variable references scale normally.
- **@directive is a number everywhere**: Rather than special-casing contexts, directives resolve to their numeric value and are valid in any position a number is valid.

## Outstanding Questions

### Deferred to Planning

- [Affects R1][Technical] How does the detector currently classify lines? What heuristic needs updating to recognize `@` as a calculation signal?
- [Affects R2][Needs research] Do natural-language function parsers (growing/declining) accept arbitrary expression nodes, or do they expect specific token types? May need parser changes.
- [Affects R3][Technical] Where in the interpreter pipeline is auto-scaling applied? What's the cleanest way to mark an expression as "scaling already handled"?
- [Affects R4][Technical] How does the suggestion engine currently discover completable tokens? Where should directive awareness be added?

## Next Steps

-> `/ce:plan` for structured implementation planning
