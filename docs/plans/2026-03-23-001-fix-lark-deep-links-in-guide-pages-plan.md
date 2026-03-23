---
title: "fix: Use /x/{slug} deep links for Lark example links in guide pages"
type: fix
status: completed
date: 2026-03-23
---

# fix: Use /x/{slug} deep links for Lark example links in guide pages

## Overview

Guide pages in `site/content/guides/` have "Try it now" links that claim to open a specific example in Lark, but they all point to the bare root `https://lark.calcmark.org` instead of the deep-link URL `https://lark.calcmark.org/x/{slug}`. Users click expecting to see the named example but land on the Lark homepage instead.

## Problem Statement

Four guide pages name a specific example but link to the wrong URL:

| Guide page | Link text | Current URL | Should be |
|---|---|---|---|
| `guides/system-sizing/_index.md:12` | "System Sizing example in Lark" | `https://lark.calcmark.org` | `https://lark.calcmark.org/x/system-sizing` |
| `guides/business-planning/_index.md:10` | "Household Budget in Lark" | `https://lark.calcmark.org` | `https://lark.calcmark.org/x/household-budget` |
| `guides/recipe-scaling/_index.md:14` | "Recipe Scaling example in Lark" | `https://lark.calcmark.org` | `https://lark.calcmark.org/x/recipe-scaling` |
| `guides/unit-conversion/_index.md:9` | "Unit Conversion example in Lark" | `https://lark.calcmark.org` | No `unit-conversion` example exists — keep root link |

Additionally, the Lark base URL (`https://lark.calcmark.org`) is hardcoded in 8+ locations across content and layouts with no centralized site param.

## Proposed Solution

### Part 1: Fix the deep links (required)

Update three guide pages to use `/x/{slug}` deep links where a matching `calcmark_source` example exists:

- `guides/system-sizing/_index.md` → `/x/system-sizing`
- `guides/business-planning/_index.md` → `/x/household-budget`
- `guides/recipe-scaling/_index.md` → `/x/recipe-scaling`

For `guides/unit-conversion/_index.md`: no `unit-conversion` worked example exists (no `calcmark_source` frontmatter), so keep the root Lark link but update the link text to not imply a specific example will open. Change to generic "CalcMark Lark playground" phrasing.

### Part 2: Create a `lark` shortcode (recommended)

Create `site/layouts/shortcodes/lark.html` that generates Lark links with optional deep-link slug:

```
{{</* lark */>}}                          → https://lark.calcmark.org
{{</* lark "system-sizing" */>}}          → https://lark.calcmark.org/x/system-sizing
```

This centralizes the base URL and makes `/x/` deep linking a first-class pattern. Convert the guide page links to use this shortcode.

### Part 3: Add `larkURL` site param (recommended)

Add `larkURL: "https://lark.calcmark.org"` to `hugo.yaml` params so all templates and shortcodes reference one source of truth. Update `section.html` and the new `lark` shortcode to use it.

## Acceptance Criteria

- [ ] `guides/system-sizing/_index.md` links to `https://lark.calcmark.org/x/system-sizing`
- [ ] `guides/business-planning/_index.md` links to `https://lark.calcmark.org/x/household-budget`
- [ ] `guides/recipe-scaling/_index.md` links to `https://lark.calcmark.org/x/recipe-scaling`
- [ ] `guides/unit-conversion/_index.md` link text does not imply a specific example will open
- [ ] New `lark` shortcode exists and supports optional slug parameter
- [ ] `hugo.yaml` has `larkURL` param used by shortcode and `section.html`
- [ ] `task site:build` succeeds (no broken shortcodes or template errors)
- [ ] Existing pages that link to bare `lark.calcmark.org` in docs/ are optionally converted to shortcode

## Context

- The `/x/{slug}` pattern is already used in `site/layouts/_default/section.html:30` for "Edit live" buttons on example pages
- The slug is derived from the `calcmark_source` filename: `path.BaseName "testdata/examples/system-sizing.cm"` → `system-sizing`
- 10 worked examples have `calcmark_source` and get deep links automatically via the section template
- The `guides/_index.md` root link is generic ("playground") and correct — no change needed
- `docs/_index.md`, `docs/getting-started.md`, `docs/examples/_index.md` link to root Lark appropriately — no change needed

## Sources

- `site/layouts/_default/section.html:26-33` — existing `/x/{slug}` pattern
- `site/layouts/shortcodes/` — 6 existing shortcodes (no Lark shortcode yet)
- `site/hugo.yaml` — site config, no `larkURL` param currently
- `docs/solutions/infrastructure/hugo-site-structure-from-scratch.md` — shortcode conventions
