# Autosuggest Enhancements: NL Examples + Position-Aware Variables

**Date:** 2026-03-04
**Status:** Ready for planning

## What We're Building

Two autosuggest enhancements:

1. **NL function examples** — Show natural language example rows alongside function-call suggestions in the popup. Selecting an NL row inserts a concrete, editable example.

2. **Position-aware variable suggestions** — Filter the existing variable autosuggest to only show variables defined *above* the current cursor position, not all variables in the document.

### User Stories

> I keep forgetting how to naturally type something like `10 MB/s per day`. I can type `conver` to get the popup for `convert_rate` but nothing helpful for the NL version. I want to see and insert a concrete NL example directly from the autosuggest.

> When I type a variable prefix, I should only see variables that have already been defined above my cursor — not variables from later in the document that I can't reference yet.

## Why This Approach

The feature registry (`spec/features/registry.go`) already models parseable NL aliases for functions. Adding an `Example` field to the `Alias` struct keeps NL knowledge centralized in the spec layer. The TUI autocomplete then reads from the registry to generate NL suggestion rows — no duplication.

## Key Decisions

1. **Separate rows, not toggle** — Each flavor (function, NL) is its own row in the popup. Uses existing up/down navigation. Simple and scannable.

2. **Concrete examples, not templates** — NL rows insert realistic values like `5 MB/s per minute` rather than placeholders like `<rate> per <time_unit>`. Users edit in-place.

3. **All NL-capable functions** — Ship NL examples for every function with a parseable alias:
   - `avg` → `average of 1, 2, 3`
   - `sqrt` → `square root of 16`
   - `accumulate` → `100 MB/s over 1 day`
   - `convert_rate` → `5 MB/s per minute`
   - `transfer_time` → `transfer 1 GB across regional gigabit`
   - `read` → `read 100 MB from ssd`
   - `compress` → `compress 1 GB using gzip`
   - `capacity` → `10 TB at 2 TB per disk`
   - `downtime` → `99.9% downtime per month`

4. **NL keywords trigger suggestions** — Typing `aver`, `squa`, `trans`, `comp` etc. surfaces NL suggestions. Makes NL discoverable from natural typing patterns.

5. **Category label visual style** — NL rows show a `nl` category tag; function rows show `fn`. Consistent with existing category display pattern.

6. **Feature registry as single source of truth** — Add `Example` field to `Alias` struct in `spec/features/registry.go`. No NL knowledge duplicated in the TUI layer.

7. **Minimum 2-character prefix** — Autosuggest popup does not appear until at least 2 characters have been typed. Reduces visual noise from single-character matches.

8. **Spec-driven variable suggestions** — Variable matching intelligence lives in the spec/interpreter layer, not the TUI. The semantic checker already has `findSimilarNames` (`spec/semantic/checker.go`) with Levenshtein + prefix matching, and `VarInfo` tracks definition ranges. The TUI's `VariableSuggestionSource` becomes a thin wrapper calling into the spec layer. This ensures "did you mean" diagnostics and autosuggest use the same logic, and future consumers (LSP, REPL) get it for free.

9. **Position-aware variable filtering** — The existing `VariableSuggestionSource` returns all variables from the environment. Enhance it to accept the current cursor line and only return variables defined on lines above it. The `getVariables` closure already has access to the model; it needs the cursor line to filter.

## Scope

### In Scope

- Add `Example` field to `spec/features/Alias` struct
- Populate examples for all parseable NL aliases in the feature registry
- Extend `FunctionSuggestionSource` to emit NL suggestion rows
- Match against NL alias names (not just function names) for prefix filtering
- Display NL rows with `nl` category label in popup
- Insert concrete example text on accept (no `(` appended)
- Filter `VariableSuggestionSource` to only return variables defined above the cursor line

### Out of Scope

- Cursor positioning within inserted example text
- Tab-stop or snippet-style placeholder navigation
- NL examples for non-function suggestions (units, variables)
- Changes to the context footer / parameter help system

## Open Questions

None — all questions resolved during brainstorming.
