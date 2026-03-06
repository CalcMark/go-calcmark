# Release Process

This document is the single source of truth for creating and publishing go-calcmark releases.

## Overview

Two things happen automatically when you push to `main` and when you push a tag:

**Push to `main`** (when `site/**` changes):
- The **Hugo site** at [calcmark.org](https://calcmark.org) is built and deployed to GitHub Pages via `.github/workflows/site.yml`

**Push a semver tag** (`v*.*.*`):
1. **Git tag is the single source of truth** for the version number
2. **GoReleaser** builds cross-platform CLI binaries, creates archives with checksums, and generates a changelog
3. **Homebrew Cask** is updated automatically (`CalcMark/homebrew-tap`)
4. **macOS binaries** are signed and notarized (when signing secrets are configured)
5. **No manual scripts** — the release is driven by `.goreleaser.yaml` and `.github/workflows/release.yml`

## First-Time Setup

Complete these steps once before the first release. If joining an existing project where these are already configured, skip to [Pre-flight Checks](#pre-flight-checks).

### 1. Create the Homebrew tap repository

If `CalcMark/homebrew-tap` already exists on GitHub, skip to step 2.

- Go to https://github.com/new
- Repository name: `homebrew-tap`
- Owner: `CalcMark` (or your org)
- Visibility: **Public** (required for Homebrew taps)
- Initialize with a README

### 2. Create a Personal Access Token (PAT)

- Go to https://github.com/settings/tokens
- Generate new token (**classic** — not fine-grained)
- Name: `homebrew-tap-access`
- Scopes: check **`repo`** (full control of private repositories)
- Copy the token immediately (you won't see it again)

### 3. Add the Homebrew secret

- Go to https://github.com/CalcMark/go-calcmark/settings/secrets/actions
- Click "New repository secret"
- Name: `HOMEBREW_TAP_GITHUB_TOKEN`
- Value: paste the PAT from step 2

### 4. macOS code signing and notarization (optional)

Signing is optional for your first release. Unsigned releases work, but macOS users must run `xattr -d com.apple.quarantine $(which cm)` after installing via Homebrew. You can add signing later.

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

### 5. Verify secrets

All secrets are configured at https://github.com/CalcMark/go-calcmark/settings/secrets/actions

| Secret | Purpose | Required |
|--------|---------|----------|
| `GITHUB_TOKEN` | Provided automatically by GitHub Actions. Creates releases, uploads artifacts, and deploys the site. | Auto |
| `HOMEBREW_TAP_GITHUB_TOKEN` | PAT with write access to `CalcMark/homebrew-tap`. Updates the Homebrew Cask. | Yes |
| `MACOS_SIGN_P12` | Base64-encoded Developer ID Application `.p12` certificate. | Optional |
| `MACOS_SIGN_PASSWORD` | Password for the `.p12` certificate. | Optional |
| `MACOS_NOTARY_ISSUER_ID` | App Store Connect API issuer UUID (shown on the Keys page). | Optional |
| `MACOS_NOTARY_KEY_ID` | App Store Connect API key ID. | Optional |
| `MACOS_NOTARY_KEY` | Base64-encoded `.p8` API private key from App Store Connect. | Optional |

Signing secrets are optional — releases still work without them, but macOS binaries will be unsigned.

## Pre-flight Checks

Before tagging a release:

```bash
# Ensure working tree is clean
git status

# Ensure you're on main
git branch --show-current

# Run full test suite
task test

# Run quality checks
task quality
```

All tests and quality checks must pass before tagging. Do not skip this.

**Dry run (optional):** To test what GoReleaser would produce without publishing:

```bash
# Requires: go install github.com/goreleaser/goreleaser/v2@latest
goreleaser release --snapshot --clean
```

This builds all archives locally in `dist/` without creating a GitHub release or pushing to the Homebrew tap.

## Tag & Push

```bash
# 1. Ensure all changes are committed and pushed to main
git push origin main

# 2. Create an annotated tag (annotated tags are required — GoReleaser
#    uses the tag object for changelog generation)
git tag -a "v0.4.0" -m "Release v0.4.0"

# 3. Push the tag (triggers the release workflow)
git push origin v0.4.0
```

Always push a single tag explicitly. Never use `git push --tags` — it pushes all local tags, including any stale or experimental ones.

Pushing the tag will automatically:
- Build `cm` binary for all platforms (macOS, Linux, Windows — 7 archives total)
- Sign and notarize macOS binaries (when signing secrets are configured)
- Create `.tar.gz` (unix) and `.zip` (windows) archives
- Generate SHA-256 checksums
- Create a GitHub release with an auto-generated changelog
- Update the Homebrew Cask in `CalcMark/homebrew-tap`

## Post-Release Verification

After the GitHub Actions workflow completes:

1. **Check the release page** at `https://github.com/CalcMark/go-calcmark/releases`
   - Verify all 7 archives are present
   - Verify `checksums.txt` exists

2. **Test Homebrew installation** (on macOS/Linux)
   ```bash
   brew tap calcmark/tap
   brew install calcmark/tap/calcmark
   cm --version
   ```

3. **Test binary download**
   - Download an archive from the releases page
   - Extract and run `./cm --version`

4. **Verify the site** at [calcmark.org](https://calcmark.org) if any `site/**` files were updated

5. **Update downstream projects** that consume go-calcmark

## Reference

### How Versioning Works

The version is injected into the `cm` binary at build time via ldflags:

```
-X main.Version={{.Version}}
-X main.BuildTime={{.Date}}
```

GoReleaser derives `{{.Version}}` from the git tag (stripping the `v` prefix). There is no version constant in the source code — the git tag is the single source of truth.

For local development builds, `task build` uses `git describe --tags --always --dirty` to set the version.

### Version Numbering

go-calcmark follows [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR**: Incompatible API or language changes
- **MINOR**: New functionality (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Release Artifacts

Each release produces the following archives:

| Platform | Archive |
|----------|---------|
| macOS (Apple Silicon) | `calcmark_VERSION_darwin_arm64.tar.gz` |
| macOS (Intel) | `calcmark_VERSION_darwin_amd64.tar.gz` |
| Linux (x64) | `calcmark_VERSION_linux_amd64.tar.gz` |
| Linux (arm64) | `calcmark_VERSION_linux_arm64.tar.gz` |
| Linux (arm v6) | `calcmark_VERSION_linux_armv6.tar.gz` |
| Windows (x64) | `calcmark_VERSION_windows_amd64.zip` |
| Windows (arm64) | `calcmark_VERSION_windows_arm64.zip` |

Plus `checksums.txt` (SHA-256 for all archives).

Each archive contains only the `cm` binary. macOS archives contain signed and notarized binaries when signing secrets are configured.

### macOS Code Signing and Notarization

GoReleaser v2 signs and notarizes macOS binaries automatically using [anchore/quill](https://github.com/anchore/quill). This runs cross-platform on the Linux CI runner — no macOS runner required.

When the `MACOS_SIGN_P12` secret is set, GoReleaser will:

1. Sign each darwin binary with a Developer ID Application certificate (hardened runtime enabled)
2. Submit the signed binary to Apple's notary service and wait for approval
3. Package the signed, notarized binary into the release archive

The `notarize.macos` block in `.goreleaser.yaml` controls this behavior. When the signing secrets are not configured (local builds, forks), signing is skipped automatically via the `enabled: '{{ isEnvSet "MACOS_SIGN_P12" }}'` guard.

### Pre-release Versions

GoReleaser auto-detects pre-releases from the tag format:

- `v0.4.0-alpha.1` → marked as pre-release on GitHub
- `v0.4.0-beta.2` → marked as pre-release on GitHub
- `v0.4.0-rc.1` → marked as pre-release on GitHub
- `v0.4.0` → marked as latest release

This is controlled by `prerelease: auto` in `.goreleaser.yaml`. Pre-release tags must be created manually (the `/release` agent command only handles patch/minor/major).

### CI/CD Details

#### GitHub Actions Workflows

**Release** (`.github/workflows/release.yml`):

1. **Trigger**: Push of tags matching `v*.*.*`
2. **Setup**: Checks out code (full history), installs Go from `go.mod`
3. **Execute**: Runs GoReleaser v2
4. **Permissions**: `contents: write` to create releases and upload artifacts

**Site deploy** (`.github/workflows/site.yml`):

1. **Trigger**: Push to `main` when `site/**` or the workflow file itself changes
2. **Build**: Installs Hugo, runs `hugo --source site --minify`
3. **Deploy**: Uploads to GitHub Pages via `actions/deploy-pages`
4. **Concurrency**: Only one site deploy runs at a time (`cancel-in-progress: true`)
5. **Permissions**: `contents: read`, `pages: write`, and `id-token: write`

### Website (calcmark.org)

The documentation site lives in `site/` and is built with [Hugo](https://gohugo.io/).

#### Local Development

```bash
# Start the dev server with live reload
task site

# Build the site without serving (output in site/public/)
task site:build
```

#### How Deployment Works

The site workflow deploys to GitHub Pages when files in `site/**` change on push to `main`. The site deployment is independent of the release workflow — updating docs does not require a new version tag, and tagging a release does not redeploy the site unless `site/**` files were also changed.

If a release includes user-facing changes, update the site content before tagging:

1. Edit pages under `site/content/docs/`
2. Preview locally with `task site`
3. Commit and push to `main` (triggers site deploy)
4. Then tag and push the release

## Troubleshooting

### Tests fail before release

Fix tests before tagging. Always run:

```bash
task test
task quality
```

### Tag already exists locally

If you previously created a local tag but didn't push it (or aborted a release):

```bash
git tag -d v0.4.0           # delete the local tag
git tag -a "v0.4.0" -m "Release v0.4.0"  # recreate it
```

### Workflow failed in CI

1. Check the Actions tab: `https://github.com/CalcMark/go-calcmark/actions`
2. Click the failed workflow run for detailed logs
3. Common CI failure causes:
   - **Go version mismatch**: Workflow reads from `go.mod`, ensure it's current
   - **Permission denied**: Check repository Settings → Actions → Workflow permissions
   - **Homebrew tap push fails**: Verify `HOMEBREW_TAP_GITHUB_TOKEN` secret exists and has write access
   - **macOS signing fails**: Verify `MACOS_SIGN_P12` is valid base64, the `.p12` password is correct, and the certificate has not expired
   - **Notarization fails**: Verify the App Store Connect API key is active and the issuer ID / key ID are correct

To retry after fixing the underlying issue:

```bash
# 1. Delete the tag locally and remotely
git tag -d v0.4.0
git push origin :refs/tags/v0.4.0

# 2. Delete the GitHub release manually (if created)
#    Go to https://github.com/CalcMark/go-calcmark/releases and delete the draft

# 3. Fix the issue and commit

# 4. Re-tag and push
git tag -a "v0.4.0" -m "Release v0.4.0"
git push origin v0.4.0
```

### Need to redo a release

If the release was published but something was wrong:

```bash
# Delete the tag locally and remotely
git tag -d v0.4.0
git push origin :refs/tags/v0.4.0

# Delete the GitHub release manually (if created)
# Then fix the issue, commit, and re-tag
git tag -a "v0.4.0" -m "Release v0.4.0"
git push origin v0.4.0
```

### Site deploy fails

1. Check the Actions tab: `https://github.com/CalcMark/go-calcmark/actions`
2. Common issues:
   - **Hugo version mismatch**: The workflow pins Hugo v0.156.0 — ensure `site/` content is compatible
   - **Pages not enabled**: Check repository Settings → Pages → Source is set to "GitHub Actions"
   - **CNAME missing**: `site/static/CNAME` must contain `calcmark.org`

### Site didn't update after push

The site workflow only triggers when `site/**` files change. If you only changed Go source code, the site will not redeploy. To force a redeploy, make a trivial change to a file under `site/` or re-run the workflow manually from the Actions tab.
