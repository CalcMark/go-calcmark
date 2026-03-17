---
title: "LSP debounce causes stale results for read-only requests"
category: integration-issues
tags: [lsp, debounce, state-management, concurrency, signature-help, completion]
module: lsp
symptom: "Signature help, completions, and hover return stale or nil results after typing"
root_cause: "Read-only LSP requests read from the debounced evaluation snapshot, which lags 150ms behind the actual document content"
date: 2026-03-16
---

## Problem

After implementing the CalcMark LSP server with a 150ms debounce on `textDocument/didChange`, read-only requests like `textDocument/signatureHelp`, `textDocument/completion`, and `textDocument/hover` returned stale or nil results. For example, typing `accumulate(` triggered signature help, but the handler read from the snapshot which still contained the old text (before `accumulate(` was typed).

## Root Cause

The LSP server stored document state in a `DocumentSnapshot` that was only updated after the debounce timer fired (150ms after the last keystroke). All request handlers — including text-based operations like extracting the function name before `(` — read from `snap.Source`. But VS Code sends `signatureHelp` immediately when `(` is typed, well within the debounce window.

```
User types "accumulate("
  → VS Code sends didChange (debounce timer starts, 150ms)
  → VS Code sends signatureHelp (immediately, triggered by "(")
  → Handler reads snap.Source → still has OLD content → no function found → nil result
  → 150ms later: snapshot updates with "accumulate(" → too late
```

## Solution

Separate the raw source text from the evaluated snapshot. Store source text immediately on every `didChange`, without debouncing. Only debounce the expensive evaluation (parsing, semantic analysis, diagnostics).

```go
type documentState struct {
    mu       sync.RWMutex
    snapshot *DocumentSnapshot  // debounced: updated after eval
    source   string             // immediate: updated on every didChange
    // ...
}
```

In `didChange`:
```go
// Store source immediately — read-only requests see latest text
ds.setSource(source)

// Debounce the expensive evaluation
ds.timer = time.AfterFunc(debounceDelay, func() {
    snap := s.evaluate(source)
    ds.setSnapshot(snap)
    s.publishDiagnostics(ctx, uri, snap)
})
```

In request handlers (completion, hover, signature help, definition, symbols):
```go
source := ds.getSource()     // always up-to-date
snap := ds.getSnapshot()     // may be stale, but has evaluated variable values
lineText := getLineText(source, line)  // use source for text-based operations
```

## Prevention

- **Separate "what the user typed" from "what we evaluated."** Any time you debounce an expensive operation, ask: "Do any request handlers need the pre-debounce input?" If yes, store it separately.
- **The pattern generalizes:** In any system where reads and writes have different latency requirements, decouple the read path from the write path. The write path (evaluation) can be debounced; the read path (text extraction) must be immediate.
- **Test with realistic timing:** Unit tests that call handlers synchronously after evaluation will never catch this bug. The e2e protocol test that sends `didChange` followed immediately by `signatureHelp` (without waiting for the debounce) is what exposed it.
