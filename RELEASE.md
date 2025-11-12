# Release Process

This document describes how to create and publish releases for go-calcmark.

## Overview

Releases are driven by the version number in `version.go`. The release process ensures:

1. **Single source of truth**: Version is defined only in `version.go`
2. **Reproducible builds**: Same process locally and in CI
3. **Versioned artifacts**: WASM files include version in filename (e.g., `calcmark-0.1.1.wasm`)
4. **Git tag validation**: Tag must match version.go

## Quick Start

```bash
# 1. Update version in version.go
#    const Version = "0.2.0"

# 2. Commit the version change
git add version.go
git commit -m "Bump version to 0.2.0"

# 3. Create annotated tag
git tag -a "v0.2.0" -m "Release v0.2.0"

# 4. Push tag to GitHub (triggers automated release)
git push origin v0.2.0
```

The GitHub Action will automatically:
- Run tests
- Build WASM artifacts
- Create GitHub release
- Attach `calcmark-0.2.0.wasm` and `wasm_exec.js`

## Local Release (Alternative)

If you prefer to publish from your local machine:

```bash
# After creating the tag locally (steps 1-3 above)

# Run release script (requires gh CLI)
./release.sh

# The script will:
# - Validate version matches tag
# - Run tests
# - Build WASM artifacts
# - Push tag to GitHub
# - Create release with artifacts
```

### Local Build Only

To build artifacts without publishing:

```bash
./release.sh --local
```

This is useful for:
- Testing the build process
- Preparing artifacts for manual upload
- Verifying WASM builds locally

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

### 2. Create Git Tag

Create an annotated tag that matches the version:

```bash
git tag -a "v0.2.0" -m "Release v0.2.0"
```

**Important**: The tag must be in the format `v{VERSION}` where `{VERSION}` exactly matches `version.go`.

Examples:
- ✅ `version.go = "0.2.0"` + tag `v0.2.0`
- ❌ `version.go = "0.2.0"` + tag `0.2.0` (missing `v` prefix)
- ❌ `version.go = "0.2.0"` + tag `v0.2.1` (mismatch)

### 3. Push Tag

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

### 4. Verify Release

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
1. Run tests locally: `go test ./...`
2. Fix failing tests
3. Commit fixes
4. Delete tag: `git tag -d v0.2.0`
5. Recreate tag on fixed commit

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

## Release Checklist

Before pushing a tag:

- [ ] All tests pass locally: `go test ./...`
- [ ] Version updated in `version.go`
- [ ] Version change committed
- [ ] Tag created: `git tag -a "vX.Y.Z" -m "Release vX.Y.Z"`
- [ ] Tag matches version.go exactly
- [ ] WASM builds locally: `./release.sh --local`
- [ ] CLAUDE.md and docs updated (if needed)
- [ ] Breaking changes documented (for major versions)

Then push:

```bash
git push origin vX.Y.Z
```

## CI/CD Details

### GitHub Action Triggers

The release workflow (`.github/workflows/release.yml`) triggers on:

- Push of tags matching `v*.*.*` pattern
- Examples: `v0.1.0`, `v1.2.3`, `v2.0.0-beta.1`

### Permissions

The workflow requires:

- `contents: write` - to create releases and upload artifacts

### Go Version

The CI uses Go 1.21 (matching development requirements).

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
