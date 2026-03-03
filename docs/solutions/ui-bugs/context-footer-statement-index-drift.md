---
title: "Context footer regression: statement index drift, self-reference, and multi-statement eval output"
date: 2026-03-03
category: ui-bugs
tags: [statement-index-drift, whitespace-guard, context-footer, self-reference, text-formatter, regression, tui-editor]
module: TUI Editor
symptom: "Context footer shows no variable references; self-references appear in footer; cm eval piped output missing intermediate results"
root_cause: "Broken whitespace guard in results.go (no-op loop), missing self-reference filter in view_footer.go, LastValue() in TextFormatter"
commit_introduced: "6ce0bb3"
commits_fixed: ["4bdf180", "f30bf54", "c58348c"]
severity: high
---

# Context Footer Statement Index Drift Regression

## Problem Summary

After commit `6ce0bb3` ("Fix variable reference detection to match whole words only"), the TUI context footer stopped displaying variable references. Investigation revealed three related bugs.

## Symptoms

1. **Status bar empty**: No variable references displayed on any line in the TUI editor
2. **Self-reference in footer**: `hundred_gig = throughput(hundred_gig)` showed `hundred_gig` as a referenced variable
3. **Missing eval output**: `echo "a = 10 kg\nb = a + 10 kg" | cm eval` only showed `20 kg` (last value)

## Root Causes

### 1. Whitespace Guard No-Op Loop (Primary)

In `cmd/calcmark/tui/editor/results.go:93-104`, the whitespace guard loop was a no-op. It iterated through characters but never executed `continue` on the outer loop, so whitespace-only lines were not skipped. This corrupted the statement index mapping -- line N in the source no longer corresponded to statement N in the evaluated results.

**Before (broken):**
```go
for _, r := range line {
    if r != ' ' && r != '\t' && r != '\r' {
        break
    }
}
```

**After (fixed):**
```go
if strings.TrimSpace(line) == "" {
    results = append(results, lr)
    lineNum++
    continue
}
```

### 2. Self-Reference Not Filtered

`extractIdentifiers` in `spec/document/deps.go` walks AST nodes including function arguments. When a function argument is an `ast.Identifier` sharing the variable name being defined, it appears as a referenced variable in the footer.

**Fix** in `cmd/calcmark/tui/editor/view_footer.go:getLineReferences`:
```go
definedVar := results[lineNum].VarName
for _, varName := range refs {
    if varName == definedVar {
        continue  // Filter self-references
    }
    // ... rest of reference lookup
}
```

### 3. TextFormatter LastValue() Bug

`TextFormatter` non-verbose mode used `block.LastValue()` which returns only the final statement's result per block.

**Fix** in `format/text_formatter.go`:
```go
for _, result := range block.Results() {
    if result != nil {
        fmt.Fprintln(w, df.Format(result))
    }
}
```

## Bug Class: Statement Index Drift

This is the **4th instance** of statement index drift in go-calcmark, where the mapping between source lines and evaluated statement indices gets corrupted. Previous instances documented in:

- [tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md](tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md) -- identical class: "blank lines cause result drift in formatters"

The prevention checklist item violated: *"Source lines and results iterated together? Must use separate resultIdx."*

## Investigation Steps

1. Ran `task test` to verify the bug existed (existing tests passed but footer was empty)
2. Added regression tests that exposed the whitespace guard failure
3. Used catwalk integration tests to verify the full TUI pipeline
4. Identified self-reference as a separate issue via manual testing with `network_functions.cm`
5. Found eval output bug through piped input testing

## Testing Layers

| Layer | File | Coverage |
|-------|------|----------|
| Unit | `results_regression_test.go` | Whitespace guard, self-ref filter, cross-block refs |
| Rendering pipeline | `context_footer_render_test.go` | Footer rendering with formatted values |
| Catwalk integration | `catwalk_context_footer_test.go` | Full TUI pipeline with key navigation |
| Formatter | `text_formatter_test.go` | Multi-statement block output |

## Prevention Strategies

### Patterns to Watch For

- **Loop-shaped no-ops**: Guard clauses that iterate but don't skip/modify state. The whitespace guard LOOKED correct but never executed `continue` on the outer loop.
- **Partial filtering**: When filtering references, verify the filter covers all cases (enum arguments, shadowed names, nested calls).
- **LastValue() aggregation**: Any time code calls `.LastValue()` to represent "the result", question whether multiple results could exist.

### Testing Recommendations

Always include these cases for index-sensitive code:
- Whitespace-only lines between executable statements
- Multiple consecutive whitespace lines
- Function arguments sharing variable names
- Multi-statement blocks in non-verbose output
- Cross-block variable references with whitespace gaps

### Code Review Checklist

- [ ] Guard clauses: Does the loop actually skip items? Trace the control flow.
- [ ] Filter completeness: What are ALL ways an item could match?
- [ ] Results aggregation: Why LastValue() instead of iterating? If no comment, ask.
- [ ] Pipeline consistency: Is the source-to-result mapping tested end-to-end?

## Related Documentation

- **Plan**: [2026-03-03-fix-context-footer-variable-references-plan.md](/docs/plans/2026-03-03-fix-context-footer-variable-references-plan.md)
- **Related solution**: [tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md](tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md)
- **Related solution**: [locale-formatting-bypass-in-tui.md](locale-formatting-bypass-in-tui.md) -- context footer locale formatting path
- **Related solution**: [preview-pane-jump-frontmatter-and-context-footer-false-positive.md](preview-pane-jump-frontmatter-and-context-footer-false-positive.md) -- testing methodology
- **Issue #10**: E matched inside EUR (led to commit 6ce0bb3 which exposed the latent bug)
- **Issue #13**: Variable reference whole-word matching (the commit that surfaced the regression)

## Key Files

| File | Change |
|------|--------|
| `cmd/calcmark/tui/editor/results.go` | Fixed whitespace guard loop |
| `cmd/calcmark/tui/editor/view_footer.go` | Added self-reference filter, accept pre-computed results |
| `cmd/calcmark/tui/editor/view.go` | Hoisted GetLineResults() to View() top (1x per frame) |
| `format/text_formatter.go` | Iterate Results() instead of LastValue() |
