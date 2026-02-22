# Plain Language Function Support for Remaining CalcMark Functions

**Date:** 2026-02-22
**Status:** Brainstorm complete
**Scope:** Add natural language syntax for `read`, `compress`, and `transfer_time`; add per-alias `Parseable` flag to feature registry; remove stale `spec/parser/tokens.go`

## What We're Building

Three CalcMark functions (`read`, `compress`, `transfer_time`) currently only support parenthesized function-call syntax. We're adding plain language alternatives so users can write more readable calculations:

| Function | Current syntax | New plain language syntax |
|---|---|---|
| `read` | `read(100 MB, ssd)` | `read 100 MB from ssd` |
| `compress` | `compress(1 GB, gzip)` | `compress 1 GB using gzip` |
| `transfer_time` | `transfer_time(1 GB, regional, gigabit)` | `transfer 1 GB across regional gigabit` |

Three other functions (`rtt`, `throughput`, `seek`) were evaluated and deliberately excluded because their function-call syntax is already concise and readable.

Additionally:
- Add a per-alias `Parseable` field to the feature registry so docs/help can distinguish between aliases that work as input syntax vs aliases that are search-only
- Remove the stale `spec/parser/tokens.go` file (dead code, never imported)

## Why This Approach

### Pattern A: Lexer Multi-Token Combination

All three new plain language forms use **Pattern A** (the same pattern as `average of` and `square root of`):

1. **Lexer** post-processes token stream to combine multi-word sequences into single tokens
2. **Parser** matches combined tokens in `parsePrimary()` and delegates to `parseNaturalLanguageFunction()`
3. **NL function handler** maps to canonical function names and builds standard `ast.FunctionCall` nodes

This was chosen over Pattern B (contextual parser keywords like `over`, `at`, `per`) because all three functions are **prefix-style** — the function keyword comes first, not between arguments.

### Keyword Choices

- **`from`** for read: Natural preposition for data source ("read from"). Already a reserved keyword in the lexer (used for date expressions like "2 days from today"). The combination only triggers when the lexer sees `IDENTIFIER("read")` followed by `FROM`, which mirrors how `average` + `of` works today.
- **`using`** for compress: New keyword. Chosen over `with` to avoid ambiguity with the capacity planning `with N% buffer` pattern.
- **`across`** for transfer: New keyword. Chosen because `over` is taken by `accumulate` and `across` implies network traversal.

### Identifier Collision Risk

`read`, `compress`, and `transfer` are NOT reserved keywords — they remain plain identifiers. The multi-token combiner will match `IDENTIFIER("read") + FROM` regardless of whether the user intended `read` as a variable name. This is the same trade-off that `average` has (it combines with `of`), and is acceptable because these words are unlikely variable names in a calculation context. If a user does need a variable called `read`, they can still use it in other positions (just not immediately before `from`).

### Transfer_time Argument Parsing

`transfer 1 GB across regional gigabit` uses **positional** parsing after `across`: the parser takes exactly two identifiers (scope, then network type). No separator keyword between them. This keeps the syntax simple and avoids adding yet another keyword.

## Key Decisions

1. **Scope: 3 of 6 functions** — Only `read`, `compress`, `transfer_time` get plain language forms. `rtt`, `throughput`, `seek` are already readable enough.

2. **Implementation pattern: Lexer multi-token combination (Pattern A)** — Consistent with `average of` and `square root of`. Lexer combines tokens, parser dispatches through `nl_functions.go`.

3. **New lexer token types needed:**
   - `FUNC_READ_FROM` — combines `IDENTIFIER("read")` + `FROM`
   - `FUNC_COMPRESS_USING` — combines `IDENTIFIER("compress")` + new `USING` keyword
   - `FUNC_TRANSFER_ACROSS` — combines `IDENTIFIER("transfer")` + new `ACROSS` keyword

4. **New reserved keywords:** `using`, `across` added to `ReservedKeywords` map in lexer.

5. **Keyword `with` avoided** for compress — `using` prevents ambiguity with capacity planning's `with N% buffer`.

6. **Positional args for transfer_time** — `transfer 1 GB across regional gigabit` parses two identifiers positionally after `across`. No separator keyword.

7. **Per-alias Parseable flag** — Change `Feature.Aliases` from `[]string` to `[]Alias` where `Alias` has `Name string` and `Parseable bool`. Enables accurate documentation generation. **Note:** This is a breaking API change to the `features` package. All callers that iterate over `Feature.Aliases` will need updating (registry tests, any help/autocomplete consumers). Since the `features` package is in `spec/`, this is a spec-level API change.

8. **Remove stale spec/parser/tokens.go** — Dead code that defines an unused parallel `TokenType` enum. The real tokens live in `spec/lexer/token.go`.

9. **Registry sync** — Update registry aliases to include new NL forms with `Parseable: true`. Mark existing unparseable aliases (like "round trip time") with `Parseable: false`.

## Changes by Layer

### spec/lexer/token.go
- Add token types: `FUNC_READ_FROM`, `FUNC_COMPRESS_USING`, `FUNC_TRANSFER_ACROSS`
- Add keyword tokens: `USING`, `ACROSS`

### spec/lexer/lexer.go
- Add `"using"` and `"across"` to `ReservedKeywords` map
- Extend `combineMultiTokenFunctions()` to combine:
  - `IDENTIFIER("read")` + `FROM` -> `FUNC_READ_FROM`
  - `IDENTIFIER("compress")` + `USING` -> `FUNC_COMPRESS_USING`
  - `IDENTIFIER("transfer")` + `ACROSS` -> `FUNC_TRANSFER_ACROSS`
- Add `"read"`, `"compress"`, `"transfer"` to `isNaturalSyntaxKeyword()` to prevent them from being consumed as unit names

### spec/parser/nl_functions.go
- Extend `parseNaturalLanguageFunction()` to handle:
  - `FUNC_READ_FROM` -> `ast.FunctionCall{Name: "read", Args: [size_expr, identifier]}` — parse expression + `from` already consumed + identifier
  - `FUNC_COMPRESS_USING` -> `ast.FunctionCall{Name: "compress", Args: [size_expr, identifier]}` — parse expression + identifier
  - `FUNC_TRANSFER_ACROSS` -> `ast.FunctionCall{Name: "transfer_time", Args: [size_expr, scope_ident, network_ident]}` — parse expression + two identifiers

### spec/parser/rdparser.go
- Add `lexer.FUNC_READ_FROM`, `lexer.FUNC_COMPRESS_USING`, `lexer.FUNC_TRANSFER_ACROSS` to the match in `parsePrimary()` that dispatches to `parseNaturalLanguageFunction()`

### spec/features/registry.go
- Change `Aliases []string` to `Aliases []Alias`
- Define `type Alias struct { Name string; Parseable bool }`
- Update all existing alias entries
- Add new NL form aliases with `Parseable: true`

### spec/parser/tokens.go
- Delete this file (dead code)

### Tests
- Lexer tests for new token combinations (case sensitivity, whitespace handling)
- Parser tests for new NL function parsing
- Interpreter integration tests (NL forms produce same results as function-call forms)
- Golden testdata files for new syntax
- Registry tests for Parseable field

## Out of Scope

- Plain language for `rtt`, `throughput`, `seek` (already readable)
- Standalone single-word aliases like `average` without `of` (would require parser changes for single-word function detection)
- Changes to the interpreter (it already dispatches on canonical function names)

## Resolved Questions

- **Which functions?** read, compress, transfer_time. Not rtt, throughput, seek.
- **Which pattern?** Lexer multi-token combination (Pattern A).
- **Keyword for compress?** `using` (not `with`, to avoid capacity planning ambiguity).
- **Keyword for transfer_time?** `across` (not `over`, which is taken by accumulate).
- **Transfer_time arg separation?** Positional (2 identifiers after `across`).
- **Registry approach?** Per-alias `Parseable` flag for accurate documentation.
- **Stale tokens.go?** Clean it up now.
