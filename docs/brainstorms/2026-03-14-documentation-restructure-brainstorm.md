# Documentation Restructure Brainstorm

**Date:** 2026-03-14
**Status:** Complete

## What We're Building

A restructured documentation site that serves two audiences equally — developers (CLI, Go library, agent integration) and knowledge workers (Lark web playground, TUI editor for real calculations). The current site has good content but it's buried in 1000+ line monolithic pages, organized by implementation layer instead of by what users want to do.

**Three problems to solve:**
1. **Discoverability** — features are buried deep in long pages; you can't find measurement conventions, functions, or frontmatter without scrolling endlessly
2. **Missing content** — new features get a row in the auto-generated feature table but no real explanation or examples
3. **Wrong structure** — docs organized by code layer (frontmatter, functions, types) not by user intent (convert units, plan a budget, size a system)

## Why This Approach

### Three pillars: Docs, Examples, Guides

| Pillar | Purpose | Audience | URL |
|--------|---------|----------|-----|
| **Docs** | Reference material — how the language works | Both | `/docs/` |
| **Examples** | Worked problems — copy-paste-modify starting points | Both | `/docs/examples/` (existing) |
| **Guides** | Domain-specific tutorials — "System Sizing with CalcMark" | Knowledge workers + domain experts | `/guides/` (new top-level) |

Guides are the missing layer. Examples show *what* CalcMark can do. Guides teach *how to think* about a domain using CalcMark. Each guide includes its worked example as a capstone.

### User Guide → task-oriented sub-pages

Break the 1,219-line monolith into focused sub-pages using the same Hugo branch bundle pattern that examples already use:

```
docs/
├── _index.md                    (docs landing)
├── getting-started.md           (keep as-is, 140 lines, excellent)
├── user-guide/
│   ├── _index.md                (overview + links to sub-pages)
│   ├── units.md                 (unit conversion, measurement conventions)
│   ├── currency.md              (exchange rates, currency conversion)
│   ├── frontmatter.md           (globals, scale, convert_to, measurement)
│   ├── functions.md             (built-in functions, NL syntax)
│   ├── formatting.md            (as napkin, as precise, locale, display)
│   ├── templates.md             ({{var}} interpolation)
│   └── editor.md                (merge current editor.md here? or keep separate)
├── language-reference.md        (keep as one page, add TOC + better anchors)
├── cli-reference.md             (keep)
├── agent-integration.md         (keep)
├── go-package.md                (keep)
├── configuration.md             (keep)
└── examples/                    (keep existing structure)
```

Each sub-page is 100-300 lines — scannable, linkable, focused on one task.

### Language Reference → one page with TOC and anchors

Keep as a single page (spec/man-page style). Add:
- Hugo TOC at the top (auto-generated from headings)
- Explicit `{#anchor}` on every function, directive, and feature
- Better heading hierarchy so the TOC is useful
- Each function gets its own `###` heading (not just a row in a table)

This gives deep-linkable URLs like `/docs/language-reference/#measurement-conventions` and `/docs/language-reference/#number-function` that can be shared and bookmarked.

### Guides → new top-level section

Domain-specific tutorials that teach CalcMark through a real-world lens:

```
guides/
├── _index.md                    (guides landing page)
├── system-sizing/
│   ├── _index.md                (tutorial: capacity planning with CalcMark)
│   └── full.md                  (complete worked example)
├── business-planning/
│   ├── _index.md                (tutorial: P&L, budgets, financial modeling)
│   └── full.md                  (services P&L or budget example)
├── recipe-scaling/
│   ├── _index.md                (tutorial: measurement, fractions, scaling)
│   └── full.md                  (recipe example)
├── engineering-estimates/
│   ├── _index.md                (tutorial: napkin math, back-of-envelope)
│   └── full.md                  (datacenter cost or system sizing)
└── unit-conversion/
    ├── _index.md                (tutorial: measurement conventions, imperial/troy)
    └── full.md                  (measurement conventions example)
```

Guides link heavily to:
- Language Reference anchors (for formal definitions)
- User Guide sub-pages (for how-to details)
- Lark playground (for "try it now" links): `lark.calcmark.org`

### Navigation changes

Current:
```
Docs | Examples | Larky | GitHub
```

Proposed:
```
Docs | Guides | Examples | Larky | GitHub
```

Sidebar under Docs:
```
Getting Started
User Guide ▸           (expandable, sub-pages)
  Units & Measurement
  Currency & Exchange
  Frontmatter
  Functions
  Formatting & Display
  Templates
Language Reference
CLI Reference
Agent & API Integration
Go Package
Configuration
```

## Key Decisions

### 1. Three pillars, not two

Docs = reference, Examples = worked problems, Guides = domain tutorials. This matches how Stripe, Tailwind, and Next.js structure their docs. Knowledge workers land on Guides, developers land on Docs, both browse Examples.

### 2. User Guide becomes sub-pages; Language Reference stays one page

The user guide is *task-oriented* — users come for "how do I convert units?" Each task is a natural sub-page. The language reference is a *spec* — users come to look up exact behavior. One page with good anchors and a TOC serves spec-readers better than fragmented pages.

### 3. Hugo TOC for long pages

Use Hugo's built-in `{{ .TableOfContents }}` in the page layout for language-reference.md and any other page over ~300 lines. This is a template change, not a content change.

### 4. Every function gets a deep-linkable anchor

Instead of a feature-table row, each function in the language reference gets a `### function-name {#function-name}` heading. The feature table remains as an overview/index, but detail sections follow with examples.

### 5. Guides link to Lark for "try it now"

Each guide includes a "Try it in Lark" link that opens the example in the web playground. Knowledge workers who don't want to install anything can jump straight to `lark.calcmark.org`.

### 6. Redirects for moved content

When breaking the user guide into sub-pages, add Hugo aliases so old URLs (`/docs/user-guide/#units`) redirect to new locations (`/docs/user-guide/units/`). No broken links.

## Resolved Questions

1. **Audience**: Both developers and knowledge workers equally. Tone should be clear and practical, not academic.
2. **Structure**: Task-oriented sub-pages for user guide, one page with anchors for language reference, new top-level Guides section for domain tutorials.
3. **Language reference format**: Single page with TOC and `#anchors` for every feature. Good for Ctrl+F and deep linking.
4. **Domain guides location**: New top-level nav item alongside Docs and Examples.
5. **Knowledge worker entry point**: Lark playground (lark.calcmark.org) for browser-based users; guides link there for "try it now" experiences.
