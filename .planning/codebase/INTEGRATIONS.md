# External Integrations

**Analysis Date:** 2026-02-01

## APIs & External Services

**None actively integrated.**

This is a self-contained language implementation with no external API dependencies. The interpreter, evaluator, and CLI operate entirely with built-in functionality and local file I/O.

## Data Storage

**Databases:**
- None - Project is stateless for document evaluation

**File Storage:**
- Local filesystem only
  - Reads: CalcMark documents from user's filesystem or stdin
  - Writes: User configuration to `~/.config/calcmark/config.toml` and `~/.calcmarkrc.toml`
  - Temporary: None (all processing in-memory)

**Caching:**
- In-memory cache: Configuration singleton pattern in `cmd/calcmark/config/config.go` (loaded once via `sync.Once`)
- No persistent caching layer

## Authentication & Identity

**Auth Provider:**
- None required - Project is a local tool with no multi-user or cloud infrastructure

**Configuration Sources:**
- User home directory detection: `os.UserHomeDir()` for XDG config paths
- No credentials or secrets management needed for core functionality

## Monitoring & Observability

**Error Tracking:**
- None - All errors surface as application-level messages to stderr

**Logs:**
- Stderr: Error messages and diagnostics (e.g., file validation errors in `cmd/calcmark/cmd/security.go`)
- Standard output: Calculation results, REPL output, formatted document output
- Development: Verbose modes available in `cm eval -v` for intermediate values

**Diagnostics:**
- Semantic validation: Diagnostic messages from `spec/semantic/` package
- File I/O validation: Path validation and file size limits in `cmd/calcmark/cmd/security.go`

## CI/CD & Deployment

**Hosting:**
- GitHub - Source code repository (CalcMark/go-calcmark)
- Binaries: Distributed as standalone executables via GitHub releases (built to `dist/` directory)

**CI Pipeline:**
- Not detected in analysis - Repository has `.github/` directory but specific CI workflow not examined
- Local validation: Full test suite and quality checks via Taskfile.yml (see `task quality`, `task release`)

**Build Automation:**
- Taskfile.yml orchestrates all build, test, and release tasks
- Multi-platform builds: Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64)
- WASM builds: Separate build target with embedded `wasm_exec.js`

## Environment Configuration

**Required env vars:**
- GOOS, GOARCH - For cross-platform builds (set by Taskfile.yml as needed)
- No runtime environment variables required for basic functionality

**Optional env vars:**
- Color mode: Can be overridden via `--color-mode` CLI flag (defaults to terminal detection)

**Secrets location:**
- None - Project has no secrets or credential requirements

## File I/O & Access

**Reading:**
- CalcMark document files: `os.ReadFile(filename)` in `cmd/calcmark/cmd/eval.go`
- Configuration files: Via `spf13/viper` from `~/.config/calcmark/config.toml` and `~/.calcmarkrc.toml`
- Path validation: `cmd/calcmark/cmd/security.go` enforces file size limits and path validation

**Writing:**
- Configuration: Loaded but not written back (read-only after startup)
- Document conversion output: `os.Create()` for `--output` flag in convert subcommand
- Stdout/Stderr: Results and errors directed appropriately

**Security:**
- File size limits enforced: See `TestFileSizeLimit` in `cmd/calcmark/security.go`
- Path validation enforced: See `TestPathValidation` in `cmd/calcmark/security.go`
- No arbitrary file access outside project scope

## Webhooks & Callbacks

**Incoming:**
- None - CLI is event-driven only via keyboard/stdin

**Outgoing:**
- None

## Terminal & UI Integration

**Terminal Libraries:**
- Charmbracelet's termenv: Terminal capability detection (color profiles, size detection)
- Clipboard integration: `atotto/clipboard` v0.1.4 for copy/paste operations
- ANSI escape codes: via `charmbracelet/x/ansi` for color and formatting

**Theme System:**
- Built-in themes: Configured in `cmd/calcmark/config/theme.go` and `cmd/calcmark/config/theme/`
- Lipgloss styling: Pre-computed styles at startup from theme configuration
- Dark theme support: Full dark theme with proper background handling (recent commit: `fdf374c`)

## Runtime Functions (Network & Storage Simulation)

**Note:** These are *calculation functions* within CalcMark documents, not external integrations.

**Network Performance Calculation:**
- Location: `impl/interpreter/network_functions.go`
- Functions: `rtt()`, `throughput()`, `transfer_time()`
- Purpose: Simulate network latencies and bandwidth for performance calculations
- Data: Constants (not external calls) for network scopes (local, regional, continental, global) and network types (gigabit, wifi, 4g, 5g, etc.)

**Storage Performance Calculation:**
- Location: `impl/interpreter/storage_functions.go`
- Functions: `read()`, `seek()`
- Purpose: Simulate storage access latencies for performance calculations
- Data: Constants for storage types (ssd, nvme, pcie_ssd, hdd) with throughput and seek times

These functions compute durations based on hardcoded performance constants; they do not make actual system calls or network requests.

## Output Formatting

**Formatters:**
- Text: Human-readable plaintext output (default)
- HTML: HTML document generation via templates (with optional Handlebars template file)
- Format location: `format/` directory with pluggable formatter interface

**Markdown Rendering:**
- Terminal markdown: Via `charmbracelet/glamour` for preview pane display
- HTML markdown: Via `gomarkdown/markdown` for full HTML conversion

---

*Integration audit: 2026-02-01*
