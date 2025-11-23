# CalcMark Encoding Specification

## Overview

CalcMark follows Unicode-based encoding principles similar to CommonMark, with explicit rules for character handling, identifier formation, and normalization.

## 1. Input Encoding

### UTF-8 by Default
- **REQUIRED**: All CalcMark documents MUST be valid UTF-8
- Implementations MUST reject invalid UTF-8 byte sequences
- Byte Order Mark (BOM) U+FEFF at the start of a document MUST be stripped

### Character Processing
- All text processing operates on **Unicode code points** (not bytes)
- Characters are represented as runes (Unicode code points) internally
- In Go: `[]rune(text)` converts UTF-8 bytes to Unicode code points

## 2. Whitespace Characters

Whitespace is defined by specific Unicode code points:

- **Space**: U+0020 (ASCII space)
- **Tab**: U+0009 (horizontal tab)
- **Carriage Return**: U+000D (CR)
- **Line Feed**: U+000A (LF)

### Line Endings
Line breaks are recognized as:
- **LF**: U+000A (Unix-style)
- **CR**: U+000D (old Mac-style)
- **CRLF**: U+000D U+000A (Windows-style)

Implementations MUST handle all three line ending styles.

## 3. Identifier Rules

Identifiers (variable names) follow Unicode-aware rules:

### Identifier Start Characters
An identifier MUST start with:
- **Unicode Letter** (category L): any letter from any language
  - Examples: `a-z`, `A-Z`, `café`, `給料`, `Москва`
- **Underscore**: U+005F (`_`)
- **Emoji**: Unicode emoji characters (category So, Emoji_Presentation)
  - Examples: `💰`, `🎯`, `📊`

### Identifier Continue Characters
After the first character, identifiers MAY contain:
- All **identifier start characters** (letters, `_`, emoji)
- **Unicode Digits** (category Nd): `0-9` and digits from other scripts
  - Examples: `０-９` (fullwidth), `٠-٩` (Arabic-Indic)
- **Combining Marks** (category M): accents and diacritical marks
  - Examples: combining acute accent U+0301, combining tilde U+0303

### Excluded Characters
Identifiers MUST NOT contain:
- Whitespace (space, tab, CR, LF)
- Reserved operators: `+`, `-`, `*`, `×`, `/`, `=`, `$`, `>`, `<`, `!`, `%`, `^`, `(`, `)`, `,`
- Digits at the start position

### Implementation in Go

```go
func isIdentifierStart(char rune) bool {
    return unicode.IsLetter(char) ||
           char == '_' ||
           isEmoji(char)
}

func isIdentifierContinue(char rune) bool {
    return unicode.IsLetter(char) ||
           unicode.IsDigit(char) ||
           unicode.IsMark(char) ||  // Combining marks
           char == '_' ||
           isEmoji(char)
}
```

## 4. Unicode Normalization

### NO Normalization Required
CalcMark **DOES NOT** require Unicode normalization. This means:

- Different Unicode representations are treated as **distinct identifiers**
- Example (these are DIFFERENT variables):
  ```
  café = 100   # U+00E9 (é as single code point)
  café = 200   # U+0065 U+0301 (e + combining acute accent)
  ```

### Rationale
- Follows CommonMark precedent
- Simplifies implementation (no normalization step required)
- Preserves author's exact input
- Avoids ambiguity about which normalization form (NFC, NFD, NFKC, NFKD)

### Implementation Implications
- Identifiers are compared **byte-for-byte** after UTF-8 decoding
- No normalization happens during lexing, parsing, or evaluation
- Visual similarity ≠ semantic equality

## 5. Case Sensitivity

### Identifiers
- **Case-sensitive**: `Total` ≠ `total` ≠ `TOTAL`
- Unicode case folding is NOT applied
- Examples:
  ```
  Москва ≠ москва  (Cyrillic capital vs lowercase)
  Café ≠ café
  ```

### Keywords and Functions
- **Case-insensitive**: `TRUE` = `True` = `true`
- Boolean keywords: `true`, `false`, `yes`, `no`, `t`, `f`
- Functions: `avg`, `sqrt`
- Implemented via `strings.ToLower()` comparison

## 6. Number Literals

### Digit Recognition
- **ASCII digits only**: U+0030-U+0039 (`0-9`)
- Other Unicode digit categories (Nd) are NOT recognized as numbers
- Example: `১২৩` (Bengali digits) is an identifier, not a number

### Decimal Separator
- **Period only**: U+002E (`.`)
- Other Unicode decimal separators are NOT recognized
- Example: `3,14` is invalid (comma is thousands separator)

### Thousands Separators
- **Comma**: U+002C (`,`)
- **Underscore**: U+005F (`_`)
- Must appear in groups of exactly 3 digits
- Examples: `1,000`, `1_000_000`

## 7. Operator Characters

All operators use ASCII characters only:

| Operator | Code Point | Character |
|----------|------------|-----------|
| Plus | U+002B | `+` |
| Minus | U+002D | `-` |
| Multiply | U+002A | `*` |
| Multiply (alt) | U+00D7 | `×` |
| Divide | U+002F | `/` |
| Modulus | U+0025 | `%` |
| Exponent | U+005E | `^` |
| Assign | U+003D | `=` |
| Greater | U+003E | `>` |
| Less | U+003C | `<` |
| Not | U+0021 | `!` |

### Unicode Alternatives
- **Multiplication sign** U+00D7 (`×`) is equivalent to `*`
- Other Unicode mathematical operators are NOT recognized

## 8. Currency and Units

### 8.1 Currency Symbols and Codes

CalcMark supports two types of currency indicators:

#### Currency Symbols (Prefix Only)

The following Unicode currency symbols are recognized:

| Symbol | Code Point | Name |
|--------|------------|------|
| $ | U+0024 | Dollar sign |
| € | U+20AC | Euro sign |
| £ | U+00A3 | Pound sign |
| ¥ | U+00A5 | Yen/Yuan sign |

These symbols are **prefix units only** (no space required): `$100`, `€50`, `£1000`, `¥500`

#### ISO 4217 Currency Codes

CalcMark supports **all valid ISO 4217 currency codes** as currency identifiers:

**Format Requirements**:
- **Exactly 3 letters**
- **MUST be uppercase** (e.g., `USD`, `GBP`, `EUR`)
- Validated against ISO 4217 standard using `golang.org/x/text/currency`
- Currency codes are **RESERVED WORDS** in CalcMark calculations

**Usage Formats**:

1. **Prefix format** (NO space required):
   ```calcmark
   GBP100      ✅ Valid - 100 British Pounds
   USD1,000    ✅ Valid - 1000 US Dollars
   EUR50.25    ✅ Valid - 50.25 Euros
   JPY500      ✅ Valid - 500 Japanese Yen
   ```

2. **Postfix format** (SPACE REQUIRED):
   ```calcmark
   100 USD     ✅ Valid - 100 US Dollars
   50 GBP      ✅ Valid - 50 British Pounds
   1000 EUR    ✅ Valid - 1000 Euros
   ```

**Invalid Usage**:
```calcmark
usd100      ❌ Lowercase not allowed
Gbp50       ❌ Mixed case not allowed
XYZ100      ⚠️  XYZ is not a valid ISO 4217 code - treated as identifier
GBP         ❌ Standalone currency code cannot be used as variable name
```

**Reserved Word Status**:
- All valid ISO 4217 currency codes are reserved words
- Cannot be used as variable names in calculations
- Can appear in markdown blocks
- Invalid codes (e.g., `XYZ`) are treated as regular identifiers

### 8.2 Unit Spacing Rules

CalcMark enforces **strict spacing rules** for quantities with units:

#### Prefix Units (Currency) - NO SPACE REQUIRED
Currency symbols are **prefix units** and do NOT require a space:

```calcmark
$100          ✅ VALID (no space)
$ 100         ❌ INVALID (space not allowed)
€50           ✅ VALID
£1000         ✅ VALID
```

**Rationale**: Currency symbols traditionally appear directly before the number (`$100`, not `$ 100`).

#### Postfix Units - SPACE REQUIRED
Non-currency units are **postfix units** and REQUIRE exactly one space (U+0020):

```calcmark
10 cm         ✅ VALID (single space)
10cm          ❌ INVALID (no space)
10  cm        ❌ INVALID (multiple spaces)
10	cm        ❌ INVALID (tab, not space)
```

**Rationale**:
- Prevents ambiguity: `10cm` could be a variable name
- Follows SI unit conventions
- Tab character (U+0009) is NOT equivalent to space for this purpose

#### Implementation Notes
- Space check uses **exact Unicode code point** U+0020
- `strings.TrimLeft()` uses `constants.Whitespace = " \t\r\n"`
- Unit detection checks `runes[i] == ' '` (not any whitespace)

### 8.3 Quantity Examples

```calcmark
# Currency Symbols (Prefix)
$100           ✅ Prefix currency symbol
€50            ✅ Prefix currency symbol
£1,000         ✅ Prefix currency symbol with thousands separator

# Currency Codes (Prefix)
USD100         ✅ Prefix currency code
GBP1,000       ✅ Prefix currency code with thousands separator
EUR50.25       ✅ Prefix currency code with decimal

# Currency Codes (Postfix)
100 USD        ✅ Postfix currency code (SPACE REQUIRED)
50 GBP         ✅ Postfix currency code
1000 EUR       ✅ Postfix currency code

# Measurement Units (Postfix)
10 cm          ✅ Postfix measurement unit (SPACE REQUIRED)
50 kg          ✅ Postfix measurement unit
100 mph        ✅ Postfix compound unit

# Invalid Examples
$100 USD       ❌ Mixed prefix/postfix
10km           ❌ Missing required space
10 	cm        ❌ Tab instead of space
usd100         ❌ Lowercase currency code
100USD         ❌ No space with postfix code
```

## 9. Implementation Checklist

Conforming implementations MUST:

- [x] Accept UTF-8 encoded input
- [x] Strip BOM if present
- [x] Process text as Unicode code points (not bytes)
- [x] Handle all three line ending styles (LF, CR, CRLF)
- [x] Support Unicode letters in identifiers
- [x] Support emoji in identifiers
- [x] Support combining marks in identifiers (after first char)
- [x] Reject invalid UTF-8 sequences
- [ ] Document Unicode version supported (recommend Unicode 15.0+)
- [x] NOT normalize Unicode (preserve exact code points)
- [x] Use case-insensitive comparison for keywords only
- [x] Use ASCII digits only for number literals

## 10. Test Cases

### Valid Identifiers
```calcmark
café = 100          # Latin with accent
給料 = $5000        # Japanese
💰 = 1000           # Emoji
total_2024 = 42     # ASCII with underscore and digit
Москва = 100        # Cyrillic
```

### Invalid Identifiers
```calcmark
2fast = 100         # Cannot start with digit
my total = 50       # Space not allowed
cost+tax = 200      # Operator not allowed
```

### Unicode Normalization (both valid but distinct)
```calcmark
café = 100          # U+00E9 (precomposed é)
café = 200          # U+0065 U+0301 (e + combining acute)
# These are DIFFERENT variables!
```

### Case Sensitivity
```calcmark
Total = 100         # Variable
total = 200         # Different variable
TRUE = true         # Keyword (case-insensitive), cannot assign
```

## 11. Test Requirements

### Required Test Coverage

Implementations MUST include tests for:

1. **UTF-8 Validation**
   - Valid UTF-8 sequences
   - Invalid UTF-8 byte sequences (MUST reject)
   - BOM handling (MUST strip U+FEFF at start)

2. **Identifier Support**
   - ASCII baseline (`income`, `_private`, `total_2024`)
   - Latin extended (`café`, `naïve`, `résumé`)
   - Non-Latin scripts (Cyrillic `Москва`, CJK `給料`, Arabic `الدخل`, Hebrew)
   - Emoji (`💰`, `🎯`, `📊`)
   - Invalid identifiers (starting with digit, containing operators, spaces)

3. **Unicode Normalization** (verify NO normalization)
   - `café` (U+00E9) ≠ `café` (U+0065 U+0301) - MUST be different variables
   - Same visual appearance, different code points = different identifiers

4. **Case Sensitivity**
   - Variables: `Total` ≠ `total` ≠ `TOTAL`
   - Keywords: `TRUE` = `true` = `True`

5. **Line Endings**
   - LF only (Unix)
   - CRLF (Windows)
   - CR only (old Mac)
   - Mixed line endings

6. **Currency and Units**
   - All supported currency symbols (`$`, `€`, `£`, `¥`)
   - ISO 4217 codes (uppercase only, valid codes)
   - Spacing rules (prefix: no space, postfix: required space)

7. **Number Literals**
   - ASCII digits only (reject other Unicode digits like Bengali `১২৩`)
   - Decimal separator (period only)
   - Thousands separators (comma, underscore)
   - Percentage literals

## 12. WASM Implications

### JavaScript String Encoding
- JavaScript uses UTF-16 internally
- WASM boundary: Convert UTF-16 (JavaScript) ↔ UTF-8 (CalcMark)
- Use `TextEncoder` / `TextDecoder` for conversion

### Example WASM Interface
```javascript
// JavaScript side
const input = "給料 = $5000";  // UTF-16 string
const utf8Bytes = new TextEncoder().encode(input);
const result = wasmEvaluate(utf8Bytes);
const output = new TextDecoder().decode(result);
```

## 13. References

- **Unicode Standard**: https://www.unicode.org/versions/latest/
- **CommonMark Spec**: https://spec.commonmark.org/ (Appendix on Unicode)
- **UTF-8**: RFC 3629
- **Go unicode package**: https://pkg.go.dev/unicode

## Version

This specification is version **1.0.0-draft** and applies to CalcMark 0.1.x.
