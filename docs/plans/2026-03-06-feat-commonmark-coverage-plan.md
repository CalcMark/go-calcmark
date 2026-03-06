---
title: "feat: CommonMark coverage for HTML and Markdown export"
type: feat
status: active
date: 2026-03-06
issue: https://github.com/CalcMark/go-calcmark/issues/33
brainstorm: docs/brainstorms/2026-03-06-commonmark-coverage-brainstorm.md
---

# feat: CommonMark Coverage for HTML and Markdown Export

## Overview

Bring CalcMark's HTML and Markdown export to full CommonMark compliance. The line detector has 6 known classification gaps, the gomarkdown renderer has 2 security vulnerabilities (raw HTML passthrough, unsafe link schemes), and test coverage for markdown features is nearly nonexistent beyond headings and bold text.

This plan follows a TDD approach: write failing tests first, then fix the code.

## Problem Statement

CalcMark documents blend CommonMark markdown with calculations. The line detector (`spec/document/detector.go`) classifies each line as markdown or calculation, but it only recognizes a subset of CommonMark constructs. Lines it doesn't recognize fall through to the CalcMark parser, where they may be silently misclassified — causing fenced code blocks to execute, indented code to run as calculations, and reference-style link definitions to break.

Additionally, the gomarkdown renderer passes raw HTML through without sanitization, creating an XSS vector in HTML export.

## Proposed Solution

A phased, test-first approach across 5 phases:

1. **Security hardening** — Fix XSS vulnerabilities in gomarkdown config and HTML formatter
2. **Detector tests** — Write failing tests for all known detector gaps
3. **Detector fixes** — Add missing markdown patterns and a minimal fenced-code-block state machine
4. **Realistic test documents** — Three domain-specific `.cm` files exercising CommonMark features with calculations
5. **Formatter fixes + focused tests** — Fix rendering bugs found by the realistic documents, add per-feature edge case tests

## Technical Approach

### Architecture

The rendering pipeline:

```
.cm source → detector (line classification) → blocks (TextBlock/CalcBlock)
  → formatter (HTML or Markdown) → output
```

**Key files:**

| File | Role |
|------|------|
| `spec/document/detector.go` | Line classification: `isMarkdownPattern()`, `IsCalculation()`, `DetectBlocks()` |
| `spec/document/block.go` | TextBlock/CalcBlock types, per-block rendering |
| `spec/document/markdown.go` | gomarkdown configuration, `renderMarkdown()` |
| `format/html_formatter.go` | HTML export, template injection via `template.HTML()` |
| `format/markdown_formatter.go` | Markdown export, TextBlock passthrough |

**Constraint:** TextBlocks render independently. Reference-style link definitions in a different TextBlock from their usage will not resolve. This is a known limitation to document, not fix (would require rearchitecting the block system).

### Implementation Phases

#### Phase 1: Security Hardening

**Goal:** Close XSS vulnerabilities before writing any test documents.

- [x] Set `html.SkipHTML` on the gomarkdown HTML renderer in `spec/document/markdown.go`
- [x] Add `html.Safelink` to renderer flags to block `javascript:`, `vbscript:`, `data:` URI schemes
- [x] Replace `parser.CommonExtensions` with explicit CommonMark-only flags: remove `Tables`, `Strikethrough`, `DefinitionLists`; keep `FencedCode`, `Autolink`, `NoIntraEmphasis`, `SpaceHeadings`, `HeadingIDs`, `BackslashLineBreak`
- [x] Add `html.EscapeString()` to the `<br>` fallback path in `format/html_formatter.go:146`
- [x] Write test: raw HTML in a TextBlock is stripped from HTML export output
- [x] Write test: `javascript:` links are neutralized in HTML export output
- [x] Write test: `<br>` fallback path escapes HTML entities
- [x] Write test: GFM table syntax renders as plain text (not `<table>`)
- [x] Audit `testdata/` and `site/content/` for existing GFM table usage — if found, decide whether to keep `parser.Tables`
- [x] Run `task test` — all existing tests must pass

**Files:**
- `spec/document/markdown.go` — renderer flag changes, parser extension cleanup
- `spec/document/markdown_test.go` — security tests
- `format/html_formatter.go` — `<br>` fallback fix
- `format/html_formatter_test.go` — security tests, GFM removal tests

#### Phase 2: Detector Gap Tests (TDD — Red)

**Goal:** Write failing tests that expose every known detector gap.

Each test calls `detector.IsCalculation(line)` or `detector.DetectBlocks(source)` and asserts the correct classification. These tests MUST fail before Phase 3.

- [x] Test: `[wiki]: https://en.wikipedia.org` classified as markdown (not calculation)
- [x] Test: `===` after a text line keeps both lines in the same TextBlock (setext heading)
- [x] Test: `---` after a text line (no blank line between) keeps both lines in the same TextBlock (setext H2)
- [x] Test: `---` after a blank line is classified as markdown (horizontal rule, not setext heading)
- [x] Test: `***`, `___` standalone classified as markdown (horizontal rule)
- [x] Test: `` ``` `` and `~~~` classified as markdown (fenced code fence)
- [x] Test: Lines inside fenced code blocks stay in the same TextBlock regardless of content
- [x] Test: `    x = 10` (4-space indent) classified as markdown (indented code block)
- [x] Test: `+ First item` classified as markdown (unordered list with `+` marker)
- [x] Test: `![alt](src)` classified as markdown (image, starts with `!` not `[`)
- [x] Run `task test` — the new tests should fail, existing tests should pass

**Files:**
- `spec/document/detector_test.go` — new test cases

#### Phase 3: Detector Fixes (TDD — Green)

**Goal:** Make all Phase 2 tests pass by fixing `isMarkdownPattern()` and adding a minimal fenced-code-block state tracker to `DetectBlocks()`.

**3a. Pattern additions to `isMarkdownPattern()`:**

- [x] Reference-style link definition: `[` prefix + `]:` pattern (no `](` required)
- [x] Horizontal rules: `***` and `___` patterns (with optional spaces between chars, 3+ chars)
- [x] Note: `---` and `===` are handled by the setext/HR context tracker in 3c, NOT by `isMarkdownPattern()`
- [x] Fenced code fences: line starts with `` ``` `` or `~~~` (3+ chars)
- [x] `+` list marker: `+ ` (plus-space) prefix, similar to existing `- ` check
- [x] Image syntax: `!` prefix followed by `[` (catches `![alt](src)`)

**3b. Indented code block fix in `IsCalculation()`:**

- [x] Before `strings.TrimSpace(line)`, check if line starts with 4+ spaces or a tab
- [x] If so, return `false` (not a calculation) — this is an indented code block line

**3c. State tracking in `DetectBlocks()`:**

Fenced code block state machine:
- [x] Add `inFencedCodeBlock bool` and `fenceMarker string` state to block detection loop
- [x] When a fence line (`` ``` `` or `~~~`) is encountered and `!inFencedCodeBlock`, set `inFencedCodeBlock = true` and record the marker
- [x] When inside a fenced block, ALL lines are classified as text regardless of content
- [x] When the closing fence is encountered (matching marker), set `inFencedCodeBlock = false`
- [x] Lines inside fenced blocks accumulate into the current TextBlock

Setext heading / horizontal rule context:
- [x] Track `previousLineWasParagraph bool` — handled via isSetextUnderline() in isMarkdownPattern(), both setext underlines and horizontal rules are now recognized as markdown patterns
- [x] When `---` or `===` is encountered: classified as text via isSetextUnderline() or isHorizontalRule()
- [x] This avoids `---` being sent to the CalcMark parser where it could be misinterpreted as subtraction

- [x] Run `task test` — all tests pass (Phase 2 tests now green, existing tests still green)

**Files:**
- `spec/document/detector.go` — pattern additions, state machine
- `spec/document/detector_test.go` — verify Phase 2 tests now pass

#### Phase 4: Realistic Test Documents

**Goal:** Three domain-specific `.cm` files that mix diverse CommonMark features with calculations. These serve as integration tests for both formatters.

**4a. Engineering calculations document (`testdata/examples/markdown_engineering.cm`):**

CommonMark features: ATX headings (H1-H4), horizontal rules between sections, inline code for formulas, fenced code blocks showing example syntax, bold for emphasis on results, hard line breaks.

CalcMark features: Unit conversions, compound calculations, function calls.

**4b. Financial report document (`testdata/examples/markdown_financial.cm`):**

CommonMark features: Ordered and unordered lists (including nested), bold emphasis on totals, inline links to references, blockquotes for disclaimers, paragraphs with soft line breaks.

CalcMark features: Currency arithmetic, percentage calculations, variable assignments.

**4c. Scientific notes document (`testdata/examples/markdown_scientific.cm`):**

CommonMark features: Blockquotes for citations (including nested), fenced code blocks for data, images (inline), emphasis variants (`*italic*`, `**bold**`, `***bold italic***`), reference-style links (within same TextBlock), autolinks.

CalcMark features: Scientific notation, unit conversions, function chaining.

**4d. Formatter integration tests:**

- [x] Write test functions that process each `.cm` file through both HTML and Markdown formatters
- [x] HTML assertions use `strings.Contains()` for structural checks (consistent with existing test patterns in `html_formatter_test.go`): verify `<h1>`, `<h2>`, `<ul>`, `<ol>`, `<blockquote>`, `<code>`, `<pre>`, `<hr>`, `<em>`, `<strong>`, `<a href=`, `<img` tags are present where expected
- [x] HTML assertions verify NO raw HTML passthrough (no `<script>`, no `<div>` from source)
- [x] Markdown assertions verify TextBlock source lines appear verbatim in output
- [x] Markdown assertions verify calc blocks are wrapped in ` ```calcmark ` fences with results
- [x] Both formatters: verify calc blocks produce correct numeric results
- [x] Run `task test` — identify any rendering bugs

**4e. Regression safety net:**

- [x] Write a test that parses all existing `.cm` files in `testdata/` through `DetectBlocks()` and asserts block type counts match expected values (prevents detector changes from silently changing classification of existing documents)

**Files:**
- `testdata/examples/markdown_engineering.cm`
- `testdata/examples/markdown_financial.cm`
- `testdata/examples/markdown_scientific.cm`
- `format/html_formatter_test.go` — integration tests
- `format/markdown_formatter_test.go` — integration tests

#### Phase 5: Formatter Fixes + Edge Case Tests

**Goal:** Fix any rendering bugs found in Phase 4, then add per-feature focused tests for edge cases.

**5a. Fix rendering bugs** (specific bugs TBD based on Phase 4 results):

- [ ] Fix each bug with a targeted test that reproduces the failure first (TDD)
- [ ] Run `task test` after each fix

**5b. Edge case tests for high-risk features:**

- [ ] Test: Reference-style link within a single TextBlock resolves correctly in HTML
- [ ] Test: Reference-style link across TextBlock boundary does NOT resolve (known limitation)
- [ ] Test: Setext heading `---` after paragraph is H2, not horizontal rule
- [ ] Test: Standalone `---` after blank line is horizontal rule, not setext heading
- [ ] Test: CalcMark expression inside fenced code block is NOT executed
- [ ] Test: Indented code block with calc-like content is NOT executed
- [ ] Test: Mixed document: heading → calc → paragraph → calc → heading roundtrips through markdown formatter

- [ ] Run `task quality` — full quality gate pass

**Files:**
- Bug-specific files TBD
- `spec/document/detector_test.go` — edge case tests
- `format/html_formatter_test.go` — edge case tests
- `format/markdown_formatter_test.go` — edge case tests

## Acceptance Criteria

### Functional Requirements

- [ ] All CommonMark block-level constructs (ATX headings, setext headings, paragraphs, blockquotes, lists, fenced code blocks, indented code blocks, horizontal rules) are correctly classified by the detector
- [ ] All CommonMark inline-level constructs (emphasis, code spans, links, images, line breaks, autolinks) render correctly in HTML export
- [ ] Reference-style links work within a single TextBlock
- [ ] Fenced code blocks do NOT execute CalcMark expressions inside them
- [ ] Indented code blocks do NOT execute CalcMark expressions inside them
- [ ] Raw HTML is stripped from HTML export output
- [ ] Unsafe link schemes (`javascript:`, `vbscript:`, `data:`) are blocked
- [ ] Three realistic test documents pass both HTML and Markdown formatter tests
- [ ] Markdown export preserves TextBlock source lines verbatim

### Non-Functional Requirements

- [ ] All existing tests continue to pass (no regressions)
- [ ] `task quality` passes after all phases complete
- [ ] No new dependencies introduced (gomarkdown configuration changes only)

### Quality Gates

- [ ] `task test` passes after each phase
- [ ] `task quality` passes after Phase 5

## Known Limitations

These are intentional design decisions, not bugs:

1. **Reference-style links across TextBlock boundaries do not resolve.** Link definitions must be in the same contiguous markdown section as their usage. This is a consequence of per-block rendering architecture. Workaround: use inline links when markdown and calc blocks are interleaved.

2. **GFM extensions (tables, strikethrough, task lists) are not supported.** CommonMark only. Documents using GFM tables will see them render as plain text after this change. This is a **breaking behavioral change** for documents that relied on the previously-enabled GFM extensions.

3. **Multi-paragraph list items and lazy continuation lines may not work across block boundaries.** If a blank line within a list item triggers a block split (2 consecutive empty lines), the list structure will break.

## Dependencies & Risks

**Risk: GFM extension removal is a breaking change.**
Existing documents using markdown tables will break. Mitigation: audit `testdata/` and `site/content/` for table usage. If found, consider keeping `parser.Tables` enabled as a CalcMark-specific extension while removing other GFM features.

**Risk: Fenced code block state machine changes detector semantics.**
The detector moves from stateless per-line to stateful multi-line for fenced blocks. This is a fundamental change. Mitigation: comprehensive regression testing of all existing golden files.

**Risk: `SkipHTML` may strip content users relied on.**
If existing CalcMark documents used raw HTML (e.g., `<br>` or `<sub>`/`<sup>` for subscripts), setting `SkipHTML` will remove that content. Mitigation: audit existing test files for raw HTML usage.

## References & Research

### Internal References

- Brainstorm: `docs/brainstorms/2026-03-06-commonmark-coverage-brainstorm.md`
- Detector: `spec/document/detector.go` — `isMarkdownPattern()` at line ~366, `IsCalculation()` at line ~160
- Renderer: `spec/document/markdown.go` — `renderMarkdown()` with gomarkdown config
- HTML formatter: `format/html_formatter.go` — TextBlock rendering at line ~145
- Markdown formatter: `format/markdown_formatter.go` — TextBlock passthrough
- Block system: `spec/document/block.go` — TextBlock per-section rendering
- Existing mixed content test: `testdata/spec/valid/documents/mixed_content.cm`
- Block boundary test: `testdata/spec/valid/documents/block_boundaries.cm`
- Security model: `SECURITY.md`

### Institutional Learnings

- **Result alignment with blank lines** (`docs/solutions/ui-bugs/tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md`): Use parallel indices for result alignment — blank lines don't produce results.
- **Map ordering non-determinism** (`docs/solutions/logic-errors/go-maps-non-deterministic-ordering-frontmatter.md`): Never use Go maps for ordered collections. Golden file tests will be flaky if output ordering is non-deterministic.
- **WASM build tags** (`docs/solutions/integration-issues/remote-store-gist-plugin-architecture.md`): Gate `os/exec` or incompatible packages with `//go:build !wasm`.
