---
title: "NL and functional syntax require independent testing — fixes do not propagate across parser paths"
date: 2026-03-09
category: integration-issues
tags:
  - nl-syntax
  - functional-syntax
  - parity-testing
  - documentation-staleness
  - testing-methodology
components:
  - spec/parser/rdparser.go
  - spec/parser/nl_growth_functions.go
  - impl/interpreter/growth_functions.go
  - site/content/
severity: moderate
github_issues:
  - 40
related_issues:
  - 30
  - 32
symptoms:
  - "Fix applied to functional syntax path does not resolve equivalent NL syntax bug"
  - "NL parser keyword guards (isNaturalSyntaxKeyword) missing new tokens causes parsePrimary to consume them prematurely"
  - "Documentation shows stale syntax examples after language changes ship"
---

# NL and functional syntax require independent testing

## The Core Lesson

In a dual-syntax language, **every fix is two fixes, every feature is two features, and every test that only covers one syntax form is half a test.**

This lesson emerged from issue #40, where fixing `compound($1K, 5%, 10, monthly)` in the interpreter did nothing for the NL form `compound $1K by 5% monthly over 10 years`. The NL path had a completely separate bug in the parser. Additionally, the documentation site was never updated to reflect the new bare-adverb syntax.

## Why Parity Breaks

CalcMark's two syntax forms go through completely separate parser code paths that converge only at the AST level:

```
Functional:  compound($1K, 5%, 10, monthly)
                 │
                 ▼
           parseFunctionCall()        ──► ast.FunctionCall{args...}
           (rdparser.go)                         │
                                                 ▼
NL:        compound $1K by 5% monthly over 10y   │
                 │                               │
                 ▼                          evalCompoundFunc()
           parseNLCompoundFunction()  ──►   (interpreter)
           (nl_growth_functions.go)
```

A fix at any layer can leave the other path broken:

| Layer | How parity breaks | Example from #40 |
|-------|-------------------|-------------------|
| **Parser** | NL parser doesn't recognize a new token | `monthly` consumed as unit suffix on `5%` |
| **Semantic checker** | Different validation for NL-constructed AST | Not hit here, but possible |
| **Interpreter** | Fix handles one AST shape but not another | `isFrequencyAdverb` only helped functional path |

## The Three-Tier Testing Pattern

Tests should be structured in three tiers to catch parity bugs:

**Tier 1 -- Absolute value tests:** Assert that a specific input produces a specific numeric output. Tests one syntax form in isolation.
```
compound($1000, 5%, 10 years, compounded monthly) = $1647.01
```

**Tier 2 -- Intra-syntax equivalence:** Assert that syntactic sugar within one form equals the explicit form.
```
compound($1000, 5%, 10 years, monthly) == compound($1000, 5%, 10 years, compounded monthly)
```

**Tier 3 -- Cross-syntax parity:** Assert that NL output equals functional output. This is the safety net.
```
"compound $1K by 5% monthly over 10 years" == "compound($1000, 5%, 10 years, monthly)"
```

Without Tier 3 tests, the bare-frequency fix would have shipped with NL syntax still broken.

### Existing parity test coverage (gaps identified)

| Function | Tier 3 parity tests? | Location |
|----------|---------------------|----------|
| read, compress, transfer | Yes | `nl_equivalence_test.go` |
| avg, sqrt | Yes | `nl_functions_comprehensive_test.go` |
| compound, grow, depreciate | **Partial** (added in #40) | `growth_functions_test.go` |
| capacity | **No** | -- |

## The Documentation Staleness Problem

Code changes without doc updates is a recurring pattern. After issue #40, these files show stale compound() syntax:

- `site/content/docs/user-guide.md` -- Only shows `compounded monthly`, not bare `monthly`
- `site/content/docs/language-reference.md` -- NL syntax table missing bare adverb form
- `site/content/docs/examples/functions-and-nl.md` -- Only shows prefixed form
- `site/content/docs/examples/investment-growth.md` -- Only shows `compounded monthly`

The fix shipped. The docs did not. Users reading the documentation won't discover the simpler syntax.

## CalcMark Change Checklist

For any change to function behavior, modifiers, or syntax:

**Syntax coverage:**
- [ ] Functional syntax (`fn(args)`) tested with expected output
- [ ] NL syntax (`fn X by Y over Z`) tested with expected output
- [ ] Tier 3 cross-syntax parity test asserting identical results
- [ ] Edge cases tested in both forms

**Parser guards:**
- [ ] `isNaturalSyntaxKeyword()` updated if a new contextual keyword was added
- [ ] `checkFrequencyAdverb()` / shared lookup tables updated if a new modifier was added
- [ ] No duplicate lookup tables between parser and interpreter (or consistency test exists)

**Golden files and registry:**
- [ ] `testdata/eval/success/features/` golden file updated with both syntax forms
- [ ] `spec/features/registry.go` Example and NLExample fields accurate
- [ ] `spec/semantic/` FunctionSpec updated if args changed

**Documentation:**
- [ ] `site/content/docs/language-reference.md` NL mapping table updated
- [ ] `site/content/docs/user-guide.md` examples reflect current behavior
- [ ] Feature-specific example pages updated
- [ ] `site/content/docs/examples/functions-and-nl.md` updated if NL form changed

## Prevention Strategies

### 1. Eliminate duplicate truth sources

The `frequencyAdverbs` map exists in both `spec/parser/nl_growth_functions.go` and `impl/interpreter/growth_functions.go`. If one is updated without the other, the parser and interpreter disagree. Either define the canonical set once in the spec layer, or add a `TestFrequencyAdverbConsistency` test.

### 2. Executable documentation

The `.cm` examples in `site/content/docs/examples/` contain expected outputs like `-> $1628.89`. A CI step could extract CalcMark code blocks, run them through `./cm`, and diff against documented values. If language semantics change, CI fails before stale docs reach users.

### 3. Registry-driven doc validation

`spec/features/registry.go` stores Example and NLExample for every feature. A test can compare registry entries against published documentation to flag divergence.

### 4. PR-level reminder

Any PR touching `spec/parser/`, `spec/semantic/`, or `impl/interpreter/` should prompt: "Did you update `site/content/` docs?"

## Known Shared-State Risks

| Shared state | Locations | Risk |
|--------------|-----------|------|
| `frequencyAdverbs` map | parser + interpreter | Adding a new adverb to one but not the other |
| `isNaturalSyntaxKeyword` | `rdparser.go` | Missing a keyword lets `parsePrimary` consume it as a unit |
| `FunctionSpecs` vs `BuiltinFunctions` | spec + impl | Arg count/optionality drift (covered by #32 consistency tests) |
| Documentation site | `site/content/` | Falls behind every code change |

## Related

- [Issue #40](https://github.com/CalcMark/go-calcmark/issues/40) -- The bug that surfaced this lesson
- [Solution: compound-bare-frequency-modifier](../logic-errors/compound-bare-frequency-modifier-silently-ignored.md) -- The specific bug fix post-mortem
- [Issue #32](https://github.com/CalcMark/go-calcmark/issues/32) -- Cross-layer consistency checks (structural, not semantic)
- [Issue #30](https://github.com/CalcMark/go-calcmark/issues/30) -- Original autosuggest bug that started this chain
- [NL syntax limitations](../../../.claude/projects/-Users-bitsbyme-projects-go-calcmark/memory/nl-syntax-limitations.md) -- Known NL gaps from system-sizing rewrite
- [Plan: growth functions](../../plans/2026-03-04-feat-growth-functions-plan.md) -- Original design defining compound modes and NL syntax
