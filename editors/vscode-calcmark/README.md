# CalcMark for VS Code

Language support for [CalcMark](https://calcmark.org) — calculations and markdown in one document.

## Features

- **Syntax highlighting** — TextMate grammar for `.cm` and `.calcmark` files
- **Diagnostics** — errors and warnings with precise source positions
- **Autocomplete** — functions, units, variables, natural-language syntax
- **Hover** — variable values, function signatures, unit descriptions
- **Go-to-definition** — jump to variable assignment
- **Document symbols** — variables and headings in outline view
- **Code actions** — quick fixes for undefined variables
- **Signature help** — parameter hints inside function calls
- **Semantic highlighting** — context-aware calc vs markdown line classification
- **Live preview** — rendered document in a side panel, updates on every keystroke

## Requirements

The `cm` binary must be installed and on your PATH.

```bash
brew install calcmark/tap/calcmark
```

Or download from [GitHub Releases](https://github.com/CalcMark/go-calcmark/releases/latest).

## Usage

1. Open any `.cm` or `.calcmark` file
2. **Cmd+Shift+P** → **CalcMark: Open Preview** for live rendered preview

To use a specific `cm` binary, set `calcmark.binaryPath` in settings.

## Codespaces

Works in GitHub Codespaces (remote backend with `cm` installed). Does not work in vscode.dev (no backend to run the language server).

## Development

See [RELEASING.md](RELEASING.md) for publishing instructions.

```bash
# From the repository root:
task build:vscode    # Build cm binary + compile extension
task run:vscode      # Build and launch VS Code with extension loaded
```
