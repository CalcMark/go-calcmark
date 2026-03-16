---
title: "VS Code Setup"
summary: "Use CalcMark in Visual Studio Code with LSP diagnostics, autocomplete, hover, and go-to-definition."
weight: 24
---

CalcMark includes a built-in language server (`cm lsp`) that provides diagnostics, autocomplete, hover information, go-to-definition, and document symbols. This guide covers two setups: using the official CalcMark extension (once released), and a dev/early-access setup using a generic LSP client.

## Option A: CalcMark Extension (Coming Soon)

Once the official **CalcMark for VS Code** extension is published to the marketplace, setup is one step:

1. Open VS Code
2. Go to Extensions (Ctrl+Shift+X)
3. Search for **CalcMark**
4. Click **Install**

The extension handles everything: syntax highlighting, LSP client, live preview panel, and binary discovery. Open any `.cm` or `.calcmark` file and you're ready to go.

---

## Option B: Dev Setup (Generic LSP Client)

If you're building CalcMark from source or want to try the LSP before the official extension ships, you can use a generic LSP client extension to connect VS Code to `cm lsp`.

### Prerequisites

- **VS Code** 1.82+
- **CalcMark** built from source (`task build` produces a `cm` binary) or installed via Homebrew

### Step 1: Install a Generic LSP Client

Open VS Code and install one of these extensions:

- **Generic LSP Client (v2)** by zsol — search `zsol.vscode-glspc` in Extensions
- **Generic LSP Client** by llllvvuu — search `llllvvuu.llllvvuu-glspc` in Extensions

Both work. The v2 fork is more actively maintained.

### Step 2: Configure the LSP Client

Add to your VS Code settings (Ctrl+Shift+P → "Preferences: Open User Settings (JSON)"):

```json
{
  "glspc.serverCommand": "/path/to/cm",
  "glspc.serverArgs": ["lsp"],
  "glspc.languages": [
    {
      "id": "calcmark",
      "extensions": [".cm", ".calcmark"]
    }
  ]
}
```

Replace `/path/to/cm` with the actual path to your `cm` binary. If you built from source, this is the path to the `cm` file in your repo root (e.g., `/Users/you/projects/go-calcmark/cm`).

If `cm` is on your PATH (e.g., installed via Homebrew), you can use just `"cm"` instead of the full path.

### Step 3: Open a .cm File

Create or open a file with a `.cm` extension. You should see:

- **Diagnostics** — type `a = 1 / 0` and see an error underline
- **Autocomplete** — type `av` and trigger completions (Ctrl+Space) to see `avg`, `average of`
- **Hover** — hover over a variable name to see its computed value
- **Go-to-definition** — Ctrl+click a variable reference to jump to its assignment

### What Works

| Feature | Status |
|---------|--------|
| Diagnostics (errors, warnings) | Works |
| Autocomplete (variables, functions, units) | Works |
| NL function completions (`average of`, `sum of`) | Works |
| Hover (variable values, function signatures, unit info) | Works |
| Go-to-definition | Works |
| Document symbols (outline view) | Works |
| Syntax highlighting | Not yet — no TextMate grammar in this setup |
| Live preview panel | Not yet — coming with the official extension |

### Limitations of the Dev Setup

The generic LSP client provides full language intelligence but not:

- **Syntax highlighting** — `.cm` files appear as plain text. The official extension will include a TextMate grammar for colored syntax.
- **Live preview** — the rendered document preview requires the official extension's webview panel.
- **Auto-install** — you manage the `cm` binary yourself. The official extension will detect or download it automatically.

## Troubleshooting

**No diagnostics or completions:**
- Open the Output panel (Ctrl+Shift+U) and select the LSP client channel to see server logs
- Verify `cm lsp` starts correctly: run `cm lsp` in a terminal — it should hang waiting for input (that's normal)
- Check that the file extension is `.cm` or `.calcmark`

**"Command not found" error:**
- The `cm` binary path in settings must be correct. Use an absolute path if in doubt.
- If you built from source, make sure you ran `task build` first

**Completions don't auto-popup:**
- Press Ctrl+Space to manually trigger completions
- VS Code should auto-popup for LSP completions by default, but check that `"editor.quickSuggestions"` is enabled in your settings

**Binary version mismatch:**
- If you're developing CalcMark, rebuild with `task build` after pulling changes to keep the LSP server up to date
