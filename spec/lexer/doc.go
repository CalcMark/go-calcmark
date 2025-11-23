// Package lexer implements the CalcMark lexer/tokenizer.
//
// The lexer converts CalcMark source code into a stream of tokens
// that can be consumed by the parser. It handles Unicode-aware
// tokenization including international characters, emojis, and
// various number formats.
//
// # Architecture
//
// The lexer is implemented as a hand-written scanner (not generated)
// to provide precise control over token recognition and error messages.
// It uses a single-pass, character-by-character approach with lookahead.
//
// # Usage
//
// Basic tokenization:
//
//	lexer := lexer.NewLexer("x = 5 + 3")
//	tokens, err := lexer.Tokenize()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, token := range tokens {
//	    fmt.Printf("%s: %s\n", token.Type, token.Value)
//	}
//
// # Token Types
//
// The lexer recognizes:
//   - Numbers: integers, decimals, with thousands separators (1,000 or 1_000)
//   - Multipliers: k, M, B, T (e.g., "1.5M" → 1,500,000)
//   - Percentages: 20% → 0.20
//   - Currency: $, €, £, ¥ and ISO 4217 codes (USD, EUR, etc.)
//   - Identifiers: Unicode-aware including emoji
//   - Operators: +, -, *, /, %, ^, =, ==, !=, <, >, <=, >=
//   - Keywords: Reserved words and boolean values
//   - Functions: avg, sqrt and natural language variants
//
// # Number Formats
//
// The lexer supports flexible number formatting:
//
//	42          // Integer
//	3.14        // Decimal
//	1,000       // Thousands separator (comma)
//	1_000_000   // Thousands separator (underscore)
//	1.5M        // Multiplier (1,500,000)
//	20%         // Percentage (0.20)
//	1.2e6       // Scientific notation (1,200,000)
//
// # Currency Recognition
//
// Currency can be specified with symbols or ISO 4217 codes:
//
//	$100        // Symbol prefix
//	100 USD     // Code suffix (space required)
//	USD100      // Code prefix (no space)
//
// # Unicode Support
//
// Identifiers can contain:
//   - Letters from any Unicode script
//   - Digits (after first character)
//   - Underscores
//   - Emoji characters
//
// Examples:
//
//	salary      // ASCII
//	給料        // Japanese
//	café        // French with accents
//	💰_total    // Emoji + underscore
//
// # Reserved Keywords
//
// The following keywords are reserved and cannot be used as identifiers:
//   - Booleans: true, false, yes, no, t, f, y, n (case-insensitive)
//   - Functions: avg, sqrt
//   - Control flow: if, then, else, for, while, return (reserved for future)
//   - Logical: and, or, not (reserved for future)
//
// # Error Handling
//
// Lexer errors include position information:
//
//	Lexer error at line 1, column 5: unexpected character '!'
//
// # Performance
//
// The lexer is optimized for speed and typically completes in hundreds
// of nanoseconds:
//   - Currency token: ~600ns
//   - Number with multiplier: ~780ns
//   - Identifier: ~900ns
//
// See benchmark_test.go in the parser package for detailed metrics.
package lexer
