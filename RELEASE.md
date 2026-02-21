# Release Process

This document describes how to create and publish releases for go-calcmark.

## Overview

Releases are automated via **GoReleaser** and **GitHub Actions**. Pushing a semver tag triggers the full pipeline:

1. **Git tag is the single source of truth** for the version number
2. **GoReleaser** builds cross-platform CLI binaries, creates archives with checksums, and generates a changelog
3. **Homebrew tap** is updated automatically (`CalcMark/homebrew-tap`)
4. **No manual scripts** — the entire process is driven by `.goreleaser.yaml` and `.github/workflows/release.yml`

## Quick Start

```bash
# 1. Ensure all tests and quality checks pass
task test
task quality

# 2. Push your commits to GitHub
git push origin main

# 3. Create an annotated tag
git tag -a "v0.3.0" -m "Release v0.3.0"

# 4. Push the tag (triggers the release workflow)
git push origin v0.3.0
```

GitHub Actions will automatically:
- Build `cm` binary for all platforms (macOS, Linux, Windows × amd64/arm64)
- Create `.tar.gz` (unix) and `.zip` (windows) archives
- Generate SHA-256 checksums
- Create a GitHub release with an auto-generated changelog
- Update the Homebrew formula in `CalcMark/homebrew-tap`

## How Versioning Works

The version is injected into the `cm` binary at build time via ldflags:

```
-X main.Version={{.Version}}
-X main.BuildTime={{.Date}}
```

GoReleaser derives `{{.Version}}` from the git tag (stripping the `v` prefix). There is no version constant in the source code — the git tag is the single source of truth.

For local development builds, `task build` uses `git describe --tags --always --dirty` to set the version.

## Release Artifacts

Each release produces the following archives:

| Platform | Archive |
|----------|---------|
| macOS (Apple Silicon) | `calcmark_VERSION_darwin_arm64.tar.gz` |
| macOS (Intel) | `calcmark_VERSION_darwin_amd64.tar.gz` |
| Linux (x64) | `calcmark_VERSION_linux_amd64.tar.gz` |
| Linux (arm64) | `calcmark_VERSION_linux_arm64.tar.gz` |
| Linux (arm v6) | `calcmark_VERSION_linux_armv6.tar.gz` |
| Windows (x64) | `calcmark_VERSION_windows_amd64.zip` |

Plus `checksums.txt` (SHA-256 for all archives).

Each archive contains only the `cm` binary.

## Pre-release Versions

GoReleaser auto-detects pre-releases from the tag format:

- `v0.3.0-alpha.1` → marked as pre-release on GitHub
- `v0.3.0-beta.2` → marked as pre-release on GitHub
- `v0.3.0-rc.1` → marked as pre-release on GitHub
- `v0.3.0` → marked as latest release

This is controlled by `prerelease: auto` in `.goreleaser.yaml`.

## Dry Run (Local Testing)

To test what GoReleaser would produce without publishing:

```bash
# Requires: go install github.com/goreleaser/goreleaser/v2@latest
goreleaser release --snapshot --clean
```

This builds all archives locally in `dist/` without creating a GitHub release or pushing to the Homebrew tap.

## CI/CD Details

### GitHub Actions Workflow

The release workflow (`.github/workflows/release.yml`):

1. **Trigger**: Push of tags matching `v*.*.*`
2. **Setup**: Checks out code (full history), installs Go from `go.mod`
3. **Execute**: Runs GoReleaser v2

### Required Secrets

| Secret | Purpose |
|--------|---------|
| `GITHUB_TOKEN` | Provided automatically by GitHub Actions. Creates the release and uploads artifacts. |
| `HOMEBREW_TAP_GITHUB_TOKEN` | Personal access token with write access to `CalcMark/homebrew-tap`. Required for updating the Homebrew formula. |

### Permissions

The workflow requires `contents: write` to create releases and upload artifacts.

## Troubleshooting

### Tests fail before release

**Solution**: Fix tests before tagging. Always run:

```bash
task test
task quality
```

### GoReleaser build fails in CI

1. Check the Actions tab: `https://github.com/CalcMark/go-calcmark/actions`
2. Click the failed workflow run for detailed logs
3. Common issues:
   - **Go version mismatch**: Workflow reads from `go.mod`, ensure it's current
   - **Permission denied**: Check repository Settings → Actions → Workflow permissions
   - **Homebrew tap push fails**: Verify `HOMEBREW_TAP_GITHUB_TOKEN` secret exists and has write access

### Need to redo a release

```bash
# Delete the tag locally and remotely
git tag -d v0.3.0
git push origin :refs/tags/v0.3.0

# Delete the GitHub release manually (if created)
# Then fix the issue, commit, and re-tag
git tag -a "v0.3.0" -m "Release v0.3.0"
git push origin v0.3.0
```

## Release Checklist

Before pushing a tag:

- [ ] All tests pass: `task test`
- [ ] Quality checks pass: `task quality`
- [ ] All changes committed and pushed to `main`
- [ ] Tag created: `git tag -a "vX.Y.Z" -m "Release vX.Y.Z"`
- [ ] Breaking changes documented (for major versions)
- [ ] `HOMEBREW_TAP_GITHUB_TOKEN` secret is configured (first release only)

## Version Numbering

go-calcmark follows [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR**: Incompatible API or language changes
- **MINOR**: New functionality (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

## Post-Release

After a successful release:

1. Verify the release at `https://github.com/CalcMark/go-calcmark/releases`
2. Test Homebrew installation: `brew install calcmark/tap/calcmark`
3. Update downstream projects that consume go-calcmark
