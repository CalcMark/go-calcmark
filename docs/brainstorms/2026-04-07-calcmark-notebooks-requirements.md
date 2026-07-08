---
date: 2026-04-07
topic: calcmark-notebooks
---

# CalcMark Notebooks

## Problem Frame

CalcMark is a powerful calculation language with rich diagnostics, autosuggest, and documentation — but it's locked behind a CLI. The language can be broad and complex because the editor experience (LSP, inline help, contextual docs) makes it approachable. What's missing is a way to create, edit, and share CalcMark documents without installing anything.

The unique wedge: **plain-text source of truth + single-binary self-hosting + calculations that read like documents**. Spreadsheets are opaque. Notebooks (Jupyter) store JSON blobs. Docs tools (Notion) can't compute. CalcMark documents are readable text files where prose and calculations coexist — and the web product makes that accessible to anyone with a browser.

## Requirements

**Editor Experience**

- R1. Jupyter-like block UI: blocks show beautifully rendered output by default. Click a block to reveal and edit the CalcMark source. Frontmatter editable via click-to-reveal.
- R2. The editor must produce and consume valid `.cm` files. The file format is the product — no proprietary models. Everything is derived from the source text.
- R3. Context-aware documentation sidebar: calcmark.org docs appear alongside the editor, contextual to what the user is typing. The same diagnostics, autosuggest, and inline help from the LSP are available in the web editor.
- R4. Beautiful markdown rendering — tables, headings, prose, interpolated values all render with high visual quality.
- R5. Single-user editing for MVP.

**Semantic Editor Intelligence (the killer feature)**

- R18. **Interpolation autosuggest**: typing `{{ ` in any markdown block triggers a dropdown of all defined CalcMark variables from anywhere in the document. Suggestions include variable name, current value, and a natural language description (e.g., "sum of hc column in people table" or "total project cost: $1.2M").
- R19. **Calc-table preview**: when a user clicks a CalcMark block that references a named table, the table renders inline as a mini-spreadsheet showing computed column values overlaid on the source data. The user sees inputs and outputs together without switching context.
- R20. **Visual table editor**: markdown tables can be edited visually — drag to reorder columns, click cells to edit values, add/remove rows. The editor produces valid markdown pipe table syntax. When a table has a CalcMark directive, editing a cell triggers re-evaluation of dependent calculations.
- R21. **Fluid prose-calculation boundary**: the editor understands the document semantically. Editing flows naturally between prose, calculations, and data without hard modal switches. CalcMark diagnostics, autosuggest, and documentation appear contextually wherever the cursor is — in a calc block, in an interpolation tag, or in a table cell.

**Storage and State**

- R6. Each document is stored as its `.cm` source text plus metadata (owner, timestamps). The source text is the single source of truth.
- R7. SQLite as the storage backend. One file, simple backups (Litestream to S3/GCS, or plain rsync/cp of the database file). No storage abstraction layer in MVP — clean Go package boundary is sufficient to enable future backend changes if needed.
- R8. "Save" persists the current source text. Backup must be as simple as rsync or streaming the SQLite WAL.

**Authentication and Sharing**

- R9. Social login only (Google, GitHub OAuth). No custom user/password system.
- R10. Documents are private by default (owner only).
- R11. Share rendered output: a readonly link renders the document beautifully with all calculations resolved. No editing, no account required to view.
- R12. "Open your own copy": any viewer of a shared document can clone it into their own authenticated workspace as a new private document.

**URL Patterns**

- R13. Clean, predictable URLs:
  - `/{id}` — rendered view (readonly, shareable)
  - `/{id}/edit` — editor (authenticated owner only)
  - `/new` — blank document (authenticated)
  - `/{id}/clone` — fork into own copy (authenticated)
  - `/{id}/raw` — plain `.cm` source text

**Deployment**

- R14. Single Go binary (`cmw`) that serves the web UI, evaluates CalcMark, and stores documents in SQLite. Zero external infrastructure required. Deploy to a VPS, Docker, or fly.io. Works identically locally: `cmw start --port 3141` and start working.
- R15. API-first: the web UI communicates with the backend via HTTP. Clean package separation in Go enables future extraction if needed, but no formal API versioning in MVP.
- R15a. `cmw backup` command: copies the SQLite database to a destination (local path, S3, etc.). For continuous backup in production, Litestream streams WAL changes. For local use, `cmw backup ./my-backup/` is enough.
- R15b. Separate project and binary (`CalcMark/calcmark-web`). Consumes `go-calcmark` as a library. Own release cycle, own deployment. Named tables is NOT a prerequisite — the web product ships with current CalcMark capabilities and gains table support when it lands in the library.

**Evaluation**

- R16. Documents are evaluated server-side using the existing CalcMark Go library.
- R17. Evaluation must be fast enough for interactive editing. Target: sub-100ms for typical documents (same as TUI performance).

## Success Criteria

- A user can sign in with Google/GitHub, create a CalcMark document in their browser, see live computed results, and share a readonly link — without installing anything.
- The shared link renders the document beautifully. A viewer can "open their own copy" to fork it.
- A self-hosted instance runs as a single binary, stores data in SQLite, and requires zero external services beyond an OAuth provider.
- Every document is a valid `.cm` file that works identically in the CLI.
- Backups are as simple as rsync, cp, or Litestream streaming to a bucket.

## Scope Boundaries

- **No multiplayer editing** — single-user editing only.
- **No team/org visibility** — documents are private (owner) or public (shared link). Team features come later if there's demand.
- **No PDF export** — HTML rendering is the primary output.
- **No mobile-optimized editor** — responsive readonly view is fine, desktop editor.
- **No storage abstraction** — SQLite directly. Refactor when a second backend is concretely needed.
- **No custom auth** — social login only (OAuth).
- **No git integration in MVP** — future consideration.

## Key Decisions

- **Consumer-oriented, not enterprise**: Social login, private by default, share via link. No team management, no RBAC, no SSO. The simplest model that works for individual knowledge workers and consultants.
- **`.cm` file as source of truth**: The web app is a lens on plain text. CLI compatibility, git-friendliness, and portability preserved. Tradeoff: the editor must parse and produce valid `.cm`.
- **Server-side evaluation**: CalcMark runs on the server (Go), not in the browser. Simplifies the frontend. Tradeoff: keystrokes need a round-trip (mitigated by debouncing). WASM is a future option if latency becomes an issue.
- **Context-aware documentation with a flywheel**: The editor surfaces CalcMark docs, diagnostics, and autosuggest inline — the same quality of help the LSP provides in IDEs. This is what makes a complex language approachable for non-developers. The docs improve continuously: gaps found in the editor → new pages added to calcmark.org with Hugo frontmatter tags → the sidebar search gets smarter. The product and documentation improve together. The editor can also suggest example documents to explore.
- **SQLite + Litestream**: One database file. Backup = stream WAL changes to S3 or just rsync the file. No Postgres, no managed DB, no ops complexity. Turso (SQLite-compatible with vector extensions) is a potential future backend for semantic doc search.
- **Jupyter-like block UI**: Click to edit source, otherwise see rendered output. Side-by-side alignment is nearly impossible in web. The block pattern is more natural for the target audience.

## Dependencies / Assumptions

- The CalcMark Go library is the evaluation backend. No engine changes needed.
- The web frontend tech stack is deferred to planning.
- The named tables feature should land before the web product (tabular data is a primary notebook use case).
- OAuth providers (Google, GitHub) are the only auth dependency.

## Outstanding Questions

### Deferred to Planning

- [Affects R1][Needs research] What web editor framework best supports the Jupyter-like block editing pattern with CalcMark syntax highlighting? (CodeMirror 6, Monaco, ProseMirror, custom)
- [Affects R3][Needs research] How to surface contextual CalcMark docs in the sidebar — embed the Hugo site content, use a search index, or build a semantic search with vector embeddings (Turso)?
- [Affects R14][Technical] How to embed the web frontend assets in the single Go binary (Go embed, or similar). The product must be a single binary deployment — no separate frontend build artifact.
- [Affects R16][Technical] Evaluation API: WebSocket for streaming updates vs HTTP request/response with debouncing.
- [Affects R9][Technical] OAuth implementation — use an existing Go library (e.g., goth, oauth2) or minimal custom implementation?

## Future Considerations

**Multiplayer editing**: Real-time collaboration via CRDTs (Yjs) over block source text. The Jupyter-like block model helps (blocks can be independently locked). MVP's single-user model should not architecturally block this.

**Semantic doc search**: Turso's vector extension could power "search CalcMark docs by meaning" in the sidebar. User types a question, gets the most relevant documentation section. Requires embedding calcmark.org content into a vector index.

**Git integration**: "Save" = "git commit" for users who want version control. Local git repo on the server or GitHub API integration.

**PDF export**: Server-side (headless Chrome) or separate service. Keep the core binary lean.

**Hugo/static site integration**: Hugo preprocessor for `.cm` files or embedded `cm` code blocks. Separate from notebooks but shares the library.

**GitHub code block rendering**: If GitHub supports custom code block renderers, CalcMark's engine + formatters would be the integration surface. Already embeddable via `calcmark.Eval()`.

**Template library**: Pre-built `.cm` templates (consulting SOW, project budget, capacity planning). Users clone templates to get started.

**Computed table rows**: Auto-inject summary rows into rendered output without special markup (from the named-tables brainstorm).

## Next Steps

-> `/ce:plan` for structured implementation planning
