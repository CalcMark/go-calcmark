# Frontmatter Stability & Preview Pane Alignment

**Date:** 2026-02-24
**Status:** Brainstorm

## What We're Building

Three related improvements to how the TUI editor handles YAML frontmatter and the preview pane's Globals display:

1. **Remove inline `@exchange`/`@global` assignment syntax from the grammar** — All exchange rates and globals must be defined in frontmatter. Inline `@exchange.USD_JPY = 150` and `@global.tax_rate = 0.25` are no longer valid Calcmark syntax. This is a deliberate breaking change that reinforces the "define once, flow down" principle of frontmatter.

2. **Fix vertical alignment between YAML frontmatter and the Globals panel** — Structural YAML lines (`---`, `exchange:`, `globals:`) should produce blank rows in the preview pane so that each key-value pair in the YAML source aligns horizontally with its rendered counterpart in the Globals panel.

3. **Remove the `[g]` keybinding hint** — The `[g]` hint displayed in the Globals panel header has no corresponding key handler. It is dead UI and should be removed.

## Why This Approach

### Removing inline assignments (grammar-level)

Frontmatter exists to make documents predictable. Inline `@exchange` redefinition undermines that by allowing mid-document rate changes that are hard to trace. The current implementation also writes inline values back into the `Frontmatter` struct (`evaluator.go:updateFrontmatterFromNodes`), which mutates the user's declared values — a confusing side effect.

Exchange rates were an exception to the "no redefinition" rule, but in practice this creates more confusion than flexibility. If a user needs different rates at different points, they should use explicit variables or separate documents.

Removing at the grammar level (not just semantic checker) keeps the language surface clean. Users who try the old syntax will get a parse error, making it clear the syntax doesn't exist rather than suggesting it's a misuse of valid syntax.

### Vertical alignment

The current rendering uses a `Globals` header row and doesn't account for non-value YAML lines, causing the preview to drift out of sync with the source. The Globals header is removed entirely from the UI — the YAML structure itself (`exchange:`, `globals:`) provides the section context. Blank preview lines are emitted for structural YAML lines (`---`, `exchange:`, `globals:`), so each frontmatter value aligns 1:1 with its source line.

### Removing `[g]`

No key handler for `g` exists in `key_dispatch.go`. The hint promises functionality that doesn't exist.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Inline `@exchange`/`@global` removal scope | Remove entirely from grammar | "Define once, flow down" — frontmatter is the single source of truth |
| Backwards compatibility | Accept breaking change | Deliberate design correction; document in changelog |
| Frontmatter write-back | Remove `updateFrontmatterFromNodes` | No inline assignments means no write-back needed |
| Preview alignment strategy | Blank rows for structural YAML lines | 1:1 line mapping between source and preview |
| Globals panel header | Remove entirely from UI | YAML structure (`exchange:`, `globals:`) provides section context; no separate header needed |
| `[g]` hint | Remove entirely | Dead UI with no handler |

## Scope & Impact

### Files affected (grammar removal)

- `spec/grammar/` — Remove `FrontmatterAssignment` production rules for `@exchange` and `@global`
- `spec/ast/` — Remove or deprecate `FrontmatterAssignment` AST node type
- `spec/semantic/checker.go` — Remove `checkFrontmatterAssignment` logic
- `impl/interpreter/variables.go` — Remove `evalFrontmatterAssignment`
- `impl/document/evaluator.go` — Remove `updateFrontmatterFromNodes`
- `testdata/spec/valid/features/exchange_rates.cm` — Update golden tests (remove redefinition examples)
- Any other golden tests using inline `@exchange`/`@global` syntax

### Files affected (alignment fix)

- `cmd/calcmark/tui/editor/view_panes.go` — `renderPreviewPaneAligned` needs structural line detection
- `cmd/calcmark/tui/editor/view_overlays.go` — Remove Globals header entirely from the UI; remove `[g]` hint
- `cmd/calcmark/tui/editor/view_state.go` — Expose frontmatter line classification (structural vs. value)
- `cmd/calcmark/tui/editor/aligned.go` — `ComputeAlignedModel` may need updates for frontmatter blank-row injection

### Files affected (`[g]` removal)

- `cmd/calcmark/tui/editor/view_overlays.go` — Remove `[g]` hint from `renderGlobalsPanel`

## Open Questions

None — all questions resolved during brainstorm.
