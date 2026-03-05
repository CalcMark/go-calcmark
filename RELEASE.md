# Release Process

This document describes how to create and publish releases for go-calcmark.

## Overview

Two things happen automatically when you push to `main` and when you push a tag:

**Push to `main`** (when `site/**` changes):
- The **Hugo site** at [calcmark.org](https://calcmark.org) is built and deployed to GitHub Pages via `.github/workflows/site.yml`

**Push a semver tag** (`v*.*.*`):
1. **Git tag is the single source of truth** for the version number
2. **GoReleaser** builds cross-platform CLI binaries, creates archives with checksums, and generates a changelog
3. **Homebrew tap** is updated automatically (`CalcMark/homebrew-tap`)
4. **No manual scripts** — the release is driven by `.goreleaser.yaml` and `.github/workflows/release.yml`

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

Pushing commits to `main` will automatically deploy the site if any `site/**` files changed.

Pushing the tag will automatically:
- Build `cm` binary for all platforms (macOS, Linux, Windows × amd64/arm64)
- Sign and notarize macOS binaries (when signing secrets are configured)
- Create `.tar.gz` (unix) and `.zip` (windows) archives
- Generate SHA-256 checksums
- Create a GitHub release with an auto-generated changelog
- Update the Homebrew Cask in `CalcMark/homebrew-tap`

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

Each archive contains only the `cm` binary. macOS archives contain signed and notarized binaries when signing secrets are configured.

## macOS Code Signing and Notarization

GoReleaser v2 signs and notarizes macOS binaries automatically using [anchore/quill](https://github.com/anchore/quill). This runs cross-platform on the Linux CI runner — no macOS runner required.

When the `MACOS_SIGN_P12` secret is set, GoReleaser will:

1. Sign each darwin binary with a Developer ID Application certificate (hardened runtime enabled)
2. Submit the signed binary to Apple's notary service and wait for approval
3. Package the signed, notarized binary into the release archive

This eliminates the need for users to run `xattr -d com.apple.quarantine $(which cm)` after installing via Homebrew.

The `notarize.macos` block in `.goreleaser.yaml` controls this behavior. When the signing secrets are not configured (local builds, forks), signing is skipped automatically via the `enabled: '{{ isEnvSet "MACOS_SIGN_P12" }}'` guard.

### Setting Up Signing Secrets (First Time Only)

You need an [Apple Developer Program](https://developer.apple.com/programs/) membership.

**Developer ID Application certificate (.p12):**

1. In Xcode or Apple Developer portal, create a "Developer ID Application" certificate
2. Export it as a `.p12` file with a password
3. Base64-encode it: `base64 < DeveloperID.p12`
4. Store as `MACOS_SIGN_P12` secret, and the password as `MACOS_SIGN_PASSWORD`

**App Store Connect API key (.p8):**

1. Go to [App Store Connect → Users and Access → Integrations → Keys](https://appstoreconnect.apple.com/access/integrations/api)
2. Create a new key (note the Key ID and Issuer ID shown on the page)
3. Download the `.p8` private key file
4. Base64-encode it: `base64 < AuthKey_XXXXXXXXXX.p8`
5. Store as `MACOS_NOTARY_KEY`, with `MACOS_NOTARY_KEY_ID` and `MACOS_NOTARY_ISSUER_ID` as separate secrets

## Pre-release Versions

GoReleaser auto-detects pre-releases from the tag format:

- `v0.3.0-alpha.1` → marked as pre-release on GitHub
- `v0.3.0-beta.2` → marked as pre-release on GitHub
- `v0.3.0-rc.1` → marked as pre-release on GitHub
- `v0.3.0` → marked as latest release

This is controlled by `prerelease: auto` in `.goreleaser.yaml`.

## Website (calcmark.org)

The documentation site lives in `site/` and is built with [Hugo](https://gohugo.io/).

### Local Development

```bash
# Start the dev server with live reload
task site

# Build the site without serving (output in site/public/)
task site:build
```

### How Deployment Works

The site workflow (`.github/workflows/site.yml`) deploys to GitHub Pages:

1. **Trigger**: Push to `main` when files in `site/**` or `.github/workflows/site.yml` change
2. **Build**: Hugo v0.156.0 builds and minifies the site
3. **Deploy**: The built site is uploaded and deployed to GitHub Pages
4. **Domain**: Served at [calcmark.org](https://calcmark.org) via the `site/static/CNAME` file

The site deployment is independent of the release workflow — updating docs does not require a new version tag, and tagging a release does not redeploy the site unless `site/**` files were also changed.

### Updating Docs for a Release

If a release includes user-facing changes, update the site content before tagging:

1. Edit pages under `site/content/docs/`
2. Preview locally with `task site`
3. Commit and push to `main` (triggers site deploy)
4. Then tag and push the release

## Dry Run (Local Testing)

To test what GoReleaser would produce without publishing:

```bash
# Requires: go install github.com/goreleaser/goreleaser/v2@latest
goreleaser release --snapshot --clean
```

This builds all archives locally in `dist/` without creating a GitHub release or pushing to the Homebrew tap.

## CI/CD Details

### GitHub Actions Workflows

**Release** (`.github/workflows/release.yml`):

1. **Trigger**: Push of tags matching `v*.*.*`
2. **Setup**: Checks out code (full history), installs Go from `go.mod`
3. **Execute**: Runs GoReleaser v2

**Site deploy** (`.github/workflows/site.yml`):

1. **Trigger**: Push to `main` when `site/**` or the workflow file itself changes
2. **Build**: Installs Hugo, runs `hugo --source site --minify`
3. **Deploy**: Uploads to GitHub Pages via `actions/deploy-pages`
4. **Concurrency**: Only one site deploy runs at a time (`cancel-in-progress: true`)

### Required Secrets

| Secret | Purpose |
|--------|---------|
| `GITHUB_TOKEN` | Provided automatically by GitHub Actions. Creates releases, uploads artifacts, and deploys the site. |
| `HOMEBREW_TAP_GITHUB_TOKEN` | Personal access token with write access to `CalcMark/homebrew-tap`. Required for updating the Homebrew Cask. |
| `MACOS_SIGN_P12` | Base64-encoded Developer ID Application `.p12` certificate. Required for macOS code signing. |
| `MACOS_SIGN_PASSWORD` | Password for the `.p12` certificate. |
| `MACOS_NOTARY_ISSUER_ID` | App Store Connect API issuer UUID (shown on the Keys page). Required for notarization. |
| `MACOS_NOTARY_KEY_ID` | App Store Connect API key ID. |
| `MACOS_NOTARY_KEY` | Base64-encoded `.p8` API private key from App Store Connect. |

Signing secrets are optional — releases still work without them, but macOS binaries will be unsigned.

### Permissions

- **Release workflow**: `contents: write` to create releases and upload artifacts.
- **Site workflow**: `contents: read`, `pages: write`, and `id-token: write` for GitHub Pages deployment.

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
   - **macOS signing fails**: Verify `MACOS_SIGN_P12` is valid base64, the `.p12` password is correct, and the certificate has not expired
   - **Notarization fails**: Verify the App Store Connect API key is active and the issuer ID / key ID are correct

### Site deploy fails

1. Check the Actions tab: `https://github.com/CalcMark/go-calcmark/actions`
2. Common issues:
   - **Hugo version mismatch**: The workflow pins Hugo v0.156.0 — ensure `site/` content is compatible
   - **Pages not enabled**: Check repository Settings → Pages → Source is set to "GitHub Actions"
   - **CNAME missing**: `site/static/CNAME` must contain `calcmark.org`

### Site didn't update after push

The site workflow only triggers when `site/**` files change. If you only changed Go source code, the site will not redeploy. To force a redeploy, make a trivial change to a file under `site/` or re-run the workflow manually from the Actions tab.

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
- [ ] Site docs updated for user-facing changes: `task site` to preview
- [ ] All changes committed and pushed to `main`
- [ ] Tag created: `git tag -a "vX.Y.Z" -m "Release vX.Y.Z"`
- [ ] Breaking changes documented (for major versions)
- [ ] `HOMEBREW_TAP_GITHUB_TOKEN` secret is configured (first release only)
- [ ] macOS signing secrets configured (first release only, see [macOS Code Signing](#macos-code-signing-and-notarization))
- [ ] Apple Developer ID certificate has not expired

## Version Numbering

go-calcmark follows [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR**: Incompatible API or language changes
- **MINOR**: New functionality (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

## Post-Release

After a successful release:

1. Verify the release at `https://github.com/CalcMark/go-calcmark/releases`
2. Test Homebrew installation: `brew upgrade calcmark/tap/calcmark`
3. Verify the site at [calcmark.org](https://calcmark.org) reflects any doc updates
4. Update downstream projects that consume go-calcmark
