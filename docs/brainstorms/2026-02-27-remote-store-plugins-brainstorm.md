# Remote Store Plugins for CalcMark

**Date:** 2026-02-27
**Status:** Brainstorm complete

## What We're Building

A plugin system for CalcMark's TUI editor that supports **"Share To"** and
**"Open From"** operations against remote data stores. V1 ships with GitHub
Gists as the only backend; the architecture accommodates future backends
without over-engineering for them now.

### Core Model

Two one-way operations with no sync state:

- **Share To** — push current document content to a remote store, receive a
  shareable URL back. V1 creates a new remote resource each time. Overwrite
  semantics (updating an existing remote resource) are a future concern owned
  by individual plugins, not by CalcMark core.
- **Open From** — user provides a URL or identifier, CalcMark fetches the
  content and opens it as a **new local document**. Ctrl+S saves locally.
  There is no persistent link back to the remote.

CalcMark itself holds zero remote tracking state. No remote IDs in document
metadata, no sync bookkeeping, no local-remote associations.

## Why This Approach

1. **Subprocess / exec-based plugins** — each plugin delegates to an external
   CLI tool (e.g., `gh` for GitHub Gists). This means:
   - Auth is handled entirely by the external tool (`gh auth`).
   - CalcMark never touches credentials or tokens.
   - Each plugin inherits the security model of its backing CLI.
   - Plugins are language-agnostic (any executable works).

2. **One-way operations, no sync** — keeps CalcMark's responsibility minimal.
   The editor is a document editor, not a sync client. Remote stores are a
   sharing mechanism, not a persistence layer.

3. **Hardcoded V1 UI, generalize later** — the gist-specific prompts
   (public/secret, description) are built directly into the TUI for V1. When a
   second plugin is added, a common manifest/schema pattern can be extracted.
   YAGNI until then.

## Key Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Plugin execution model | Subprocess / exec-based | Delegates auth, leverages mature CLIs, language-agnostic |
| Sync state | None | CalcMark is an editor, not a sync client |
| V1 backend | GitHub Gists via `gh` CLI | Widely used, excellent CLI, free API |
| "Share To" semantics | One-way push → URL | V1 is create-new only; overwrite is a future plugin concern |
| "Open From" semantics | One-way pull → new local doc | No persistent remote link; Ctrl+S saves locally |
| Plugin UI for V1 | Hardcoded gist options | Generalize to manifest/schema when second plugin arrives |
| Open From input | Paste URL or gist ID | No browse/list functionality in V1 |
| Surface area | TUI editor only | No CLI subcommands for share/open |
| TUI prompts library | Charm's Huh library | Terminal forms (select, input, confirm) without custom widgets |

## V1: GitHub Gists Flow

### Share To Gist

1. User triggers "Share To" in TUI (keybinding or command)
2. CalcMark shows Huh form:
   - **Visibility**: Public / Secret (select)
   - **Description**: optional text input
3. CalcMark pipes document content to:
   ```
   gh gist create --public -f <filename>.cm
   ```
   (or without `--public` for secret gists)
4. `gh` handles auth (prompts user if not authenticated)
5. CalcMark displays the returned gist URL to the user

### Open From Gist

1. User triggers "Open From" in TUI
2. CalcMark shows Huh text input: "Gist URL or ID"
3. CalcMark calls:
   ```
   gh gist view <id> -r
   ```
4. Content is loaded into a new untitled local document
5. Ctrl+S prompts for local save location

## Future Backends (Documented, Not Planned)

Research identified these as viable future plugins, roughly ordered by fit:

### Paste Services (e.g., Pastes.io, dpaste)

- **Model**: Simple REST — POST to create, GET to read. URL-based sharing.
- **Fit**: Lowest friction for "share this calculation with someone." No
  account required for public pastes.
- **Auth**: API key or anonymous for public pastes.
- **CLI**: Would need a thin wrapper script or custom executable.

### URL-Encoded Sharing (No Server)

- **Model**: Encode the entire CalcMark document in the URL itself
  (base64 + compression). Zero infrastructure.
- **Fit**: Ideal for browser/WASM distribution. No backend needed.
- **Limitation**: URL length (~2KB safe, ~8KB practical).
- **Note**: This doesn't fit the subprocess plugin model — would need
  different integration (likely WASM-specific).

### GitLab Snippets

- **Model**: Mirrors GitHub Gists. Git-backed, versioned, public/private.
- **Fit**: Strong for teams already on GitLab. Official Go SDK available.
- **Auth**: PAT or OAuth 2.0 via GitLab CLI (`glab`).
- **CLI**: `glab snippet create` / `glab snippet view`.

### S3-Compatible Storage (S3, R2, MinIO)

- **Model**: PUT/GET on a bucket. Users bring their own credentials.
- **Fit**: "Save to my cloud" rather than "share via link." Pre-signed URLs
  enable sharing.
- **Auth**: AWS credentials / service-specific auth.
- **CLI**: `aws s3 cp` or `rclone`.

### WebDAV (Nextcloud, ownCloud)

- **Model**: Standard file protocol for corporate/self-hosted environments.
- **Fit**: Teams behind firewalls. Auth handled by WebDAV server.
- **CLI**: `curl` with WebDAV methods or `cadaver`.

## Open Questions

None — all questions resolved during brainstorming.

## Implementation Considerations

### Prerequisite: `gh` CLI

V1 requires the GitHub CLI (`gh`) to be installed and on PATH. CalcMark
should check for `gh` availability and surface a clear error message
("Install gh: https://cli.github.com") rather than failing with a cryptic
exec error.

### TUI / Subprocess Interaction

CalcMark's TUI uses bubbletea, which owns the terminal. When shelling out
to `gh`, if the CLI needs to prompt for auth, that prompt must render in
the same terminal. Bubbletea's `tea.Exec` command temporarily suspends the
TUI to hand the terminal to a subprocess, then resumes when the subprocess
exits. This is the standard pattern for TUI-to-subprocess handoff in the
Charm ecosystem.

## WASM Consideration

The subprocess plugin model does not work in a WASM/browser context.
Browser-based sharing would need a different integration path (likely
direct HTTP calls via Go's WASM-compatible net/http, or the URL-encoded
approach which needs no backend at all). This is explicitly out of scope
for V1 but worth noting for future architecture.
