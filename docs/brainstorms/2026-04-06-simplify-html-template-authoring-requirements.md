---
date: 2026-04-06
topic: simplify-html-template-authoring
issue: 114
---

# Simplify Custom HTML Template Authoring

## Problem Frame

Custom HTML templates must duplicate ~80 lines of Go template logic from `default.gohtml` — block iteration, conditional error/result/diagnostic rendering, frontmatter layout. This duplication already exists internally: `default.gohtml` and `preview.gohtml` contain identical block-rendering logic. External consumers (like the Lark playground) create a third copy.

When core template structure changes (as happened during #112 error recovery), every template must be updated in lockstep or it silently breaks — missing errors, wrong layout, duplicate messages.

The CSS has the same problem: structural styles (calc-block layout, diagnostic rendering, error borders) are mixed with theme styles (colors, fonts, spacing) in one monolithic `calcmark.css`. Consumers who want different branding must either override everything or risk breaking diagnostic display.

## Requirements

### Go template partials

- R1. Extract shared rendering logic into named Go template partials: `frontmatter`, `calc-block`, `text-block`. Each partial owns its structural HTML (iteration, conditionals, error/diagnostic rendering).
- R2. `default.gohtml` becomes a thin shell: HTML document wrapper (`<html>`, `<head>`, `<body>`) that calls the shared partials via `{{template "name" .}}`.
- R3. `preview.gohtml` becomes a thin shell: content-only fragment (no document wrapper) that calls the same shared partials. LSP clients (VS Code, etc.) provide their own page shell and light/dark context — the preview template inherits theming via the `--cm-*` CSS variables set by the host editor.
- R4. Custom templates (like Lark) provide their own page shell — `<html>`, `<head>`, `<body>`, custom scripts, meta tags — and call the shared content partials for block rendering. No duplication of block iteration or diagnostic logic.
- R5. The `--template` flag continues to work. Custom templates are parsed together with the base partials so they can reference them.
- R6. Partials are granular enough that a custom template can compose them flexibly: render all content in one call (`{{template "content" .}}`), or call individual partials (`{{template "frontmatter" .}}`, `{{template "blocks" .}}`) for more control over placement within a custom page layout.

### CSS separation and theming

- R7. Split `calcmark.css` into structural CSS (layout, display semantics for calc-blocks, diagnostics, errors, warnings) and theme CSS (colors, fonts, spacing, border-radius).
- R8. All theme values use CSS custom properties (`--cm-*` namespace) so consumers can restyle CalcMark by overriding a handful of variables — no need to rewrite structural rules. This is the primary extension point for visual customization (covers ~50% of what the Lark template does today).
- R9. Use modern CSS throughout: custom properties, `flex`/`grid` layout (already present), logical properties where appropriate. No legacy hacks or vendor prefixes for evergreen browsers.
- R10. The `--cm-*` variable design must make light/dark theme switching trivial — a consumer like Lark should be able to swap themes by redefining the color variables under `@media (prefers-color-scheme: dark)` or a `.dark` class, with no structural CSS changes.
- R11. The default theme ships as the default variable values — existing output is visually unchanged.

### Zero-config defaults

- R14. All partials and CSS variables ship with production-ready defaults. A consumer who provides no overrides gets the current CalcMark look and feel — all rendering logic comes from the embedded partials and CSS. Customization is purely additive: override one CSS variable for a different font, or call individual partials for layout control, but neither is required.

### Backwards compatibility

- R12. `cm convert --to html` output is visually unchanged with no flags. Existing HTML output is not a breaking change.
- R13. The template data model (`TemplateBlock`, `TemplateLine`, `TemplateFrontmatter`, etc.) does not change. Custom templates that use the current data model continue to work.

## Non-goals

- Changing the template data model (`TemplateBlock`, `TemplateLine`, etc.). The `HTMLFormatter.Format()` Go code will change to parse custom templates together with partials (R5), but the data contract stays the same.
- Supporting non-HTML output formats in this change.
- Building a template marketplace or registry.
- Shipping a built-in dark theme for CalcMark's default output (separate concern) — but the CSS variable design must make it trivial for consumers to add one.

## Success Criteria

- `default.gohtml` and `preview.gohtml` share all block-rendering logic via partials — zero duplicated iteration/conditional code between them.
- A custom template that only wants different branding can be written in <15 lines (wrapper + CSS variables), not 80+.
- Changing diagnostic rendering (e.g., adding a new diagnostic severity) requires editing one partial, not multiple templates.
- `task test` passes. HTML output golden tests show no visual regressions.
