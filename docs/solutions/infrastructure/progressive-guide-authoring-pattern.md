---
title: "Progressive Guide Authoring Pattern: Understanding Measurements"
category: infrastructure
date: 2026-03-15
tags: [documentation, hugo, guides, tabs, brainstorm, workflow]
module: site/content/guides
severity: low
---

## Problem

Documentation for CalcMark's measurement system (quantities, unit conversions, fractions, napkin math, `convert_to`, measurement conventions) was scattered across 5+ pages. Users had to bounce between the units user guide, formatting guide, frontmatter reference, language reference, and configuration docs to build a mental model. No single progressive walkthrough existed.

## Process That Produced a High-Quality Result

The key insight: **brainstorming the narrative arc before writing any content** produced a dramatically better guide than jumping straight into docs authoring. The brainstorm-then-plan-then-build pipeline resolved critical design questions early:

### 1. Brainstorm phase surfaced the right questions

- **Placement**: New guide vs expanding existing docs vs tutorial section → decided on `guides/` section
- **Narrative arc**: Bottom-up vs scenario-driven vs FAQ-style → chose scenario-driven (kitchen/recipe)
- **Output format**: Code+JSON pairs vs screenshots vs tabbed views → chose tabbed (CalcMark/Editor/JSON)
- **Aha moment delivery**: Woven-in vs explicit comparison vs both → chose both

### 2. Verified JSON output before writing content

Running `echo '...' | ./cm --format json` for every example BEFORE writing the guide caught critical issues:
- `cups in grams` fails (volume ≠ mass) — plan had assumed it would work
- `as napkin` transforms `numeric_value` AND `unit` in JSON, not just display — brainstorm had characterized it as "display hint only"
- The three-way comparison (bare vs convert_to vs napkin) revealed the `value` field always normalizes display while `numeric_value + unit` preserves the original unless explicitly transformed

### 3. User feedback during review caught important issues

- **Duplicate H1**: Hugo template already renders title from frontmatter; the `# Title` in content was redundant
- **Horizontal rules**: `---` between sections was redundant since H2 headings already have `border-bottom` styling
- **Pronoun correction**: Larky uses they/them, not he/him
- **Custom units**: User suggested adding `salt = 1 pinch` and `yield = 24 cookies` to show units that are unaffected by `convert_to` — a gap the original outline missed
- **Discoverability**: Guide wasn't reachable from the home page sidebar — needed a `hugo.yaml` menu entry

## Solution: Reusable Tab Component

The CSS-only tab component built for this guide is reusable across the site:

**Shortcode usage:**
```markdown
{{</* tabs group="output" */>}}
{{</* tab name="CalcMark" */>}}
\`\`\`text
flour = 2 cups
\`\`\`
{{</* /tab */>}}
{{</* tab name="Editor" */>}}
...
{{</* /tab */>}}
{{</* /tabs */>}}
```

**Architecture:**
- `site/layouts/shortcodes/tabs.html` — parent shortcode, parses `data-tab-name` attributes from inner content via `findRE`
- `site/layouts/shortcodes/tab.html` — child shortcode, outputs panel div with `data-tab-name`, uses `markdownify` on inner content
- `site/assets/css/components.css` — CSS-only switching via hidden radio inputs + `:nth-of-type():checked ~ .tabs-panels > .tabs-panel:nth-child()` selectors (supports up to 4 tabs)
- `site/assets/js/tabs.js` — ~25 lines for localStorage persistence + same-group syncing across all tab instances on the page

**Key design decision:** Using `{{</* */>}}` (raw) delimiters + `markdownify` in the tab shortcode avoids triggering the `render-codeblock-calcmark` hook. This gives full control over what each tab panel shows — the CalcMark tab shows source only, the Editor tab shows source + results, the JSON tab shows machine-readable output.

## Prevention / Best Practices

1. **Always verify example output before writing docs**: Run `cm --format json` for every code example. Don't assume behavior — the actual output often differs from expectations in subtle ways.
2. **Remove `# Title` from content when frontmatter has `title:`**: Hugo page templates render the title automatically. Adding it in content creates duplicate H1s.
3. **Check theme CSS before adding `---` separators**: Heading elements may already have borders/spacing that make horizontal rules redundant.
4. **Add sidebar menu entries for new pages**: Creating content under a section doesn't automatically add it to the sidebar — `hugo.yaml` `docs:` menu needs an explicit entry.
5. **Use the brainstorm → plan → build pipeline for non-trivial docs**: The brainstorm phase catches scope, audience, and structure issues that are expensive to fix after content is written.
