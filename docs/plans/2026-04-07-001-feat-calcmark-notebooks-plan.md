---
title: "feat: CalcMark Notebooks — Web App MVP"
type: feat
status: active
date: 2026-04-07
origin: docs/brainstorms/2026-04-07-calcmark-notebooks-requirements.md
---

# feat: CalcMark Notebooks — Web App MVP

## Overview

A web application for creating, editing, saving, and sharing CalcMark documents. Single Go binary with embedded Svelte frontend, SQLite storage (with Litestream for continuous backup), and social login (Google/GitHub OAuth). Jupyter-like block editor where blocks show rendered output by default and reveal source on click.

**This is a separate project/repository** (e.g., `CalcMark/calcmark-web`) that consumes `github.com/CalcMark/go-calcmark` as a Go library dependency. The `go-calcmark` repo stays clean: language spec, interpreter, formatters, CLI/TUI. The web product has its own binary, release cycle, and deployment model.

## Problem Frame

CalcMark is a powerful calculation language locked behind a CLI. The web product makes it accessible to anyone with a browser — create a document, see live results, share a readonly link. Files are always valid `.cm`, backups are a simple file copy. (see origin: `docs/brainstorms/2026-04-07-calcmark-notebooks-requirements.md`)

## Requirements Trace

- R1. Jupyter-like block UI (rendered by default, click to edit source)
- R2. Files are valid `.cm` — format is the product
- R3. Context-aware documentation sidebar
- R4. Beautiful markdown rendering
- R5. Single-user editing
- R6. Document = `.cm` source + metadata, source is truth
- R7. SQLite storage, backup = rsync/cp
- R8. Save persists source text
- R9. Social login (Google, GitHub OAuth)
- R10. Private by default
- R11. Share rendered output via readonly link
- R12. Clone: "open your own copy"
- R13. Clean URL patterns
- R14. Single Go binary deployment
- R15. API-first (HTTP between frontend and backend)
- R16. Server-side evaluation
- R17. Sub-100ms evaluation target
- Export: single doc raw download, bulk zip export

## Scope Boundaries

- No multiplayer editing
- No team/org visibility — private or public (shared link) only
- No PDF export
- No mobile-optimized editor
- No storage abstraction — SQLite directly (modernc.org/sqlite, pure Go)
- No Turso/vector search in MVP — add when semantic doc search is needed
- No custom CalcMark Lezer grammar in MVP — use CodeMirror's plain text mode with CalcMark evaluation for feedback. Custom grammar is a future enhancement.
- No git integration

## Context & Research

### Relevant Code and Patterns

- **Existing HTTP server**: `cmd/calcmark/cmd/watch.go` — stdlib `net/http`, WebSocket for live reload, security headers (CSP, X-Frame-Options, nosniff). Uses `calcmark.Convert()` directly.
- **Public API**: `calcmark.Convert(input, opts)` → rendered HTML string. `calcmark.Eval(input)` → Result with values + diagnostics. `calcmark.NewSession()` → stateful evaluation across calls.
- **Go embed pattern**: `format/html_formatter.go` embeds CSS/templates via `//go:embed`. Same pattern for frontend assets.
- **HTML templates**: `format/templates/default.gohtml` (standalone page), `format/templates/preview.gohtml` (fragment). `format.StyleCSS()` provides shared CalcMark CSS.
- **GoReleaser**: `.goreleaser.yaml` — single binary, CGO_ENABLED=0, cross-platform. Must remain CGO-free (hence modernc.org/sqlite over go-libsql).

### External References

- **SvelteKit + Go embed**: Build SvelteKit with `adapter-static`, embed `build/` directory via `//go:embed all:build`, serve with `http.FileServer`. Use `all:` prefix for underscore files.
- **OAuth**: `golang.org/x/oauth2` (minimal) or `github.com/go-pkgz/auth/v2` (batteries-included with JWT sessions, XSRF, multi-provider). Recommend go-pkgz/auth for faster MVP.
- **CodeMirror 6 + Svelte**: `svelte-codemirror-editor` package (v2.1.0, Svelte 5 compatible). Exclude from Vite optimizeDeps.
- **SQLite pure Go**: `modernc.org/sqlite` — no CGO, registers as `"sqlite"` with `database/sql`.

## Key Technical Decisions

- **modernc.org/sqlite over Turso for MVP**: Pure Go, no CGO, preserves CGO_ENABLED=0 cross-compilation. Turso (with vector extensions for doc search) is a future upgrade path — the database/sql interface makes swapping straightforward.
- **go-pkgz/auth for OAuth**: Handles JWT sessions, cookies, XSRF, multiple providers with `AddProvider()`. Avoids writing session management from scratch. MIT licensed.
- **SvelteKit with adapter-static**: Compiles to static HTML/JS/CSS, embedded in Go binary via `//go:embed`. Single binary contains everything. During dev, run SvelteKit dev server separately.
- **stdlib net/http (no router library)**: The existing watch server uses stdlib. The API surface is small enough (10-15 endpoints) that a router library adds dependency without value. Use `http.NewServeMux` with Go 1.22+ pattern matching.
- **CalcMark evaluation via Convert()**: The existing `calcmark.Convert(input, Options{Format: "html"})` produces full rendered HTML. The web backend calls this on every save/evaluation request. No new evaluation infrastructure needed.
- **Document ID generation**: Short random alphanumeric IDs (8 chars, like `a3kf92x1`). URL-safe, collision-resistant at small scale, human-shareable.

## Open Questions

### Resolved During Planning

- **SQLite or Turso?** SQLite (modernc.org/sqlite) for MVP. Pure Go, no CGO, simple backups. Turso later for vector search.
- **OAuth library?** go-pkgz/auth/v2 — handles sessions, JWT, XSRF, multi-provider out of the box.
- **Frontend framework?** SvelteKit with adapter-static, embedded in Go binary.
- **How to embed frontend?** `//go:embed all:build` in a `web/frontend/embed.go` file. `go generate` runs `npm run build`.
- **Licensing?** All dependencies MIT or permissive. Safe for commercial use.

### Deferred to Implementation

- Exact SQLite schema (migrations, indexes) — discover during Unit 2
- CodeMirror block-level editing UX details — prototype during Unit 6
- Documentation sidebar content source and search — future feature, not MVP
- WebSocket vs HTTP polling for live evaluation updates — start with HTTP, add WebSocket if latency is a problem

## High-Level Technical Design

> *This illustrates the intended approach and is directional guidance for review, not implementation specification. The implementing agent should treat it as context, not code to reproduce.*

```
Architecture:

  Browser (SvelteKit SPA)
    ↕ HTTP/JSON
  Go HTTP Server (net/http)
    ├── /auth/*          → go-pkgz/auth (OAuth, sessions)
    ├── /api/docs        → Document CRUD
    ├── /api/docs/:id/eval → CalcMark evaluation
    ├── /api/export      → Bulk zip download
    ├── /d/:id           → Rendered readonly view (server-rendered HTML)
    ├── /d/:id/raw       → Raw .cm source
    └── /*               → Embedded SvelteKit static files
    ↕
  SQLite (modernc.org/sqlite)
    └── documents table (id, owner_id, source, visibility, created_at, updated_at)
    └── users table (id, provider, provider_id, name, email, avatar_url, created_at)

Request Flow (evaluation):
  1. User edits block in CodeMirror → Svelte sends source text to /api/docs/:id/eval
  2. Go handler calls calcmark.Convert(source, Options{Format: "html"})
  3. Returns {html: "...", diagnostics: [...], values: [...]}
  4. Svelte renders HTML in the block, shows diagnostics inline

Request Flow (sharing):
  1. Owner clicks "Share" → POST /api/docs/:id/share → sets visibility=public
  2. Returns shareable URL: /d/:id
  3. Viewer hits /d/:id → server calls calcmark.Convert(), returns full HTML page
  4. "Open your own copy" button → POST /api/docs/:id/clone (requires auth)
```

## Implementation Units

### Phase 1: Backend Foundation

- [ ] **Unit 1: Go HTTP Server Scaffold + SQLite Schema**

**Goal:** Minimal Go HTTP server that serves a "Hello CalcMark" page, connects to SQLite, and runs migrations.

**Requirements:** R7, R14, R15

**Dependencies:** None — foundation for everything.

**Files:**
- Create: `web/server.go` (HTTP server, mux, middleware)
- Create: `web/db.go` (SQLite connection, migrations)
- Create: `web/db_test.go`
- Create: `web/server_test.go`

**Approach:**
- `http.NewServeMux` with Go 1.22+ pattern matching (e.g., `GET /api/docs/{id}`).
- SQLite via `modernc.org/sqlite` with `database/sql`. Schema: `users` (id, provider, provider_id, name, email, avatar_url, created_at) and `documents` (id TEXT PRIMARY KEY, owner_id, source TEXT, visibility TEXT DEFAULT 'private', title TEXT, created_at, updated_at).
- Document IDs are 8-char random alphanumeric strings generated at creation time.
- Migrations: simple version-numbered SQL files embedded via `//go:embed`. Run on startup.
- Security middleware: CSP headers, X-Frame-Options, nosniff (follow `watch.go` pattern).

**Patterns to follow:**
- `cmd/calcmark/cmd/watch.go` for HTTP server structure, security headers
- `format/html_formatter.go` for `//go:embed` pattern

**Test scenarios:**
- Happy path: Server starts, connects to SQLite, runs migrations, responds to health check
- Happy path: Schema creates users and documents tables with correct columns
- Edge case: Server starts with empty/new database — migrations run cleanly
- Error path: Invalid database path — server fails with clear error message
- Integration: Migrations are idempotent — running twice produces no errors

**Verification:**
- `go test ./web/...` passes
- Server starts on a port and responds to HTTP requests

---

- [ ] **Unit 2: Document CRUD API**

**Goal:** REST API for creating, reading, updating, listing, and deleting documents.

**Requirements:** R6, R8, R13, R15

**Dependencies:** Unit 1 (server + database)

**Files:**
- Create: `web/handlers_docs.go`
- Create: `web/handlers_docs_test.go`

**Approach:**
- `POST /api/docs` — create new document (generates ID, stores source text, returns doc metadata)
- `GET /api/docs` — list user's documents (requires auth, returns metadata array)
- `GET /api/docs/{id}` — get single document (source + metadata)
- `PUT /api/docs/{id}` — update source text (owner only)
- `DELETE /api/docs/{id}` — delete document (owner only)
- `GET /api/docs/{id}/raw` — raw `.cm` source text (plain text content type)
- All endpoints return JSON except `/raw` which returns `text/plain`.
- Owner validation on write operations. Public documents readable by anyone.

**Patterns to follow:**
- Standard Go HTTP handler patterns with `http.NewServeMux`

**Test scenarios:**
- Happy path: Create document → returns ID and metadata
- Happy path: List documents → returns only user's documents
- Happy path: Get document by ID → returns source and metadata
- Happy path: Update document → source text changes, updated_at advances
- Happy path: Get raw → returns plain .cm text
- Edge case: Create document with empty source → succeeds (blank document)
- Error path: Get nonexistent document → 404
- Error path: Update document owned by another user → 403
- Error path: Delete document owned by another user → 403
- Integration: Create → Get → Update → Get → Delete → Get returns 404

**Verification:**
- `go test ./web/...` passes
- Full CRUD lifecycle works via HTTP client tests

---

- [ ] **Unit 3: OAuth Authentication**

**Goal:** Social login with Google and GitHub. Session management via JWT cookies.

**Requirements:** R9, R10

**Dependencies:** Unit 1 (server + database)

**Files:**
- Create: `web/auth.go` (provider config, user upsert)
- Create: `web/auth_test.go`
- Modify: `web/server.go` (mount auth routes, add auth middleware)

**Approach:**
- Use `github.com/go-pkgz/auth/v2` for OAuth flow, JWT sessions, XSRF protection.
- `service.AddProvider("github", clientID, clientSecret)` + `service.AddProvider("google", clientID, clientSecret)`.
- On successful auth, upsert user in SQLite (provider + provider_id as unique key). Store name, email, avatar_url.
- Auth middleware on `/api/*` routes. Readonly routes (`/d/{id}` rendered view) do not require auth.
- Configuration via environment variables: `CALCMARK_GITHUB_CLIENT_ID`, `CALCMARK_GITHUB_CLIENT_SECRET`, `CALCMARK_GOOGLE_CLIENT_ID`, `CALCMARK_GOOGLE_CLIENT_SECRET`, `CALCMARK_JWT_SECRET`.

**Patterns to follow:**
- go-pkgz/auth documentation for handler mounting and middleware usage

**Test scenarios:**
- Happy path: OAuth callback creates new user in database
- Happy path: Returning user — OAuth callback finds existing user, updates name/avatar
- Happy path: Authenticated request to /api/docs succeeds with user context
- Error path: Unauthenticated request to /api/docs → 401
- Error path: Missing OAuth env vars → server fails to start with clear error
- Edge case: User logs in with GitHub, then later with Google (same email) — treated as separate accounts (provider + provider_id is the key)

**Verification:**
- `go test ./web/...` passes
- OAuth flow works end-to-end in a browser (manual verification)

---

### Phase 2: Evaluation + Rendering

- [ ] **Unit 4: CalcMark Evaluation API**

**Goal:** API endpoint that accepts `.cm` source text, evaluates it, and returns rendered HTML + diagnostics.

**Requirements:** R16, R17

**Dependencies:** Unit 2 (document CRUD for context)

**Files:**
- Create: `web/handlers_eval.go`
- Create: `web/handlers_eval_test.go`

**Approach:**
- `POST /api/docs/{id}/eval` — accepts `{source: "..."}`, calls `calcmark.Convert(source, Options{Format: "html"})`, returns `{html: "...", diagnostics: [...]}`.
- Also supports `POST /api/eval` (anonymous, no document context) for quick evaluation without saving.
- Diagnostics extracted from the evaluation error (CalcMark returns partial results + errors).
- Response includes both the full rendered HTML and structured diagnostics (line, message, severity) for inline editor display.

**Patterns to follow:**
- `calcmark.Convert()` in `convert.go` — the existing public API
- `cmd/calcmark/cmd/watch.go` — calls Convert() on file change

**Test scenarios:**
- Happy path: Valid .cm source → returns rendered HTML with computed values
- Happy path: Source with diagnostics → returns HTML + diagnostics array
- Happy path: Evaluation completes within 100ms for a typical document (10 blocks)
- Edge case: Empty source → returns empty HTML, no diagnostics
- Error path: Completely invalid source → returns partial HTML + error diagnostics
- Integration: Update document source via PUT, then eval → results reflect updated source

**Verification:**
- `go test ./web/...` passes
- A consulting SOW `.cm` document evaluates and renders correctly via the API

---

- [ ] **Unit 5: Rendered Readonly View (Sharing)**

**Goal:** Server-rendered HTML page for shared documents. Beautiful output at `GET /d/{id}`.

**Requirements:** R4, R11, R13

**Dependencies:** Unit 2 (document CRUD), Unit 4 (evaluation)

**Files:**
- Create: `web/handlers_share.go`
- Create: `web/handlers_share_test.go`
- Create: `web/templates/view.gohtml` (rendered document page template)

**Approach:**
- `GET /d/{id}` — fetch document, evaluate with `calcmark.Convert()`, render into a full HTML page using a Go template.
- The page template includes CalcMark's CSS (`format.StyleCSS()`), the rendered HTML content, a header with document title, and a "Open your own copy" button (links to `/{id}/clone`).
- Public documents: no auth required. Private documents: 404 (don't leak existence).
- `POST /api/docs/{id}/share` — toggle visibility to "public", returns shareable URL.
- `POST /api/docs/{id}/clone` — requires auth. Creates a new private document with the same source text, owned by the authenticated user.

**Patterns to follow:**
- `format/templates/default.gohtml` for HTML page structure
- `cmd/calcmark/cmd/watch.go` for injecting StyleCSS into page templates

**Test scenarios:**
- Happy path: Public document at /d/{id} → returns beautiful rendered HTML page
- Happy path: Clone → creates new document with same source, different ID, owned by cloner
- Happy path: Share toggle → document becomes public, URL is returned
- Error path: Private document at /d/{id} → 404
- Error path: Nonexistent document → 404
- Error path: Clone without auth → 401
- Edge case: Clone a document with evaluation errors → clone succeeds (source is copied, not evaluated)

**Verification:**
- `go test ./web/...` passes
- A shared document URL renders a beautiful standalone page in a browser

---

### Phase 3: Frontend (Svelte)

- [ ] **Unit 6: SvelteKit Scaffold + Go Embed**

**Goal:** SvelteKit project with adapter-static, embedded in the Go binary, served as the SPA.

**Requirements:** R14

**Dependencies:** Unit 1 (HTTP server to serve the files)

**Files:**
- Create: `web/frontend/` (SvelteKit project: package.json, svelte.config.js, vite.config.js, src/)
- Create: `web/frontend/embed.go` (`//go:embed all:build`)
- Modify: `web/server.go` (serve embedded SvelteKit files as fallback)

**Approach:**
- `npx sv create web/frontend` (SvelteKit skeleton with TypeScript).
- Configure `adapter-static` with `fallback: 'index.html'` for SPA routing.
- `embed.go` with `//go:generate npm --prefix . run build` and `//go:embed all:build`.
- Go server: API routes first (`/api/*`, `/auth/*`, `/d/*`), then fallback to SvelteKit static files for everything else.
- Development mode: `CALCMARK_DEV=1` env var proxies frontend requests to SvelteKit dev server (port 5173) instead of serving embedded files.
- Vite config: exclude `svelte-codemirror-editor` and `@codemirror/*` from optimizeDeps.

**Patterns to follow:**
- `format/html_formatter.go` for `//go:embed` pattern
- SvelteKit + Go embed pattern from external research

**Test scenarios:**
- Happy path: `go generate ./web/frontend/...` + `go build` produces a binary that serves the SvelteKit app
- Happy path: SPA routing works — `/new`, `/{id}/edit` all resolve to index.html with client-side routing
- Edge case: API routes take priority over SPA fallback — `/api/docs` hits the API, not the SPA
- Integration: Binary starts, serves both API and frontend from a single port

**Verification:**
- Binary serves the SvelteKit app at `/`
- Navigation between pages works without full page reloads

---

- [ ] **Unit 7: Document List + Management Page**

**Goal:** Svelte page showing the user's documents with create/delete/share actions.

**Requirements:** R13

**Dependencies:** Unit 2 (CRUD API), Unit 3 (auth), Unit 6 (SvelteKit scaffold)

**Files:**
- Create: `web/frontend/src/routes/+page.svelte` (document list — home page)
- Create: `web/frontend/src/routes/+layout.svelte` (app shell, nav, auth state)
- Create: `web/frontend/src/lib/api.ts` (API client helper)

**Approach:**
- Home page (`/`) shows the authenticated user's documents as cards (title, last updated, visibility badge).
- "New Document" button → `POST /api/docs` → redirect to `/{id}/edit`.
- Delete, Share toggle actions on each card.
- Auth state in layout: if not logged in, show login buttons (Google, GitHub). If logged in, show user avatar + name.
- `api.ts` helper: wraps `fetch()` with auth headers, base URL handling, JSON parsing.

**Patterns to follow:**
- Standard SvelteKit page/layout patterns

**Test scenarios:**
- Happy path: Authenticated user sees their documents listed
- Happy path: "New Document" creates a doc and navigates to editor
- Happy path: Delete removes document from list
- Happy path: Share toggle changes visibility badge
- Edge case: User with zero documents sees empty state with "Create your first document" prompt
- Error path: API error → user-friendly error message displayed

**Verification:**
- User can create, see, and manage documents from the browser

---

- [ ] **Unit 8: Block Editor (CodeMirror 6)**

**Goal:** Jupyter-like block editing experience. Blocks show rendered HTML by default; click to reveal CodeMirror source editor.

**Requirements:** R1, R2, R4, R5, R16

**Dependencies:** Unit 4 (evaluation API), Unit 6 (SvelteKit), Unit 7 (navigation)

**Files:**
- Create: `web/frontend/src/routes/[id]/edit/+page.svelte` (editor page)
- Create: `web/frontend/src/lib/components/Block.svelte` (single block: rendered/editing states)
- Create: `web/frontend/src/lib/components/Editor.svelte` (document editor orchestrator)
- Create: `web/frontend/src/lib/components/FrontmatterPanel.svelte` (frontmatter click-to-edit)

**Approach:**
- Parse `.cm` source into blocks (split on `---` separators, same as CalcMark's document model).
- Each block is a `<Block>` component with two states: **rendered** (shows HTML from evaluation) and **editing** (shows CodeMirror with source text).
- Click a rendered block → switch to editing. Click outside or press Escape → save block, re-evaluate, switch to rendered.
- On block save: reassemble full `.cm` source from all blocks, send to `/api/docs/{id}/eval`, update all rendered blocks with new HTML.
- CodeMirror instance per block using `svelte-codemirror-editor`. Plain text mode for MVP (no custom Lezer grammar).
- Debounce evaluation: 300ms after last keystroke (same pattern as LSP debounce).
- Frontmatter panel: click the document header area to reveal/edit YAML frontmatter.
- Auto-save: `PUT /api/docs/{id}` on evaluation success (debounced, not on every keystroke).

**Patterns to follow:**
- Jupyter Notebook's cell editing model
- `svelte-codemirror-editor` component API

**Test scenarios:**
- Happy path: Load document → all blocks render as beautiful HTML
- Happy path: Click block → CodeMirror appears with source text
- Happy path: Edit block, click away → block re-evaluates, shows updated rendered output
- Happy path: Add new block (type `---` at end) → new empty block appears
- Happy path: Frontmatter editable via click-to-reveal panel
- Edge case: Document with single block (no separators) → one editable block
- Edge case: Document with only TextBlocks (no calculations) → renders as beautiful markdown
- Error path: Evaluation returns diagnostics → shown inline in the block
- Integration: Edit → auto-save → refresh page → edits persisted

**Verification:**
- User can open a document, click blocks to edit, see live evaluation results, and save

---

### Phase 4: Export + Polish

- [ ] **Unit 9: Export APIs**

**Goal:** Download individual documents as `.cm` files and bulk export all documents as a zip.

**Requirements:** Export requirement from brainstorm

**Dependencies:** Unit 2 (CRUD API), Unit 3 (auth)

**Files:**
- Create: `web/handlers_export.go`
- Create: `web/handlers_export_test.go`

**Approach:**
- `GET /api/docs/{id}/raw` already exists from Unit 2 (returns `text/plain` `.cm` source).
- `GET /api/export` — authenticated. Queries all user's documents, creates a zip archive in memory, streams it as `application/zip` with `Content-Disposition: attachment; filename="calcmark-export.zip"`. Each document is `{title_or_id}.cm` in the zip.
- `GET /api/docs/{id}/export` — single document download with `Content-Disposition` header for browser download prompt.

**Patterns to follow:**
- Go `archive/zip` for in-memory zip creation

**Test scenarios:**
- Happy path: Export single document → downloads as `.cm` file
- Happy path: Export all → downloads zip containing all user's documents
- Edge case: User with zero documents → empty zip (or 204 No Content)
- Edge case: Two documents with same title → filenames deduplicated (append ID)
- Error path: Unauthenticated bulk export → 401

**Verification:**
- Downloaded `.cm` files are valid and work in `cm eval`
- Zip contains all expected documents

---

- [ ] **Unit 10: Binary Entry Point + Deployment**

**Goal:** Main binary entry point for the web product. Dockerfile with Litestream for continuous backup.

**Requirements:** R14, R7, R8

**Dependencies:** All previous units

**Files:**
- Create: `main.go` (entry point — starts HTTP server)
- Create: `Dockerfile` (multi-stage: build frontend + Go binary, runtime with Litestream)
- Create: `docker-compose.yml` (binary + Caddy + Litestream example)
- Create: `litestream.yml` (Litestream config for S3/GCS WAL streaming)
- Create: `README.md` (deployment guide)

**Approach:**
- Binary is the web product's own binary (separate repo from go-calcmark), not a `cm` subcommand.
- Flags: `--port` (default 8080), `--data-dir` (default `./data/`), `--dev` (proxy to SvelteKit dev server).
- SQLite database at `{data-dir}/calcmark.db`.
- OAuth config from environment variables.
- Graceful shutdown on SIGINT/SIGTERM.
- Startup banner: prints URL, data directory, auth status.
- No OAuth env vars → public-only mode (no auth, all docs public).
- **Continuous backup**: Litestream runs alongside the binary (in Docker, as a wrapper process). Streams SQLite WAL changes to S3/GCS/any S3-compatible bucket. Restore = `litestream restore`. Backup is continuous and automatic.
- **Simple backup alternative**: `cp data/calcmark.db backup/` or `rsync` — SQLite is a single file.

**Patterns to follow:**
- Litestream deployment patterns (litestream.io)
- `cmd/calcmark/cmd/watch.go` for CLI flag patterns

**Test scenarios:**
- Happy path: Binary starts, prints banner with URL and data directory
- Happy path: `--port 8080` binds to custom port
- Happy path: No OAuth vars → starts in public-only mode with warning
- Edge case: Data directory doesn't exist → created automatically
- Error path: Port already in use → clear error message
- Integration: Docker build produces a single image with embedded frontend

**Verification:**
- Binary starts the full web application
- Litestream config streams changes to configured bucket
- `cp` or `rsync` of the SQLite file produces a valid backup

## System-Wide Impact

- **Interaction graph:** The web server calls `calcmark.Convert()` and `calcmark.Eval()` — the existing public API. No changes to the CalcMark engine. The web layer is purely additive.
- **Error propagation:** CalcMark evaluation errors become JSON diagnostic arrays in the API response. The Svelte frontend renders them inline. No new error types needed.
- **State lifecycle risks:** SQLite WAL mode for concurrent reads. Only one writer at a time (single-user editing). Auto-save debouncing prevents rapid writes.
- **API surface parity:** The web app uses the same `calcmark.Convert()` that the CLI uses. Documents created in the web app work identically in `cm eval` and `cm edit`.
- **Binary size:** Adding the web frontend (SvelteKit static build + SQLite driver) will increase the binary. Estimate: +5-10MB. Acceptable for a single-binary deployment.
- **Unchanged invariants:** All existing CLI commands, TUI, LSP, and the CalcMark engine are completely unaffected. The web product is a separate repository and binary — it consumes `go-calcmark` as a library dependency. Zero changes to `go-calcmark` are required.
- **Backup strategy:** Litestream for continuous WAL streaming (S3/GCS). Simple alternative: `cp`/`rsync` of the SQLite file. Both are documented deployment patterns.

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| SvelteKit + Go embed build complexity | `go generate` automates npm build. CI/CD runs `go generate` before `go build`. Dev mode proxies to SvelteKit dev server. |
| Binary size increase from embedded frontend | SvelteKit adapter-static produces small bundles. Monitor size. If too large, consider gzip compression of embedded assets. |
| CodeMirror block editing UX is complex | Start with a simple two-state component (rendered/editing). Iterate on UX during implementation. The block model is well-understood from Jupyter. |
| modernc.org/sqlite performance for concurrent reads | SQLite WAL mode handles concurrent reads well. Single-user editing means minimal write contention. Benchmark during implementation. |
| OAuth requires HTTPS in production | Caddy handles TLS automatically. Dev mode uses HTTP on localhost (OAuth providers allow localhost callbacks). |
| go-pkgz/auth may not fit perfectly | Library is well-maintained and MIT licensed. If it doesn't fit, fall back to `golang.org/x/oauth2` with manual session management. |

## Documentation / Operational Notes

- Add `cm serve` to CLI help and documentation
- Document deployment: single binary + Caddy + Litestream pattern
- Document environment variables for OAuth configuration
- Add a "Getting Started" page for the web product on calcmark.org

## Sources & References

- **Origin document:** [docs/brainstorms/2026-04-07-calcmark-notebooks-requirements.md](docs/brainstorms/2026-04-07-calcmark-notebooks-requirements.md)
- **Existing watch server:** `cmd/calcmark/cmd/watch.go`
- **CalcMark public API:** `convert.go`, `eval.go`, `session.go`
- **GoReleaser config:** `.goreleaser.yaml`
- **SQLite pure Go:** `modernc.org/sqlite`
- **OAuth library:** `github.com/go-pkgz/auth/v2`
- **SvelteKit adapter-static:** `@sveltejs/adapter-static`
- **CodeMirror Svelte wrapper:** `svelte-codemirror-editor`
- Related issue: CalcMark/go-calcmark#118
