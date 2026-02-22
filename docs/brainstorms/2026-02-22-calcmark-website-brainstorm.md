# CalcMark Website Brainstorm

**Date:** 2026-02-22
**Status:** Draft

## What We're Building

A Hugo-based static website for CalcMark at **calcmark.org**, hosted on GitHub Pages with automatic deployment on push to main. The site serves as both a landing page to attract new users and the primary documentation hub for the project.

**Target audience:** Software engineers, system architects, data engineers, scientists, and anyone comfortable with CLI/TUI tools.

## Why This Approach

- **Hugo from scratch (no theme):** Full control over CSS variables, dark/light theming, and brand identity. No fighting against theme opinions. The site is small enough that custom layouts are manageable and won't balloon.
- **Monorepo (`site/` directory):** Keeps website and code together. One PR can update both docs content and the feature it documents. CI/CD is simpler with everything in one repo.
- **Copy and adapt content:** Docs are copied into `site/content/` with Hugo front matter added. This allows website-specific restructuring, navigation ordering, and formatting without constraining the original repo docs. The repo markdown files remain the "developer quick-reference" and the website becomes the polished "user-facing docs."
- **GitHub Actions deploy on push to main:** Simple, reliable, no external dependencies. Standard Hugo + GitHub Pages workflow.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Domain | calcmark.org | Matches branding directly |
| Branding | "CalcMark" throughout | Domain and brand align |
| Site location | `site/` in monorepo | One repo, one PR, simple CI |
| Hugo theme | Custom from scratch | Full CSS variable control, no theme fights |
| Content strategy | Copy and adapt | Website-optimized docs with Hugo front matter |
| Docs scope | User-facing only | User guide, language spec, config, examples. No contributor/internal docs on site. |
| Navigation | Grouped sections | Home, then Docs section with sub-pages |
| Hero visual | Existing hero.gif | Already shows CalcMark in action |
| Fonts | Google Fonts (TBD) | Clean, readable, developer-friendly |
| Theming | CSS variables, dark/light | User-tweakable, minimal JS |

## Site Structure

```
Home (landing page)
  - Hero with animated GIF demo
  - What is CalcMark (brief)
  - Key features
  - Install instructions
  - Link to docs

Docs/
  - Getting Started (from docs/README.md quick-start section)
  - User Guide (from docs/README.md main content)
  - Language Reference (from spec/LANGUAGE_SPEC.md)
  - Configuration (from CONFIG.md)
  - Examples (from docs/examples/*.cm files, rendered with descriptions)
```

## Content Migration Map

| Source File | Website Destination | Notes |
|-------------|-------------------|-------|
| `README.md` | Home page hero + install section | Extract install table, features list |
| `docs/README.md` | Docs > Getting Started + User Guide | Split into two pages: quick-start and full guide |
| `spec/LANGUAGE_SPEC.md` | Docs > Language Reference | Adapt formatting, add Hugo nav |
| `CONFIG.md` | Docs > Configuration | Add front matter, minimal changes |
| `docs/examples/*.cm` | Docs > Examples | Each example as a sub-page with description + code |
| `docs/images/hero.gif` | Home page hero | Copy to site static assets |
| `docs/images/tui-screenshot.png` | Available for docs pages | Copy to site static assets |

## Hugo Directory Layout

```
site/
  hugo.toml              # Site config (baseURL, title, menus, params)
  content/
    _index.md            # Home page
    docs/
      _index.md          # Docs landing/overview
      getting-started.md
      user-guide.md
      language-reference.md
      configuration.md
      examples/
        _index.md
        household-budget.md
        job-offer.md
        project-workback.md
        recipe-scaling.md
        system-sizing.md
  layouts/
    _default/
      baseof.html        # Base template with <head>, nav, footer
      single.html        # Single page layout
      list.html          # List/section layout
    index.html           # Home page template
    partials/
      head.html          # Meta, fonts, CSS
      nav.html           # Top navigation
      sidebar.html       # Docs sidebar navigation
      footer.html        # Footer
      theme-toggle.html  # Dark/light switch
  static/
    images/
      hero.gif
      tui-screenshot.png
    css/
      variables.css      # All CSS custom properties (colors, fonts, spacing)
      base.css           # Reset, typography, layout
      components.css     # Nav, sidebar, code blocks, buttons
      utilities.css      # Dark mode overrides, responsive
    js/
      theme.js           # Dark/light toggle with localStorage
  .github/               # (workflow lives at repo root, not here)
```

## CSS Variables Approach

```css
:root {
  /* Colors - light mode defaults */
  --color-bg: #ffffff;
  --color-text: #1a1a2e;
  --color-heading: #16213e;
  --color-accent: /* TBD - CalcMark brand color */;
  --color-code-bg: #f4f4f8;
  --color-border: #e0e0e0;
  --color-nav-bg: /* TBD */;

  /* Typography */
  --font-body: /* TBD Google Font */, system-ui, sans-serif;
  --font-heading: /* TBD Google Font */, system-ui, sans-serif;
  --font-mono: 'JetBrains Mono', 'Fira Code', monospace;

  /* Spacing */
  --space-xs: 0.25rem;
  --space-sm: 0.5rem;
  --space-md: 1rem;
  --space-lg: 2rem;
  --space-xl: 4rem;

  /* Layout */
  --content-max-width: 48rem;
  --sidebar-width: 16rem;
}

[data-theme="dark"] {
  --color-bg: #0d1117;
  --color-text: #c9d1d9;
  /* ... overrides ... */
}
```

## GitHub Actions Workflow

- Trigger: push to `main` (with path filter for `site/**`)
- Steps: checkout, setup Hugo, build (`hugo --minify`), deploy to GitHub Pages
- CNAME file for calcmark.org in `site/static/CNAME`

## Open Questions

None - all key decisions resolved through brainstorming.
