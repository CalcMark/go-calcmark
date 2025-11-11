import { describe, it, expect } from 'vitest';
import {
  runeToUtf16Position,
  utf16ToRunePosition,
  substringRunes,
  countRunes
} from './unicode';

describe('Unicode position conversion', () => {
  describe('runeToUtf16Position', () => {
    it('handles ASCII characters (1 rune = 1 UTF-16 unit)', () => {
      const text = 'hello world';
      expect(runeToUtf16Position(text, 0)).toBe(0);
      expect(runeToUtf16Position(text, 5)).toBe(5);
      expect(runeToUtf16Position(text, 11)).toBe(11);
    });

    it('handles basic emoji (1 rune = 2 UTF-16 units)', () => {
      const text = '🏠 = $1_500';
      expect(runeToUtf16Position(text, 0)).toBe(0); // Start
      expect(runeToUtf16Position(text, 1)).toBe(2); // After 🏠
      expect(runeToUtf16Position(text, 2)).toBe(3); // After space
    });

    it('handles multiple emoji', () => {
      const text = '🏠🍕💡';
      expect(runeToUtf16Position(text, 0)).toBe(0);
      expect(runeToUtf16Position(text, 1)).toBe(2); // After 🏠
      expect(runeToUtf16Position(text, 2)).toBe(4); // After 🍕
      expect(runeToUtf16Position(text, 3)).toBe(6); // After 💡
    });

    it('handles emoji with skin tone modifiers', () => {
      const text = '👋🏽'; // Wave + skin tone = 2 runes, 4 UTF-16 units
      expect(runeToUtf16Position(text, 0)).toBe(0);
      expect(runeToUtf16Position(text, 1)).toBe(2); // After 👋
      expect(runeToUtf16Position(text, 2)).toBe(4); // After skin tone
    });

    it('handles Chinese characters (1 rune = 1 UTF-16 unit for BMP)', () => {
      const text = '工资 = $5000';
      expect(runeToUtf16Position(text, 0)).toBe(0);
      expect(runeToUtf16Position(text, 1)).toBe(1); // After 工
      expect(runeToUtf16Position(text, 2)).toBe(2); // After 资
    });

    it('handles mixed content', () => {
      const text = '🏠 工资 = $1_500';
      expect(runeToUtf16Position(text, 0)).toBe(0); // Start
      expect(runeToUtf16Position(text, 1)).toBe(2); // After 🏠
      expect(runeToUtf16Position(text, 2)).toBe(3); // After space
      expect(runeToUtf16Position(text, 3)).toBe(4); // After 工
      expect(runeToUtf16Position(text, 4)).toBe(5); // After 资
    });

    it('handles edge case: empty string', () => {
      expect(runeToUtf16Position('', 0)).toBe(0);
    });

    it('handles edge case: position beyond string', () => {
      const text = 'abc';
      expect(runeToUtf16Position(text, 10)).toBe(3); // Stops at end
    });
  });

  describe('utf16ToRunePosition', () => {
    it('handles ASCII characters', () => {
      const text = 'hello';
      expect(utf16ToRunePosition(text, 0)).toBe(0);
      expect(utf16ToRunePosition(text, 3)).toBe(3);
      expect(utf16ToRunePosition(text, 5)).toBe(5);
    });

    it('handles emoji (2 UTF-16 units = 1 rune)', () => {
      const text = '🏠 = test';
      expect(utf16ToRunePosition(text, 0)).toBe(0);
      expect(utf16ToRunePosition(text, 2)).toBe(1); // After 🏠
      expect(utf16ToRunePosition(text, 3)).toBe(2); // After space
    });

    it('handles multiple emoji', () => {
      const text = '🏠🍕💡';
      expect(utf16ToRunePosition(text, 0)).toBe(0);
      expect(utf16ToRunePosition(text, 2)).toBe(1); // After 🏠
      expect(utf16ToRunePosition(text, 4)).toBe(2); // After 🍕
      expect(utf16ToRunePosition(text, 6)).toBe(3); // After 💡
    });

    it('roundtrips with runeToUtf16Position', () => {
      const texts = [
        'hello',
        '🏠 = $1_500',
        '工资 = $5000',
        '🏠🍕💡',
        '👋🏽 test'
      ];

      for (const text of texts) {
        const runeCount = countRunes(text);
        for (let rune = 0; rune <= runeCount; rune++) {
          const utf16 = runeToUtf16Position(text, rune);
          const backToRune = utf16ToRunePosition(text, utf16);
          expect(backToRune).toBe(rune);
        }
      }
    });
  });

  describe('substringRunes', () => {
    it('extracts ASCII substring', () => {
      const text = 'hello world';
      expect(substringRunes(text, 0, 5)).toBe('hello');
      expect(substringRunes(text, 6, 11)).toBe('world');
    });

    it('extracts single emoji', () => {
      const text = '🏠 = $1_500';
      expect(substringRunes(text, 0, 1)).toBe('🏠');
    });

    it('extracts emoji sequence', () => {
      const text = '🏠🍕💡';
      expect(substringRunes(text, 0, 1)).toBe('🏠');
      expect(substringRunes(text, 1, 2)).toBe('🍕');
      expect(substringRunes(text, 2, 3)).toBe('💡');
      expect(substringRunes(text, 0, 3)).toBe('🏠🍕💡');
    });

    it('extracts Chinese characters', () => {
      const text = '工资 = $5000';
      expect(substringRunes(text, 0, 2)).toBe('工资');
    });

    it('extracts mixed content', () => {
      const text = '🏠 工资 = $1_500';
      expect(substringRunes(text, 0, 1)).toBe('🏠');
      expect(substringRunes(text, 2, 4)).toBe('工资');
    });

    it('handles emoji with skin tone', () => {
      const text = '👋🏽 test';
      expect(substringRunes(text, 0, 1)).toBe('👋');
      expect(substringRunes(text, 1, 2)).toBe('🏽'); // Skin tone modifier
      expect(substringRunes(text, 0, 2)).toBe('👋🏽');
    });
  });

  describe('countRunes', () => {
    it('counts ASCII characters', () => {
      expect(countRunes('hello')).toBe(5);
      expect(countRunes('hello world')).toBe(11);
    });

    it('counts emoji as single runes', () => {
      expect(countRunes('🏠')).toBe(1);
      expect(countRunes('🏠🍕💡')).toBe(3);
    });

    it('counts emoji with skin tone as separate runes', () => {
      expect(countRunes('👋🏽')).toBe(2); // Base + modifier
    });

    it('counts Chinese characters', () => {
      expect(countRunes('工资')).toBe(2);
    });

    it('counts mixed content', () => {
      expect(countRunes('🏠 工资')).toBe(4); // emoji, space, 2 Chinese
    });

    it('handles empty string', () => {
      expect(countRunes('')).toBe(0);
    });

    it('matches JavaScript string iteration', () => {
      const texts = ['hello', '🏠🍕', '工资', '👋🏽'];
      for (const text of texts) {
        let count = 0;
        for (const _ of text) count++;
        expect(countRunes(text)).toBe(count);
      }
    });
  });

  describe('Real-world CalcMark examples', () => {
    it('handles emoji variable assignment', () => {
      const line = '🏠 = $1_500';
      // Token positions from Go lexer (in runes):
      // 🏠: 0-1, =: 2-3, $1_500: 4-11

      expect(substringRunes(line, 0, 1)).toBe('🏠');
      expect(substringRunes(line, 4, 11)).toBe('$1_500');
    });

    it('handles Chinese variable assignment', () => {
      const line = '工资 = $5_000';
      // Token positions (in runes):
      // 工资: 0-2, =: 3-4, $5_000: 5-11

      expect(substringRunes(line, 0, 2)).toBe('工资');
      expect(substringRunes(line, 5, 11)).toBe('$5_000');
    });

    it('handles mixed emoji and operators', () => {
      const line = '🏠 + 🍕 + 💡';
      // Token positions (in runes):
      // 🏠: 0-1, +: 2-3, 🍕: 4-5, +: 6-7, 💡: 8-9

      expect(substringRunes(line, 0, 1)).toBe('🏠');
      expect(substringRunes(line, 2, 3)).toBe('+');
      expect(substringRunes(line, 4, 5)).toBe('🍕');
      expect(substringRunes(line, 8, 9)).toBe('💡');
    });

    it('extracts whitespace correctly between emoji tokens', () => {
      const line = '🏠 = $1_500';
      const utf16Start = runeToUtf16Position(line, 1); // After 🏠
      const utf16End = runeToUtf16Position(line, 2); // Before =
      const whitespace = line.substring(utf16Start, utf16End);
      expect(whitespace).toBe(' ');
    });
  });
});
