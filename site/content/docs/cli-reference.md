---
title: "CLI Reference"
summary: "Complete reference for CalcMark CLI commands, flags, and shell completion."
weight: 35
---

## Overview

The CalcMark CLI (`cm`) provides commands for editing, evaluating, converting, and exploring CalcMark documents.

```bash
cm [file]            # Open editor (default command)
cm eval [file.cm]    # Evaluate and print results
cm convert <file.cm> # Convert to another format
cm remote            # Open a remote document
cm config            # Print or create configuration
cm functions         # List all CalcMark functions
cm constants         # List all unit constants
cm help [command]     # Help for any command
cm version           # Print version info
cm completion [shell] # Generate shell completions
```

### Global Flags

| Flag | Values | Description |
|------|--------|-------------|
| `--color-mode` | `auto`, `light`, `dark` | Override terminal color detection |
| `--locale` | `en-US`, `de-DE`, `fr-FR` | Display locale for number formatting |
| `--format` | `text`, `json`, `html`, `md`, `cm` | Output format when stdin is piped (default: `text`) |
| `--verbose` / `-v` | | Show all intermediate values when stdin is piped |

The `--color-mode` flag is available on all commands. Use `light` or `dark` to match your terminal background. The value `auto` is accepted but deprecated and treated as `dark`.

The `--locale` flag controls decimal and thousands separators in output. See [Configuration: Display Locale](/docs/configuration/#locale) for details.

---

## `cm [file]` {#cm}

Open the CalcMark editor. With no arguments, starts with an empty document. With a file argument, opens that file for editing.

When stdin is piped, `cm` automatically evaluates the input and prints results to stdout instead of launching the editor. This makes CalcMark usable in shell pipelines and by AI agents.

### Flags (piped mode)

| Flag | Short | Description |
|------|-------|-------------|
| `--format` | | Output format: `text` (default), `json`, `html`, `md`, `cm` |
| `--verbose` | `-v` | Show all intermediate values, not just the final result |

### Examples

```bash
cm                              # New document (interactive editor)
cm budget.cm                    # Edit existing file
cm --color-mode=dark            # Force dark mode

echo "x = 42" | cm             # Evaluate piped input (text output)
echo "x = 42" | cm --format json  # JSON output for scripting/agents
echo "x = 10" | cm -v          # Verbose output with all values
cat budget.cm | cm --format json   # Pipe a file through cm
```

See [Editor Shortcuts](/docs/user-guide/#editor-shortcuts) in the User Guide for keyboard shortcuts.

---

## `cm eval [file.cm]` {#eval}

Evaluate a CalcMark file or stdin and print results.

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--verbose` | `-v` | Show all intermediate values, not just the final result |
| `--format` | | Output format: `text` (default), `json`, `html`, `md`, `cm` |

### Examples

```bash
cm eval budget.cm                    # Print final results
cm eval -v budget.cm                 # Show all intermediate values
echo "x = 10" | cm eval             # Evaluate from stdin
echo "5 miles in km" | cm eval
cm eval --locale=de-DE budget.cm     # German number formatting
cm eval --format json budget.cm      # JSON output with type decomposition
cm eval --format html doc.cm         # HTML document
cm eval --format md budget.cm        # Markdown with embedded results
```

When reading from a file, only the last evaluated result is printed unless `-v` is used. With `-v`, every line that produces a value is shown.

Use `--format` to select the output format. The available formats are the same as `cm convert --to` (see [Output Formats](#convert)). The default format can be changed in [Configuration](/docs/configuration/) via `default_format`.

Use `--locale` to format output with locale-specific separators (e.g., `1.500,00` instead of `1,500.00` for de-DE).

---

## `cm convert <file.cm>` {#convert}

Convert a CalcMark file to another format. Requires the `--to` flag.

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--to` | `-t` | **(required)** Output format: `html`, `md`, `json`, `text`, `cm` |
| `--output` | `-o` | Write to file instead of stdout |
| `--template` | `-T` | Custom Go template (html format only) |
| `--show-template` | | Print the default HTML template and exit |

### Examples

```bash
cm convert doc.cm --to=html              # HTML to stdout
cm convert doc.cm --to=md -o doc.md      # Markdown to file
cm convert doc.cm --to=json              # JSON to stdout
cm convert doc.cm --to=text              # Plain text
cm convert doc.cm --to=html -T tpl.html  # Custom HTML template
cm convert --show-template               # Print default HTML template
cm convert doc.cm --to=json --locale=de-DE  # JSON with German formatting
```

### Output Formats

| Format | Description |
|--------|-------------|
| `html` | Full HTML document with evaluated results |
| `md` | Markdown with calculation results inline |
| `json` | Structured JSON with type-aware decomposition fields |
| `text` | Plain text output (same as `cm eval -v`) |
| `cm` | Normalized CalcMark source (always locale-independent) |

> **Note:** The `cm` format is always locale-independent to ensure portability. All other formats respect the `--locale` flag.

### Custom HTML Templates {#custom-templates}

CalcMark uses Go's `html/template` package for HTML output. You can provide your own template to control the HTML structure and styling.

**Get the default template as a starting point:**

```bash
cm convert --show-template > my-template.html
```

**Use your custom template:**

```bash
cm convert doc.cm --to=html -T my-template.html
```

#### Template Data Model

The template receives a root object with two fields:

| Field | Type | Description |
|-------|------|-------------|
| `.Frontmatter` | object (nil if absent) | Document frontmatter (globals, exchange rates) |
| `.Blocks` | list | Ordered list of content blocks |

**Block fields** (each item in `.Blocks`):

| Field | Type | Description |
|-------|------|-------------|
| `.Type` | string | `"calculation"` or `"text"` |
| `.SourceLines` | list | Calculation lines (only for `calculation` blocks) |
| `.Error` | string | Error message (only for `calculation` blocks) |
| `.HTML` | HTML | Rendered markdown (only for `text` blocks) |

**SourceLine fields** (each item in `.SourceLines`):

| Field | Type | Description |
|-------|------|-------------|
| `.Source` | string | The CalcMark source expression |
| `.Result` | string | Locale-formatted evaluation result (empty if line has no result) |
| `.RawResult` | string | Machine-readable ASCII result, always en-US format (HTML templates only) |

> **JSON output:** In JSON format, each result includes `value` (locale-formatted for display), `type` (CalcMark type name), `numeric_value` (machine-readable number), and `unit` (unit identifier). Use `type` for dispatch, `numeric_value` + `unit` for computation. See [Configuration: JSON Output and Locale](/docs/configuration/#json-raw-value).

**Frontmatter fields** (when `.Frontmatter` is non-nil):

| Field | Type | Description |
|-------|------|-------------|
| `.Frontmatter.Globals` | list | Global variables (`@var = value`) |
| `.Frontmatter.Exchange` | list | Exchange rates |

Each global has `.Name` and `.Value`. Each exchange rate has `.From`, `.To`, and `.Rate`.

#### Minimal Custom Template

```html
<!DOCTYPE html>
<html>
<body>
  {{range .Blocks}}
    {{if eq .Type "calculation"}}
      <pre>
      {{range .SourceLines}}
        {{.Source}}{{if .Result}} = {{.Result}}{{end}}
      {{end}}
      </pre>
    {{else}}
      <div>{{.HTML}}</div>
    {{end}}
  {{end}}
</body>
</html>
```

---

## `cm remote` {#remote}

Fetch a CalcMark document from a remote source and open it in the editor.

### Flags

| Flag | Description |
|------|-------------|
| `--gist` | GitHub Gist URL or ID (requires `gh` CLI) |
| `--http` | Public HTTP(S) URL to fetch via GET |

Exactly one of `--gist` or `--http` must be provided.

### Examples

```bash
cm remote --gist abc123def                           # Open by gist ID
cm remote --gist https://gist.github.com/user/abc123 # Open by gist URL
cm remote --http https://example.com/budget.cm        # Open from any public URL
```

### Gist Mode (`--gist`)

The `--gist` flag accepts any identifier that `gh gist view` supports: a gist ID, a full gist URL, or a URL to any user's public gist. For multi-file gists, the first `.cm` file is opened (or the first file if no `.cm` file exists).

Requires the [GitHub CLI](https://cli.github.com) (`gh`) to be installed and authenticated (`gh auth login`).

### HTTP Mode (`--http`)

The `--http` flag fetches content from a public URL via HTTP GET. Only `http://` and `https://` URLs are accepted.

**Safety limits:**

- Maximum response size: 1 MB
- Binary content is rejected
- Empty responses are rejected
- Redirect targets must stay on HTTP(S)
- Connection timeout: 30 seconds

---

## `cm config` {#config}

Print the current effective configuration or create a starter config file.

### Flags

| Flag | Description |
|------|-------------|
| `--create` | Create a starter config file at the XDG config path |
| `--check` | Validate config files and report errors (exit 0 = ok, exit 1 = errors) |

### Examples

```bash
cm config                    # Print effective configuration as TOML
cm config > backup.toml      # Save current config to a file
cm config --create           # Create ~/.config/calcmark/config.toml
cm config --check            # Validate all config files
```

Without flags, `cm config` prints the fully-resolved effective configuration (embedded defaults merged with user overrides and CLI flag overrides) as valid TOML to stdout.

With `--create`, a starter config file is written to `~/.config/calcmark/config.toml` (or `$XDG_CONFIG_HOME/calcmark/config.toml` if set) with all values commented out and descriptive comments included. The command refuses to overwrite an existing file.

With `--check`, all config files are validated for TOML syntax, valid hex colors (`#RGB` or `#RRGGBB`), valid `color_mode` values (`light`, `dark`, `auto`), valid `default_format` values, and unknown keys. File discovery results and any errors are printed to stderr. Exits with code 0 if everything is valid, or code 1 if any errors were found.

---

## `cm functions` / `cm constants` {#help-topics}

Browse CalcMark's built-in functions and unit constants.

| Command | Description |
|---------|-------------|
| `cm functions` | All built-in functions with descriptions and usage patterns |
| `cm constants` | All unit constants grouped by quantity type |

### Examples

```bash
cm functions          # Show all functions
cm constants          # Show all unit constants
```

---

## `cm help` {#help}

Show help for any command. Equivalent to passing `--help` to a command.

```bash
cm help              # Show top-level help
cm help eval         # Help for the eval command
cm help convert      # Help for the convert command
```

---

## `cm version` {#version}

Print the CalcMark version and build time.

```bash
cm version
# CalcMark v1.5.0
#   built: 2026-03-06T19:40:55Z
```

The version and build time reflect the installed release. Development builds show `CalcMark dev`.

---

## `cm completion [shell]` {#completion}

Generate shell completion scripts. Supports `bash`, `zsh`, `fish`, and `powershell`.

### Setup

**Bash:**
```bash
source <(cm completion bash)
# Or persist in ~/.bashrc:
cm completion bash >> ~/.bashrc
```

**Zsh:**
```bash
cm completion zsh > "${fpath[1]}/_cm"
# Then restart shell or run: compinit
```

**Fish:**
```bash
cm completion fish > ~/.config/fish/completions/cm.fish
```

**PowerShell:**
```powershell
cm completion powershell | Out-String | Invoke-Expression
```
