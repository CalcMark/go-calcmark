# Hugo Site Structure from Scratch

---
problem_type: infrastructure
component: website
severity: medium
tags: [hugo, static-site, github-pages, dark-mode, css-architecture]
date_solved: 2026-02-22
---

## Problem

CalcMark needed a documentation website hosted at calcmark.org. The site had to be a Hugo static site built from scratch (no theme), deployed via GitHub Actions on push to main, with dark/light theming, and content migrated from existing repo docs.

## Symptoms

- No website existed; the GitHub repo README was the only landing page
- Documentation scattered across `docs/README.md`, `spec/LANGUAGE_SPEC.md`, `CONFIG.md`
- No automated deployment pipeline for a static site

## Root Cause

This was a greenfield infrastructure task, not a bug. The core challenge was structuring a Hugo site correctly within a monorepo, getting Hugo Pipes fingerprinting working, and avoiding common pitfalls with template context and asset processing.

## Investigation Steps

1. Explored existing repo docs to understand content scope (docs/README.md, CONFIG.md, spec/LANGUAGE_SPEC.md, 5 example .cm files)
2. Identified available assets (hero.gif, tui-screenshot.png)
3. Researched Hugo best practices for custom sites without themes
4. Identified key architectural decisions: `assets/` vs `static/`, template naming, Hugo Pipes

## Solution

### Directory Structure

```
site/
  hugo.yaml                    # YAML config (not TOML)
  content/                     # Markdown pages with front matter
    _index.md                  # Home page
    docs/
      _index.md                # Docs landing
      getting-started.md
      user-guide.md
      language-reference.md
      configuration.md
      examples/
        _index.md
        household-budget.md
        job-offer.md
        ...
  layouts/
    _default/
      baseof.html              # Base template with {{ block "main" }}
      home.html                # Home page (root _index.md)
      page.html                # Single content pages
      section.html             # Section listing pages
    partials/
      head.html                # Meta, no-FOUC script, Hugo Pipes CSS
      nav.html                 # Sticky top nav from menus.main
      sidebar.html             # Docs sidebar from menus.docs
      footer.html
      theme-toggle.html
    shortcodes/
      callout.html
      cm-example.html
  assets/                      # Processed by Hugo Pipes
    css/
      variables.css            # CSS custom properties (design tokens)
      base.css                 # Reset, typography
      components.css           # Nav, sidebar, hero, cards, responsive
      utilities.css            # .container, .visually-hidden, etc.
    js/
      theme.js                 # Dark/light toggle with localStorage
  static/                      # Copied verbatim (no processing)
    images/
      hero.gif
      tui-screenshot.png
    CNAME                      # Contains "calcmark.org"
```

### Key Decision: `assets/` vs `static/`

- **`assets/`** for CSS and JS: processed through Hugo Pipes with fingerprinting for cache-busting
- **`static/`** for images and CNAME: copied verbatim with no processing

In `head.html`, CSS is loaded via Hugo Pipes:

```html
{{ $vars := resources.Get "css/variables.css" }}
{{ $base := resources.Get "css/base.css" }}
{{ $components := resources.Get "css/components.css" }}
{{ $utils := resources.Get "css/utilities.css" }}
{{ $styles := slice $vars $base $components $utils | resources.Concat "css/site.css" | fingerprint }}
<link rel="stylesheet" href="{{ $styles.RelPermalink }}">
```

This concatenates all CSS into one file and generates a hashed filename like `site.abc123.css`.

### Key Decision: Template Naming

Following Hugo conventions from [bitsby.me Hugo guide](https://bitsby.me/2025/12/create-a-hugo-site-from-scratch/):

| Template | Maps to | Purpose |
|----------|---------|---------|
| `home.html` | Root `_index.md` | Landing page |
| `page.html` | Regular `.md` files | Single content pages |
| `section.html` | Directory `_index.md` files | Section listing pages |
| `baseof.html` | All pages | Base shell with `{{ block "main" }}` |

All templates use `{{ define "main" }}...{{ end }}` to inject into the base template.

### Key Decision: Menu-Driven Navigation

Navigation is defined in `hugo.yaml`, not hardcoded in templates:

```yaml
menus:
  main:          # Top nav bar
    - name: Docs
      url: /docs/
      weight: 10
  docs:          # Sidebar
    - name: Getting Started
      url: /docs/getting-started/
      weight: 10
```

Templates iterate over menus with active state detection:

```html
{{ range .Site.Menus.docs }}
<a class="sidebar-link{{ if or ($.IsMenuCurrent "docs" .) ($.HasMenuCurrent "docs" .) }} sidebar-link--active{{ end }}"
   href="{{ .URL }}">{{ .Name }}</a>
{{ end }}
```

**Critical:** Use `$.IsMenuCurrent` (page context via `$`), NOT `.IsMenuCurrent` (menu entry context inside `{{ range }}`). The latter causes `can't evaluate field HasMenuCurrent in type *navigation.MenuEntry`.

### Key Decision: Dark/Light Toggle Without FOUC

1. **Inline script in `<head>`** runs before render to apply stored theme:
```html
<script>
  (function() {
    var t = localStorage.getItem('theme');
    if (t) document.documentElement.setAttribute('data-theme', t);
    else if (window.matchMedia('(prefers-color-scheme: dark)').matches)
      document.documentElement.setAttribute('data-theme', 'dark');
  })();
</script>
```

2. **CSS custom properties** on `:root` (light) and `[data-theme="dark"]` (dark overrides)
3. **`theme.js`** handles toggle clicks, localStorage persistence, and OS preference changes
4. **Chroma syntax highlighting** uses CSS classes (`noClasses: false`) so dark/light switching works for code blocks

### Key Decision: GitHub Actions Deployment

```yaml
on:
  push:
    branches: [main]
    paths: ["site/**", ".github/workflows/site.yml"]
```

- Path-filtered: only deploys when site files change
- `fetch-depth: 0` required for `enableGitInfo: true` (git-based "last updated" dates)
- `concurrency` group prevents parallel deployments

### Hugo Config Settings

```yaml
baseURL: "https://calcmark.org/"
disableKinds: [taxonomy, term, RSS]    # Not needed for docs site
enableGitInfo: true                     # Auto "last updated" from git
markup:
  goldmark:
    renderer:
      unsafe: true                      # Some migrated docs have raw HTML
  highlight:
    noClasses: false                    # CSS classes for dark/light toggle
```

## Prevention Strategies

### Before Starting a Hugo Site

1. Decide `assets/` vs `static/` upfront — CSS/JS in `assets/` for fingerprinting, everything else in `static/`
2. Use YAML config format (`hugo.yaml`) for consistency with front matter
3. Plan menu structure in config rather than hardcoding navigation

### Common Pitfalls

- **`HasMenuCurrent` on wrong context**: Inside `{{ range .Site.Menus.foo }}`, `.` is the menu entry, not the page. Use `$` for page context.
- **Forgetting `fetch-depth: 0`**: `enableGitInfo` silently fails without full git history in CI
- **Inline styles for Chroma**: Default `noClasses: true` prevents dark mode syntax highlighting
- **No-FOUC**: Must be an inline `<script>` in `<head>`, not an external JS file

### Maintenance Checklist

- [ ] Test `hugo --source site` builds with zero errors after content changes
- [ ] Verify dark/light toggle works on new pages
- [ ] Check mobile responsive layout at 768px breakpoint
- [ ] Ensure all internal links use `{{</* ref "docs/page" */>}}` shortcodes
- [ ] CNAME file present in `site/static/CNAME`

## Related

- [Hugo documentation](https://gohugo.io/documentation/)
- [Hugo Pipes](https://gohugo.io/hugo-pipes/introduction/)
- [GitHub Pages deployment](https://docs.github.com/en/pages)
- Brainstorm doc: `docs/brainstorms/2026-02-22-calcmark-website-brainstorm.md`
