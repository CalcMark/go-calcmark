/**
 * Unicode position conversion utilities.
 *
 * Bridges the gap between Go's rune-based positions (Unicode code points)
 * and JavaScript's UTF-16 code unit based string operations.
 *
 * Background:
 * - Go lexer reports token positions in RUNES (Unicode code points)
 * - JavaScript strings use UTF-16 code units for indexing
 * - Many Unicode characters (emoji, Chinese, etc.) require multiple UTF-16 code units
 *
 * Examples:
 * - ASCII 'a': 1 rune = 1 UTF-16 code unit
 * - Emoji 🏠: 1 rune = 2 UTF-16 code units (surrogate pair)
 * - Emoji with skin tone 👋🏽: 2 runes = 4 UTF-16 code units
 */

/**
 * Converts a rune position (from Go) to a UTF-16 code unit position (for JavaScript).
 *
 * @param text - The text string to measure
 * @param runePos - The position in runes (0-indexed)
 * @returns The corresponding position in UTF-16 code units
 *
 * @example
 * const text = "🏠 = $1_500";
 * const utf16Pos = runeToUtf16Position(text, 1); // Position after 🏠
 * // utf16Pos = 2 (because 🏠 is 2 UTF-16 code units)
 *
 * @example
 * const text = "工资 = $5000";
 * const utf16Pos = runeToUtf16Position(text, 2); // Position after "工资"
 * // utf16Pos = 2 (Chinese characters are 1 UTF-16 code unit each)
 */
export function runeToUtf16Position(text: string, runePos: number): number {
  let utf16Pos = 0;
  let currentRune = 0;

  // Iterate over string using for...of which handles surrogate pairs correctly
  for (const char of text) {
    if (currentRune >= runePos) break;
    utf16Pos += char.length; // char.length is 1 for BMP, 2 for surrogate pairs
    currentRune++;
  }

  return utf16Pos;
}

/**
 * Converts a UTF-16 code unit position to a rune position.
 *
 * @param text - The text string to measure
 * @param utf16Pos - The position in UTF-16 code units (0-indexed)
 * @returns The corresponding position in runes
 *
 * @example
 * const text = "🏠 = $1_500";
 * const runePos = utf16ToRunePosition(text, 2); // After 🏠 (which is 2 UTF-16 units)
 * // runePos = 1 (because 🏠 is 1 rune)
 */
export function utf16ToRunePosition(text: string, utf16Pos: number): number {
  let currentUtf16 = 0;
  let runePos = 0;

  for (const char of text) {
    if (currentUtf16 >= utf16Pos) break;
    currentUtf16 += char.length;
    runePos++;
  }

  return runePos;
}

/**
 * Extracts a substring using rune-based positions.
 *
 * @param text - The text string to extract from
 * @param runeStart - Start position in runes (0-indexed)
 * @param runeEnd - End position in runes (exclusive)
 * @returns The extracted substring
 *
 * @example
 * const text = "🏠 = $1_500";
 * const token = substringRunes(text, 0, 1); // Extract first rune
 * // token = "🏠"
 */
export function substringRunes(text: string, runeStart: number, runeEnd: number): string {
  const utf16Start = runeToUtf16Position(text, runeStart);
  const utf16End = runeToUtf16Position(text, runeEnd);
  return text.substring(utf16Start, utf16End);
}

/**
 * Counts the number of runes in a string.
 *
 * @param text - The text to measure
 * @returns The number of Unicode code points (runes)
 *
 * @example
 * countRunes("🏠") // 1
 * countRunes("hello") // 5
 * countRunes("👋🏽") // 2 (base emoji + skin tone modifier)
 */
export function countRunes(text: string): number {
  let count = 0;
  for (const _ of text) {
    count++;
  }
  return count;
}
