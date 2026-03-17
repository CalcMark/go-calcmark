---
title: "feat: IDE extensions and LSP support for CalcMark"
type: feat
status: active
date: 2026-03-16
deepened: 2026-03-16
origin: docs/brainstorms/2026-03-16-ide-extensions-and-lsp-brainstorm.md
---

# IDE Extensions and LSP Support for CalcMark

## Enhancement Summary

**Deepened on:** 2026-03-16
**Agents used:** architecture-strategist, performance-oracle, security-sentinel, code-simplicity-reviewer, pattern-recognition-specialist, kieran-typescript-reviewer, new-calcmark-feature skill, learnings-researcher, Context7 (tree-sitter, vscode-languageserver-node)

### Key Improvements
1. **Restructured from 7 phases to 3** — Ship `cm lsp` + VS Code extension first, defer tree-sitter/Zed/`cm watch` to Phase 3
2. **Critical dependency fix** — Extract `Suggestion`/`SuggestionSource` from `cmd/tui/components/` to `spec/features/` to avoid import violation
3. **Performance optimization** — Use single-pass `Evaluate()` instead of `EvaluateBlock()` (halves eval time); cache HTML template; immutable snapshots
4. **Security hardening** — Per-evaluation timeouts, loopback-only HTTP binding, WebSocket origin validation, binary trust model, bluemonday HTML sanitization
5. **VS Code extension architecture** — 4-module split, CSP for webview, `data-source-line` attributes for scroll sync, severity mapping

### New Considerations Discovered
- `lsp/` cannot import `cmd/calcmark/tui/components/` — requires extracting shared types to `spec/features/`
- Two-pass `EvaluateBlock()` is unnecessary for LSP — single-pass `Evaluate()` halves CPU work
- `template.HTML()` cast in HTML formatter trusts gomarkdown without secondary sanitization — add bluemonday pass
- VS Code webview needs explicit Content Security Policy and `data-source-line` attributes (cross-cutting Go change)
- CalcMark is rune-oriented throughout — negotiate UTF-32 position encoding with LSP clients to avoid lossy UTF-16 conversion

---

## Overview

Build a Language Server Protocol (LSP) server in Go (`cm lsp` subcommand), a TextMate grammar for VS Code, and a VS Code extension with live preview. The goal is **adoption** — people discover CalcMark through their editor, see a live rendered document preview, and get the full IDE experience (diagnostics, autocomplete, hover, go-to-definition, document symbols).

The architecture is **LSP-first**: one Go server serves all editors, the `cm` binary ships it for free via GoReleaser, and editor extensions are thin wrappers that spawn `cm lsp` and wire up UI.

### Research Insights: Scope

The simplicity review recommended aggressively focusing Phase 1 on **`cm lsp` + VS Code extension only**. VS Code covers ~70% of developers. Tree-sitter grammar, Zed extension, `cm watch`, semantic tokens, formatting, and code actions are deferred to later phases gated on user feedback — not planned upfront. This cuts ~50% of initial scope while preserving the core value proposition.

## Problem Statement / Motivation

CalcMark's built-in TUI editor is excellent, but most developers live in VS Code, Zed, Neovim, or Emacs. Requiring them to switch tools is a friction point for adoption. By bringing CalcMark's language intelligence and live preview into native editors, we:

1. Lower the barrier to trying CalcMark (install an extension, open a `.cm` file, see results)
2. Meet developers where they already are
3. Leverage the TUI's proven patterns (diagnostics, autocomplete, variable state) over a standard protocol

(see brainstorm: `docs/brainstorms/2026-03-16-ide-extensions-and-lsp-brainstorm.md`)

## Proposed Solution

### Architecture

```
┌─────────────┐     stdio      ┌──────────────────────┐
│  VS Code    │◄──────────────►│                      │
│  Extension  │                │    cm lsp            │
├─────────────┤                │                      │
│  Neovim     │◄──────────────►│  (Go LSP server      │
│  (built-in) │                │   using GLSP)        │
├─────────────┤                │                      │
│  Emacs      │◄──────────────►│  Consumes:           │
│  (eglot)    │                │  - spec/semantic     │
├─────────────┤                │  - spec/features     │
│  Zed        │◄──────────────►│  - impl/document     │
│  (Phase 3)  │                │  - format/html       │
└─────────────┘                └──────────────────────┘
```

### Key Technical Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| LSP library | **GLSP** (`github.com/tliron/glsp`) | Clean handler API, proven in mpls (Markdown Preview LSP), supports stdio/TCP/WebSocket |
| Document sync | **Full sync** first | CalcMark docs are small (<1MB limit), avoids sync bugs. Incremental later if needed |
| Evaluation strategy | **Single-pass `Evaluate()`** | Performance review found `EvaluateBlock()` does 2 full passes with environment cloning. Single-pass halves CPU work. |
| Syntax highlighting | **TextMate grammar** for VS Code (Phase 1); **Tree-sitter** for Zed/Neovim/Emacs (Phase 3) | VS Code's tree-sitter API is experimental. Ship what works now. |
| Preview transport | **Custom LSP notification** `calcmark/documentRendered` for VS Code webview | `cm watch` deferred to Phase 3 when non-VS-Code users request it |
| Evaluation debounce | **150ms debounce** on `didChange` before re-evaluation | Prevents server overload at 10+ keystrokes/sec. Cancels in-flight evaluation on new change. |
| Position encoding | **UTF-32** negotiated via LSP `positionEncoding` capability (LSP 3.17) | CalcMark is rune-oriented throughout (TUI uses `runeSlice`/`runeLen`). Avoids lossy UTF-16 conversion. Fallback to UTF-16 if client doesn't support it — verify GLSP supports this negotiation. |
| Code location | **Monorepo** (go-calcmark) | Single repo, no version drift, shared CI (see brainstorm decision #8) |

### Repo Structure

```
lsp/                           # LSP server implementation (Go)
  server.go                    # Server setup, handler registration, GLSP wiring, lifecycle
  diagnostics.go               # publishDiagnostics (maps semantic.Diagnostic → LSP)
  completion.go                # textDocument/completion (+ featureToCompletionItem conversion)
  hover.go                     # textDocument/hover (+ token identification)
  position.go                  # Shared position conversion (CalcMark 1-indexed → LSP 0-indexed)
  server_test.go               # Data-driven integration tests

cmd/calcmark/cmd/
  lsp.go                       # `cm lsp` Cobra subcommand (thin: RunE → runLsp())

editors/                       # External editor integrations (non-Go)
  vscode-calcmark/             # VS Code extension
    package.json               # Extension manifest
    src/
      extension.ts             # Activation only: register commands, wire modules
      binaryDiscovery.ts       # Find cm binary, validate version, prompt user
      lspClient.ts             # Create/configure LanguageClient, handle custom notifications
      previewPanel.ts          # Webview lifecycle, HTML updates, scroll sync
    syntaxes/calcmark.tmLanguage.json  # TextMate grammar
    language-configuration.json
```

### Research Insights: Package Placement

**`lsp/` at the top level is correct** (architecture + pattern reviewers agree). It mirrors the existing `format/` package: a protocol adapter consuming `spec/`, `impl/`, and `format/` that provides a distinct capability. It is not part of `spec/` (doesn't define the language), not part of `impl/` (doesn't evaluate expressions), and not part of `cmd/` (contains substantial testable library logic). Add a package doc comment:

> `lsp/` is a protocol adapter that bridges CalcMark's spec/impl/format layers to the Language Server Protocol. Dependencies flow inward: `lsp/` → `spec/`, `lsp/` → `impl/`, `lsp/` → `format/`. No package should ever import `lsp/`.

**`editors/` (not `editor/`)** to avoid ambiguity with the existing `cmd/calcmark/tui/editor/` package (100+ files). The plural emphasizes these are configurations for multiple external editors.

**No `convert.go` monolith.** Co-locate type-mapping functions with each handler (`diagnostics.go` contains `diagnosticToLSP()`, `completion.go` contains `featureToCompletionItem()`). Only `position.go` is shared — position conversion is a true cross-cutting concern.

**Start with 5-6 files, not 13.** Split when files grow past 300-400 lines. The Phase 1 LSP server needs: `server.go` (lifecycle + state + sync), `diagnostics.go`, `completion.go`, `hover.go`, `position.go`, `server_test.go`.

### Research Insights: Prerequisite Refactoring

**CRITICAL — Dependency violation to fix before Phase 1:**

The plan originally said to "wire up the three existing suggestion sources from `cmd/calcmark/tui/components/suggest.go`." This creates an illegal dependency: `lsp/` → `cmd/calcmark/tui/components/`. The current architecture never imports `cmd/` packages from library packages.

**Fix:** Extract `Suggestion` struct and `SuggestionSource` interface to `spec/features/suggestions.go`. Both the TUI and the LSP then import from `spec/features/`. The TUI-specific types (`AutosuggestState`, rendering logic) stay in `cmd/calcmark/tui/components/`.

Similarly, `LineResult` from `cmd/calcmark/tui/editor/results.go` is TUI-internal (contains `WasChanged`, `IsBlocked`, `PopupRow` fields). The LSP should consume `spec/document` and `impl/document` directly, building its own LSP-specific mappings.

## Technical Approach

### Phase 1: `cm lsp` + VS Code Extension (Ship It)

The minimum viable product that achieves the adoption goal: install VS Code extension, open `.cm` file, see diagnostics + autocomplete + hover + live preview.

#### Phase 1a: LSP Server Core

**Preparatory tasks:**
1. Extract `Suggestion`/`SuggestionSource` from `cmd/calcmark/tui/components/suggest.go` to `spec/features/suggestions.go`
2. Add GLSP dependency (`github.com/tliron/glsp`) to `go.mod`
3. Create `lsp/` package with `server.go` (lifecycle, handler registration, document state)
4. Create `cmd/calcmark/cmd/lsp.go` Cobra subcommand following `RunE` → `runLsp()` delegation pattern

**Document synchronization:**
5. Implement `textDocument/didOpen` — parse document, evaluate using single-pass `Evaluate()`, create immutable snapshot, store state
6. Implement `textDocument/didChange` (full sync, `TextDocumentSyncKind=1`) — cancel any in-flight evaluation, debounce 150ms, re-parse, re-evaluate, produce new snapshot
7. Implement `textDocument/didClose` — cleanup state, release snapshot

**Diagnostics:**
8. Implement `textDocument/publishDiagnostics` — map `document.Diagnostic` to LSP diagnostics
   - Use `DocLine` for document-absolute positions (see learnings: `docline-diagnostic-line-numbers.md`)
   - Consume `Detailed` field directly for diagnostic message context (see learnings: `diagnostic-detailed-field-pipeline.md`)
   - Convert 1-indexed CalcMark positions to 0-indexed LSP positions via `position.go`
   - Severity mapping: CalcMark `Error` → LSP `Error`, `Warning` → LSP `Warning`, `Hint` → LSP `Hint` (not `Information` — Hint renders as faded ellipsis in VS Code, which is the right UX for soft hints)

**Autocomplete:**
9. Implement `textDocument/completion` — wire up function, unit, and variable suggestion sources from the extracted `spec/features/suggestions.go`
   - Trigger characters: alphabetic characters only
   - Context-sensitive filtering: after `=` → all sources; after `in`/`as` → units only; after `@globals.` → global names only; markdown-classified lines → suppress completions
   - Handle NL multi-token completions (`average of`, `sum of`) — use `Feature.Aliases` with `Parseable=true` filter
   - Map to LSP `CompletionItem` with `Kind` (Variable, Function, Unit, Keyword) co-located in `completion.go`

**Hover:**
10. Implement `textDocument/hover` — identify token under cursor using AST position data
    - Variables: `name = value` with type and definition line
    - Functions: syntax, description, NL example from `features.Registry`
    - Units: full name, category, canonical aliases
    - Format as Markdown for rich rendering

**Go-to-definition:**
11. Implement `textDocument/definition` — variable references → assignment line via `semantic.Environment.GetInfo(name).Range`

**Document symbols:**
12. Implement `textDocument/documentSymbol` — variable assignments as `SymbolKind.Variable`, markdown headings as `SymbolKind.String`

**Preview notification:**
13. After each evaluation cycle, generate HTML using `format.HTMLFormatter.Format()` (with cached compiled template) and send `calcmark/documentRendered` notification: `{uri: string, html: string}`
14. Add `data-source-line` attributes to HTML formatter template output for scroll sync (cross-cutting change to `format/templates/default.html`)

**Safety:**
15. Wrap the evaluation entry point in `recover()` — the LSP server must never crash on malformed input. Log the panic, publish a diagnostic, and continue serving.
16. Add `data:` URI test case to `TestMarkdownUnsafeLinkSchemes` in `markdown_test.go` (gap identified by security review)

**Testing:**
17. Data-driven integration tests using `cockroachdb/datadriven` — send LSP requests, assert on responses via golden files
18. Consistency test: `TestEveryFeatureProducesValidCompletion` (mirrors existing `TestEveryBuiltinFunctionHasFunctionSpec` pattern)
19. AST Range completeness gate: `TestAllFunctionCallsHaveRange` (from learnings: `nl-function-missing-ast-range.md`)

### Research Insights: Performance (Phase 1)

From the performance review, with concrete numbers:

| Document Size | Lines | Single-pass Eval | HTML Render | Total |
|---|---|---|---|---|
| Small | 100 | ~1ms | ~500us | ~1.5ms |
| Medium | 500 | ~5ms | ~2ms | ~7ms |
| Large | 1000 | ~10ms | ~4ms | ~14ms |

All sizes fit comfortably within the 150ms debounce window.

**Must-do optimizations:**
1. **Use `Evaluate()` not `EvaluateBlock()`** — The two-pass reactive evaluation in `EvaluateBlock()` exists for the TUI where later assignments override earlier ones. The LSP only needs single top-down pass. This eliminates all `Environment.Clone()` overhead (O(V * B) map copying per keystroke).
2. **Cache compiled HTML template via `sync.Once`** — `template.New("html").Parse(templateContent)` costs 10-50us per call and re-parses on every `Format()`. Parse once at startup.
3. **Diff interpolated values before marking dirty** — `interpolateTextBlocks` unconditionally calls `SetDirty(true)`, triggering gomarkdown re-render (50-200us per block) even when values haven't changed.

**State management:**
4. **Immutable `DocumentSnapshot`** — The evaluator produces a frozen snapshot (blocks, diagnostics, variables, rendered HTML). LSP request handlers read the latest snapshot without locks. Classic single-writer / multiple-reader pattern. Prevents the stale-closure bug documented in learnings (`go-closure-capturing-stale-value-type.md`) — use pointer indirection for shared mutable state.
5. **Parse once, serve many** — One evaluation pass feeds diagnostics, completions, hover, preview. Never re-parse per-request. (From learnings: `pure-functional-layout-calculations.md`)

### Research Insights: Security (Phase 1)

**Per-evaluation timeout (HIGH priority):**
Add `context.WithTimeout` (1 second) around every evaluation. SECURITY.md documents this pattern but no code enforces it. A long-running LSP server must never hang on a malicious document.

**Document size limit enforcement:**
Enforce the existing 1MB limit on `textDocument/didOpen` and `textDocument/didChange`. Reject oversized documents with a diagnostic, don't attempt to parse.

**Panic safety:**
The LSP server must never crash. Extend the existing fuzz test ("NewDocument must never panic on any input") to cover the LSP evaluation entry point. The NaN/Inf/YAML panic vector (from learnings: `nan-inf-panic-yaml-frontmatter-scale.md`) applies directly since the LSP parses YAML frontmatter.

**HTML sanitization (defense-in-depth):**
Add a bluemonday `.SanitizeBytes()` pass on gomarkdown output before the `template.HTML()` cast in `html_formatter.go`. bluemonday is already an indirect dependency (via glamour). This protects against future gomarkdown bypasses.

#### Phase 1b: VS Code Extension

**Module structure (4 files, not 1):**

```
src/
  extension.ts          # Activation only: register commands, wire modules
  binaryDiscovery.ts    # Find cm binary, validate version, prompt user
  lspClient.ts          # Create/configure LanguageClient, handle custom notifications
  previewPanel.ts       # Webview lifecycle, HTML updates, scroll sync
```

**Binary discovery (`binaryDiscovery.ts`):**
1. Check `calcmark.binaryPath` VS Code setting (highest priority — user-configured)
2. Search PATH for `cm` (never search `node_modules/.bin/` or project-local `bin/` — PATH injection risk)
3. Call `cm version` to validate the binary version matches extension expectations
4. If not found, show actionable error with install instructions
5. Handle macOS Gatekeeper `EACCES`/`EPERM` with specific error message
6. Listen for `workspace.onDidChangeConfiguration` — restart LSP client when `calcmark.binaryPath` changes

**LSP client (`lspClient.ts`):**
1. Spawn `cm lsp` via `vscode-languageclient` v9+ (pin major version)
2. `ServerOptions`: `{ command: cmPath, args: ["lsp"] }` with `TransportKind.stdio`
3. `documentSelector`: `[{ scheme: "file", language: "calcmark" }]`
4. Handle custom notification with type-safe interface:

```typescript
interface DocumentRenderedParams {
  readonly uri: string;
  readonly html: string;
}
client.onNotification('calcmark/documentRendered', (params: unknown) => {
  const rendered = validateDocumentRenderedParams(params);
  if (rendered) previewPanel.update(rendered);
});
```

5. Error handler: restart on crash with exponential backoff, max 3 retries, then degrade to "binary not found" state

**Preview panel (`previewPanel.ts`):**
1. Command: "CalcMark: Open Preview" → `vscode.window.createWebviewPanel`
2. Use `retainContextWhenHidden: true` (avoids flicker on tab switch) AND implement `WebviewPanelSerializer` for session restore
3. Content Security Policy: `default-src 'none'; style-src 'unsafe-inline'; img-src ${webview.cspSource}` (adjust based on template content)
4. Scroll sync: unidirectional editor-to-preview only (start simple). Read `data-source-line` attributes from rendered HTML, scroll webview to matching element on `onDidChangeTextEditorVisibleRanges`
5. No-op handler when preview panel is closed but notifications still arrive

**TextMate grammar (`calcmark.tmLanguage.json`):**
1. Follow TextMate scope naming hierarchy: `variable.other.calcmark`, `constant.numeric.integer.calcmark`, `constant.numeric.currency.calcmark`, `entity.name.function.calcmark`
2. Multi-token NL functions: `\\b(average\\s+of|square\\s+root\\s+of|sum\\s+of)\\b` → `entity.name.function.calcmark`
3. Currency literals: `\\$[0-9][0-9_,.]*` → `constant.numeric.currency.calcmark` (careful regex to avoid `$` anchor collision)
4. **Sync requirement:** Any new NL function alias added to `spec/features/registry.go` must also be added to the TextMate grammar. Document this in a CONTRIBUTING note.

**`package.json`:**
1. `activationEvents`: `["onLanguage:calcmark"]` (not `*`)
2. `extensionKind`: `["workspace"]` (required for Codespaces — extension runs server-side)
3. `engines.vscode`: `"^1.82.0"` (minimum for vscode-languageclient v9)
4. Register both `.cm` and `.calcmark` extensions
5. Configuration schema for `calcmark.binaryPath` setting

**Testing:**
6. Unit tests for `binaryDiscovery` (mock PATH lookup, settings API)
7. Unit tests for notification handler (validates params, handles missing panel)
8. Integration tests with `@vscode/test-electron`

**VS Code Web:** Works in Codespaces (remote backend with `cm` installed). Does NOT work in vscode.dev (no backend). State this explicitly in the extension README.

**Success criteria:**
- `cm lsp` starts and responds to LSP initialize
- Opening a `.cm` file in VS Code shows error/warning squiggles
- All 17 diagnostic codes correctly mapped
- Autocomplete shows variables, functions, units in calculation lines; suppressed in markdown lines
- Hover shows computed values and function signatures
- Preview panel updates on every keystroke
- NL multi-token completions work (`average of`, `sum of`)

### Phase 2: Polish (Driven by Feedback)

After Phase 1 ships and users provide feedback, add:

- **Semantic tokens** — context-aware highlighting that overrides TextMate for ambiguous lines (important for the markdown-vs-calculation classification problem). Token legend:

  | CalcMark concept | LSP token type | Modifier |
  |-----------------|----------------|----------|
  | Variable assignment LHS | `variable` | `declaration` |
  | Variable reference | `variable` | — |
  | Number/currency/fraction | `number` | — |
  | Unit name | `type` | — |
  | Function name / NL phrase | `function` | — |
  | Keyword (`in`, `as`, `of`) | `keyword` | — |
  | Operator | `operator` | — |
  | Markdown line text | `comment` | `documentation` |

- **Document formatting** — normalize block separators, trim trailing whitespace
- **Code actions** — "Did you mean?" quick fix for `undefined_variable`
- **Incremental document sync** (`TextDocumentSyncKind=2`) if latency feedback warrants it — the codebase already has `EvaluateAffectedBlocks` for block-level dependency tracking

### Phase 3: Ecosystem (Driven by Demand)

Gated on user requests from non-VS-Code editors:

- **Tree-sitter grammar** (`editors/tree-sitter-calcmark/`) — conservative grammar + LSP semantic tokens for context-dependent lines. Consider generating from `spec/SYNTAX_HIGHLIGHTER_SPEC.json` to maintain a single source of truth. External scanner in C for line-start classification.
- **Zed extension** (`editors/zed-calcmark/`) — TOML config, tree-sitter queries, `language_server_command()` returning `cm lsp`. No webview (Zed limitation as of 2026).
- **`cm watch` subcommand** — file-watching HTTP+WebSocket server for browser-based preview. Security requirements: loopback-only binding (`127.0.0.1`, never `0.0.0.0`), random session token in URL path, WebSocket origin validation (accept only `127.0.0.1`/`localhost`), CSP headers, `validateReadFilePath()`/`validateFileContent()` reuse.
- **Neovim/Emacs documentation** — 5-line and 4-line setup guides respectively

## System-Wide Impact

### Interaction Graph

`cm lsp` → `spec/document.NewDocument()` → `spec/lexer.Tokenize()` → `spec/parser.Parse()` → `spec/semantic.Checker.Check()` → `impl/document.Evaluator.Evaluate()` → `format.HTMLFormatter.Format()`. This is the same pipeline the CLI and TUI already use — the LSP is a new entry point, not a new pipeline.

### Research Insights: Cross-Layer Concerns

From the feature-skill analysis, the LSP touches **layers 3-8** of the CalcMark stack:
- Parser (AST with Range positions for hover/go-to-definition)
- Classifier (via Document block detection — determines which lines get calc-aware completions)
- Semantic spec (diagnostics, environment with variable definitions)
- Interpreter (computed values for hover, diagnostics)
- Feature Registry (completions data source — functions, units, aliases)

The two error paths in `parseFunctionCall()` (empty-args vs post-parse-args) produce different error shapes. The LSP diagnostic publisher must handle both.

### Error Propagation

LSP JSON-RPC errors propagate to the editor as error responses. Evaluation errors become `publishDiagnostics` notifications. The LSP server itself must never crash on malformed input — wrap evaluation in `recover()` as a safety net, and extend fuzz testing to cover LSP entry points.

### State Lifecycle Risks

- **Memory growth**: each open document holds an immutable snapshot. Mitigate with `didClose` cleanup and ensuring old snapshots are not retained by pending responses.
- **Stale state**: full sync mode eliminates divergence (server always receives the full document).
- **Concurrent access**: immutable snapshots solve this — multiple request handlers read the latest snapshot without locks.
- **Index drift**: Use Range-based (content-addressed) mapping between source positions and analysis results, never positional indices that assume 1:1 correspondence. (From learnings: `doceval-progressive-index-block-splitting-misalignment.md`)

### API Surface Parity

The LSP server exposes the same language intelligence as the TUI editor. Both consume `spec/features/SuggestionSource` (extracted from TUI) and `spec/document.Diagnostic`. Any new feature added to the TUI should also be reflected in the LSP.

### Integration Test Scenarios

1. Open a document with a `division_by_zero` error → diagnostic appears at correct line:column
2. Type `a = 1 + 1` → autocomplete offers `a` on subsequent lines with value `2`
3. Edit frontmatter `scale: 2` → all calculation results update, all diagnostics refresh
4. Hover over a NL function like `average of` → shows signature and description
5. Open a document with both NL and functional syntax → both produce correct completions

## Acceptance Criteria

### Functional Requirements (Phase 1)

- [ ] `cm lsp` subcommand starts an LSP server over stdio
- [ ] All 17 diagnostic codes from `spec/semantic/diagnostics.go` mapped to LSP diagnostics
- [ ] Autocomplete provides variables (position-aware), functions (NL + traditional), units, constants
- [ ] Autocomplete suppressed in markdown-classified lines
- [ ] NL multi-token completions work (`average of`, `sum of`, `square root of`)
- [ ] Hover shows computed values for variables, signatures for functions, descriptions for units
- [ ] Go-to-definition works for variable references → assignment line
- [ ] Document symbols show assignments and headings in outline view
- [ ] Live preview updates on every keystroke in VS Code (via `calcmark/documentRendered`)
- [ ] VS Code extension provides TextMate grammar, LSP client, preview webview
- [ ] Both `.cm` and `.calcmark` file extensions registered

### Non-Functional Requirements

- [ ] LSP diagnostics published within 200ms of keystroke (150ms debounce + 50ms eval)
- [ ] Preview HTML rendering <100ms P99
- [ ] LSP server memory stable over extended editing sessions (no leaks via snapshot retention)
- [ ] `cm lsp` startup time <500ms
- [ ] Per-evaluation timeout enforced (1 second via `context.WithTimeout`)
- [ ] Document size limit (1MB) enforced on `didOpen`/`didChange`
- [ ] Graceful degradation: if `cm` binary is missing, extension shows clear error message with install link

### Quality Gates

- [ ] Data-driven integration tests for all LSP handlers (request → response golden files)
- [ ] `TestEveryFeatureProducesValidCompletion` consistency test
- [ ] `TestAllFunctionCallsHaveRange` AST completeness gate
- [ ] VS Code extension unit tests for binary discovery and notification handling
- [ ] VS Code extension integration tests with `@vscode/test-electron`
- [ ] `task test` and `task quality` pass
- [ ] SECURITY.md updated with LSP-specific considerations (eval timeout, document size limit)
- [ ] Fuzz tests extended to cover LSP evaluation entry point

## Dependencies & Prerequisites

**Phase 1:**
- **GLSP library** (`github.com/tliron/glsp`) — Go LSP SDK
- **bluemonday** (already indirect dep via glamour) — HTML sanitization for preview
- **vscode-languageclient v9+** (npm) — for VS Code extension
- Prerequisite refactoring: extract `Suggestion`/`SuggestionSource` to `spec/features/`
- Cross-cutting change: add `data-source-line` attributes to `format/templates/default.html`

**Phase 3 (deferred):**
- **fsnotify** (already in go.mod) — for `cm watch` file watching
- **tree-sitter CLI** (`npm install tree-sitter-cli`) — for grammar development
- **zed_extension_api** (Rust crate) — for Zed extension

## Risk Analysis & Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| VS Code tree-sitter API ships, making TextMate grammar redundant | Low | Low | TextMate grammar is small. Migrate later if API ships |
| GLSP library becomes unmaintained | Low | High | Wraps stable LSP 3.17 protocol. Fork or switch to go.lsp.dev |
| Large documents cause LSP latency | Low | Medium | Single-pass eval + full sync handles 1000 lines in ~14ms. Incremental eval available if needed |
| NaN/Inf in YAML frontmatter crashes LSP | Medium | High | Already fixed in interpreter. Fuzz test LSP entry point as safety net |
| Binary PATH injection in VS Code extension | Low | Critical | Prefer user-configured path. Never search project-local directories. Verify checksum on auto-download |
| gomarkdown XSS bypass | Low | High | Add bluemonday sanitization pass before `template.HTML()` cast |
| Stale closures capturing value-type state | Medium | Medium | Use pointer indirection for shared state in Go server struct (from learnings) |

## Future Considerations

- **WASM-compiled interpreter** in VS Code webview for zero-latency preview
- **Incremental document sync** (`TextDocumentSyncKind=2`) leveraging `EvaluateAffectedBlocks`
- **Multi-file workspace support** if CalcMark adds imports
- **Debug adapter protocol** for step-through evaluation
- **Notebook integration** (VS Code notebooks) as an alternative to `.cm` files
- **Zed webview panel** when/if Zed ships the API
- **Adaptive debouncing** — scale interval with document size (150ms for <500 lines, 250ms for larger)
- **Copy-on-write environment** if `Environment.Clone()` becomes a bottleneck

## Documentation Plan

- User-facing: "Editor Setup" guide on the CalcMark website (VS Code first, others as they ship)
- Developer-facing: `lsp/` package doc comment explaining architectural position
- VS Code extension README for marketplace listing (including Codespaces note)
- CONTRIBUTING note about TextMate grammar ↔ features.Registry sync requirement

## Sources & References

### Origin

- **Brainstorm document:** [docs/brainstorms/2026-03-16-ide-extensions-and-lsp-brainstorm.md](docs/brainstorms/2026-03-16-ide-extensions-and-lsp-brainstorm.md) — Key decisions carried forward: LSP-first architecture, `cm lsp` subcommand, monorepo structure, every-keystroke preview, full language support at launch

### Internal References (Learnings Applied)

- Diagnostic pipeline pattern: `docs/solutions/code-organization/diagnostic-detailed-field-pipeline.md` — consume `Detailed` field directly, never re-parse error messages
- Document-absolute line numbers: `docs/solutions/code-organization/docline-diagnostic-line-numbers.md` — use `DocLine` for LSP positions
- AST Range completeness: `docs/solutions/logic-errors/nl-function-missing-ast-range.md` — all FunctionCall nodes must have valid Range
- Stale closure bug: `docs/solutions/logic-errors/go-closure-capturing-stale-value-type.md` — pointer indirection for shared mutable state
- NaN/Inf panic: `docs/solutions/security-issues/nan-inf-panic-yaml-frontmatter-scale.md` — fuzz test LSP entry points
- Module cohesion: `docs/solutions/code-organization/split-view-go-into-cohesive-modules.md` — one file per LSP capability
- Pure functional state: `docs/solutions/code-organization/pure-functional-layout-calculations.md` — parse once, serve cached results to all handlers
- Index drift: `docs/solutions/logic-errors/doceval-progressive-index-block-splitting-misalignment.md` — Range-based mapping, not positional indices
- Dual-syntax parity: `docs/solutions/integration-issues/nl-functional-syntax-parity-and-doc-staleness.md` — test both NL and functional syntax in LSP

### Internal References (Code)

- Suggestion interface: `cmd/calcmark/tui/components/suggest.go` (to be extracted to `spec/features/`)
- Semantic checker: `spec/semantic/diagnostics.go`
- Features registry: `spec/features/registry.go`
- HTML formatter: `format/html_formatter.go`
- HTML template: `format/templates/default.html`
- AST positions: `spec/ast/position.go`
- Evaluator: `impl/document/evaluator.go` (use `Evaluate()`, not `EvaluateBlock()`)
- Syntax highlighter spec: `spec/SYNTAX_HIGHLIGHTER_SPEC.json`
- Cobra subcommand pattern: `cmd/calcmark/cmd/eval.go`
- Security validation: `cmd/calcmark/cmd/security.go`

### External References

- [GLSP: Go LSP SDK](https://github.com/tliron/glsp)
- [mpls: Markdown Preview Language Server](https://github.com/mhersson/mpls)
- [LSP 3.17 Specification](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/)
- [VS Code Webview API](https://code.visualstudio.com/api/extension-guides/webview)
- [vscode-languageserver-node](https://github.com/microsoft/vscode-languageserver-node) — custom notifications, semantic tokens
- [tree-sitter Creating Parsers](https://tree-sitter.github.io/tree-sitter/creating-parsers/) (Phase 3)
- [Zed Language Extensions](https://zed.dev/docs/extensions/languages) (Phase 3)
- [Neovim 0.11 LSP config](https://gpanders.com/blog/whats-new-in-neovim-0-11/) (Phase 3)
- [Emacs eglot](https://www.gnu.org/software/emacs/manual/html_node/eglot/Setting-Up-LSP-Servers.html) (Phase 3)
