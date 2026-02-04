# Phase 7: Distribution - Context

**Gathered:** 2026-02-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Package and distribute CalcMark so users can install via Homebrew (macOS/Linux) or download prebuilt binaries from GitHub releases. Make the tool accessible across macOS, Linux, and Windows platforms.

**Scope change:** WASM distribution is removed from this phase. WASM infrastructure will be deleted as part of cleanup.

</domain>

<decisions>
## Implementation Decisions

### Release channels
- Homebrew tap: `calcmark/tap` (standard pattern: `brew tap calcmark/tap && brew install calcmark`)
- No Scoop bucket for Windows — users download zip from GitHub releases
- No shell completions in release archives — users run `cm completion bash > ~/.bashrc` manually

### Binary naming & structure
- Binary name: `cm` (short, current name)
- Archive format: Claude's discretion (follow platform conventions)
- Archive contents: Just the binary (no LICENSE, README, etc.)
- Platforms to build:
  - macOS amd64, arm64
  - Linux amd64, arm64, arm (32-bit)
  - Windows amd64, arm64
  - **Total: 7 binaries**

### WASM removal
- Remove all WASM infrastructure from the project
- Delete wasm/ directory and WASM build targets from Taskfile
- Update roadmap success criteria to remove WASM requirement
- Do this as first task in Phase 7 Plan 01

### Signing & verification
- No macOS code signing or notarization (users bypass Gatekeeper if needed)
- No Windows code signing (users may see SmartScreen warning)
- Checksums: Claude's discretion
- GPG signatures: Claude's discretion

### Claude's Discretion
- Version tag format (v1.0.0 vs 1.0.0) — follow Go ecosystem norms
- Archive format per platform (tar.gz vs zip)
- Checksum format (single SHA256SUMS file vs per-file .sha256)
- Whether to include GPG signatures

</decisions>

<specifics>
## Specific Ideas

- Keep it simple — no paid certificates for code signing
- Homebrew is the primary installation path for macOS/Linux users
- Windows is secondary — zip download from releases is sufficient

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 07-distribution*
*Context gathered: 2026-02-04*
