# Phase 9: Interpreter Correctness - Research

**Researched:** 2026-02-06
**Domain:** Interpreter bug fixes - type preservation, unit conversion, function forms
**Confidence:** HIGH

## Summary

This phase addresses calculation bugs in the go-calcmark interpreter, focusing on the critical napkin conversion type erasure bug and auditing all conversion paths. The research confirms:

1. **The napkin bug is precisely located**: `impl/interpreter/napkin_eval.go` lines 24-29 extract only the numeric value, and line 99 returns `types.NewNumber()` instead of preserving the input type. The fix requires type-aware return logic.

2. **Type system is well-defined**: The codebase has clear types (`*types.Quantity`, `*types.Rate`, `*types.Duration`, `*types.Currency`, `*types.Number`) in `spec/types/`. Each type preserves its unit/metadata through operations.

3. **Natural language forms are limited**: The lexer defines only `FUNC_AVERAGE_OF` and `FUNC_SQUARE_ROOT_OF`. The parser maps these to `avg` and `sqrt`. Testing should cover these specific forms, not imagined others.

4. **Unit conversion uses martinlindhe/unit library**: The `impl/interpreter/unit_library.go` provides category-based conversion through a registry. Roundtrip accuracy depends on float64 precision in conversions.

**Primary recommendation:** Fix napkin_eval.go to return the same type as input (with rounded value), then audit all switch-on-type patterns for similar type erasure.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/shopspring/decimal` | current | Arbitrary precision arithmetic | Already used throughout codebase |
| `github.com/martinlindhe/unit` | current | Unit conversion factors | Already used in unit_library.go |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `testing` | stdlib | Test framework | All test files |
| `github.com/CalcMark/go-calcmark/spec/types` | internal | Type definitions | Type-aware returns |
| `github.com/CalcMark/go-calcmark/spec/parser` | internal | Parse test inputs | Integration tests |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| N/A | N/A | Internal codebase fixes - no new libraries needed |

**Installation:**
No new dependencies required.

## Architecture Patterns

### Existing Interpreter Structure
```
impl/interpreter/
├── interpreter.go       # Main Interpreter struct, evalNode dispatcher
├── operators.go         # Binary/unary operation evaluation
├── functions.go         # FunctionDef registry, function dispatch
├── rate_functions.go    # accumulate, convertRateTimeUnit
├── rate_eval.go         # RateLiteral evaluation
├── unit_conversion.go   # convertQuantity, evalQuantityOperation
├── unit_conversion_eval.go  # evalUnitConversion (explicit "in" syntax)
├── napkin_eval.go       # evalNapkinConversion (BUG LOCATION)
├── napkin.go            # formatNapkin display formatter
├── unit_library.go      # Unit registry with category-based conversion
└── literals.go          # Literal evaluation (numbers, currencies, etc.)
```

### Pattern 1: Type-Preserving Switch
**What:** When processing typed values, return the same type structure with modified value.
**When to use:** Any operation that should preserve type metadata (unit, currency symbol, rate unit).
**Example:**
```go
// Source: Current pattern in operators.go lines 342-351
// CORRECT: evalUnaryOperation preserves type
if qty, ok := operand.(*types.Quantity); ok {
    switch operator {
    case "-":
        return &types.Quantity{Value: qty.Value.Neg(), Unit: qty.Unit}, nil
    case "+":
        return qty, nil
    }
}
```

### Pattern 2: Type-Erasing Switch (ANTI-PATTERN)
**What:** Extract numeric value from typed value, return plain Number.
**When to use:** NEVER for operations that should preserve type.
**Example:**
```go
// Source: napkin_eval.go lines 24-29, 99 (THE BUG)
// WRONG: This erases type information
case *types.Quantity:
    numValue = v.Value  // Extracts only the number, loses unit
// ...
return types.NewNumber(decimal.NewFromFloat(roundedFloat)), nil  // Returns Number, not Quantity
```

### Anti-Patterns to Avoid
- **Type Erasure in Conversions:** Never return `*types.Number` when input was `*types.Quantity`, `*types.Rate`, etc. The napkin bug is exactly this anti-pattern.
- **Incomplete Type Switches:** Every type switch must handle ALL relevant types or explicitly error. Missing cases lead to silent failures.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Unit conversion factors | Custom conversion math | `unit_library.go` registry | Already handles all unit categories consistently |
| Time unit normalization | String matching | `types.NormalizeTimeUnit()` | Handles all aliases (s, sec, second, seconds) |
| Decimal precision | float64 arithmetic | `shopspring/decimal` | Prevents floating point errors |
| Function lookup | Custom name matching | `BuiltinFunctions` slice | Single source of truth for all functions |

**Key insight:** The codebase already has proper abstractions. The bugs are in specific implementations that don't use these abstractions correctly.

## Common Pitfalls

### Pitfall 1: Type Erasure in Switch Statements
**What goes wrong:** A switch extracts a numeric value from a typed value, then returns a plain Number.
**Why it happens:** Developer focuses on the calculation, forgets to reconstruct the original type.
**How to avoid:** Template pattern - always return same type as input:
```go
func processTypedValue(input types.Type, transform func(decimal.Decimal) decimal.Decimal) (types.Type, error) {
    switch v := input.(type) {
    case *types.Quantity:
        return types.NewQuantity(transform(v.Value), v.Unit), nil
    case *types.Rate:
        return types.NewRate(&types.Quantity{Value: transform(v.Amount.Value), Unit: v.Amount.Unit}, v.PerUnit), nil
    // ... handle all types
    }
}
```
**Warning signs:** Any function returning `*types.Number` when it could receive `*types.Quantity`.

### Pitfall 2: Float64 Precision Loss in Roundtrips
**What goes wrong:** Converting meters -> feet -> meters doesn't return exact original value.
**Why it happens:** The unit_library.go uses `float64` for conversion factors.
**How to avoid:** Accept tolerance in tests, use decimal.Decimal where possible.
**Warning signs:** Tests comparing exact equality on unit conversion roundtrips.

### Pitfall 3: Missing Natural Language Form Tests
**What goes wrong:** Assuming more natural language forms exist than are actually implemented.
**Why it happens:** The lexer only defines `FUNC_AVERAGE_OF` and `FUNC_SQUARE_ROOT_OF`.
**How to avoid:** Test only what the lexer actually tokenizes (see spec/lexer/token.go lines 93-94).
**Warning signs:** Tests for "sum of", "accumulate of" - these don't exist in the grammar.

### Pitfall 4: Napkin Auto-Normalization Edge Cases
**What goes wrong:** "Human-friendly" unit scaling may be unexpected (422000 MB -> 400 GB).
**Why it happens:** Napkin is for quick estimates, not precise values.
**How to avoid:** Define clear rules for when to scale up (e.g., value > 1000 of current unit).
**Warning signs:** User confusion about why units changed.

## Code Examples

Verified patterns from the codebase:

### Type-Preserving Napkin Conversion (PROPOSED FIX)
```go
// Source: Proposed fix for napkin_eval.go
func (interp *Interpreter) evalNapkinConversion(n *ast.NapkinConversion) (types.Type, error) {
    value, err := interp.evalNode(n.Expression)
    if err != nil {
        return nil, err
    }

    switch v := value.(type) {
    case *types.Number:
        rounded := roundToNapkinPrecision(v.Value)
        return types.NewNumber(rounded), nil
    case *types.Quantity:
        rounded := roundToNapkinPrecision(v.Value)
        normalizedUnit, normalizedValue := autoNormalize(rounded, v.Unit)
        return types.NewQuantity(normalizedValue, normalizedUnit), nil
    case *types.Rate:
        rounded := roundToNapkinPrecision(v.Amount.Value)
        normalizedUnit, normalizedValue := autoNormalize(rounded, v.Amount.Unit)
        return types.NewRate(&types.Quantity{Value: normalizedValue, Unit: normalizedUnit}, v.PerUnit), nil
    case *types.Duration:
        rounded := roundToNapkinPrecision(v.Value)
        return &types.Duration{Value: rounded, Unit: v.Unit}, nil
    case *types.Currency:
        rounded := roundToNapkinPrecision(v.Value)
        return types.NewCurrency(rounded, v.Symbol), nil
    default:
        return nil, fmt.Errorf("napkin conversion requires a numeric value, got %T", value)
    }
}
```

### Accumulate Function (CORRECT - Reference)
```go
// Source: impl/interpreter/rate_functions.go lines 15-42
// This correctly returns *types.Quantity preserving the rate's unit
func accumulateRate(rate *types.Rate, timePeriod decimal.Decimal, periodUnit string) (*types.Quantity, error) {
    // ... calculation ...
    return &types.Quantity{
        Value: totalAmount,
        Unit:  rate.Amount.Unit,  // Preserves unit from rate
    }, nil
}
```

### Unit Conversion Roundtrip Test Pattern
```go
// Source: Pattern from impl/interpreter/unit_conversion_test.go
func TestUnitConversionRoundtrip(t *testing.T) {
    tests := []struct {
        name      string
        value     float64
        unit1     string
        unit2     string
        tolerance float64
    }{
        {"meters-feet-meters", 100.0, "meters", "feet", 0.0001},
        {"kg-pounds-kg", 50.0, "kg", "lb", 0.0001},
    }
    for _, tt := range tests {
        // Convert unit1 -> unit2 -> unit1
        // Assert result within tolerance of original
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Return Number from napkin | Return same type as input | Phase 9 fix | Correct `accumulate() as napkin` behavior |
| Ad-hoc type switches | Consistent type-preserving pattern | Phase 9 | Prevents future type erasure bugs |

**Deprecated/outdated:**
- N/A - no deprecated patterns in this domain

## Open Questions

Things that need clarification during implementation:

1. **Auto-normalization rules for napkin**
   - What we know: User wants "422000 MB -> 400 GB"
   - What's unclear: Exact threshold for when to normalize up (>1000? >500?)
   - Recommendation: Use 1000 as threshold (K/M/G/T boundaries)

2. **Error message format**
   - What we know: Include line number, show signature for function errors
   - What's unclear: Exact format strings and how line numbers are passed through
   - Recommendation: Investigate AST Range field usage for line numbers

3. **Natural language function coverage**
   - What we know: Only "average of" and "square root of" exist in lexer
   - What's unclear: Are there plans for more? (e.g., "sum of")
   - Recommendation: Test only what exists, don't implement new forms

## Sources

### Primary (HIGH confidence)
- `impl/interpreter/napkin_eval.go` - Direct inspection of bug location
- `impl/interpreter/rate_functions.go` - Reference for correct type-preserving pattern
- `spec/types/` directory - Authoritative type definitions
- `impl/interpreter/unit_library.go` - Unit conversion registry
- `spec/lexer/token.go` lines 93-94 - Natural language function tokens

### Secondary (MEDIUM confidence)
- `impl/interpreter/unit_conversion_test.go` - Test patterns for roundtrip accuracy
- `impl/interpreter/functions_test.go` - Test patterns for function evaluation
- `testdata/eval/success/features/napkin.cm` - Golden examples for napkin

### Tertiary (LOW confidence)
- N/A - all findings verified in codebase

## Metadata

**Confidence breakdown:**
- Napkin bug location: HIGH - Direct code inspection confirms exact lines
- Type preservation pattern: HIGH - Multiple examples in codebase
- Natural language forms: HIGH - Lexer token definitions are authoritative
- Unit conversion accuracy: MEDIUM - Depends on float64 precision in library

**Research date:** 2026-02-06
**Valid until:** 60 days (stable codebase, internal bug fixes)
