---
name: release
description: Prepare and execute a semver release with full quality gates
argument-hint: "<patch|minor|major>"
allowed-tools:
  - Read
  - Bash
  - Grep
  - Glob
  - AskUserQuestion
---

Prepare and publish a CalcMark release. This command enforces quality gates and produces a changelog before tagging.

## Step 1: Determine release type and next version

Run this to find the latest tag:

```bash
git tag --sort=-version:refname | head -1
```

Parse the latest tag into MAJOR.MINOR.PATCH components. Based on the argument (`$ARGUMENTS`), compute the next version:

- `patch` → increment PATCH
- `minor` → increment MINOR, reset PATCH to 0
- `major` → increment MAJOR, reset MINOR and PATCH to 0

If `$ARGUMENTS` is empty, ask the user:

```
AskUserQuestion(
  header: "Release type",
  question: "What kind of release is this?",
  options: [
    { label: "minor", description: "New features, backward compatible" },
    { label: "patch", description: "Bug fixes only" },
    { label: "major", description: "Breaking changes" }
  ]
)
```

Print the computed version (e.g., "Next release: v0.4.0") and continue.

## Step 2: Ensure working tree is clean

```bash
git status --porcelain
```

If there are uncommitted changes, **stop** and tell the user:

> "Working tree is not clean. Commit or stash all changes before releasing."

Do NOT proceed until the working tree is clean.

## Step 3: Ensure we are on main

```bash
git branch --show-current
```

If not on `main`, **warn** the user and ask for confirmation before continuing. Releases from non-main branches are unusual and should be intentional.

## Step 4: Run full test suite

```bash
task test
```

If **any** test fails, **stop immediately**. Print the failure summary and tell the user:

> "Tests must pass before releasing. Fix the failures above and run `/release` again."

Do NOT proceed past a test failure. Do NOT offer to skip tests. Do NOT tag.

## Step 5: Run quality checks

```bash
task quality
```

Review the output. The `modernize` step may produce warnings for pre-existing issues — that is acceptable. But if `go fmt`, `go vet`, or `staticcheck` fail, **stop** and report the errors.

## Step 6: Generate changelog

Generate a human-readable changelog of what's new since the last tag. Run:

```bash
git log $(git tag --sort=-version:refname | head -1)..HEAD --oneline --no-merges
```

Present the commits to the user in a clean summary grouped by type:

- **Features**: commits starting with `feat`
- **Fixes**: commits starting with `fix`
- **Other**: everything else (refactor, docs, chore, test, perf, etc.)

Then ask the user:

```
AskUserQuestion(
  header: "Changelog",
  question: "Does this changelog look correct? Should anything be added or removed before releasing?",
  options: [
    { label: "Looks good", description: "Proceed with the release" },
    { label: "Hold on", description: "I want to make changes first" }
  ]
)
```

If the user says "Hold on", **stop** and wait for further instructions.

If there are **zero commits** since the last tag, **stop** and tell the user:

> "No commits since the last release. There is nothing to release."

## Step 7: Tag and push

Only after all gates pass and the user approves the changelog:

```bash
git tag -a "vX.Y.Z" -m "Release vX.Y.Z"
```

Then ask the user for final confirmation before pushing:

```
AskUserQuestion(
  header: "Push tag",
  question: "Ready to push v{X.Y.Z}? This will trigger the GitHub Actions release workflow and publish to Homebrew.",
  options: [
    { label: "Push it", description: "Push the tag to origin and trigger the release" },
    { label: "Wait", description: "Keep the tag local for now" }
  ]
)
```

If approved:

```bash
git push origin vX.Y.Z
```

## Step 8: Post-release summary

After pushing, print:

```
Release v{X.Y.Z} tagged and pushed.

GitHub Actions is now building the release.
Run /release-status to check workflow progress, or monitor at:
  https://github.com/CalcMark/go-calcmark/actions

Once complete:
  - Release page: https://github.com/CalcMark/go-calcmark/releases/tag/vX.Y.Z
  - Homebrew: brew upgrade calcmark/tap/calcmark
```

## Hard Rules

- **Never skip tests.** If `task test` fails, the release stops. Period.
- **Never tag on a dirty working tree.** All changes must be committed first.
- **Never push without user confirmation.** The tag push is irreversible in practice.
- **Always show the changelog.** The user must see and approve what's being released.
- **Never amend or force-push tags.** If something went wrong, tell the user to delete the tag and start over per RELEASE.md troubleshooting.
