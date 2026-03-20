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

## Cookbook: "Add a new expression form to CalcMark"

Based on this implementation, here's the checklist for making any new expression form work in all positions:

1. **Lexer** — Does the token already exist? If not, add it.
2. **Parser (`parsePrimary`)** — Can it be parsed as a primary expression? Add unit lookahead if the value can be unit-annotated.
3. **AST** — Is there an existing node type, or do you need a new one? Prefer extending existing nodes with optional fields over proliferating node types.
4. **Semantic checker** — Does the walker visit your node? Add a case if needed.
5. **Classifier (`spec/classifier/classifier.go`)** — Add a token scan check. **THIS IS THE MOST COMMONLY MISSED LAYER.**
6. **Document detector (`spec/document/detector.go`)** — Check this too (different eval context).
7. **Interpreter** — Handle the new node/field in the evaluator.
8. **Scale exemption (`ast.ContainsScaleRef`)** — Does it walk into your new node? Add a case.
9. **Interpolation** — Does the regex match your syntax? Update if needed.
10. **Autocomplete** — Add a suggestion source or extend existing prefix matching.
11. **TUI preview pane** — Verify rendering works with the new expression form.
12. **LSP** — Verify completion/hover/diagnostics if applicable.
