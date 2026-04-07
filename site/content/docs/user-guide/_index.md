---
title: "User Guide"
summary: "Complete guide to CalcMark features, editor shortcuts, and workflows."
weight: 20
---

This guide covers everything you need to use CalcMark effectively -- from editor shortcuts and export options to the full language feature set. Use the links below to jump to a specific topic, or read through the sections on this page first.

## Contents {#contents}

- [Editor Shortcuts](#editor-shortcuts) -- Keyboard shortcuts for the CalcMark editor
- [Locale Formatting](#locale-formatting) -- Display numbers in your locale
- [Exporting Results](#exporting-results) -- `cm convert` and `cm eval`
- [Embedded Mode](#embedded-mode) -- CalcMark blocks inside standard Markdown files
- [Sharing with GitHub Gist](#sharing-gist) -- Share and open documents via GitHub Gist
- [Language Features](#language-features)
  - [Dates & Time](dates/) -- Date creation, relative dates, quarters, fiscal periods, `ago`/`from`, `end of`
  - [Units & Measurement](units/) -- Physical units, conversion, and measurement conventions
  - [Currency & Exchange Rates](currency/) -- Exchange rates in frontmatter
  - [Frontmatter Directives](frontmatter/) -- Globals, directives, templates, scale and convert
  - [Functions & NL Syntax](functions/) -- Built-in functions, NL forms, network, storage, growth
  - [Formatting & Display](formatting/) -- Rates, dates, napkin math, precise display, percentages
  - [Worked Examples & Tips](examples/) -- Complete scenarios, tips, and troubleshooting

---

## Editor Shortcuts {#editor-shortcuts}

The CalcMark editor provides keyboard shortcuts for common actions. Press **F1** for full help inside the editor.

### File {#shortcuts-file}

| Shortcut | Action |
|----------|--------|
| Ctrl+N | New empty document |
| Ctrl+S | Save document |
| Ctrl+O | Open file |
| Ctrl+T | Export to format |
| Ctrl+Q | Quit editor |

### Edit {#shortcuts-edit}

| Shortcut | Action |
|----------|--------|
| Ctrl+Z | Undo last change |
| Ctrl+Y | Redo last change |
| Ctrl+K | Delete current line |
| Ctrl+F | Add YAML frontmatter |
| Ctrl+A | Select all |
| Ctrl+C | Copy selection |
| Ctrl+X | Cut selection |
| Ctrl+V | Paste |
| Ctrl+Backspace | Delete word |

### View {#shortcuts-view}

| Shortcut | Action |
|----------|--------|
| Ctrl+P | Cycle preview mode |
| Ctrl+H / F1 | Open command menu |

### Navigation {#shortcuts-navigation}

| Shortcut | Action |
|----------|--------|
| Opt+Left / Opt+B | Move to previous word |
| Opt+Right / Opt+F | Move to next word |
| Ctrl+Home | Jump to document start |
| Ctrl+End | Jump to document end |
| Ctrl+Left / Ctrl+Right | Move word left/right |
| Ctrl+D | Scroll down half page |
| Ctrl+U | Scroll up half page |
| Shift+Arrow | Extend selection |

## Locale Formatting {#locale-formatting}

CalcMark can format output numbers using locale-specific decimal and thousands separators. This affects how results are displayed in all output modes (editor, REPL, eval, convert).

### Setting Your Locale

**Per-command** with the `--locale` flag:

```bash
cm eval --locale=de-DE budget.cm
cm convert doc.cm --to=html --locale=fr-FR
```

**Permanently** in your config file:

```toml
# ~/.config/calcmark/config.toml
locale = "de-DE"
```

The `--locale` flag overrides the config file for that invocation.

### Example: Same Document, Different Locales

Given a CalcMark document:

```calcmark
price = $1500
pi = 3.14159
users = 1500000
weight = 50.5 kg
```

| Value | en-US (default) | de-DE | fr-FR |
|-------|-----------------|-------|-------|
| `price` | `$1,500.00` | `$1.500,00` | `$1 500,00` |
| `pi` | `3.14159` | `3,14159` | `3,14159` |
| `users` | `1.5M` | `1,5M` | `1,5M` |
| `weight` | `50.5 kg` | `50,5 kg` | `50,5 kg` |

### What Stays the Same

Regardless of locale, these never change:

- **Input syntax** -- always use `.` for decimals and `,` or `_` for thousands in your source
- **K/M/B/T suffixes** -- always English letters
- **Currency symbols** -- always prefix position (`$`, `€`, etc.)
- **CalcMark format** -- `cm convert --to=cm` is always locale-independent

### JSON Output

When exporting to JSON, each result includes a locale-formatted display value and structured type information:

```bash
cm convert budget.cm --to=json --locale=de-DE
```

```json
{
  "source": "price = $1500",
  "value": "$1.500,00",
  "type": "currency",
  "numeric_value": 1500,
  "unit": "USD",
  "variable": "price"
}
```

Use `type` for dispatch, `numeric_value` + `unit` for computation, and `value` for display. The `type`, `numeric_value`, and `unit` fields are always locale-independent. See [Configuration: JSON Output](/docs/configuration/#json-raw-value) for details.

## Exporting Results {#exporting-results}

Convert CalcMark files to other formats using `cm convert`:

```bash
cm convert doc.cm --to=html              # HTML to stdout
cm convert doc.cm --to=md -o doc.md      # Markdown file
cm convert doc.cm --to=json              # JSON to stdout
cm convert doc.cm --to=html -T tpl.html  # Custom HTML template
```

Use `cm eval` for quick evaluation, or pipe directly into `cm`:

```bash
cm eval budget.cm            # Print final results
cm eval -v budget.cm         # Show all intermediate values
echo "1 + 2" | cm           # Evaluate piped input (auto-detects pipe)
echo "1 + 2" | cm --format json  # JSON output for scripting/agents
cm eval --locale=de-DE budget.cm  # German number formatting
```

All export formats respect the `--locale` flag (or config setting) except `cm` format, which stays locale-independent.

### Embedded Mode {#embedded-mode}

You can embed CalcMark blocks inside standard Markdown files — blog posts, READMEs, Hugo content — and use `cm convert --embedded` to evaluate just the calculation blocks while leaving everything else untouched.

```bash
cm convert report.md --embedded            # Evaluate cm blocks, output to stdout
cm convert report.md --embedded -o out.md  # Write to file
```

Each ` ```cm ` or ` ```calcmark ` fenced code block is treated as an independent CalcMark document. Blocks can have their own frontmatter (exchange rates, scale directives), but they don't share variables with each other. Everything outside the blocks — prose, Hugo frontmatter, footnotes, tables, HTML — passes through byte-for-byte.

This makes CalcMark work like a preprocessor in your static site build pipeline, similar to how [D2](https://d2lang.com) handles diagram blocks. See the [CLI reference](/docs/cli-reference/#embedded-mode) for full details and the [embedded datacenter cost example](/docs/examples/embedded-datacenter-cost/) for a complete walkthrough.

## Sharing with GitHub Gist {#sharing-gist}

Share CalcMark documents as GitHub Gists directly from the editor. This feature requires the [GitHub CLI (`gh`)](https://cli.github.com) to be installed and authenticated.

### Prerequisites

Install and authenticate the GitHub CLI:

```bash
# Install
brew install gh          # macOS
sudo apt install gh      # Debian/Ubuntu

# Authenticate
gh auth login
```

### Share To Gist

Open the command menu (**Ctrl+H** or **F1**), select **Share To Gist**, then choose visibility (public or secret) and add an optional description. Press **Enter** to share. The Gist URL is copied to your clipboard.

If you are not authenticated, CalcMark will launch `gh auth login` interactively and retry after you sign in.

### Open From Gist

Open the command menu, select **Open From Gist**, then paste a Gist URL or ID. CalcMark fetches the Gist content and loads it into the editor. If the Gist contains multiple files, `.cm` files are preferred.

> **Note:** Sharing with GitHub Gist is not available in the browser (WASM) build.

## Language Features {#language-features}

The CalcMark language is covered in detail across these sub-pages:

- **[Dates & Time](dates/)** -- Date creation, relative dates (`next Friday`, `this quarter`), fiscal periods, `ago`/`from`, `start of`/`end of`, calendar-correct month arithmetic
- **[Units & Measurement](units/)** -- Supported units, unit conversion with `in`/`as`, and measurement conventions (US, Imperial, Troy)
- **[Currency & Exchange Rates](currency/)** -- Currency conversion using frontmatter exchange rates
- **[Frontmatter Directives](frontmatter/)** -- Global variables, `@scale`/`@globals` references, template interpolation, and document-wide scale/convert transforms
- **[Functions & NL Syntax](functions/)** -- All built-in functions with examples, natural language syntax, network/storage/growth domain functions
- **[Formatting & Display](formatting/)** -- Rates and `over`, date arithmetic, napkin math, precise display, capacity planning, multiplier suffixes, percentages
- **[Worked Examples & Tips](examples/)** -- Complete calculation scenarios, organizational tips, and troubleshooting common errors
