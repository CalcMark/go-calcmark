---
name: release-status
description: Check the status of the latest GitHub Actions release workflow
argument-hint: "[tag]"
allowed-tools:
  - Bash
  - AskUserQuestion
---

Check the GoReleaser workflow status for a release tag. Uses `gh` CLI to query GitHub Actions directly. See RELEASE.md for the full release process and troubleshooting.

## Step 1: Determine which tag to check

If `$ARGUMENTS` is provided, use it as the tag (add `v` prefix if missing).

If `$ARGUMENTS` is empty, find the most recent tag:

```bash
git tag --sort=-version:refname | head -1
```

Print: "Checking release status for v{X.Y.Z}..."

## Step 2: Verify gh CLI is available

```bash
gh --version
```

If `gh` is not installed, **stop** and tell the user:

> "The `gh` CLI is required. Install with `brew install gh` and authenticate with `gh auth login`."

## Step 3: Find the workflow run for this tag

```bash
gh run list --workflow=release.yml --limit=5 --json databaseId,headBranch,status,conclusion,createdAt,url
```

Find the run that corresponds to the tag. The `headBranch` will match the tag name (e.g., `v1.1.0`).

If no matching run is found, tell the user:

> "No release workflow run found for v{X.Y.Z}. The tag may not have been pushed yet, or the workflow hasn't started."

## Step 4: Report workflow status

Based on the run's `status` and `conclusion`:

### If `status` is `in_progress` or `queued`:

Print:

```
⏳ Release v{X.Y.Z} is in progress...

  Workflow: {url}
  Started:  {createdAt}
```

Then ask:

```
AskUserQuestion(
  header: "Wait?",
  question: "The workflow is still running. Want me to watch it until it finishes?",
  options: [
    { label: "Watch it", description: "Poll every 15 seconds until it completes" },
    { label: "That's fine", description: "I'll check back later" }
  ]
)
```

If the user wants to watch, run:

```bash
gh run watch {databaseId} --exit-status
```

This blocks until the run completes and exits non-zero if the run failed.

After it completes, continue to Step 5.

### If `status` is `completed` and `conclusion` is `success`:

Continue to Step 5.

### If `status` is `completed` and `conclusion` is `failure`:

Continue to Step 6.

## Step 5: Report success

```bash
gh release view v{X.Y.Z} --json tagName,name,createdAt,assets --jq '{tag: .tagName, name: .name, created: .createdAt, assets: [.assets[].name]}'
```

Print a summary:

```
✅ Release v{X.Y.Z} published successfully!

  Tag:     v{X.Y.Z}
  Created: {createdAt}

  Assets:
    - {list each asset name}

  Links:
    - Release: https://github.com/CalcMark/go-calcmark/releases/tag/v{X.Y.Z}
    - Homebrew: brew upgrade calcmark/tap/calcmark
```

## Step 6: Report failure

Get the failed job logs:

```bash
gh run view {databaseId} --json jobs --jq '.jobs[] | select(.conclusion == "failure") | {name: .name, conclusion: .conclusion}'
```

Then fetch the log output for the failed run:

```bash
gh run view {databaseId} --log-failed 2>&1 | tail -40
```

Print:

```
❌ Release v{X.Y.Z} workflow failed.

  Workflow: {url}

  Failed jobs:
    {list failed job names}

  Last 40 lines of failure log:
    {log output}
```

Then advise:

```
See RELEASE.md > Troubleshooting > Workflow Failed in CI for recovery steps.
```

## Hard Rules

- **Read-only.** This command never modifies the repo, pushes, or deletes anything.
- **Always show the workflow URL** so the user can click through to the full GitHub Actions log.
- **Never suggest force-pushing or amending tags.** Always point to the delete-and-retag flow.
