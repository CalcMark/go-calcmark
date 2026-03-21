---
title: "Contributing"
summary: "How to contribute to CalcMark — architecture, development setup, and conventions."
weight: 90
---

CalcMark is open source and welcomes contributions. These resources live in the [go-calcmark repository](https://github.com/CalcMark/go-calcmark) and cover everything you need to get started.

## Architecture

The [Architecture Guide](https://github.com/CalcMark/go-calcmark/blob/main/ARCHITECTURE.md) describes CalcMark's internals layer by layer:

- **Evaluation pipeline** — how source text flows through lexer, parser, semantic checker, and interpreter
- **spec/ vs impl/** — the language specification layer and its strict separation from the interpreter
- **Clients** — how the CLI, TUI editor, LSP server, and documentation site compose the library
- **Type system** — Number, Currency, Quantity, Duration, Rate, Fraction, Percentage, Date, Boolean
- **Performance targets** and testing strategy

## Development Setup

The [Contributing Guide](https://github.com/CalcMark/go-calcmark/blob/main/CONTRIBUTING.md) covers:

- Prerequisites (Go, Task, Node.js)
- Building and running tests (`task test`, `task quality`)
- How to add new operators, functions, and language features
- Test structure and conventions (table-driven, golden files, catwalk)
- Code style and error message guidelines

## Quick Start

```bash
git clone https://github.com/CalcMark/go-calcmark
cd go-calcmark
task test      # Run all tests
task quality   # Lint + modernize + staticcheck
task build     # Build the cm binary
task dev       # Start the REPL
```

## Key Directories

| Directory | What it contains |
|-----------|-----------------|
| `spec/` | Language specification — grammar, types, units, validation |
| `impl/` | Interpreter — evaluation, functions, operators |
| `cmd/calcmark/` | CLI and TUI editor |
| `lsp/` | Language Server Protocol implementation |
| `format/` | Output formatters (HTML, JSON, Markdown, Text) |
| `site/` | This documentation site (Hugo) |
| `testdata/` | Golden test files — valid and invalid CalcMark examples |
