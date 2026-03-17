---
title: "IDE & Editor Setup"
summary: "Use CalcMark in VS Code, Zed, Neovim, or Emacs with full language intelligence."
weight: 23
---

CalcMark ships an LSP server (`cm lsp`) that provides diagnostics, autocomplete, hover, go-to-definition, document symbols, semantic highlighting, and quick fixes in any editor that supports the Language Server Protocol.

## VS Code

Install the **CalcMark** extension from the VS Code marketplace, or from the `editors/vscode-calcmark/` directory in the repository.

The extension automatically finds the `cm` binary in your PATH. To use a specific binary:

```json
{
  "calcmark.binaryPath": "/path/to/cm"
}
```

## Zed

Install the **CalcMark** extension from Zed's extension marketplace.

For local development, symlink or copy `editors/zed-calcmark/` to your Zed extensions directory:

```bash
ln -s /path/to/go-calcmark/editors/zed-calcmark ~/.local/share/zed/extensions/installed/calcmark
```

The extension uses tree-sitter for syntax highlighting and `cm lsp` for language intelligence.

## Neovim

Neovim 0.11+ has built-in LSP support. Add to your config:

```lua
vim.lsp.config.calcmark = {
  cmd = { 'cm', 'lsp' },
  filetypes = { 'calcmark' },
  root_markers = { '.git' },
}
vim.lsp.enable('calcmark')
vim.filetype.add({ extension = { cm = 'calcmark', calcmark = 'calcmark' } })
```

## Emacs

With eglot (built-in since Emacs 29):

```elisp
(add-to-list 'auto-mode-alist '("\\.cm\\'" . fundamental-mode))
(add-to-list 'auto-mode-alist '("\\.calcmark\\'" . fundamental-mode))
(add-to-list 'eglot-server-programs '((fundamental-mode :language-id "calcmark") "cm" "lsp"))
(add-hook 'fundamental-mode-hook #'eglot-ensure)
```

## Live Preview

For a browser-based live preview (any editor):

```bash
cm watch budget.cm
```

This starts a local HTTP server that watches the file and auto-reloads the browser on every save. The URL is printed to stderr — it includes a random session token for security.

## What the LSP Provides

| Feature | Description |
|---------|-------------|
| Diagnostics | Errors and warnings with precise source positions |
| Autocomplete | Functions, units, variables, NL syntax |
| Hover | Variable values, function signatures, unit descriptions |
| Go-to-definition | Jump to variable assignment |
| Document symbols | Variables and headings in outline view |
| Semantic tokens | Context-aware highlighting (calc vs markdown lines) |
| Code actions | Quick fixes for undefined variables ("did you mean?") |
| Signature help | Parameter hints inside function calls |
