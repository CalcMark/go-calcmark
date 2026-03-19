---
date: 2026-03-18
topic: embedded-mode
---

# Embedded Mode for `cm convert`

## Problem Frame

CalcMark documents today must be `.cm` files where the entire document is CalcMark syntax. This limits where CalcMark can be used — authors writing standard markdown (blog posts, READMEs, Hugo content, reports) can't embed live calculations without switching to a fully CalcMark document. Embedded mode lets CalcMark act as a surgical preprocessor: evaluate only the calculation blocks, leave everything else untouched.

## Requirements

- R1. `cm convert --embedded <file>` processes a standard markdown file, finds fenced code blocks tagged `cm` or `calcmark` (backtick or tilde fences), evaluates each block as a self-contained CalcMark document, and replaces it with the evaluated markdown output.
- R2. `--embedded` is always explicit — file extension alone never triggers embedded mode.
- R3. Output format is always markdown. `--to md` is implied; other `--to` values are an error when `--embedded` is set.
- R4. Each `cm`/`calcmark` block is an independent, self-contained CalcMark document. No shared state, variables, or dependencies between blocks.
- R5. CalcMark frontmatter (`scale`, `convert_to`, etc.) lives inside individual `cm` blocks. The outer document frontmatter belongs to Hugo/the host system and is passed through unchanged.
- R6. All non-`cm`/`calcmark` content is passed through byte-for-byte unchanged — whitespace, frontmatter, HTML, GFM extensions, footnotes, everything.
- R7. Block output uses the existing markdown formatter (`format/markdown_formatter.go`) — the same output as `cm convert --to md` for a standalone `.cm` file.
- R8. On block evaluation error: replace the block with a visible inline error (blockquote format) and continue processing remaining blocks.
- R9. Exit code is non-zero if any block had errors. Output is always produced regardless of errors.
- R10. Block detection uses a simple line scanner (no markdown AST library). Supports both backtick (```) and tilde (`~~~`) fences per CommonMark spec.

## Success Criteria

- A standard markdown file with `cm`/`calcmark` fenced blocks round-trips through `cm convert --embedded` with only those blocks replaced.
- Can be wired into a Hugo build pipeline as a preprocessor (like D2 for diagrams).
- Blocks with errors produce visible inline errors without halting the pipeline.
- Non-CalcMark content is identical between input and output (byte-for-byte).

## Scope Boundaries

- No `--to html` or other output formats in embedded mode — pipe through pandoc/Hugo for that.
- No cross-block variable sharing or interpolation into surrounding markdown.
- No reading or interpreting the outer document frontmatter.
- No new markdown library dependency — line scanner only.
- No `{{var}}` interpolation in surrounding markdown text.

## Key Decisions

- **`convert` not `eval` or new subcommand**: `convert` already means "transform a document." `--embedded` is a mode flag on the existing command.
- **Line scanner over AST parser**: Fenced code blocks have trivial grammar. A scanner avoids roundtrip fidelity risks and keeps the dependency footprint zero. Goldmark can be added later if richer markdown parsing is ever needed.
- **Blocks are fully independent**: Simplest mental model. Each block is its own CalcMark document with its own frontmatter. No shared state to debug or reason about.
- **Outer frontmatter is opaque**: Clean separation — CalcMark never touches Hugo/Jekyll/whatever frontmatter. CalcMark config goes inside blocks.

## Outstanding Questions

### Deferred to Planning

- [Affects R10][Technical] Exact scanner behavior for edge cases: indented fences (CommonMark allows 0-3 spaces), fence length matching (opening fence of N backticks requires closing fence of >= N), and info string variations (e.g., `cm {.highlight}`).
- [Affects R8][Technical] Exact format of the inline error blockquote — should it include the original source, just the error message, or both?
- [Affects R7][Technical] Whether the markdown formatter needs any adaptation for embedded context (e.g., suppressing document-level wrappers if any exist).

## Next Steps

-> `/ce:plan` for structured implementation planning
