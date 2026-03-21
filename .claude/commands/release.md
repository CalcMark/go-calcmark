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

Prepare and publish a CalcMark release. See RELEASE.md for the full process and troubleshooting.

Pre-release tags (alpha/beta/rc) are not supported by this command. See RELEASE.md > Pre-release Versions to tag those manually.

## Step 1: Determine release type and next version

```bash
git tag --sort=-version:refname | head -1
```

Parse the latest tag into MAJOR.MINOR.PATCH. Based on `$ARGUMENTS`, compute the next version:

- `patch` → increment PATCH
- `minor` → increment MINOR, reset PATCH to 0
- `major` → increment MAJOR, reset MINOR and PATCH to 0

If `$ARGUMENTS` is empty, ask:

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

Print: "Next release: v{X.Y.Z}"

## Step 2: Clean working tree

```bash
git status --porcelain
```

If not clean, **stop**: "Working tree is not clean. Commit or stash all changes before releasing."

## Step 3: On main branch

```bash
git branch --show-current
```

If not on `main`, **warn** and ask for confirmation.

## Step 4: Run tests

```bash
task test
```

If **any** test fails, **stop immediately**. Do NOT proceed. Do NOT offer to skip tests. Do NOT tag.

## Step 5: Run quality checks

```bash
task quality
```

If `go fmt`, `go vet`, or `staticcheck` fail, **stop** and report.

## Step 6: Generate changelog

```bash
git log $(git tag --sort=-version:refname | head -1)..HEAD --oneline --no-merges
```

If there are **zero commits** since the last tag, **stop**: "No commits since the last release. There is nothing to release."

Present commits grouped by type:

- **Features**: commits starting with `feat`
- **Fixes**: commits starting with `fix`
- **Other**: everything else

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

If "Hold on", **stop** and wait.

## Step 7: Tag and push

First check if the tag already exists locally:

```bash
git tag -l "vX.Y.Z"
```

If the tag exists, **stop**: "Tag vX.Y.Z already exists locally. Delete it with `git tag -d vX.Y.Z` and run `/release` again. See RELEASE.md > Troubleshooting for details."

Create the tag:

```bash
git tag -a "vX.Y.Z" -m "Release vX.Y.Z"
```

Ask for final confirmation:

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

```
Release v{X.Y.Z} tagged and pushed.

GitHub Actions is now building the release.
Run /release-status to check workflow progress, or monitor at:
  https://github.com/CalcMark/go-calcmark/actions

Once complete:
  - Release page: https://github.com/CalcMark/go-calcmark/releases/tag/vX.Y.Z
  - Homebrew: brew tap calcmark/tap && brew upgrade calcmark/tap/calcmark
```

## Step 9: Velocity discussion (automated)

The `velocity-release.yml` GitHub Actions workflow automatically posts a release quality Discussion when the release is published. **Do not post manually** — the workflow handles it.

## Hard Rules

- **Never skip tests.** If `task test` fails, the release stops. Period.
- **Never tag on a dirty working tree.** All changes must be committed first.
- **Never push without user confirmation.** The tag push is irreversible in practice.
- **Always show the changelog.** The user must see and approve what's being released.
- **Never amend or force-push tags.** If something went wrong, see RELEASE.md > Troubleshooting.
