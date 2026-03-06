---
title: "macOS Gatekeeper quarantine flag on Homebrew-installed cm binary"
date: 2026-03-05
updated: 2026-03-06
category: build-errors
tags: [macos, homebrew, gatekeeper, goreleaser]
component: release-pipeline
severity: low
symptoms:
  - "macOS Gatekeeper blocks cm execution with unidentified developer warning when downloading binary directly"
root_cause: "Release binaries are not code-signed — notarization is overkill for open source CLI tools"
resolution: "Dropped notarization. Added post-install hook to Homebrew Cask that strips quarantine flag automatically via xattr."
---

# macOS Gatekeeper Quarantine on `cm` Binary

## Problem

macOS users installing the `cm` CLI binary via Homebrew Cask had to manually run `xattr -d com.apple.quarantine $(which cm)` after every update. Attempted to solve with Apple notarization, but the notarization process was unreliable (8+ hour hangs via `notarytool submit --wait`).

## Investigation

1. Explored `rcodesign`, custom signing scripts, and GoReleaser's built-in `notarize.macos` support.
2. GoReleaser notarization (via `anchore/quill`) was configured but Apple's notary service proved unreliable — `notarytool submit --wait` hung for 8+ hours.
3. Surveyed how major open source Go CLI tools handle this (Hugo, lazygit, fzf, charm tools, goreleaser itself): **none of them notarize**.
4. GoReleaser v2.10+ recommends `homebrew_casks` for pre-compiled binaries (both CLI and GUI). The `hooks.post.install` block can strip the quarantine flag automatically.

## Root Cause

Apple notarization is designed for commercially distributed macOS software. For open source CLI tools, it adds significant complexity (Apple Developer Program membership, certificate management, unreliable API) with minimal benefit. The quarantine flag can be stripped automatically by the Homebrew Cask post-install hook.

## Solution

1. **Removed the `notarize` block** from `.goreleaser.yaml`
2. **Added `hooks.post.install`** to the `homebrew_casks` config to strip quarantine automatically:
   ```yaml
   hooks:
     post:
       install: |
         if OS.mac?
           system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{staged_path}/cm"]
         end
   ```
3. **Removed macOS signing secrets** from the release workflow
4. **Deleted `notarize.sh`**
5. **Removed Gatekeeper workaround notes** from README and site docs — the hook handles it

For users who download the binary directly (not via Homebrew), the standard macOS workaround applies: right-click and Open, or `xattr -d com.apple.quarantine ./cm`. This is the same experience as every other open source CLI tool.

## Prevention

### Don't notarize open source CLI tools

Apple notarization is for commercial macOS app distribution. Open source CLI tools distributed via Homebrew should use the post-install hook to strip quarantine instead. This is what GoReleaser itself does.

## Related

- [GoReleaser Homebrew Formulas docs](https://goreleaser.com/customization/homebrew/)
- Phase 07 distribution context: `.planning/phases/07-distribution/07-CONTEXT.md`
