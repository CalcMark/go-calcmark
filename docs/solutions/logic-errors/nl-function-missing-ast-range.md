---
title: "NL function error diagnostics displayed on wrong result row"
date: 2026-03-09
tags:
  - tui
  - diagnostics
  - nl-syntax
  - ast
  - parser
  - cross-layer
severity: medium
component:
  - spec/parser
  - impl/document/evaluator
  - cmd/calcmark/tui/editor
symptom: >
  When a CalcMark NL expression on line N produces an evaluation error,
  the error diagnostic appears on a different result row (typically line 1)
  instead of the row corresponding to line N.
root_cause: >
  All 10 NL, implicit, and postfix function parsers returned ast.FunctionCall
  nodes with a nil Range field. Without source position data,
  node.GetRange() returned nil (Line=0), causing the TUI result mapper
  to fall back to a heuristic that placed the error on the first
  non-empty source line.
resolution: >
  Set Range on every ast.FunctionCall node using the keyword token position.
  Added tokenRange() and rangeOrFallback() helpers in rdparser.go.
  Covered with TestAllFunctionCallsHaveRange (AST walk),
  TestEvalErrorDiagnosticLine (functional diagnostic accuracy),
  and a catwalk TUI regression test.
issue_numbers:
  - 41
  - 36
commit: "08a9037"
---

## Symptom

Two lines typed in the TUI:

```
compound $1000 by 5% over 10 years       <- valid, should show result
compound $1000 by 5% monthly over 10 ye  <- invalid "ye", should show error
```

The `invalid duration unit "ye"` error appeared on line 1's result row instead of line 2's.

## Root Cause

All NL and implicit/postfix function parsers returned `ast.FunctionCall` nodes without setting the `Range` field. The evaluator uses `node.GetRange()` to create diagnostics with source line numbers. Without Range, it got nil/Line=0 and fell back to showing the error on the first non-empty line.

10 parsers were affected:

| File | Functions |
|------|-----------|
| `spec/parser/nl_growth_functions.go` | `compound`, `grow`, `depreciate` |
| `spec/parser/nl_functions.go` | `average of`, `square root of`, `read`, `compress`, `transfer` |
| `spec/parser/rdparser.go` | `downtime`, `accumulate` (implicit/postfix) |

The functional parser (`parseFunctionCall`) already set Range from the function name token. NL parsers are separate code paths that missed it.

## Investigation

1. Wrote `TestEvalErrorDiagnosticLine` — pure functional test creating a two-line document and checking `diagnostic.Line`. Result: `diagnostic.Line = 0` for NL compound (expected 2).
2. Traced through evaluator: `node.GetRange()` returned nil because `FunctionCall.Range` was never set.
3. Confirmed the functional parser sets Range at `rdparser.go:1257` but NL parsers at `nl_growth_functions.go:73` did not.

## Fix

### 1. Helpers in `spec/parser/rdparser.go`

```go
func tokenRange(tok lexer.Token) *ast.Range {
    return &ast.Range{
        Start: ast.Position{Line: tok.Line, Column: tok.Column},
        End:   ast.Position{Line: tok.Line, Column: tok.Column + len(tok.Value)},
    }
}

func rangeOrFallback(node ast.Node, fallback lexer.Token) *ast.Range {
    if r := node.GetRange(); r != nil && r.Start.Line > 0 {
        return r
    }
    return tokenRange(fallback)
}
```

### 2. NL parsers capture keyword token at entry

```go
func (p *RecursiveDescentParser) parseNLCompoundFunction() (ast.Node, error) {
    keyword := p.previous() // "compound" token — for range tracking
    // ... parse args ...
    return &ast.FunctionCall{
        Name:      "compound",
        Arguments: args,
        Range:     tokenRange(keyword),
    }, nil
}
```

### 3. Postfix parsers use `rangeOrFallback`

```go
// downtime: left operand (percentage literal) may lack range
return &ast.FunctionCall{
    Name:  "downtime",
    Arguments: []ast.Node{left, ...},
    Range: rangeOrFallback(left, downtimeToken),
}, nil
```

### 4. Cross-cutting prevention test

```go
func TestAllFunctionCallsHaveRange(t *testing.T) {
    expressions := []struct{ name, expr string }{
        {"avg functional", "avg(1, 2, 3)"},
        {"compound NL", "compound $1000 by 5% over 10"},
        {"downtime implicit", "99.9% downtime per month"},
        // ... 25 total covering every function syntax
    }
    // Parse each, walk AST, assert every FunctionCall has Range with Line > 0
}
```

## Verification

- `TestEvalErrorDiagnosticLine` — pure functional test for diagnostic line accuracy (3 cases)
- `TestAllFunctionCallsHaveRange` — 25 expressions covering every function syntax
- Catwalk test `diagnostic_wrong_line_compound` — TUI regression test
- `task test` — full suite passes
- `task quality` — all checks pass

## Prevention

### Why this keeps happening

NL and implicit parsers are created via separate code paths from the functional parser. The functional parser (`parseFunctionCall`) has a single construction site that sets Range. NL parsers are ad-hoc — each manually constructs `&ast.FunctionCall{}` and forgetting `Range` is a silent omission (it's a pointer field defaulting to nil).

### Checklist for new NL parsers

1. Capture the keyword token first: `keyword := p.previous()`
2. Parse arguments using existing helpers
3. Return FunctionCall with Range: `Range: tokenRange(keyword)`
4. Add both functional and NL test cases to `TestAllFunctionCallsHaveRange`
5. Run `task test` and `task quality`

### Future: constructor function

The highest-leverage prevention would be replacing all `&ast.FunctionCall{}` literals with a constructor that requires Range as a parameter. This converts a runtime bug into a compile-time error.

## Related

- **Issue #36**: BinaryOp/ComparisonOp/UnaryOp missing Range (Wave 2 of this bug class)
- **Issue #41**: NL FunctionCall nodes missing Range (Wave 3 — this fix)
- [`docs/solutions/ui-bugs/tui-editor-rendering-divider-status-bar-error-line.md`](../ui-bugs/tui-editor-rendering-divider-status-bar-error-line.md) — Wave 1: evaluator used wrong type assertion for line mapping
- [`docs/solutions/ui-bugs/context-footer-statement-index-drift.md`](../ui-bugs/context-footer-statement-index-drift.md) — Statement index drift in results.go
- [`docs/solutions/integration-issues/nl-functional-syntax-parity-and-doc-staleness.md`](../integration-issues/nl-functional-syntax-parity-and-doc-staleness.md) — NL/functional parity methodology
- [`docs/solutions/logic-errors/doceval-progressive-index-block-splitting-misalignment.md`](doceval-progressive-index-block-splitting-misalignment.md) — Same symptom class in doceval layer

### Diagnostic misalignment fix history

| Wave | Issue | Root Cause | Scope |
|------|-------|-----------|-------|
| 1 | — | Evaluator wrong type assertion | Single site |
| 2 | #36 | BinaryOp/ComparisonOp/UnaryOp missing Range | 9 parser sites |
| 3 | #41 | NL FunctionCall nodes missing Range | 10 parser sites + cross-cutting test |
