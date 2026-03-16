---
title: "Hugo guide pages rendering CalcMark evaluation errors"
category: build-errors
date: 2026-03-16
tags:
  - hugo
  - doceval
  - calcmark-build
  - progressive-mode
  - code-blocks
  - site-generation
component: site/content, cmd/doceval
severity: high
symptoms:
  - "Error blocks rendered on live calcmark.org guide pages"
  - "accumulate() first argument must be a rate"
  - "undefined_variable errors in guide walkthrough steps"
  - "@scale requires 'scale:' in frontmatter"
  - "Duplicate H1 headings on guide pages"
---

# Hugo Guide Pages Rendering CalcMark Evaluation Errors

## Problem

Multiple guide pages on calcmark.org displayed red error blocks instead of computed results. Affected pages: `/guides/system-sizing/`, `/guides/business-planning/`, `/guides/recipe-scaling/`, and `/docs/agent-integration/`. Additionally, 4 guides had duplicate H1 headings (Hugo renders `title:` from frontmatter, so `# Title` in the body doubled it).

## Root Cause

Three distinct issues:

1. **Missing `calcmark_build: progressive`**: Guide walkthroughs define variables in early code blocks and reference them in later blocks. Without `progressive` mode, `doceval` evaluates each `calcmark` block independently — variables don't carry across blocks. The `worked-example` archetype includes `calcmark_build: progressive`, but these guides were created without it.

2. **Wrong type for rate accumulation**: `peak_rps = 10K` creates a plain number. The `over` keyword (`avg_rps over 1 day`) requires a Rate type (e.g., `10K/s`). Multiplying a number by a scalar produces a number, not a rate.

3. **Undefined variable in standalone block**: `agent-integration.md` had a code example using `salary` without defining it in the same block.

## Solution

**PR:** https://github.com/CalcMark/go-calcmark/pull/68

- Added `calcmark_build: progressive` to frontmatter of system-sizing, business-planning, and recipe-scaling guides
- Changed `peak_rps = 10K` to `peak_rps = 10K/s` (rate, not number)
- Removed duplicate `# Title` H1 from 4 guides
- Changed recipe-scaling Step 3's `yaml` fence to `text` to prevent `extractCMFrontmatter` from extracting it in progressive mode (would cause double-frontmatter)
- Added `salary = $120000` to the agent-integration globals example

After fixes: `doceval` produces zero errors across all 279 blocks in 50 files.

## Prevention

- **Use the `worked-example` archetype** when creating new guide pages: `hugo new --kind worked-example guides/<name>/_index.md`. It includes `calcmark_build: progressive` by default.
- **Run `go run ./cmd/doceval` locally** before pushing site content changes. It reports all evaluation errors to stderr and exits non-zero if any exist.
- **When a guide walkthrough builds on prior steps**, it needs `calcmark_build: progressive`. If each block is self-contained, `standalone` (default) is fine.
- **Don't use `yaml` fences for illustrative frontmatter examples** in progressive-mode pages — `extractCMFrontmatter` will extract them. Use `text` fences instead.

## Related

- PR: https://github.com/CalcMark/go-calcmark/pull/68
- Archetype: `site/archetypes/worked-example.md`
- Render hook: `site/layouts/_default/_markup/render-codeblock-calcmark.html`
- Doceval tool: `cmd/doceval/main.go`
