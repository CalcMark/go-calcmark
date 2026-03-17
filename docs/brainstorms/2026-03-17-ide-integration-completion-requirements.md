---
date: 2026-03-17
topic: ide-integration-completion
---

# IDE Integration Completion

## Problem Frame

The CalcMark LSP server and editor extensions (VS Code, Zed, tree-sitter) are ~60% complete against the plan (`docs/plans/2026-03-16-002-feat-ide-extensions-and-lsp-support-plan.md`). The LSP core handlers are solid (diagnostics, completion, hover, go-to-definition, document symbols, code actions, semantic tokens, signature help), but the VS Code extension is a minimal skeleton missing the "wow" feature (live preview), syntax highlighting, and robustness. Cross-cutting items (security hardening, test gates) and documentation consolidation are also incomplete.

Users installing the VS Code extension today get LSP intelligence but no syntax coloring and no live preview — the two features that make CalcMark's value proposition immediately obvious.

## Requirements

### VS Code Preview Chain

- R1. The LSP server sends a `calcmark/documentRendered` custom notification (`{uri: string, html: string}`) after each successful evaluation cycle.
- R2. The VS Code extension renders the notification HTML in a webview panel ("CalcMark: Open Preview" command) with `retainContextWhenHidden: true`.
- R3. The HTML formatter outputs `data-source-line` attributes on rendered blocks for scroll sync.
- R4. The VS Code preview panel scrolls to the corresponding source line when the editor cursor moves (editor-to-preview, unidirectional).
- R5. The webview uses a Content Security Policy: `default-src 'none'; style-src 'unsafe-inline'`.

### VS Code Syntax Highlighting

- R6. A TextMate grammar (`syntaxes/calcmark.tmLanguage.json`) provides baseline syntax highlighting for `.cm` and `.calcmark` files in VS Code. Scope naming follows TextMate conventions (`variable.other.calcmark`, `constant.numeric.calcmark`, `entity.name.function.calcmark`).
- R7. NL multi-token functions (`average of`, `sum of`, `square root of`) are highlighted as `entity.name.function.calcmark`.

### VS Code Robustness

- R8. Binary discovery calls `cm version` to validate the binary supports `lsp`. If the version is too old, show an actionable error message.
- R9. The LSP client restarts on crash with exponential backoff (max 3 retries), then degrades gracefully.

### Security Hardening

- R10. Add a bluemonday `.SanitizeBytes()` pass on gomarkdown output before the `template.HTML()` cast in `html_formatter.go`.
- R11. Update SECURITY.md with an LSP-specific section documenting: per-evaluation timeout (1s), document size limit (1MB), panic recovery, loopback-only binding for future `cm watch`.

### Test Gates

- R12. `TestEveryFeatureProducesValidCompletion` — every feature in `spec/features/registry.go` produces a valid LSP CompletionItem.
- R13. `TestAllFunctionCallsHaveRange` — every FunctionCall AST node has a non-zero Range (prevents hover/go-to-definition failures on NL functions).

### Documentation Consolidation

- R14. Merge `vscode-setup.md` (troubleshooting, dev/generic-client setup) and `neovim-setup.md` (nvim-cmp setup, troubleshooting) into `ide-setup.md` as one authoritative editor setup page.
- R15. Delete `vscode-setup.md` and `neovim-setup.md` after merging their unique content.
- R16. The consolidated page preserves: VS Code troubleshooting, Neovim nvim-cmp auto-completion setup, Neovim troubleshooting, and the dev/generic-client setup option.

## Success Criteria

- Installing the VS Code extension on a machine with `cm` in PATH provides syntax highlighting, diagnostics, autocomplete, hover, and a live preview panel — all working on first open of a `.cm` file.
- The Zed extension provides syntax highlighting (tree-sitter) and full LSP intelligence.
- `task test` and `task quality` pass with the new test gates.
- The CalcMark website has exactly one editor setup page with no duplicated content.

## Scope Boundaries

- **Not in scope:** VS Code Marketplace publishing, Zed extension registry publishing, `cm watch` subcommand, incremental document sync, WASM preview, adaptive debouncing.
- **Not in scope:** Refactoring `extension.ts` into 4 separate modules (binaryDiscovery.ts, lspClient.ts, previewPanel.ts). Keep it simple — split only if the file grows past 300 lines.
- **Not in scope:** Bidirectional scroll sync (preview-to-editor). Start with unidirectional.

## Key Decisions

- **One consolidated doc page:** `ide-setup.md` becomes the single reference. Standalone editor pages are deleted. Rationale: reduces maintenance burden and eliminates user confusion about which page is authoritative.
- **TextMate grammar, not tree-sitter, for VS Code:** VS Code's tree-sitter support remains experimental. TextMate is the proven path. Tree-sitter serves Zed and Neovim.
- **Unidirectional scroll sync:** Editor-to-preview only. Simpler to implement, covers the primary use case. Bidirectional can be added later if users request it.

## Outstanding Questions

### Deferred to Planning

- [Affects R6][Needs research] What is the minimum set of TextMate scopes needed for a good CalcMark highlighting experience? The plan lists several but prototyping may reveal gaps (e.g., frontmatter YAML, block separators).
- [Affects R10][Technical] bluemonday is an indirect dependency via glamour. Confirm it can be imported directly without version conflicts in go.mod.
- [Affects R1][Technical] GLSP's custom notification API — verify how to send server-initiated notifications (not responses) via GLSP. The mpls project may have examples.
- [Affects R3][Technical] Determine which HTML elements in `format/templates/default.html` should receive `data-source-line` attributes. Likely: each calc result block and each heading.

## Next Steps

`/ce:plan` for structured implementation planning.
