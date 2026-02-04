# Phase 7: Distribution - Research

**Researched:** 2026-02-04
**Domain:** Go binary distribution, GoReleaser, Homebrew taps
**Confidence:** HIGH

## Summary

This phase involves packaging and distributing CalcMark binaries across macOS, Linux, and Windows platforms. The standard approach in the Go ecosystem is using GoReleaser, which handles cross-compilation, archive creation, checksum generation, and publishing to GitHub Releases and Homebrew taps in a single automated workflow.

The current project has a custom `release.sh` script focused on WASM distribution. This will be replaced with GoReleaser configuration, which is the industry standard for Go CLI tool distribution. GoReleaser integrates directly with GitHub Actions and can automatically generate Homebrew formulas for tap repositories.

The phase also requires removing the existing WASM infrastructure (as per CONTEXT.md decisions), updating the release workflow, and creating a Homebrew tap repository for macOS/Linux users.

**Primary recommendation:** Use GoReleaser v2.x with GitHub Actions for automated multi-platform builds, checksums, and Homebrew formula generation.

## Standard Stack

The established tools for Go binary distribution:

### Core
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| GoReleaser | v2.x | Build, package, and release Go binaries | De facto standard for Go CLI distribution; used by Hugo, Terraform, goreleaser itself |
| GitHub Actions | N/A | CI/CD automation | Native integration with GitHub Releases |
| Homebrew | N/A | macOS/Linux package manager | Primary installation method for CLI tools on macOS |

### Supporting
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| gh CLI | latest | GitHub operations | Creating tap repository, testing releases |
| goreleaser-action | v6 | GitHub Action for GoReleaser | Automated release workflow |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| GoReleaser | Manual scripts | Manual scripts already exist but are WASM-focused; GoReleaser is more maintainable |
| GoReleaser | ko | ko is for container images, not CLI binaries |
| Homebrew tap | homebrew-core | homebrew-core has strict review process and delays; tap is immediate |

**Installation (development only):**
```bash
# GoReleaser (for local testing)
brew install goreleaser

# Or via go install
go install github.com/goreleaser/goreleaser/v2@latest
```

## Architecture Patterns

### Recommended Project Structure
```
/
├── .goreleaser.yaml       # GoReleaser configuration
├── .github/
│   └── workflows/
│       └── release.yml    # Updated GitHub Actions workflow
├── cmd/calcmark/
│   └── main.go            # Entry point with version variables
└── version.go             # Version constant (existing)
```

### Pattern 1: Version Injection via ldflags

**What:** Inject version, commit, and build date at compile time using Go's ldflags mechanism.

**When to use:** Always for release builds.

**Current state:** The project already uses ldflags in Taskfile.yml. GoReleaser has its own defaults that need to align with the existing `cmd/calcmark/main.go` variables.

**Example:**
```yaml
# .goreleaser.yaml
# Source: https://goreleaser.com/cookbooks/using-main.version/
builds:
  - main: ./cmd/calcmark
    binary: cm
    ldflags:
      - -s -w
      - -X main.Version={{.Version}}
      - -X main.BuildTime={{.Date}}
    env:
      - CGO_ENABLED=0
```

The existing `cmd/calcmark/main.go` already has:
```go
var (
    Version   = "dev"
    BuildTime = "unknown"
)
```

GoReleaser's `{{.Version}}` provides the Git tag without the `v` prefix (e.g., `v1.0.0` becomes `1.0.0`).

### Pattern 2: Platform-Specific Archive Formats

**What:** Use tar.gz for Unix-like systems, zip for Windows.

**When to use:** Standard convention that users expect.

**Example:**
```yaml
# .goreleaser.yaml
# Source: https://goreleaser.com/customization/archive/
archives:
  - format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    name_template: >-
      {{ .ProjectName }}_
      {{- .Version }}_
      {{- .Os }}_
      {{- .Arch }}
      {{- if .Arm }}v{{ .Arm }}{{ end }}
    files:
      - none*  # Binary only, per CONTEXT.md
```

### Pattern 3: Homebrew Tap with Formula Generation

**What:** GoReleaser automatically generates and pushes Homebrew formulas to a tap repository.

**When to use:** When setting up `brew install` capability.

**Example:**
```yaml
# .goreleaser.yaml
# Source: https://goreleaser.com/customization/homebrew/, https://bindplane.com/blog/creating-homebrew-formulas-with-goreleaser
brews:
  - name: calcmark
    repository:
      owner: calcmark
      name: homebrew-tap
      branch: main
    folder: Formula
    homepage: "https://github.com/CalcMark/go-calcmark"
    description: "CalcMark - calculations in markdown"
    license: "MIT"  # Verify actual license
    install: |
      bin.install "cm"
    test: |
      system "#{bin}/cm", "version"
```

### Anti-Patterns to Avoid

- **Hand-rolling release scripts:** The existing `release.sh` is WASM-focused and will be replaced. Don't extend it.
- **Mixing version sources:** Don't have both `version.go` constants AND ldflags injection. Choose one. Recommendation: Use ldflags (GoReleaser default) and remove or deprecate `version.go`.
- **Building on multiple runners:** Don't use matrix strategy to build on darwin/linux/windows runners. GoReleaser cross-compiles from a single ubuntu-latest runner.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cross-compilation | Shell script with GOOS/GOARCH loops | GoReleaser builds section | Handles all edge cases (GOARM, file extensions, etc.) |
| Archive creation | tar/zip shell commands | GoReleaser archives section | Consistent naming, format selection |
| Checksums | sha256sum commands | GoReleaser checksum section | Standard format, all algorithms supported |
| Homebrew formula | Manual Ruby file | GoReleaser brews section | Auto-generated, auto-pushed to tap |
| Release notes | Manual writing | GoReleaser changelog + GitHub auto-notes | Consistent, based on commits |

**Key insight:** GoReleaser exists specifically because release automation is deceptively complex. Every edge case (Windows .exe extension, ARM version suffixes, checksum formatting) has been solved.

## Common Pitfalls

### Pitfall 1: Version Mismatch Between Tag and Binary

**What goes wrong:** Binary reports different version than git tag.

**Why it happens:** ldflags not configured correctly, or using `version.go` constant instead of injected value.

**How to avoid:**
1. Ensure ldflags in `.goreleaser.yaml` matches the variable names in `main.go`
2. Test with `goreleaser build --snapshot --clean` before tagging
3. Verify with `./dist/cm_linux_amd64_v1/cm version`

**Warning signs:** `cm version` returns "dev" or "unknown" after release.

### Pitfall 2: Missing fetch-depth in GitHub Actions

**What goes wrong:** GoReleaser fails with "git tag not found" or changelog is empty.

**Why it happens:** GitHub Actions shallow clone doesn't include tags.

**How to avoid:**
```yaml
# Source: https://goreleaser.com/ci/actions/
- uses: actions/checkout@v4
  with:
    fetch-depth: 0  # Required for GoReleaser
```

**Warning signs:** Release notes say "No commits since last release" or tag-based logic fails.

### Pitfall 3: Homebrew Tap Token Permissions

**What goes wrong:** GoReleaser can't push formula to tap repository.

**Why it happens:** GITHUB_TOKEN only has write access to current repo, not the tap repo.

**How to avoid:**
1. Create a Personal Access Token (PAT) with `repo` scope
2. Add as repository secret: `HOMEBREW_TAP_GITHUB_TOKEN`
3. Reference in workflow:
```yaml
env:
  GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```
4. In `.goreleaser.yaml`:
```yaml
brews:
  - repository:
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
```

**Warning signs:** "403 Forbidden" or "push rejected" errors in release logs.

### Pitfall 4: GOARM Defaults

**What goes wrong:** ARM 32-bit binary doesn't run on target device.

**Why it happens:** GOARM=6 is default, but some devices need GOARM=7.

**How to avoid:** Per CONTEXT.md, build for `arm` with default GOARM=6. This covers Raspberry Pi 1/Zero and all newer devices.

**Warning signs:** "Illegal instruction" on ARM devices.

### Pitfall 5: Windows SmartScreen Without Signing

**What goes wrong:** Users see scary "Windows protected your PC" warning.

**Why it happens:** Per CONTEXT.md, we're not doing code signing.

**How to avoid:** This is expected behavior. Document in README that users may need to click "More info" then "Run anyway".

**Warning signs:** User reports (expected, not a bug).

## Code Examples

Verified patterns from official sources:

### Complete .goreleaser.yaml

```yaml
# Source: https://goreleaser.com/quick-start/, https://goreleaser.com/customization/builds/go/
version: 2

project_name: calcmark

before:
  hooks:
    - go mod tidy
    - go generate ./...

builds:
  - id: cm
    main: ./cmd/calcmark
    binary: cm
    env:
      - CGO_ENABLED=0
    goos:
      - darwin
      - linux
      - windows
    goarch:
      - amd64
      - arm64
      - arm  # Linux 32-bit ARM per CONTEXT.md
    goarm:
      - "6"  # Default, covers RPi 1/Zero and newer
    ignore:
      - goos: darwin
        goarch: arm  # No macOS 32-bit ARM
      - goos: windows
        goarch: arm  # No Windows 32-bit ARM (only arm64 per CONTEXT.md)
    ldflags:
      - -s -w
      - -X main.Version={{.Version}}
      - -X main.BuildTime={{.Date}}

archives:
  - id: default
    format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    name_template: >-
      {{ .ProjectName }}_
      {{- .Version }}_
      {{- .Os }}_
      {{- .Arch }}
      {{- if .Arm }}v{{ .Arm }}{{ end }}
    files:
      - none*  # Binary only per CONTEXT.md

checksum:
  name_template: 'checksums.txt'
  algorithm: sha256

changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^chore:'
      - Merge pull request
      - Merge branch

brews:
  - name: calcmark
    repository:
      owner: calcmark
      name: homebrew-tap
      branch: main
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    folder: Formula
    homepage: "https://github.com/CalcMark/go-calcmark"
    description: "CalcMark - calculations embedded in markdown"
    license: "MIT"
    install: |
      bin.install "cm"
    test: |
      system "#{bin}/cm", "version"

release:
  github:
    owner: CalcMark
    name: go-calcmark
  draft: false
  prerelease: auto
```

### Complete GitHub Actions Workflow

```yaml
# .github/workflows/release.yml
# Source: https://goreleaser.com/ci/actions/
name: Release

on:
  push:
    tags:
      - 'v*.*.*'

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Required for GoReleaser changelog

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```

### Homebrew Tap Repository Structure

```
calcmark/homebrew-tap/
├── Formula/
│   └── calcmark.rb  # Auto-generated by GoReleaser
└── README.md
```

The `calcmark.rb` formula is auto-generated. Example of what GoReleaser creates:

```ruby
# Source: Generated by GoReleaser, example from https://bindplane.com/blog/creating-homebrew-formulas-with-goreleaser
class Calcmark < Formula
  desc "CalcMark - calculations embedded in markdown"
  homepage "https://github.com/CalcMark/go-calcmark"
  version "1.0.0"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/CalcMark/go-calcmark/releases/download/v1.0.0/calcmark_1.0.0_darwin_amd64.tar.gz"
      sha256 "abc123..."
    end
    on_arm do
      url "https://github.com/CalcMark/go-calcmark/releases/download/v1.0.0/calcmark_1.0.0_darwin_arm64.tar.gz"
      sha256 "def456..."
    end
  end

  on_linux do
    # Similar blocks for linux amd64, arm64, arm
  end

  def install
    bin.install "cm"
  end

  test do
    system "#{bin}/cm", "version"
  end
end
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual shell scripts | GoReleaser | ~2018 | Standard for Go projects |
| GoReleaser v1 | GoReleaser v2 | 2024 | New config format, better defaults |
| homebrew_formulas | homebrew_casks/brews | 2025 | Formulas deprecated, use brews section |
| Single checksum file | Per-file or single | Stable | Both supported, single is simpler |

**Deprecated/outdated:**
- `homebrew_formulas` section in GoReleaser: Deprecated, use `brews` instead
- Custom `release.sh` scripts: Replace with GoReleaser for consistency
- Manual Homebrew formula maintenance: GoReleaser auto-generates and pushes

## Claude's Discretion Recommendations

Per CONTEXT.md, these items are at Claude's discretion:

### Version Tag Format
**Recommendation:** Use `v` prefix (e.g., `v1.0.0`)

**Rationale:**
- Go modules require the `v` prefix for version tags
- GoReleaser expects `v` prefix by default
- All Go ecosystem tools (go get, go install) expect this format
- Source: [Go Module Version Numbering](https://go.dev/doc/modules/version-numbers)

### Archive Format Per Platform
**Recommendation:** tar.gz for macOS/Linux, zip for Windows

**Rationale:**
- tar.gz is native to Unix systems
- zip is native to Windows
- GoReleaser's `format_overrides` handles this automatically
- Source: [GoReleaser Archives](https://goreleaser.com/customization/archive/)

### Checksum Format
**Recommendation:** Single `checksums.txt` file with SHA256

**Rationale:**
- SHA256 is industry standard (secure, fast)
- Single file is easier for users to verify
- Matches what most Go projects do (e.g., Terraform, Hugo)
- Source: [GoReleaser Checksums](https://goreleaser.com/customization/checksum/)

### GPG Signatures
**Recommendation:** Skip GPG signatures for now

**Rationale:**
- Adds complexity (key management, passphrase in CI)
- SHA256 checksums provide integrity verification
- Most users don't verify GPG signatures anyway
- Can be added later if needed (GoReleaser supports it)
- CONTEXT.md mentions "keep it simple"

## Open Questions

Things that couldn't be fully resolved:

1. **Tap Repository Ownership**
   - What we know: CONTEXT.md says `calcmark/tap`
   - What's unclear: Is there an existing CalcMark GitHub organization? The go.mod shows `github.com/CalcMark/go-calcmark` (capital C)
   - Recommendation: Create `CalcMark/homebrew-tap` to match existing naming

2. **License for Homebrew Formula**
   - What we know: Formula needs a license field
   - What's unclear: What is the project's actual license? Need to check for LICENSE file
   - Recommendation: Verify and use correct license identifier

3. **Existing version.go Usage**
   - What we know: `version.go` defines `const Version = "0.2.0"`
   - What's unclear: Is this used elsewhere in the codebase? Should it be kept?
   - Recommendation: Search codebase for imports of this constant before deciding

4. **Release Artifact Cleanup**
   - What we know: Existing `release.sh` and WASM artifacts need removal
   - What's unclear: Are there any downstream dependencies on current release format?
   - Recommendation: Document breaking changes in release notes

## Sources

### Primary (HIGH confidence)
- [GoReleaser Quick Start](https://goreleaser.com/quick-start/) - Setup and initialization
- [GoReleaser Go Builds](https://goreleaser.com/customization/builds/go/) - Build configuration
- [GoReleaser Archives](https://goreleaser.com/customization/archive/) - Archive formats
- [GoReleaser Checksums](https://goreleaser.com/customization/checksum/) - Checksum generation
- [GoReleaser GitHub Actions](https://goreleaser.com/ci/actions/) - CI workflow
- [GoReleaser Templates](https://goreleaser.com/customization/templates/) - Template variables
- [Go Module Version Numbering](https://go.dev/doc/modules/version-numbers) - Version tag format

### Secondary (MEDIUM confidence)
- [Homebrew Tap Creation](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap) - Tap setup
- [BindPlane GoReleaser Homebrew](https://bindplane.com/blog/creating-homebrew-formulas-with-goreleaser) - Complete brews example
- [Applied Go GoReleaser](https://appliedgo.net/release2/) - Practical example

### Tertiary (LOW confidence)
- WebSearch results for GOARM configuration - verified with GoReleaser docs
- WebSearch results for Windows ARM64 - verified with go.dev docs

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - GoReleaser is the clear industry standard for Go CLI distribution
- Architecture: HIGH - Patterns are well-documented in official GoReleaser docs
- Pitfalls: HIGH - Based on official troubleshooting docs and common issues

**Research date:** 2026-02-04
**Valid until:** 2026-05-04 (90 days - GoReleaser is stable, major changes rare)
