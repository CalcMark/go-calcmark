# Technology Stack

**Analysis Date:** 2026-02-01

## Languages

**Primary:**
- Go 1.24.4 - Full implementation of language specification, interpreter, CLI, TUI, and WASM target

**Secondary:**
- None

## Runtime

**Environment:**
- Go runtime (native execution)
- WebAssembly (GOOS=js GOARCH=wasm) - First-class build target alongside native binaries

**Package Manager:**
- Go modules (go.mod)
- Lockfile: go.sum (present)

## Frameworks

**Core:**
- Charmbracelet suite:
  - `bubbles` v0.21.0 - TUI component library
  - `bubbletea` v1.3.10 - TUI framework (Elm architecture)
  - `lipgloss` v1.1.1 - Terminal styling and theming (custom fork for background handling)
  - `glamour` v0.10.0 - Markdown rendering to terminal
  - `muesli/termenv` v0.16.0 - Terminal environment detection

**Markdown Processing:**
- `gomarkdown/markdown` v0.0.0-20250810172220-2e2c11897d1a - CommonMark parser
- `yuin/goldmark` v1.7.8 - Alternative markdown processor (indirect dependency)

**CLI:**
- `spf13/cobra` v1.10.1 - Command-line framework
- `spf13/viper` v1.21.0 - Configuration management (TOML format)

**Testing:**
- `knz/catwalk` v0.1.4 - Data-driven TUI testing framework

**Build/Dev:**
- `task` (Taskfile.yml) - Task runner (not a Go dependency; shell-based orchestration)

## Key Dependencies

**Critical:**
- `shopspring/decimal` v1.4.0 - Arbitrary precision decimal arithmetic for calculations (prevents floating-point errors in financial/scientific calculations)

**Infrastructure:**
- `google/uuid` v1.6.0 - UUID generation for documents
- `martinlindhe/unit` v0.0.0-20230420213220-4adfd7d0a0d6 - Unit conversion library (dates, times, physical units)

**Development & Indirect:**
- `charmbracelet/colorprofile` - Terminal color capability detection
- `charmbracelet/x/ansi` - ANSI code handling
- `charmbracelet/x/cellbuf` - Terminal cell buffer management
- `charmbracelet/x/term` - Terminal operations
- `cockroachdb/datadriven` - Test data-driven testing
- `dlclark/regexp2` - Regex engine (via syntax highlighting)
- `alecthomas/chroma/v2` - Syntax highlighting
- `golang.org/x/text` v0.30.0 - Unicode and text handling
- `gopkg.in/yaml.v3` v3.0.1 - YAML parsing (configuration)

## Configuration

**Environment:**
- Embedded configuration: `cmd/calcmark/config/defaults.toml` (compiled into binary via `//go:embed`)
- User configuration (XDG standard): `~/.config/calcmark/config.toml` (TOML format)
- Legacy fallback: `~/.calcmarkrc.toml`
- Configuration loading: `cmd/calcmark/config/config.go` - Singleton pattern with `sync.Once`

**Build:**
- Configuration files: `.golangci.yml` (linting), `Taskfile.yml` (task orchestration)
- Build flags: Version and BuildTime set via ldflags at build time
- LDFLAGS: `-s -w -X main.Version={{.VERSION}} -X main.BuildTime={{.BUILD_TIME}}`

## Platform Requirements

**Development:**
- Go 1.24.4 or later
- Unix-like shell (for Taskfile.yml task execution)
- For lint checks: `golangci-lint` (optional, run via go run if not installed)
- For modernize checks: `golang.org/x/tools/gopls` (run via go run)
- For staticcheck: `honnef.co/go/tools` (run via go run)

**Production:**
- Native binaries: Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64)
- WASM: Browser environment with JavaScript support (wasm_exec.js bundled in `dist/`)
- Runtime: Minimal - no external services required for core functionality
- File I/O: Standard filesystem access (reads/writes to `~/.config/calcmark/`)

**Deployment Targets:**
- CLI: Standalone binary distribution in `dist/` directory
- WASM: Browser-based via `dist/calcmark.wasm` + `dist/wasm_exec.js`
- Installation: `go install` or direct binary usage

## Version Management

**Versioning:**
- Semantic versioning from git tags (via `git describe --tags --always --dirty`)
- Version v0.1.1 (Phase 1 complete - Reserved keywords and multi-token functions)
- Build metadata: Version and BuildTime injected at compile time

---

*Stack analysis: 2026-02-01*
