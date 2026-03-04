---
title: "feat: Add NL examples and position-aware variables to autosuggest"
type: feat
status: completed
date: 2026-03-04
brainstorm: docs/brainstorms/2026-03-04-nl-autosuggest-examples-brainstorm.md
related_todo: .planning/todos/pending/2026-02-07-function-help-natural-language-examples.md
---

# feat: Add NL examples and position-aware variables to autosuggest

## Overview

Two autosuggest enhancements:

1. **NL function examples** — Show NL example rows alongside function-call suggestions. When a user types a prefix matching a function name or NL alias keyword, they see two rows: the parenthesized function form (`fn`) and a concrete NL example (`nl`). Selecting the NL row inserts a realistic, editable example like `average of 1, 2, 3`.

2. **Position-aware variable suggestions** — Filter variable suggestions to only show variables defined above the current cursor line. Currently `VariableSuggestionSource` returns all variables from the environment regardless of position.

## Problem Statement / Motivation

Users forget NL syntax for CalcMark functions. The autosuggest popup currently only offers the parenthesized form (e.g., `convert_rate(`). Users who prefer the NL form (e.g., `5 MB/s per minute`) have no discoverability path — they must remember the syntax or check documentation.

Additionally, variable suggestions currently include variables defined anywhere in the document, including below the cursor. This is misleading — variables defined below the cursor aren't yet in scope at that point in the document.

## Proposed Solution

### Architecture: Single-word prefix matching

The autocomplete uses **single-word prefix matching** (the existing `isWordRune`-based `getCurrentWordPrefix()`). Multi-word NL aliases like `"average of"` are matched by their first word — typing `aver` matches `"average of"`. This avoids changing the space-key handler (which dismisses autocomplete) or the prefix extraction logic.

Rationale: This matches how most IDEs handle multi-word completions and keeps the change minimal.

### Architecture: Feature registry as NL data source

Add an `Example` field to the `Alias` struct in `spec/features/registry.go`. The `FunctionSuggestionSource` is extended to also read from the feature registry, emitting NL rows alongside existing fn rows. The fn rows continue to come from `interpreter.BuiltinFunctions`.

### Scope of NL keyword triggers

Functions with **parseable NL aliases** in the feature registry get NL keyword triggering (typing the alias name prefix surfaces the NL row). These are:

| Function | Alias Name | Example | Keyword trigger |
|---|---|---|---|
| `avg` | `average of` | `average of 1, 2, 3` | `aver` |
| `sqrt` | `square root of` | `square root of 16` | `squa` |
| `transfer_time` | `transfer...across` | `transfer 1 GB across regional gigabit` | `trans` |
| `read` | `read...from` | `read 100 MB from ssd` | (same as fn name) |
| `compress` | `compress...using` | `compress 1 GB using gzip` | `comp` |

Functions **without** parseable NL aliases get NL rows triggered only by function name prefix:

| Function | Example | Trigger |
|---|---|---|
| `accumulate` | `100 MB/s over 1 day` | `accum` |
| `convert_rate` | `5 MB/s per minute` | `conver` |
| `capacity` | `10 TB at 2 TB per disk` | `capac` |
| `downtime` | `99.9% downtime per month` | `down` |

These 4 functions use NL keywords (`over`, `per`, `at`) that overlap with existing keyword features in the registry, so no parseable aliases are added for them. The NL example is stored on a new `Feature.NLExample` field (separate from `Feature.Example` which is used for help display).

### Suggestion field mapping for NL rows

| Field | fn row (existing) | nl row (new) |
|---|---|---|
| `Name` | `"avg (average, mean)"` | `"average of"` (alias name, cleaned of `...` template notation) |
| `Category` | `"Math"` | `"example"` (new category for visual tag + sort order) |
| `Syntax` | `"avg(a, b, c, ...)"` (contains `(`) | `"average of 1, 2, 3"` (no `(`, prevents `(` appending) |
| `Description` | `"Calculate the average..."` | `""` (empty — syntax is self-documenting) |
| `InsertText` | `"avg"` (+ `(` appended by acceptAutocomplete) | `"average of 1, 2, 3"` (inserted as-is, no `(`) |

### Popup rendering

Add `fn`/`nl` category tags to the beginning of each popup row:

```
┌──────────────────────────────────┐
│ fn  avg(a, b, c, ...) (average)  │
│ nl  average of 1, 2, 3           │
├──────────────────────────────────┤
│ fn  accumulate(rate, time)       │
│ nl  100 MB/s over 1 day          │
└──────────────────────────────────┘
```

The `fn`/`nl` tag is derived from the `Category` field: `"example"` → `"nl"`, all other function categories → `"fn"`.

### Sort order

NL rows for a function appear immediately after their fn row. Achieved by:
1. `FunctionSuggestionSource.GetSuggestions` emits `[fn_row, nl_row]` pairs in sequence
2. `CombinedSuggestionSource` sorting must keep pairs together. Use `sort.SliceStable` and give NL rows the **same** `catOrder` value as their parent function's category (e.g., an NL row for `avg` gets order 0 like "Math"). The `"example"` category is used only for display tagging, not for sort order
3. Since `FunctionSuggestionSource` emits fn then nl in order, and `sort.SliceStable` preserves insertion order for equal keys, pairs stay adjacent

## Technical Considerations

### `acceptAutocomplete` — no `(` for NL rows

The heuristic at `model.go:1021` (`strings.Contains(selected.Syntax, "(")`) correctly skips `(` appending for NL rows because their `Syntax` field contains no parentheses. No changes needed to `acceptAutocomplete`.

### Prefix matching for template aliases

Aliases like `"transfer...across"` use `...` as a gap placeholder. For prefix matching, the text before `...` is extracted: `"transfer"`. Typing `trans` matches `"transfer"`. The `...` and text after it are ignored for matching purposes.

### Popup width calculation

`calculatePopupDimensions` at `model.go:942` computes width from `len(s.Syntax)` and `len(s.Name)`. NL example text may be longer than function signatures. The category tag prefix (`fn ` / `nl `) adds 4 characters. Width calculation must account for both.

### Undo behavior

No changes needed. `acceptAutocomplete` records `OpReplace` with `OldText = prefix` and `NewText = insertText`. For NL rows, `insertText` is the full example. `ForceBoundary()` calls ensure single-step undo.

### Minimum prefix length

Add a 2-character minimum prefix threshold in `updateAutocompleteState` (`model.go:898`). If `len(prefix) < 2`, dismiss the popup (or don't show it). This reduces noise from single-character matches like `a` matching `avg`, `accumulate`, `at`, etc. The explicit Tab trigger (`triggerAutocomplete`) should also respect this threshold.

### Popup height cap

Keep the existing 8-item cap with scroll indicator. With fn+nl pairs and a 2-char minimum, the number of matches is naturally reduced.

## Acceptance Criteria

### Functional

- [x] Typing a function name prefix (e.g., `avg`, `conver`) shows both fn and nl rows in the popup
- [x] Typing an NL alias keyword prefix (e.g., `aver`, `squa`, `trans`) shows the nl row (and fn row if it also matches)
- [x] Selecting an nl row inserts the concrete example text (e.g., `average of 1, 2, 3`)
- [x] Selecting an nl row does NOT append `(`
- [x] Selecting an fn row continues to append `(` (existing behavior preserved)
- [x] Popup rows show `fn` or `nl` category tags
- [x] fn and nl rows for the same function are adjacent in the popup
- [x] Undo after accepting an nl suggestion restores the original prefix
- [x] Autosuggest popup does not appear until at least 2 characters have been typed

### Variable Position Awareness

- [x] Variable suggestions only include variables defined on lines above the cursor
- [x] Variables defined on the same line or below the cursor are excluded
- [x] Built-in constants (e.g., `E`, `PI`) are always included regardless of position

### Non-Functional

- [x] All 9 NL-capable functions have example text
- [x] Feature registry `Alias.Example` field is the single source of NL example data
- [x] No changes to `acceptAutocomplete` logic (NL rows naturally bypass `(` appending)
- [x] Existing autocomplete tests updated for tag rendering and still pass

## Implementation Phases

### Phase 1: Spec layer — Add `Alias.Example` and populate NL examples

**Files:**
- `spec/features/registry.go:28-31` — Add `Example string` field to `Alias` struct
- `spec/features/registry.go:136-235` — Populate `Example` on parseable aliases and add NL examples to `Feature.Example` for functions without parseable aliases

**Tasks:**
1. Add `Example` field to `Alias` struct
2. Add examples to existing parseable aliases: `avg`→`"average of"`, `sqrt`→`"square root of"`, `transfer_time`→`"transfer...across"`, `read`→`"read...from"`, `compress`→`"compress...using"`
3. For functions without parseable aliases (`accumulate`, `convert_rate`, `capacity`, `downtime`), add NL example text to a new `NLExample` field on `Feature` (do NOT overwrite the existing `Feature.Example` which is used for help display)
4. Write unit tests verifying `Alias.Example` is populated for all parseable aliases

**Success criteria:** All parseable aliases have non-empty `Example` fields. Tests pass.

### Phase 2: TUI autocomplete — Emit NL suggestion rows

**Files:**
- `cmd/calcmark/tui/editor/autocomplete.go:13-62` — Extend `FunctionSuggestionSource` to emit NL rows
- `cmd/calcmark/tui/editor/autocomplete.go:167-182` — Add `"example"` to `catOrder` in `CombinedSuggestionSource`

**Tasks:**
1. Import `spec/features` in `autocomplete.go`
2. In `FunctionSuggestionSource`, after emitting the fn row, look up the function in the feature registry and emit an nl row for each parseable alias with a non-empty `Example`
3. For functions without parseable aliases but with NL-style `Feature.Example`, emit an nl row using the feature example
4. Set nl row fields: `Category: "example"`, `Syntax: alias.Example`, `InsertText: alias.Example`, `Name: cleanAliasName(alias.Name)`. Also store the parent function's category (e.g., "Math") for sort order
5. Add `cleanAliasName()` helper to strip `...` from template aliases (e.g., `"transfer...across"` → `"transfer across"`)
6. Match prefix against cleaned alias names (first word before `...` or space)
7. Update `CombinedSuggestionSource` to use `sort.SliceStable` and sort NL rows by their parent function's category (not `"example"`), so fn/nl pairs stay adjacent
8. Write unit tests for `FunctionSuggestionSource` verifying NL rows are emitted and correctly populated

**Success criteria:** `GetSuggestions("avg")` returns both fn and nl rows. `GetSuggestions("aver")` returns the nl row. NL rows have correct field values.

### Phase 3: Popup rendering — Add category tags

**Files:**
- `cmd/calcmark/tui/editor/view_overlays.go:40-68` — Add `fn`/`nl` tag rendering
- `cmd/calcmark/tui/editor/model.go:942-967` — Update popup width calculation for tag prefix

**Tasks:**
1. In `renderAutocompletePopup`, derive tag from `Category`: `"example"` → `"nl"`, all function categories → `"fn"`, units → category abbreviation, variables → `"var"`
2. Prepend tag to content: `" fn  " + content` or `" nl  " + content`
3. Update `calculatePopupDimensions` to account for 4-char tag prefix in width calculation
4. Add 2-character minimum prefix check in `updateAutocompleteState` (`model.go:898`): if `len(prefix) < 2`, dismiss/don't show popup. Also apply to `triggerAutocomplete`
5. Write catwalk tests verifying popup renders with `fn`/`nl` tags
6. Write catwalk test verifying single-character prefix does not trigger popup

**Success criteria:** Popup visually shows `fn`/`nl` labels. Width accommodates longer NL examples without excessive truncation.

### Phase 4: Catwalk integration tests

**Files:**
- `cmd/calcmark/tui/editor/testdata/autocomplete_nl` (new) — NL-specific catwalk tests
- `cmd/calcmark/tui/editor/testdata/autocomplete` — Verify existing tests still pass with new tags

**Tasks:**
1. Catwalk test: type `avg` → popup shows fn and nl rows → select nl row → `average of 1, 2, 3` inserted (no `(`)
2. Catwalk test: type `aver` → popup shows nl row → accept → correct insertion
3. Catwalk test: type `conver` → popup shows fn and nl rows for `convert_rate`
4. Catwalk test: accept nl then Ctrl+Z → restores prefix
5. Update existing autocomplete catwalk test expectations if popup rendering changes affect them (tag prefix)

**Success criteria:** All catwalk tests pass. Existing autocomplete tests updated for new tag rendering.

### Phase 5: Position-aware variable suggestions (spec-driven)

**Files:**
- `spec/semantic/environment.go:10-11` — `VarInfo` already tracks `Range` (definition location)
- `spec/semantic/checker.go:11-69` — `findSimilarNames` (existing matching logic)
- `cmd/calcmark/tui/editor/autocomplete.go:114-146` — `VariableSuggestionSource` becomes a thin wrapper
- `cmd/calcmark/tui/editor/model.go:391-402` — Pass cursor line to the `getVariables` closure

**Architecture:** The variable suggestion logic belongs in the spec/interpreter layer, not the TUI. The semantic checker already has:
- `VarInfo` struct (`spec/semantic/environment.go:10`) with `Range` tracking definition position
- `findSimilarNames` (`spec/semantic/checker.go:11-69`) with Levenshtein + prefix matching
- Diagnostic severity levels: `Error`, `Warning`, `Hint` (`spec/semantic/diagnostics.go:5-14`)

The TUI's `VariableSuggestionSource` should be a thin wrapper that calls into the spec/interpreter layer for variable discovery and matching. The matching intelligence lives in the spec layer; the TUI just presents results.

**Tasks:**
1. Add a method to the interpreter environment (or a new utility in `spec/semantic/`) that returns variables visible at a given line position, using `VarInfo.Range` for filtering. Built-in constants (no definition range) are always included
2. Reuse `findSimilarNames` for matching — it already handles prefix matching, case-insensitive matching, and Levenshtein distance for typos
3. Simplify `VariableSuggestionSource` to be a thin wrapper: call the spec/interpreter layer method, convert results to `[]Suggestion`
4. In the `getVariables` closure at `model.go:391`, pass `m.cursorLine` to the position-aware method
5. Update `updateAutocompleteState` to pass current `m.cursorLine` when querying suggestions (may require extending the `SuggestionSource` interface or passing via closure)
6. Write unit tests in the spec/interpreter layer verifying position filtering and matching
7. Write catwalk test: define `x = 10` on line 1, move to line 3, type `x` → see suggestion; define `y = 20` on line 5, cursor on line 3, type `y` → no suggestion

**Design principle:** The TUI is a light wrapper. Matching intelligence lives in spec/interpreter where it can be shared with "did you mean" diagnostics and any future consumers (LSP, REPL, etc.).

**Success criteria:** Variables below cursor are excluded. Built-in constants always appear. Variable matching reuses spec-layer `findSimilarNames`. TUI `VariableSuggestionSource` is a thin wrapper. Tests pass.

## Dependencies & Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Existing autocomplete tests break from tag rendering change | High | Low | Update test expectations in Phase 4 |
| Popup too wide from NL examples | Low | Low | Truncation already works at `view_overlays.go:60-62` |
| NL alias keyword overlaps with non-function suggestions | Low | Medium | `catOrder` sorting keeps NL rows grouped after fn rows |
| Variable-to-line mapping not available in environment | Medium | Medium | Fall back to `GetLineResults()` which tracks which lines produced which variables |
| `SuggestionSource.GetSuggestions` interface lacks cursor context | Medium | Low | Either add a `GetSuggestionsAt(prefix, line)` method or pass line via closure |

## References & Research

### Internal References
- Brainstorm: `docs/brainstorms/2026-03-04-nl-autosuggest-examples-brainstorm.md`
- Related TODO: `.planning/todos/pending/2026-02-07-function-help-natural-language-examples.md`
- Feature registry: `spec/features/registry.go:28-41` (Alias, Feature structs)
- Suggestion data model: `cmd/calcmark/tui/components/suggest.go:12-24`
- Function suggestion source: `cmd/calcmark/tui/editor/autocomplete.go:13-62`
- Accept logic: `cmd/calcmark/tui/editor/model.go:1004-1062`
- Popup rendering: `cmd/calcmark/tui/editor/view_overlays.go:15-82`
- Catwalk tests: `cmd/calcmark/tui/editor/testdata/autocomplete`
- Testing guide: `cmd/calcmark/tui/editor/TESTING.md`

### Institutional Learnings
- Overlay rendering must use `OverlayStyle` with explicit backgrounds (see `docs/solutions/ui-bugs/overlay-compositing-ansi-state-bleed-through.md`)
- Mode transitions must reset all related state (see `docs/solutions/ui-bugs/tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md`)
- Catwalk tests are the authoritative way to test TUI features (see `CLAUDE.md`)
