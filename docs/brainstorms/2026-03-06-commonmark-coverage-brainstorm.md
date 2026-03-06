# CommonMark Coverage for HTML and Markdown Export

**Date:** 2026-03-06
**Status:** Draft
**Issue:** https://github.com/CalcMark/go-calcmark/issues/33

## What We're Building

Comprehensive CommonMark compliance testing and bug-fixing for CalcMark's HTML and Markdown export formatters. The work ensures that all standard CommonMark features render correctly when a CalcMark document mixes markdown prose with calculations.

### Goals

1. **Realistic test documents** — Three domain-specific `.cm` files (engineering, financial, scientific) that exercise diverse CommonMark features alongside calculations
2. **Per-feature focused tests** — Extracted from failures found in the realistic documents, covering individual CommonMark features in isolation
3. **Bug fixes** — Any rendering bugs uncovered by the new tests get fixed in the HTML formatter, Markdown formatter, or block detection logic
4. **Multi-line block correctness** — Multi-line markdown constructs (nested lists, multi-paragraph blockquotes, fenced code blocks) MUST work correctly through the line detector and block system

### Non-Goals

- GFM extensions (tables, task lists, strikethrough, autolinks) — CommonMark only
- TUI editor rendering — out of scope for this effort
- Documentation/site updates — tests and fixes only
- Automated conformance suite import
- Raw HTML blocks — intentionally blocked for security. Tests should verify HTML is safely stripped/ignored, not rendered.
- Inline variable interpolation in markdown prose — future feature, not in scope

## Why This Approach

**Realistic documents first** because:
- Tests actual user workflows, not synthetic edge cases
- Naturally surfaces the most impactful bugs first
- Creates useful example files that double as documentation
- Three domains (engineering, financial, scientific) maximize feature coverage organically

Per-feature test extraction happens only where realistic documents reveal failures, keeping the test suite lean and purposeful.

## Key Decisions

1. **CommonMark only** — No GFM extensions. Keeps scope bounded and avoids gomarkdown extension configuration complexity.
2. **Both HTML and Markdown formatters** — Tests cover `format/html_formatter.go` and `format/markdown_formatter.go`. TUI rendering is out of scope.
3. **Fix bugs found** — This is not just a test-writing exercise. If the tests reveal rendering bugs in either formatter or in the block detector, those get fixed.
4. **Multi-line blocks must work** — The line detector (`spec/document/detector.go`) and block system must correctly handle multi-line markdown constructs. Fixing block detection logic is in-scope.
5. **Test structure: both** — Per-feature golden files for targeted coverage PLUS integration files that mix markdown with calculations.
6. **Raw HTML blocked** — Raw HTML blocks are stripped for security. Tests verify safe handling, not rendering.
7. **Three realistic documents:**
   - **Engineering calculations** — Headings, inline code, formulas with units, horizontal rules between sections
   - **Financial/budget report** — Lists (ordered + unordered), bold emphasis, nested sections, links
   - **Scientific notes** — Blockquotes for citations, fenced code blocks for data, images, emphasis variants

## CommonMark Features to Cover

Based on the [CommonMark spec](https://spec.commonmark.org/), these features need test coverage:

### Block-level
- Headings (ATX `#` and setext `===`/`---`)
- Paragraphs and blank lines
- Block quotes (single and nested)
- Ordered and unordered lists (including nested)
- Fenced code blocks (backtick and tilde)
- Indented code blocks
- Horizontal rules (`---`, `***`, `___`)
- HTML blocks — verify raw HTML is safely stripped (security requirement)

### Inline-level
- Emphasis (`*italic*`, `**bold**`, `***bold italic***`)
- Code spans (`` `inline code` ``)
- Inline links (`[text](url)` and `[text](url "title")`)
- Reference links (`[text][id]` with `[id]: url` defined elsewhere in the document)
- Images (`![alt](src)`) — both inline and reference-style
- Hard line breaks (trailing spaces, backslash)
- Soft line breaks
- Autolinks (`<url>`) — this is CommonMark, not GFM autolinks

### High-Risk Features (likely to interact badly with line detection)
- **Reference-style links** — The link definition `[id]: url` at the bottom of a document could be misclassified as a calculation (the `[` and `]` syntax, or the `:` assignment-like pattern). The definition and usage may also end up in separate TextBlocks, breaking the link resolution.
- **Multi-paragraph list items** — Continuation lines after blank lines within a list item
- **Nested blockquotes** — `> > nested` across multiple lines
- **Lazy continuation lines** — Paragraphs inside blockquotes that don't repeat the `>` marker
- **Link definitions interleaved with calculations** — A document with link definitions at the bottom after calc blocks
- **Setext headings vs horizontal rules** — Both use `---` syntax. A line of `---` after a paragraph is a setext heading (H2), but a standalone `---` after a blank line is a horizontal rule. The line detector must handle this context sensitivity correctly.

### Interactions with CalcMark
- Markdown headings followed by calculation lines
- Mixed blocks: markdown paragraph → calc block → markdown paragraph
- Blank line handling between markdown and calc blocks

## Architecture Impact

### Files likely affected
- `spec/document/detector.go` — if multi-line block detection has bugs
- `spec/document/block.go` — if TextBlock assembly is incorrect for complex markdown
- `spec/document/markdown.go` — if gomarkdown configuration needs adjustment
- `format/html_formatter.go` — HTML output bugs
- `format/html_formatter_test.go` — new test cases
- `format/markdown_formatter.go` — Markdown roundtrip bugs
- `format/markdown_formatter_test.go` — new test cases
- `testdata/` — new `.cm` golden files and expected outputs

### Rendering pipeline
```
.cm file → detector (line classification) → blocks (TextBlock/CalcBlock)
  → formatter (HTML or Markdown) → output
```

The critical correctness boundary is the **detector**: it must correctly classify multi-line markdown constructs as contiguous TextBlocks rather than splitting them or misidentifying lines as calculations.

## Risk Areas

Reference-style links are the highest-risk CommonMark feature for CalcMark. The line detector likely sees `[id]: https://example.com` as something other than markdown (the bracket/colon pattern resembles nothing in calc syntax but may confuse classification). Even if classified correctly, the link definition and its usage reference must end up in the same TextBlock (or the renderer must see both) for the link to resolve. This may require the block system to support document-wide link definitions rather than per-block rendering.

## Open Questions

None — all key decisions resolved during brainstorming.
