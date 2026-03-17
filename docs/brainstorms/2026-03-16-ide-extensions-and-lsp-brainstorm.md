# IDE Extensions and LSP Support for CalcMark

**Date:** 2026-03-16
**Status:** Brainstorm

## What We're Building

Editor integrations that let people use CalcMark in their native IDE with full language support and a live rendered document preview. The goal is **adoption** — people discover CalcMark through their editor, see the live preview, and go "oh wow, I need this."

The architecture is **LSP-first**: a Go language server built into the `cm` binary (`cm lsp` subcommand), with thin editor-specific wrappers for VS Code, Zed, Vim/nvim, and Emacs. A tree-sitter grammar provides fast syntax highlighting across all editors.

## Why This Approach

CalcMark already has the pieces — a spec-layer tokenizer/parser/semantic analyzer with position-aware AST nodes, HTML/JSON formatters, and `SYNTAX_HIGHLIGHTER_SPEC.json`. The TUI editor proves the live-preview UX works. The missing piece is bringing that experience to where developers already live.

**LSP-first** was chosen because:
- One server implementation serves all editors
- The `cm` binary already ships cross-platform via GoReleaser — `cm lsp` means zero extra installation
- Editor extensions stay thin (launch `cm lsp`, wire up webview) — less maintenance per editor
- Follows proven patterns: gopls, rust-analyzer, typescript-language-server

## Key Decisions

### 1. `cm lsp` subcommand in the main binary

The LSP server lives in the go-calcmark monorepo alongside the spec and impl packages it depends on. No separate binary, no version drift. Editor extensions just spawn `cm lsp` as a child process.

The `cm` binary may also support a `cm serve` or `cm watch` mode for editors without webview support (user opens a browser tab for the preview).

### 2. File associations

Editor extensions register `.cm` and `.calcmark` file extensions for activation and language detection.

### 3. Tree-sitter grammar for syntax highlighting

A tree-sitter grammar (`tree-sitter-calcmark`) provides static highlighting across all target editors:
- **Zed** — native tree-sitter support
- **nvim** — via nvim-treesitter
- **Emacs** — via tree-sitter integration (Emacs 29+)
- **VS Code** — tree-sitter support in VS Code is less mature than in Zed/nvim. VS Code's primary highlighting mechanism is still TextMate grammars. The VS Code extension may need a **TextMate grammar as well** (or rely heavily on LSP semantic tokens) for reliable highlighting. This is a research item.

The grammar lives in the go-calcmark monorepo (e.g., `editor/tree-sitter-calcmark/`). The existing `SYNTAX_HIGHLIGHTER_SPEC.json` serves as the reference for token types.

### 4. Full language support at launch

The LSP server should ship with the complete IDE experience from day one:
- **Diagnostics** — errors and warnings from semantic analysis (squiggly lines)
- **Autocomplete** — variables, functions (both NL and traditional syntax), units, constants
- **Hover info** — variable values, function signatures, unit descriptions
- **Go-to-definition** — jump to where a variable was assigned
- **Document symbols / outline** — see document structure (sections, variables) in the sidebar
- **Document formatting** — `cm fmt` equivalent via LSP format-on-save
- **Code actions** — quick fixes for common errors (e.g., "did you mean this unit?")
- **Live rendered preview** — the "wow" feature

### 5. Single-file scope

CalcMark documents are self-contained single files (no imports). This simplifies the LSP — no workspace-level dependency tracking needed. Each open `.cm` file is an independent evaluation context.

### 6. Live preview architecture (layered)

Two complementary mechanisms, not mutually exclusive:

**Layer 1: `cm watch` (universal baseline)**
- Watches a file, re-renders on change, serves HTML on localhost
- Works with ANY editor — even Vim/Emacs users just open a browser tab
- Similar to how calcmark-lark already works

**Layer 2: LSP + Webview (tight integration)**
- LSP server evaluates the document and pushes rendered HTML via custom LSP notifications
- Editor extensions render it in a webview panel (VS Code, Zed)
- Reacts to unsaved buffer changes — no file-save trigger needed
- The preview lives *inside* the editor

### 7. Rendered document preview (not just inline results)

The preview shows the full rendered document — headings, prose, and computed results woven together — not just a results column. This is the format that makes CalcMark's value proposition immediately obvious to new users.

### 8. Monorepo structure

Everything lives in the go-calcmark repo:
```
editor/
  tree-sitter-calcmark/   # Tree-sitter grammar
  vscode-calcmark/         # VS Code extension (thin wrapper)
  zed-calcmark/            # Zed extension (thin wrapper)
cmd/calcmark/cmd/
  lsp.go                   # `cm lsp` subcommand
lsp/                       # LSP server implementation (Go)
```

## Editor-Specific Notes

| Editor | Highlighting | LSP | Live Preview |
|--------|-------------|-----|-------------|
| VS Code | Tree-sitter (via anycode) + LSP semantic tokens | `cm lsp` via extension | Webview panel |
| Zed | Tree-sitter (native) | `cm lsp` via extension | Webview panel (if supported*) |
| Neovim | Tree-sitter (nvim-treesitter) | `cm lsp` via lspconfig | `cm watch` + browser |
| Emacs | Tree-sitter (Emacs 29+) | `cm lsp` via eglot | `cm watch` + browser |

## Risk Assessment

**Low risk:** The TUI editor (`cmd/calcmark/tui/`) already consumes diagnostics, autocomplete suggestions, and variable state through rich patterns. The LSP server is essentially exposing the same data over a different protocol — not inventing new capabilities.

**Medium risk:** Tree-sitter grammar for a context-dependent language. Needs prototyping.

**Low risk:** Performance. Sub-100ms P99 on full HTML rendering means every-keystroke preview is viable without optimization.

## Existing Assets to Leverage

- **`spec/SYNTAX_HIGHLIGHTER_SPEC.json`** — machine-readable token definitions, tested on every CI run
- **`spec/lexer/`, `spec/parser/`, `spec/semantic/`** — the real parser/analyzer the LSP wraps
- **`format/html_formatter.go`** — production-ready HTML rendering with templates
- **`spec/features/registry.go`** — all functions, units, constants (for autocomplete)
- **`markdown_wasm.go`** — hints at WASM support for future in-browser preview
- **Position-aware AST** — Range info on all nodes, ready for LSP diagnostics

## Resolved Questions

1. **Preview debouncing** — Every keystroke. Full HTML rendering is already <100ms P99 on the Lark website, so this is viable without incremental evaluation.

2. **Incremental evaluation** — Not needed at launch. Full re-eval is fast enough for typical documents. Can optimize later if large documents surface as a real use case.

## Open Questions (deferred to planning)

1. **Tree-sitter grammar complexity** — CalcMark's line classification is context-dependent (a line can switch from markdown to calc when a variable it references is defined above). Needs a prototype to determine how much tree-sitter can handle vs how much must be covered by LSP semantic tokens.

2. **Extension versioning** — Lock to `cm` versions or version independently? Figure out during CI/CD setup.

3. **`cm lsp` stdio vs socket** — Start with stdio (the standard). Add socket mode later if sharing a process with `cm serve` proves valuable.

4. **Extension distribution** — Publishing strategy for VS Code Marketplace, Zed extension registry, nvim plugin managers. Details for the planning phase.

5. **Zed webview support** — Zed's extension API is still maturing. Verify whether webview panels are available for extensions. If not, Zed falls back to `cm watch` + browser like Vim/Emacs.

6. **VS Code highlighting strategy** — Tree-sitter support in VS Code is less mature than in Zed/nvim. May need a TextMate grammar alongside tree-sitter, or lean on LSP semantic tokens. Needs prototyping.

7. **LSP testing strategy** — How to test the LSP server end-to-end. There are LSP testing frameworks (e.g., `go-lsp-test`) that can simulate editor interactions. Decide during planning.
