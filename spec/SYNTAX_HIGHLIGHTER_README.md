# CalcMark Syntax Highlighter Integration

## Overview

The `SYNTAX_HIGHLIGHTER_SPEC.json` file contains everything your TypeScript syntax highlighter needs to know about CalcMark tokens and keywords.

**No build tools or grammar parsers required** - it's just a JSON reference document.

## Quick Integration

```typescript
import spec from './SYNTAX_HIGHLIGHTER_SPEC.json';

// Get all reserved keywords (can't be used as variable names)
const reservedKeywords = [
  ...spec.tokens.keywords.logicalOperators.tokens,      // and, or, not
  ...spec.tokens.keywords.controlFlow.tokens,           // if, then, else, etc.
  ...Object.keys(spec.tokens.keywords.functions.canonical) // avg, sqrt
];

// Check if a word is reserved
function isReservedKeyword(word: string): boolean {
  return reservedKeywords.some(kw => kw.toLowerCase() === word.toLowerCase());
}

// Simple tokenizer example
function tokenize(line: string): Array<{type: string, value: string}> {
  const tokens = [];
  const words = line.split(/(\s+|[(),+\-*×/=<>!%^])/);

  for (const word of words) {
    if (!word || /^\s+$/.test(word)) continue;

    const lower = word.toLowerCase();

    // Check reserved keywords first
    if (reservedKeywords.includes(lower)) {
      tokens.push({ type: 'keyword', value: word });
    }
    // Check literals
    else if (/^\$/.test(word)) {
      tokens.push({ type: 'currency', value: word });
    }
    else if (/^\d/.test(word)) {
      tokens.push({ type: 'number', value: word });
    }
    else if (['true', 'false', 'yes', 'no', 't', 'f', 'y', 'n'].includes(lower)) {
      tokens.push({ type: 'boolean', value: word });
    }
    // Check operators
    else if (/^[+\-*×/=<>!%^]+$/.test(word)) {
      tokens.push({ type: 'operator', value: word });
    }
    // Check punctuation
    else if (/^[(),]$/.test(word)) {
      tokens.push({ type: 'punctuation', value: word });
    }
    // Must be identifier (variable name)
    else {
      tokens.push({ type: 'identifier', value: word });
    }
  }

  return tokens;
}
```

## Key Information in the Spec

### Reserved Keywords (case-insensitive)

```typescript
// Logical operators
spec.tokens.keywords.logicalOperators.tokens
// ["and", "or", "not"]

// Control flow (reserved for future, not yet implemented)
spec.tokens.keywords.controlFlow.tokens
// ["if", "then", "else", "elif", "end", "for", "in", "while", "return", "break", "continue", "let", "const"]

// Function names
Object.keys(spec.tokens.keywords.functions.canonical)
// ["avg", "sqrt"]
```

### Multi-Token Functions

```typescript
// These word sequences are combined into single tokens
spec.tokens.keywords.multiTokenFunctions.tokens
// [
//   { pattern: "average of", canonical: "avg", ... },
//   { pattern: "square root of", canonical: "sqrt", ... }
// ]
```

### Literal Patterns

```typescript
// Number pattern (supports thousand separators)
spec.tokens.literals.number.pattern
// "\d+(\.\d+)?"
spec.tokens.literals.number.thousandsSeparators
// [",", "_"]

// Currency pattern
spec.tokens.literals.currency.pattern
// "\$\d+(\.\d+)?"

// Boolean values
spec.tokens.literals.boolean.trueValues
// ["true", "yes", "t", "y"]
spec.tokens.literals.boolean.falseValues
// ["false", "no", "f", "n"]
```

### Identifier Rules

```typescript
// What can be an identifier?
spec.tokens.identifiers.allowedCharacters
// "Any Unicode character except whitespace and reserved operators"

spec.tokens.identifiers.notAllowed
// ["Spaces", "Reserved keywords", "Reserved operators"]

// BREAKING CHANGE: Spaces not allowed in identifiers
// Use underscores instead: my_budget (not "my budget")
```

## Example: Simple Syntax Highlighter

```typescript
interface HighlightedToken {
  type: 'keyword' | 'function' | 'identifier' | 'number' | 'currency' | 'boolean' | 'operator' | 'punctuation';
  value: string;
  cssClass: string;
}

function highlightCalcMark(code: string): HighlightedToken[] {
  const tokens: HighlightedToken[] = [];
  const lines = code.split('\n');

  for (const line of lines) {
    // Skip markdown lines
    if (/^[#>*\-\d]/.test(line.trim())) {
      tokens.push({ type: 'identifier', value: line, cssClass: 'markdown' });
      continue;
    }

    const lineTokens = tokenize(line);

    for (const token of lineTokens) {
      const cssClass = `calcmark-${token.type}`;
      tokens.push({ ...token, cssClass } as HighlightedToken);
    }
  }

  return tokens;
}
```

## CSS Classes

```css
.calcmark-keyword { color: #569cd6; font-weight: bold; }
.calcmark-function { color: #dcdcaa; }
.calcmark-number { color: #b5cea8; }
.calcmark-currency { color: #4ec9b0; }
.calcmark-boolean { color: #569cd6; }
.calcmark-operator { color: #d4d4d4; }
.calcmark-identifier { color: #9cdcfe; }
.calcmark-punctuation { color: #d4d4d4; }
.markdown { color: #6a9955; font-style: italic; }
```

## Breaking Changes

Check `spec.breakingChanges` for migration information between versions:

```typescript
const changes = spec.breakingChanges["v1.0.0"];
// [
//   {
//     change: "Spaces no longer allowed in identifiers",
//     before: "my budget = 1000",
//     after: "my_budget = 1000",
//     rationale: "Required for multi-token functions",
//     migration: "Replace spaces with underscores"
//   }
// ]
```

## That's It!

No grammar parsers, no build tools, no validation scripts. Just a simple JSON file with the language specification. Your TypeScript client can import it and use the information directly.
