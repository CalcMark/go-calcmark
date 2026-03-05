---
title: "macOS Gatekeeper quarantine flag on Homebrew-installed cm binary"
date: 2026-03-05
category: build-errors
tags: [macos, code-signing, notarization, goreleaser, homebrew, gatekeeper, ci]
component: release-pipeline
severity: medium
symptoms:
  - "Users must run xattr -d com.apple.quarantine $(which cm) after every Homebrew install or upgrade"
  - "macOS Gatekeeper blocks cm execution with unidentified developer warning"
root_cause: "Release binaries were not code-signed with a Developer ID certificate or notarized with Apple"
resolution: "Use GoReleaser v2 built-in notarize.macos config block with Developer ID certificate and App Store Connect API key secrets"
---

# macOS Gatekeeper Quarantine on Homebrew-installed `cm` Binary

## Problem

macOS users installing the `cm` CLI binary via Homebrew Cask had to manually run `xattr -d com.apple.quarantine $(which cm)` after every update. macOS Gatekeeper quarantines unsigned binaries downloaded from the internet, making the post-install workaround a recurring friction point for every release.

## Investigation

1. Explored `rcodesign` (from `indygreg/apple-codesign`), a Rust-based cross-platform Apple signing tool that can run on Linux CI.
2. Built a custom `scripts/apple-codesign.sh` with subcommands (`decode-secrets`, `sign`, `notarize`, `cleanup`) and wired it into GoReleaser post-build hooks for darwin targets.
3. Added Rust toolchain installation and cargo caching to the GitHub Actions release workflow.
4. Added separate notarization and secret cleanup steps to the workflow.
5. Stepped back and recognized the approach was over-engineered — the project was carrying a custom shell script, a Rust toolchain dependency, and multiple new workflow steps for something that should be a solved problem.
6. Discovered GoReleaser v2 has built-in `notarize.macos` support powered by `anchore/quill`, which handles both signing and notarization cross-platform (works on Linux CI runners).
7. Also found `indygreg/apple-code-sign-action` as an alternative GitHub Action.
8. Chose the GoReleaser built-in approach as the simplest path — zero extra tooling, zero custom scripts, just configuration.

## Root Cause

The release pipeline produced unsigned, un-notarized darwin binaries. macOS Gatekeeper flags any unsigned binary downloaded from the internet with the `com.apple.quarantine` extended attribute, which blocks execution until the user manually clears it. The fix requires Apple code signing with a Developer ID Application certificate and notarization via App Store Connect.

## Solution

All that is needed is a `notarize` block in `.goreleaser.yaml` and five secrets passed as environment variables in the GitHub Actions workflow.

### `.goreleaser.yaml`

```yaml
notarize:
  macos:
    - enabled: '{{ isEnvSet "MACOS_SIGN_P12" }}'
      sign:
        certificate: "{{.Env.MACOS_SIGN_P12}}"
        password: "{{.Env.MACOS_SIGN_PASSWORD}}"
      notarize:
        issuer_id: "{{.Env.MACOS_NOTARY_ISSUER_ID}}"
        key_id: "{{.Env.MACOS_NOTARY_KEY_ID}}"
        key: "{{.Env.MACOS_NOTARY_KEY}}"
```

The `enabled: '{{ isEnvSet "MACOS_SIGN_P12" }}'` guard means local builds and forks without the secrets still work — signing is simply skipped.

### `.github/workflows/release.yml`

Pass env vars to the existing GoReleaser step:

```yaml
env:
  MACOS_SIGN_P12: ${{ secrets.MACOS_SIGN_P12 }}
  MACOS_SIGN_PASSWORD: ${{ secrets.MACOS_SIGN_PASSWORD }}
  MACOS_NOTARY_ISSUER_ID: ${{ secrets.MACOS_NOTARY_ISSUER_ID }}
  MACOS_NOTARY_KEY_ID: ${{ secrets.MACOS_NOTARY_KEY_ID }}
  MACOS_NOTARY_KEY: ${{ secrets.MACOS_NOTARY_KEY }}
```

### GitHub Actions secrets

| Secret | Value |
|--------|-------|
| `MACOS_SIGN_P12` | `base64 < DeveloperID.p12` |
| `MACOS_SIGN_PASSWORD` | .p12 certificate password |
| `MACOS_NOTARY_ISSUER_ID` | App Store Connect API issuer UUID |
| `MACOS_NOTARY_KEY_ID` | App Store Connect API key ID |
| `MACOS_NOTARY_KEY` | `base64 < AuthKey_XXXX.p8` |

No custom scripts, no Rust toolchain, no extra workflow steps. GoReleaser handles signing and notarization internally via `anchore/quill` before packaging the release artifacts.

## Prevention

### Check existing tooling before building custom solutions

Before writing any CI script, shell wrapper, or custom tooling, spend 15 minutes reading the current docs of the build tool you already use. GoReleaser, Fastlane, and similar tools add native features every major version. Search the tool's changelog and GitHub issues for your keywords before assuming the feature does not exist.

### Treat Gatekeeper compliance as a release gate

macOS Gatekeeper quarantine is not a cosmetic issue. Users cannot run the binary without manual intervention, which is a dealbreaker for Homebrew Cask distribution.

### Anti-pattern: custom CI scripts before checking native support

A 50-line shell script has more failure modes than 8 lines of declarative YAML interpreted by a well-tested tool. Custom rcodesign scripts, Rust toolchain installation in CI, and bespoke shell wrappers all need ongoing maintenance when CI runners, OS versions, or Apple's notarization API changes. The rule: before building custom CI infrastructure for a build/release concern, check the current version's documentation first.

## Follow-up

Once signing secrets are configured and the first signed release ships, remove the Gatekeeper workaround notes from:

- `README.md` (lines 19-20)
- `site/content/docs/getting-started.md` (lines 15-16)

## Related

- [GoReleaser notarize docs](https://goreleaser.com/customization/notarize/)
- [indygreg/apple-code-sign-action](https://github.com/indygreg/apple-code-sign-action) (alternative approach)
- [rcodesign notarizing docs](https://gregoryszorc.com/docs/apple-codesign/stable/apple_codesign_rcodesign_notarizing.html)
- Phase 07 distribution context: `.planning/phases/07-distribution/07-CONTEXT.md` (originally decided against signing)
