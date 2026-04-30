---
date: 2026-04-30
topic: rate-arithmetic-least-surprise
---

# Rate Arithmetic — The "Least Surprise" Principle

## Problem Frame

CalcMark's rate type (`$/hour`, `MB/s`, `30 cakes/day`) is half-built. Multiplying a rate by a duration or by a quantity with cancellable units fails today with `cannot multiply rate (...) and duration (...)`. A user typing the most natural-language thing they can — `cost = $100/hour; work = 3 weeks; fee = cost * work` — hits a wall in the language and has to manually convert (`3 weeks in hours`, then multiply, then drop the unit). The same wall blocks `100 cakes/box × 5 boxes`, `5 m/s × 3 seconds`, and any other domain where humans casually compose a rate with what cancels its denominator.

The shipped 2.0.0-rc.1 inherits this gap from v1; rc.1 is incomplete in the sense that the rate type doesn't behave the way humans expect. PR #148 implemented a time-only slice and was paused for design.

This brainstorm establishes the **principle** that anchors a complete fix, then locks the scope.

---

## The Principle

> **CalcMark works when a normal human reading the expression aloud would expect it to work — and refuses with a clear message when they wouldn't.**

The principle is a *test*, not an algorithm. For any candidate expression: read it aloud as a sentence. If it sounds like math a non-technical person would say in a meeting or write on a napkin, CalcMark must produce an answer. If no human would say it and mean anything, CalcMark must refuse.

The principle accepts educational surprises. `3 weeks × $100/hour = $50,400` because `3 weeks` is `504 work-hours` if you actually work 24/7 — but a casual user might have meant `3 work weeks = 120 hours = $12,000`. That's a user-error worth correcting on the user's side, not a CalcMark error worth refusing on. The number is right for the input given.

---

## Requirements

### R1 — Rate × Duration must compute when the rate's denominator is a time unit

`$100/hour * 3 weeks` → `$50,400` (Currency).
`$100/hour * 1 hour` → `$100` (Currency, no conversion).
`30 cakes/day * 1.5 months` → `~45 cakes` (Quantity, with the standard month-as-30-days approximation).

The duration is converted into the rate's `PerUnit`, multiplied against the rate's `Amount`, and the time dimension is consumed. The result type tracks the rate's numerator: currency-numerator → Currency, quantity-numerator → Quantity, unitless-numerator → Number.

### R2 — Rate × Quantity must compute when the rate's denominator matches the quantity's unit (or converts)

`100 cakes/box * 5 boxes` → `500 cakes` (Quantity, count-unit cancellation).
`5 GB/server * 50 servers` → `250 GB` (Quantity, count-unit cancellation).
`60 mph * 2 hours` → `120 miles` (Quantity, time/speed cancellation — already works via the existing speed↔rate bridge).

Same engine as R1, but the predicate generalises from "is a time unit" to "shares a unit category and converts." The converters that already exist for time, mass, length, data, volume, etc. all compose without further work; the only new logic is the shared-category detection and the result-type reconstruction.

### R3 — Rate literals accept a Duration numerator

`40 hours / week` constructs a `Rate{Amount: 40 hours, PerUnit: week}`. Today this errors with "rate amount must be a number, quantity, or currency, got *types.Duration". Without R3, R1's `40 hours/week × 3 weeks → 120 hours` chained form is unreachable.

### R4 — Rate × Rate cancels matching units across the two factors

`$/hour * hours/week → $/week` (a Rate).
`$/hour * 40 hours/week * 3 weeks → $12,000` (Currency, by left-associative reduction).

The cancellation predicate from R2 applies symmetrically: if `left.Amount.Unit` cancels `right.PerUnit`, the result is a Rate with the surviving numerator + denominator; if a chained step reduces both denominators away, the result is the surviving numerator type.

### R5 — Rate × Number scales the rate (existing behavior, locked)

`$100/hour * 5` → `$500/hour` (a scaled Rate, not a Currency). Confirms today's behavior. A bare integer is *not* interpreted as an implicit duration in the rate's unit — that would be too far from how humans actually talk.

### R6 — Inconsistent quantity types must error with a clear message

`$100/hour * 5 kg` → ERROR (kg is mass, the rate's denominator is time, no cancellation possible).
`$100 * $50` → ERROR (no human says "fifty dollars times a hundred dollars" meaning anything).
`5 kg * 3 seconds` → ERROR (no shared category, no cancellation).

The error message must:
1. Name both operands and their kinds (e.g., "100 $/hour" and "5 kg").
2. Explain *why* they don't compose ("kg is a mass unit; the rate's denominator is hour, a time unit — there's nothing to cancel").
3. Avoid suggesting fixes that don't exist (no "did you mean...?" unless the alternative is well-defined).

### R7 — Result-type reconstruction preserves currency identity

`100 USD/hour * 3 weeks` → `50,400 USD` (preserves the ISO code).
`$100/hour * 3 weeks` → `$50,400` (preserves the symbol).

Today the rate stores currency as `Quantity{Value, Unit: "$"}` (or `"USD"` etc.) — losing the distinction between symbol and code path. R7 requires that the reconstructed Currency carries forward what the user wrote, not a normalised form.

---

## Acceptance Examples

**AE1.** Covers R1, R7. The user's original 2026-04-30 case:
```calcmark
cost = $100/hour
work = 3 weeks
fee = cost * work     # → $50,400.00
```

**AE2.** Covers R1, R3, R4. The chained form:
```calcmark
rate = $100 / hour
density = 40 hours / week
weeks = 3 weeks
salary = rate * density * weeks   # → $12,000.00
hourly_to_weekly = rate * density # → $4,000.00/week (intermediate Rate)
```

**AE3.** Covers R2 (count-unit cancellation, non-time domain):
```calcmark
per_box = 100 cakes / box
order = 5 boxes
total = per_box * order    # → 500 cakes (Quantity)
```

**AE4.** Covers R2 (existing speed↔distance):
```calcmark
speed = 60 mph
trip = 2 hours
distance = speed * trip    # → 120 miles
```

**AE5.** Covers R6 (refusal with helpful message):
```calcmark
hourly = $100 / hour
weight = 5 kg
nope = hourly * weight     # → ERROR: cannot multiply $100/hour by 5 kg —
                           #   kg is a mass unit; the rate's denominator
                           #   is hour, a time unit. No shared dimension
                           #   to cancel.
```

**AE6.** Covers R5 (Rate × Number is scaling, not implicit duration):
```calcmark
rate = $100 / hour
doubled = rate * 2         # → $200/hour (Rate, scaled)
                           # NOT $200 (would imply "2 hours")
```

---

## Success Criteria

1. **Every example a normal human would write** — across the wages, recipe, capacity, and physical-math domains demonstrated in AE1-AE4 — produces an answer without explicit `in <unit>` conversions.
2. **Every dimensionally-incoherent expression** that wouldn't sound like math when read aloud (AE5 and analogues) errors with a message that names both operands and explains the non-cancellation.
3. **No silent coercion remains** for cases the language can't actually compute. The long-standing `Rate × Quantity → Quantity` rule that ignores `PerUnit` is replaced by R2's explicit cancellation check.
4. **The 7 concrete value tests in the existing PR #148 still pass.** They encode R1/R3/R4 and survive the design generalisation.
5. **A new test for each refusal** (AE5 and analogues) locks the error message contract — both that an error fires and that it names the offending units.

---

## Scope Boundaries

**In scope:**
- Time-unit cancellation (the R1 case).
- Cross-domain same-category cancellation (R2): time, mass, length, data, volume, count, anything `units` package knows how to convert.
- Duration-numerator rates (R3).
- Two-rate cancellation in left-associative chains (R4).
- Currency-symbol vs ISO-code preservation in reconstructed Currency results (R7).
- Refusal-with-message for dimensional mismatches (R6).

**Deferred for later (post-v2 follow-ups):**
- **Operator precedence.** `3 weeks * $100/hour` parses as `((3 weeks) * $100) / hour` rather than `(3 weeks) * ($100/hour)`. The natural human reading is the latter; rate-on-left is the canonical workaround. A parser-level rate-literal precedence change is a separate brainstorm — the principle would say it should work, but the parser change is non-trivial.
- **Subtractive arithmetic** between rates and durations (`$5000 / hour - 3 weeks`?). Probably nonsense in most domains; revisit if a real use case appears.
- **Inverse composition** (`work / rate = duration`). `$5000 / ($100/hour) = 50 hours` is something humans say. Same engine could handle it; out of scope here to keep the multiplication contract focused.
- **Adding rates with different denominators** (`$100/hour + $50/day`). Mathematically definable (convert one to the other) but rarely written by humans; defer until requested.

**Outside this product's identity:**
- **Full dimensional algebra.** No `kg·m/s² → newton` reduction, no `m/s × m/s → m²/s²`, no exponent-vector unit tracking. CalcMark is a calculation notebook, not Wolfram Alpha. The principle "humans say this should work" doesn't extend to expressions with squared units — humans don't talk that way outside physics class.
- **Implicit unit inference.** `5 * 3 seconds` does not become `5 seconds × 3 seconds`. Bare numbers stay bare numbers (R5).
- **Currency × Currency arithmetic.** Multiplying two currencies has no meaning; refuse always.

---

## Key Decisions

1. **Approach B over A and C.** Same-dimension cancellation across all unit categories — not just time (A would honor the principle only for time-domain users) and not full dimensional algebra (C would be months of work for cases CalcMark's audience doesn't write).

2. **Conversion direction: into the rate's `PerUnit`.** When `Rate × Duration` and the units don't match, convert the duration into the rate's denominator unit (not the other way). Rationale: the rate is the more "structured" of the two — it explicitly named a denominator unit — so converting toward it preserves the rate's frame. Symmetric for `Rate × Quantity`.

3. **Result-type reconstruction.** The cancelled denominator leaves the rate's numerator standing. If the numerator is a currency-shaped Quantity (Unit = "$" / "USD" / etc.), the result is a Currency. If the numerator is a unitless Quantity (Unit = ""), the result is a Number. Otherwise it's a Quantity. The lookup uses the existing `SymbolToCode` map plus a reverse-pass for ISO codes.

4. **Refusal messaging is part of the contract, not an afterthought.** Every error in R6's class is tested for content (R6.1-R6.3 in the success criteria). The message names both operands' kinds and explains the non-cancellation. No "did you mean...?" hints unless the alternative is well-defined.

5. **PR #148 implementation pivots, not rebuilds.** The existing `rateTimesDuration` and `tryRateRateCancellation` functions become the time-domain specialisation of a `tryRateCancellation` that takes a category-comparison predicate. The 7 value tests stay; new tests cover R2's non-time-domain cases plus R6's refusals.

---

## Dependencies / Assumptions

- The existing `units` package (`spec/units/`) already exposes converters for every category CalcMark supports (time, mass, length, data, volume, etc.). R2 reuses these directly. **Assumption to verify in planning:** the converter API is uniform enough that a generic `convertWithinCategory(value, fromUnit, toUnit)` can be written without category-specific branches.
- The current rate-literal parser correctly emits `RateLiteral{Amount, PerUnit}` for all the input shapes the principle requires (`X / Y`, `X per Y`, multi-word `X per Z Y`). **Assumption to verify:** `40 hours / week` parses identically to `40 hour/week`. Tests today imply yes; planning should confirm.
- `types.Currency.Symbol` carries the user-typed form ("$" or "USD") and `types.Currency.Code` carries the canonical ISO. R7 depends on these being preserved through `evalRateLiteral` — they currently are not (rate stores only the symbol). The fix is local but it's a small data-shape change to `Rate.Amount`.

---

## Outstanding Questions

None blocking. The five questions raised in the PR #148 pause comment are now resolved:

| PR #148 question | Resolution in this doc |
|---|---|
| Conversion direction | Key Decision 2 — convert toward the rate's `PerUnit`. |
| Dimensional mismatch | R6 + Success Criteria 2-3 + AE5 — refuse with named-units message. |
| Currency-numerator detection | Key Decision 3 + R7 — reverse-lookup against `SymbolToCode`, preserve user's typed form. |
| Operator precedence | Scope Boundaries → Deferred — rate-on-left is canonical; parser change is a separate brainstorm. |
| Full dimensional algebra | Scope Boundaries → Outside identity — explicit non-goal. |

---

## Next Steps

1. **`/ce-plan`** against this requirements doc. Planning will produce U-IDs that map onto:
   - The generic cancellation engine (replaces PR #148's time-only helpers).
   - The Duration-as-rate-numerator change (R3, ~10 lines, two sites).
   - The result-type reconstructor with currency-identity preservation (R7).
   - The refusal-message contract (R6) including the new tests.
   - Dropping or replacing the long-standing `Rate × Quantity → Quantity` permissive rule.
2. **PR #148 disposition.** The existing implementation is a time-only slice of the right design. Planning should decide whether to rebase-and-extend or close-and-replace; the 7 value tests survive either way.
3. **Ship as v2.1.0** post v2.0.0 stable. Out of the soak window — adding behavior to rc.1 isn't called for, and v2.1.0 is the natural home for "rate type completes."
