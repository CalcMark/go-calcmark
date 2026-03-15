---
title: "feat: support decimal numbers without leading zero"
type: feat
status: completed
date: 2026-03-15
github_issue: 66
---

# feat: support decimal numbers without leading zero

Today, `.5 tomatoes` fails because the lexer requires a leading digit before the decimal point. After this change, `.5` will be a valid number literal equivalent to `0.5`.

## Problem Statement

The lexer's `Tokenize()` dispatch loop (lexer.go:839) gates number reading on `unicode.IsDigit(char)`. When `.` is the current character, no case matches and it falls through to "Unexpected character '.'" at line 1122. The language spec (LANGUAGE_SPEC.md) explicitly marks `.5` as invalid.

## Proposed Solution

Add a leading-dot check in the lexer dispatch loop before the unknown-character fallback. When `.` is followed by a digit, route to `readNumber()`. The existing `readNumber()` already handles an empty integer part gracefully — the integer-reading loop breaks immediately, then the decimal-point check at line 227 consumes `.` + digits.

No parser or interpreter changes are needed. The lexer produces standard NUMBER tokens (and variants like NUMBER_PERCENT, NUMBER_K, NUMBER_SCI) which downstream code already handles. `decimal.NewFromString(".5")` works in Go.

## Technical Considerations

### Architecture — Lexer Only

The change is confined to `spec/lexer/lexer.go`. The token type remains NUMBER (or NUMBER_PERCENT, NUMBER_K, etc.). Parser, semantic checker, and interpreter see no difference from `0.5`.

### Key Interactions Verified

| Concern | Status | Notes |
|---------|--------|-------|
| Fraction ambiguity (`.5/2`) | Safe | `tryParseFraction` (fraction.go:20-24) already rejects numerators containing `.` |
| `@globals.field` DOT tokens | Safe | DOT is only emitted inside the `@` handler (lexer.go:1067-1107), which is a separate dispatch path |
| Ordered lists (`1. Item`) | Safe | Ordered lists require digits *before* the dot; `.5` has none |
| Currency (`$.5`) | Safe | CURRENCY_SYM is emitted first, then the next loop iteration sees `.5` |
| Line classification | Safe | `isNumberToken()` in detector.go checks token type, not string value |
| Unit attachment (`.5 tomatoes`, `.5kg`) | Safe | `readNumber()` handles units after the number is read |
| Unary operators (`-.5`, `+.5`) | Safe | MINUS/PLUS emitted first, `.5` tokenized on next iteration |

### Duration Lookahead Gap

The duration literal lookahead (lexer.go:839-856) is gated on `unicode.IsDigit(char)`. This means `.5 days` would bypass the duration path and produce QUANTITY(`.5:days`) instead of DURATION_LITERAL. This differs from `0.5 days` which correctly produces DURATION_LITERAL.

**Recommendation:** Extend the duration lookahead to also trigger on `.` followed by a digit. This keeps `.5 days` equivalent to `0.5 days`.

### Value Normalization

The token's `Value` field will store `".5"` (no leading zero), matching the source text. `OriginalText` will also be `".5"`. This is consistent with how the lexer preserves user input — `decimal.NewFromString()` handles both forms. Display formatting normalizes through the types layer, not the lexer.

## Acceptance Criteria

- [x] `.5` produces a NUMBER token equivalent to `0.5`
- [x] `.5 tomatoes` and `.5kg` produce QUANTITY tokens
- [x] `.5k`, `.5M`, `.5B`, `.5T` produce NUMBER_K/M/B/T tokens
- [x] `.5%` produces NUMBER_PERCENT token
- [x] `.5e3` produces NUMBER_SCI token
- [x] `-.5` and `+.5` work (unary operators)
- [x] `$.5`, `€.5`, `£.5` work (currency + leading-dot)
- [x] `a = .5 tomatoes` works in assignment
- [x] `.5/second over 1 day` works in NL growth functions
- [x] `.5/2` is division (not fraction) — already handled by fraction guard
- [x] `.5 days` produces QUANTITY (matching `0.5 days` behavior — duration lookahead only catches integers)
- [x] `.` alone remains an error
- [x] `..5` remains an error
- [x] Update `spec/LANGUAGE_SPEC.md` to mark `.5` as valid
- [x] All existing tests pass (`task test`)

## Implementation Phases

### Phase 1: Tests First (TDD)

Add golden test files:
- `testdata/spec/valid/features/leading_dot_decimals.cm` — all valid cases
- `testdata/spec/invalid/leading_dot_edge_cases.cm` — `.`, `..5` error cases (if not already covered)
- `testdata/eval/success/features/leading_dot_decimals.cm` — evaluation correctness
- Lexer unit tests in `spec/lexer/lexer_test.go` for token types and values

### Phase 2: Lexer Change

1. **Leading-dot dispatch** (lexer.go, before the digit check at line 839): Add a new case for `.` followed by a digit. The key subtlety is the duration lookahead — `readNumberString()` only reads digits, so it won't advance past `.`. The lookahead must manually skip past the dot and decimal digits:

   ```go
   if char == '.' && unicode.IsDigit(l.peek(1)) {
       // Duration lookahead: peek past dot + digits + whitespace to check for time unit
       savedPos := l.pos
       l.advance() // skip '.'
       _ = l.readNumberString() // read digits after dot
       l.skipWhitespace()
       if _, ok := l.tryReadTimeUnit(); ok {
           l.pos = savedPos
           tokens = append(tokens, l.readDurationLiteral())
           continue
       }
       l.pos = savedPos
       tokens = append(tokens, l.readNumber())
       continue
   }
   ```

2. **`readDurationLiteral()`**: Verify it handles an empty integer part before the decimal point. If it calls `readNumberString()` internally, it will also need to handle the leading dot (advance past `.` before reading digits). Trace the call to confirm.

### Phase 3: Spec Update

Update `spec/LANGUAGE_SPEC.md` number literals table: change `.5` from `Must have leading zero` to valid.

### Phase 4: Validate

- `task test` — full suite
- `task quality` — lint and vet
- Verify no regressions in `@globals`, ordered lists, fractions, or existing number parsing

## Sources

- GitHub issue: #66
- Lexer dispatch: `spec/lexer/lexer.go:839`
- Number reading: `spec/lexer/lexer.go:154-300`
- Fraction guard: `spec/lexer/fraction.go:14-25`
- Duration lookahead: `spec/lexer/lexer.go:839-856`
- Language spec: `spec/LANGUAGE_SPEC.md`
- Cross-layer checklist: `docs/solutions/logic-errors/adding-new-type-fraction-cross-layer-checklist.md`
