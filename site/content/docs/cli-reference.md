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
cm help [topic]      # Browse functions and constants
cm version           # Print version info
cm completion [shell] # Generate shell completions
```

### Global Flags

| Flag | Values | Description |
|------|--------|-------------|
| `--color-mode` | `auto`, `light`, `dark` | Override terminal color detection |

The `--color-mode` flag is available on all commands. When set to `auto` (default), CalcMark detects the terminal background color. Use `light` or `dark` to override.

---

## `cm [file]` {#cm}

Open the CalcMark editor. With no arguments, starts with an empty document. With a file argument, opens that file for editing.

```bash
cm                    # New document
cm budget.cm          # Edit existing file
cm --color-mode=dark  # Force dark mode
```

See [Editor Shortcuts](/docs/user-guide/#editor-shortcuts) in the User Guide for keyboard shortcuts.

---

## `cm eval [file.cm]` {#eval}

Evaluate a CalcMark file or stdin and print results.

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--verbose` | `-v` | Show all intermediate values, not just the final result |

### Examples

```bash
cm eval budget.cm             # Print final results
cm eval -v budget.cm          # Show all intermediate values
echo "x = 10" | cm eval      # Evaluate from stdin
echo "5 miles in km" | cm eval
```

When reading from a file, only the last evaluated result is printed unless `-v` is used. With `-v`, every line that produces a value is shown.

---

## `cm convert <file.cm>` {#convert}

Convert a CalcMark file to another format. Requires the `--to` flag.

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--to` | `-t` | **(required)** Output format: `html`, `md`, `json`, `text`, `cm` |
| `--output` | `-o` | Write to file instead of stdout |
| `--template` | `-T` | Custom Go template (html format only) |

### Examples

```bash
cm convert doc.cm --to=html              # HTML to stdout
cm convert doc.cm --to=md -o doc.md      # Markdown to file
cm convert doc.cm --to=json              # JSON to stdout
cm convert doc.cm --to=text              # Plain text
cm convert doc.cm --to=html -T tpl.html  # Custom HTML template
```

### Output Formats

| Format | Description |
|--------|-------------|
| `html` | Full HTML document with evaluated results |
| `md` | Markdown with calculation results inline |
| `json` | Structured JSON of all variables and results |
| `text` | Plain text output (same as `cm eval -v`) |
| `cm` | Normalized CalcMark source |

---

## `cm help [topic]` {#help}

Display help for CalcMark topics. Without a topic, lists available topics.

### Topics

| Topic | Description |
|-------|-------------|
| `functions` | All built-in functions with descriptions and usage patterns |
| `constants` | All unit constants grouped by quantity type |

### Examples

```bash
cm help               # List available topics
cm help functions     # Show all functions
cm help constants     # Show all unit constants
```

---

## `cm version` {#version}

Print the CalcMark version and build time.

```bash
cm version
# CalcMark v0.8.0
#   built: 2026-02-20T10:30:00Z
```

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
