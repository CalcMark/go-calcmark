---
title: "feat: Embedded mode for cm convert"
type: feat
status: completed
date: 2026-03-18
origin: docs/brainstorms/2026-03-18-embedded-mode-requirements.md
---

# feat: Embedded mode for cm convert

## Overview

Add `--embedded` flag to `cm convert` that processes standard markdown files, evaluates `cm`/`calcmark` fenced code blocks as independent CalcMark documents, and replaces them with evaluated markdown output. Everything else passes through byte-for-byte. This lets CalcMark act as a Hugo/static-site preprocessor — like D2 for diagrams, but for calculations.

## Problem Statement / Motivation

CalcMark documents today must be `.cm` files. Authors writing standard markdown (blog posts, Hugo content, reports) cannot embed live calculations without switching to a fully CalcMark document. Embedded mode lets CalcMark be wired into any markdown pipeline as a surgical preprocessor. (see origin: `docs/brainstorms/2026-03-18-embedded-mode-requirements.md`)

## Proposed Solution

A new `--embedded` boolean flag on the `convert` command. When set:

1. Read the input file (relaxed extension validation to accept `.md`/`.markdown`)
2. Line-scan for `cm`/`calcmark` fenced code blocks (backtick and tilde fences)
3. For each matched block: parse as standalone CalcMark doc, evaluate, format with `MarkdownFormatter`
4. Splice evaluated output back, replacing the original fenced block (opening fence through closing fence)
5. Pass all non-cm content through byte-for-byte unchanged
6. On block error: emit inline error blockquote, continue processing
7. Exit non-zero if any block errored; output is always produced

Output keeps ```` ```calcmark ```` fences — blocks render as styled code blocks in Hugo showing source + results with arrow annotations.

## Technical Considerations

### Architecture

The embedded pipeline is fundamentally different from the normal convert pipeline (which parses the whole file as CalcMark). It should be a separate function (`runConvertEmbedded`) called from the cobra `RunE`, not a modification to `runConvert`.

**Key layers:**

- **Scanner** (`impl/embedded/scanner.go`): Line-based fenced code block detection. Produces a sequence of `Segment` values — either passthrough text or extracted cm block content with byte offsets.
- **Evaluator loop** (in `cmd/calcmark/cmd/convert.go` or `cmd/calcmark/cmd/embedded.go`): Iterates segments, evaluates cm blocks via `document.NewDocument()` + `implDoc.NewEvaluator()` + `MarkdownFormatter.Format()`, writes passthrough segments unchanged.
- **Error collection**: Tracks which blocks errored for the exit code.

### Existing reference: `cmd/doceval/main.go`

`extractCMBlocks()` (line 432-462) already scans markdown for calcmark fences. However it has limitations the new scanner must fix:
- Only matches `calcmark`, not `cm` (see origin R1)
- Only backtick fences, no tildes (see origin R10)
- Uses `strings.TrimSpace` which accepts any indentation (CommonMark allows 0-3 spaces only)
- Does not match fence lengths (opening ``` of 4+ backticks needs 4+ backtick closer)
- Does not track byte offsets for splicing

The new scanner should be a shared package that doceval can eventually migrate to. (Deferred: actual doceval migration is out of scope for this plan.)

### Security: Extension validation

`security.go:63-67` enforces `.cm`/`.calcmark` extensions via `filecheck.IsCalcMarkExtension()`. Embedded mode needs `.md`/`.markdown`. Approach: add `filecheck.IsMarkdownExtension()` and call it when `--embedded` is set, bypassing the CalcMark extension check. Binary content validation (`validateFileContent`) still applies. File size limit (1MB) still applies.

### Markdown formatter reuse

`MarkdownFormatter` is stateless and can be instantiated per-block. Per-block pipeline:

```
blockSource string
  → document.NewDocument(blockSource)
  → implDoc.NewEvaluator().Evaluate(doc)
  → MarkdownFormatter.Format(&buf, doc, opts)
  → buf.String() replaces original fenced block
```

Frontmatter from cm blocks should NOT appear in output (it's CalcMark config, not content). Use `FrontmatterAsCodeFence: false` and suppress frontmatter serialization — add a new `Options.SuppressFrontmatter` flag to the formatter, or strip frontmatter from the buffer output.

### Learnings to apply

- **Content-addressed mapping, not index-based** (from `docs/solutions/logic-errors/doceval-progressive-index-block-splitting-misalignment.md`): The CalcMark parser can split a single source block into multiple internal blocks. Since we format the entire evaluated document per-block, this is handled naturally — the formatter iterates all blocks in the evaluated doc.
- **DocLine offsets** (from `docs/solutions/code-organization/docline-diagnostic-line-numbers.md`): Error messages should reference the line number in the host markdown file. Compute offset = host file line of the fence opener + 1 (for the content start).
- **Cobra flag pattern** (from `docs/solutions/code-organization/custom-help-hardcoding-flags.md`): Register with `cmd.Flags().BoolVar()`. Cobra auto-includes in help. Write a test asserting flag presence.
- **Test behavior, not implementation** (from `docs/solutions/test-failures/test-behavior-not-implementation.md`): Integration tests should pipe real markdown and assert on output content.

## System-Wide Impact

- **Interaction graph**: `--embedded` flag → `runConvertEmbedded()` → scanner → per-block `NewDocument` + `Evaluate` + `Format` → stdout/file. No callbacks, no middleware, no side effects beyond stdout.
- **Error propagation**: Block-level errors are caught and converted to inline blockquotes. The function collects error count and returns a non-nil error at the end if any blocks failed, which cobra translates to exit code 1.
- **State lifecycle risks**: None — each block gets a fresh `NewDocument` + `NewEvaluator`. No shared state between blocks (see origin R4).
- **API surface parity**: Only affects CLI. No TUI, no library API changes beyond the new scanner package and formatter option.
- **Integration test scenarios**: (1) Markdown with mixed cm/calcmark blocks and prose → correct output. (2) Block with error → inline error + non-zero exit. (3) No cm blocks → identical pass-through. (4) Hugo frontmatter preserved byte-for-byte.

## Acceptance Criteria

### Functional Requirements

- [ ] `cm convert --embedded file.md` processes markdown, evaluates `cm`/`calcmark` fenced blocks, outputs markdown with blocks replaced
- [ ] Both backtick (```) and tilde (`~~~`) fences recognized, with correct fence-length matching
- [ ] Info string matching: first token must be exactly `cm` or `calcmark` (case-sensitive); `cmake`, `cm-extra` etc. do not match
- [ ] 0-3 spaces of leading indentation accepted per CommonMark; 4+ spaces is not a fence
- [ ] `--embedded` is explicit — file extension alone never triggers it
- [ ] `--to` is implied as `md`; `--to md` accepted redundantly; other `--to` values error
- [ ] `--template` errors when combined with `--embedded`
- [ ] `--show-template` still works independently (no file needed)
- [ ] Each block evaluated independently — no shared variables or state between blocks
- [ ] Block-level CalcMark frontmatter (scale, convert_to) works within each block but does not appear in output
- [ ] Outer document frontmatter (Hugo YAML/TOML) passed through unchanged
- [ ] All non-cm content passed through byte-for-byte (whitespace, line endings, HTML, GFM, footnotes)
- [ ] Block errors produce inline blockquote: `> **CalcMark Error:** <message>` with host-file line number
- [ ] Exit code 1 if any block had errors; output always produced
- [ ] Unclosed cm fence: pass through unchanged (not evaluated)
- [ ] Empty cm block: evaluate normally (produces empty output)
- [ ] `.md` and `.markdown` extensions accepted when `--embedded` is set
- [ ] Binary content validation still applies
- [ ] File size limit (1MB) still applies
- [ ] Help output includes `--embedded` flag description

### Non-Functional Requirements

- [ ] Scanner is O(n) single-pass over the input
- [ ] No new dependencies (line scanner, no markdown AST library)

### Quality Gates

- [ ] Unit tests for scanner: backtick/tilde fences, fence lengths, indentation, info strings, unclosed fences, nested fences, empty blocks
- [ ] Unit tests for per-block evaluation: valid blocks, error blocks, blocks with frontmatter
- [ ] Integration tests: build binary, run `cm convert --embedded` with real markdown files, assert output
- [ ] Flag interaction tests: `--embedded` with `--to`, `--template`, `--show-template`
- [ ] Pass-through fidelity test: file with no cm blocks → output identical to input
- [ ] `task test` and `task quality` pass

## Implementation Phases

### Phase 1: Scanner package (`impl/embedded/`)

**Deliverables:**
- `impl/embedded/scanner.go`: `Scan(input string) []Segment` function
- `impl/embedded/scanner_test.go`: Comprehensive test suite
- `Segment` type: either `Passthrough{Text string}` or `Block{Content string, OpenLine int, FenceStyle string}`

**Test cases (TDD — write tests first):**
- Backtick fence with `cm` info string
- Backtick fence with `calcmark` info string
- Tilde fence with `cm` and `calcmark`
- Fence with 4+ characters (must match on close)
- 0, 1, 2, 3 spaces indentation (accepted)
- 4+ spaces indentation (not a fence, passthrough)
- Info string `cmake`, `cm-extra`, `CM` (rejected — passthrough)
- Info string `cm {.highlight}`, `calcmark title="foo"` (accepted)
- Unclosed fence (passthrough)
- Empty block (accepted)
- Nested fences (tilde opener with backtick inside — backtick is not a closer)
- Multiple blocks interleaved with prose
- No blocks at all (everything is passthrough)
- Document with only a single block, no surrounding content

### Phase 2: Formatter adaptation

**Deliverables:**
- Add `SuppressFrontmatter bool` to `format.Options`
- Update `MarkdownFormatter.Format()` to skip frontmatter serialization when set
- Unit test for the new option

This is a small, surgical change — one `if` guard around the existing frontmatter block (markdown_formatter.go:23-41).

### Phase 3: Security extension

**Deliverables:**
- `filecheck.IsMarkdownExtension(path)` — returns true for `.md`, `.markdown`
- Update `security.go` or add `validateReadFilePathEmbedded()` that uses the markdown extension check
- Unit tests for extension validation

### Phase 4: CLI wiring + embedded pipeline

**Deliverables:**
- `convertEmbedded bool` flag registered in `init()`
- `runConvertEmbedded(filename string) error` function:
  1. Validate flag interactions (`--to`, `--template`)
  2. Validate file path (markdown extension check)
  3. Read file + validate content
  4. Scan for blocks
  5. For each segment: passthrough → write directly; block → evaluate + format → write output
  6. Collect errors, return aggregate error if any blocks failed
- Integration tests via `buildCM(t)` + `exec.Command` pattern (from `pipe_test.go`)
- Help text updated with `--embedded` example

### Phase 5: Error formatting + line offsets

**Deliverables:**
- Error blockquote generation: `> **CalcMark Error:** <message> (line <N>)` where N is the host-file line number
- Per-block line offset computation: scanner's `Block.OpenLine` + 1
- Tests for error output format and line number accuracy

## Scope Boundaries (from origin)

- No `--to html` or other output formats in embedded mode
- No cross-block variable sharing or `{{var}}` interpolation into surrounding markdown
- No reading or interpreting the outer document frontmatter
- No new markdown library dependency
- No stdin support (matches current `convert` behavior; can be added later)
- No doceval migration to the new scanner (follow-up work)
- No bold arrow/result styling (separate enhancement to markdown formatter)

## Dependencies & Risks

- **Risk: Formatter frontmatter leakage.** If `SuppressFrontmatter` is not implemented, block-level CalcMark frontmatter will appear in output. Mitigated by Phase 2.
- **Risk: Extension validation bypass.** Must ensure the embedded path still validates content and size. Mitigated by using the same `validateFileContent` and size check.
- **Dependency: `filecheck` package.** Extension check logic lives here; adding markdown extensions is straightforward.

## Success Metrics

- A Hugo blog post with `cm`/`calcmark` blocks can be preprocessed with a single `cm convert --embedded` command
- Output is valid markdown that Hugo renders correctly
- Non-cm content is byte-for-byte identical between input and output
- Errors are visible in the rendered page without breaking the build pipeline

## Sources & References

### Origin

- **Origin document:** [docs/brainstorms/2026-03-18-embedded-mode-requirements.md](docs/brainstorms/2026-03-18-embedded-mode-requirements.md) — Key decisions: `convert` command with explicit `--embedded` flag, line scanner (no AST library), independent blocks, outer frontmatter opaque, markdown-only output, inline errors with non-zero exit

### Internal References

- Existing block extraction: `cmd/doceval/main.go:432-462` (`extractCMBlocks`)
- Per-block evaluation pattern: `cmd/doceval/main.go:353-387` (`evalBlockIndependent`)
- Convert command: `cmd/calcmark/cmd/convert.go`
- Markdown formatter: `format/markdown_formatter.go`
- Security validation: `cmd/calcmark/cmd/security.go` + `cmd/calcmark/filecheck/`
- Extension check: `cmd/calcmark/filecheck/extensions.go`
- Fence detection reference: `spec/document/detector.go:617-659` (`getFenceMarker`, `isMatchingCloseFence`)
- Integration test pattern: `cmd/calcmark/cmd/pipe_test.go`

### Learnings Applied

- Content-addressed mapping: `docs/solutions/logic-errors/doceval-progressive-index-block-splitting-misalignment.md`
- DocLine offsets for diagnostics: `docs/solutions/code-organization/docline-diagnostic-line-numbers.md`
- Cobra flag registration: `docs/solutions/code-organization/custom-help-hardcoding-flags.md`
- Test behavior not implementation: `docs/solutions/test-failures/test-behavior-not-implementation.md`
