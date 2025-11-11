# Units System Design (Unified Quantity Approach)

**Status:** Design Document - Ready for Implementation
**Version:** 2.0 (Unified)

## Overview

This document outlines the design for a **unified Quantity type** that handles all numbers with units (currency, measurements, etc.) in a consistent, extensible way.

## Key Decision: Unify Currency and Measurements

**Previous approach**: Separate `Currency` (prefix, special) and `Measurement` (suffix) types
**New approach**: Single `Quantity` type that handles both

### Rationale

1. **Real-world currency notation varies**:
   - `$100` (prefix symbol, US)
   - `100 USD` (suffix code, ISO 4217)
   - `USD 100` (prefix code, financial)
   - `€50` / `50€` (both seen in Europe)
   - `100 GBP`, `£100`, `¥1000`, `1000円`

2. **Semantic similarity**: Currency and measurements are the same concept (number + unit)

3. **ISO 4217 standard**: Uses 3-letter codes (USD, EUR, GBP) typically as suffix

4. **Consistency**: One token format, one type, one set of rules

## Goals

1. **Library-agnostic grammar**: Lexer/parser define syntax, evaluator uses go-units
2. **Extensible**: Support 260+ units without hardcoding
3. **Flexible**: Support prefix ($100) and suffix (100 USD) notations
4. **User intent preservation**: SourceFormat captures exact user input
5. **Clear syntax**: Unambiguous parsing rules

## Type System

### Single Unified Type: Quantity

```go
type Quantity struct {
    Value        decimal.Decimal
    Unit         string  // "$", "USD", "cm", "Pa", "kg", etc.
    SourceFormat string  // Preserves user's exact notation
}
```

**Examples**:
```go
Quantity{Value: 100, Unit: "$", SourceFormat: "$100"}
Quantity{Value: 100, Unit: "USD", SourceFormat: "100 USD"}
Quantity{Value: 100, Unit: "USD", SourceFormat: "USD 100"}
Quantity{Value: 50, Unit: "€", SourceFormat: "50€"}
Quantity{Value: 100, Unit: "cm", SourceFormat: "100cm"}
Quantity{Value: 50.5, Unit: "Pa", SourceFormat: "50.5 Pa"}
```

### Token Type

```go
QUANTITY  // Unified token for all numbers with units
```

### Token Format (Always value:unit)

```
$100        → "100:$"
100 USD     → "100:USD"
USD 100     → "100:USD"
€50         → "50:€"
50€         → "50:€"
100cm       → "100:cm"
100 Pa      → "100:Pa"
1,234.5 kg  → "1234.5:kg"
```

**Consistency**: Always `"value:unit"` regardless of prefix/suffix in source.

## Grammar Rules

### Unit Adjacency Rule

**CRITICAL RULE**: Suffix units **REQUIRE** a space (U+0020) between number and unit.

1. **Prefix unit + number** (0-1 spaces allowed):
   ```
   $100        ✓ No space (recommended)
   $ 100       ✓ One space (allowed)
   $  100      ✗ Two spaces → MARKDOWN
   ```

2. **Number + suffix unit** (EXACTLY one space REQUIRED):
   ```
   100 cm      ✓ One space (REQUIRED)
   100cm       ✗ No space → Reserved for future magnitude suffixes (1.5k)
   100  cm     ✗ Two spaces → MARKDOWN
   ```

**Rationale**:
- Reserving no-space suffix (`100cm`) for future magnitude notation (`1.5k` = 1500)
- Prevents ambiguity: `1.5k` (1500) vs `1.5 K` (1.5 Kelvin)
- Prefix units don't need space because no magnitude suffix starts with $€£¥

### Number Formats Supported

All existing number formats work with units:
```
1 cm         ✓ Integer + space + unit
0.1 Pa       ✓ Decimal + space + unit
1,234 cm     ✓ Thousands separator + space + unit
1,234.5 kg   ✓ Thousands + decimal + space + unit
$5,000.00    ✓ Currency with formatting (prefix, no space required)
$ 5,000      ✓ Currency with space (allowed)
.5 cm        ✗ Must have leading zero (0.5 cm)
100cm        ✗ No space → INVALID (reserved for future)
```

### Multi-Character Units

Units can contain:
- **Letters**: `cm`, `Pa`, `USD`, `EUR`
- **Special chars**: `m/s`, `kg/m³`, `m²`
- **Case-sensitive**: `Pa` ≠ `pa`, `M` (mega) ≠ `m` (meter)

## Unit Registry

The lexer uses a registry to validate units (library-agnostic interface).

### Registry Interface

```go
// UnitRegistry validates unit symbols (used by lexer)
type UnitRegistry interface {
    IsValidUnit(symbol string) bool
    IsPrefix(symbol string) bool  // true for $€£¥
    IsSuffix(symbol string) bool  // true for USD, cm, Pa, etc.
    GetCanonicalSymbol(input string) string  // "meter" → "m"
}
```

### Prefix vs Suffix Units

**Prefix units** (appear before number):
```
$ € £ ¥ (currency symbols)
```

**Suffix units** (appear after number):
```
USD EUR GBP JPY (ISO 4217 codes)
cm m km Pa kPa kg g (measurements)
s min h d (time)
... 260+ more from go-units
```

### Dual-Mode Units

Some units support both:
```
€50         ✓ Prefix (symbol)
50 EUR      ✓ Suffix (code)
Both → Quantity{Value: 50, Unit: "EUR"}
```

### Registry Implementation (go-units)

```go
import units "github.com/bcicen/go-units"

var prefixUnits = map[string]bool{
    "$": true, "€": true, "£": true, "¥": true,
}

type GoUnitsRegistry struct {
    cache map[string]bool  // Performance optimization
}

func (r *GoUnitsRegistry) IsValidUnit(symbol string) bool {
    // Check prefix symbols
    if prefixUnits[symbol] {
        return true
    }

    // Check suffix units via go-units
    _, err := units.Find(symbol)
    return err == nil
}

func (r *GoUnitsRegistry) IsPrefix(symbol string) bool {
    return prefixUnits[symbol]
}

func (r *GoUnitsRegistry) IsSuffix(symbol string) bool {
    return !r.IsPrefix(symbol) && r.IsValidUnit(symbol)
}
```

### Supported Units (260+ via go-units)

**Currency** (ISO 4217): USD, EUR, GBP, JPY, CAD, AUD, CHF, CNY, INR, etc.
**Length**: m, cm, mm, km, in, ft, yd, mi, nm, μm
**Mass**: g, kg, mg, lb, oz, t (metric ton)
**Temperature**: C, F, K (Celsius, Fahrenheit, Kelvin)
**Time**: s, ms, μs, ns, min, h, d, wk, mo, yr
**Pressure**: Pa, kPa, MPa, bar, psi, atm, torr
**Volume**: L, mL, gal, qt, pt, cup, fl oz
**Energy**: J, kJ, MJ, cal, kcal, Wh, kWh
**Power**: W, kW, MW, hp
**Speed**: m/s, km/h, mph, knot
**Data**: B, KB, MB, GB, TB, bit, kbit, Mbit

## Lexer Algorithm

### Prefix Unit Lexing

```
Current char is a prefix unit symbol ($€£¥):
  1. Record symbol
  2. Advance past symbol
  3. Skip optional space (max 1)
  4. Read number (with thousands separators)
  5. Emit QUANTITY token: "value:unit"
  6. Store SourceFormat: "$100" or "$ 100"
```

### Suffix Unit Lexing

```
After reading a NUMBER token:
  1. Check current char:
     - Letter → potential suffix unit
     - Space + letter → potential suffix unit (max 1 space)
     - Otherwise → just NUMBER, continue

  2. Extract unit string:
     - Read consecutive: letters, digits, '/', '^', '·', '²', '³'
     - Stop at: space (after first), operator, punctuation, newline, EOF

  3. Validate unit via registry:
     - If valid → upgrade NUMBER to QUANTITY token
     - If invalid → backtrack, keep as NUMBER

  4. Store SourceFormat: "100cm" or "100 cm"
```

### Example Flows

**Input**: `$100`
```
1. See '$' → prefix unit
2. Read '100'
3. Emit QUANTITY "100:$"
4. SourceFormat "$100"
```

**Input**: `100 USD`
```
1. Read '100' → NUMBER
2. See space + 'U' → potential suffix
3. Read 'USD'
4. Registry: valid suffix unit
5. Upgrade to QUANTITY "100:USD"
6. SourceFormat "100 USD"
```

**Input**: `USD 100`
```
Problem: 'USD' at start → IDENTIFIER token
Cannot be recognized as prefix unit

**Limitation**: We only support single-char prefix units ($€£¥)
Multi-char prefix codes (USD 100) require different parsing strategy
Recommendation: Document as not supported in v1
```

**Input**: `100cm + 50 kg`
```
1. Read '100' → NUMBER
2. See 'c' → potential suffix
3. Read 'cm'
4. Registry: valid
5. Emit QUANTITY "100:cm"
6. PLUS "+"
7. Read '50' → NUMBER
8. See space + 'k' → potential suffix
9. Read 'kg'
10. Registry: valid
11. Emit QUANTITY "50:kg"
```

## Disambiguation Rules

### Rule 1: Unit After Number Only

```
100cm       → QUANTITY (cm follows number)
cm = 5      → IDENTIFIER (cm alone is variable)
x = 100cm   → Assignment with QUANTITY literal
y = cm * 2  → IDENTIFIER (cm as variable reference)
```

### Rule 2: Registry Lookup Required

```
100xyz      → NUMBER + IDENTIFIER (xyz not in registry)
100cm       → QUANTITY (cm in registry)
100 unknown → NUMBER + IDENTIFIER (unknown not in registry)
```

### Rule 3: Space Limit (Max 1)

```
100 cm      → QUANTITY (one space ok)
100  cm     → NUMBER + IDENTIFIER (two spaces)
$  100      → '$' + NUMBER (two spaces, no unit)
```

### Rule 4: Prefix Takes Precedence

```
$100        → QUANTITY "100:$" (prefix symbol)
100$        → NUMBER + ??? (suffix $ not supported)
€50         → QUANTITY "50:€"
50€         → QUANTITY "50:€" (if € registered as suffix too)
```

### Rule 5: Thousands Separator Context

```
1,234cm         → QUANTITY "1234:cm" (thousands in number)
avg(100cm, 50cm) → Function call (comma separates args)
```

## Operation Rules

### Same Units → Preserve

```
100cm + 50cm = 150cm
$100 + $50 = $150
10Pa * 2 = 20Pa
100kg / 2 = 50kg
```

### Same Dimension, Different Units → Convert

```
100cm + 1m = 200cm    (convert m→cm, use first unit)
1kg + 500g = 1.5kg    (convert g→kg)
$100 + $50 = $150     (same already)
100 USD + 50 USD = 150 USD
```

**Conversion rule**: Use first operand's unit as result unit.

### Different Dimensions → Error

```
100cm + 50Pa    → ERROR: Cannot add length and pressure
10kg * 2m       → ERROR: Cannot multiply mass and length
$100 + 50cm     → ERROR: Cannot mix currency and length
100 USD + 50 EUR → ERROR: Different currency units (no auto-conversion)
```

**Note**: Currency units are different dimensions (USD ≠ EUR), no auto-exchange rate.

### Unit Compatibility Table

| Op | Same Unit | Same Dim | Diff Dim | Number |
|----|-----------|----------|----------|--------|
| `+`| Preserve  | Convert  | Error    | Error  |
| `-`| Preserve  | Convert  | Error    | Error  |
| `*`| Error?    | Error    | Error    | Scale  |
| `/`| Ratio     | Convert  | Error    | Scale  |
| `^`| Error     | Error    | Error    | Error  |

**Special cases**:
- `100cm * 2` → `200cm` (scaling by number)
- `100cm / 2` → `50cm` (division by number)
- `100cm / 50cm` → `2` (ratio, dimensionless)
- `$100 * 0.1` → `$10` (percentage/scaling)

### Functions with Units

```
avg(100cm, 200cm, 150cm) → 150cm  (same unit preserved)
avg(100cm, 1m, 50cm)     → 100cm  (convert to first, preserve)
avg($100, $200)          → $150   (same unit)
avg(100 USD, 50 USD)     → 75 USD (same unit)
avg(100cm, 50Pa)         → ERROR  (different dimensions)
avg($100, 100 EUR)       → ERROR  (different currencies)
sqrt(100cm²)             → 10cm   (if cm² supported later)
```

## Testing Strategy (Extensive)

### Lexer Tests (Token Generation)

#### Prefix Units
```go
TestPrefixUnits:
- "$100"          → QUANTITY "100:$"
- "$ 100"         → QUANTITY "100:$"
- "$  100"        → CURRENCY + NUMBER (two spaces, invalid)
- "$1,000"        → QUANTITY "1000:$"
- "$1,000.50"     → QUANTITY "1000.50:$"
- "€50.99"        → QUANTITY "50.99:€"
- "£25"           → QUANTITY "25:£"
- "¥1000"         → QUANTITY "1000:¥"
```

#### Suffix Units - With Space (REQUIRED)
```go
TestSuffixUnitsWithSpace:
- "100 cm"        → QUANTITY "100:cm"
- "0.1 Pa"        → QUANTITY "0.1:Pa"
- "1,234 kg"      → QUANTITY "1234:kg"
- "100 USD"       → QUANTITY "100:USD"
- "50 EUR"        → QUANTITY "50:EUR"
- "1,234.5 kg"    → QUANTITY "1234.5:kg"
```

#### Suffix Units - No Space (INVALID - Reserved for Future)
```go
TestSuffixUnitsNoSpace:
- "100cm"         → NUMBER + IDENTIFIER (reserved for 1.5k notation)
- "0.1Pa"         → NUMBER + IDENTIFIER (invalid)
- "1.5kg"         → NUMBER + IDENTIFIER (reserved)
- "1,234cm"       → NUMBER + IDENTIFIER (invalid)
- "100USD"        → NUMBER + IDENTIFIER (invalid)
- "50EUR"         → NUMBER + IDENTIFIER (invalid)
```

#### Invalid Spacing
```go
TestInvalidSpacing:
- "100  cm"       → NUMBER + IDENTIFIER (two spaces)
- "$  100"        → CURRENCY + NUMBER
- "100   USD"     → NUMBER + IDENTIFIER
```

#### Unknown Units
```go
TestUnknownUnits:
- "100xyz"        → NUMBER + IDENTIFIER
- "100 unknown"   → NUMBER + IDENTIFIER
- "100zz"         → NUMBER + IDENTIFIER
```

#### Edge Cases
```go
TestEdgeCases:
- "cm"            → IDENTIFIER (unit alone)
- "cm = 5"        → IDENTIFIER + ASSIGN + NUMBER
- "100"           → NUMBER (no unit)
- "100 "          → NUMBER (trailing space, no unit follows)
- "m = 5; 100m"   → two statements (var + quantity)
```

### Parser Tests

```go
TestParseQuantity:
- "$100"          → QuantityLiteral{Value: 100, Unit: "$"}
- "100 USD"       → QuantityLiteral{Value: 100, Unit: "USD"}
- "100cm"         → QuantityLiteral{Value: 100, Unit: "cm"}
- "x = $100"      → Assignment
- "y = 100cm"     → Assignment
```

### Evaluator Tests (Operations)

#### Same Unit Operations
```go
TestSameUnitOps:
- "$100 + $50"    → $150
- "100cm + 50cm"  → 150cm
- "10Pa * 2"      → 20Pa (number scaling)
- "100kg / 2"     → 50kg (number scaling)
- "100cm / 50cm"  → 2 (ratio, dimensionless)
```

#### Unit Conversion
```go
TestUnitConversion:
- "100cm + 1m"    → 200cm (m→cm)
- "1m + 50cm"     → 1.5m (cm→m, first unit)
- "1kg + 500g"    → 1.5kg (g→kg)
- "500g + 1kg"    → 1500g (kg→g, first unit)
```

#### Mixed Dimensions (Errors)
```go
TestMixedDimensions:
- "100cm + 50Pa"  → ERROR
- "$100 + 50cm"   → ERROR
- "10kg * 2m"     → ERROR
- "100 USD + 50 EUR" → ERROR
```

#### Functions
```go
TestFunctions:
- "avg($100, $200)"      → $150
- "avg(100cm, 200cm)"    → 150cm
- "avg(100cm, 1m, 50cm)" → 100cm (convert all to cm)
- "avg($100, 100 EUR)"   → ERROR (different currencies)
- "sqrt(100)"            → 10 (no unit)
```

### Integration Tests

#### Complex Expressions
```go
TestComplexExprs:
- "distance = 100cm + 50cm; total = distance * 2"
  → distance = 150cm, total = 300cm

- "price = $100; tax = price * 0.1; total = price + tax"
  → price = $100, tax = $10, total = $110

- "length = 1m; width = 50cm; area = length * width"
  → ERROR (m * cm not supported)
```

#### Variable Shadowing
```go
TestVariableShadowing:
- "m = 5; x = m * 2"      → m=5 (variable), x=10
- "y = 100m"              → y=100m (quantity, meters)
- "z = m + 100m"          → ERROR (5 + 100m, incompatible)
```

#### Real-World Scenarios
```go
TestRealWorld:
- Budget calculator with multiple currencies
- Distance calculations with mixed units (cm, m, km)
- Mass calculations (g, kg, lb)
- Pressure calculations (Pa, kPa, bar)
```

### Test Coverage Goals

- ✅ All supported units (spot check 50+)
- ✅ All number formats (int, decimal, thousands)
- ✅ All spacing variations (none, one, two+)
- ✅ All operation types (+, -, *, /, ^)
- ✅ Function calls with units
- ✅ Error conditions (wrong dimension, invalid unit)
- ✅ Edge cases (variable shadowing, ambiguity)
- ✅ SourceFormat preservation

## Implementation Phases

### Phase 1: Foundation (CURRENT)
- ✅ Add QUANTITY token type
- ✅ Create UnitRegistry interface
- ✅ Implement basic registry (hardcoded prefix units)
- ⏸️ Update lexer for prefix units ($€£¥)

### Phase 2: Suffix Units (Space Required)
- ⏸️ Add suffix unit detection in lexer (REQUIRES space)
- ⏸️ Integrate go-units for validation
- ⏸️ Reject no-space suffixes (100cm → NUMBER + IDENTIFIER)
- ⏸️ Test with 20-30 common units (cm, kg, Pa, USD, EUR, etc.)

### Phase 3: Space Validation
- ⏸️ Ensure exactly one space for suffix units
- ⏸️ Reject multi-space (two+ spaces)
- ⏸️ Comprehensive edge case testing

### Phase 4: Quantity Type Implementation
- ⏸️ Add Quantity struct to types/types.go
- ⏸️ Implement operations (same unit, conversion, errors)
- ⏸️ Update parser to handle QUANTITY tokens
- ⏸️ Update evaluator for unit arithmetic

### Phase 5: Functions with Units
- ⏸️ Update function evaluation for units
- ⏸️ Implement unit preservation/conversion in avg()
- ⏸️ Implement unit preservation in sqrt()

### Phase 6: Advanced Units
- ⏸️ Support complex units (m/s, kg/m³)
- ⏸️ Support Unicode units (², ³)
- ⏸️ Unit name aliases ("meter" → "m")

## Limitations (v1)

### Not Supported

1. **Multi-char prefix codes**: `USD 100`, `EUR 50`
   - Reason: Requires lookahead/special parsing
   - Workaround: Use `100 USD`, `50 EUR` (suffix)

2. **Dual suffix symbols**: `100$`, `50€`
   - Reason: Ambiguous with identifier
   - Workaround: Use `$100`, `€50` (prefix) or `100 USD`, `50 EUR` (code)

3. **Auto currency conversion**: `$100 + €50`
   - Reason: Requires exchange rate data
   - Result: ERROR in v1

4. **Compound units**: `kg·m/s²`, `N` (newton)
   - Reason: Complex parsing and dimensional analysis
   - Future: Phase 6+

5. **Unit conversion syntax**: `100cm as m`
   - Reason: Adds language complexity
   - Future: Possible v2 feature

6. **Custom units**: User-defined units
   - Reason: Registry is fixed in v1
   - Future: Possible v2 feature

## Migration from Current Currency System

### Backward Compatibility

**Goal**: Existing CalcMark with $€£¥ continues to work identically.

**Changes required**:
1. Rename CURRENCY token → QUANTITY (internal only)
2. Change token format "$:100" → "100:$" (internal only)
3. Keep Currency type for now (evaluator wraps as Quantity)
4. Update WASM bridge to use new format

**User-facing**: No breaking changes for existing syntax.

### Deprecation Path

1. **v1.0**: Introduce QUANTITY, keep CURRENCY as alias
2. **v1.1**: Deprecate CURRENCY token type
3. **v2.0**: Remove CURRENCY, unify under QUANTITY

## Open Questions & Decisions

### Q1: Auto-conversion direction for mixed units
**Decision**: Use first operand's unit
```
100cm + 1m  → 200cm  (not 2m)
1m + 100cm  → 2m     (not 200cm)
```
**Rationale**: Predictable, left-to-right evaluation

### Q2: Currency auto-conversion?
**Decision**: ERROR in v1, possible future feature
```
$100 + €50  → ERROR (v1)
$100 + 50 EUR → ERROR (v1)
```
**Rationale**: Requires exchange rate API, out of scope

### Q3: Suffix currency symbols?
**Decision**: Not supported in v1
```
100$   → NUMBER + IDENTIFIER (not recognized)
50€    → Could add if € registered as suffix too
```
**Rationale**: Ambiguous parsing, use prefix or code

### Q4: Case sensitivity?
**Decision**: Case-sensitive
```
Pa  → pascals (valid)
pa  → invalid
M   → mega prefix (if supported)
m   → meter
```
**Rationale**: Scientific convention, prevents ambiguity

---

**Status**: Design approved, ready for Phase 1 implementation

**Next Steps**:
1. ✅ Update UNITS_DESIGN.md (this file)
2. ⏸️ Implement Phase 1 (prefix units refactor)
3. ⏸️ Write extensive lexer tests (50+ test cases)
4. ⏸️ Implement Phase 2 (suffix units)
5. ⏸️ Write evaluator tests (30+ test cases)
6. ⏸️ Update LANGUAGE_SPEC.md
7. ⏸️ Update syntax highlighter spec
