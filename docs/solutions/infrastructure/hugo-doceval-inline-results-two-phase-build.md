---
title: "Hugo render hook hash mismatch due to reversed TrimRight argument order"
date: 2026-03-06
category: infrastructure
component: site/cmd/doceval
severity: high
symptoms:
  - Every cm code block renders with missing evaluation results
  - All Hugo-side SHA-256 hashes resolve to the hash of an empty string (e3b0c44298fc...)
  - cm_results.json contains correct hashes but Hugo never matches any of them
tags:
  - hugo
  - render-hook
  - sha256
  - doceval
  - two-phase-build
  - template-gotcha
  - frontmatter-config
---

# Two-Phase Hugo Inline Results with Hash-Based Lookup

## Problem

Hugo cannot execute shell commands or invoke interpreters from templates. The CalcMark documentation site has ` ```cm ` code blocks that need to display evaluated results inline. Hugo's render hook system can customize how code blocks render, but it has no way to actually *run* the code.

## Architecture

A two-phase build bridges the gap with SHA-256 hash-based lookup:

```
Phase 1: Go preprocessor (cmd/doceval)
  scan markdown → extract ```calcmark blocks → evaluate → write cm_results.json (keyed by SHA-256)

Phase 2: Hugo build
  render-codeblock-calcmark.html → compute same SHA-256 → look up results → render inline table
```

Both sides normalize content identically: (1) split on `\n`, (2) trim trailing whitespace per line, (3) trim the whole block, (4) SHA-256. The hash is the contract.

## Root Cause: Hugo's Reversed Argument Order

Hugo template functions take the "main" argument **last** (for pipe compatibility). This is the opposite of Go's stdlib:

| Context | Call | Meaning |
|---------|------|---------|
| Go stdlib | `strings.TrimRight(s, " \t")` | Trim cutset from s |
| Hugo template | `strings.TrimRight " \t" .` | Trim cutset from `.` (pipe target) |

The original template had:
```go-html-template
{{- $cleaned = $cleaned | append (strings.TrimRight . " \t") -}}
```

Hugo interpreted this as `TrimRight(" \t", .)` — trimming characters from the *cutset* string `" \t"` rather than from the actual line. Every line became empty, every hash was SHA-256 of `""`.

**The fix:**
```go-html-template
{{- $cleaned = $cleaned | append (strings.TrimRight " \t" .) -}}
```

### How we found it

Hugo's `--templateMetrics` confirmed the render hook fired 201 times. Adding `data-hash="{{ $hash }}"` to the HTML revealed every block had hash `e3b0c44298fc...` — the well-known SHA-256 of the empty string. Comparing against the JSON file keys made the mismatch obvious.

## Progressive vs Standalone Evaluation

A second problem emerged: interleaved worked examples define variables in early blocks and reference them in later blocks. Evaluating each block independently produces "undefined variable" errors.

**Solution:** `calcmark_build` Hugo frontmatter parameter.

```yaml
---
title: "Datacenter Build Cost"
calcmark_build: progressive
---
```

| Value | Behavior | Use for |
|-------|----------|---------|
| `standalone` (default) | Each block gets its own interpreter | Reference docs (reuse variable names) |
| `progressive` | All blocks share one interpreter | Worked examples (variables carry across) |

Progressive mode concatenates all blocks into one CalcMark document, evaluates once, then maps results back by statement index. It also extracts CalcMark frontmatter from ` ```yaml ` blocks containing `exchange:` or `globals:` keys.

**Errors in progressive pages fail the build.** Standalone errors are tolerated (blocks may demonstrate frontmatter features unavailable in isolation).

## Documentation Errors Surfaced

Evaluating all code blocks surfaced real documentation bugs:

- Exchange rate format shown as `USD/EUR` but CalcMark requires `USD_EUR`
- `pace = time / distance` (duration/quantity division) not supported
- `$12000 * 6 months` (currency * duration) not supported
- Syntax template blocks (e.g., `rate over duration`) tagged as ` ```cm ` instead of ` ```text `
- Variable redefinitions across independent sections in reference docs

These were all genuine errors invisible before the code was actually evaluated.

## Prevention

### Hugo template argument order

Hugo's pipe convention means **every** string function takes the operand last:

| Function | Hugo template order |
|---|---|
| `strings.TrimRight` | `cutset s` |
| `strings.TrimPrefix` | `prefix s` |
| `strings.HasPrefix` | `prefix s` |
| `strings.Replace` | `old new n s` |

**Always verify with a minimal template first.** Use `warnf` for debug output:
```go-html-template
{{ warnf "content length: %d" (len $normalized) }}
```

### Detect hash mismatches

The SHA-256 of an empty string is `e3b0c44298fc1c149afbf4c8996fb924...`. If you ever see this hash in rendered output, something is hashing empty input. Add an assertion:

```go
if normalized == "" && content != "" {
    log.Printf("WARNING: normalization emptied non-empty content")
}
```

### Treat doc code blocks as tests

Every ` ```cm ` block is now an executable test. The `task site:build` pipeline fails if progressive pages have evaluation errors. This catches documentation drift automatically.

## Key Files

| File | Purpose |
|------|---------|
| `cmd/doceval/main.go` | Go pre-evaluation tool |
| `cmd/doceval/README.md` | Full documentation of the system |
| `site/layouts/_default/_markup/render-codeblock-calcmark.html` | Hugo render hook |
| `site/assets/css/components.css` (lines 389+) | Result panel styling |
| `Taskfile.yml` (`site:build`) | Canonical build command |
| `.github/workflows/site.yml` | GitHub Pages deployment |

## Related

- [Hugo site structure](../infrastructure/hugo-site-structure-from-scratch.md)
- [Document refresh skill](../../.claude/skills/document-refresh/SKILL.md) — includes doceval conventions
