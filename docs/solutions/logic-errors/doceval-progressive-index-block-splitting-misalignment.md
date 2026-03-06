---
title: "doceval progressive evaluation produced misaligned results due to index-based block mapping"
date: 2026-03-06
category: logic-errors
component: cmd/doceval
severity: high
symptoms:
  - "RESULTS panels on Hugo site showed wrong values next to wrong source lines"
  - "Some RESULTS panels were entirely missing"
  - "Only progressive pages (calcmark_build: progressive) were affected; standalone pages worked correctly"
tags:
  - doceval
  - progressive-evaluation
  - index-mapping
  - block-classification
  - hugo-site
  - two-phase-build
---

# doceval Progressive Evaluation: Index-Based Block Mapping Misalignment

## Problem

Progressive pages on the Hugo site showed misaligned RESULTS panels — wrong values next to wrong source lines, and some panels missing entirely. Standalone pages were unaffected.

## Root Cause

The CalcMark interpreter's document classifier does not preserve a 1:1 mapping between source code blocks and interpreter blocks. When `evalProgressive` concatenated all `` ```calcmark `` blocks into a single document and evaluated it, the classifier would split mixed content (e.g., a block containing both assignment lines and blank lines, or interleaved text and calculations) into separate `CalcBlock` and `TextBlock` nodes.

The old approach collected all statements from the evaluated document into a flat `allStmts` slice, then used `mapBlockResults` to split that slice back into per-block results by counting lines — assuming statement N in the flat list corresponded to line N in the original source. Because the interpreter's block splitting changed the count and ordering of statements relative to source lines, this created **index drift**: results attributed to wrong source lines, or the mapping running out of statements before exhausting source lines.

Example: a source block with 5 lines (3 calc + 2 text) becomes 3 interpreter blocks (CalcBlock, TextBlock, CalcBlock). The flat statement list has 3 entries, but the original block has 5 lines. Index-based mapping assigns results to the wrong lines.

## Solution

Replace index-based mapping with **source-line text matching**. After evaluating the combined document, build a lookup map keyed by trimmed source text:

```go
type lineResult struct {
    result   string
    variable string
}
resultBySource := make(map[string]lineResult)

for _, node := range doc.GetBlocks() {
    if cb, ok := node.Block.(*document.CalcBlock); ok {
        for _, stmt := range format.AlignResults(cb) {
            if stmt.Result != nil && strings.TrimSpace(stmt.Source) != "" {
                resultBySource[strings.TrimSpace(stmt.Source)] = lineResult{
                    result:   df.Format(stmt.Result),
                    variable: stmt.Variable,
                }
            }
        }
    }
}
```

Then for each original block, iterate source lines and look up results by text:

```go
for _, block := range blocks {
    key := hashKey(block)
    lines := strings.Split(block, "\n")
    var lineResults []LineResult
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        lr := LineResult{Source: line, IsBlank: trimmed == ""}
        if r, ok := resultBySource[trimmed]; ok {
            lr.Result = r.result
            lr.Variable = r.variable
        }
        lineResults = append(lineResults, lr)
    }
    results[key] = BlockResult{Lines: lineResults}
}
```

It doesn't matter how many `CalcBlock`/`TextBlock` nodes the interpreter creates internally — results are matched back purely by content.

### Edge Cases

Duplicate source lines across blocks map to the same `resultBySource` entry. This is acceptable in progressive mode because all blocks share a single interpreter scope: identical expressions yield identical results. CalcMark variables are immutable, so the same expression always produces the same value within a document.

## Prevention

### Key Principle

**"Never assume 1:1 correspondence between input and output when a transformation can split or merge elements."**

Index-based mapping (`results[i]` corresponds to `sources[i]`) is a fragile contract requiring every intermediate transformation to preserve cardinality. Source-line text matching is self-describing — the mapping carries its own proof of correctness. This is the same principle behind the SHA-256 hash contract between doceval and Hugo: content-addressed, not position-addressed.

### Testing

Test cases that catch this class of regression:

1. **Mixed text+calc block** — a block containing both prose and calculations. The exact case that broke.
2. **Text-only block followed by calc-only block** — ensures non-calculable blocks don't consume result slots.
3. **Multiple mixed blocks in sequence** — confirms index drift doesn't accumulate.
4. **Empty block between calc blocks** — whitespace-only blocks shouldn't shift indices.
5. **Round-trip hash stability** — confirm doceval's hash matches Hugo's hash for mixed blocks.

## Key Files

| File | Purpose |
|------|---------|
| `cmd/doceval/main.go` | Go pre-evaluation tool (contains the fix) |
| `cmd/doceval/README.md` | Full documentation of the doceval system |
| `site/layouts/_default/_markup/render-codeblock-calcmark.html` | Hugo render hook (SHA-256 lookup side) |

## Related

- [Two-phase build system and Hugo hash mismatch](../infrastructure/hugo-doceval-inline-results-two-phase-build.md) — the original architecture doc covering SHA-256 contracts and Hugo template argument order
- [Hugo site structure](../infrastructure/hugo-site-structure-from-scratch.md) — foundational site setup
- [Document refresh skill](../../../.claude/skills/document-refresh/SKILL.md) — doceval conventions for documentation workflows
