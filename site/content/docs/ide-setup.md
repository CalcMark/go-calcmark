---
title: "IDE & Editor Setup"
summary: "Use CalcMark in VS Code, Zed, Neovim, or Emacs with full language intelligence."
weight: 23
---

CalcMark ships an LSP server (`cm lsp`) that provides diagnostics, autocomplete, hover, go-to-definition, document symbols, semantic highlighting, and quick fixes in any editor that supports the Language Server Protocol.

## VS Code

Install the [**CalcMark** extension from the VS Code Marketplace](https://marketplace.visualstudio.com/items?itemName=calcmark.vscode-calcmark), or from the `editors/vscode-calcmark/` directory in the repository.

The extension automatically finds the `cm` binary in your PATH. To use a specific binary:

```json
{
  "calcmark.binaryPath": "/path/to/cm"
}
```

<details>
<summary>Dev Setup (Generic LSP Client)</summary>

If you're building CalcMark from source or want to try the LSP before the official extension ships, you can use a generic LSP client extension to connect VS Code to `cm lsp`.

**Prerequisites:** VS Code 1.82+ and CalcMark built from source (`task build`) or installed via Homebrew.

**Step 1:** Install **Generic LSP Client (v2)** by zsol — search `zsol.vscode-glspc` in Extensions.

**Step 2:** Open your VS Code settings JSON (Ctrl+Shift+P -> "Preferences: Open User Settings (JSON)") and add:

```json
"glspc.server.command": "/path/to/cm",
"glspc.server.commandArguments": ["lsp"],
"glspc.server.languageId": ["calcmark"]
```

Replace `/path/to/cm` with the actual path to your `cm` binary. If `cm` is on your PATH (e.g., installed via Homebrew), you can use just `"cm"`.

You also need to tell VS Code that `.cm` files are the `calcmark` language:

```json
"files.associations": {
    "*.cm": "calcmark",
    "*.calcmark": "calcmark"
}
```

**Step 3:** Open a `.cm` file. You should see diagnostics, autocomplete (Ctrl+Space), hover info, and go-to-definition (Ctrl+click).

**Limitations:** The generic LSP client does not provide syntax highlighting (files appear as plain text), live preview, or auto-install of the `cm` binary. The official extension will include all of these.

</details>

### Troubleshooting

**First step for any issue:** Open the Output panel (Ctrl+Shift+U) and select **"CalcMark LSP"** from the dropdown on the right. This shows the server's communication log and any errors.

**No diagnostics or completions:**
- Check the CalcMark LSP output channel for errors
- Verify `cm lsp` starts correctly: run `cm lsp` in a terminal -- it should hang waiting for input (that's normal)
- Check that the file extension is `.cm` or `.calcmark`

**"Command not found" error:**
- The `cm` binary path in settings must be correct. Use an absolute path if in doubt.
- If you built from source, make sure you ran `task build` first

**Completions don't auto-popup:**
- Press Ctrl+Space to manually trigger completions
- VS Code should auto-popup for LSP completions by default, but check that `"editor.quickSuggestions"` is enabled in your settings

**Server crashes on startup (error -32097):**
- The `cm` binary on your PATH may be an older version without `lsp` support
- Set `calcmark.binaryPath` in settings to the full path of the correct `cm` binary
- Check the CalcMark LSP output channel -- it will show what command failed

**AI autocomplete blocking CalcMark suggestions:**
- GitHub Copilot or similar extensions can intercept completions. Disable for CalcMark files:
  ```json
  "github.copilot.enable": { "calcmark": false }
  ```

**Binary version mismatch:**
- If you're developing CalcMark, rebuild with `task build` after pulling changes to keep the LSP server up to date

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

These are Zed's built-in LSP keybindings -- they work automatically with the CalcMark extension:

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

Run it via **Cmd+Shift+P** -> **task: spawn** -> **CalcMark: Preview**, or bind it to a key in `~/.config/zed/keymap.json`:

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

Neovim 0.11+ has built-in LSP support.

Add to your `init.lua`:

```lua
vim.filetype.add({ extension = { cm = 'calcmark', calcmark = 'calcmark' } })
vim.lsp.enable({ 'calcmark' })
```

Create `~/.config/nvim/lsp/calcmark.lua`:

```lua
return {
  cmd = { 'cm', 'lsp' },
  filetypes = { 'calcmark' },
  root_markers = { '.git' },
}
```

That's it. Open any `.cm` file and the language server attaches automatically.

If `cm` isn't on your PATH, use the full path:

```lua
cmd = { '/path/to/cm', 'lsp' },
```

### Verify It Works

Open a `.cm` file and run:

```vim
:lua vim.print(vim.lsp.get_clients())
```

You should see a `calcmark-lsp` client. Try typing `a = 1 / 0` -- a diagnostic should appear.

### Autocomplete

Neovim's built-in completion requires a manual trigger:

- **Ctrl+X Ctrl+O** -- trigger omni-completion (LSP completions)
- **Ctrl+N / Ctrl+P** -- cycle through results

For auto-popup completions as you type, install **nvim-cmp** (see below).

<details>
<summary>Auto-completions with nvim-cmp</summary>

The built-in Ctrl+X Ctrl+O workflow is functional but not ideal. **nvim-cmp** gives you automatic popup completions as you type.

#### Install with lazy.nvim

Add to your lazy.nvim plugin spec:

```lua
{
  'hrsh7th/nvim-cmp',
  dependencies = {
    'hrsh7th/cmp-nvim-lsp',
  },
  config = function()
    local cmp = require('cmp')
    cmp.setup({
      sources = {
        { name = 'nvim_lsp' },
      },
      mapping = cmp.mapping.preset.insert({
        ['<C-Space>'] = cmp.mapping.complete(),
        ['<CR>'] = cmp.mapping.confirm({ select = true }),
        ['<C-n>'] = cmp.mapping.select_next_item(),
        ['<C-p>'] = cmp.mapping.select_prev_item(),
      }),
    })
  end,
},
```

#### Install with packer.nvim

```lua
use {
  'hrsh7th/nvim-cmp',
  requires = { 'hrsh7th/cmp-nvim-lsp' },
  config = function()
    local cmp = require('cmp')
    cmp.setup({
      sources = { { name = 'nvim_lsp' } },
      mapping = cmp.mapping.preset.insert({
        ['<C-Space>'] = cmp.mapping.complete(),
        ['<CR>'] = cmp.mapping.confirm({ select = true }),
      }),
    })
  end,
}
```

After installing, restart Neovim. Completions will auto-popup as you type in `.cm` files -- variables, functions (including natural-language syntax like `average of`), and units.

</details>

### Troubleshooting

**LSP not attaching:**
- Check `:lua vim.print(vim.lsp.get_clients())` -- should show `calcmark-lsp`
- Verify the filetype: `:set ft?` should show `calcmark`
- Check the server can start: run `cm lsp` in a terminal (it should hang waiting for input -- that's correct)

**No diagnostics appearing:**
- Ensure `vim.diagnostic` is enabled (it is by default in Neovim 0.11+)
- Try `:lua vim.diagnostic.get()` to see if diagnostics exist but aren't rendering

**Completions not showing with nvim-cmp:**
- Verify cmp-nvim-lsp is installed: `:lua print(require('cmp_nvim_lsp'))` should not error
- Check the LSP is providing completions: `:lua vim.print(vim.lsp.get_clients()[1].server_capabilities.completionProvider)`

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

This starts a local HTTP server that watches the file and auto-reloads the browser on every save. The URL is printed to stderr -- it includes a random session token for security.

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
