---
title: "refactor: Simplify HTML template authoring with shared partials and CSS custom properties"
type: refactor
status: active
date: 2026-04-06
origin: docs/brainstorms/2026-04-06-simplify-html-template-authoring-requirements.md
---

# Simplify HTML Template Authoring

## Overview

Extract shared Go template partials from `default.gohtml` and `preview.gohtml` to eliminate ~65 lines of duplicated block-rendering logic, and replace hardcoded CSS values with `--cm-*` custom properties so consumers can theme CalcMark output by overriding a handful of variables.

## Problem Frame

Three template consumers — `default.gohtml` (full HTML page), `preview.gohtml` (LSP/editor fragment), and external custom templates (like the Lark playground) — all duplicate the same block iteration, diagnostic rendering, and frontmatter layout logic. When core rendering changes (as during #112 error recovery), every template must be updated in lockstep. CSS has the same problem: all colors and fonts are hardcoded, forcing consumers to copy the entire stylesheet to change branding.

(see origin: `docs/brainstorms/2026-04-06-simplify-html-template-authoring-requirements.md`)

## Requirements Trace

- R1–R6. Go template partials with granular composition (`cm-content`, `cm-frontmatter`, `cm-blocks`)
- R7–R9. CSS structural/theme split via `--cm-*` custom properties
- R10. Light/dark theme switching via variable overrides
- R11. Default theme ships as variable defaults — visually unchanged
- R12–R13. Backwards compatibility: same HTML output, same data model
- R14. Zero-config defaults — no overrides needed for production-ready output

## Scope Boundaries

- **In scope:** `default.gohtml`, `preview.gohtml`, `calcmark.css`, `HTMLFormatter.Format()` parsing logic, `--show-template` flag. Improve the default HTML output to be beautiful, legible, and a11y-compliant. Use the Lark purple theme as a real-world test case to validate the new templating structure.
- **Out of scope:** Embedded mode (`wrapEmbeddedHTML` in `convert.go` uses `.Content` not `.Blocks` — partials don't apply). Template data model changes. Built-in dark theme. `watch.go`'s `strings.Replace` CSS injection (it consumes `StyleCSS()` which is the raw CSS string — unaffected by template partials).

## Context & Research

### Relevant Code and Patterns

- `format/html_formatter.go:125` — Current template parsing: `template.New("html").Parse(templateContent)`
- `format/templates/default.gohtml` — 80 lines, full HTML document
- `format/templates/preview.gohtml` — 65 lines, content-only fragment
- `format/templates/calcmark.css` — 322 lines, three sections: calc blocks, text blocks/frontmatter, embedded `#content` rules
- `cmd/calcmark/cmd/convert.go:57` — `--template` and `--show-template` flag definitions
- `lsp/server.go:275` — LSP uses `PreviewHTMLTemplate()` directly
- `cmd/calcmark/cmd/watch.go:331` — Watch page uses `strings.Replace` with `format.StyleCSS()`
- `site/assets/css/variables.css` — Existing CSS custom properties pattern (`:root` + `[data-theme="dark"]`) in the Hugo site

### Institutional Learnings

- `template.CSS` type is mandatory when passing CSS into Go templates — plain `string` triggers `ZgotmplZ` escaping (from `docs/solutions/integration-issues/cm-watch-html-templating-css-and-live-reload.md`)
- Never inline CSS in `.gohtml` files — all styles go in embedded `.css` files via `go:embed`
- Inside `{{range}}` blocks, `.` rebinds to the range element; use `$` for outer context
- `watch.go` uses `strings.Replace`, not Go templates — CSS changes must work for both consumption paths

## Key Technical Decisions

- **Composition model: `{{template}}` (call-only), not `{{block}}` (overridable):** Custom templates compose by calling partials, not overriding them. If Lark wants different frontmatter layout, it simply doesn't call `{{template "cm-frontmatter" .}}`. This is simpler, avoids parse-order issues, and matches the stated requirements.

- **Partial name prefix: `cm-`:** All partial names use a `cm-` prefix (`cm-content`, `cm-frontmatter`, `cm-blocks`, `cm-calc-block`, `cm-text-block`) to avoid collisions with names custom templates may define. Go `html/template` uses last-definition-wins within a parse set, so unprefixed names like "content" could silently override.

- **Partials as a separate embedded file:** A single `format/templates/partials.gohtml` file containing all `{{define}}` blocks, embedded via `//go:embed` alongside the existing template files. Composed at parse time by parsing partials first, then the page template into the same template set.

- **`data-source-line` always emitted:** `preview.gohtml` adds `data-source-line` on `.calc-block`; `default.gohtml` doesn't. The unified partial always emits it — harmless in standalone HTML and enables scroll sync for any consumer.

- **`{{.Style}}` stays as single `template.CSS` field:** Structural and theme CSS are concatenated into one stylesheet. Consumers override via CSS custom properties, not by selectively including CSS files. Backwards-compatible — no data model change.

- **CSS custom properties follow Hugo site pattern:** `:root` block with `--cm-*` defaults, overridable via `@media (prefers-color-scheme: dark)` or a `.dark` class. Follow the proven pattern from `site/assets/css/variables.css`.

## Open Questions

### Resolved During Planning

- **How do partials physically ship?** Separate `//go:embed` file (`partials.gohtml`), parsed into the template set before the page template. Multi-file parse via `tmpl.Parse(partials); tmpl.Parse(pageTemplate)`.
- **What about `--show-template`?** Output the default page shell (thin wrapper) with a comment listing the available `cm-*` partials and the data model fields. Self-contained enough to be useful as a starting point.
- **What about the watch server?** Unaffected. `watch.go` consumes `format.StyleCSS()` (raw CSS string) via `strings.Replace`. Template partials don't touch this path. CSS variable extraction works because `StyleCSS()` returns the full CSS content.

### Deferred to Implementation

- Exact set of `--cm-*` variable names — will be determined by auditing `calcmark.css` during implementation
- Whether `--show-template` should embed the partial definitions inline or just reference them — depends on what feels most useful after seeing the actual output

## High-Level Technical Design

> *This illustrates the intended approach and is directional guidance for review, not implementation specification. The implementing agent should treat it as context, not code to reproduce.*

```
┌────────────────────────────────────────────────���─────┐
│  partials.gohtml (shared, embedded)                  │
│  ┌──────────────────────────────────────────────┐    │
│  │ {{define "cm-frontmatter"}} ... {{end}}      │    │
│  │ {{define "cm-calc-block"}}  ... {{end}}      │    │
│  │ {{define "cm-text-block"}}  ... {{end}}      │    │
│  │ {{define "cm-blocks"}}                       │    │
│  │   {{range .Blocks}}                          │    │
│  │     {{if eq .Type "calculation"}}            │    │
│  │       {{template "cm-calc-block" .}}         │    │
│  │     {{else}}                                 │    │
│  │       {{template "cm-text-block" .}}         │    │
│  │     {{end}}                                  │    │
│  │   {{end}}                                    │    │
│  │ {{end}}                                      │    │
│  │ {{define "cm-content"}}                      │    │
│  │   {{template "cm-frontmatter" .}}            │    │
│  │   {{template "cm-blocks" .}}                 │    │
│  │ {{end}}                                      │    │
│  └──────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────┘

┌─ default.gohtml ─────────┐  ┌─ preview.gohtml ──┐  ┌─ lark.gohtml (external) ──────┐
│ <!DOCTYPE html>           │  │ {{template        │  │ <!DOCTYPE html>                │
│ <html><head>              │  │   "cm-content" .}}│  │ <html><head>                   │
│   <style>{{.Style}}</style│  │                   │  │   <style>                      │
│ </head><body>             │  └───────────────────┘  │     {{.Style}}                 │
│   {{template              │                         │     :root { --cm-accent: teal } │
│     "cm-content" .}}      │                         │   </style>                     │
│ </body></html>            │                         │   <script src="lark.js"></script│
│                           │                         │ </head><body>                   │
└───────────────────────────┘                         │   {{template "cm-content" .}}   │
                                                      │ </body></html>                  │
                                                      ��────────────────────────────────┘
```

```
calcmark.css (after refactor):

:root {
  --cm-accent: #0066cc;
  --cm-error: #d73a49;
  --cm-warning: #b08800;
  --cm-bg: #f8f9fa;
  --cm-text: #333;
  --cm-font-mono: 'SF Mono', Monaco, ... monospace;
  --cm-font-sans: -apple-system, ... sans-serif;
  /* ... */
}

.calc-block {
  border-left: 4px solid var(--cm-accent);  /* structural uses variable */
  background: var(--cm-bg);
}
```

## Implementation Units

- [ ] **Unit 1: Create partials and refactor internal templates**

**Goal:** Extract shared rendering logic into `partials.gohtml` and refactor both internal templates to call partials, eliminating all duplicated block-rendering code.

**Requirements:** R1, R2, R3, R6, R12, R14

**Dependencies:** None

**Files:**
- Create: `format/templates/partials.gohtml`
- Modify: `format/templates/default.gohtml`
- Modify: `format/templates/preview.gohtml`
- Modify: `format/html_formatter.go`
- Test: `format/html_formatter_test.go`

**Approach:**
- Create `partials.gohtml` with `{{define "cm-frontmatter"}}`, `{{define "cm-calc-block"}}`, `{{define "cm-text-block"}}`, `{{define "cm-blocks"}}`, `{{define "cm-content"}}`. Move all block iteration, diagnostic rendering, and frontmatter layout from existing templates into these definitions.
- Always emit `data-source-line` on `.calc-block` div (unifying the default/preview divergence).
- Refactor `default.gohtml` to thin HTML shell (~15 lines) calling `{{template "cm-content" .}}`.
- Refactor `preview.gohtml` to a one-liner calling `{{template "cm-content" .}}`.
- Update `HTMLFormatter.Format()` to embed `partials.gohtml` via `//go:embed` and parse it first into the template set, then parse the page template (default or custom) into the same set. Use `tmpl, _ := template.New("html").Parse(partialsContent)` then `tmpl.Parse(pageTemplateContent)`.
- Add `PartialsTemplate()` public accessor alongside existing `DefaultHTMLTemplate()` and `PreviewHTMLTemplate()`.

**Patterns to follow:**
- Existing `//go:embed` pattern in `format/html_formatter.go:31-38`
- `template.CSS` typing for any CSS injection (institutional learning)

**Test scenarios:**
- Happy path: `cm convert --to html` produces identical output to current default template (golden comparison)
- Happy path: Preview template produces identical fragment output (golden comparison)
- Happy path: `data-source-line` attribute present on `.calc-block` divs in both default and preview output
- Edge case: Empty document (no blocks, no frontmatter) renders without error in both templates
- Edge case: Document with only text blocks (no calc blocks) renders correctly
- Edge case: Document with diagnostics, cascading errors, and warnings renders all diagnostic HTML
- Integration: `ZgotmplZ` does not appear in any rendered output (template.CSS safety)

**Verification:**
- `task test` passes with no regressions in `format/html_formatter_test.go`
- `default.gohtml` is <20 lines; `preview.gohtml` is <5 lines
- Zero duplicated block iteration or diagnostic logic between any template files

---

- [ ] **Unit 2: Update custom template parsing to support partials**

**Goal:** Custom templates passed via `--template` get partials automatically, enabling the <15-line custom template story.

**Requirements:** R4, R5, R6, R12, R13

**Dependencies:** Unit 1

**Files:**
- Modify: `format/html_formatter.go` (already changed in Unit 1 — this extends the parsing)
- Modify: `cmd/calcmark/cmd/convert.go` (`--show-template` output)
- Test: `format/html_formatter_test.go`
- Test: `cmd/calcmark/cmd/convert_test.go` (if custom template CLI tests exist)

**Approach:**
- The parsing change from Unit 1 (parse partials first, then page template) already makes partials available to custom templates. Verify this works end-to-end with a custom template that calls `{{template "cm-content" .}}`.
- Update `--show-template` to output the default shell with a comment block documenting available `cm-*` partials and their data model expectations. This becomes the "starting point for custom templates" per existing CLI help text.
- Existing custom templates that don't use partials continue to work — they define their own rendering inline and the partials are simply unused in the template set.

**Patterns to follow:**
- Existing `resolveTemplate()` in `cmd/calcmark/cmd/convert.go`
- `DefaultHTMLTemplate()` accessor pattern

**Test scenarios:**
- Happy path: Custom template calling `{{template "cm-content" .}}` produces correct HTML with frontmatter and blocks
- Happy path: Custom template calling individual partials (`cm-frontmatter`, `cm-blocks`) separately produces correct output
- Happy path: Legacy custom template (full inline rendering, no partial calls) still works unchanged
- Edge case: Custom template defining `{{define "cm-frontmatter"}}` overrides the partial (last-definition-wins) — verify this does not break but produces the custom version
- Happy path: `--show-template` outputs a valid template that renders correctly when passed back via `--template`

**Verification:**
- A custom template with just a wrapper + `{{template "cm-content" .}}` produces full CalcMark HTML
- Existing custom template tests pass unchanged

---

- [ ] **Unit 3: Extract CSS custom properties**

**Goal:** Replace hardcoded color, font, and spacing values in `calcmark.css` with `--cm-*` CSS custom properties, enabling theming via variable overrides.

**Requirements:** R7, R8, R9, R10, R11, R14

**Dependencies:** Unit 1 (templates must work before changing CSS)

**Files:**
- Modify: `format/templates/calcmark.css`
- Test: `format/html_formatter_test.go`
- Test: `cmd/calcmark/cmd/watch_test.go`

**Approach:**
- Add `:root` block at top of `calcmark.css` defining all `--cm-*` variables with current hardcoded values as defaults.
- Replace hardcoded colors (`#0066cc`, `#d73a49`, `#333`, `#f8f9fa`, etc.), font stacks, and key spacing values with `var(--cm-*)` references throughout the file.
- Follow the Hugo site pattern from `site/assets/css/variables.css` for variable naming.
- Group variables semantically: accent/brand, error/warning states, backgrounds, text colors, font families.
- Keep `{{.Style}}` as single `template.CSS` field — no data model change.
- Preserve both CM-mode (`.calc-block`, `.text-block`) and embedded-mode (`#content`) selector scopes.

**Patterns to follow:**
- `site/assets/css/variables.css` — `:root` with light defaults, `[data-theme="dark"]` override structure
- Current `calcmark.css` organization (three sections)

**Test scenarios:**
- Happy path: `cm convert --to html` output includes `:root` block with `--cm-*` variable definitions
- Happy path: All `.calc-block`, `.calc-error`, `.calc-warning`, `.frontmatter` rules use `var(--cm-*)` references instead of hardcoded values
- Happy path: Watch page CSS (from `format.StyleCSS()`) includes the `:root` variables — `watch_test.go` assertions still pass
- Integration: Rendered HTML is visually identical (same computed values) — golden test comparison confirms no unintended changes beyond the variable indirection
- Edge case: `#content` scoped embedded-mode rules also use the same `--cm-*` variables

**Verification:**
- `task test` passes
- No hardcoded color hex values remain in `calcmark.css` (except within `:root` defaults)
- A consumer can override `--cm-accent` and see it reflected in calc-block borders, inline results, and blockquote accents

---

- [ ] **Unit 4: End-to-end verification and documentation**

**Goal:** Verify all three consumer paths work correctly and update any relevant documentation.

**Requirements:** R12, R14

**Dependencies:** Units 1–3

**Files:**
- Modify: `cmd/calcmark/cmd/convert.go` (help text update if needed)
- Test: `format/html_formatter_test.go` (integration test)
- Test: `cmd/calcmark/cmd/watch_test.go`

**Approach:**
- Write an integration test that exercises the full pipeline: CalcMark document with frontmatter, calc blocks, text blocks, diagnostics, and warnings → `cm convert --to html` → verify partials produced correct structure.
- Verify LSP preview path: `PreviewHTMLTemplate()` produces content fragment with partials.
- Verify watch path: `format.StyleCSS()` returns CSS with `--cm-*` variables; watch page shell still works.
- Run `task quality` for full quality gate.

**Test scenarios:**
- Integration: Full document with all block types → HTML output matches expected structure
- Integration: LSP preview fragment contains all expected diagnostic HTML
- Integration: Watch page CSS contains `--cm-*` variables and expected class rules
- Happy path: `task quality` passes clean

**Verification:**
- `task test` and `task quality` both pass
- All three consumer paths (default HTML, preview fragment, watch page) produce correct output

---

- [ ] **Unit 5: Accessibility and design polish for default HTML**

**Goal:** Ensure the default CalcMark HTML output is beautiful, legible, and a11y-compliant — not just functional.

**Requirements:** R11, R14

**Dependencies:** Unit 3 (CSS variables in place)

**Files:**
- Modify: `format/templates/calcmark.css`
- Modify: `format/templates/default.gohtml` (semantic HTML improvements)
- Modify: `format/templates/partials.gohtml` (ARIA attributes if needed)
- Test: `format/html_formatter_test.go`

**Approach:**
- Audit color contrast ratios against WCAG 2.1 AA (4.5:1 for normal text, 3:1 for large text). Adjust `--cm-*` variable defaults if any fail.
- Ensure semantic HTML: use `<main>`, `<section>`, appropriate heading hierarchy, `role` attributes where meaningful (e.g., `role="alert"` for errors).
- Ensure error/warning/diagnostic text is not communicated by color alone — existing `✗` and `⚠` prefixes help, verify they're present in all diagnostic paths.
- Improve typography: comfortable line length (already `max-width: 900px`), adequate spacing, readable font sizes.
- Verify focus styles exist for any interactive elements (links in rendered markdown).

**Test scenarios:**
- Happy path: Default HTML output includes semantic elements (`<main>`, appropriate heading levels)
- Happy path: Error diagnostics include text indicators (`✗`, `⚠`) not just color
- Edge case: Color contrast of `--cm-error` against error background meets WCAG AA
- Edge case: Color contrast of `--cm-warning` against warning background meets WCAG AA

**Verification:**
- `task test` passes
- Manual review of rendered HTML confirms improved readability and a11y basics

---

- [ ] **Unit 6: Lark purple theme as validation test case**

**Goal:** Create a Lark-style custom template that exercises the new partials and CSS variable system, proving the <15-line custom template story works in practice.

**Requirements:** R4, R6, R8, R10, R14

**Dependencies:** Units 1–5

**Files:**
- Create: `format/templates/testdata/lark-theme.gohtml` (test fixture, not shipped)
- Create: `format/templates/testdata/lark-theme.css` (test fixture — CSS variable overrides only)
- Test: `format/html_formatter_test.go`

**Approach:**
- Create a Lark-style custom template that provides its own `<html>/<head>/<body>` shell, injects a `<script>` tag, overrides `--cm-accent` to Lark's purple, changes the font stack, and calls `{{template "cm-content" .}}` for all rendering.
- The template + CSS overrides combined should be <15 lines of custom code.
- Test that the rendered output has Lark's purple accent in the CSS variable definitions, the custom script tag, and all structural CalcMark HTML from the partials.
- Add a light/dark theme test: override `--cm-*` variables under `.dark` class, verify the CSS includes both theme definitions.

**Test scenarios:**
- Happy path: Lark template produces valid HTML with custom accent color, custom script, and all CalcMark structural content from partials
- Happy path: Lark template is <15 lines of custom code (wrapper + variable overrides)
- Happy path: Light/dark CSS variable override under `.dark` class works — both sets of variables present in output
- Integration: Lark template with a full CalcMark document (frontmatter, calc blocks, text blocks, diagnostics) renders all content correctly

**Verification:**
- The Lark test template demonstrates the complete custom template story end-to-end
- Template is concise — validates the "no duplication" requirement

## System-Wide Impact

- **Interaction graph:** `HTMLFormatter.Format()` is called by CLI convert, LSP server, watch server (via `Convert()` API), and the public `calcmark.Convert()` function. All paths go through the same formatter — changing parsing once affects all consumers.
- **Error propagation:** Template parse errors from malformed custom templates should surface clearly. The two-phase parse (partials then page template) means a parse error in the custom template still returns a clear error — no change to error behavior.
- **State lifecycle risks:** None — templates are stateless, parsed fresh per render.
- **API surface parity:** `DefaultHTMLTemplate()`, `PreviewHTMLTemplate()`, and `StyleCSS()` public accessors continue to work. New `PartialsTemplate()` accessor added.
- **Unchanged invariants:** The template data model (`TemplateBlock`, `TemplateLine`, `TemplateFrontmatter`), the `Options.Template` field contract, and the `convert.go` embedded-mode path (`wrapEmbeddedHTML`) are all unchanged.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Custom template namespace collision (defines a name like `cm-content`) | `cm-` prefix makes accidental collision unlikely. Last-definition-wins means intentional override still works. |
| Partials parse order matters in Go templates | Unit 1 tests verify parsing works. Partials always parsed first. |
| CSS variable extraction changes computed values | Golden test comparison catches any visual regression. Variables default to current hardcoded values. |
| `--show-template` output becomes confusing with partials | Include a clear comment block explaining the partial model. |
| `wrapEmbeddedHTML` in `convert.go` also parses templates | Explicitly out of scope — embedded mode uses `.Content`, not `.Blocks`. Partials are not parsed for embedded mode. |

## Sources & References

- **Origin document:** [docs/brainstorms/2026-04-06-simplify-html-template-authoring-requirements.md](docs/brainstorms/2026-04-06-simplify-html-template-authoring-requirements.md)
- Related issue: #114
- Related PR (error recovery): #112
- Institutional learning: `docs/solutions/integration-issues/cm-watch-html-templating-css-and-live-reload.md`
- Institutional learning: `docs/solutions/infrastructure/hugo-site-structure-from-scratch.md`
- CSS variables pattern: `site/assets/css/variables.css`
