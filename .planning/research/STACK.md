# Stack Research

**Domain:** Go CLI/TUI developer tools (v1 release polish for CalcMark interpreter)
**Researched:** 2026-02-02
**Confidence:** HIGH (existing stack verified against current releases; additions verified via WebSearch + official docs)

## Current Stack Audit

CalcMark already has a working interpreter and TUI. This research validates the existing stack and recommends additions for v1 release features: TUI editor rewrite, help system, autocomplete, YAML front matter, documentation, and prebuilt binaries.

### What Already Works (Keep)

| Technology | Current Version | Latest Stable | Status | Action |
|------------|----------------|---------------|--------|--------|
| Go | 1.24.4 | 1.24.12 | Patch behind | **Update to 1.24.12** (security fixes in crypto/tls, net/url, compiler) |
| bubbletea | v1.3.10 | v1.3.10 | Current | **Keep** -- v2 is RC only, not stable yet |
| bubbles | v0.21.0 | v0.21.0 | Current | **Keep** -- textarea component is solid for editor rewrite |
| lipgloss | v1.1.1-0.2025... (pseudo) | v1.x stable | Pre-release pin | **Evaluate** -- pinned to a pseudo-version; consider stabilizing |
| glamour | v0.10.0 | v0.10.0 | Current | **Keep** -- markdown rendering for preview pane |
| cobra | v1.10.1 | v1.10.2 | Patch behind | **Update to v1.10.2** |
| viper | v1.21.0 | v1.21.x | Current | **Keep** -- config management with TOML |
| gopkg.in/yaml.v3 | v3.0.1 | v3.0.1 | Current but unmaintained | **Evaluate migration** (see below) |
| shopspring/decimal | v1.4.0 | v1.4.x | Current | **Keep** -- decimal arithmetic for calculations |
| knz/catwalk | v0.1.4 | v0.1.4 | Current | **Keep** -- TUI data-driven testing |
| go-task/task | v3 (Taskfile.yml) | v3.48.0 | N/A (external tool) | **Keep** -- task runner |

## Recommended Stack

### Core Technologies (Already In Use -- Validated)

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go | 1.24.12 | Language runtime | Current stable with security patches. Go 1.25 available but 1.24 is the safer choice for a v1 release -- it has 12 patch releases of stability. Go 1.26 RC1 exists but is too new. |
| charmbracelet/bubbletea | v1.3.10 | TUI framework (Elm architecture) | The industry standard for Go TUI apps (9,310+ importers). v2 is at RC2 but NOT stable -- stay on v1 for v1 release. v1 has no breaking changes commitment. |
| charmbracelet/bubbles | v0.21.0 | TUI components (textarea, viewport) | The textarea component handles multi-line editing with line numbers, cursor, scrolling. Already used in ModelV2. v2 is at RC1 -- stay on v1. |
| charmbracelet/lipgloss | v1.x stable | Terminal styling/layout | Style definitions, layout primitives, side-by-side rendering. Already deeply integrated. Pin to a stable tag instead of pseudo-version. |
| charmbracelet/glamour | v0.10.0 | Markdown rendering in terminal | Stylesheet-based markdown rendering for the preview pane. v2 exists but is early (5 importers vs 710). Stay on v1. |
| spf13/cobra | v1.10.2 | CLI command framework | Industry standard (Kubernetes, Docker, Hugo). Built-in shell completion, man page generation, help formatting. Update from v1.10.1 for latest fixes. |
| spf13/viper | v1.21.0 | Configuration management | Handles TOML config with embedded defaults, XDG paths. Already working well. |
| knz/catwalk | v0.1.4 | TUI data-driven testing | Critical for editor testing. Enables key-sequence reproduction of bugs. No alternatives in this niche. |

### New Libraries to Add

| Library | Version | Purpose | Why Recommended | Confidence |
|---------|---------|---------|-----------------|------------|
| adrg/frontmatter | latest | YAML front matter parsing | Pure Go, MIT, supports YAML/TOML/JSON front matter with `---` delimiters. Returns remaining content as []byte -- perfect for CalcMark's document parsing pipeline. Zero dependencies beyond go-yaml. | HIGH |
| spf13/cobra/doc | (included in cobra) | Man page + markdown doc generation | Already a dependency via cobra. Generates man pages, markdown, and RST from command tree. `GenManTree` for man pages, `GenMarkdownTree` for reference docs. Zero additional deps. | HIGH |
| goreleaser | v2.13.3 | Binary building and release automation | Standard tool for Go binary distribution. Handles cross-compilation, GitHub releases, Homebrew taps, checksums, SBOM generation. Use `.goreleaser.yml` v2 config format. | HIGH |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| go-task/task v3.48.0 | Build/test/quality runner | Already in use. Add new tasks for `docs:generate`, `release:snapshot`, `completion:generate`. |
| goreleaser v2.13.3 | Release automation | Install via `go install github.com/goreleaser/goreleaser@latest` or brew. Use `goreleaser release --snapshot --clean` for local testing. |
| golangci-lint | Linting suite | Already in use via `task lint:strict`. Keep as-is. |
| staticcheck | Static analysis | Already in use. Keep as-is. |

## Additions by Feature Area

### 1. TUI Editor Rewrite

**No new dependencies needed.** The existing bubbletea v1 + bubbles v0.21.0 + lipgloss v1 stack is correct. The rewrite is architectural, not library-level.

Key libraries already available:
- `bubbles/textarea` -- multi-line text editing with line numbers, scrolling, cursor management
- `bubbles/viewport` -- scrollable content viewing (useful if preview pane needs independent scrolling)
- `lipgloss` -- layout computation, side-by-side rendering, theming
- `knz/catwalk` -- data-driven testing of key sequences

**Architecture note:** The project has both `Model` (custom editor) and `ModelV2` (textarea-based). The rewrite should converge on one approach. The textarea component handles most editing concerns (cursor, selection, scrolling, undo) that the custom Model implements manually.

### 2. Help System

**No new dependencies needed.** Use cobra's built-in capabilities:

| Feature | Implementation | Library |
|---------|---------------|---------|
| `--help` formatting | Cobra's built-in help template system | spf13/cobra |
| Man pages | `cobra/doc.GenManTree()` | spf13/cobra/doc (already a dep) |
| Markdown reference | `cobra/doc.GenMarkdownTree()` | spf13/cobra/doc |
| In-TUI help overlay | Custom bubbletea component using glamour | charmbracelet/glamour (already a dep) |
| Interactive help | Consider charmbracelet/boa for interactive cobra help | Optional -- evaluate during implementation |

**Recommendation:** Use `cobra/doc` to auto-generate man pages and markdown docs at build time. Add a `task docs:generate` target. For in-TUI help (e.g., `?` key showing help overlay), render markdown with glamour inside a bubbletea viewport.

### 3. Autocomplete

**Two distinct autocomplete contexts:**

#### Shell Completion (CLI level)
**No new dependencies.** Cobra has built-in shell completion for bash, zsh, fish, and PowerShell.

```go
// Already available -- just enable it
rootCmd.CompletionOptions.DisableDefaultCmd = false // Currently disabled!
```

Add `ValidArgsFunction` for file arguments (`.cm` files) and `RegisterFlagCompletionFunc` for flag values.

#### In-Editor Autocomplete (TUI level)
**No new dependencies.** The project already has `components/suggest.go` with `SuggestionSource` interface, `AutosuggestState`, rendering functions, and a `VariableSuggestionSource`.

What needs building is the **suggestion data sources** (functions, units, variables from the CalcMark spec) and the **trigger logic** in the editor model. The rendering infrastructure exists.

### 4. YAML Front Matter

**One new dependency:** `adrg/frontmatter`

| Option | Recommendation | Rationale |
|--------|---------------|-----------|
| adrg/frontmatter | **Use this** | Clean API: `frontmatter.Parse(reader, &metadata)` returns remaining content. Supports `---` YAML delimiters. MIT license. 335+ stars. |
| Manual parsing | Reject | Parsing `---` delimiters manually is error-prone (edge cases with `---` in content, blank lines, BOM). |
| goldmark/frontmatter | Reject | Tied to goldmark markdown parser; CalcMark uses its own parser pipeline. |

**Note on go-yaml:** The project uses `gopkg.in/yaml.v3` which was marked unmaintained in April 2025. The new community-maintained fork is at `go.yaml.in/yaml/v3`. However, `adrg/frontmatter` uses `gopkg.in/yaml.v3` internally, and CalcMark already depends on it. **Do not migrate yaml libraries during the v1 release.** Flag for post-v1 evaluation.

### 5. Documentation Tooling

**No new Go dependencies.** Documentation strategy:

| Doc Type | Tool | Notes |
|----------|------|-------|
| CLI reference (man pages) | `cobra/doc.GenManTree()` | Auto-generated from cobra command tree |
| CLI reference (markdown) | `cobra/doc.GenMarkdownTree()` | For GitHub wiki or docs site |
| Language reference | Hand-written markdown | CalcMark spec is already well-documented in `./spec/` |
| README/landing | Hand-written markdown | GitHub renders natively |
| API documentation | pkg.go.dev | Automatic via Go module publishing |
| User guide | Markdown files in `/docs/` | Keep it simple -- GitHub renders well |

**Reject:** MkDocs, Hugo, Docusaurus -- these are overkill for a CLI tool's documentation. GitHub's native markdown rendering plus auto-generated cobra docs is sufficient for v1.

### 6. Prebuilt Binaries

**One new tool dependency:** GoReleaser v2.13.3

| Component | Implementation | Notes |
|-----------|---------------|-------|
| Cross-compilation | `.goreleaser.yml` builds section | linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64 |
| GitHub Releases | GoReleaser + GitHub Actions | Triggered on tag push (e.g., `v1.0.0`) |
| Checksums | GoReleaser default | SHA256 checksums file auto-generated |
| Homebrew | GoReleaser homebrew tap | Optional -- publish to a `homebrew-tap` repo |
| WASM binary | Custom build step | Already have `task build:wasm` -- include in release workflow |
| Version embedding | ldflags (already in Taskfile) | `-X main.Version={{.VERSION}}` -- GoReleaser handles this natively |
| Shell completions | GoReleaser + cobra | Bundle completion scripts in release archives |

**The project already has cross-compilation in `Taskfile.yml`** (`build:linux`, `build:darwin`, `build:windows`). GoReleaser replaces this with a more standardized approach that also handles release notes, checksums, and distribution.

## Installation Commands

```bash
# No new Go module dependencies needed for most features
# Only addition for YAML front matter:
go get github.com/adrg/frontmatter

# Update existing deps to latest patch versions:
go get github.com/spf13/cobra@v1.10.2

# Tool dependencies (not in go.mod):
go install github.com/goreleaser/goreleaser/v2@v2.13.3

# Update Go patch version:
# Download Go 1.24.12 from https://go.dev/dl/
```

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| bubbletea v1 | bubbletea v2 (RC2) | After v2 reaches stable release AND CalcMark v1 is shipped. v2 has breaking changes (new import path `charm.land/bubbletea/v2`, View() returns struct instead of string, mouse API overhaul). Not worth the risk for v1. |
| bubbles textarea | Custom editor model (current Model) | Never for basic editing. The custom Model is ~1000 lines of cursor/scroll/wrap logic that textarea handles. Keep custom only for domain-specific behavior (aligned panes, calc evaluation). |
| adrg/frontmatter | Manual `---` parsing | Only if you need zero dependencies. But go-yaml is already a dep, so frontmatter adds almost nothing. |
| goreleaser | Manual `GOOS/GOARCH` cross-compilation | Only for one-off builds. For repeatable releases with checksums, taps, and release notes, goreleaser is the standard. |
| cobra/doc man pages | cobraman (rayjohnson/cobra-man) | Only if cobra/doc's man page output is insufficient. cobraman uses Go templates for more flexible output but adds a dependency. |
| gopkg.in/yaml.v3 | go.yaml.in/yaml/v3 | Post-v1 migration. The old import path still works. The new maintainers are working on making the transition smooth. |
| glamour v1 | glamour v2 | After v2 matures (currently 5 importers vs 710). v2 exists but the ecosystem hasn't migrated yet. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| bubbletea v2 RC | Not stable, breaking changes to View(), imports, mouse API. Shipping a v1 product on a v2 RC framework is risky. | bubbletea v1.3.10 |
| lipgloss v2 beta | Not stable, color API changed, compat package needed for migration | lipgloss v1.x |
| charmbracelet/huh | Form library for surveys/prompts -- wrong paradigm for a document editor | Custom bubbletea components |
| tcell/tview | Lower-level TUI library. Would require rewriting everything. bubbletea is better for this architecture. | bubbletea + bubbles |
| termbox-go | Unmaintained since 2020. | bubbletea |
| goccy/go-yaml | Better YAML spec compliance but different API. Switching would break viper integration and require testing all config paths. | gopkg.in/yaml.v3 (current) |
| MkDocs / Hugo / Docusaurus | Overkill for CLI tool docs. Adds Python/Node dependency to a pure-Go project. | cobra/doc + handwritten markdown |
| go install for distribution | Requires users have Go toolchain installed. Most CalcMark users want a binary. | GoReleaser prebuilt binaries |

## Stack Patterns by Feature Phase

**If building TUI editor rewrite first:**
- Use bubbles/textarea as the editing core
- Keep lipgloss for layout and theming
- Use catwalk for all new test coverage
- No new deps needed

**If building help system first:**
- Enable cobra's built-in completion command
- Add `cobra/doc` generation to Taskfile
- Use glamour for in-TUI help rendering
- No new deps needed

**If building YAML front matter first:**
- Add `adrg/frontmatter` dependency
- Integrate at the document parsing layer (`spec/document`)
- Front matter feeds metadata to the evaluator
- One new dep

**If building release pipeline first:**
- Add `.goreleaser.yml` configuration
- Add GitHub Actions workflow
- Replace manual `build:all` with goreleaser
- No new Go deps, one new tool dep

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| bubbletea v1.3.10 | bubbles v0.21.0 | Both v1 line, tested together |
| bubbletea v1.3.10 | lipgloss v1.x | Both v1 line. Do NOT mix with lipgloss v2. |
| cobra v1.10.2 | viper v1.21.0 | Standard pairing, no conflicts |
| cobra v1.10.2 | pflag v1.0.10 | cobra v1.10.x requires pflag >= v1.0.8 |
| adrg/frontmatter | gopkg.in/yaml.v3 | frontmatter uses yaml.v3 internally -- same dep |
| Go 1.24.12 | All above | All libraries support Go 1.24.x |
| goreleaser v2.13.3 | Go 1.24.x | GoReleaser v2.12+ explicitly supports Go 1.24 and 1.25 |

## Risk Assessment: Charm v2 Migration

The entire Charm ecosystem (bubbletea, bubbles, lipgloss, glamour) is moving to v2 with release candidates available. This is the elephant in the room for any Go TUI project.

**Recommendation: Ship v1 on Charm v1, plan v2 migration post-release.**

Rationale:
1. Charm v2 is at RC stage -- close to stable but not there yet
2. v2 has significant breaking changes (import paths, View() API, mouse events, key handling)
3. CalcMark has deep lipgloss integration (background handling, color profiles, themed styles) that would need careful migration
4. The v1 line receives bugfixes and is explicitly supported until v2 is stable
5. Shipping on stable dependencies is the right call for a v1 release

**Timeline estimate:** Charm v2 stable likely lands in Q1-Q2 2026. Plan CalcMark v2 migration as a dedicated milestone after v1 ships.

## Sources

- [Bubble Tea releases](https://github.com/charmbracelet/bubbletea/releases) -- verified v1.3.10 is latest stable (HIGH confidence)
- [Bubble Tea v2 migration guide](https://github.com/charmbracelet/bubbletea/discussions/1374) -- breaking changes documented (HIGH confidence)
- [Bubbles releases](https://github.com/charmbracelet/bubbles/releases) -- verified v0.21.0 is latest stable (HIGH confidence)
- [Lipgloss releases](https://github.com/charmbracelet/lipgloss/releases) -- v1 stable, v2 in beta (HIGH confidence)
- [Glamour releases](https://github.com/charmbracelet/glamour/releases) -- v0.10.0 latest stable (HIGH confidence)
- [Cobra pkg.go.dev](https://pkg.go.dev/github.com/spf13/cobra) -- v1.10.2 latest (HIGH confidence)
- [Cobra shell completion docs](https://cobra.dev/docs/how-to-guides/shell-completion/) -- built-in completion guide (HIGH confidence)
- [cobra/doc pkg.go.dev](https://pkg.go.dev/github.com/spf13/cobra/doc) -- man page generation API (HIGH confidence)
- [GoReleaser releases](https://github.com/goreleaser/goreleaser/releases) -- v2.13.3 latest (HIGH confidence)
- [GoReleaser quick start](https://goreleaser.com/quick-start/) -- configuration best practices (HIGH confidence)
- [adrg/frontmatter GitHub](https://github.com/adrg/frontmatter) -- YAML front matter parsing (HIGH confidence)
- [adrg/frontmatter pkg.go.dev](https://pkg.go.dev/github.com/adrg/frontmatter) -- API documentation (HIGH confidence)
- [go-yaml maintenance transition](https://github.com/go-yaml/yaml) -- marked unmaintained April 2025, new home at go.yaml.in (MEDIUM confidence)
- [Go 1.24 release notes](https://go.dev/doc/go1.24) -- features and patch history (HIGH confidence)
- [Go release history](https://go.dev/doc/devel/release) -- 1.24.12 is latest 1.24.x (HIGH confidence)
- [Task releases](https://github.com/go-task/task/releases) -- v3.48.0 latest (HIGH confidence)

---
*Stack research for: CalcMark v1 release -- Go CLI/TUI developer tools*
*Researched: 2026-02-02*
