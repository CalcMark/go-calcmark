---
title: "compound() ignores bare frequency modifiers (monthly, quarterly)"
date: 2026-03-09
category: logic-errors
tags:
  - compound-interest
  - frequency-adverbs
  - financial-functions
  - nl-parser
  - interpreter
components:
  - impl/interpreter/growth_functions.go
  - spec/parser/nl_growth_functions.go
  - spec/parser/rdparser.go
severity: high
github_issues:
  - 40
related_issues:
  - 30
  - 32
symptoms:
  - "compound($1K, 5%, 10, monthly) and compound($1K, 5%, 10, quarterly) produce identical results ($1,628.89)"
  - "Bare frequency adverbs (monthly, quarterly, weekly, daily, yearly) silently ignored in interpreter"
  - "NL parser consumes frequency adverbs as units on preceding percentage (5% monthly becomes QuantityLiteral)"
---

# compound() ignores bare frequency modifiers (monthly, quarterly)

## Symptom

`compound($1K, 5%, 10, monthly)` and `compound($1K, 5%, 10, quarterly)` both produce `$1,628.89`. Changing the 4th argument has no effect on the output. The NL form `compound $1K by 5% monthly over 10 years` also produces the wrong result.

Expected: `monthly` ($1,647.01) and `quarterly` ($1,643.62) should produce different results via the financial compounding formula `A = P(1+r/n)^(nt)`.

## Root Cause (Two-Part)

### Part A: Interpreter ignoring bare adverbs

In `evalCompoundFunc`, the 4-arg path only checked for the `compounded:` prefix on the modifier identifier. When the user wrote `compound($1000, 5%, 10, monthly)`, the modifier reached the interpreter as a bare identifier `"monthly"`. It had no `compounded:` prefix, so it fell through to "Mode 2" which ran `compoundGrowth(principal, rate, periodsNum)` — ignoring the modifier entirely.

This is the **"open-ended else" anti-pattern**: a chain of `if/else if` ending with a catch-all that silently succeeds. Unrecognized inputs produce wrong results instead of errors.

### Part B: Parser eating adverbs as unit names

In the NL syntax path, `parsePrimary` in `rdparser.go` greedily consumed any following identifier as a unit suffix on a number literal. When parsing `5% monthly`, it produced `QuantityLiteral{Value: "5%", Unit: "monthly"}`. The token `monthly` was consumed before the NL compound parser ever saw it.

The root cause: `isNaturalSyntaxKeyword` did not include frequency adverbs, so the unit-consumption logic treated them as arbitrary units.

## Investigation Steps

1. Reproduced with CLI: `echo 'compound($1K, 5%, 10, monthly)' | ./cm --format json` — confirmed identical output for both `monthly` and `quarterly`.
2. Traced `evalCompoundFunc` — confirmed the `compounded:` prefix check was the only entry point for financial compounding.
3. Wrote failing test `TestCompoundBareFrequencyModifier` — confirmed the functional syntax bug.
4. Fixed interpreter — test passed but `TestCompoundGrowthMode2` broke, revealing the need to distinguish frequency adverbs from base period names.
5. Tested NL syntax — discovered the second bug: parser consuming adverbs as units.
6. Traced `parsePrimary` — confirmed `isNaturalSyntaxKeyword` was missing frequency adverbs.

## Solution

### Fix 1: Interpreter — `isFrequencyAdverb` guard (`growth_functions.go`)

Added a `frequencyAdverbs` map and `isFrequencyAdverb` helper. In `evalCompoundFunc`, after the `compounded:` prefix check fails, a new branch checks for bare frequency adverbs and routes to financial compounding:

```go
var frequencyAdverbs = map[string]bool{
    "daily": true, "weekly": true, "monthly": true,
    "quarterly": true, "yearly": true,
}

func isFrequencyAdverb(name string) bool {
    return frequencyAdverbs[strings.ToLower(name)]
}
```

### Fix 2: NL parser — `checkFrequencyAdverb` (`nl_growth_functions.go`)

Added a `checkFrequencyAdverb()` method. In `parseNLCompoundFunction`, after checking for `per` and `compounded`, a third branch handles bare adverbs by rewriting them to the canonical `compounded:<freq>` AST form:

```go
} else if p.checkFrequencyAdverb() {
    freq := strings.ToLower(string(p.peek().Value))
    p.advance()
    modifier = &ast.Identifier{Name: "compounded:" + freq}
}
```

### Fix 3: Parser — `isNaturalSyntaxKeyword` (`rdparser.go`)

Added frequency adverbs to the keyword exclusion list so `parsePrimary` doesn't consume them as unit suffixes:

```go
case "daily", "weekly", "monthly", "quarterly", "yearly":
    return true // Frequency adverbs used in compound() NL syntax
```

## Key Design Decision: Frequency Adverbs vs. Base Period Names

The fix draws a deliberate semantic distinction:

| Form | Type | Example | Behavior |
|------|------|---------|----------|
| `month`, `quarter`, `year` | Base period name (noun) | `compound($1K, 5%, 12, month)` | Mode 2: semantic annotation, math = 3-arg form |
| `monthly`, `quarterly`, `yearly` | Frequency adverb | `compound($1K, 5%, 10 years, monthly)` | Financial compounding: `A = P(1+r/n)^(nt)` |

**One-line summary: "month" tells the interpreter what the rate means; "monthly" tells the interpreter how to compound.** Conflating them produces silently wrong math.

## Prevention

### Patterns to watch for

1. **Silent fallthrough on unrecognized identifiers.** When a function dispatches on an identifier, the fallback must be an error, not silent success.
2. **Parser consuming tokens that belong to a different grammar production.** Any new modifier word must be added to `isNaturalSyntaxKeyword`.
3. **Duplicate truth sources.** The `frequencyAdverbs` map exists in both the parser and interpreter. A `TestFrequencyAdverbConsistency` test would catch divergence.
4. **Arg-count-correct but semantically-wrong arguments.** Cross-layer consistency checks (#32) validate count, not meaning. Functions with discriminant arguments need allowlist validation.

### Tests that catch this

| Test | What it catches |
|------|----------------|
| `TestCompoundBareFrequencyModifier` | Bare adverbs not triggering financial compounding |
| `TestCompoundNLBareFrequencyModifier` | NL syntax parity with functional syntax |
| `TestCompoundGrowthMode2` | Base period names incorrectly triggering financial compounding |
| `TestCompoundGrowthMode3_Ordering` | Monotonicity: yearly < quarterly < monthly < weekly < daily |

## Related

- [Issue #30](https://github.com/CalcMark/go-calcmark/issues/30) — Autosuggest/status bar missing Growth category and compound 4th param
- [Issue #32](https://github.com/CalcMark/go-calcmark/issues/32) — Cross-layer consistency checks for function arg specs
- [Issue #40](https://github.com/CalcMark/go-calcmark/issues/40) — This bug
- [Solution: autosuggest-status-bar-category-tag-display](../ui-bugs/autosuggest-status-bar-category-tag-display.md) — The #30 post-mortem that motivated #32 and led to discovering this bug
- [Plan: growth-functions](../../plans/2026-03-04-feat-growth-functions-plan.md) — Original growth functions design

## Chain of Discovery

Issue #30 (autosuggest bug) -> Issue #32 (cross-layer consistency checks) -> Issue #40 (bare frequency modifiers silently ignored). Each fix revealed the next layer of the problem.
