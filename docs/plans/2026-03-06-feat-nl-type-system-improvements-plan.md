---
title: "feat: NL variable references and arithmetic type widening"
type: feat
status: completed
date: 2026-03-06
deepened: 2026-03-06
---

# NL Variable References and Arithmetic Type Widening

## Enhancement Summary

**Deepened on:** 2026-03-06
**Research agents used:** repo-research-analyst, architecture-strategist, code-simplicity-reviewer, learnings-researcher, specflow-analyzer

### Key Improvements from Research

1. **Hoist unitless Quantity normalization** to top of `evalBinaryOperation` — eliminates 4 scattered special cases with 2 lines
2. **Inline detector check** instead of creating `isNLFunctionWithKeyword` helper — 3 static cases at 1 call site don't need an abstraction
3. **Export NL function triggers from parser** — detector already imports parser, no new dependency edge
4. **NL parse functions use `parseExponent()`** not `parseExpression()` — critical for avoiding FROM ambiguity when adding IDENTIFIER support

### New Considerations Discovered

- Assignment lines (`result = compress data using gzip`) are already handled by the detector at line 274-277 — the `IDENTIFIER IDENTIFIER` rejection only affects bare NL function calls
- The `from` keyword is a reserved token (`lexer.FROM`) while `using`/`across` are contextual identifiers — the parser check for `read` is different from `compress`/`transfer`
- Existing unitless re-dispatch at lines 190-196 only fires for `Quantity op Quantity`, not `Rate * Quantity` or `Number / Quantity` — top-level normalization fixes all cases

## Overview

Four gaps in CalcMark's NL syntax and type system prevent natural expression composition in capacity planning documents. Users must work around limitations with function-call syntax or intermediate accumulations. These four changes close the gaps:

1. **NL functions accept variable references** — `compress data using gzip` works like `compress(data, gzip)`
2. **Number * Rate → Rate** — commutative: `3 * read_rate` works like `read_rate * 3`
3. **Number / Quantity → Number** — when quantity is unitless: `400M / accumulate_result`
4. **Rate * Quantity → Quantity** — scaling: `peak_read_rate * 10 KB` → bandwidth per second

## Problem Statement

Discovered while rewriting the system-sizing example (`testdata/examples/system-sizing.cm`). Each workaround makes documents harder to read:

| Issue | Workaround | Why it's bad |
|---|---|---|
| `compress data using gzip` → markdown | `compress(data, gzip)` | Inconsistent — literals use NL, vars use function syntax |
| `daily_users * 2/week` → error | `daily_users * (posts_per_user over 1 day)` | Extra line, extra variable, cognitive load |
| `daily_reads / daily_posts` → error (unitless qty) | Keep both as same type via plain arithmetic | Prevents using `accumulate`/`over` |
| `peak_read_rate * 10 KB` → error | `(peak_read_rate over 1 second) * 10 KB` | Accumulate-then-multiply is unnatural |

## Proposed Solution

Four independent changes, ordered by risk (lowest first). Phases 1, 2, and 4 share a simplification: hoist unitless Quantity normalization to the top of `evalBinaryOperation`.

## Technical Approach

### Phase 0: Hoist Unitless Quantity Normalization (prerequisite)

**Rationale:** Unitless quantities (from `accumulate`/`over` on unitless rates) should behave identically to Numbers in all arithmetic. Currently this re-dispatch is scattered at lines 190-196 for `Quantity op Quantity` only. Hoisting it to the top of `evalBinaryOperation` eliminates the need for per-type unitless checks in Phases 2 and 4.

**Files:**
- `impl/interpreter/operators.go` — add normalization at top of `evalBinaryOperation`, remove lines 190-196

**Implementation:**
```go
func evalBinaryOperation(left, right types.Type, operator string) (types.Type, error) {
    // Normalize unitless quantities to numbers before type dispatch.
    // Unitless quantities arise from accumulate/over on unitless rates.
    if q, ok := left.(*types.Quantity); ok && q.Unit == "" {
        return evalBinaryOperation(types.NewNumber(q.Value), right, operator)
    }
    if q, ok := right.(*types.Quantity); ok && q.Unit == "" {
        return evalBinaryOperation(left, types.NewNumber(q.Value), operator)
    }

    // ... existing type dispatch follows
}
```

Then **remove** the existing unitless re-dispatch at lines 190-196:
```go
// DELETE these lines:
if leftQty.Unit == "" {
    return evalBinaryOperation(types.NewNumber(leftQty.Value), right, operator)
}
if rightQty.Unit == "" {
    return evalBinaryOperation(left, types.NewNumber(rightQty.Value), operator)
}
```

**Test cases:**
- All existing tests must pass (the behavior is identical, just relocated)
- `unitless_qty(5) * 3` → `15` (unitless left, re-dispatched as Number)
- `3 * unitless_qty(5)` → `15` (unitless right, re-dispatched as Number)
- `unitless_qty(10) / unitless_qty(2)` → `5` (both re-dispatched)
- `unitless_qty(5) * 10 KB` → `50 KB` (left re-dispatched to Number, then Number * Quantity)

**Research insight:** This pattern eliminates the scattered unitless checks that the simplicity reviewer identified — 4 special cases replaced by 2 lines at the top.

### Phase 1: Number * Rate → Rate (operators.go only)

**Semantics:** `Number * Rate → Rate` — scale the rate's amount. Mirror of existing `Rate * Number → Rate` at line 159.

**Files:**
- `impl/interpreter/operators.go:69-97` — add `*types.Rate` case in Number left-operand section
- `impl/interpreter/operators_test.go` — new test cases

**Implementation:**
```go
// In the Number left-operand section (after existing Duration case):
if rightRate, ok := right.(*types.Rate); ok {
    switch operator {
    case "*":
        return types.NewRate(
            types.NewQuantity(leftNum.Value.Mul(rightRate.Amount.Value), rightRate.Amount.Unit),
            rightRate.PerUnit,
        ), nil
    }
}
```

**Research insight:** The Rate type stores `Amount` as `*Quantity` (not plain decimal) and `PerUnit` as a normalized time unit string. The `NewRate` constructor calls `NormalizeTimeUnit()` on the PerUnit. Use `types.NewQuantity` for the amount to preserve the rate's amount unit.

**Test cases:**
- `3 * 100/second` → `300/s` (basic scaling)
- `0.5 * 100 MB/s` → `50 MB/s` (fractional, preserves amount unit)
- `0 * 100/second` → `0/s` (zero)
- `3 / 100/second` → error (only multiplication supported)
- Regression: `100/second * 3` still works (existing Rate * Number path)

### Phase 2: Number / Quantity → Number (handled by Phase 0)

**With Phase 0's top-level unitless normalization, this is already handled.** When the right operand is a unitless Quantity, it gets normalized to a Number before reaching the type dispatch. `Number / Number` already works.

**For unit-bearing quantities**, `Number / Quantity` should remain an error. The existing code at lines 214-227 handles `Number op Quantity` with `*`, `+`, `-` — no change needed for `/` on unit-bearing quantities since the error is the correct behavior.

**No code changes needed beyond Phase 0.**

**Test cases (verify Phase 0 handles these):**
- `10 / unitless_qty(5)` → `2` (unitless right, re-dispatched as Number / Number)
- `10 / unitless_qty(0)` → division-by-zero error
- `10 / 5 dogs` → error "cannot divide number and quantity" (unit-bearing, not re-dispatched)
- Regression: `10 * 5 dogs` → `50 dogs` still works
- Regression: `10 + 5 dogs` → `15 dogs` still works

### Phase 3: NL functions accept variable references

**Semantics:** `compress data using gzip`, `read data from ssd`, `transfer data across regional gigabit` work when `data` is a variable that resolves to a Quantity at eval time.

Two layers must change. Assignment lines (`result = compress data using gzip`) already work because the detector checks for `IDENTIFIER ASSIGN` at line 274-277 before reaching the `IDENTIFIER IDENTIFIER` rejection.

#### 3a. Parser lookahead gates (rdparser.go)

**Files:**
- `spec/parser/rdparser.go:1091-1114` — extend NL function lookahead to accept `IDENTIFIER`
- `spec/parser/nl_functions.go:66-134` — extend NL parse functions to handle `IDENTIFIER` tokens

**Current gate pattern (all three functions):**
```go
if identName == "compress" && p.check(lexer.QUANTITY) {
    if p.peekAhead(1).Type == lexer.IDENTIFIER && strings.ToLower(string(p.peekAhead(1).Value)) == "using" {
        return p.parseNLCompressFunction()
    }
}
```

**New gate — add IDENTIFIER path with keyword lookahead:**
```go
// Existing: compress <QUANTITY> using <algo>
if identName == "compress" && p.check(lexer.QUANTITY) {
    if p.peekAhead(1).Type == lexer.IDENTIFIER && strings.ToLower(string(p.peekAhead(1).Value)) == "using" {
        return p.parseNLCompressFunction()
    }
}
// New: compress <IDENTIFIER> using <algo>
if identName == "compress" && p.check(lexer.IDENTIFIER) {
    if p.peekAhead(1).Type == lexer.IDENTIFIER && strings.ToLower(string(p.peekAhead(1).Value)) == "using" {
        return p.parseNLCompressFunction()
    }
}
```

**Critical difference for `read`:** The `from` keyword is a reserved token (`lexer.FROM`), not a contextual identifier. So the `read` lookahead checks `p.peekAhead(1).Type == lexer.FROM` — same check works for both QUANTITY and IDENTIFIER data arguments:

```go
// Existing: read <QUANTITY> from <storage>
if identName == "read" && p.check(lexer.QUANTITY) {
    if p.peekAhead(1).Type == lexer.FROM {
        return p.parseNLReadFunction()
    }
}
// New: read <IDENTIFIER> from <storage>
if identName == "read" && p.check(lexer.IDENTIFIER) {
    if p.peekAhead(1).Type == lexer.FROM {
        return p.parseNLReadFunction()
    }
}
```

**NL function parse changes** (`nl_functions.go`):

Each NL parse function currently calls `p.parseExponent()` for the data argument. `parseExponent()` handles both `QUANTITY` literals and `IDENTIFIER` references — it chains through `parseUnary()` → `parsePostfix()` → `parsePrimary()`, which handles both token types. **No changes needed to `nl_functions.go`** — `parseExponent()` already returns an `ast.Identifier` node when it encounters an `IDENTIFIER` token.

**Research insight:** The NL functions use `parseExponent()` (not `parseExpression()`) deliberately to avoid consuming `FROM` as part of a date expression. This constraint still holds for IDENTIFIER arguments since `parseExponent()` stops before binary operators.

#### 3b. Document detector (detector.go)

**Files:**
- `spec/document/detector.go:322-342` — modify `IDENTIFIER IDENTIFIER` rejection
- `spec/document/detector_test.go` — new test cases

**Architecture decision:** The architecture review recommends exporting NL function triggers from the parser. However, the simplicity review correctly notes that 3 static cases at 1 call site don't need an abstraction. **Inline the check directly:**

```go
if second.Type == lexer.IDENTIFIER {
    // NL function patterns: compress <var> using, read <var> from, transfer <var> across
    firstLower := strings.ToLower(string(first.Value))
    if len(tokens) >= 3 {
        thirdLower := strings.ToLower(string(tokens[2].Value))
        if (firstLower == "compress" && thirdLower == "using") ||
           (firstLower == "read" && (tokens[2].Type == lexer.FROM || thirdLower == "from")) ||
           (firstLower == "transfer" && thirdLower == "across") {
            return true
        }
    }
    return false
}
```

**Note on `read`:** Check both `tokens[2].Type == lexer.FROM` (reserved token) and string match as a belt-and-suspenders approach. The lexer produces a `FROM` token for the keyword `from`, so the type check is the canonical path.

**Test cases (parser):**
- `compress data using gzip` → `FunctionCall{compress, [Identifier{data}, Identifier{gzip}]}`
- `read data from ssd` → `FunctionCall{read, [Identifier{data}, Identifier{ssd}]}`
- `transfer data across regional gigabit` → `FunctionCall{transfer_time, [Identifier{data}, Identifier{regional}, Identifier{gigabit}]}`
- Regression: `compress 1 GB using gzip` still works (QUANTITY path)
- Regression: `read 5 MB from hdd` still works
- Regression: `using = 5` still parses as assignment (backward compat)
- Regression: `across = 10` still parses as assignment (backward compat)

**Test cases (detector):**
- `compress data using gzip` → classified as calculation
- `read data from ssd` → classified as calculation
- `transfer data across regional gigabit` → classified as calculation
- `Read more about this topic` → classified as prose (no `from` at position 2)
- `Compress your files` → classified as prose (no `using` follows)
- `transfer data somewhere else` → classified as prose (no `across` at position 2)

**Test cases (integration/eval — golden files):**
- `data = 1 GB` then `compress data using gzip` → ~333 MB
- `data = 5 MB` then `read data from ssd` → duration result
- `data = 500 KB` then `transfer data across regional gigabit` → duration result
- Variable undefined → clear error message
- Variable is wrong type (Number instead of Quantity) → clear error from function implementation

### Phase 4: Rate * Quantity → Quantity (and Quantity * Rate → Quantity)

**Semantics:** `Rate.Amount.Value * Quantity.Value`, preserving the Quantity's unit. Conceptually: "13,889 requests/second times 10 KB per request = 138,890 KB."

**With Phase 0:** Unitless quantities on the right are already normalized to Numbers, so `Rate * unitless_qty` naturally becomes `Rate * Number` (existing path). Only unit-bearing quantities need a new case.

**Files:**
- `impl/interpreter/operators.go:156-184` — add `*types.Quantity` case in Rate left-operand section
- `impl/interpreter/operators.go` — add `Quantity * Rate` case in Quantity left-operand section
- `impl/interpreter/operators_test.go` — new test cases

**Implementation (Rate * Quantity):**
```go
// In Rate left-operand section, after Rate / Rate:
if rightQty, ok := right.(*types.Quantity); ok {
    switch operator {
    case "*":
        result := leftRate.Amount.Value.Mul(rightQty.Value)
        return types.NewQuantity(result, rightQty.Unit), nil
    }
}
```

**Implementation (Quantity * Rate — commutative mirror):**
```go
// In Quantity left-operand section:
if rightRate, ok := right.(*types.Rate); ok {
    switch operator {
    case "*":
        result := leftQty.Value.Mul(rightRate.Amount.Value)
        return types.NewQuantity(result, leftQty.Unit), nil
    }
}
```

**Test cases:**
- `100/second * 10 KB` → `1000 KB` (basic scaling)
- `10 KB * 100/second` → `1000 KB` (commutativity)
- `0/second * 10 KB` → `0 KB` (zero rate)
- `100/second * 0 KB` → `0 KB` (zero quantity)
- `100 MB/s * 10 KB` → `1000 KB` (rate amount unit ignored, quantity unit preserved)
- `100/second / 10 KB` → error (only multiplication)
- Regression: `100/second * 3` still works (Rate * Number path)

**Edge case:** `Rate * Quantity` where both have data-size units (e.g., `100 MB/s * 10 KB`). The result is `1000 KB` — the rate's amount unit (`MB`) is effectively a scalar multiplier, and the quantity's unit (`KB`) is preserved. This matches the user's mental model for the system-sizing use case.

## Acceptance Criteria

- [x] `3 * read_rate` produces a Rate (Phase 1)
- [x] `unitless_qty / unitless_qty` produces a Number (Phase 0)
- [x] `400M / 5 dogs` produces a clear error (Phase 0 — unit-bearing not normalized)
- [x] `compress data using gzip` evaluates correctly when `data` is a variable holding a Quantity (Phase 3)
- [x] `read data from ssd` evaluates correctly with variable (Phase 3)
- [x] `transfer data across regional gigabit` evaluates correctly with variable (Phase 3)
- [x] `Read more about this topic` remains prose (Phase 3)
- [x] `peak_read_rate * 10 KB` produces a Quantity (Phase 4)
- [x] All existing tests pass (`task test`)
- [x] System-sizing example can use NL syntax with variables where appropriate
- [x] `task quality` passes

## Dependencies & Risks

**No risk (Phase 0):** Behavior-preserving refactor. Moves existing unitless normalization from scattered locations to top of function. All existing tests must pass unchanged.

**Low risk (Phases 1, 4):** Pure operator additions in `operators.go`. No parser or detector changes. Cannot break existing behavior — only new type combinations are added.

**Medium risk (Phase 3):** Changes to the document detector's line classification. False positives (prose classified as calculation) would silently change behavior. Mitigated by requiring keyword match at token position 2 (`using`/`from`/`across`), not just NL function name at position 0.

**Relevant learnings:**
- `docs/solutions/test-failures/user-config-leaks-into-tests.md` — if adding new test files that touch config, use `TestMain` + `config.Reload()` pattern for isolation
- `docs/solutions/logic-errors/go-maps-non-deterministic-ordering-frontmatter.md` — no maps are being added to frontmatter handling, so not applicable

## References

- NL syntax limitations: `memory/nl-syntax-limitations.md`
- Original NL functions plan: `docs/plans/2026-02-22-feat-plain-language-functions-plan.md`
- Parser NL triggers: `spec/parser/rdparser.go:1091-1114`
- NL parse functions: `spec/parser/nl_functions.go:66-134`
- NL parse uses `parseExponent()` not `parseExpression()`: `spec/parser/nl_functions.go:67-68`
- Document detector: `spec/document/detector.go:322-342`
- Assignment detection (pre-empts IDENT IDENT rejection): `spec/document/detector.go:274-277`
- Binary operator dispatch: `impl/interpreter/operators.go:52-231`
- Existing unitless re-dispatch: `impl/interpreter/operators.go:190-196`
- Rate type (Amount is *Quantity, PerUnit is string): `spec/types/rate.go:21-37`
- Quantity type: `spec/types/quantity.go:12-18`
- NL parse tests: `spec/parser/nl_read_compress_transfer_test.go`
- Backward compat tests for `using`/`across` as variables: `spec/parser/nl_read_compress_transfer_test.go:279-310`
- Division by zero pattern: 5 locations in `operators.go` using `IsZero()` check
- Test patterns: table-driven with subtests, type assertions for verification
