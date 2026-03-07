# doceval — CalcMark Documentation Evaluator

`doceval` pre-evaluates all ` ```calcmark ` fenced code blocks in the Hugo site's
markdown files and writes the results to `site/data/cm_results.json`. Hugo's
[render-codeblock-calcmark.html](../../site/layouts/_default/_markup/render-codeblock-calcmark.html)
render hook reads this file to display inline calculation results below each
code block.

## How It Works

```
┌─────────────┐     ┌─────────────┐     ┌──────────────────┐
│  Markdown   │────▶│   doceval   │────▶│ cm_results.json   │
│(```calcmark)│     │  (Go tool)  │     │ (SHA-256 keyed)   │
└─────────────┘     └─────────────┘     └────────┬─────────┘
                                                  │
                    ┌─────────────┐               ▼
                    │ Hugo build  │◀── render-codeblock-calcmark.html
                    │             │     looks up hash → results
                    └─────────────┘
```

1. `doceval` scans `site/content/**/*.md` for ` ```calcmark ` code blocks
2. Each block is evaluated through the CalcMark interpreter
3. Results are written to JSON, keyed by SHA-256 hash of the normalized block content
4. Hugo's render hook computes the same hash and looks up results at build time

## Build Modes: `calcmark_build`

Add `calcmark_build` to Hugo frontmatter to control how code blocks are evaluated.

### `standalone` (default)

Each ` ```calcmark ` block is evaluated independently — its own interpreter, its own
variable scope. Use this for reference documentation where different sections
demonstrate the same concepts with overlapping variable names.

```yaml
---
title: "Language Reference"
# calcmark_build defaults to standalone — no need to set it
---
```

**Errors in standalone blocks are tolerated.** Blocks that depend on frontmatter
features (exchange rates, globals) will fail in standalone mode because doceval
can't inject that context. These blocks display without inline results, which is
fine for documentation that shows the frontmatter separately.

### `progressive`

All ` ```calcmark ` blocks on the page share a single interpreter context. Variables
defined in an earlier block are visible to later blocks — just like a single
CalcMark document split across sections with prose in between.

```yaml
---
title: "Datacenter Build Cost"
calcmark_build: progressive
---
```

**Errors in progressive pages fail the build.** If any block in a progressive
page fails, the entire page is an error. This is intentional — progressive pages
are worked examples where every block should evaluate correctly.

## When to Use Each Mode

| Scenario | Mode | Why |
|----------|------|-----|
| Worked example with interleaved prose | `progressive` | Blocks build on each other |
| Language reference showing syntax | `standalone` | Sections are independent, may reuse names |
| Getting started / quickstart | `standalone` | Short independent snippets |
| Tutorial with building complexity | `progressive` | Each step depends on the previous |

## Gotchas

**Variable immutability across blocks.** In `progressive` mode, all blocks share
scope. CalcMark variables are immutable, so you cannot redefine a variable name
that was set in an earlier block. If you need to show the same concept with
different values, use distinct variable names.

**YAML frontmatter extraction.** In `progressive` mode, doceval also scans for
` ```yaml ` blocks containing CalcMark frontmatter keys (`exchange:`, `globals:`,
`scale:`, `convert_to:`) and prepends them as the document's frontmatter. This
lets progressive pages demonstrate frontmatter features in a separate yaml block
while still having the calcmark blocks use those values.

**Hash matching.** The SHA-256 hash must match between doceval (Go) and Hugo
(template). Both normalize by trimming trailing whitespace per line, then
trimming the whole block. If you see blocks without results, check for
whitespace differences (tabs vs spaces, trailing newlines).

## Running

```bash
# Standalone
go run ./cmd/doceval

# As part of the site build pipeline
task generate-docs   # runs docgen + doceval
task site:build      # runs generate-docs + hugo
```

## Output Format

`site/data/cm_results.json` maps SHA-256 hashes to block results:

```json
{
  "a1b2c3...": {
    "lines": [
      { "source": "price = $100", "result": "$100.00", "variable": "price" },
      { "source": "tax = price * 8.5%", "result": "$8.50", "variable": "tax" }
    ]
  }
}
```

Blocks that fail evaluation include an `error` field instead:

```json
{
  "d4e5f6...": {
    "lines": null,
    "error": "undefined_variable: undefined variable \"x\""
  }
}
```
