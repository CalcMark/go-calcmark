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

### Dev install (local development)

Install the extension from your local checkout:

1. Open Zed
2. Open the command palette: **Cmd+Shift+P**
3. Run **zed: install dev extension**
4. Select the `editors/zed-calcmark/` directory

The extension uses tree-sitter for syntax highlighting and `cm lsp` for language intelligence.

### Uninstall dev extension

1. Open the command palette: **Cmd+Shift+P**
2. Run **zed: uninstall dev extension**
3. Select **calcmark**

Or remove it manually:

```bash
rm -rf ~/.local/share/zed/extensions/installed/calcmark
```

Then restart Zed.

### Enable semantic highlighting

Zed disables LSP semantic tokens by default. Add this to your Zed settings (**Cmd+,**) to enable context-aware highlighting for CalcMark:

```json
{
  "languages": {
    "CalcMark": {
      "semantic_tokens": "full"
    }
  }
}
```

### Useful keybindings in Zed

These are Zed's built-in LSP keybindings — they work automatically with the CalcMark extension:

| Shortcut | Action |
|----------|--------|
| **Cmd+.** | Code actions (quick fixes for undefined variables) |
| **Cmd+Shift+O** | Document symbols (variables and headings outline) |
| **Cmd+click** or **F12** | Go to definition (jump to variable assignment) |
| **Cmd+Shift+M** | Open diagnostics panel (errors and warnings) |
| hover | Hover info (variable values, function signatures) |
| type | Autocomplete (functions, units, variables appear automatically) |
| **Cmd+Shift+Space** | Signature help (parameter hints inside function calls) |

### Live preview from Zed

Zed doesn't support webview panels, so use `cm watch` for live preview.

Add a task to `.zed/tasks.json` in your project root:

```json
[
  {
    "label": "CalcMark: Preview",
    "command": "cm watch \"$ZED_FILE\"",
    "reveal": "always",
    "hide": "on_success"
  }
]
```

Run it via **Cmd+Shift+P** → **task: spawn** → **CalcMark: Preview**, or bind it to a key in `~/.config/zed/keymap.json`:

```json
[
  {
    "context": "Workspace",
    "bindings": {
      "cmd-shift-r": ["task::Spawn", { "task_name": "CalcMark: Preview" }]
    }
  }
]
```

Now **Cmd+Shift+R** on any `.cm` file launches a live preview in your browser that auto-reloads on every save.

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

Options:

```bash
cm watch --port 8080 budget.cm    # Custom port (default: 3141)
```

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
