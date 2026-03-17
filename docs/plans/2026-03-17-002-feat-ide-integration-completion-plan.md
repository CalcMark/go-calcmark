---
title: "feat: IDE integration completion — preview, TextMate grammar, security, docs"
type: feat
status: completed
date: 2026-03-17
origin: docs/brainstorms/2026-03-17-ide-integration-completion-requirements.md
---

# IDE Integration Completion

## Overview

Complete the remaining gaps between the IDE/LSP plan (`docs/plans/2026-03-16-002-feat-ide-extensions-and-lsp-support-plan.md`) and the current implementation. The LSP server core is ~95% done. The critical missing pieces are: VS Code live preview chain (custom notification + webview + scroll sync), TextMate grammar, binary version validation, security hardening (bluemonday), test consistency gates, and documentation consolidation.

(see origin: `docs/brainstorms/2026-03-17-ide-integration-completion-requirements.md`)

## Problem Statement / Motivation

Users installing the VS Code extension today get LSP intelligence (diagnostics, autocomplete, hover, go-to-definition) but no syntax coloring and no live preview — the two features that make CalcMark's value proposition immediately obvious. The "wow" moment is missing.

Additionally, the CalcMark website has three overlapping editor setup pages creating user confusion about which is authoritative.

## Proposed Solution

Four implementation phases, ordered by dependency chain:

1. **Cross-cutting foundation** — `data-source-line` attributes + bluemonday sanitization in the format layer
2. **LSP custom notification** — `calcmark/documentRendered` from the server
3. **VS Code extension completion** — TextMate grammar, preview panel, binary validation, crash recovery
4. **Cleanup** — test gates, SECURITY.md, documentation consolidation

## Technical Approach

### Phase 1: Cross-Cutting Foundation (Go)

These changes enable both the preview chain and security hardening.

#### 1a. Add `data-source-line` attributes to HTML output (R3)

**File: `format/html_formatter.go`**

Extend `TemplateLine` (line ~43) with a `DocLine int` field:

```go
type TemplateLine struct {
    Source   string
    Result   string
    DocLine  int  // document-absolute line number for scroll sync
}
```

In the `Format()` method, when building `TemplateLine` slices for each CalcBlock, populate `DocLine` from the block's starting position plus the line offset within the block. Use `block.Range().Start.Line` (1-indexed, document-absolute) as the base.

For TextBlocks, add a `DocLine` field to the text block template data and populate from the block's starting position.

**File: `format/templates/default.html`**

Add `data-source-line` attributes:

- On each `<div class="calc-line">`: `data-source-line="{{$line.DocLine}}"`
- On each `<div class="text-block">`: `data-source-line="{{.DocLine}}"`

**Gotcha (learning: statement index drift):** When iterating source lines to build `TemplateLine` slices, use a separate `resultIdx` counter that only advances for non-blank source lines. Never use the loop index `i` to index into results. This is the most common bug class in the format/ package (4+ instances documented in `docs/solutions/ui-bugs/context-footer-statement-index-drift.md`).

**Test:** Verify `data-source-line` appears in HTML output for a document with calc blocks and text blocks. Add to existing HTML formatter tests in `format/html_formatter_test.go`.

#### 1b. Add bluemonday HTML sanitization (R10)

**File: `format/html_formatter.go`**

Import `github.com/microcosm-cc/bluemonday` (already an indirect dependency via glamour at v1.0.27 in go.mod).

In the text block rendering path, after gomarkdown renders HTML and before the `template.HTML()` cast, add:

```go
p := bluemonday.UGCPolicy()
sanitized := p.Sanitize(renderedHTML)
tb.HTML = template.HTML(sanitized)
```

Use `UGCPolicy()` — it allows safe HTML tags from markdown rendering (headings, lists, links, code blocks, emphasis) while stripping anything dangerous (script tags, event handlers, data URIs).

**Note:** The `spec/document/markdown.go` `Render()` function already uses gomarkdown with `html.SkipHTML` to strip raw HTML tags, but this is regex-based, not a proper sanitizer. bluemonday provides defense-in-depth against future gomarkdown bypasses.

**Test:** Add a test case with a document containing markdown that gomarkdown renders. Verify the output contains expected tags but not script/event handler content. Also add the `data:` URI test case to `TestMarkdownUnsafeLinkSchemes` in `spec/document/markdown_test.go` (gap identified by the security review in the original plan).

**Dependency:** Run `go get github.com/microcosm-cc/bluemonday` to promote from indirect to direct dependency.

### Phase 2: LSP Custom Notification (Go)

#### 2a. Send `calcmark/documentRendered` notification (R1)

**File: `lsp/diagnostics.go`**

After the `publishDiagnostics` call in the debounced evaluation callback (around line 124-132), add a `calcmark/documentRendered` notification:

```go
// Define custom notification params
type DocumentRenderedParams struct {
    URI  string `json:"uri"`
    HTML string `json:"html"`
}

// After publishDiagnostics, send rendered HTML
html := s.renderHTML(snapshot)
ctx.Notify("calcmark/documentRendered", DocumentRenderedParams{
    URI:  params.TextDocument.URI,
    HTML: html,
})
```

The `renderHTML` method on `Server` creates an `HTMLFormatter` and calls `Format()` on the snapshot's document. Cache the compiled HTML template via `sync.Once` (from the performance recommendations in the original plan — `template.New("html").Parse(templateContent)` costs 10-50us per call).

**Key insight (learning: debounce architecture):** The `ctx` (`*glsp.Context`) is already captured from the most recent `didOpen`/`didChange` notification. The `ctx.Notify()` mechanism is the same one used for `publishDiagnostics`. No new GLSP API needed.

**File: `lsp/server.go`**

Add an `htmlFormatter` field to the `Server` struct, initialized once in `NewServer()`. This avoids re-parsing the HTML template on every evaluation.

**Test:** In `lsp/server_test.go`, verify that after evaluating a document, the server's snapshot contains rendered HTML. (Direct notification testing requires a mock GLSP context, which is out of scope — the unit test verifies the HTML is generated correctly.)

### Phase 3: VS Code Extension Completion (TypeScript)

#### 3a. TextMate grammar (R6, R7)

**New file: `editors/vscode-calcmark/syntaxes/calcmark.tmLanguage.json`**

Create a TextMate grammar with `scopeName: "source.calcmark"`. Key patterns:

| CalcMark concept | TextMate scope | Pattern |
|-----------------|----------------|---------|
| Markdown headings | `markup.heading.calcmark` | `^#{1,6}\s` |
| Variable assignment LHS | `variable.other.calcmark` | `^[a-zA-Z_]\w*(?=\s*=)` |
| Number literals | `constant.numeric.calcmark` | `\d[\d,_]*(\.\d+)?([eE][+-]?\d+)?[kKmMbBtT]?` |
| Currency literals | `constant.numeric.currency.calcmark` | `\$[\d][\d,_]*(\.\d+)?` |
| Percentage | `constant.numeric.percentage.calcmark` | `\d[\d,_]*(\.\d+)?%` |
| Function names | `entity.name.function.calcmark` | `[a-zA-Z_]\w*(?=\()` |
| NL functions | `entity.name.function.calcmark` | `\b(average\s+of\|sum\s+of\|square\s+root\s+of)\b` (case insensitive) |
| Keywords | `keyword.other.calcmark` | `\b(in\|as\|and\|or\|not\|true\|false)\b` |
| Operators | `keyword.operator.calcmark` | `[+\-*/^%=<>!]+` |
| Comments (markdown text) | `comment.line.calcmark` | Lines that don't match any calc pattern (fallback) |
| Strings | `string.quoted.double.calcmark` | `"[^"]*"` |
| Frontmatter | `comment.block.frontmatter.calcmark` | `^---$` delimited block at document start |

**Sync requirement (from original plan):** Any new NL function alias added to `spec/features/registry.go` must also be added to the TextMate grammar. Document this in a comment at the top of the grammar file.

**File: `editors/vscode-calcmark/package.json`**

Add to `contributes`:

```json
"grammars": [
  {
    "language": "calcmark",
    "scopeName": "source.calcmark",
    "path": "./syntaxes/calcmark.tmLanguage.json"
  }
]
```

#### 3b. Preview panel (R2, R4, R5)

**File: `editors/vscode-calcmark/src/extension.ts`**

Add preview panel functionality directly in `extension.ts` (not a separate file — see origin scope boundary: "split only if the file grows past 300 lines").

**Custom notification handler:**

```typescript
interface DocumentRenderedParams {
  readonly uri: string;
  readonly html: string;
}

client.onNotification('calcmark/documentRendered', (params: unknown) => {
  const rendered = params as DocumentRenderedParams;
  if (rendered?.uri && rendered?.html && previewPanel) {
    updatePreview(rendered);
  }
});
```

**Preview command:** Register a `calcmark.openPreview` command in `package.json` and handle it in `activate()`:

```typescript
const openPreview = vscode.commands.registerCommand('calcmark.openPreview', () => {
  previewPanel = vscode.window.createWebviewPanel(
    'calcmarkPreview',
    'CalcMark Preview',
    vscode.ViewColumn.Beside,
    { enableScripts: false, retainContextWhenHidden: true }
  );
  previewPanel.onDidDispose(() => { previewPanel = undefined; });
});
context.subscriptions.push(openPreview);
```

**Content Security Policy (R5):**

```typescript
function getPreviewHTML(html: string): string {
  return `<!DOCTYPE html>
<html>
<head>
  <meta http-equiv="Content-Security-Policy"
        content="default-src 'none'; style-src 'unsafe-inline';">
</head>
<body>${html}</body>
</html>`;
}
```

**Scroll sync (R4):** Unidirectional editor-to-preview. Since `enableScripts: false` in the webview, we cannot use `postMessage` for scroll sync. Instead, on editor visible range change, find the corresponding `data-source-line` element and set it as an anchor in the HTML:

```typescript
vscode.window.onDidChangeTextEditorVisibleRanges((e) => {
  if (!previewPanel || e.textEditor.document.languageId !== 'calcmark') return;
  const topLine = e.visibleRanges[0]?.start.line ?? 0;
  // Re-render with anchor: add id="scroll-target" to matching data-source-line element
  updatePreviewWithAnchor(topLine);
});
```

**Edge case: preview closed but notifications arrive** — The `previewPanel` variable is set to `undefined` on dispose. The notification handler checks `if (previewPanel)` before updating. No-op when panel is closed.

**Edge case: multiple .cm files open** — The notification includes `uri`. Track which URI the preview is showing. Only update when the notification URI matches. If the user switches active editors, update the preview to show the new file.

**File: `editors/vscode-calcmark/package.json`**

Add the command:

```json
"commands": [
  {
    "command": "calcmark.openPreview",
    "title": "CalcMark: Open Preview",
    "icon": "$(open-preview)"
  }
]
```

Add `extensionKind` for Codespaces support:

```json
"extensionKind": ["workspace"]
```

#### 3c. Binary version validation (R8)

**File: `editors/vscode-calcmark/src/extension.ts`**

In `findBinary()`, after locating the binary, run `cm version` and parse the output. If the version doesn't support `lsp` (pre-LSP versions), show:

```typescript
async function validateBinary(cmPath: string): Promise<boolean> {
  try {
    const { stdout } = await execFile(cmPath, ['version']);
    // cm version outputs "cm version X.Y.Z" — parse and check minimum
    return true; // version check passes
  } catch {
    window.showErrorMessage(
      `CalcMark: '${cmPath}' does not support LSP. Update to the latest version.`
    );
    return false;
  }
}
```

As a simpler first pass: just try running `cm lsp --help` or `cm version`. If it fails, the binary is too old. The exact version parsing can come later.

#### 3d. Crash recovery with exponential backoff (R9)

The `vscode-languageclient` library already provides restart behavior. Configure it:

```typescript
const clientOptions: LanguageClientOptions = {
  documentSelector: [{ scheme: "file", language: "calcmark" }],
  outputChannelName: "CalcMark LSP",
  errorHandler: {
    error: () => ({ action: ErrorAction.Continue }),
    closed: () => ({
      action: CloseAction.Restart,
      message: "CalcMark LSP server crashed. Restarting..."
    }),
  },
};
```

The built-in client handles exponential backoff and max retries. Configure `maxRestartCount: 3` if supported by the client version.

### Phase 4: Cleanup

#### 4a. Test consistency gates (R12, R13)

**File: `lsp/completion_test.go` (new or add to existing)**

**R12: `TestEveryFeatureProducesValidCompletion`**

```go
func TestEveryFeatureProducesValidCompletion(t *testing.T) {
    registry := features.NewRegistry()
    for _, f := range registry.All() {
        t.Run(f.Name, func(t *testing.T) {
            // Build a completion item the same way the LSP does
            item := featureToCompletionItem(f)
            if item.Label == "" {
                t.Errorf("feature %q produced empty completion label", f.Name)
            }
            // Verify Kind is set
            if item.Kind == 0 {
                t.Errorf("feature %q produced zero completion kind", f.Name)
            }
        })
    }
}
```

This mirrors the existing `TestEveryBuiltinFunctionHasFunctionSpec` pattern.

**File: `spec/parser/parser_test.go` or `lsp/hover_test.go`**

**R13: `TestAllFunctionCallsHaveRange`**

Parse golden examples from `testdata/` that contain function calls. Walk the AST and verify every `FunctionCall` node has a non-zero `Range`. This catches the NL function range bug documented in `docs/solutions/logic-errors/nl-function-missing-ast-range.md`.

```go
func TestAllFunctionCallsHaveRange(t *testing.T) {
    // Parse documents with function calls from testdata/
    // Walk AST, assert Range.Start.Line > 0 for every FunctionCall
}
```

**Test philosophy (learning: test behavior not implementation):** These tests verify observable properties (completions exist, ranges exist) not internal formulas. If the production code were rewritten, these tests would still catch bugs.

#### 4b. SECURITY.md LSP section (R11)

**File: `SECURITY.md`**

Add a new section:

```markdown
## Language Server Protocol (`cm lsp`)

The LSP server inherits all protections from the evaluation pipeline:

- **Per-evaluation timeout**: Each evaluation is wrapped in `context.WithTimeout` (1 second). Malicious documents cannot hang the server.
- **Document size limit**: Documents larger than 1MB are rejected on `textDocument/didOpen` and `textDocument/didChange` with a diagnostic.
- **Panic recovery**: The evaluation entry point is wrapped in `recover()`. The server logs the panic, publishes a diagnostic, and continues serving. It never crashes on malformed input.
- **HTML sanitization**: Rendered HTML passes through bluemonday before being sent to editor webviews, preventing XSS even if gomarkdown has a bypass.
- **Loopback-only binding**: The future `cm watch` HTTP server binds to `127.0.0.1` only, never `0.0.0.0`. A random session token in the URL prevents cross-origin access.
```

#### 4c. Documentation consolidation (R14, R15, R16)

**File: `site/content/docs/ide-setup.md`**

Merge unique content from the two standalone pages:

From `vscode-setup.md`, add to the VS Code section:
- Dev setup with generic LSP client (`zsol.vscode-glspc`) as a collapsible section
- Full troubleshooting section (6 specific issues: no diagnostics, command not found, completions don't auto-popup, server crashes -32097, AI autocomplete blocking, binary version mismatch)

From `neovim-setup.md`, add to the Neovim section:
- nvim-cmp auto-completion setup (lazy.nvim and packer.nvim examples)
- Verification steps (`vim.print(vim.lsp.get_clients())`)
- Troubleshooting section (LSP not attaching, no diagnostics, completions not showing)

**Files to delete:** `site/content/docs/vscode-setup.md`, `site/content/docs/neovim-setup.md`

**Weight:** Keep `ide-setup.md` at weight 23. No new pages needed.

## System-Wide Impact

### Interaction Graph

`calcmark/documentRendered` notification fires after every debounced evaluation cycle: `didChange` → 150ms debounce → `evaluate()` → `publishDiagnostics` → `renderHTML()` → `calcmark/documentRendered`. The VS Code extension receives the notification → updates webview HTML → browser renders.

### Error Propagation

- Evaluation errors become diagnostics (existing path). Preview still renders — it shows the last successful evaluation's HTML with error indicators.
- If `renderHTML()` fails (e.g., template error), log the error and skip the notification. Do not block diagnostics.
- bluemonday sanitization is a pure transform — it cannot fail. If the input is empty, the output is empty.

### State Lifecycle Risks

- **Preview panel disposed while notification in flight**: The notification handler checks `if (previewPanel)` before updating. Safe.
- **Multiple documents open**: Track `activePreviewURI`. Only update preview when notification URI matches. Switch on active editor change.
- **Large documents near 1MB**: The 1MB limit is already enforced. HTML rendering of a 1000-line document takes ~4ms (from original plan benchmarks). No risk.

### API Surface Parity

The `calcmark/documentRendered` notification is a new VS Code-specific feature. It does not affect the TUI, CLI, or Zed extension. Zed users get tree-sitter highlighting + LSP semantic tokens but no preview (Zed limitation).

### Integration Test Scenarios

1. Open a `.cm` file → TextMate grammar highlights keywords, numbers, functions → LSP semantic tokens override for calc vs markdown classification
2. Type `a = 1 + 1` → diagnostics clear → preview updates showing `a = 2` → hover over `a` shows `2`
3. Close preview panel → continue editing → reopen preview → shows current state
4. Open file with `cm` binary missing → error message with install instructions → no crash

## Acceptance Criteria

### Functional Requirements

- [ ] `calcmark/documentRendered` notification sent after each evaluation (R1)
- [ ] VS Code preview panel renders HTML with CSP (R2, R5)
- [ ] `data-source-line` attributes appear on calc-line and text-block elements (R3)
- [ ] Preview scrolls to editor cursor position (R4)
- [ ] TextMate grammar highlights all major token types (R6)
- [ ] NL functions highlighted as `entity.name.function` (R7)
- [ ] Binary version validated on extension startup (R8)
- [ ] LSP client restarts on crash (R9)
- [ ] gomarkdown output sanitized by bluemonday (R10)
- [ ] SECURITY.md documents LSP protections (R11)
- [ ] Every feature produces a valid CompletionItem (R12)
- [ ] Every FunctionCall AST node has a non-zero Range (R13)
- [ ] One authoritative ide-setup.md page (R14)
- [ ] vscode-setup.md and neovim-setup.md deleted (R15)
- [ ] Consolidated page includes troubleshooting and nvim-cmp content (R16)

### Quality Gates

- [ ] `task test` passes
- [ ] `task quality` passes
- [ ] VS Code extension loads and connects to `cm lsp` in a fresh VS Code window
- [ ] Preview panel shows rendered HTML on first open

## Dependencies & Prerequisites

- bluemonday: promote from indirect to direct dependency (`go get github.com/microcosm-cc/bluemonday`)
- GLSP `ctx.Notify()` for custom notifications — already used for `publishDiagnostics`, same mechanism
- `vscode-languageclient` v9 — already in package.json

## Risk Analysis & Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Scroll sync jitter from re-rendering full HTML on every cursor move | Medium | Low | Only update anchor on visible range change, not cursor position. Debounce if needed. |
| bluemonday strips legitimate markdown HTML tags | Low | Medium | Use `UGCPolicy()` which preserves safe tags. Test with representative CalcMark documents. |
| TextMate grammar conflicts with LSP semantic tokens | Low | Low | LSP semantic tokens override TextMate by default. Grammar provides baseline; tokens refine. |
| Statement index drift in data-source-line population | Medium | High | Use separate `resultIdx` counter per learning #4. Add regression test. |

## Sources & References

### Origin

- **Origin document:** [docs/brainstorms/2026-03-17-ide-integration-completion-requirements.md](docs/brainstorms/2026-03-17-ide-integration-completion-requirements.md) — Key decisions: one consolidated doc page, TextMate for VS Code, unidirectional scroll sync
- **Parent plan:** [docs/plans/2026-03-16-002-feat-ide-extensions-and-lsp-support-plan.md](docs/plans/2026-03-16-002-feat-ide-extensions-and-lsp-support-plan.md) — Comprehensive architecture and decisions for the full IDE/LSP vision

### Internal References (Learnings Applied)

- Statement index drift: `docs/solutions/ui-bugs/context-footer-statement-index-drift.md` — separate resultIdx counter
- LSP debounce architecture: `docs/solutions/integration-issues/lsp-debounce-staleness-read-requests.md` — immediate source, debounced evaluation
- Unified feature registry: `docs/solutions/code-organization/unified-feature-registry-three-to-one.md` — use `spec/features.Feature` as single source
- DocLine positions: `docs/solutions/code-organization/docline-diagnostic-line-numbers.md` — 1-indexed, document-absolute
- NL/functional parity: `docs/solutions/integration-issues/nl-functional-syntax-parity-and-doc-staleness.md` — test both paths independently
- Test behavior not implementation: `docs/solutions/test-failures/test-behavior-not-implementation.md`

### Internal References (Code)

- GLSP notification pattern: `lsp/diagnostics.go:124-127` — `ctx.Notify(method, params)`
- Server handler registration: `lsp/server.go:94-112`
- HTML formatter: `format/html_formatter.go`
- HTML template: `format/templates/default.html`
- TemplateLine struct: `format/html_formatter.go:43`
- VS Code extension: `editors/vscode-calcmark/src/extension.ts`
- VS Code package.json: `editors/vscode-calcmark/package.json`
- Feature registry: `spec/features/registry.go`
- Suggestions interface: `spec/features/suggestions.go`
