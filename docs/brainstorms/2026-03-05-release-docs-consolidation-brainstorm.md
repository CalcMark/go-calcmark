# Brainstorm: Consolidate Release Documentation

**Date:** 2026-03-05
**Status:** Ready for planning

## What We're Building

Consolidate four overlapping release docs into a single canonical process document (RELEASE.md) with streamlined agent commands that reference it. Eliminate RELEASE_CHECKLIST.md entirely.

## Why This Approach

The release process is currently described in four places with significant overlap:

- **RELEASE.md** (277 lines) — comprehensive but duplicates the agent commands
- **RELEASE_CHECKLIST.md** (74 lines) — almost entirely redundant with RELEASE.md
- **`.claude/commands/release.md`** — duplicates RELEASE.md's quick start and checklist
- **`.claude/commands/release-status.md`** — duplicates post-release verification and troubleshooting

When something changes (e.g., a new quality gate, a new signing step), it needs updating in multiple places. The agent commands and RELEASE.md already contradict in minor ways (e.g., RELEASE.md uses lightweight tags in the quick start example, the agent command uses annotated tags).

## Key Decisions

1. **RELEASE.md is the single source of truth.** It gets restructured around the release lifecycle and absorbs RELEASE_CHECKLIST.md's unique content (Homebrew tap setup).

2. **Delete RELEASE_CHECKLIST.md.** Its Homebrew tap setup instructions move into RELEASE.md under a "First-Time Setup" section. Everything else is already covered.

3. **Agent commands keep inline essentials only.** They retain the bash commands, user interaction flow, and error handling they need to execute — but drop all explanatory prose. Add "See RELEASE.md" references where context would be helpful (e.g., troubleshooting, signing details).

4. **RELEASE.md restructured around the lifecycle.** New section order:
   - **Overview** — what happens automatically (push to main, push a tag)
   - **First-Time Setup** — Homebrew tap, macOS signing secrets (merged from RELEASE_CHECKLIST.md)
   - **Pre-flight** — tests, quality, clean tree, main branch
   - **Tag & Push** — version computation, annotated tag, push
   - **Post-Release Verification** — GitHub release, Homebrew, site
   - **Reference** — artifacts table, CI/CD details, versioning scheme, pre-releases, dry run, website/docs deployment
   - **Troubleshooting** — redo a release, common CI failures, site deploy issues

## Scope of Changes

### RELEASE.md
- Restructure into lifecycle order (see section 4 above)
- Merge Homebrew tap setup from RELEASE_CHECKLIST.md into "First-Time Setup"
- Keep all existing reference material (artifacts, signing, CI, secrets table)
- Ensure annotated tags are used consistently (not lightweight)

### RELEASE_CHECKLIST.md
- Delete entirely

### .claude/commands/release.md
- Strip explanatory prose from each step
- Keep: bash commands, AskUserQuestion interactions, conditional logic, Hard Rules
- Add: "See RELEASE.md" references for troubleshooting, signing, and process details
- No functional changes to the release flow itself

### .claude/commands/release-status.md
- Strip the retry/redo instructions from Step 6 — replace with "See RELEASE.md > Troubleshooting"
- Otherwise minimal changes (it's already lean and focused)

## What's NOT Changing

- The actual release process (test, quality, tag, push, verify)
- GoReleaser config, GitHub Actions workflows, Taskfile
- The commands' allowed-tools or interaction model
- Any CI/CD infrastructure

## Open Questions

None — all key decisions resolved through discussion.
