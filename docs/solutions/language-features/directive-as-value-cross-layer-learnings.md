---
title: "Making @directives first-class values: cross-layer learnings"
category: language-features
date: 2026-03-20
severity: high
---

# Making @directives first-class values in expressions

## Context

`@scale` and `@globals.x` were originally directive references usable only in simple arithmetic. This documents what was required to make them work everywhere a number literal is valid — including unit-annotated positions (`@scale meters/second`), interpolation (`{{ @scale }}`), and autocomplete.

## Layer-by-layer findings

### 1. Lexer — No changes needed

The lexer already emits `AT_SIGN`, `IDENTIFIER`, and contextual `DOT` tokens. All downstream layers consume these tokens correctly.

### 2. Parser — Unit lookahead after DirectiveRef (`spec/parser/rdparser.go`)

**Gap found:** After parsing a `DirectiveRef` in `parsePrimary`, the parser returned immediately without checking for a following unit IDENTIFIER. The NUMBER token path had a lookahead that consumed units (`3 meters` → `QuantityLiteral`), but DirectiveRef didn't.

**Solution:** Added unit lookahead after `DirectiveRef` parsing — mirroring the NUMBER path. The key insight: `QuantityLiteral` stores `Value` as a string, so it can't hold a `DirectiveRef`. Rather than creating a new AST node, we added an optional `Expr ast.Node` field to `QuantityLiteral`. When `Expr` is non-nil, the interpreter evaluates it for the numeric value instead of parsing the `Value` string. This keeps the change minimal — only the interpreter's `evalQuantityLiteral` and the parser's DirectiveRef path need updating.

**Rate handling:** `@scale meters / second` works automatically once `@scale meters` parses as a `QuantityLiteral` — the rate parser at the `parseMultiplicative` level already wraps any `QuantityLiteral` in a `RateLiteral` when it sees `/ unit`.

### 3. AST — QuantityLiteral.Expr field + ContainsScaleRef walker (`spec/ast/nodes.go`)

**Change:** Added optional `Expr Node` field to `QuantityLiteral`. Updated `ContainsScaleRef` to add a `*QuantityLiteral` case that recurses into `Expr`. Without this, `@scale meters` would not be detected as containing a scale reference, causing double-scaling.

### 4. Semantic checker — Minimal change (`spec/semantic/checker.go`)

**Change:** Updated `checkQuantityLiteral` to recurse into `Expr` when non-nil. This ensures `@scale` inside `@scale meters` gets validated against frontmatter.

### 5. Classifier — AT_SIGN token scan (`spec/classifier/classifier.go`)

**Gap found:** The session-level classifier had no awareness of `AT_SIGN` tokens. Lines like standalone `@scale` or `@scale meters` fell through all checks and were classified as Markdown.

**Solution:** Added `containsDirective()` check (scanning for AT_SIGN in tokens) between the special keywords check and the operators check. Also added explicit `*ast.DirectiveRef` case to `allIdentifiersDefined` — it was accidentally working via the `default → true` arm, which is fragile.

**Important pattern:** The classifier is the #1 most commonly missed layer when adding language features (confirmed by `docs/solutions/logic-errors/adding-new-type-fraction-cross-layer-checklist.md`). Always check BOTH `spec/document/detector.go` AND `spec/classifier/classifier.go` — they serve different eval contexts (document vs session).

### 6. Interpreter — QuantityLiteral.Expr evaluation (`impl/interpreter/literals.go`)

**Change:** When `q.Expr` is non-nil, evaluate it via `interp.evalNode(q.Expr)`, extract the numeric value via type switch (`*types.Number`, `*types.Quantity`, etc.), and use it as the quantity value.

### 7. Interpolation — Regex and resolution (`impl/document/interpolation.go`)

**Gap found:** The interpolation regex `\{\{\s*(\w+)\s*\}\}` only matches `\w+` (word characters). `@scale` has `@` which is not a word character. `@globals.tax_rate` has both `@` and `.`.

**Solution:** Extended the regex to `\{\{\s*(@?\w+(?:\.\w+)?)\s*\}\}`. Added `resolveDirectiveForInterpolation()` to resolve `@scale` and `@globals.field` from frontmatter. Directive-resolved values skip the `transform.Apply` step to avoid double-scaling (consistent with R3: explicit @scale opts out).

### 8. Scale exemption — Already complete (mostly)

`ast.ContainsScaleRef()` recursively walks all node types. Added a `*QuantityLiteral` case to recurse into the new `Expr` field. This correctly detects `@scale` inside `@scale meters`.

### 9. TUI Autocomplete (`cmd/calcmark/tui/editor/`)

**Gap found:** Two issues:
1. `isWordRune()` excludes `@`, so typing `@sc` gives prefix `sc` not `@sc`
2. No `DirectiveSuggestionSource` exists

**Solution:**
1. Modified `getCurrentWordPrefix` to extend the prefix to include a leading `@` when found, and to handle `@globals.field` patterns by scanning through `.` and preceding word. Did NOT modify `isWordRune` — that would break `email@example` patterns.
2. Created `DirectiveSuggestionSource` using the callback pattern from `VariableSuggestionSource` (lazy frontmatter access via closure). Only offers `@scale` when frontmatter has `scale:`, and only offers `@globals.x` when globals are defined.
3. Added `"directive"` category with `"@"` tag in the suggestion popup, sorted first (`-1` priority).

**Bubble Tea gotcha:** Per `docs/solutions/logic-errors/go-closure-capturing-stale-value-type.md`, use lazy callbacks rather than capturing state at construction time.

## Review-discovered issues and fixes

### Parser unit-lookahead duplication

The NUMBER path and DirectiveRef path both had ~22 lines of identical unit-lookahead logic (check for IDENTIFIER, skip NL keywords, normalize, handle multi-word). Extracted into `tryConsumeUnit() (string, bool)` helper on the parser. Both call sites collapsed to 3-4 lines each.

**Pattern:** When adding a new expression form that can be unit-annotated, call `p.tryConsumeUnit()` after parsing the primary value. No need to duplicate the lookahead logic.

### Autocomplete `@` prefix edge cases

Three issues found during review:

1. **`@globals.` two-stage completion:** `getCurrentWordPrefix` returned `""` when cursor was right after a dot with no field chars. Fixed by adding a third branch that scans backward through `@word.` when no word chars are found at cursor position.

2. **`email@example` false positive:** The `@` extension logic included `@` when preceded by word chars (`email`). Fixed by adding `isWordRuneBefore()` guard — only extends to include `@` when preceded by non-word chars (space, operator, SOL).

3. **Bare `@`:** Returns `"@"` (length 1), which is below `minAutocompletePrefix` (2). This means typing just `@` doesn't trigger autocomplete, but `@s` or `@g` does. Acceptable tradeoff — `@` alone is too ambiguous to trigger suggestions.

### Interpolation performance: cache parsed globals

`resolveDirectiveForInterpolation` was calling `ParseGlobals()` (a full parse cycle) on every `{{ @globals.field }}` regex match. For documents with many globals references, this was O(L * parse_cost) redundant work. Fixed by pre-parsing all globals once in `interpolateTextBlocks` and passing the cache through.

### `allIdentifiersDefined` latent correctness gap

The `QuantityLiteral` node fell through to the `default → true` arm in `allIdentifiersDefined`, just like `DirectiveRef` did before it was fixed. Added an explicit case that recurses into `Expr` when non-nil. This closes the same fragile-default pattern the plan explicitly identified.

### TUI Side-by-Side preview: `GetLineResults` must use `InterpolatedSource`

**Bug:** `{{@scale}}` and `{{@globals.field}}` showed raw in Side-by-Side preview while `{{variable}}` worked. Root cause: `GetLineResults()` used `b.Source()` (raw) for TextBlock lines. The Side-by-Side mode (`PreviewRendered`) goes through `renderTextBlocks()` → `InterpolatedSource()`, so it WAS correct for Rendered mode. But the `GetLineResults` path also feeds into per-line alignment for text lines.

**Fix:** Changed TextBlock case in `GetLineResults` to use `b.InterpolatedSource()` which falls back to raw `Source()` when no interpolation exists.

**Key insight:** Interpolation in CalcMark happens as a post-evaluation transform on `TextBlock` objects via `SetInterpolatedSource()`. Any code path that reads TextBlock content for display MUST use `InterpolatedSource()`, never `Source()`. The two paths that consume TextBlock content are:
1. `renderTextBlocks()` — already correct (used by Rendered/Reading modes)
2. `GetLineResults()` — was using raw Source, now fixed

## Cookbook: "Add a new expression form to CalcMark"

Based on this implementation, here's the checklist for making any new expression form work in all positions:

1. **Lexer** — Does the token already exist? If not, add it.
2. **Parser (`parsePrimary`)** — Can it be parsed as a primary expression? Use `p.tryConsumeUnit()` if the value can be unit-annotated.
3. **AST** — Is there an existing node type, or do you need a new one? Prefer extending existing nodes with optional fields over proliferating node types.
4. **Semantic checker** — Does the walker visit your node? Add a case if needed.
5. **Classifier (`spec/classifier/classifier.go`)** — Add a token scan check. **THIS IS THE MOST COMMONLY MISSED LAYER.** Also update `allIdentifiersDefined` — don't rely on the `default → true` arm.
6. **Document detector (`spec/document/detector.go`)** — Check this too (different eval context).
7. **Interpreter** — Handle the new node/field in the evaluator.
8. **Scale exemption (`ast.ContainsScaleRef`)** — Does it walk into your new node? Add a case.
9. **Interpolation** — Does the regex match your syntax? Update if needed. Cache parsed values to avoid per-match re-parsing.
10. **Autocomplete** — Add a suggestion source or extend existing prefix matching. Guard `@` extension against `email@example` false positives with `isWordRuneBefore`.
11. **TUI preview pane** — Verify rendering works with the new expression form.
12. **LSP** — Verify completion/hover/diagnostics if applicable.
