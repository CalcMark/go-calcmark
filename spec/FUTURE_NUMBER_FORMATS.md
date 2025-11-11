# Future Number Format Support

**Status**: Planned (Phase 2/3)
**Dependencies**: Phase 1 (thousands separator preservation) must be complete

## Overview

This document outlines planned support for additional human-readable number formats beyond basic thousands separators.

## Phase 2: Magnitude Suffixes

### Formats to Support

| Suffix | Value | Example | Parsed As |
|--------|-------|---------|-----------|
| k, K | thousands | `1k`, `1.5k` | 1000, 1500 |
| M | millions | `2M`, `1.5M` | 2000000, 1500000 |
| B | billions | `3B`, `2.5B` | 3000000000, 2500000000 |
| T | trillions | `1T`, `1.5T` | 1000000000000, 1500000000000 |

### Implementation Requirements

1. **Lexer Changes**
   - Extend `readNumber()` to recognize magnitude suffixes
   - Store original format (e.g., "1.5M") in `SourceFormat`
   - Parse and normalize value (1.5M → 1500000)

2. **Formatting Computed Values** ⚠️ **Design Decision Needed**
   - Should `avg(1M, 2M)` display as `1.5M` or `1500000`?
   - Options:
     - **A**: Never format computed values (always canonical: `1500000`)
     - **B**: Smart formatting (use M if result > 1M, k if > 1k)
     - **C**: Use format of first argument (`1M + 2k` → result in M)
     - **D**: User-configurable preference

   **Recommendation**: Start with Option A (no formatting), add smart formatting in Phase 2.1

3. **Case Sensitivity**
   - `k` vs `K`: Both accepted as thousands
   - `M`, `B`, `T`: Only uppercase (lowercase conflicts with units)

### Potential Conflicts

| Suffix | Conflict | Resolution |
|--------|----------|------------|
| K | Kelvin temperature | Accept both k/K for thousands. Units TBD. |
| M | Meters (length) | Magnitude takes precedence. `5m` for meters requires space or unit system. |
| E | Mathematical constant | Constant `E` already defined. Scientific notation deferred to Phase 3. |

## Phase 3: Scientific Notation

### Formats to Support

- `1e3`, `1E3` = 1000
- `1.5e6`, `1.5E6` = 1500000
- `2.5e-3` = 0.0025

### Implementation Requirements

1. **Conflict Resolution with Constant E**
   - Context-aware parsing:
     - `E` alone → mathematical constant (2.718...)
     - `E` after digits → exponent separator
   - Examples:
     - `E` → 2.718281828459045
     - `1E3` → 1000
     - `E * 2` → 5.43656...

2. **Lexer State Machine**
   - Track whether we're in "number context"
   - If following digits + optional decimal, treat E as exponent
   - Otherwise, treat E as identifier (constant)

3. **Edge Cases**
   ```
   1E3      → 1000 (scientific)
   E        → 2.718... (constant)
   1 E 3    → 1 * E * 3 (expression)
   1E+3     → 1000 (positive exponent)
   1E-3     → 0.001 (negative exponent)
   ```

## Additional Considerations

### International Number Formats
- Space separator: `1 000` (European)
- Period separator: `1.000` (European - conflicts with decimal!)
- **Decision**: Defer international formats until we have locale system

### Binary/Hex Prefixes (Low Priority)
- `0x` prefix for hexadecimal
- `0b` prefix for binary
- **Decision**: Not needed for CalcMark's use case (financial/general calculations)

### Percentage Literals
- Already supported via `%` operator
- `50%` works as `50 / 100`
- No changes needed

## Testing Strategy

When implementing:
1. Comprehensive lexer tests for all magnitude suffixes
2. Round-trip tests: parse → evaluate → format
3. Edge cases: `1kk` (invalid), `1.5.5M` (invalid)
4. Conflict tests: `E`, `1E3`, `E3` disambiguation
5. Integration tests with functions: `avg(1k, 2k, 3k)`

## Open Questions

1. **Formatting Strategy**: How should computed values be formatted?
2. **Precedence**: Should `2k + 500` be allowed? How to display result?
3. **Mixing Formats**: `1k + 1,000` → display as what?
4. **Currency with Suffixes**: Should `$1M` be supported? (Yes, probably)

## References

- Phase 1 Implementation: `SourceFormat` field in Number/Currency types
- Related: [SYNTAX_SPEC.md](./SYNTAX_SPEC.md) - Grammar specification
- Related: [DIAGNOSTIC_LEVELS.md](./DIAGNOSTIC_LEVELS.md) - Error handling
