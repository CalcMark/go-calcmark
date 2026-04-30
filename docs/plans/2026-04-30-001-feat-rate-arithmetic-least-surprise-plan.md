---
title: "feat: Rate arithmetic — same-dimension cancellation across all units"
type: feat
status: active
date: 2026-04-30
origin: docs/brainstorms/2026-04-30-rate-arithmetic-least-surprise-requirements.md
---

# Rate Arithmetic — Same-Dimension Cancellation Across All Units

## Overview

Generalize PR #148's time-only rate-cancellation engine to work across every unit category CalcMark already supports. The user-facing principle (from the origin brainstorm): **"CalcMark works when a normal human reading the expression aloud would expect it to work — and refuses with a clear message when they wouldn't."**

When this lands, `$100/hour × 3 weeks`, `100 cakes/box × 5 boxes`, `60 mph × 2 hours`, and `$100/hour × 40 hours/week × 3 weeks` all produce the obviously-correct answer; `$100/hour × 5 kg` errors with a message naming both operands and explaining the non-cancellation; the long-standing too-permissive `Rate × Quantity → Quantity` rule is replaced by an explicit cancellation check.

Ships as `v2.1.0` post `v2.0.0` stable.

---

## Problem Frame

The Rate type is half-built. Multiplying a rate by a duration or a cancellable quantity errors today (`cannot multiply rate (...) and duration (...)`) or silently coerces (the line-367 `Rate × Quantity` rule that ignores `PerUnit`). Users typing the most natural-language thing they can think of hit a wall.

The shipped `v2.0.0-rc.1` inherits this gap from v1; the rate type doesn't behave the way humans expect across multiple domains (wages, recipes, capacity planning, physical math). PR #148 implemented a time-only first slice and was paused for design. The brainstorm settled the design (Approach B: same-dimension cancellation across all categories, refuse on dimensional mismatch). This plan turns that into U-IDs.

(see origin: `docs/brainstorms/2026-04-30-rate-arithmetic-least-surprise-requirements.md`)

---

## Requirements Trace

- R1. Rate × Duration must compute when the rate's denominator is a time unit (with auto-conversion).
- R2. Rate × Quantity must compute when the rate's denominator matches the quantity's unit (or converts within the same category).
- R3. Rate literals accept a Duration numerator (`40 hours / week`).
- R4. Rate × Rate cancels matching units across the two factors (chained reduction).
- R5. Rate × Number stays as scaling — no implicit duration coercion (existing behaviour, locked).
- R6. Inconsistent quantity types must error with a clear message naming both operands and explaining the non-cancellation.
- R7. Result-type reconstruction preserves currency identity (`100 USD/hour × 3 weeks → 50,400 USD`, not normalised away).

**Origin acceptance examples:**
- AE1 (covers R1, R7) — `cost = $100/hour; work = 3 weeks; fee = cost * work` → `$50,400`.
- AE2 (covers R1, R3, R4) — chained `rate * density * weeks` → `$12,000`; intermediate `rate * density` → `$4,000/week`.
- AE3 (covers R2) — `100 cakes/box × 5 boxes` → `500 cakes`.
- AE4 (covers R2) — `60 mph × 2 hours` → `120 miles` (existing speed↔rate bridge co-exists with the new engine; see Decision 5).
- AE5 (covers R6) — `$100/hour × 5 kg` errors, message names both operands + units.
- AE6 (covers R5) — `$100/hour × 2` → `$200/hour` (Rate, scaled), NOT `$200`.

---

## Scope Boundaries

**In scope:**
- Same-dimension cancellation across every category in `spec/units/conversion.go` plus time units (which live in `spec/types/duration.go`).
- Custom units (e.g., `cakes`, `boxes`) cancel by exact-string equality; no conversion.
- Duration-numerator rates (R3 — already in PR #148, kept).
- Two-rate cancellation in left-associative chains (R4 — generalised from PR #148's time-only).
- Currency-symbol-or-ISO-code preservation in reconstructed Currency results (R7).
- Refusal-with-message for dimensional mismatches (R6) — replaces the existing line-367 silent-coercion rule.
- Pre-existing widening at `operators.go:107-122` (Rate-on-RIGHT) extended to refuse on dimensional mismatch instead of silently dropping the rate's denominator.

**Deferred to Follow-Up Work:**
- Operator precedence for `3 weeks × $/hour` (origin scope, deferred). Rate-on-left is canonical.
- Inverse division `work / rate = duration` (origin scope, deferred).
- Rate addition with different denominators `$100/hour + $50/day` (origin scope, deferred).
- `Rate / Rate` cross-PerUnit cancellation with conversion (`100/hour / 50/minute → 0.0333`). The existing same-PerUnit rule at `operators.go:362` uses string-equality; extending to category-equality is a natural follow-up but out of scope here (see Decision 8). Pre-existing `docs/solutions/feature-gaps/rate-type-arithmetic-widening.md` flagged the gap.

**Outside this product's identity:**
- Full dimensional algebra (no `m²/s²`, no `kg·m/s² → newton` reduction). CalcMark is a calculation notebook, not a CAS.
- Implicit unit inference (bare numbers stay bare numbers).
- Currency × Currency arithmetic (refused with R6-style message).

---

## Context & Research

### Relevant Code and Patterns

- `spec/types/rate.go` — `Rate{Amount *Quantity, PerUnit string}`. Note: `Amount` is always `*Quantity` (currency identity is normalized to symbol on construction). `NormalizeTimeUnit` (line 209) canonicalizes time-unit aliases. `IsTimeUnit(unit)` and `TimeUnitToSeconds(unit)` are public helpers.
- `spec/types/currency.go` — `SymbolToCode` (line 20), `CodeToSymbol` (line 77), `IsCurrencyCode` (line 85). `NewCurrency(value, symbolOrCode)` already canonicalizes both directions — pass either `"$"` or `"USD"` and it computes the missing field.
- `spec/types/duration.go` — `Duration.Convert(targetUnit)` (line 100). `IsValidDurationUnit(unit)` (line 60). Time units live here, NOT in `spec/units/conversionRegistry`.
- `spec/units/conversion.go` — `Convert(value, fromUnit, toUnit) (decimal.Decimal, error)` (line 34). Already enforces "same category required" with the diagnostic `"different categories: %s vs %s"` (lines 49-52). `CategoryForUnit(unitName) string` (line 68) returns canonical category and falls back to `CategoryCustom` for unknown.
- `impl/interpreter/operators.go:339-372` — Rate-on-the-LEFT dispatch (canonical fix site). Line 367-371 is the silent-coercion `Rate × Quantity` rule this plan replaces.
- `impl/interpreter/operators.go:107-122` — Rate-on-the-RIGHT widening. Currently strips `.Amount` and recurses; `Number * Rate` works because of this. Plan extends it to refuse on dimensional mismatch (rather than silently drop the denominator).
- `impl/interpreter/rate_eval.go:33-36` and `spec/document/literal_eval.go:212-216` — twin rate-construction sites. Both reduce `*Currency` to `Quantity{Value, Unit: v.Symbol}` today. Both get the `case *types.Duration:` arm from PR #148.
- PR #148 (`origin/feat/rate-duration-arithmetic`, commit `25019d9`) — time-only first slice. 5 helpers (`rateTimesDuration`, `tryRateRateCancellation`, `rateNumeratorAsResult`, `isTimeUnit`, `convertTimeValue`, `isCurrencyCode`) live in `impl/interpreter/rate_arithmetic.go`. The 7 value tests in `impl/interpreter/rate_duration_arithmetic_test.go` survive the generalisation rebase unchanged.
- Test idiom — inline `evalSingleResult(t, src) → types.Type` helper from PR #148's test file. Use this for value assertions; use the table-driven pattern from `rate_eval_test.go` for shape assertions on Rate construction.

### Institutional Learnings

- `docs/solutions/feature-gaps/rate-type-arithmetic-widening.md` — establishes existing dispatch table and the asymmetric-widening rule. Flags two specific gotchas this plan inherits the opportunity to fix: (a) `Rate / Rate` with mismatched PerUnits silently falls to a generic error, and (b) currency rates currently widen to `Quantity` not `Currency` (R7 fixes this for cancellation results).
- `docs/solutions/logic-errors/adding-new-type-fraction-cross-layer-checklist.md` — operator-dispatch trap pattern: when a type meets Currency/Duration/Rate, normalize and fall through; do not add N×M combinations. The same discipline applies here — the cancellation engine sits at the existing dispatch site, not as a parallel one.
- `docs/solutions/ui-bugs/display-formatter-overrides-explicit-unit-conversion.md` — `IsExplicit` flag pattern for arithmetic results. **Decision deferred** to implementation: cancellation results will inherit the existing default (no `IsExplicit`), matching PR #148's choice. Revisit if display auto-scaling produces surprising output during execution.
- `docs/solutions/logic-errors/currency-synonym-symbol-vs-code-comparison.md` — `IsSameCurrency()` over `.Symbol` comparison. Not directly invoked by this plan, but result-construction's reverse-lookup must use `IsCurrencyCode` (which compares Code) rather than `Symbol == "$"` style checks.

### External References

None — local patterns are sufficient. The `units` package converter API is uniform; the `Duration.Convert` API is well-understood; the rate dispatch site is well-mapped. External research would not add value.

---

## Key Technical Decisions

1. **Cancellation predicate is "shared category".** Two units cancel when (a) they're identical strings, OR (b) `categoryOf(left) == categoryOf(right) && categoryOf is convertible`. Identical-string case handles Custom units (`box == box` regardless of plurality, assuming the lexer normalizes); category-match handles cross-unit conversion within a category.
   *Rationale:* Honors the brainstorm's principle ("would a human say this?") for every domain CalcMark already supports. Custom units cancel only when literally the same string because there's no converter for them.

2. **Time-units live in their own bucket; everything else uses `units.Convert`.** A new `categoryOf(string) string` helper checks `IsValidDurationUnit` first (returns `"time"`), falls through to `units.CategoryForUnit` (returns `Length / Mass / DataSize / Custom / etc.`). A new `convertWithinCategory(value, fromUnit, toUnit) (decimal.Decimal, error)` helper dispatches `time → Duration.Convert`, everything else → `units.Convert`.
   *Rationale:* The two converter packages aren't unified today and unifying them is a bigger change with no other motivation. A 30-line shim is cheaper.

3. **Currency identity preservation via reverse-lookup at result time** (user choice from planning).
   At result construction (`rateNumeratorAsResult`), if the surviving Amount.Unit string is in `SymbolToCode` keys OR satisfies `IsCurrencyCode`, return `NewCurrency(value, unit)` (which canonicalizes Symbol vs Code automatically). Otherwise return Quantity. No data-shape change to `Rate`.
   *Rationale:* Simpler than threading `Currency.Code` through `Rate.Amount`; matches PR #148's existing `rateNumeratorAsResult` shape; avoids touching every test that builds rates manually.

4. **The line-367 silent-coercion rule is replaced, not preserved.** PR #148 left it untouched as a soak-window safety net; this plan removes it. Any `Rate × Quantity` that doesn't cancel via the new engine errors with R6's contract.
   *Rationale:* Brainstorm Success Criteria 3 explicitly retires this rule. Soak window for v2.1 is the right place to land the cleanup. (PR #148's parked draft remains discoverable but is replaced, not merged-as-is.)

5. **Speed↔rate bridge needs explicit invocation in `evalBinaryOperation` for AE4.** Doc review caught the original assumption: `coerceSpeedToRate` exists in `impl/interpreter/speed_bridge.go` but is only invoked by `unit_conversion_eval.go:41` (the `as`-conversion path), NOT by binary-op dispatch. So `60 mph × 2 hours` (AE4) does NOT work today — `Quantity{60, "mph"} × Duration{2, "hour"}` falls through with no handler.
   *Plan resolution:* U6 adds a left-side Quantity-coercion branch that recognises Speed-shaped units, calls `coerceSpeedToRate`, and re-dispatches as `Rate × Duration` through the engine from U2. Mirrors the existing right-side rate-widening at `operators.go:107-122` (which the plan already extends in U4).
   *Rationale:* Reuses existing speed-bridge code rather than building parallel logic. Keeps AE4 in scope as the brainstorm intended without introducing general `Quantity × Quantity` dimensional analysis (which is explicitly out-of-identity).

6. **Refusal-message contract is substring-match, not verbatim** (resolves origin G5).
   Tests assert that the error contains:
   - The left rate's display string (or its component units)
   - The right operand's display string (or its component units)
   - At minimum one of: a category name (`mass`, `time`), the offending unit name (`kg`, `hour`), or the phrase `cancel`.
   Tests do NOT assert the exact verbatim format of the message.
   *Rationale:* Locks the user-facing contract without freezing wording. Implementer can refine phrasing without breaking tests.

7. **PR #148 disposition: rebase-and-extend, not close-and-replace.** The 5 helpers are correctly named and shaped. Generalisation is mechanical: `isTimeUnit` becomes `categoryOf` equality; `convertTimeValue` becomes `convertWithinCategory`. The 7 value tests survive unchanged. New tests added for AE3/AE4/AE5/AE6.

8. **`Rate / Rate` cross-PerUnit cancellation is deferred.** The existing rule at `operators.go:362` (`Rate / Rate` same-PerUnit → Number ratio) uses string-equality on `PerUnit`. Extending it to category-equality with conversion (`100/hour / 50/minute → 0.0333` after converting minute → hour) is a natural extension of the principle but is NOT in scope for this plan. The brainstorm's R1-R7 cover multiplication forms; division-with-cancellation is a follow-up. Documented in Scope Boundaries as `Deferred to Follow-Up Work`.
   *Rationale:* Keeps the plan tight. The `learnings/rate-type-arithmetic-widening.md` doc flagged this as a known gotcha but the brainstorm didn't pick it up; this plan honours the brainstorm's scope.

---

## Open Questions

### Resolved During Planning

- **G1 (Duration-numerator rate × non-time operand).** Resolved: Duration-numerator rates behave identically to Quantity-numerator rates for type reconstruction. `40 hours/week × 5` (R5 scaling) → `200 hours/week` (Rate). Same dispatch path; no special case needed because the Quantity stored at construction time already encodes the time-unit numerator as a unit string.
- **G2 (currency code preservation through chains).** Resolved by Decision 3: the reverse-lookup happens at every result-construction call site, so a chained `100 USD/hour × 40 hours/week × 3 weeks` flows USD-symboled Quantities through every intermediate, then reconstructs as Currency at the end via `NewCurrency(value, "USD")`.
- **G3 (unknown-symbol currency-numerator detection).** Resolved by Decision 3: only units in `SymbolToCode` keys or passing `IsCurrencyCode` reconstruct as Currency; everything else falls through to Quantity. A custom currency-shaped unit not registered in SymbolToCode produces a Quantity result, not a runtime panic.
- **G5 (error-message format contract).** Resolved by Decision 6: substring-match contract, not verbatim.
- **G9 (speed-bridge coexistence).** Resolved by Decision 5: existing speed↔rate bridge stays; cancellation engine sees the post-coerced Rate × Duration form. Test asserts AE4 still works.

### Deferred to Implementation

- **G4 (zero/negative/fractional duration).** Decision will fall out of `decimal.Decimal` arithmetic naturally — zero gives zero, negative gives negative, fractional works. If the implementer hits a surprising case, surface it in PR review rather than predicting it here.
- **G6 (refusal-message readability for nested operands).** Use `formatTypeForError` for operand display — already handles Rate as `rate (...)`. If the message becomes unreadable in chained-failure cases, adjust during implementation; not a planning-time decision.
- **G7 (empty PerUnit on a Rate).** Refuse with a "rate has no denominator unit" message. Concretely: `categoryOf("")` returns `""`; the cancellation predicate fails; R6 refusal fires. Trust the dispatch.
- **G8 (unknown time-unit like `fortnight`).** `Duration.Convert` errors with `"invalid duration unit"`; the error wraps cleanly in the R6 message. No special handling needed.
- **G10 (currency × currency message).** Same R6 contract — name both operands, explain non-cancellation, substring-match test. Implementer chooses wording only; this is NOT a structurally distinct error path. The brainstorm scope says "Currency × Currency arithmetic — refuse always" (Outside this product's identity); R6's contract applies.
- **G11 (rate-on-right deferred form).** No new tests required — the existing parse-precedence behavior continues. Document the workaround in the user guide as part of U5 if the implementer notices a natural place.
- **`IsExplicit` flag policy on cancellation results.** Inherit PR #148's choice (no flag). Revisit if display auto-scaling produces surprising output during execution.

---

## Implementation Units

- U1. **Generic category and conversion helpers**

**Goal:** Replace PR #148's `isTimeUnit` + `convertTimeValue` with category-aware versions that work across time, units-package categories, and Custom (exact-string) units. Foundation for U2/U3/U4/U6.

**Requirements:** R1, R2, R3, R4 (foundation — every cancellation path uses these helpers)

**Dependencies:** None.

**Files:**
- Modify: `impl/interpreter/rate_arithmetic.go` (rename and rewrite `isTimeUnit` and `convertTimeValue`; preserve helper-list shape)
- Test: `impl/interpreter/rate_arithmetic_test.go` (new — tests the helpers in isolation)

**Approach:**
- New helper `categoryOf(unit string) string`. Returns `"time"` if `types.IsValidDurationUnit(unit)`. Otherwise returns `units.CategoryForUnit(unit)` (which gives `"Length"` / `"Mass"` / `"DataSize"` / `"Custom"` / etc.). Empty input returns `""`.
- New helper `convertWithinCategory(value decimal.Decimal, fromUnit, toUnit string) (decimal.Decimal, error)`. If `fromUnit == toUnit`, return `value` unchanged. Else dispatch by category: time uses `Duration.Convert`; everything else uses `units.Convert`. Custom-category units only succeed when `fromUnit == toUnit`; otherwise return an error naming both units.
- Keep `isCurrencyCode` (currency reconstruction support — used in U3) untouched; it's correctly named and category-orthogonal.

**Patterns to follow:**
- PR #148's `rate_arithmetic.go` shape — small helpers with focused responsibilities.
- `spec/units/conversion.go:34-66` — error message format for cross-category attempts.

**Test scenarios:**
- *Happy path:* `categoryOf("hour")` → `"time"`. `categoryOf("kg")` → `"Mass"`. `categoryOf("box")` → `"Custom"`. `categoryOf("")` → `""`.
- *Happy path:* `convertWithinCategory(3, "week", "hour")` → `504, nil`. `convertWithinCategory(5, "kg", "g")` → `5000, nil`.
- *Edge case:* `convertWithinCategory(5, "box", "box")` → `5, nil` (same-string Custom).
- *Error path:* `convertWithinCategory(5, "box", "boxes")` → error if lexer doesn't normalize plurals to identical strings — verify behavior, document expectation.
- *Error path:* `convertWithinCategory(5, "kg", "hour")` → error mentioning both units (cross-category).
- *Error path:* `convertWithinCategory(5, "fortnight", "hour")` → error from `Duration.Convert` propagates cleanly.

**Verification:**
- The two helpers replace `isTimeUnit` and `convertTimeValue` at every call site in `rate_arithmetic.go`.
- All helper tests pass.
- `go build ./...` clean.

---

- U2. **Generalize the cancellation engine + retire the line-367 silent-coercion rule**

**Goal:** Make `Rate × Duration`, `Rate × Quantity`, and `Rate × Rate` use the same cancellation engine driven by U1's `categoryOf` predicate. Replace `operators.go:367-371`'s permissive `Rate × Quantity → Quantity` rule with the new engine + R6 refusal arm.

**Requirements:** R1, R2, R3, R4, R6

**Dependencies:** U1

**Files:**
- Modify: `impl/interpreter/rate_arithmetic.go` (rewrite `tryRateRateCancellation` to use category equality; rewrite `rateTimesDuration` to call `convertWithinCategory`; add `rateTimesQuantity` for the cross-Quantity case)
- Modify: `impl/interpreter/operators.go` (remove the line-367 silent-coercion branch; the new dispatch routes Rate × {Duration, Quantity, Rate} through the engine and falls through to R6 refusal in U4)
- Test: `impl/interpreter/rate_duration_arithmetic_test.go` (extend with AE3 cakes/box test, AE4 mph test for non-regression, R5 unchanged-scaling test)

**Approach:**
- Predicate: cancellation applies when `left.SomeUnit == right.SomeUnit` OR `categoryOf(left.SomeUnit) == categoryOf(right.SomeUnit)` AND that category supports conversion.
- For `Rate × Duration`: cancel `rate.PerUnit ↔ duration.Unit`. Multiply `rate.Amount.Value × convertedDuration.Value`. Reconstruct via `rateNumeratorAsResult` (untouched in this unit; U3 extends it).
- For `Rate × Quantity`: cancel `rate.PerUnit ↔ quantity.Unit`. Same multiply + reconstruct path.
- For `Rate × Rate`: two cancellation shapes from PR #148's `tryRateRateCancellation` (Shape 1: `left.PerUnit ↔ right.Amount.Unit`; Shape 2: `left.Amount.Unit ↔ right.PerUnit`). Both predicates generalize to `categoryOf` equality. Shape 1 produces `Rate{Amount: left.Amount × converted_right.Amount, PerUnit: right.PerUnit}` (cancellation reduces left's denominator); Shape 2 mirrors.
- Sequencing inside `evalBinaryOperation`'s rate-on-left branch:
  1. Rate × Number → Rate (existing, scale)
  2. Rate / Number → Rate (existing, scale)
  3. Rate / Rate same-PerUnit → Number (existing, ratio — left untouched per Decision 8)
  4. **Rate × Duration → cancellation engine** (NEW, generalized)
  5. **Rate × Rate → cancellation engine** (NEW, generalized)
  6. **Rate × Quantity → cancellation engine** (REPLACES old line-367 silent rule)
  7. Falls through to R6 refusal (U4 wires the message)
- **Predicate-matched-but-conversion-failed routing.** When the cancellation predicate matches (`categoryOf(left) == categoryOf(right)`) but `convertWithinCategory` returns an error (e.g., `cake × box` — both `Custom` but not equal strings), the engine treats this as a refusal case, NOT a generic error leak. The engine returns `(nil, false, nil)` (predicate didn't actually cancel) so dispatch falls through to U4's R6 contract message naming both operands. The convert-error string is dropped; the user-facing message is R6's friendly form. This avoids leaking internal `"cannot convert cake to box"` strings as user-facing errors.

**Execution note:** Test-first for the new acceptance examples (AE3, AE4, AE5, AE6). Each test names a specific input, action, and expected outcome.

**Patterns to follow:**
- PR #148's `tryRateRateCancellation` structure — the two-shape cancellation pattern is correctly factored.
- `spec/units/conversion.go` — error wrapping idiom (`fmt.Errorf("...: %w", err)`).

**Test scenarios:**
- *Happy path — Covers AE3.* `100 cakes / box * 5 boxes` → `Quantity{500, "cake"}` (or `"cakes"` depending on lexer normalization; assert via Value.String() and Unit field).
- *Happy path — Covers AE6.* `$100 / hour * 2` → `Rate{Amount: $200, PerUnit: hour}` (NOT a currency `$200`). Existing PR #148 test for this case stays.
- *Happy path:* `$100 / hour * 1 hour` → `$100` (Currency). Existing PR #148 test.
- *Happy path:* `$100 / hour * 3 weeks` → `$50,400` (Currency). Existing PR #148 test, verifies time-conversion path through `convertWithinCategory`.
- *Edge case:* `100 cakes/box * 0 boxes` → `0 cakes`. Numeric edges fall out of decimal arithmetic.
- *Integration:* `$100/hour * 40 hours/week * 3 weeks` → `$12,000` (Currency, chained reduction). Existing PR #148 test stays; verifies left-associative reduction through the generalized engine.
- *Error path — pre-condition for U4.* `$100 / hour * 5 kg` → returns no result and no panic from this unit; the operators.go fall-through routes to U4's refusal arm. Validate by asserting an error fires (exact message contract is U4's test scenarios).

**Verification:**
- All 7 PR #148 tests still pass.
- New AE3/AE4/AE5-pre/AE6 tests pass.
- `operators.go:367-371` silent-coercion rule is gone (grep returns no match).
- `go test ./...` clean.

---

- U3. **Currency identity preservation in result construction**

**Goal:** When the cancellation engine produces a Currency-shaped result, return a `*types.Currency` with both Symbol and Code populated correctly — not a `*types.Quantity` with the symbol stuck in `Unit`.

**Requirements:** R7

**Dependencies:** U2

**Files:**
- Modify: `impl/interpreter/rate_arithmetic.go` (`rateNumeratorAsResult` — extend with `IsCurrencyCode` check)
- Test: `impl/interpreter/rate_duration_arithmetic_test.go` (extend with R7 currency-preservation tests)

**Approach:**
- `rateNumeratorAsResult(rate, value)` already checks `unit ∈ SymbolToCode keys` and returns `Currency`. Extend to also call `types.IsCurrencyCode(unit)` (which checks the Code-side of the map). If either matches, call `types.NewCurrency(value, unit)` — `NewCurrency` already canonicalizes Symbol vs Code (line 30-44 in `currency.go`).
- Empty unit → Number (existing behavior, locked).
- Otherwise → Quantity (existing behavior).
- No data-shape change to `Rate`. Decision 3 in this plan.

**Patterns to follow:**
- `spec/types/currency.go:30-44` — `NewCurrency` is the canonical constructor; trust its bidirectional logic.
- `docs/solutions/logic-errors/currency-synonym-symbol-vs-code-comparison.md` — never compare `.Symbol` directly; use `IsCurrencyCode` and `IsSameCurrency`.

**Test scenarios:**
- *Happy path — Covers AE1.* `cost = $100/hour; work = 3 weeks; fee = cost * work`. Result is `*types.Currency`. `result.Symbol == "$"`, `result.Code == "USD"`, `result.Value.String() == "50400"`.
- *Happy path:* `100 USD / hour * 3 weeks` → `*types.Currency`. `result.Symbol == "USD"`, `result.Code == "USD"`, `result.Value.String() == "50400"`. (User wrote ISO code; preserves it.)
- *Happy path:* `€50 / hour * 8 hours` → `*types.Currency`. `result.Symbol == "€"`, `result.Code == "EUR"`, `result.Value.String() == "400"`.
- *Happy path — Covers AE2 chain.* `100 USD/hour * 40 hours/week * 3 weeks` → `*types.Currency` with Code `"USD"`, value `"12000"`. Tests USD survives every intermediate.
- *Edge case:* Quantity-numerator rate (`40 hours/week × 3 weeks`) → `*types.Quantity` with `Unit == "hour"`, NOT Currency. Confirms the IsCurrencyCode branch doesn't false-positive on non-currency units.
- *Edge case:* unitless-numerator rate (`60 / second × 5 seconds`) → `*types.Number` with value `300`. Empty-unit branch.

**Verification:**
- Every Currency-rate-times-cancellable acceptance case returns `*types.Currency`, not `*types.Quantity`.
- `result.Symbol` matches the user's typed form ($, €, USD, EUR, etc.).
- `result.Code` is correctly canonicalized in every case.
- Existing rate tests in `rate_eval_test.go` still pass (no regression on rate construction).

---

- U4. **R6 refusal contract — explicit dimensional-mismatch errors**

**Goal:** When the cancellation engine doesn't fire, replace the old silent-coercion / generic-error fall-through with an explicit refusal that names both operands and explains the non-cancellation. Locks the substring contract from Decision 6.

**Requirements:** R6

**Dependencies:** U2 (provides the dispatch slot where U4's refusal lives)

**Files:**
- Modify: `impl/interpreter/rate_arithmetic.go` (new helper `rateMismatchError(left, right, reason) error`)
- Modify: `impl/interpreter/operators.go` (Rate-on-the-LEFT fall-through emits R6 refusal; Rate-on-the-RIGHT widening at lines 107-122 also gates on cancellability — silent-drop becomes refusal when the right rate's PerUnit doesn't cancel anything on the left)
- Test: `impl/interpreter/rate_duration_arithmetic_test.go` (refusal tests with substring assertions)

**Approach:**
- New helper `rateMismatchError(left *types.Rate, right types.Type, leftCat, rightCat string) error`. Returns an error message that includes:
  - `formatTypeForError(left)` — gives `"rate (100 $/h)"` or similar (existing utility).
  - `formatTypeForError(right)` — gives `"duration (5 kg)"` etc.
  - Both unit strings explicitly.
  - The category names when known (`"kg is a mass unit"`, `"hour is a time unit"`).
  - The phrase `"cancel"` so the test substring contract has an anchor.
- Special-case currency × currency: a different template explaining that multiplying two currencies has no meaning.
- Special-case empty PerUnit on a rate: a "rate has no denominator unit" message.
- Wire into operators.go at two sites:
  1. Rate-on-left fall-through (after U2's branches don't match).
  2. Rate-on-right widening (lines 107-122): if `left` isn't a number/quantity-that-cancels, refuse instead of silent-drop.

**Patterns to follow:**
- `impl/interpreter/operators.go:894-899` — existing `cannot multiply X by Y directly` template style.
- `formatTypeForError` (line ~916) — operand display utility.

**Test scenarios:**
- *Error path — Covers AE5.* `$100 / hour * 5 kg` → error. Substring contract: error contains `"5 kg"` (or `"kg"`), contains `"hour"` (or `"time"`), contains `"cancel"`. Does NOT match the verbatim message format (allows refinement).
- *Error path:* `5 kg * 3 seconds` → error. Substring contract: contains both `"kg"` and `"seconds"` (or both unit names + categories).
- *Error path:* `$100 * $50` → error mentioning both currencies; uses the currency-×-currency template, not the cancellation template.
- *Error path:* `$100 / hour * 100 cakes` → error (Custom unit doesn't share category with time). Substring contract: `"cakes"` and `"hour"` mentioned.
- *Edge case:* Empty PerUnit (constructed in test directly, not via parser). Error message names "rate has no denominator unit" or equivalent.
- *Integration:* In a chained expression `$/hour * hours/week * 5 kg`, the first multiplication succeeds (gives `$/week`), the second fails. Error message names the intermediate `$/week` rate as the left operand (verifies `formatTypeForError` handles intermediate rates).

**Verification:**
- Every refusal test asserts via `strings.Contains` on the error message, not equality.
- No silent coercion remains. Grep `impl/interpreter/operators.go` for the old line-367 pattern returns nothing.
- `go test ./...` clean.
- Documents at `docs/solutions/feature-gaps/rate-type-arithmetic-widening.md` flag (Rate/Rate mismatched-PerUnit silent-error) is now closed by U2+U4 — can be updated in U5 if motivated.

---

- U5. **Documentation, registry, and regression goldens**

**Goal:** Update user-facing documentation to describe the new behavior, register the cancellation feature in the features registry, refresh the rates regression golden file, and stage the CHANGELOG entry for v2.1.

**Requirements:** Cross-cutting (every R is touched).

**Dependencies:** U2, U3, U4, U6 (all behavioral changes must exist before docs describe them).

**Files:**
- Modify: `CHANGELOG.md` — replace the current `[Unreleased]` rate-arithmetic stub (which describes time-only) with the generalized story matching this plan's scope.
- Modify: `spec/features/registry.go` — add or update the rate-arithmetic feature entry. Description names cross-domain cancellation, not just time. *(Mechanical: this is the registry that drives LSP completion + `cm features` output; unregistered feature → no completion hint for users typing `Rate × Duration`. Not origin-mandated, but operationally required for the feature to be discoverable.)*
- Modify: `site/content/docs/language-reference.md` (if it discusses Rate × Quantity behavior — verify in implementation; update if so).
- Modify: `site/content/docs/user-guide/units.md` (or equivalent — verify location during implementation; add a worked example for AE1, AE3, AE5).
- Modify: `testdata/eval/success/features/rates.cm` — add new test cases covering AE1-AE6. This is a regression golden; run `go test` to update expected output.

**Approach:**
- The CHANGELOG entry currently in `[Unreleased]` is a draft from PR #148's branch. Update its phrasing to match this plan's scope (cross-domain cancellation, R6 refusal contract, R7 currency identity).
- Features-registry entry: locate the existing rate-related entry (if any) and add a new one or extend it. Description should mention cancellation across categories.
- Regression golden in `testdata/eval/success/features/rates.cm`: append new cases. Each case is a calc line that exercises one acceptance example. Run the relevant test (likely `go test ./impl/document/...` or similar) to regenerate the expected output file if pattern-matching used.

**Test scenarios:**
- Test expectation: none — pure documentation and registration. The implementation work in U2/U3/U4 carries the behavioral test coverage. CHANGELOG and docs-site changes don't have unit tests; the docgen test verifies feature-registry shape after entries are added.

**Verification:**
- `go test ./spec/features/...` passes (registry shape stays valid).
- `go test ./impl/document/...` passes (regression goldens still match).
- `task check` (or equivalent) passes — confirms docs build and lint.
- A user reading `CHANGELOG.md` understands what shipped without needing to read the brainstorm doc.

---

- U6. **Quantity-with-speed-unit coercion enables `mph × hours` (AE4)**

**Goal:** Make `60 mph × 2 hours → 120 miles` work by coercing the left-side Speed-Quantity to a Rate before the cancellation engine sees the operands. Mirrors the existing right-side rate-widening at `operators.go:107-122`.

**Requirements:** R2 (specifically AE4 — the cross-domain physical-math case the brainstorm called out as canonical).

**Dependencies:** U2 (the Rate × Duration cancellation engine must exist for the coerced form to land somewhere).

**Files:**
- Modify: `impl/interpreter/operators.go` (add a left-side widening branch ahead of the rate-on-left dispatch — specifically: when `left` is `*types.Quantity` with a Speed unit and `right` is `*types.Duration` and `operator == "*"`, call `coerceSpeedToRate(left)` and recurse with the coerced Rate as the new left)
- Test: `impl/interpreter/rate_duration_arithmetic_test.go` (add the AE4 test that this unit unblocks)

**Approach:**
- Use the existing `coerceSpeedToRate` function in `impl/interpreter/speed_bridge.go`. It already handles `mph → miles/hour`, `kph → km/hour`, `mps → m/s`, `knots → nautical-miles/hour`. No new conversion logic.
- Branch placement: BEFORE the existing rate-on-left dispatch (so the coerced form flows through U2's engine). Likely between operators.go's existing widening blocks (around line 122-156 area).
- Symmetric coercion for `Duration × Quantity-with-speed-unit` is NOT in scope (operator precedence + existing widening handles `Number × Rate` widening; speed-quantity-on-right would need a parallel branch). The brainstorm puts rate-on-left as canonical; AE4 is `60 mph × 2 hours` (rate-shaped on left). Leave the commuted form as a follow-up if a real use case appears.
- Out of scope: any other `Quantity × Duration` cancellation. Only Speed-shaped Quantities widen here. `5 kg × 3 hours` does not widen (no compound-rate interpretation); falls through to U4's R6 refusal.

**Patterns to follow:**
- `impl/interpreter/operators.go:107-122` — existing right-side widening is the structural twin.
- `impl/interpreter/speed_bridge.go` — `coerceSpeedToRate` signature and behavior.

**Test scenarios:**
- *Happy path — Covers AE4.* `60 mph * 2 hours` → `Quantity{120, "mile"}` (or `"miles"` per lexer normalization). Assert via Value.String() and Unit field.
- *Happy path:* `100 kph * 30 minutes` → `Quantity{50, "km"}` (after coercion: `100 km/hour * 30 minutes` → cancel `hour` ↔ `minutes` via U2 → `50 km`).
- *Edge case:* `5 kg * 3 hours` → ERROR (kg is not a Speed unit; coercion doesn't fire; falls through to U4's R6 refusal). Confirms U6 is narrow.
- *Edge case:* `60 mph * 2 hours in km` → `120 miles in km` evaluates correctly through both U6's coercion and the existing `as`-conversion path.
- *Integration:* Verify the existing `60 mph in km/hour` (the speed-bridge's primary use case) still works — no regression on the `as` path.

**Verification:**
- AE4 test passes.
- The existing `testdata/eval/success/features/speed_rate_bridge.cm` continues to pass (no regression on the `as` path).
- `go test ./...` clean.

---

## System-Wide Impact

- **Interaction graph:** `evalBinaryOperation` is the single dispatch site. Three branches are touched: Rate-on-left dispatch (`operators.go:339-372`, replaced by U2's engine + U4's refusal), Rate-on-right widening (`operators.go:107-122`, gated by U4 to refuse on dimensional mismatch), and a new left-side Quantity-with-Speed-unit widening (added by U6, ahead of the rate branches). No callbacks, observers, or middleware involved.
- **Error propagation:** R6 refusals bubble up through the existing diagnostic path. No new error types; messages compose into the standard "evaluation error" wrapper.
- **State lifecycle risks:** None. Pure functions throughout; no persistence, no caching.
- **API surface parity:** `impl/interpreter/rate_eval.go` (the runtime evaluator) and `spec/document/literal_eval.go` (the frontmatter-globals lite evaluator) both have rate-construction switches. Both already get the `case *types.Duration:` arm from PR #148. R3 is verified twice.
- **Integration coverage:** AE2's chained reduction (`rate * density * weeks`) is the integration scenario par excellence — it touches Rate × Rate (intermediate) and Rate × Duration (final) in a single expression. AE5 + the chained-failure test in U4 verify error propagation through intermediates.
- **Unchanged invariants:**
  - The `Rate` struct shape (`{Amount *Quantity, PerUnit string}`) is NOT changed. Decision 3 (reverse-lookup) avoids the data-shape change.
  - PR #148's helper-list shape in `rate_arithmetic.go` is preserved (rebase, not rewrite).
  - The speed↔rate bridge in operators.go is untouched (Decision 5).
  - `Rate × Number → Rate` (R5 scaling) is locked unchanged.
  - The 7 existing PR #148 value tests survive unchanged.

---

## Risks & Dependencies

| Risk | Mitigation |
|---|---|
| Custom-unit cancellation depends on the lexer normalizing plurals (`box` vs `boxes`). If it doesn't, AE3 would fail with a "cannot convert box to boxes" error. | U1's test scenarios explicitly verify this assumption. If the lexer doesn't normalize, surface in PR review and either (a) add normalization in `categoryOf` for Custom units, or (b) document the limitation. |
| Replacing line-367's permissive `Rate × Quantity` rule may break tests outside the rate-arithmetic suite that depended on the silent coercion. | U2's test sweep runs `go test ./...` — any regression surfaces immediately. The pre-existing learnings doc flags the rule as a gotcha; production callers depending on it are unlikely. |
| Currency-code preservation through chains assumes every intermediate Rate's Amount.Unit carries the user's typed Symbol/Code string. Verified by reading PR #148's `rate_eval.go` change but worth re-confirming during U2 implementation. | U3's test scenarios include both `$` and `USD` chains; if either drops the identity, the test fails. |
| The pre-existing Rate-on-RIGHT widening at operators.go:107-122 silently drops the rate's denominator. U4 changes this to a refusal. May break surprising existing tests. | Run `go test ./...` after U4. If a legitimate use case surfaces (e.g., `Number × Rate` for scaling), keep the widening for that case and refuse only when the right Rate's denominator can't cancel anything. |
| `IsExplicit` flag policy on cancellation results is deferred. Display auto-scaling could produce surprising output (e.g., `120 hours` displayed as `5 days`). | Implementer flags this as a follow-up if it surfaces. Not blocking for v2.1. |

---

## Documentation / Operational Notes

- v2.1.0 release notes (CHANGELOG entry) mention the breaking change from line-367's silent coercion to refusal. Users with documents that exercised the old `Rate × Quantity` silent rule will see new errors. Migration: if the user wrote `100 X/server * 5 servers` expecting it to compose, this still works (server unit cancellation). If they wrote `$/hour * 5 kg` expecting `500 kg`, they'll now see a clear error directing them to fix the expression.
- No rollout, monitoring, or feature-flag concerns — pure language semantics; no production state.
- The brainstorm doc and this plan together form the v2.1.0 design record. Linkable from the CHANGELOG entry.

---

## Sources & References

- **Origin document:** `docs/brainstorms/2026-04-30-rate-arithmetic-least-surprise-requirements.md`
- **Reference PR:** `feat/rate-duration-arithmetic` branch (commit `25019d9`) — time-only first slice; rebase target.
- **Related code:**
  - `impl/interpreter/operators.go` — dispatch site
  - `impl/interpreter/rate_arithmetic.go` (PR #148; absorbed and extended here)
  - `spec/types/rate.go`, `spec/types/duration.go`, `spec/types/currency.go`
  - `spec/units/conversion.go` — generic converter
  - `impl/interpreter/rate_eval.go`, `spec/document/literal_eval.go` — twin construction sites
- **Related learnings:**
  - `docs/solutions/feature-gaps/rate-type-arithmetic-widening.md`
  - `docs/solutions/logic-errors/adding-new-type-fraction-cross-layer-checklist.md`
  - `docs/solutions/ui-bugs/display-formatter-overrides-explicit-unit-conversion.md`
  - `docs/solutions/logic-errors/currency-synonym-symbol-vs-code-comparison.md`
- **Related PRs/issues:** #148 (parked draft, time-only first slice — to be rebased onto this plan or closed-and-replaced).
