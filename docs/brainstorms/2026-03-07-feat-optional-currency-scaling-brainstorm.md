---
topic: Optional currency scaling and @directive references
date: 2026-03-07
status: decided
---

# Optional Currency Scaling and @Directive References

This brainstorm covers two related features that work together:

1. **Currency opt-in for scale** — Add `Currency` to `unit_categories` so costs scale with the recipe.
2. **`@scale` and `@globals.` directives** — Reference frontmatter values directly in expressions.

## What We're Building

### Feature 1: Currency Opt-In for Scale

Add `Currency` as a valid value in the `unit_categories` list for the `scale` frontmatter directive. When included, currency values (`$`, `€`, `£`, etc.) are multiplied by the scale factor. When omitted (the default), currency remains immune — fully backward compatible.

### Feature 2: @Directive References

Expose two frontmatter directives as read-only references in expressions:

- `@scale` — resolves to the numeric scale factor
- `@globals.name` — resolves to the named global value

This replaces the current behavior where globals are injected as plain variables. `@globals.tax_rate` replaces bare `tax_rate`. This is a **breaking change** — hard break, update all files, no deprecation period.

### How They Work Together

```text
---
scale:
  factor: 3
  unit_categories: [Mass, Volume, Currency]
convert_to: si
---

## Ingredients
flour = 2 cups              -> 6 cups (scaled), converted to ml
sugar = 1 cup               -> 3 cups (scaled), converted to ml
butter = 0.5 cups           -> 1.5 cups (scaled), converted to ml

## Cost
cost_flour = $0.50          -> $1.50 (scaled — Currency in unit_categories)
cost_sugar = $0.30          -> $0.90
total_cost = cost_flour + cost_sugar + ...
per_loaf = total_cost / @scale    -> divides by 3, stays in sync with frontmatter
```

Without `@scale`, `per_loaf = total_cost / 3` hardcodes the divisor. If the scale changes to 5, the user must update both the frontmatter and the expression.

## Why This Approach

- **Consistent**: `unit_categories` already controls scaling for Mass, Volume, etc. Currency follows the same pattern.
- **Backward compatible** (for currency): Simple `scale: 3` does NOT scale currency.
- **Breaking change** (for globals): `@globals.name` replaces plain variable injection. Readability wins — you can immediately see where a value comes from.
- **Minimal @namespace**: Only `@scale` and `@globals.` are exposed. No `@exchange`, no `@convert_to`. Tight scope.

## Key Decisions

1. **Currency excluded by default** — Simple `scale: 3` does not scale currency. `unit_categories: [Currency]` opts in.
2. **`@scale` always resolves to the factor** — Whether scale is simple (`scale: 3`) or map form (`scale: {factor: 3, unit_categories: [...]}`), `@scale` returns the numeric factor.
3. **Globals move behind `@globals.`** — Hard break. `tax_rate` as a bare variable stops working. Must use `@globals.tax_rate`. All existing .cm files with globals must update.
4. **Only two directives** — `@scale` and `@globals.name`. Nothing else exposed.
5. **Recipe example updated** — Show opt-in currency scaling with `@scale` for per-loaf division.

## Architecture

Three-layer split following existing spec→impl dependency:

### Parser (`spec/parser/`)

- Tokenize `@` followed by identifier as a new token type
- `@scale` → `DirectiveRef{Path: ["scale"]}`
- `@globals.tax_rate` → `DirectiveRef{Path: ["globals", "tax_rate"]}`
- Produce a new AST node type: `DirectiveRef`

### Semantic Analyzer (`spec/` layer)

- Validate directive references against frontmatter schema
- `@scale` valid only if `scale:` exists in frontmatter
- `@globals.tax_rate` valid only if `globals: {tax_rate: ...}` exists
- `@exchange`, `@convert_to`, `@foo` → semantic error
- Unknown globals → semantic error (e.g., `@globals.nonexistent`)

### Interpreter (`impl/interpreter/`)

- Resolve validated `DirectiveRef` to runtime values
- `@scale` → `decimal.Decimal(3)` (the factor)
- `@globals.tax_rate` → `decimal.Decimal(0.085)` (the global value)

## Implementation Notes

### Currency scaling in transform.go

```go
case *types.Currency:
    if scale != nil && categoryMatch("Currency", scale.UnitCategories) {
        return types.NewCurrency(v.Value.Mul(scale.Factor), v.Symbol)
    }
    return result
```

### Files to change

**Currency opt-in:**
- `spec/transform/transform.go` — Add `*types.Currency` case
- `spec/document/frontmatter.go` — Accept "Currency" in unit_categories
- `spec/units/canonical.go` — Add "Currency" to valid categories list

**@Directives:**
- `spec/parser/lexer.go` — New token type for `@`
- `spec/parser/rdparser.go` — Parse `DirectiveRef` nodes
- `spec/ast/` — New `DirectiveRef` AST node
- `spec/document/` — Semantic validation of directive refs against frontmatter
- `impl/interpreter/` — Resolve DirectiveRef to values
- `impl/interpreter/` — Remove plain globals variable injection

**Docs and examples:**
- `testdata/examples/recipe-scaling.cm` — Update with currency scaling + `@scale`
- `site/content/docs/examples/recipe-scaling.md` — Update documentation
- All `.cm` files using globals — Update to `@globals.name`
- `site/content/docs/language-reference.md` — Document @directive syntax
- `site/content/docs/user-guide.md` — Document @directive usage
- `AGENTS.md`, `site/content/docs/agent-integration.md` — Update

## Open Questions

None — all questions resolved during brainstorm.
