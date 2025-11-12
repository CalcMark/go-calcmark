# Release Process

This document describes how to create and publish releases for go-calcmark.

## Overview

Releases are driven by the version number in `version.go`. The release process ensures:

1. **Single source of truth**: Version is defined only in `version.go`
2. **Reproducible builds**: Same process locally and in CI (via `release.sh`)
3. **Versioned artifacts**: WASM files include version in filename (e.g., `calcmark-0.1.1.wasm`)
4. **Git tag validation**: Tag must match version.go
5. **CI-first**: GitHub Actions automatically builds and publishes when you push a tag

## Two Ways to Release

| Method | When to Use |
|--------|-------------|
| **GitHub Actions** (recommended) | Normal releases - push tag and CI does everything |
| **Local Script** | Testing, debugging, or when CI is unavailable |

Both methods run the same `release.sh` script, so the process is identical.

## Quick Start

```bash
# 1. Update version in version.go
#    const Version = "0.2.0"

# 2. Commit the version change
git add version.go
git commit -m "Bump version to 0.2.0"

# 3. Push the commit to GitHub
git push origin main  # or your branch name

# 4. Create annotated tag
git tag -a "v0.2.0" -m "Release v0.2.0"

# 5. Push tag to GitHub (triggers automated release)
git push origin v0.2.0
```

The GitHub Action will automatically:
- Validate version matches tag
- Run all tests (excluding `impl/wasm`)
- Build CLI tools (`calcmark`, `cmspec`)
- Build WASM artifacts
- Create GitHub release with generated notes
- Attach `calcmark-{VERSION}.wasm` and `wasm_exec.js`

**Note**: The entire release process runs via `release.sh` in CI. The script detects CI mode automatically and runs non-interactively.

## Local Release (Alternative)

If you prefer to publish from your local machine instead of using GitHub Actions:

```bash
# After creating the tag locally (steps 1-4 above)

# Run release script (requires gh CLI: brew install gh)
./release.sh

# The script will:
# - Validate version matches tag
# - Run all tests (excluding impl/wasm)
# - Build CLI tools
# - Build WASM artifacts
# - Push tag to GitHub (if not already pushed)
# - Create release with artifacts
```

**Requirements for local release**:
- `gh` CLI installed and authenticated (`gh auth login`)
- Git configured with push access to the repository
- Clean working tree (no uncommitted changes)

### Testing Before Release

To test the full build process without publishing:

```bash
./release.sh --local
```

This runs all validation and builds artifacts locally but **does not**:
- Push tags to GitHub
- Create GitHub releases
- Upload artifacts

Use this to verify everything works before releasing:

```bash
# Test the build
./release.sh --local

# If successful, release for real
./release.sh
```

## Release Process Steps

### 1. Update Version

Edit `version.go`:

```go
// Version is the current version of the CalcMark implementation.
const Version = "0.2.0"
```

Commit the change:

```bash
git add version.go
git commit -m "Bump version to 0.2.0"
```

### 2. Push Commit to GitHub

Push your commit to GitHub (important: do this before creating the tag):

```bash
git push origin main  # or your branch name
```

**Why push first?** Git tags point to commits. If you push a tag that points to a commit that doesn't exist on GitHub yet, the CI won't be able to check it out.

### 3. Create Git Tag

Create an annotated tag that matches the version:

```bash
git tag -a "v0.2.0" -m "Release v0.2.0"
```

**Important**: The tag must be in the format `v{VERSION}` where `{VERSION}` exactly matches `version.go`.

Examples:
- ✅ `version.go = "0.2.0"` + tag `v0.2.0`
- ❌ `version.go = "0.2.0"` + tag `0.2.0` (missing `v` prefix)
- ❌ `version.go = "0.2.0"` + tag `v0.2.1` (mismatch)

### 4. Push Tag

Push the tag to GitHub:

```bash
git push origin v0.2.0
```

This triggers the GitHub Action which will:

1. **Validate** version matches tag
2. **Run tests** - all must pass
3. **Build CLI tools** (`calcmark`, `cmspec`)
4. **Build WASM** using `calcmark wasm`
5. **Create release** on GitHub
6. **Upload artifacts**:
   - `calcmark-{VERSION}.wasm`
   - `wasm_exec.js`

### 5. Verify Release

After the GitHub Action completes (usually 2-3 minutes):

1. Visit: `https://github.com/CalcMark/go-calcmark/releases`
2. Verify the release exists with correct version
3. Download and test WASM artifacts

## Release Artifacts

Each release includes:

### WASM Module (`calcmark-{VERSION}.wasm`)

The compiled CalcMark library for WebAssembly, including:
- Lexer (tokenization)
- Parser (AST generation)
- Validator (semantic validation)
- Evaluator (execution)
- Classifier (line type detection)

Filename includes version for cache busting and version pinning.

### JavaScript Glue Code (`wasm_exec.js`)

The Go WASM runtime bridge from the Go standard library. Required to load and execute the WASM module in JavaScript environments.

## Version Numbering

go-calcmark follows [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR** version: Incompatible API changes
- **MINOR** version: New functionality (backward compatible)
- **PATCH** version: Bug fixes (backward compatible)

### Pre-release Versions

For alpha, beta, or release candidates:

```go
const Version = "0.2.0-alpha.1"
const Version = "0.2.0-beta.2"
const Version = "0.2.0-rc.1"
```

Tag format: `v0.2.0-alpha.1`

## Troubleshooting

### Tag doesn't match version.go

**Error**: `Tag version (0.2.1) doesn't match version.go (0.2.0)`

**Solution**: Update `version.go` to match the tag, or delete and recreate the tag:

```bash
git tag -d v0.2.1
git tag -a "v0.2.0" -m "Release v0.2.0"
```

### Tests fail in CI

**Error**: `Error: Tests failed. Fix tests before releasing.`

**Solution**:
1. Run tests locally: `go test $(go list ./... | grep -v '/impl/wasm$')`
2. Fix failing tests
3. Commit fixes
4. Delete tag: `git tag -d v0.2.0`
5. Recreate tag on fixed commit

**Note**: Tests exclude `impl/wasm` package which requires `GOOS=js GOARCH=wasm` build constraints.

### WASM build fails

**Error**: `Error: WASM artifacts not created`

**Solution**:
1. Test locally: `./release.sh --local`
2. Check build errors in `impl/wasm/`
3. Fix build issues
4. Retry release

### GitHub CLI not found (local release)

**Error**: `Error: 'gh' CLI not found`

**Solution**: Install GitHub CLI:

```bash
brew install gh
gh auth login
```

Or use `--local` mode and manually create the release.

### GitHub Action fails

**Error**: Workflow run fails in GitHub Actions

**Solution**:
1. Check the Actions tab: `https://github.com/CalcMark/go-calcmark/actions`
2. Click the failed workflow run to see detailed logs
3. Common issues:
   - **Tests fail**: Same as "Tests fail in CI" above
   - **WASM build fails**: Check Go version matches (1.21+)
   - **Release already exists**: Workflow auto-deletes and recreates
   - **Permission denied**: Check repository settings → Actions → Workflow permissions

**To retry after fixing**:
```bash
# Delete the failed tag locally and remotely
git tag -d v0.2.0
git push origin :refs/tags/v0.2.0

# Fix the issue, commit
git add .
git commit -m "Fix release issue"
git push origin main

# Recreate and push tag
git tag -a "v0.2.0" -m "Release v0.2.0"
git push origin v0.2.0
```

### Simulating CI behavior locally

To test exactly what will run in CI:

```bash
# Simulate CI mode (non-interactive, auto-delete existing releases)
CI=true ./release.sh --local

# Or test the full flow with GitHub publish (requires gh CLI + auth)
CI=true GITHUB_TOKEN=$(gh auth token) ./release.sh
```

This helps debug issues before pushing tags.

## Release Checklist

Before pushing a tag:

- [ ] All tests pass locally: `go test $(go list ./... | grep -v '/impl/wasm$')`
- [ ] Version updated in `version.go`
- [ ] Version change committed
- [ ] Tag created: `git tag -a "vX.Y.Z" -m "Release vX.Y.Z"`
- [ ] Tag matches version.go exactly
- [ ] WASM builds locally: `./release.sh --local`
- [ ] CLAUDE.md and docs updated (if needed)
- [ ] Breaking changes documented (for major versions)

Then push commit and tag:

```bash
git push origin main      # Push the commit first
git push origin vX.Y.Z    # Then push the tag
```

## CI/CD Details

### How GitHub Actions Works

The release workflow (`.github/workflows/release.yml`) is minimal by design:

1. **Trigger**: Push of tags matching `v*.*.*` pattern (e.g., `v0.1.0`, `v1.2.3`, `v2.0.0-beta.1`)
2. **Setup**: Checks out code, installs Go 1.21, installs `gh` CLI
3. **Execute**: Runs `release.sh` in CI mode

The script automatically detects it's running in CI (`CI=true`) and:
- Runs non-interactively (no user prompts)
- Uses `GITHUB_TOKEN` for authentication
- Skips tag push (tag is already on remote)
- Auto-deletes existing release if retrying

**Benefits of this approach**:
- ✅ Same logic runs locally and in CI (no drift)
- ✅ Easy to debug (test exact CI behavior with `CI=true ./release.sh`)
- ✅ Single source of truth (`release.sh`)
- ✅ Minimal GitHub Action YAML

### Environment Variables

| Variable | Set By | Purpose |
|----------|--------|---------|
| `CI` | GitHub Action | Enables non-interactive mode |
| `GITHUB_TOKEN` | GitHub Actions | Authenticates `gh` CLI |
| `GITHUB_ACTIONS` | GitHub Actions | Additional CI detection |

### Permissions

The workflow requires `contents: write` to create releases and upload artifacts.

## Manual Release Creation

If automation fails, you can manually create a release:

1. Build artifacts locally:
   ```bash
   ./release.sh --local
   ```

2. Go to: `https://github.com/CalcMark/go-calcmark/releases/new`

3. Select the tag (e.g., `v0.2.0`)

4. Upload files from `release-artifacts/`:
   - `calcmark-{VERSION}.wasm`
   - `wasm_exec.js`

5. Generate release notes automatically or write custom notes

6. Publish release

## Post-Release

After a successful release:

1. **Update downstream projects** that consume go-calcmark:
   - CalcMark Server (github.com/CalcMark/server)
   - CalcMark Web (github.com/CalcMark/calcmark)

2. **Announce release** (if appropriate):
   - Update documentation sites
   - Notify community/users
   - Update integration guides

3. **Prepare next version**:
   - Optionally bump version.go to next -dev version
   - Example: `const Version = "0.2.1-dev"`
   - This makes it clear HEAD is not a release

## Questions?

For issues with the release process:
- Check GitHub Actions logs: `https://github.com/CalcMark/go-calcmark/actions`
- Test locally with `./release.sh --local`
- Review this document for common issues
