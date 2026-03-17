---
title: "Neovim Setup"
summary: "Use CalcMark in Neovim with LSP diagnostics, autocomplete, hover, and go-to-definition."
weight: 23
---

CalcMark includes a built-in language server (`cm lsp`) that provides diagnostics, autocomplete, hover information, go-to-definition, and document symbols in any LSP-capable editor. This guide covers Neovim 0.11+.

## Prerequisites

- **Neovim 0.11+** (check with `nvim --version`)
- **CalcMark** installed and on your PATH (check with `cm version`)

## Basic Setup (5 lines)

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

## Verify It Works

Open a `.cm` file and run:

```vim
:lua vim.print(vim.lsp.get_clients())
```

You should see a `calcmark-lsp` client. Try typing `a = 1 / 0` — a diagnostic should appear.

## Autocomplete

Neovim's built-in completion requires a manual trigger:

- **Ctrl+X Ctrl+O** — trigger omni-completion (LSP completions)
- **Ctrl+N / Ctrl+P** — cycle through results

For auto-popup completions as you type, install **nvim-cmp** (see below).

## What the LSP Provides

| Feature | How to use |
|---------|-----------|
| **Diagnostics** | Appear automatically — errors and warnings show inline |
| **Autocomplete** | Ctrl+X Ctrl+O (or auto-popup with nvim-cmp) |
| **Hover** | `K` or `:lua vim.lsp.buf.hover()` over a variable, function, or unit |
| **Go-to-definition** | `gd` or `:lua vim.lsp.buf.definition()` on a variable reference |
| **Document symbols** | `:lua vim.lsp.buf.document_symbol()` to see all variables and headings |

## Recommended: nvim-cmp for Auto-Completions

The built-in Ctrl+X Ctrl+O workflow is functional but not ideal. **nvim-cmp** gives you automatic popup completions as you type.

### Install with lazy.nvim

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

### Install with packer.nvim

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

After installing, restart Neovim. Completions will auto-popup as you type in `.cm` files — variables, functions (including natural-language syntax like `average of`), and units.

## Troubleshooting

**LSP not attaching:**
- Check `:lua vim.print(vim.lsp.get_clients())` — should show `calcmark-lsp`
- Verify the filetype: `:set ft?` should show `calcmark`
- Check the server can start: run `cm lsp` in a terminal (it should hang waiting for input — that's correct)

**No diagnostics appearing:**
- Ensure `vim.diagnostic` is enabled (it is by default in Neovim 0.11+)
- Try `:lua vim.diagnostic.get()` to see if diagnostics exist but aren't rendering

**Completions not showing with nvim-cmp:**
- Verify cmp-nvim-lsp is installed: `:lua print(require('cmp_nvim_lsp'))` should not error
- Check the LSP is providing completions: `:lua vim.print(vim.lsp.get_clients()[1].server_capabilities.completionProvider)`
