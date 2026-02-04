---
phase: 07-distribution
plan: 01
subsystem: release
tags: [goreleaser, github-actions, homebrew, cross-compilation]

dependency_graph:
  requires: []
  provides:
    - GoReleaser configuration for 7 platform builds
    - GitHub Actions release workflow with GoReleaser
    - Homebrew formula generation configuration
  affects:
    - 07-02: tap repo setup will use this configuration
    - 07-03: CHANGELOG.md can integrate with GoReleaser changelog

tech_stack:
  added:
    - goreleaser@v2
    - goreleaser/goreleaser-action@v6
  patterns:
    - ldflags version injection at build time
    - Cross-compilation via CGO_ENABLED=0

key_files:
  created:
    - .goreleaser.yaml
  modified:
    - Taskfile.yml
    - .github/workflows/release.yml
  deleted:
    - impl/wasm/main.go
    - impl/wasm/README.md
    - impl/wasm/json_structure_test.go
    - impl/wasm/tokenize_positions_test.go
    - release.sh

decisions:
  - decision: Use GoReleaser v2 with brews section (despite deprecation warning)
    rationale: brews is still correct for CLI formulas; homebrew_casks is for GUI apps
    scope: release configuration

metrics:
  duration: 5min
  completed: 2026-02-04
---

# Phase 7 Plan 01: GoReleaser Configuration Summary

**One-liner:** GoReleaser v2 cross-compiles 7 platform binaries with version injection and Homebrew formula generation

## What Was Built

1. **GoReleaser configuration** (`.goreleaser.yaml`)
   - Builds for 7 platforms: macOS (amd64, arm64), Linux (amd64, arm64, arm), Windows (amd64, arm64)
   - Uses ldflags to inject `main.Version` and `main.BuildTime` at compile time
   - Archives: tar.gz for Unix, zip for Windows
   - Checksums: SHA256 in single `checksums.txt` file
   - Homebrew formula auto-generation for `CalcMark/homebrew-tap`

2. **GitHub Actions workflow update** (`.github/workflows/release.yml`)
   - Replaced custom `release.sh` with `goreleaser/goreleaser-action@v6`
   - Added `HOMEBREW_TAP_GITHUB_TOKEN` for tap push
   - Kept `fetch-depth: 0` for changelog generation

3. **WASM infrastructure removal**
   - Deleted `impl/wasm/` directory (main.go, tests, README, binary files)
   - Deleted `release.sh` script
   - Removed all WASM references from `Taskfile.yml`
   - Simplified test, lint, vet, modernize, staticcheck tasks

## Verification Results

| Check | Result |
|-------|--------|
| `goreleaser check` | Passes (with expected brews deprecation warning) |
| `goreleaser build --snapshot --clean` | 7 binaries built |
| Version injection | `0.1.23-SNAPSHOT-69d6972` correctly shown |
| `task test` | All tests pass |
| `task lint` | Passes |

## Deviations from Plan

None - plan executed exactly as written.

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | 69d6972 | Remove WASM infrastructure |
| 2 | cc76053 | Add GoReleaser configuration |
| 3 | ef20700 | Update release workflow |

## Next Phase Readiness

**For Plan 02 (Homebrew tap):**
- GoReleaser brews section configured for `CalcMark/homebrew-tap`
- Need to create tap repository and `HOMEBREW_TAP_GITHUB_TOKEN` secret

**For Plan 03 (CHANGELOG):**
- GoReleaser changelog section configured with conventional commit filters
- Excludes: docs, test, chore, merge commits
