---
title: "Frontmatter Stability & Preview Pane Alignment"
type: refactor
status: active
date: 2026-02-24
brainstorm: docs/brainstorms/2026-02-24-frontmatter-stability-brainstorm.md
---

# Frontmatter Stability & Preview Pane Alignment

## Overview

Three coordinated changes that reinforce frontmatter as the single source of truth for exchange rates and globals:

1. **Remove inline `@exchange`/`@global` assignment syntax from the grammar** — breaking change
2. **Fix vertical alignment** between YAML frontmatter and the preview pane Globals display
3. **Remove dead `[g]` keybinding hint** from the Globals panel

## Problem Statement

Inline `@exchange`/`@global` syntax undermines frontmatter's "define once, flow down" principle. The evaluator writes inline values back into the `Frontmatter` struct (`updateFrontmatterFromNodes`), mutating user-declared YAML. The preview pane's Globals display is vertically misaligned with the YAML source because the "Globals" header row offsets values by one line. The `[g]` hint promises a keybinding that doesn't exist.

## Proposed Solution

Remove inline assignment syntax at the grammar level (lexer + parser), clean up all downstream consumers (AST, semantic checker, interpreter, evaluator, dependency analyzer, formatters, TUI results), fix the preview pane alignment by emitting blank rows for structural YAML lines, and remove the dead `[g]` UI.

## Technical Approach

### Phase 1: Remove inline `@exchange`/`@global` from the grammar

The full removal chain, in dependency order:

#### 1a. Lexer — Remove `AT_PREFIX` token

**File**: `spec/lexer/lexer.go:953-982`

Remove the `@` tokenization block that emits `AT_PREFIX` when `@` is followed by an identifier character. The `@` character will fall through to the default case and produce an error token, giving the user a clear parse error.

**File**: `spec/lexer/token.go:37`

Remove the `AT_PREFIX` token type constant and its string representation.

**Tests to remove**: `spec/lexer/lexer_reserved_test.go:603-696` — `TestFrontmatterTokenization` and `TestAtPrefixTokenValue`.

**Edge case**: The `@` character is not used elsewhere in Calcmark (no email addresses in calc blocks, no decorators). Removing `AT_PREFIX` is safe. If a user types `@exchange.USD_JPY = 150`, they'll get an unexpected character error on `@`.

#### 1b. Parser — Remove `parseFrontmatterAssignment`

**File**: `spec/parser/rdparser.go:211-306`

Remove the `AT_PREFIX` check in `parseStatement()` (line 214) and delete the entire `parseFrontmatterAssignment()` function (lines 258-306).

**Tests to remove**: `spec/parser/rdparser_test.go:66-163` — entire `TestFrontmatterAssignment` function.

#### 1c. AST — Remove `FrontmatterAssignment` node

**File**: `spec/ast/nodes.go:317-335`

Remove the `FrontmatterAssignment` struct. This will cause compile errors in all downstream consumers, which is exactly what we want — the compiler finds every reference.

#### 1d. Semantic Checker — Remove `checkFrontmatterAssignment`

**File**: `spec/semantic/checker.go:142-143,208-216`

Remove the `case *ast.FrontmatterAssignment` in the type switch and the `checkFrontmatterAssignment` method.

#### 1e. Interpreter — Remove `evalFrontmatterAssignment`

**File**: `impl/interpreter/interpreter.go:57-58` — Remove `case *ast.FrontmatterAssignment` from `evalNode`.

**File**: `impl/interpreter/variables.go:26-97` — Remove `evalFrontmatterAssignment()`, `parseExchangeKey()`, and `isValidCurrencyCode()`.

**Important**: `parseExchangeKey()` and `isValidCurrencyCode()` may be used by other code paths (frontmatter parsing). Check references before deleting — if they're only used by `evalFrontmatterAssignment`, they go. If shared, they stay.

#### 1f. Document Evaluator — Remove `updateFrontmatterFromNodes`

**File**: `impl/document/evaluator.go:465-511`

Remove the `updateFrontmatterFromNodes()` function and its call site (line 467). With no inline assignments, there's nothing to write back.

**Institutional learning**: This removal also fixes the `rawSource` mutation issue — `SetExchangeRate()`/`SetGlobal()` clear `rawSource` to force YAML reconstruction. Without write-back, `rawSource` stays intact, preserving the user's exact YAML text (per `docs/solutions/ui-bugs/frontmatter-editing-keyboard-dispatch-fixes.md`).

#### 1g. Dependency Analyzer — Remove `FrontmatterAssignment` handling

**File**: `spec/document/deps.go:57-66,104-106`

Remove the `FrontmatterAssignment` cases in `AnalyzeBlock()` and `extractIdentifiers()`.

#### 1h. TUI Editor Results — Remove `FrontmatterAssignment` case

**File**: `cmd/calcmark/tui/editor/results.go:260-261`

Remove the `case *ast.FrontmatterAssignment` in `getAssignmentVarName()`.

#### 1i. Formatters — Remove `FrontmatterAssignment` handling

**File**: `format/markdown_formatter_test.go:283-315` — Remove `TestMarkdownFormatterFiltersFrontmatterBlocks`.

Check all formatters (`format/markdown_formatter.go`, `format/json_formatter.go`, `format/plain_formatter.go`) for `FrontmatterAssignment` references.

#### 1j. Golden tests — Update or remove inline assignment examples

| Golden File | Action |
|-------------|--------|
| `testdata/spec/valid/features/exchange_rates.cm:42,46` | Remove "Inline exchange rate assignment" section (`@exchange.USD_GBP = 0.79` and `@exchange.USD_GBP = 0.72`) |
| `testdata/eval/success/features/currency_conversion.cm:55,61` | Remove "Inline exchange rate assignment" section |
| `eval_test.go:288-296` | Remove the `@exchange inline then convert` test case |

#### 1k. Interpreter tests — Remove frontmatter assignment tests

**File**: `impl/interpreter/interpreter_test.go:282-397`

Remove `TestFrontmatterGlobalAssignment`, `TestFrontmatterExchangeRateAssignment`, `TestFrontmatterInvalidExchangeKeys`, `TestFrontmatterUnknownNamespace`.

### Phase 2: Fix preview pane vertical alignment

#### 2a. Classify frontmatter source lines as structural vs. value

The preview pane needs to know which frontmatter source lines are "structural" (should render as blank rows) vs. "value" (should render as key-value pairs).

**Structural lines**: `---` (open/close delimiters), `exchange:`, `globals:`, blank lines, comments.
**Value lines**: `  USD_EUR: 0.6`, `  tax_rate: 0.32`, etc.

**Approach**: Add a classification method to the frontmatter model or the TUI's view state. Pattern matching on trimmed source text is sufficient since YAML frontmatter has a constrained format:
- Lines matching `^---$` → structural
- Lines matching `^(exchange|globals):` → structural
- Lines that are blank or comment-only → structural
- All other frontmatter lines → value

**File**: `cmd/calcmark/tui/editor/view_state.go` — Add `classifyFrontmatterLine(line string) bool` (returns true for structural).

#### 2b. Remove the Globals panel header entirely

**File**: `cmd/calcmark/tui/editor/view_overlays.go`

The `renderGlobalsPanel()` function currently renders a header line (`"▸ Globals (N)"` collapsed, `"▾ Globals (N)"` expanded). Remove this header entirely. The YAML structure (`exchange:`, `globals:`) provides section context.

When frontmatter is present, the globals panel becomes a pure 1:1 mapping of frontmatter value lines to their rendered values. No header, no collapse/expand toggle.

When frontmatter is NOT present, there are no globals or exchange rates (since inline assignment is removed), so no globals panel is needed at all.

**Impact on `globalsExpanded` state**: The expand/collapse toggle (`Ctrl+F` sets `globalsExpanded = true`) may need rethinking. With no header, there's no collapsed state. The globals panel is always "expanded" when frontmatter exists. Consider removing the `globalsExpanded` field or keeping it for potential future use but defaulting to true.

#### 2c. Emit blank preview rows for structural YAML lines

**File**: `cmd/calcmark/tui/editor/view_panes.go` — `renderPreviewPaneAligned()` (line 177)

Currently, ALL frontmatter lines get their preview content replaced by globals panel lines via `globalsPanelIdx`. Instead:

1. For each frontmatter preview line, check if the corresponding source line is structural
2. If structural → emit a blank preview line (no globals panel content)
3. If value → emit the corresponding globals panel value

This requires the `globalsPanelLines` array to contain ONLY value entries (no header), and the iteration to skip structural lines when indexing into `globalsPanelLines`.

**Expected alignment result**:

```
Source                          Preview
1  ---                          (blank)
2  exchange:                    (blank)
3    USD_EUR: 0.6               USD_EUR          0.6000
4    USD_JPY: 0.001             USD_JPY          0.0010
5  globals:                     (blank)
6    tax_rate: 0.32             tax_rate         0.32
7  ---                          (blank)
```

#### 2d. Update `ComputeAlignedModel` if needed

**File**: `cmd/calcmark/tui/editor/aligned.go`

The `IsFrontmatter` field on `AlignedLine` (line 44) exists but is never set in `ComputeAlignedModel`. Currently, frontmatter detection happens in `view.go:271` via `al.BlockID == "" && !al.IsCalc`. If the structural-vs-value classification needs to flow through the aligned model, set `IsFrontmatter` during alignment computation.

#### 2e. Remove the separator line after globals

The horizontal separator (`"---"` repeated chars) appended after globals content (`view_overlays.go`) should also be removed — the closing `---` delimiter in the YAML source provides this naturally.

#### 2f. Handle edge cases

- **Empty sections** (`exchange:` with no entries): Blank row for `exchange:`, no value rows. Next section or `---` follows.
- **Missing sections** (no `globals:` key): Only `exchange:` entries shown. No blank row for absent `globals:`.
- **Comments in YAML**: Blank preview row (structural).
- **Malformed YAML**: `frontmatterErr` already handles this — show error indicator, don't attempt alignment.

#### 2g. Catwalk tests

**Per CLAUDE.md and `TESTING.md`**: Every user-facing TUI change MUST have a catwalk test.

Write catwalk tests for:
- Frontmatter with both `exchange:` and `globals:` — verify 1:1 alignment
- Frontmatter with only `exchange:` — verify no `globals:` blank row
- Frontmatter with only `globals:` — verify no `exchange:` blank row
- Empty frontmatter (`---` / `---` only) — verify blank rows only
- No frontmatter — verify no globals panel at all

**Institutional learning**: Test at multiple widths (60, 80, 120) per `docs/solutions/ui-bugs/preview-pane-jump-frontmatter-and-context-footer-false-positive.md`.

### Phase 3: Remove `[g]` keybinding hint

#### 3a. Remove `[g]` from `view_overlays.go`

**File**: `cmd/calcmark/tui/editor/view_overlays.go`

Remove `hint := "[g]"` and its rendering in both collapsed (line 29) and expanded (line 60) states.

#### 3b. Remove `[g]` from `components/globals.go`

**File**: `cmd/calcmark/tui/components/globals.go:81`

Remove `"(%d items, press g to expand)"` text or replace with a hint-free count display.

#### 3c. Regenerate catwalk test expectations

Catwalk golden files that render `[g]` in their expected output:
- `cmd/calcmark/tui/editor/testdata/autocomplete`
- `cmd/calcmark/tui/editor/testdata/delete_empty_line`
- `cmd/calcmark/tui/editor/testdata/selection`
- Possibly others — run `task test` to find all failures

Regenerate expectations after the code change.

## Acceptance Criteria

### Functional Requirements

- [ ] `@exchange.USD_JPY = 150` produces a parse error (unexpected character `@`)
- [ ] `@global.tax_rate = 0.25` produces a parse error
- [ ] Exchange rates defined in frontmatter work correctly for currency conversions
- [ ] Globals defined in frontmatter work correctly as variables
- [ ] Frontmatter YAML source text is never mutated by the evaluator
- [ ] Preview pane values align 1:1 with their corresponding YAML source lines
- [ ] Structural YAML lines (`---`, `exchange:`, `globals:`) show blank preview rows
- [ ] No "Globals" header appears in the preview pane
- [ ] No `[g]` hint appears anywhere in the UI
- [ ] `task test` passes with zero failures
- [ ] `task quality` passes

### Non-Functional Requirements

- [ ] No performance regression — frontmatter rendering should not add measurable latency
- [ ] Error messages for `@` character are clear and actionable

### Quality Gates

- [ ] All golden tests updated (no inline `@exchange`/`@global` syntax remains)
- [ ] Catwalk tests cover alignment for: both sections, single section, empty frontmatter, no frontmatter
- [ ] Catwalk tests regenerated for `[g]` removal
- [ ] Breaking change documented in changelog

## Dependencies & Risks

| Risk | Mitigation |
|------|-----------|
| Breaking change affects external users | Document clearly in changelog; this is a deliberate design correction |
| `parseExchangeKey`/`isValidCurrencyCode` shared by frontmatter parsing | Check references before deleting — keep if shared |
| Preview alignment regression on edge cases | Test at multiple widths (60, 80, 120) per institutional learnings |
| `globalsExpanded` state becomes orphaned | Simplify or remove if no longer needed |
| WASM target may need separate testing | Run `task test` which covers WASM if configured |

## Implementation Order

Phase 1 (grammar removal) should be done first — it's the breaking change and affects the most files. Phase 3 (`[g]` removal) is trivial and can be done alongside Phase 2. Phase 2 (alignment) depends on Phase 1 being complete since the Globals panel rendering changes.

Recommended commit order:
1. Remove inline `@exchange`/`@global` from grammar + all downstream cleanup + golden test updates
2. Remove `[g]` hint (small, independent)
3. Fix preview pane vertical alignment (depends on globals panel header removal from step 2)

## References

### Internal References

- Brainstorm: `docs/brainstorms/2026-02-24-frontmatter-stability-brainstorm.md`
- Frontmatter data model: `spec/document/frontmatter.go`
- Lexer `@` handling: `spec/lexer/lexer.go:953-982`
- Parser frontmatter assignment: `spec/parser/rdparser.go:258-306`
- AST node: `spec/ast/nodes.go:317-335`
- Evaluator write-back: `impl/document/evaluator.go:465-511`
- Preview pane rendering: `cmd/calcmark/tui/editor/view_panes.go:177`
- Globals panel overlay: `cmd/calcmark/tui/editor/view_overlays.go:12-135`
- Catwalk test guide: `cmd/calcmark/tui/editor/TESTING.md`

### Institutional Learnings

- `docs/solutions/logic-errors/exchange-rate-frontmatter-validation.md` — validate at every entry point
- `docs/solutions/logic-errors/go-maps-non-deterministic-ordering-frontmatter.md` — always use ordered key iteration
- `docs/solutions/ui-bugs/frontmatter-editing-keyboard-dispatch-fixes.md` — rawSource preservation
- `docs/solutions/ui-bugs/preview-pane-jump-frontmatter-and-context-footer-false-positive.md` — check isFrontmatter before editBuf; test at multiple widths
- `docs/solutions/code-organization/split-view-go-into-cohesive-modules.md` — rendering file cohesion
