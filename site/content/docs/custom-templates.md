---
title: "Custom HTML Templates"
summary: "Theme CalcMark HTML output with CSS variables, or build a fully custom page using shared template partials."
weight: 38
---

CalcMark ships shared template partials and CSS custom properties that make it easy to customize HTML output — from a one-line color override to a fully branded page with custom scripts.

## Quick Start: Override a CSS Variable

The fastest way to restyle CalcMark output is to override one or more `--cm-*` CSS variables. Create a template file (e.g., `my-theme.gohtml`):

```gotemplate
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>My CalcMark Doc</title>
<style>
{{.Style}}
:root { --cm-accent: purple; --cm-font-sans: 'Georgia', serif; }
</style>
</head>
<body><main>{{template "cm-content" .}}</main></body>
</html>
```

Then render with it:

```bash
cm convert budget.cm --to=html -T my-theme.gohtml -o budget.html
```

The `{{.Style}}` block injects CalcMark's built-in CSS (with all the structural layout rules), and your `:root` overrides change the accent color and font. All block rendering — calc results, diagnostics, errors, frontmatter — comes from the shared `cm-content` partial.

## Available CSS Variables

All theme values use the `--cm-*` namespace with sensible defaults. Override any of them in `:root` or a scoped selector.

| Variable | Default | Purpose |
|----------|---------|---------|
| `--cm-accent` | `#0066cc` | Primary brand color — calc block borders, result values, blockquote accents |
| `--cm-accent-light` | `#0550ae` | Lighter accent — frontmatter variable names |
| `--cm-text` | `#333` | Body text |
| `--cm-text-secondary` | `#57606a` | Muted text — frontmatter headings, blockquotes |
| `--cm-text-code` | `#24292e` | Code/monospace text |
| `--cm-text-muted` | `#6e7781` | Subtle text — frontmatter `@` prefix, exchange labels |
| `--cm-bg` | `#f8f9fa` | Calc block background |
| `--cm-bg-subtle` | `#f6f8fa` | Code block background |
| `--cm-bg-frontmatter` | `#f0f4f8` | Frontmatter section background |
| `--cm-border` | `#d0d7de` | Borders — tables, frontmatter |
| `--cm-error` | `#b91c1c` | Error text |
| `--cm-error-bg` | `#ffeef0` | Error background |
| `--cm-warning` | `#866100` | Warning text |
| `--cm-warning-bg` | `#fff8e1` | Warning background |
| `--cm-warning-border` | `#e6a817` | Warning left border |
| `--cm-cascading` | `#7c5c00` | Cascading error (hint severity) |
| `--cm-font-sans` | System sans-serif stack | Body font |
| `--cm-font-mono` | `SF Mono, Monaco, ...` | Code font |

## Dark Mode

To add dark mode support, override the color variables inside a media query or class selector:

```css
@media (prefers-color-scheme: dark) {
  :root {
    --cm-accent: #a78bfa;
    --cm-text: #e5e5e5;
    --cm-text-code: #e5e5e5;
    --cm-bg: #1e1e2e;
    --cm-bg-subtle: #313244;
    --cm-bg-frontmatter: #2a2a3e;
    --cm-border: #45475a;
    --cm-error: #f38ba8;
    --cm-error-bg: #302030;
  }
  body { background: #1e1e2e; }
}
```

For toggle-based dark mode (like a button in your UI), use a class or data attribute instead:

```css
[data-theme="dark"] {
  --cm-accent: #a78bfa;
  --cm-text: #e5e5e5;
  /* ... */
}
```

## Template Partials

Custom templates have access to shared partials that handle all the complex rendering logic. You never need to duplicate block iteration, diagnostic display, or frontmatter layout.

### Available Partials

| Partial | Purpose |
|---------|---------|
| `cm-content` | Renders frontmatter + all blocks. Most templates use just this. |
| `cm-frontmatter` | Frontmatter section only (globals, exchange rates, scale, etc.) |
| `cm-blocks` | All blocks — iterates and dispatches to `cm-calc-block` / `cm-text-block` |
| `cm-calc-block` | Single calculation block with per-line results, errors, and diagnostics |
| `cm-text-block` | Single text block (rendered Markdown) with warnings |

Call them via `{{template "cm-content" .}}` or use individual partials for layout control:

```gotemplate
<div class="sidebar">{{template "cm-frontmatter" .}}</div>
<div class="main">{{template "cm-blocks" .}}</div>
```

### Template Data Model

The data struct passed to every template:

| Field | Type | Description |
|-------|------|-------------|
| `.Style` | `template.CSS` | Full CalcMark CSS stylesheet |
| `.Frontmatter` | `*TemplateFrontmatter` | Frontmatter config (nil if none) |
| `.Blocks` | `[]TemplateBlock` | All document blocks |
| `.Content` | `template.HTML` | Embedded-mode pre-rendered HTML (empty in CM mode) |

Each `TemplateBlock` has `.Type` (`"calculation"` or `"text"`), `.SourceLines`, `.Error`, `.HasLineDiagnostic`, `.Warnings`, `.HTML`, and `.DocLine`. Each `TemplateLine` has `.Source`, `.Result`, `.Error`, `.IsCascading`, and `.DocLine`.

Use `cm convert --show-template` to see the default template as a starting point.

## Real-World Examples

### CalcMark Lark (Playground)

[CalcMark Lark](https://github.com/CalcMark/calcmark-lark) is the web-based playground at [lark.calcmark.org](https://lark.calcmark.org). Its template uses partials for CM-mode rendering and handles embedded mode with a separate branch:

```gotemplate
<style>
{{.Style}}
:root { --cm-accent: #7D56F4; --cm-font-sans: "Inter", system-ui, sans-serif; }
@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) {
    --cm-accent: #a371f7; --cm-bg: #0d1117; --cm-text: #e5e5e5; /* ... */
  }
}
</style>
<!-- ... -->
{{if .Content}}
<div class="embedded-content">{{.Content}}</div>
{{else}}
{{template "cm-content" .}}
{{end}}
```

The full template is ~75 lines — most of which is the embedded-content CSS. The CM-mode rendering is a single `{{template "cm-content" .}}` call.

### LSP Preview (Editor Webview)

The built-in preview template (`format.PreviewHTMLTemplate()`) is the simplest possible consumer — a content-only fragment with no page wrapper:

```gotemplate
{{template "cm-content" .}}
```

LSP clients (VS Code, etc.) provide their own HTML shell and inject the fragment. The `--cm-*` variables inherit from whatever theme the editor sets.

## Go API

For programmatic use, the `format` package exposes these accessors:

```go
import "github.com/CalcMark/go-calcmark/format"

format.StyleCSS()            // Raw CSS string (for injection via strings.Replace, etc.)
format.PartialsTemplate()    // Shared partials source (cm-content, cm-blocks, etc.)
format.DefaultHTMLTemplate() // Default full-page template
format.PreviewHTMLTemplate() // Content-only fragment template
```

To render with a custom template via the Go API:

```go
result, err := calcmark.Convert(source, calcmark.Options{
    Format:   "html",
    Template: myCustomTemplate, // Go html/template string
})
```

Partials are automatically available — your custom template can call `{{template "cm-content" .}}` without any extra setup.

## Further Reading

- [CLI Reference]({{< ref "docs/cli-reference" >}}) — `cm convert` flags including `--template` and `--show-template`
- [Go Package]({{< ref "docs/go-package" >}}) — programmatic API for rendering
- [CalcMark Lark source](https://github.com/CalcMark/calcmark-lark) — full working example
