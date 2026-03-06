---
title: "refactor: Consolidate release documentation into single source of truth"
type: refactor
status: completed
date: 2026-03-05
brainstorm: docs/brainstorms/2026-03-05-release-docs-consolidation-brainstorm.md
---

# Consolidate Release Documentation

## Overview

Four files describe the release process with significant overlap and contradictions. Consolidate into one canonical doc (RELEASE.md) with streamlined agent commands that reference it.

**Files affected:**
- `RELEASE.md` — restructure around lifecycle, absorb RELEASE_CHECKLIST.md content
- `RELEASE_CHECKLIST.md` — delete
- `.claude/commands/release.md` — slim to essentials, add RELEASE.md references
- `.claude/commands/release-status.md` — replace inline retry instructions with reference

## Problem Statement

The release process is described in four places. When something changes, it needs updating in multiple files. The files already contradict each other:
- Tag commands: annotated (`git tag -a`) in RELEASE.md and `/release`, lightweight (`git tag`) in RELEASE_CHECKLIST.md
- Push commands: `git push origin vX.Y.Z` (safe) vs `git push --tags` (dangerous)
- Platform count: 6 archives in RELEASE.md artifact table vs "7 platforms" in RELEASE_CHECKLIST.md
- `brew upgrade` assumes the tap exists; `brew tap` step only in RELEASE_CHECKLIST.md

## Contradictions to Resolve First

Before writing any new content, resolve these discrepancies:

- [x] **Audit `.goreleaser.yaml`** to determine actual archive count (6 or 7). Update artifact table accordingly.
- [x] **Standardize on annotated tags** (`git tag -a`) everywhere. Add one-sentence rationale (GoReleaser derives changelog from tag).
- [x] **Standardize on explicit push** (`git push origin vX.Y.Z`), never `git push --tags`.
- [x] **Add `brew tap calcmark/tap`** before `brew upgrade` in post-release verification for fresh-machine compatibility.

## Phase 1: Restructure RELEASE.md

Rewrite RELEASE.md organized around the release lifecycle. New section order:

### 1.1 Overview (keep existing, minor edits)
What happens automatically: push to main deploys site, push a tag triggers release.

### 1.2 First-Time Setup (NEW — merged from RELEASE_CHECKLIST.md)
Numbered steps in explicit order:
1. **Homebrew tap repository** — create `CalcMark/homebrew-tap` (public, initialized with README). Note: "If this repository already exists, skip to step 2."
2. **Personal Access Token** — generate classic token (NOT fine-grained) with `repo` scope. Preserve the exact scope detail from RELEASE_CHECKLIST.md.
3. **Add `HOMEBREW_TAP_GITHUB_TOKEN` secret** to go-calcmark repo settings.
4. **macOS signing (optional)** — Developer ID Application certificate + App Store Connect API key. Explicitly state: "Signing is optional for your first release. Unsigned releases work but macOS users must clear quarantine manually."
5. **Verify secrets** — table of all secrets with purpose and required/optional status.

### 1.3 Pre-flight Checks
- Clean working tree (`git status --porcelain`)
- On `main` branch
- `task test` passes
- `task quality` passes
- Dry run: `goreleaser release --snapshot --clean` (moved from buried reference section)

### 1.4 Tag & Push
- Compute next semver version
- Create annotated tag: `git tag -a "vX.Y.Z" -m "Release vX.Y.Z"`
- Push single tag: `git push origin vX.Y.Z`
- What happens next (GoReleaser builds, signs, creates release, updates Homebrew)

### 1.5 Post-Release Verification
- Check GitHub releases page (verify all archives present)
- Test Homebrew: `brew tap calcmark/tap && brew install calcmark/tap/calcmark`
- Test binary download: download archive, extract, run `cm --version`
- Verify site at calcmark.org if docs were updated

### 1.6 Reference
Keep existing detailed sections, reorganized:
- How versioning works (ldflags, git describe)
- Release artifacts table (with corrected count)
- macOS code signing & notarization (detailed setup, signing secrets reference)
- Pre-release versions (auto-detection from tag format)
- CI/CD details (workflows, permissions)
- Website deployment (Hugo, site workflow)
- Version numbering (semver)

### 1.7 Troubleshooting
Keep existing entries, add:
- **Workflow failed in CI** (NEW) — distinct from "need to redo a release". Covers: inspect job logs via Actions tab, common CI failure causes, and the delete-and-retag recovery path. This is what `/release-status` will reference.
- **Tag already exists locally** (NEW) — delete local tag with `git tag -d`, then retry.
- **Tests fail before release** (existing)
- **GoReleaser build fails** (existing, expanded)
- **Site deploy fails** (existing)
- **Need to redo a release** (existing)

## Phase 2: Delete RELEASE_CHECKLIST.md

- [x] Verify all content from RELEASE_CHECKLIST.md is present in new RELEASE.md
- [x] `git rm RELEASE_CHECKLIST.md`
- [x] Grep codebase for references to RELEASE_CHECKLIST.md and update them

## Phase 3: Slim `.claude/commands/release.md`

Keep: frontmatter, step structure, bash commands, AskUserQuestion interactions, conditional gates, Hard Rules.

Remove: explanatory prose that duplicates RELEASE.md.

Specific changes:
- [x] **Step 1** — keep semver computation logic. Add note: "Pre-release tags (alpha/beta/rc) are not supported by this command. See RELEASE.md > Pre-release Versions."
- [x] **Step 2** — keep bash command and gate. Remove prose about why.
- [x] **Step 3** — keep bash command and gate. Remove prose.
- [x] **Step 4** — keep `task test` and hard stop. Remove prose.
- [x] **Step 5** — keep `task quality` and gate. Remove prose about modernize warnings.
- [x] **Step 6** — keep changelog generation and presentation. This is agent-specific logic, not duplicated in RELEASE.md.
- [x] **Step 7** — add `git tag -l "vX.Y.Z"` pre-check before `git tag -a`. If tag exists, tell user to delete it per RELEASE.md troubleshooting.
- [x] **Step 8** — update post-release summary to include `brew tap calcmark/tap` before `brew upgrade`. Add: "Run `/release-status` to check workflow progress."
- [x] **Hard Rules** — keep all. Update troubleshooting reference to specific RELEASE.md section anchor.

## Phase 4: Slim `.claude/commands/release-status.md`

Minimal changes — this file is already lean.

- [x] **Step 6** — replace inline retry instructions (lines 147–153) with: "See RELEASE.md > Troubleshooting > Workflow Failed in CI for recovery steps."
- [x] **Hard Rules** — keep all, unchanged.

## Phase 5: Cross-Reference Audit

- [x] Grep for `RELEASE_CHECKLIST` across all files — update or remove references
- [x] Grep for `RELEASE.md` section anchors — verify they match new structure
- [x] Check README.md for any release process references
- [x] Check CONTRIBUTING.md for any release process references
- [x] Check site/content/docs/ for any stale Gatekeeper workaround notes (per institutional learning from `docs/solutions/build-errors/macos-gatekeeper-goreleaser-notarize.md`)

## Acceptance Criteria

- [ ] RELEASE.md is the single source of truth for the release process
- [ ] RELEASE_CHECKLIST.md no longer exists
- [ ] Agent commands contain only executable essentials (bash, prompts, gates)
- [ ] No contradictions between RELEASE.md and agent commands (tag style, push style, platform count)
- [ ] All cross-references point to valid locations
- [ ] A new contributor can complete first-time setup by reading only RELEASE.md
- [ ] `/release` and `/release-status` commands still function correctly (manual verification)

## What's NOT Changing

- The actual release process (test, quality, tag, push, verify)
- GoReleaser config, GitHub Actions workflows, Taskfile
- The commands' allowed-tools, frontmatter, or interaction model
- Any CI/CD infrastructure

## References

- Brainstorm: `docs/brainstorms/2026-03-05-release-docs-consolidation-brainstorm.md`
- Institutional learning: `docs/solutions/build-errors/consolidate-taskfile-race-condition-orphaned-tasks.md` — "Stale documentation is worse than no documentation"
- Institutional learning: `docs/solutions/build-errors/macos-gatekeeper-goreleaser-notarize.md` — follow-up cleanup for stale Gatekeeper workaround notes
