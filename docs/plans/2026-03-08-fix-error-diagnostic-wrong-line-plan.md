---
title: "fix: Error diagnostic appears next to wrong line in TUI"
type: fix
status: completed
date: 2026-03-08
issue: 36
deepened: 2026-03-08
---

# fix: Error diagnostic appears next to wrong line in TUI

## Enhancement Summary

**Deepened on:** 2026-03-08
**Sections enhanced:** 5
**Research sources:** Source code analysis, existing institutional learnings (3 solution docs), architecture + simplicity reviews

### Key Improvements
1. Widened scope: `ComparisonOp` (1 site) and `UnaryOp` (2 sites) also lack Range — 9 total sites
2. Helper placed in `spec/ast/position.go` as exported `SpanNodes()` — respects dependency direction, reusable
3. Evaluator guard tightened: also reject zero-valued ranges (`r.Start.Line > 0`)

### New Considerations Discovered
- 20+ other AST node creation sites use empty `&ast.Range{}` (Line=0, Column=0) — passes nil check but produces useless position data. Separate follow-up.
- The "2 days from today" synthetic BinaryOp (line 1090) has no operator token but `spanNodes(baseDate, durationNode)` still produces correct spanning range.

## Overview

When a calculation on line 2 produces a runtime error (e.g., type mismatch), the error diagnostic appears next to line 1 in the TUI Results pane. The error should appear next to the line that actually caused it.

## Problem Statement

**Repro:**
1. Type `b = 12 apples` on line 1
2. Type `10 / b` on line 2
3. Observe: error "cannot divide number (10)..." appears next to line 1

**Expected:** Error on line 2 (where `10 / b` is).

## Root Cause

`BinaryOp`, `ComparisonOp`, and `UnaryOp` AST nodes are created **without a `Range`** in the parser (`spec/parser/rdparser.go`).

**All affected creation sites:**

| Line | Node Type | Context |
|------|-----------|---------|
| 274 | BinaryOp | `parseOr()` — `or` operator |
| 298 | BinaryOp | `parseAnd()` — `and` operator |
| 323 | ComparisonOp | `parseComparison()` — `>`, `<`, `==`, etc. |
| 348 | BinaryOp | `parseAdditive()` — `+`, `-` |
| 445 | BinaryOp | `parseMultiplicative()` — `*`, `/`, `%` |
| 700 | BinaryOp | `parseExponent()` — `^` |
| 726 | UnaryOp | `parseUnary()` — unary `+`/`-` |
| 744 | UnaryOp | `parseUnary()` — `not` |
| 1090 | BinaryOp | `parsePrimary()` — "2 days from today" synthetic |

The failure chain:
1. Parser creates `BinaryOp{Operator: "/", Left: NumberLiteral, Right: Identifier, Range: nil}`
2. Evaluator (`impl/document/evaluator.go:517`) calls `node.GetRange()` → returns `nil`
3. `diag.Line` stays at default `0`
4. `results.go:80` excludes diagnostics with `Line <= 0` from `diagByLine`
5. Fallback at `results.go:137-169` activates — shows error on `stmtIdx == 0` (first non-empty line)

### Institutional Learning

This is the same class of bug documented in `docs/solutions/ui-bugs/tui-editor-rendering-divider-status-bar-error-line.md`. Previous fix (commit `7c3aed8`) changed the evaluator from type-asserting `*ast.Assignment` to using the `GetRange()` interface method. That fix was correct but incomplete — it only works when the node actually HAS a Range set.

From `docs/solutions/ui-bugs/context-footer-statement-index-drift.md`: this is the 5th instance of source-line-to-result mapping bugs in go-calcmark. Prevention: always verify line-to-result mapping with separate indices.

## Proposed Solution

### 1. Add `SpanNodes()` helper in `spec/ast/position.go`

Exported function in the ast package (not the parser), since it operates purely on ast types and respects the dependency direction. Reusable by semantic checker, future transforms, etc.

```go
// SpanNodes returns a Range spanning from the start of left to the end of right.
// Returns the best partial range if one side is nil; returns nil if both are nil.
func SpanNodes(left, right Node) *Range {
    lr := left.GetRange()
    rr := right.GetRange()
    if lr == nil && rr == nil {
        return nil
    }
    if lr == nil {
        return rr
    }
    if rr == nil {
        return lr
    }
    return &Range{Start: lr.Start, End: rr.End}
}
```

**Partial range handling:** If only one child has a Range, return it rather than nil. A partial range (even just the left operand's position) is more useful than no range at all.

### 2. Update 9 creation sites in `spec/parser/rdparser.go`

For BinaryOp (6 sites) and ComparisonOp (1 site):
```go
left = &ast.BinaryOp{
    Operator: string(op.Value),
    Left:     left,
    Right:    right,
    Range:    ast.SpanNodes(left, right),
}
```

For UnaryOp (2 sites), span from the operator to the operand:
```go
return &ast.UnaryOp{
    Operator: string(op.Value),
    Operand:  right,
    Range:    right.GetRange(), // unary: use operand's range (operator is prefix)
}
```

### 3. Tighten evaluator guard in `impl/document/evaluator.go:517`

Guard against zero-valued ranges (from `&ast.Range{}` sites):
```go
if r := node.GetRange(); r != nil && r.Start.Line > 0 {
    diag.Line = r.Start.Line
    diag.Column = r.Start.Column
}
```

### Why not fix in the evaluator/results layer?

The evaluator already does the right thing — it reads `node.GetRange()`. The bug is that the data isn't there. Fixing at the parser level means ALL consumers of the AST (evaluator, formatter, LSP, etc.) benefit from accurate position info. No evaluator tree-walking fallback — fix the root cause.

## Acceptance Criteria

- [x] `BinaryOp` nodes have `Range` set in all 6 creation sites in the parser
- [x] `ComparisonOp` has `Range` set (1 site)
- [x] `UnaryOp` has `Range` set (2 sites)
- [x] Evaluator guards against zero-valued ranges (`r.Start.Line > 0`)
- [x] Error diagnostic for `10 / b` appears next to line 2, not line 1, in the TUI
- [x] Error diagnostic for any bare expression error appears on the correct line
- [x] Existing parser tests pass (Range is additive — no behavior change)
- [x] New catwalk test reproducing the exact bug scenario
- [x] `task test` and `task quality` pass

## MVP

### Test: `cmd/calcmark/tui/editor/testdata/diagnostic_wrong_line`

New catwalk test with `observe=results` reproducing the exact scenario from issue #36.

### Fix: 3 files

1. **`spec/ast/position.go`** — Add `SpanNodes(left, right Node) *Range`
2. **`spec/parser/rdparser.go`** — Update 9 creation sites with `Range: ast.SpanNodes(left, right)` or `Range: right.GetRange()`
3. **`impl/document/evaluator.go:517`** — Add `r.Start.Line > 0` guard

### Verify

Run `task test` and `task quality`.

## Technical Considerations

- `SpanNodes` gracefully handles nil ranges — returns best partial range, nil only if both children lack range
- `BinaryOp` is built recursively in chains (e.g., `a + b + c` = `BinaryOp(BinaryOp(a, b), c)`), so the outer BinaryOp's left already has a Range from the inner one — composes correctly
- The "2 days from today" synthetic BinaryOp (line 1090) has no operator token, but `SpanNodes(baseDate, durationNode)` produces a correct spanning range
- `Assignment` already has Range set (lines 240-249) — no change needed
- 20+ other node types use empty `&ast.Range{}` — separate follow-up issue, not in scope

## Dependencies & Risks

- **Low risk**: Adding Range is purely additive — no existing behavior changes
- **Follow-up**: File issue for 20+ empty `&ast.Range{}` sites that produce Line=0 diagnostics

## References

- Issue: [#36](https://github.com/CalcMark/go-calcmark/issues/36)
- Prior fix: `docs/solutions/ui-bugs/tui-editor-rendering-divider-status-bar-error-line.md` (commit `7c3aed8`)
- Prevention: `docs/solutions/ui-bugs/context-footer-statement-index-drift.md`
- Key files: `spec/parser/rdparser.go`, `spec/ast/position.go`, `impl/document/evaluator.go:504-522`, `cmd/calcmark/tui/editor/results.go:76-169`
