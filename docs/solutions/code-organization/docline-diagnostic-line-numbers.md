---
title: "DocLine: Document-Absolute Line Numbers in Diagnostics"
category: code-organization
date: 2026-03-12
tags: [diagnostics, line-numbers, frontmatter, semantic-checker, evaluator, tui]
components: [spec/semantic/checker.go, impl/document/evaluator.go, spec/document/document.go, cmd/calcmark/tui/editor/results.go]
---

## Problem

Diagnostic error messages like `"first defined at line 2"` used block-relative line numbers. With frontmatter present, "line 2" is wrong — the user sees it as line 6 in their document. The CLI (`cm eval`) didn't report line numbers at all.

Any non-TUI consumer (CLI, REPL, LSP) had no way to know which document line an error was on without reverse-engineering the block-to-line mapping that only the TUI performed.

## Root Cause

The CalcMark lexer always starts at `line: 1` for each calc block independently. AST line numbers are block-relative by design. The evaluator stored these block-relative numbers directly in `document.Diagnostic.Line` without adjusting for frontmatter or preceding blocks.

The TUI worked around this by maintaining its own `lineNum` counter in `results.go` that accumulates across frontmatter and blocks — but this knowledge was trapped in the TUI layer.

## Solution

### 1. `Diagnostic.DocLine` field

Added a `DocLine int` field to `document.Diagnostic` — the 1-indexed document-absolute line number. `Line` stays block-relative for the TUI's existing per-block lookup.

```go
type Diagnostic struct {
    // ...
    Line    int // 1-indexed line number within the block (for block-internal lookups)
    Column  int // 1-indexed column number
    DocLine int // 1-indexed document-absolute line number (0 if unknown)
}
```

### 2. `Frontmatter.LineCount()` method

Counts lines from `rawSource` (not `Serialize()`, which adds an extra CommonMark blank line).

### 3. `blockLineOffset()` evaluator helper

Computes offset = frontmatter lines + source lines of all preceding blocks.

### 4. `Checker.SetLineOffset()` for message text

The semantic checker's human-readable messages (e.g., `"first defined at line N"`) use the offset. The checker also fixed a pre-existing off-by-one: AST positions are 1-based (lexer starts at `line: 1`), so the old `Line + 1` was wrong.

### 5. CLI errors now include line numbers

`"line 7: undefined_variable: undefined variable \"x\""` instead of just the error code and message.

## Why the TUI Does NOT Use DocLine

The TUI has **two separate line-number concerns**:

1. **Data layer** (`results.go`): Maps diagnostics to source lines using a per-block `diagByLine` map keyed by block-relative `diag.Line`. This works correctly and is well-tested. Switching to `DocLine` would save a few lines but risk regressing the lookup.

2. **Rendering layer** (`aligned.go`, `view_panes.go`, `linemodel.go`): Maps source lines to *visual* lines accounting for wrapping, pane widths, scroll offsets, and cursor position. `DocLine` is irrelevant here — this layer transforms `LineResult.LineNum` (0-indexed source position) through wrapping/alignment into visual coordinates.

The TUI already solves its own problem correctly. `DocLine` is additive for non-TUI consumers (CLI, REPL, LSP, JSON output) that don't have the TUI's block-iteration machinery.

## Key Files

| File | Change |
|------|--------|
| `spec/document/document.go` | Added `DocLine` field to `Diagnostic` |
| `spec/document/frontmatter.go` | Added `LineCount()` method |
| `spec/semantic/checker.go` | Added `lineOffset` field, `SetLineOffset()`, fixed off-by-one |
| `impl/document/evaluator.go` | Added `blockLineOffset()`, sets `DocLine` + `SetLineOffset` at all diagnostic creation sites |
| `cmd/calcmark/tui/editor/results.go` | Unchanged — continues using block-relative `diag.Line` |

## Prevention

- When adding fields that carry position info, always distinguish block-relative vs document-absolute in the field name and comment.
- The lexer always starts at `line: 1` per block — this is by design and should not change. Offset computation belongs in the evaluator, not the lexer or parser.
- `Serialize()` adds a trailing `\n` (CommonMark blank line) — don't use it for line counting. Use `rawSource` directly via `LineCount()`.
