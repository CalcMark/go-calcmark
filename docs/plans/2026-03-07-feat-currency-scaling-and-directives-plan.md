---
title: "feat: Optional currency scaling and @directive references"
type: feat
status: active
date: 2026-03-07
---

# Optional Currency Scaling and @Directive References

## Overview

Two related features that work together:

1. **Currency opt-in for scale** — Add `Currency` as a valid `unit_categories` value so currency values scale when explicitly listed.
2. **@directive references** — New syntax `@scale` and `@globals.name` that reference frontmatter values in expressions. Globals move behind `@globals.` (breaking change).

Together these let a recipe-scaling document scale costs automatically and reference the scale factor in expressions like `per_loaf = total_cost / @scale`.

## Problem Statement

Currency values are immune to the `scale` frontmatter directive. In the recipe-scaling example, the user must manually divide `total_cost / 2` and hardcode the divisor. If the scale changes from 2 to 5, the user must update both the frontmatter and the expression. There is no way to reference the scale factor in an expression.

Globals are currently injected as plain variables (`tax_rate` works directly), which makes it impossible to tell whether a variable was defined in the document or comes from frontmatter. Moving globals behind `@globals.` makes the source explicit.

## Proposed Solution

### Currency Scaling

```yaml
---
scale:
  factor: 3
  unit_categories: [Mass, Volume, Currency]
convert_to: si
---
```

- Add `"Currency"` to valid `unit_categories` values
- Add `*types.Currency` case in `spec/transform/transform.go` `Apply()` function
- Default behavior unchanged: simple `scale: 3` does NOT scale currency

### @Directive References

```text
per_loaf = total_cost / @scale
tax = income * @globals.tax_rate
```

- `@scale` resolves to the numeric scale factor (always `*types.Number`)
- `@globals.name` resolves to the typed value of the named global
- Only `@scale` and `@globals` exposed — `@exchange`, `@convert_to` are errors
- **Breaking change**: bare globals (`tax_rate`) stop working; must use `@globals.tax_rate`

## Technical Approach

### Architecture

Three-layer split following existing spec→impl dependency:

```
spec/lexer/         → Tokenize @ as AT_SIGN, handle DOT in @globals.name context
spec/ast/           → New DirectiveRef node type
spec/parser/        → Parse @scale and @globals.name in parsePrimary()
spec/semantic/      → Validate DirectiveRef against frontmatter (needs frontmatter access)
spec/transform/     → Add *types.Currency case in Apply()
spec/units/         → Add CategoryCurrency constant and include in Categories()
spec/document/      → Remove plain globals env.Set() injection
impl/interpreter/   → Resolve DirectiveRef to runtime values
impl/document/      → Pass frontmatter to interpreter
```

### Implementation Phases

#### Phase 0: Currency Opt-In for Scale

Smallest, self-contained change. No parser work. Can ship independently.

- [x] Add `CategoryCurrency = "Currency"` constant in `spec/units/categories.go`
- [x] Add `seen[CategoryCurrency] = true` in `Categories()` function
- [x] Add `*types.Currency` case in `spec/transform/transform.go` `Apply()`:
  ```go
  case *types.Currency:
      if scale != nil && categoryMatches("Currency", scale.UnitCategories) {
          scaled := v.Value.Mul(scale.Factor)
          return types.NewCurrency(scaled, v.Symbol)
      }
      return result
  ```
- [x] Add test `TestApply_CurrencyScaledWithCategory` in `spec/transform/transform_test.go`
- [x] Verify existing `TestApply_CurrencyImmune` still passes (no unit_categories → immune)
- [x] Add golden test `testdata/eval/success/features/scale_currency.cm`
- [x] Update `spec/features/registry.go` scale description to mention Currency opt-in
- [x] Run `task test` — all tests pass

**Success criteria**: `scale: {factor: 3, unit_categories: [Currency]}` triples currency values. Simple `scale: 3` does not.

#### Phase 1: Lexer — @ Token and DOT-in-@-context

- [x] Add `AT_SIGN` token type in `spec/lexer/token.go`
- [x] Handle `@` in `Tokenize()` in `spec/lexer/lexer.go`:
  - Emit `AT_SIGN` token
  - Read following identifier characters as a single `IDENTIFIER` token (e.g., `scale`, `globals`)
  - If next char after identifier is `.` AND the identifier is `globals`, emit `DOT` token then read next identifier
  - Otherwise, just emit `AT_SIGN` + `IDENTIFIER`
- [x] Ensure `.` in `3.14` is NOT affected — DOT only emitted in `@globals.` context
- [x] Add lexer tests:
  - `@scale` → `[AT_SIGN, IDENTIFIER("scale")]`
  - `@globals.tax_rate` → `[AT_SIGN, IDENTIFIER("globals"), DOT, IDENTIFIER("tax_rate")]`
  - `@` alone → lexer error
  - `@123` → lexer error
  - `@globals.a.b` → `[AT_SIGN, IDENTIFIER("globals"), DOT, IDENTIFIER("a"), DOT, IDENTIFIER("b")]` (parser will reject the second dot)
  - `3.14` → still `NUMBER(3.14)` (decimal parsing unchanged)
- [x] Run `task test` — all tests pass

**Success criteria**: Lexer correctly tokenizes `@scale` and `@globals.name` without breaking decimal number parsing.

#### Phase 2: AST Node and Parser

- [ ] Add `DirectiveRef` AST node in `spec/ast/nodes.go`:
  ```go
  type DirectiveRef struct {
      Directive string   // "scale" or "globals"
      Field     string   // "" for @scale, "tax_rate" for @globals.tax_rate
      Range     *Range
  }
  ```
  Implement `Node` interface (`String()`, `GetRange()`)
- [ ] Add `DirectiveRef` parsing in `parsePrimary()` in `spec/parser/rdparser.go`:
  - When current token is `AT_SIGN`:
    - Consume `AT_SIGN`
    - Expect `IDENTIFIER` — the directive name
    - If directive is `"globals"` and next is `DOT`, consume `DOT` + `IDENTIFIER` for field
    - If directive is `"globals"` and no DOT, parser error: `@globals requires a field name (e.g., @globals.tax_rate)`
    - Return `&ast.DirectiveRef{Directive: name, Field: field}`
  - Reject second DOT: `@globals.a.b` → parser error (only one level of dot-access)
- [ ] Add parser tests:
  - `x = @scale` parses to assignment with DirectiveRef
  - `x = @globals.tax_rate` parses to assignment with DirectiveRef
  - `x = @globals` → parser error
  - `x = @globals.a.b` → parser error (nested dots)
  - `x = @exchange` → parses (semantic checker will reject)
  - `x = 1 + @scale * 2` → correct precedence
- [ ] Run `task test` — all tests pass

**Success criteria**: Parser produces `DirectiveRef` AST nodes for valid `@` references and rejects malformed ones.

#### Phase 3: Semantic Validation

- [ ] Add frontmatter awareness to `Checker` in `spec/semantic/checker.go`:
  - Add `Frontmatter *document.Frontmatter` field to `Checker` struct
  - Update `NewChecker()` to accept optional frontmatter
  - Update callers of `NewChecker()` to pass frontmatter when available
- [ ] Add `case *ast.DirectiveRef:` in `checkNode()`:
  - `@scale`: valid only if frontmatter has `scale:` config
  - `@globals.name`: valid only if frontmatter has `globals:` with that key
  - `@exchange`, `@convert_to`, etc.: error — `'@exchange' is not a supported directive; use @scale or @globals.name`
  - Unknown: error — `unknown directive '@foo'; valid directives are @scale and @globals.name`
- [ ] Error messages should list available values:
  - `@globals.nonexistent` → `undefined global 'nonexistent'; defined globals: budget, tax_rate`
  - `@scale` without scale → `@scale requires 'scale:' in frontmatter`
- [ ] Add semantic checker tests for all error cases
- [ ] Run `task test` — all tests pass

**Success criteria**: Semantic checker catches invalid directive references with helpful error messages before the interpreter runs.

#### Phase 4: Interpreter Resolution

- [ ] Add `case *ast.DirectiveRef:` in `evalNode()` in `impl/interpreter/interpreter.go`
- [ ] Add frontmatter access to `Interpreter`:
  - Option A: Add `frontmatter *document.Frontmatter` field, set during construction
  - Option B: Pre-resolve directives into the environment with `@` prefix during `ApplyFrontmatter()`
  - **Preferred: Option A** — cleaner separation, no magic variable names
- [ ] `evalDirectiveRef()` logic:
  - `@scale`: return `types.NewNumber(frontmatter.Scale.Factor)` (or error if no scale)
  - `@globals.name`: look up in parsed globals map, return the typed value
- [ ] Update `impl/document/evaluator.go` to pass frontmatter when constructing interpreter
- [ ] Add interpreter tests:
  - `@scale` resolves to the numeric factor
  - `@scale` with map form resolves to factor
  - `@globals.tax_rate` resolves to the typed value
  - `@globals.budget` where budget is currency resolves correctly
  - Arithmetic with directives: `cost * @scale`, `income * (1 - @globals.tax_rate)`
- [ ] Run `task test` — all tests pass

**Success criteria**: `@scale` and `@globals.name` resolve to correct typed values in expressions.

#### Phase 5: Breaking Change — Remove Plain Globals Injection

- [ ] Remove `env.Set(name, value)` loop in `spec/document/document.go` `ApplyFrontmatter()` (line ~448)
- [ ] Remove parallel injection in `impl/document/evaluator.go` `eval.go` if present
- [ ] Update ALL files that use globals as bare variables:
  - [ ] `testdata/examples/datacenter-cost.cm` — bare globals → `@globals.name`
  - [ ] `testdata/eval/success/features/currency_conversion.cm` — `budget` → `@globals.budget`
  - [ ] `testdata/spec/valid/features/exchange_rates.cm` — `budget` → `@globals.budget`
  - [ ] `cmd/calcmark/tui/editor/default_frontmatter.cm` — verify usage
  - [ ] Any other `.cm` files found by: `grep -r "globals:" testdata/ --include="*.cm" -l`
- [ ] Update golden files for changed `.cm` files
- [ ] Update site documentation:
  - [ ] `site/content/docs/examples/datacenter-cost.md` — update code blocks and prose
  - [ ] `site/content/docs/user-guide.md` — globals section, remove bare variable examples
  - [ ] `site/content/docs/language-reference.md` — add @directive section, update globals
- [ ] Run `task test` — all tests pass
- [ ] Run `task site:build` — site builds cleanly

**Success criteria**: Bare globals produce undefined variable errors. Only `@globals.name` works.

#### Phase 6: Detector and Line Classification

- [ ] Update `looksLikeCalculation()` in `spec/document/detector.go` to recognize `@` as a valid calculation token
- [ ] Lines like `per_loaf = total_cost / @scale` must classify as CALCULATION, not MARKDOWN
- [ ] Lines starting with `@` (e.g., standalone `@scale`) should also classify correctly
- [ ] Add detector tests for `@`-containing lines
- [ ] Run `task test` — all tests pass

**Success criteria**: Lines containing `@` directives are correctly classified as calculations.

#### Phase 7: TUI and Documentation Polish

- [ ] Update TUI globals panel (`cmd/calcmark/tui/components/globals.go`) — change `@global.` to `@globals.` (plural)
- [ ] Consider adding `@scale` display to the globals panel
- [ ] Update recipe-scaling example:
  - [ ] `testdata/examples/recipe-scaling.cm` — add Currency to unit_categories, use `@scale` for per-loaf
  - [ ] `site/content/docs/examples/recipe-scaling.md` — update documentation
- [ ] Update `site/content/docs/language-reference.md`:
  - [ ] Add @Directive References section with syntax, validation rules, type table
  - [ ] Update Frontmatter section to document Currency in unit_categories
- [ ] Update `site/content/docs/user-guide.md`:
  - [ ] Add @Directive usage examples
  - [ ] Update globals section
- [ ] Update `spec/features/registry.go` — add @directive feature entries
- [ ] Update `site/content/docs/agent-integration.md` — mention @directives in pipe interface
- [ ] Update `AGENTS.md` if needed
- [ ] Run `task site:build` — clean build
- [ ] Run `task quality` — passes

**Success criteria**: All documentation reflects the new features. Site builds cleanly.

## Acceptance Criteria

### Functional Requirements

- [ ] `scale: {factor: 3, unit_categories: [Currency]}` multiplies currency values by 3
- [ ] Simple `scale: 3` does NOT scale currency (backward compatible)
- [ ] `@scale` resolves to the numeric scale factor in expressions
- [ ] `@scale` with map form (`scale: {factor: 4, ...}`) resolves to 4
- [ ] `@globals.tax_rate` resolves to the typed global value
- [ ] Bare globals (`tax_rate`) produce undefined variable errors
- [ ] `@exchange`, `@convert_to`, `@foo` produce clear semantic errors
- [ ] `@globals` without `.name` produces a parser error
- [ ] `@globals.a.b` (nested dots) produces a parser error
- [ ] Lines with `@` directives classify as CALCULATION
- [ ] `@scale` and `@globals.name` work in all arithmetic contexts (precedence, NL syntax, rate widening)

### Non-Functional Requirements

- [ ] No performance regression — directive resolution is O(1) map lookup
- [ ] Error messages list valid directives/globals for discoverability
- [ ] DOT token emission does not affect decimal number parsing (`3.14`)

### Quality Gates

- [ ] `task test` passes with zero failures
- [ ] `task quality` passes
- [ ] `task site:build` builds cleanly
- [ ] All golden tests updated
- [ ] Rate widening tests still pass with @directive values

## Dependencies & Risks

### Dependencies

- Phase 0 (currency scaling) is independent — can ship without @directives
- Phases 1-4 (lexer → parser → semantic → interpreter) are sequential
- Phase 5 (breaking change) depends on Phase 4
- Phases 6-7 (detector, docs) can run after Phase 4

### Risks

| Risk | Mitigation |
|------|-----------|
| DOT token conflicts with decimal parsing | Lexer only emits DOT in `@globals.` context; extensive tests |
| Breaking change causes user pain | Hard break is deliberate; grep all `.cm` files; update atomically |
| Semantic checker lacks frontmatter context | Add optional Frontmatter field to Checker struct |
| Currency scaling interacts with currency conversion | Scale applies post-evaluation, before display; conversion is separate |
| Double-scaling when both currency and quantity operands match | Scale applies to final result only, not operands |

## References

### Internal References

- Brainstorm: `docs/brainstorms/2026-03-07-feat-optional-currency-scaling-brainstorm.md`
- Rate widening solution: `docs/solutions/feature-gaps/rate-type-arithmetic-widening.md`
- Frontmatter ordering: `docs/solutions/logic-errors/go-maps-non-deterministic-ordering-frontmatter.md`
- Exchange validation: `docs/solutions/logic-errors/exchange-rate-frontmatter-validation.md`

### Key Files

| File | Purpose |
|------|---------|
| `spec/transform/transform.go:24-38` | Transform Apply() — add Currency case |
| `spec/units/categories.go:9-22` | Categories() — add Currency |
| `spec/lexer/token.go:9-119` | Token types — add AT_SIGN |
| `spec/lexer/lexer.go:798-1034` | Tokenize() — handle @ |
| `spec/ast/nodes.go` | AST nodes — add DirectiveRef |
| `spec/parser/rdparser.go:814-889` | parsePrimary() — add @ case |
| `spec/semantic/checker.go:101-173` | Checker — add frontmatter + DirectiveRef validation |
| `impl/interpreter/interpreter.go:49-98` | evalNode() — add DirectiveRef case |
| `spec/document/document.go:425-454` | ApplyFrontmatter() — remove globals env.Set() |
| `spec/document/detector.go:260-354` | looksLikeCalculation() — recognize @ |
| `cmd/calcmark/tui/components/globals.go:103-104` | TUI globals panel — fix prefix |
