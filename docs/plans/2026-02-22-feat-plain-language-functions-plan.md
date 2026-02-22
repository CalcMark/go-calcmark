---
title: "feat: Add plain language syntax for read, compress, and transfer_time"
type: feat
status: completed
date: 2026-02-22
brainstorm: docs/brainstorms/2026-02-22-plain-language-functions-brainstorm.md
---

# feat: Add plain language syntax for read, compress, and transfer_time

## Overview

Add natural language alternatives for three CalcMark functions so users can write more readable calculations:

| Current syntax | New plain language syntax |
|---|---|
| `read(100 MB, ssd)` | `read 100 MB from ssd` |
| `compress(1 GB, gzip)` | `compress 1 GB using gzip` |
| `transfer_time(1 GB, regional, gigabit)` | `transfer 1 GB across regional gigabit` |

Additionally: add per-alias `Parseable` flag to feature registry for accurate documentation generation, and remove the stale `spec/parser/tokens.go` file.

## Problem Statement / Motivation

CalcMark's design philosophy blends markdown readability with calculations. Functions like `avg()` already support plain language (`average of 1, 2, 3`), but 6 infrastructure functions only support parenthesized syntax. Three of these (`read`, `compress`, `transfer_time`) benefit significantly from plain language forms because they describe real-world operations ("read 100 MB from ssd" reads like English).

The feature registry also advertises aliases (like "round trip time" for `rtt`) that the parser cannot actually handle, creating a documentation honesty gap.

## Proposed Solution

### Architecture Correction from Brainstorm

The brainstorm proposed using **Pattern A (lexer multi-token combination)** for all three functions. However, SpecFlow analysis revealed this pattern **does not fit**: the existing lexer combiner matches *adjacent* tokens (`average` + `of`), but the new patterns have a QUANTITY token between the function name and keyword (`read` + `100 MB` + `from`).

**Corrected approach: Parser lookahead (Pattern B).** The NL detection happens in the parser's `parsePrimary()` at the IDENTIFIER branch, using lookahead to check for the pattern `IDENTIFIER("read"|"compress"|"transfer") + QUANTITY + FROM|"using"|"across"`. This follows the established pattern for `downtime` detection (line 479 of rdparser.go).

### Contextual Keywords (Not Reserved)

The brainstorm proposed making `using` and `across` reserved keywords. SpecFlow analysis flagged this as a backwards compatibility risk — any existing document using `using` or `across` as variable names would break.

**Corrected approach:** `using` and `across` are **contextual keywords** (like `downtime`), NOT reserved. They remain `IDENTIFIER` tokens in the lexer. The parser checks their value contextually. This means `using = 5` continues to work.

`from` is already a reserved keyword (`FROM` token) — no change needed there.

## Technical Approach

### Phase 1: Dead Code Cleanup + Registry Refactor

Safe, non-behavioral changes first. Establishes the `Alias` type before NL syntax work begins.

#### 1a. Remove stale `spec/parser/tokens.go`

- Delete `spec/parser/tokens.go` (verified unused — no imports of `parser.TokenType` anywhere)
- Run `go build ./...` to confirm no compile errors

**Files:** `spec/parser/tokens.go` (delete)

#### 1b. Add `Alias` struct to feature registry

- Define `type Alias struct { Name string; Parseable bool }` in `spec/features/registry.go`
- Change `Feature.Aliases` from `[]string` to `[]Alias`
- Update `Feature.Match()` to use `alias.Name` instead of `alias`
- Update all `getFunctions()`, `getUnits()`, `getDateFeatures()`, `getNetworkFeatures()`, `getStorageFeatures()`, `getCompressionFeatures()`, `getKeywords()`, `getOperators()` alias entries
- For `getUnits()`: adapt the loop that reads from `units.StandardUnits[].Aliases` (which is `[]string`) — convert each to `Alias{Name: x, Parseable: false}`
- Mark existing parseable aliases (NL forms that the parser already supports):
  - `avg`: `{Name: "average of", Parseable: true}`, `{Name: "average", Parseable: false}`
  - `sqrt`: `{Name: "square root of", Parseable: true}`
  - `rtt`: `{Name: "round trip time", Parseable: false}` (search-only)
  - `requires`: `{Name: "capacity", Parseable: false}` (search-only — the "at...per" syntax maps to `capacity`, not `requires`)
- All other existing aliases: `Parseable: false`
- Note: `accumulate` ("over"), `convert_rate` ("per"), `downtime` ("downtime per"), and `capacity` ("at...per") have NL support but through infix keywords, not aliases — these don't need `Parseable` alias entries since the NL syntax uses different grammar, not an alternate function name
- Update `spec/features/registry_test.go`

**Files:**
- `spec/features/registry.go` (edit)
- `spec/features/registry_test.go` (edit)

#### 1c. Update REPL consumer

- Update `cmd/calcmark/tui/repl/model.go` if it directly accesses `Feature.Aliases` entries as strings (the only external consumer of the `features` package)

**Files:** `cmd/calcmark/tui/repl/model.go` (check + edit if needed)

**Phase 1 gate:** `task test` passes, `task quality` passes.

---

### Phase 2: Parser NL Detection Infrastructure (TDD)

Write tests first, then implement the parser lookahead pattern.

#### 2a. Add `isNLFunctionKeyword()` helper

Add a helper to `spec/parser/rdparser.go` that checks if an identifier name is an NL function trigger:

```go
// spec/parser/rdparser.go
func isNLFunctionKeyword(name string) bool {
    switch strings.ToLower(name) {
    case "read", "compress", "transfer":
        return true
    }
    return false
}
```

Add `"read"`, `"compress"`, `"transfer"` to `isNaturalSyntaxKeyword()` (line 1204) to prevent them from being consumed as unit names after numbers. (This prevents `5 read` from becoming a quantity.)

**Files:** `spec/parser/rdparser.go` (edit)

#### 2b. Write parser tests (TDD — tests first, all should fail)

Create `spec/parser/nl_read_compress_transfer_test.go` with test cases:

**NL syntax produces correct AST:**
- `read 100 MB from ssd` → `FunctionCall{Name: "read", Args: [Quantity{100, MB}, Identifier{ssd}]}`
- `compress 1 GB using gzip` → `FunctionCall{Name: "compress", Args: [Quantity{1, GB}, Identifier{gzip}]}`
- `transfer 1 GB across regional gigabit` → `FunctionCall{Name: "transfer_time", Args: [Quantity{1, GB}, Identifier{regional}, Identifier{gigabit}]}`

**Case insensitivity:**
- `Read 100 MB From SSD` → same as lowercase
- `COMPRESS 1 GB USING GZIP` → same as lowercase

**Backwards compatibility (must still work):**
- `read(100 MB, ssd)` → standard function call (unchanged)
- `compress(1 GB, gzip)` → standard function call (unchanged)
- `transfer_time(1 GB, regional, gigabit)` → standard function call (unchanged)
- `read = 42` → assignment (read as variable)
- `using = 5` → assignment (using as variable, NOT reserved)
- `across = 10` → assignment (across as variable, NOT reserved)

**Error cases:**
- `read 100 MB` (no `from`) → parsed as variable `read` + standalone quantity (two statements on one line = error)
- `compress 1 GB using` (no algorithm) → error: expected identifier after `using`
- `transfer 1 GB across regional` (only one identifier) → error: expected network type
- `transfer 1 GB across` (no identifiers) → error: expected scope after `across`

**Operator composition:**
- `read 100 MB from ssd + read 200 MB from nvme` → BinaryOp{+, FunctionCall{read...}, FunctionCall{read...}}
- `compress 1 GB using gzip * 3` → BinaryOp{*, FunctionCall{compress...}, 3}

**Files:** `spec/parser/nl_read_compress_transfer_test.go` (new)

#### 2c. Implement parser lookahead in `parsePrimary()`

In `parsePrimary()`, at the IDENTIFIER branch (line ~1060), add lookahead **before** the existing `parseFunctionCall()` check:

```go
// spec/parser/rdparser.go — inside parsePrimary(), at the IDENTIFIER match
if p.match(lexer.IDENTIFIER) {
    name := p.previous()
    identName := strings.ToLower(string(name.Value))

    // NL function lookahead: "read <qty> from <ident>"
    if identName == "read" && p.check(lexer.QUANTITY) {
        // Save position for backtracking
        // Peek ahead: QUANTITY + FROM?
        if p.peekAhead(1).Type == lexer.FROM {
            return p.parseNLReadFunction()
        }
    }

    // NL function lookahead: "compress <qty> using <ident>"
    if identName == "compress" && p.check(lexer.QUANTITY) {
        if p.peekAhead(1).Type == lexer.IDENTIFIER && strings.ToLower(string(p.peekAhead(1).Value)) == "using" {
            return p.parseNLCompressFunction()
        }
    }

    // NL function lookahead: "transfer <qty> across <ident> <ident>"
    if identName == "transfer" && p.check(lexer.QUANTITY) {
        if p.peekAhead(1).Type == lexer.IDENTIFIER && strings.ToLower(string(p.peekAhead(1).Value)) == "across" {
            return p.parseNLTransferFunction()
        }
    }

    // Existing: standard function call with parens
    if p.check(lexer.LPAREN) {
        return p.parseFunctionCall()
    }

    return &ast.Identifier{Name: string(name.Value)}, nil
}
```

**Note:** This requires a `peekAhead(n)` helper if one doesn't exist. Check if the parser already has one; if not, add it (simple offset from current position).

**Files:** `spec/parser/rdparser.go` (edit)

#### 2d. Implement NL function parse methods

Add to `spec/parser/nl_functions.go`:

```go
// parseNLReadFunction parses: read <quantity> from <identifier>
// Precondition: "read" already consumed, next token is QUANTITY
func (p *RecursiveDescentParser) parseNLReadFunction() (ast.Node, error) {
    // Parse size using parseExponent() — NOT parseExpression() — to avoid
    // consuming FROM as part of a date expression or conversion context.
    // parseExponent() handles: QUANTITY, NUMBER, unary, and exponentiation,
    // but stops before keywords like FROM, PER, OVER, IN, etc.
    size, err := p.parseExponent()
    if err != nil {
        return nil, err
    }
    // Expect FROM
    if !p.match(lexer.FROM) {
        return nil, p.error("expected 'from' after size in 'read <size> from <storage>'")
    }
    // Expect storage type identifier
    if !p.match(lexer.IDENTIFIER) {
        return nil, p.error("expected storage type after 'from' (e.g., ssd, nvme, hdd)")
    }
    storageType := p.previous()
    return &ast.FunctionCall{
        Name:      "read",
        Arguments: []ast.Node{size, &ast.Identifier{Name: string(storageType.Value)}},
    }, nil
}

// parseNLCompressFunction parses: compress <quantity> using <identifier>
func (p *RecursiveDescentParser) parseNLCompressFunction() (ast.Node, error) {
    // Use parseExponent() to avoid consuming "using" as part of the expression
    size, err := p.parseExponent()
    if err != nil {
        return nil, err
    }
    // Expect "using" (contextual keyword — it's an IDENTIFIER, not reserved)
    if !p.match(lexer.IDENTIFIER) || strings.ToLower(string(p.previous().Value)) != "using" {
        return nil, p.error("expected 'using' after size in 'compress <size> using <algorithm>'")
    }
    if !p.match(lexer.IDENTIFIER) {
        return nil, p.error("expected algorithm after 'using' (e.g., gzip, lz4, zstd)")
    }
    algo := p.previous()
    return &ast.FunctionCall{
        Name:      "compress",
        Arguments: []ast.Node{size, &ast.Identifier{Name: string(algo.Value)}},
    }, nil
}

// parseNLTransferFunction parses: transfer <quantity> across <scope> <network>
func (p *RecursiveDescentParser) parseNLTransferFunction() (ast.Node, error) {
    // Use parseExponent() to avoid consuming "across" as part of the expression
    size, err := p.parseExponent()
    if err != nil {
        return nil, err
    }
    // Expect "across" (contextual keyword)
    if !p.match(lexer.IDENTIFIER) || strings.ToLower(string(p.previous().Value)) != "across" {
        return nil, p.error("expected 'across' after size in 'transfer <size> across <scope> <network>'")
    }
    // Expect scope identifier
    if !p.match(lexer.IDENTIFIER) {
        return nil, p.error("expected network scope after 'across' (e.g., local, regional, continental, global)")
    }
    scope := p.previous()
    // Expect network type identifier
    if !p.match(lexer.IDENTIFIER) {
        return nil, p.error("expected network type (e.g., gigabit, wifi, four_g)")
    }
    network := p.previous()
    return &ast.FunctionCall{
        Name:      "transfer_time",
        Arguments: []ast.Node{
            size,
            &ast.Identifier{Name: string(scope.Value)},
            &ast.Identifier{Name: string(network.Value)},
        },
    }, nil
}
```

**Key design note:** `using` and `across` are matched as `IDENTIFIER` tokens with value checks (contextual), NOT as reserved keyword tokens. `from` is matched as `FROM` (already reserved).

**Files:** `spec/parser/nl_functions.go` (edit)

**Phase 2 gate:** `task test` passes. All new parser tests pass. All existing tests still pass.

---

### Phase 3: Interpreter Equivalence Tests

Verify that NL syntax produces identical runtime results to standard function syntax.

#### 3a. Write interpreter equivalence tests

Create `impl/interpreter/nl_read_compress_transfer_test.go`:

```go
// Test that NL forms produce identical results to standard forms
func TestNLReadEquivalence(t *testing.T) {
    tests := []struct{ nl, standard string }{
        {"read 100 MB from ssd", "read(100 MB, ssd)"},
        {"read 1 GB from nvme", "read(1 GB, nvme)"},
        {"read 500 MB from hdd", "read(500 MB, hdd)"},
    }
    for _, tt := range tests {
        nlResult := eval(t, tt.nl)
        stdResult := eval(t, tt.standard)
        assertEqual(t, nlResult, stdResult)
    }
}

// Similar for compress and transfer_time
```

Also test:
- `compress 1 GB using gzip` vs `compress(1 GB, gzip)` (all 5 algorithms)
- `transfer 1 GB across regional gigabit` vs `transfer_time(1 GB, regional, gigabit)` (multiple scope/network combos)
- NL syntax in expressions: `read 100 MB from ssd + read 200 MB from nvme`

**Files:** `impl/interpreter/nl_read_compress_transfer_test.go` (new)

#### 3b. Add golden testdata files

Create `testdata/spec/valid/features/nl_read_compress_transfer.cm`:

```
read 100 MB from ssd
read 1 GB from nvme
compress 1 GB using gzip
compress 500 MB using lz4
transfer 1 GB across regional gigabit
transfer 100 MB across local ten_gig
```

Create matching `testdata/eval/success/features/nl_read_compress_transfer.cm` with expected output.

**Files:**
- `testdata/spec/valid/features/nl_read_compress_transfer.cm` (new)
- `testdata/eval/success/features/nl_read_compress_transfer.cm` (new)

**Phase 3 gate:** `task test` passes. `task quality` passes. All equivalence tests pass.

---

### Phase 4: Update Feature Registry Aliases

#### 4a. Add new NL aliases to registry

In `spec/features/registry.go`, update the function entries:

- `read`: Add `Alias{Name: "read...from", Parseable: true}`
- `compress`: Add `Alias{Name: "compress...using", Parseable: true}`
- `transfer_time`: Add `Alias{Name: "transfer...across", Parseable: true}`

**Files:** `spec/features/registry.go` (edit)

#### 4b. Update registry tests

Verify new aliases appear in search results and have correct `Parseable` values.

**Files:** `spec/features/registry_test.go` (edit)

**Phase 4 gate:** `task test` passes.

---

## Acceptance Criteria

- [x] `read 100 MB from ssd` produces identical result to `read(100 MB, ssd)`
- [x] `compress 1 GB using gzip` produces identical result to `compress(1 GB, gzip)`
- [x] `transfer 1 GB across regional gigabit` produces identical result to `transfer_time(1 GB, regional, gigabit)`
- [x] Case insensitive: `Read 100 MB From SSD` works
- [x] Backwards compatible: `read(100 MB, ssd)` still works (and all other standard function calls)
- [x] Backwards compatible: `read = 42` still works (read as variable name)
- [x] Backwards compatible: `using = 5` still works (using is NOT reserved)
- [x] Backwards compatible: `across = 10` still works (across is NOT reserved)
- [x] NL syntax composes with operators: `read 100 MB from ssd + read 200 MB from nvme`
- [x] Incomplete NL patterns produce clear error messages
- [x] `Feature.Aliases` uses `[]Alias` with per-alias `Parseable` field
- [x] `spec/parser/tokens.go` is removed
- [x] `task test` passes with no regressions
- [x] `task quality` passes

## Success Metrics

- All 3 NL function forms parse correctly and produce results identical to their standard function-call counterparts
- Zero backwards compatibility regressions (measured by full test suite)
- Error messages for incomplete NL patterns are actionable (include expected syntax)

## Dependencies & Risks

**Breaking change:** `Feature.Aliases` type change from `[]string` to `[]Alias`. Mitigated by: only one external consumer (`cmd/calcmark/tui/repl/model.go`), mechanical update.

**`from` keyword overload:** Already used for date expressions ("2 days from today"). Mitigated by: parser lookahead only triggers when preceded by `IDENTIFIER("read")` + `QUANTITY`, which is unambiguous vs. `DURATION_LITERAL` + `FROM`.

**Positional args for transfer_time:** Two identifiers after `across` with no separator. Risk: if `transfer_time` ever gains a 4th parameter, NL syntax can't accommodate it. Mitigated by: this is a brainstorm-accepted trade-off; a separator keyword can be added later if needed.

**Identifier collision:** `read`, `compress`, `transfer` followed by their trigger keywords will always be interpreted as NL functions, even if the user intended them as variable names. Mitigated by: same trade-off as `average` + `of`; these words are unlikely variable names in calculation contexts.

## Alternative Approaches Considered

1. **Lexer multi-token combiner** (original brainstorm proposal) — Rejected because the new patterns have a QUANTITY token between the function name and keyword, which doesn't fit the existing combiner's adjacent-token design.

2. **Reserved keywords for `using`/`across`** (original brainstorm proposal) — Rejected because it breaks backwards compatibility for any user with variables named `using` or `across`. Contextual keywords (like `downtime`) are the established pattern.

## References & Research

### Internal References
- Brainstorm: `docs/brainstorms/2026-02-22-plain-language-functions-brainstorm.md`
- Existing NL functions: `spec/parser/nl_functions.go`
- Parser primary dispatch: `spec/parser/rdparser.go:960-971`
- Lexer multi-token combiner: `spec/lexer/lexer.go:1015-1085`
- Contextual keyword pattern: `spec/parser/rdparser.go:479` (downtime detection)
- Natural syntax keywords: `spec/parser/rdparser.go:1201-1225`
- Feature registry: `spec/features/registry.go:27-34`
- NL equivalence tests: `impl/interpreter/nl_functions_comprehensive_test.go`
- REPL consumer: `cmd/calcmark/tui/repl/model.go:12,25-26`
