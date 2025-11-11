# Unicode Utilities

This module provides utilities for bridging Go's rune-based string positions with JavaScript's UTF-16 code unit based string operations.

## Problem

When integrating with Go backends (especially WASM), position information is reported in **runes** (Unicode code points), but JavaScript strings use **UTF-16 code units** for indexing. This causes issues with:

- Emoji (e.g., 🏠 is 1 rune but 2 UTF-16 code units)
- Emoji with skin tone modifiers (e.g., 👋🏽 is 2 runes but 4 UTF-16 code units)
- Some CJK characters outside the Basic Multilingual Plane
- Other multi-byte Unicode characters

## Solution

This module provides conversion functions to translate between rune positions and UTF-16 positions, enabling correct substring extraction and position tracking.

## API

### `runeToUtf16Position(text: string, runePos: number): number`

Converts a rune position (from Go) to a UTF-16 code unit position (for JavaScript).

```typescript
const text = "🏠 = $1_500";
const utf16Pos = runeToUtf16Position(text, 1); // Position after 🏠
// utf16Pos = 2 (because 🏠 is 2 UTF-16 code units)
```

### `utf16ToRunePosition(text: string, utf16Pos: number): number`

Converts a UTF-16 code unit position to a rune position.

```typescript
const text = "🏠 = $1_500";
const runePos = utf16ToRunePosition(text, 2); // After 🏠 (which is 2 UTF-16 units)
// runePos = 1 (because 🏠 is 1 rune)
```

### `substringRunes(text: string, runeStart: number, runeEnd: number): string`

Extracts a substring using rune-based positions.

```typescript
const text = "🏠 = $1_500";
const token = substringRunes(text, 0, 1); // Extract first rune
// token = "🏠"
```

### `countRunes(text: string): number`

Counts the number of runes in a string.

```typescript
countRunes("🏠") // 1
countRunes("hello") // 5
countRunes("👋🏽") // 2 (base emoji + skin tone modifier)
```

## Usage Example

```typescript
import { runeToUtf16Position, substringRunes } from '$lib/utils/unicode';

// Token from Go lexer has rune-based positions
const token = { start: 0, end: 1, type: 'IDENTIFIER' };
const lineText = "🏠 = $1_500";

// Convert to UTF-16 positions for JavaScript substring
const utf16Start = runeToUtf16Position(lineText, token.start);
const utf16End = runeToUtf16Position(lineText, token.end);
const tokenText = lineText.substring(utf16Start, utf16End);
// tokenText = "🏠"

// Or use substringRunes directly
const tokenText2 = substringRunes(lineText, token.start, token.end);
// tokenText2 = "🏠"
```

## Testing

Run the test suite with:

```bash
npm test
```

The test suite includes 29 comprehensive tests covering:
- ASCII characters
- Basic emoji
- Emoji with skin tone modifiers
- Chinese characters
- Mixed content
- Edge cases
- Real-world CalcMark examples
