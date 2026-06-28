# go-calcmark fixes from the calcmark-web fit-and-finish pass

Branch: `fix/cmw-fit-and-finish`. These are go-calcmark-side items surfaced
while QA-ing the calcmark-web editor. Each notes root cause, approach, value,
and status. Cross-references are to the calcmark-web repo.

---

## 1. Positionless argument-type errors → editors can't squiggle the bad arg
**Status: tracked, NOT implemented (bigger change). Filed as issue #164.**
Value: medium–high (no inline squiggle on the offending argument).

Symptom: `sx_grown = grow(100, 20%, 5)` reports a correct error, but with no
source position, so an editor can show it in the gutter yet cannot underline
the bad `20%`.

Root cause: the growth / capacity / rate builtins reject arguments with a
bare positionless error:

```go
// impl/interpreter/growth_functions.go (also capacity_functions.go, rate_eval.go)
return decZero, fmt.Errorf(
    "%s: %s must be a number, quantity, or currency — got percentage",
    funcName, paramName)
```

`spec/document.Diagnostic` already has `Line/Column/EndLine/EndColumn`, but the
interpreter error carries no AST position, so the resulting diagnostic is
line-only.

Two fix options (either suffices; the second is cleaner long-term):
1. **Interpreter** — attach the rejected argument's AST `Range` to the emitted
   diagnostic (the call node knows each argument's position). Requires
   threading the arg node/position into the error path of these builtins.
2. **Semantic checker** — encode these builtins' parameter-type contracts
   (e.g. `grow`'s increment ∈ {number, quantity, currency}) in
   `spec/semantic` so invalid args are caught at *check* time with a `Range`,
   the same path that already produces ranged diagnostics for undefined
   variables. Bonus: surfaces the error before evaluation.

Consumers already map a diagnostic `Range` → inline squiggle, so either fix
lights up the underline with zero consumer changes.

---

## 2. Frontmatter scalars coerce on parse → `version: 1.0` displays as `1`
**Status: DONE on this branch.**
Value: low (cosmetic), change: contained + backward-compatible.

Symptom: a frontmatter `version: 1.0` rendered as `1` because `Extra.Value`
holds the typed YAML value (`float64(1)`).

Fix (committed here): added `ExtraField.RawValue string` —
`spec/document/frontmatter.go` now captures the verbatim YAML scalar text from
the node tree (`parseYAMLMapping`) alongside the coerced `Value`. Scalars get
the literal (`"1.0"`, `"3.140"`); maps/lists keep `RawValue` empty. Additive
field — existing `Key`/`Value` consumers (e.g. `format/html_formatter.go`) are
untouched. Covered by `TestParseFrontmatter_ExtraRawValuePreservesScalarText`.

Consumer follow-up (calcmark-web): the frontmatter card should prefer
`RawValue` over `Value` for scalar OTHER fields when displaying. Needs a
go-calcmark release + version bump in cmw.

---

## 3. `read from` is offered as a completion but absent from the Help index
**Status: tracked, routing unclear (likely calcmark-web, not go-calcmark).**
Value: low.

`read from` is a real registered NL function (functions.go / registry.go /
nl_functions.go), so offering it as a completion is correct. It's missing from
calcmark-web's generated Help index. Determine whether go-calcmark's
reference/help data includes `read from` (if not → go-calcmark reference-data
gap) or whether cmw's `gen-help-index` drops it (→ calcmark-web fix). No code
change made until routing is confirmed.

---

## Not included here (stay in calcmark-web)
- Diagnostic-message **styling** (rendering a quoted identifier like
  `"seniors"` as `code`): a presentation concern handled in cmw's
  `diagnostic-message-parts.ts`, not a go-calcmark message-text change.
- Signature-help flakiness for paren calls: the LSP answers correctly
  (`lsp/signature.go` + tests cover `avg`/`accumulate`); the flakiness is in
  the cmw client's probe timing.
