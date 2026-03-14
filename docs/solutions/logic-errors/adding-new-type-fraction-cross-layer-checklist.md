---
title: "Adding a new data type (Fraction) requires changes across all 8 layers"
date: 2026-03-14
tags:
  - type-system
  - cross-layer
  - lexer
  - parser
  - interpreter
  - formatter
  - tui
  - architecture
  - checklist
severity: high
component:
  - spec/types
  - spec/lexer
  - spec/ast
  - spec/parser
  - spec/classifier
  - spec/document/detector
  - spec/semantic
  - impl/interpreter
  - format/display
  - format/json_formatter
  - cmd/calcmark/tui/editor
  - cmd/calcmark/config
  - spec/features/registry
symptom: >
  When adding a new value type to CalcMark, incomplete layer coverage causes
  silent failures: values display as decimals instead of fractions, unit
  normalization is skipped, the classifier misses fraction-only lines, and
  machine output leaks display-only formatting.
root_cause: >
  CalcMark has 8+ layers that must all recognize a new type. Missing any one
  layer causes subtle bugs that are hard to trace because the system degrades
  gracefully (e.g., falling through to a default case) rather than erroring.
---

## The Problem

Adding the `Fraction` type (`math/big.Rat`) to CalcMark required touching 20+ files across 8 architectural layers. Several bugs were discovered only through manual testing because the system silently degraded rather than failing:

1. **Unit dropped on mixed numbers** — `12 1/2 pints` lost its unit because `evalFractionOperation` only preserved the left operand's unit (first-unit-wins), but mixed number desugaring puts the unit on the right operand (the fraction part).

2. **Display normalization skipped** — `287 1/2 pint` displayed as `287 1/2 pint` instead of auto-converting to `~36 gal` because the display formatter's `*types.Fraction` case called `v.String()` directly, bypassing `FormatQuantity` → `NormalizeForDisplay`.

3. **Markdown SmartypantsFractions** — gomarkdown's `html.CommonFlags` includes `SmartypantsFractions` which converts `1/2` → `½` in HTML. This could corrupt interpolated fraction values, but the sentinel markers (`\x02`/`\x03`) already protect them. This was a non-bug, but required investigation and regression tests to prove safety.

## Cross-Layer Checklist for New Types

Every new CalcMark type must be added to ALL of these locations. Use this as a checklist.

### Layer 1: Type Definition (`spec/types/`)

- [ ] New type struct (e.g., `Fraction` wrapping `*big.Rat`)
- [ ] Constructor with validation (e.g., `NewFraction` rejects zero denominator)
- [ ] `String()` method for ASCII display
- [ ] Add case to `ToDecimal()` in `number.go`
- [ ] Add case to `typeName()` in `number.go`

### Layer 2: Lexer (`spec/lexer/`)

- [ ] New `TokenType` constant (e.g., `FRACTION`)
- [ ] Add to `TokenType.String()` switch
- [ ] Scanning function — prefer pure function in separate file (e.g., `tryParseFraction`)
- [ ] Integrate into `readNumber()` or appropriate scan path
- [ ] Add to fuzzer corpus (`fuzz_test.go`)

### Layer 3: AST (`spec/ast/`)

- [ ] New node struct (e.g., `FractionLiteral`) with `Range` field
- [ ] `String()` and `GetRange()` methods
- [ ] Add case to `ContainsScaleRef()` (even if it returns false)

### Layer 4: Parser (`spec/parser/`)

- [ ] Handle new token in `parsePrimary()`
- [ ] Unit attachment logic (if the type can have units)
- [ ] Compound syntax (e.g., mixed numbers: `11 3/8`)
- [ ] Add to parser fuzzer corpus
- [ ] Range tracking: use `tokenRange()` for single tokens, `ast.SpanNodes()` for compound

### Layer 5: Classification (`spec/classifier/` + `spec/document/detector.go`)

- [ ] Classifier: add token to `containsOperators` or single-token literal check
- [ ] Detector `isNumberToken()`: add new token type
- [ ] Detector `looksLikeCalculation()`: add new token as calculation indicator

**Why this matters:** Without classifier/detector support, lines containing only the new type (e.g., `1/3`) are classified as markdown prose, not calculations. They silently disappear from evaluation.

### Layer 6: Semantic Analysis (`spec/semantic/`)

- [ ] Add case in `checkNode()` switch
- [ ] Implement `checkXxxLiteral()` with validation (e.g., zero denominator)
- [ ] Unit category restrictions (if applicable)

### Layer 7: Interpreter (`impl/interpreter/`)

- [ ] `evalNode()`: add case for new AST node → `evalXxxLiteral()`
- [ ] `evalXxxLiteral()`: construct the runtime type (with defense-in-depth validation)
- [ ] `evalBinaryOperation()`: normalization block at TOP for new type combinations
- [ ] `evalUnaryOperation()`: negation support
- [ ] `evalComparison()`: comparison support (often: convert to decimal, delegate)
- [ ] `ExtractNumber()`: for `number()` function support
- [ ] `evalSqrt()`: type-specific behavior (e.g., perfect-square detection)
- [ ] `evalNapkinConversion()`: napkin rounding for new type
- [ ] `evalPreciseConversion()`: usually a no-op pass-through
- [ ] `aggregateValues()`: for `sum()`/`avg()` support
- [ ] Helper functions: `typeToNumber()`, `numberToType()` conversion bridges

**Operator dispatch trap:** When the new type meets a non-numeric type (Currency, Duration, Rate), convert to Number/Quantity and fall through to existing dispatch. Don't add N×M type combinations.

### Layer 8: Formatters + Display

- [ ] `format/display/formatter.go` `Format()`: add case for new type
- [ ] If type has units → delegate to `FormatQuantity()` for unit normalization (don't bypass `NormalizeForDisplay`)
- [ ] `format/json_formatter.go` `populateResult()`: add case with type-specific fields
- [ ] JSON schema: add new fields to `JSONResult` struct (overflow-safe with `IsInt64()` checks)
- [ ] Display format must be ASCII-only for machine output
- [ ] Unicode display variant (optional): separate rendering function, config toggle

### Layer 9: App Integration

- [ ] `spec/features/registry.go`: register features for discoverability/autocomplete
- [ ] Config: add any new settings (e.g., `tui.unicode_fractions`)
- [ ] Config validation: add to `knownKeys` map
- [ ] Defaults: add to `defaults.toml`
- [ ] Docs: update `agent-integration.md` with syntax, JSON examples, result fields

### Regression Safety

- [ ] Golden test file in `testdata/eval/success/features/`
- [ ] Fuzzer corpus entries in both lexer and parser
- [ ] TUI catwalk test proving display correctness
- [ ] Export round-trip test (source → evaluate → export → re-parse)
- [ ] Markdown interaction test (SmartypantsFractions, interpolation safety)
- [ ] Full suite: `task test` zero failures, `task quality` clean

## Key Architectural Insights

### Unit preservation through arithmetic

When two values combine and one has a unit, the result must inherit the unit. The "first-unit-wins" heuristic fails for compound syntax like mixed numbers where the unit is on the second operand. Fix: fall back to the other operand's unit when the first has none.

### Display normalization must not be bypassed

New types with units must go through the same `NormalizeForDisplay` pipeline as Quantity. Directly calling `v.String()` in the formatter skips auto-scaling (pints → gallons). The fix: convert to Quantity and delegate to `FormatQuantity`.

### ASCII for machines, Unicode for humans

Machine-readable output (JSON, text, `.cm` export) must always use ASCII. Unicode display characters are TUI-only and must be gated behind a user-configurable toggle. Sentinel markers (`\x02`/`\x03`) protect interpolated values from markdown post-processing.

### Defense-in-depth for validation

The `Evaluate()` API can bypass semantic analysis. Any validation done in the semantic checker (e.g., zero denominator) must be independently repeated in the interpreter's `evalXxxLiteral()`.

## Prevention

When adding a new type in the future, copy this checklist and work through it layer by layer. The order matters: types → lexer → AST → parser → classifier/detector → semantic → interpreter → formatters → app integration → regression tests.

Run `task test` after completing each layer to catch regressions early rather than debugging a pile of failures at the end.
