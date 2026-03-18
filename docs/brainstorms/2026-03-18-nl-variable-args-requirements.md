---
date: 2026-03-18
topic: nl-variable-args
---

# NL Function Variable Arguments

## Problem Frame

CalcMark's NL (natural language) function syntax rejects variable references as arguments for growth functions (`compound`, `grow`, `depreciate`). When a user writes `compound price by 5% over 10 years`, the document detector misclassifies the line as markdown prose because `looksLikeCalculation()` doesn't recognize the IDENTIFIER + IDENTIFIER pattern for growth function keywords. Users must fall back to functional syntax (`compound(price, 5%, 10 years)`) to use variables.

System NL functions (`read`, `compress`, `transfer`) already accept variables correctly — the detector handles them. The gap is only in growth functions.

Additionally, five locations must be updated for each new NL function (parser gates, NL parse functions, detector heuristics, classifier, LSP completions), creating ongoing drift risk. The existing feature registry (`spec/features/registry.go`) is the natural single source of truth but doesn't yet describe NL token patterns.

## Requirements

- R1. **Growth function variable args**: `compound <var> by`, `grow <var> by`, `depreciate <var> by` must classify as calculations in the document detector and evaluate correctly when the variable is defined.
- R2. **NL pattern registry**: Extend the existing feature registry's `Alias` pattern to describe NL function trigger keywords so the detector and classifier can derive their behavior from one definition. Follow the current approach where the LSP reads from the registry — no parser/lexer logic in the LSP.
- R3. **Parity test**: A test must exercise every NL function with both literal and variable arguments through both classification paths (detector + classifier) and assert they agree.

## Success Criteria

- `compound price by 5% over 10 years` evaluates correctly when `price` is defined (not treated as markdown)
- Same for `grow` and `depreciate` with variable arguments
- Adding a new NL function requires updating only the feature registry + parser; detector and classifier derive behavior automatically
- Parity test fails if a new NL function is added to the parser but not the registry

## Scope Boundaries

- Parser gates and NL parse functions themselves are NOT changing — they already handle variables correctly via `parseExponent()`.
- The classifier (`spec/classifier/classifier.go`) already handles growth keywords; no changes needed there.
- Not changing how NL functions evaluate — only how lines are classified and how the LSP offers completions.

## Key Decisions

- **Registry-driven approach over ad-hoc patches**: Rather than just adding growth patterns to `looksLikeCalculation()`, we're extending the feature registry to be the source of truth. This prevents the same drift from recurring.
- **No new LSP logic**: The LSP already reads from the feature registry cleanly. NL completions continue to flow through the existing `Alias`/`Example` pattern — no parser/lexer concerns leak into the LSP.

## Dependencies / Assumptions

- The feature registry already has `Alias.Parseable` and `Alias.Example` fields — R2 extends this with structured pattern data rather than replacing it.
- The LSP completion handler already reads from the feature registry (lsp/completion.go:104) — no changes needed there.

## Outstanding Questions

### Deferred to Planning

- [Affects R2][Needs research] What's the minimal struct shape for NL patterns in the registry? Need to examine all 6 NL functions to find the common pattern (trigger + args + separators).

## Next Steps

→ `/ce:plan` for structured implementation planning
