---
title: "refactor: Restructure documentation into task-oriented sub-pages with Guides pillar"
type: refactor
status: completed
date: 2026-03-14
origin: docs/brainstorms/2026-03-14-documentation-restructure-brainstorm.md
---

# Documentation Restructure

## Overview

Restructure the CalcMark documentation site from two monolithic pages (user-guide.md at 1,219 lines, language-reference.md at 1,118 lines) into task-oriented sub-pages, add a Hugo TOC, and introduce a new top-level "Guides" section for domain-specific tutorials. The goal is discoverability — every feature should be findable, linkable, and useful to both developers and knowledge workers.

## Problem Statement

Three compounding problems (see brainstorm: docs/brainstorms/2026-03-14-documentation-restructure-brainstorm.md):

1. **Discoverability** — features like measurement conventions, `number()`, and frontmatter directives are buried in 1000+ line pages
2. **Missing content** — new features get an auto-generated feature-table row but no real explanation
3. **Wrong structure** — organized by implementation layer (frontmatter, functions, types) not by user intent (convert units, plan a budget, size a system)

## Proposed Solution

### Three pillars (see brainstorm: §Three pillars, not two)

| Pillar | Purpose | Audience | URL |
|--------|---------|----------|-----|
| **Docs** | Reference — how the language works | Both | `/docs/` |
| **Examples** | Worked problems — copy-paste-modify | Both | `/docs/examples/` |
| **Guides** | Domain tutorials — how to think about a problem | Knowledge workers + domain experts | `/guides/` |

### Implementation Phases

#### Phase 1: Hugo TOC and anchors on Language Reference

Add table of contents to long pages and ensure every function/feature has a deep-linkable `{#anchor}`.

**Tasks:**

- [ ] Add `{{ .TableOfContents }}` to the page layout template for docs pages (check if the theme's `single.html` or `page.html` supports it, or add to `baseof.html`)
- [ ] Audit `language-reference.md` headings — every function needs its own `### function-name {#function-name}` heading (not just a row in the feature table)
- [ ] Verify anchors work: `/docs/language-reference/#measurement-conventions`, `#number-function`, `#convert-to`, etc.
- [ ] Add a manual TOC at the top of `language-reference.md` if Hugo's auto TOC doesn't cover `###` level headings
- [ ] Test that shared links with `#anchors` scroll to the right section

**Files:**

| File | Change |
|------|--------|
| `site/layouts/_default/single.html` or `page.html` | Add `{{ .TableOfContents }}` |
| `site/content/docs/language-reference.md` | Add/fix `{#anchor}` on all headings |
| `site/assets/css/` | Style the TOC (sticky sidebar or top-of-page) |

**Success criteria:** Every function and directive has a shareable URL. TOC visible on language-reference page.

#### Phase 2: Break User Guide into sub-pages

Convert the monolithic user-guide.md into Hugo branch bundles following the pattern already used by examples.

**Tasks:**

- [ ] Create `site/content/docs/user-guide/` directory
- [ ] Create `site/content/docs/user-guide/_index.md` — overview page with links to sub-pages, keep Getting Started-style orientation content
- [ ] Extract and create `site/content/docs/user-guide/units.md` — unit conversion, measurement conventions, inline qualifiers, `in`/`as` keywords
- [ ] Extract and create `site/content/docs/user-guide/currency.md` — exchange rates, currency conversion, `in EUR` syntax
- [ ] Extract and create `site/content/docs/user-guide/frontmatter.md` — globals, scale, convert_to, measurement directive, `@scale`/`@globals` references
- [ ] Extract and create `site/content/docs/user-guide/functions.md` — built-in functions, NL syntax, function reference table
- [ ] Extract and create `site/content/docs/user-guide/formatting.md` — `as napkin`, `as precise`, locale, display formatting, `number()`
- [ ] Extract and create `site/content/docs/user-guide/templates.md` — `{{var}}` interpolation, forward references
- [ ] Add Hugo aliases on the old user-guide.md anchors so old URLs redirect: `aliases: ["/docs/user-guide/"]`
- [ ] Update `site/hugo.yaml` docs menu — replace single "User Guide" entry with expandable sub-menu or list the sub-pages individually
- [ ] Update all internal cross-references throughout the site (other docs pages, examples) that link to `user-guide.md#section`
- [ ] Each sub-page should be 100-300 lines, with a clear task focus and links to the Language Reference for formal definitions

**Files:**

| File | Change |
|------|--------|
| `site/content/docs/user-guide.md` | Delete (replaced by sub-pages) |
| `site/content/docs/user-guide/_index.md` | New — overview |
| `site/content/docs/user-guide/units.md` | New — units + measurement |
| `site/content/docs/user-guide/currency.md` | New — currency conversion |
| `site/content/docs/user-guide/frontmatter.md` | New — frontmatter directives |
| `site/content/docs/user-guide/functions.md` | New — functions + NL syntax |
| `site/content/docs/user-guide/formatting.md` | New — display + number() |
| `site/content/docs/user-guide/templates.md` | New — interpolation |
| `site/hugo.yaml` | Update menu structure |

**Success criteria:** Every user guide section is a separate URL. Old links redirect. Each page is scannable (100-300 lines).

#### Phase 3: Create Guides top-level section

Add domain-specific tutorial pages that teach CalcMark through real-world problem-solving.

**Tasks:**

- [ ] Create `site/content/guides/` directory with `_index.md` landing page
- [ ] Add "Guides" to Hugo main menu in `site/hugo.yaml` (between Docs and Examples, weight: 15)
- [ ] Create initial guides — each as a branch bundle with `_index.md` (tutorial) and optional `full.md` (complete worked example):
  - [ ] `guides/system-sizing/_index.md` — capacity planning with CalcMark (link to existing system-sizing example)
  - [ ] `guides/business-planning/_index.md` — P&L, budgets, financial modeling (link to services-pl and household-budget examples)
  - [ ] `guides/recipe-scaling/_index.md` — measurement conventions, fractions, scaling (link to recipe-scaling example, showcase `measurement:` frontmatter)
  - [ ] `guides/unit-conversion/_index.md` — measurement conventions, imperial/troy, inline qualifiers (new content, links to measurement-conventions example)
- [ ] Each guide includes "Try it in Lark" links to `lark.calcmark.org`
- [ ] Each guide links to Language Reference anchors for formal definitions
- [ ] Each guide links to relevant User Guide sub-pages for how-to details

**Files:**

| File | Change |
|------|--------|
| `site/content/guides/_index.md` | New — guides landing |
| `site/content/guides/system-sizing/_index.md` | New — capacity planning tutorial |
| `site/content/guides/business-planning/_index.md` | New — financial modeling tutorial |
| `site/content/guides/recipe-scaling/_index.md` | New — cooking + measurement tutorial |
| `site/content/guides/unit-conversion/_index.md` | New — measurement conventions tutorial |
| `site/hugo.yaml` | Add Guides to main menu |

**Success criteria:** Guides appear in top nav. Each guide teaches a domain concept using CalcMark, with links to reference docs and Lark.

#### Phase 4: Cross-linking and polish

Wire everything together so users can navigate between pillars.

**Tasks:**

- [ ] Add "Related Guides" links to relevant User Guide sub-pages (e.g., units.md links to recipe-scaling guide and unit-conversion guide)
- [ ] Add "Reference" links from Guides back to Language Reference anchors
- [ ] Add "Try in Lark" buttons/links where appropriate (guides, examples)
- [ ] Verify no broken internal links: `hugo --source site --minify` should produce no warnings
- [ ] Update getting-started.md "Next Steps" section to mention Guides
- [ ] Update docs landing page `_index.md` to describe the three-pillar structure

**Success criteria:** Every page has clear next-step links. No broken links. Three pillars are visible from the landing page.

## Navigation Structure

Current:
```
Docs | Examples | Larky | GitHub
```

Proposed:
```
Docs | Guides | Examples | Larky | GitHub
```

Docs sidebar:
```
Getting Started
User Guide ▸
  Units & Measurement
  Currency & Exchange
  Frontmatter
  Functions
  Formatting & Display
  Templates
The Editor
Language Reference
CLI Reference
Agent & API Integration
Go Package
Configuration
```

## Acceptance Criteria

### Phase 1 (TOC + anchors)
- [ ] Language Reference has a visible table of contents
- [ ] Every function/directive has a `#anchor` that works as a shareable URL
- [ ] `hugo --source site --minify` builds without warnings

### Phase 2 (User Guide sub-pages)
- [ ] User guide is split into 6-7 sub-pages, each 100-300 lines
- [ ] Old `/docs/user-guide/#section` URLs redirect to new sub-page URLs
- [ ] Hugo sidebar shows expandable User Guide with sub-pages
- [ ] All internal cross-references updated

### Phase 3 (Guides)
- [ ] "Guides" appears in top nav
- [ ] At least 4 domain guides published
- [ ] Each guide includes "Try in Lark" link
- [ ] Each guide links to Language Reference and User Guide

### Phase 4 (Cross-linking)
- [ ] No broken internal links
- [ ] Every sub-page has "Related" links to other pillars
- [ ] Getting started mentions Guides

## Dependencies & Risks

- **Phase 1 is independent** — can ship immediately
- **Phase 2 requires careful redirect handling** — old URLs must not break
- **Phase 3 depends on Phase 2** for the User Guide sub-page links
- **Risk: Hugo menu depth** — need to verify the current theme supports nested sidebar menus. If not, may need a layout partial change.
- **Risk: doceval** — splitting the user guide might affect how `doceval` finds and evaluates `calcmark` code fences. Test early.

## Sources & References

### Origin

- **Brainstorm document:** [docs/brainstorms/2026-03-14-documentation-restructure-brainstorm.md](docs/brainstorms/2026-03-14-documentation-restructure-brainstorm.md) — Key decisions: three pillars (Docs/Examples/Guides), user guide → sub-pages, language reference stays one page with TOC, Lark as "try it now" entry point

### Internal References

- Hugo config: `site/hugo.yaml`
- Example sub-page pattern: `site/content/docs/examples/household-budget/` (_index.md + full.md)
- Layout templates: `site/layouts/_default/page.html`, `site/layouts/partials/sidebar.html`
- Feature table shortcode: `site/layouts/shortcodes/feature-table.html`
- Current user guide: `site/content/docs/user-guide.md` (1,219 lines)
- Current language reference: `site/content/docs/language-reference.md` (1,118 lines)
