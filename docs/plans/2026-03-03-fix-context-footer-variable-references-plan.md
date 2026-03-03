---
title: "fix: Context footer variable references not displaying after AST-driven refactor"
type: fix
status: active
date: 2026-03-03
regression_commit: 6ce0bb3
related_issue: "#10"
deepened: 2026-03-03
---

# fix: Context footer variable references not displaying after AST-driven refactor

## Enhancement Summary

**Deepened on:** 2026-03-03
**Research agents used:** pattern-recognition, performance-oracle, code-simplicity, architecture-strategist, git-history-analyzer, spec-flow-analyzer, bug-reproduction-validator, learnings-researcher

### Key Improvements
1. Narrowed from 4 hypotheses to 1 actionable root cause (H1: whitespace guard) with H3/H4 eliminated as logically impossible regression causes
2. Added performance fix: eliminate triple `GetLineResults()` call per render frame (3x wasted work)
3. Expanded test cases from 3 to 8, covering cross-block refs, frontmatter globals, built-in constants, maxRefs truncation, and error priority

### New Considerations Discovered
- The whitespace guard has been broken since the initial commit (`d36390ff`, 2025-11-29) — it is pre-existing, not introduced by `6ce0bb3`
- Bug reproduction validator could NOT reproduce "no refs at all" — the happy path works. The bug manifests as spurious refs on whitespace-only lines and potential index drift for documents containing whitespace-only calc lines
- `GetLineResults()` is called 3x per frame (not 2x) — also from `computeAlignedModelFresh`
- Do NOT modify the `results` observer format — would break all existing catwalk test expectations

## Overview

The context footer no longer displays the list of variables referenced by the current calculation line. This is a regression introduced in commit `6ce0bb3` ("Fix variable reference detection to match whole words only"), which replaced text-based variable detection (`FindLineReferences` using `strings.Contains`) with AST-driven extraction (`ExtractStatementReferences`).

The user sees this as a "status bar" regression, but the variable references are rendered by `renderContextFooter` in `view_footer.go`, displayed just above the actual status bar in the UI layout (see `view.go:186`).

## Problem Statement

**Before (text-based):** `getLineReferences` called `FindLineReferences(line, knownVars, 4)` which scanned the raw line text against all known environment variables using `strings.Contains`. This was O(V*L) but always produced results when variables were present in the text.

**After (AST-based):** `getLineReferences` reads `results[lineNum].ReferencedVars` from `GetLineResults()`, which is populated by `document.ExtractStatementReferences(statements[stmtIdx])` during the result-building loop. This is O(n) per statement but depends on:
1. `b.Statements()` being populated (requires successful parse)
2. `stmtIdx < len(statements)` (requires correct line-to-statement mapping)
3. Variables being present in `env.GetAllVariables()` (requires evaluation)

## Root Cause Analysis

### Research Insight: Narrowed to One Hypothesis

Per code-simplicity and architecture analysis, H3 (debounce timing) and H4 (environment not available) are **eliminated** — both are shared limitations between old and new code and cannot cause a regression. H2 (empty statements) is deprioritized — existing tests pass for freshly created documents.

### H1: Statement-to-line index misalignment (Confirmed Root Cause)

The `countNonEmptyLinesBefore` function maps source line indices to statement indices. This mapping assumes a 1:1 correspondence between non-empty source lines and parser-produced AST nodes.

**The whitespace-only line guard at `results.go:93-104` is a no-op loop:**

```go
trimmed := line              // (A) trimmed = line
for _, c := range line {
    if c != ' ' && c != '\t' {
        trimmed = line       // (B) trimmed = line (SAME value!)
        break
    }
}
if len(trimmed) == 0 || trimmed == "" {
    // Only catches truly empty strings ("")
    // Whitespace-only "   " NEVER caught
```

**Git history confirms**: This code was introduced in commit `d36390ff` (2025-11-29) and has **never been correct**. It was never modified in any subsequent commit. This is the 4th occurrence of the same class of line/statement alignment bug in this codebase — the formatters were fixed in commit `42409f8` but `results.go` was not.

**The definition mismatch**: `countNonEmptyLinesBefore` uses `strings.TrimSpace` (correct), but the inline guard does not (broken). When a whitespace-only line like `"   "` passes through:
1. The guard doesn't skip it (broken trim)
2. `countNonEmptyLinesBefore` doesn't count it (correct TrimSpace)
3. The whitespace-only line consumes the `stmtIdx` of the next real statement
4. For documents with whitespace-only lines in calc blocks, `ReferencedVars` gets populated from the wrong statement or not at all

### Research Insight: Institutional Knowledge

From `docs/solutions/ui-bugs/tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md`:
> "Source-line loop index used to access results array (blank lines don't produce results)"

The prevention checklist warns: **"Source lines and results iterated together? Must use separate `resultIdx`."** The current `results.go` code violates this exact checklist item.

## Technical Approach

### Phase 1: Reproduce & Test (TDD)

Write targeted tests that exercise `getLineReferences` end-to-end. Run tests BEFORE fix to confirm failure.

#### 1a. Unit test: `getLineReferences` end-to-end

**File**: `cmd/calcmark/tui/editor/view_footer_test.go`

```go
func TestGetLineReferences(t *testing.T) {
    tests := []struct {
        name      string
        source    string
        lineIdx   int
        wantNames []string
    }{
        {
            name:      "single block references",
            source:    "x = 10\ny = x + 5\n",
            lineIdx:   1,
            wantNames: []string{"x"},
        },
        {
            name:      "multi-variable reference sorted",
            source:    "a = 1\nb = 2\nc = a + b\n",
            lineIdx:   2,
            wantNames: []string{"a", "b"},
        },
        {
            name:      "no refs for literal-only assignment",
            source:    "x = 42\n",
            lineIdx:   0,
            wantNames: nil,
        },
        {
            name:      "anonymous expression references variable",
            source:    "x = 10\nx * 2\n",
            lineIdx:   1,
            wantNames: []string{"x"},
        },
        {
            name:      "whitespace-only line does not misalign",
            source:    "x = 10\n   \ny = x + 5\n",
            lineIdx:   2,
            wantNames: []string{"x"},
        },
        {
            name:      "whitespace-only line gets no spurious refs",
            source:    "x = 10\n   \ny = x + 5\n",
            lineIdx:   1,
            wantNames: nil, // whitespace line should have NO refs
        },
        {
            name:      "multiple consecutive whitespace lines",
            source:    "x = 10\n   \n  \ny = x + 5\n",
            lineIdx:   3,
            wantNames: []string{"x"},
        },
        {
            name:      "cross-block variable reference",
            source:    "rent = 1500\n\n# Totals\n\ntotal = rent + 200\n",
            lineIdx:   4,
            wantNames: []string{"rent"},
        },
    }
    // ... test body calls New(doc), m.eval.Evaluate(m.doc), m.getLineReferences(tt.lineIdx)
}
```

### Research Insight: Additional Test Cases from Spec Flow Analysis

The following additional test cases were identified by spec-flow and pattern-recognition analysis. Add to test 1a:

- **Built-in constant positive test**: `source: "area = PI * 5\n"` → wantNames: `["PI"]` (complement to issue #10 negative test)
- **Error line priority**: `source: "y = undefined_var + 5\n"` → wantNames: nil (error takes priority, refs not shown)
- **maxRefs=4 truncation**: 5+ variable refs → only first 4 returned (sorted alphabetically)

### Phase 2: Fix Root Cause

**Fix the whitespace-only line guard** in `results.go:93-104`. Replace 8 lines of broken logic with 1 correct line:

```go
// Before (broken — 8 lines, trimmed is always equal to line):
trimmed := line
for _, c := range line {
    if c != ' ' && c != '\t' {
        trimmed = line
        break
    }
}
if len(trimmed) == 0 || trimmed == "" {

// After (correct — 1 line, consistent with countNonEmptyLinesBefore):
if strings.TrimSpace(line) == "" {
```

`strings.TrimSpace` is already imported and used by `countNonEmptyLinesBefore` at line 211. This aligns both checks to the same definition of "empty."

### Phase 3: Performance Fix (Eliminate Triple `GetLineResults()`)

### Research Insight: Performance Oracle Finding

`GetLineResults()` is called **3 times per `View()` frame**:
1. `computeAlignedModelFresh` (view.go:290) — alignment computation
2. `renderContextFooter` (view_footer.go:14) — error/calc state check
3. `getLineReferences` (view_footer.go:98) — reads ReferencedVars

Each call rebuilds the entire results slice: iterates all blocks, all lines, performs AST walks, allocates maps. For a 200-line document, this is 450 AST walks per frame.

**Fix**: Pass pre-computed `results` into `getLineReferences` as a parameter:

```go
// Before: getLineReferences calls GetLineResults() again
func (m Model) getLineReferences(lineNum int) []components.VarReference {
    results := m.GetLineResults() // REDUNDANT — already computed in caller
    ...
}

// After: accept pre-computed results
func (m Model) getLineReferences(lineNum int, results []LineResult) []components.VarReference {
    if lineNum >= len(results) || len(results[lineNum].ReferencedVars) == 0 {
        return nil
    }
    ...
}
```

Then in `renderContextFooter`:
```go
state.References = m.getLineReferences(m.cursorLine, results)
```

This eliminates 1 of the 3 redundant calls with a 2-line signature change.

### Phase 4: Harden with Tests

1. **Add `footer` observer** to catwalk infrastructure in `catwalk_test.go` — output formatted `VarReference{Name, Value}` pairs (not raw strings)
2. **Do NOT modify existing `results` observer format** — this would break all existing catwalk test expectations and require mass regeneration
3. **Add catwalk test** for cursor navigation: move to referencing line (footer shows refs), move to text line (footer empty), move back (refs reappear)

### Research Insight: Testing Strategy from Institutional Knowledge

From `docs/solutions/ui-bugs/preview-pane-jump-frontmatter-and-context-footer-false-positive.md`:
> Never parse `View()` output to verify context footer — the `│` separator is ambiguous between pane divider and value separator.

Test `getLineReferences` directly or use the dedicated catwalk `footer` observer. Do NOT split `View()` output on `│`.

## Acceptance Criteria

- [ ] At least one test reproduces the whitespace-guard bug (fails before fix, passes after)
- [ ] Context footer displays variable references when cursor is on a calc line that references other variables
- [ ] Whitespace-only lines in calc blocks get NO spurious references
- [ ] Currency literals like `EUR` are NOT mistaken for variable `E` (issue #10 fix preserved)
- [ ] `getLineReferences` no longer calls `GetLineResults()` redundantly
- [ ] `task test` passes
- [ ] `task quality` passes

## Key Files

| File | Role | Change |
|------|------|--------|
| `cmd/calcmark/tui/editor/results.go:93-104` | Broken whitespace-only line guard | Replace 8-line no-op with `strings.TrimSpace` |
| `cmd/calcmark/tui/editor/view_footer.go:97-121` | `getLineReferences` | Accept `[]LineResult` parameter, remove redundant `GetLineResults()` call |
| `cmd/calcmark/tui/editor/view_footer.go:13-41` | `renderContextFooter` | Pass `results` to `getLineReferences` |
| `cmd/calcmark/tui/editor/view_footer_test.go` | New test file | 8+ test cases for `getLineReferences` |
| `cmd/calcmark/tui/editor/catwalk_test.go` | Catwalk infrastructure | Add `footer` observer |

## References

- Regression commit: `6ce0bb3` (Fix variable reference detection to match whole words only)
- Original issue: #10 (E matched inside EUR)
- Whitespace guard origin: `d36390ff` (2025-11-29, broken since day one)
- Same-class bug fix: `42409f8` (formatter blank-line indexing)
- Related learning: `docs/solutions/ui-bugs/locale-formatting-bypass-in-tui.md` — confirms footer rendering path is sound when data is supplied
- Related learning: `docs/solutions/ui-bugs/tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md` — identical index-drift class, prevention checklist applies
- Related learning: `docs/solutions/ui-bugs/preview-pane-jump-frontmatter-and-context-footer-false-positive.md` — do NOT test through `View()` output
