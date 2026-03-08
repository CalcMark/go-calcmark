---
name: github-project
description: Manage GitHub issues and project status for CalcMark work. Use when starting work, transitioning status, creating issues, or finishing features. Handles both local-only workflows (merge to main) and remote PR workflows.
allowed-tools: Bash(gh:*),Bash(git:*),Bash(jq:*),Bash(echo:*),Bash(printf:*),Read,Glob,Grep
---

# GitHub Project Management

Manage the CalcMark Tracker project board, GitHub issues, and PRs throughout the development lifecycle. Supports two workflow modes: **local** (solo, merge to main) and **remote** (PR-based, collaborative).

## Project Configuration

```
Project:    CalcMark Tracker
Project #:  1
Owner:      CalcMark
Project ID: PVT_kwDODnnY_M4BJ1QY

Status Field ID: PVTSSF_lADODnnY_M4BJ1QYzg54SEs

Status Option IDs:
  Backlog:     f75ad846
  Ready:       61e4505c
  In progress: 47fc9ee4
  In review:   df73e18b
  Done:        98236657
```

## Status Lifecycle

```
Backlog ──► Ready ──► In progress ──► In review ──► Done
  │           │                           │
  │  planning/research done      fix pushed / review
  │                              agent running
  └── idea filed
```

| Status | Meaning | Who sets it |
|--------|---------|-------------|
| Backlog | Idea filed, not yet planned | Agent or human |
| Ready | Plan/research complete, ready to code | Agent |
| In progress | Code is being written | Agent |
| In review | PR waiting for human review, or review agents running | Agent |
| Done | Work shipped and verified | **Human only** — NEVER set this automatically |

**Closing an issue automatically sets Done. NEVER close issues — only humans do that.**

## Workflow Modes

### Mode 1: Local (Solo Engineer)

Use when the change is small-to-medium and doesn't need external review. Work happens on a local branch, merged to main locally.

```
1. Find or create GitHub issue
2. Set status: In progress
3. Create local branch: fix/description or feat/description
4. Do the work (plan, code, test)
5. Set status: In review (while review agents run)
6. Merge to main locally (git merge --no-ff)
7. Push main to origin
8. Set status: In review (human verifies)
```

### Mode 2: Remote PR

Use when the change needs external review, is large, or uses worktrees for parallel work.

```
1. Find or create GitHub issue
2. Set status: In progress
3. Create branch and push to origin
4. Do the work (plan, code, test)
5. Create PR referencing the issue
6. Set status: In review
7. Human merges PR and closes issue
```

## Issue Discovery

When starting work, determine the tracking issue:

1. **User provided an issue number** — Use it directly: `gh issue view <number>`
2. **User described a bug/feature** — Search for an existing issue:
   ```bash
   gh issue list --state open --search "<keywords>" --json number,title,state --jq '.[] | "\(.number): \(.title)"'
   ```
3. **No issue found** — Create one:
   ```bash
   gh issue create --title "<type>: <description>" --body "<details>"
   ```

After creating or finding the issue, add it to the project if not already there:
```bash
# Add issue to project and get the item ID
gh project item-add 1 --owner CalcMark --url "https://github.com/CalcMark/go-calcmark/issues/<NUMBER>" --format json | jq -r '.id'
```

## Status Transitions

Use this helper pattern for all status changes:

```bash
# Get the project item ID for an issue
ITEM_ID=$(gh project item-list 1 --owner CalcMark --format json | jq -r '.items[] | select(.content.number == <ISSUE_NUMBER>) | .id')

# Set status (replace <OPTION_ID> with the status option ID from config above)
gh project item-edit --id "$ITEM_ID" \
  --field-id PVTSSF_lADODnnY_M4BJ1QYzg54SEs \
  --project-id PVT_kwDODnnY_M4BJ1QY \
  --single-select-option-id <OPTION_ID>
```

Shorthand for common transitions:

```bash
# → Ready (planning done)
gh project item-edit --id "$ITEM_ID" --field-id PVTSSF_lADODnnY_M4BJ1QYzg54SEs --project-id PVT_kwDODnnY_M4BJ1QY --single-select-option-id 61e4505c

# → In progress (coding started)
gh project item-edit --id "$ITEM_ID" --field-id PVTSSF_lADODnnY_M4BJ1QYzg54SEs --project-id PVT_kwDODnnY_M4BJ1QY --single-select-option-id 47fc9ee4

# → In review (PR created or review agents running)
gh project item-edit --id "$ITEM_ID" --field-id PVTSSF_lADODnnY_M4BJ1QYzg54SEs --project-id PVT_kwDODnnY_M4BJ1QY --single-select-option-id df73e18b
```

## Metrics on Completion

When work is finished (status set to "In review" and handed to human), print a metrics summary for the completed issue. This data feeds lead time and cycle time tracking.

### Lead Time (issue created → now)

```bash
# Get issue creation time and compute lead time
ISSUE_JSON=$(gh issue view <NUMBER> --json createdAt,title,number,closedAt,state)
CREATED=$(echo "$ISSUE_JSON" | jq -r '.createdAt')
TITLE=$(echo "$ISSUE_JSON" | jq -r '.title')
NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Calculate lead time in hours (macOS)
CREATED_EPOCH=$(date -jf "%Y-%m-%dT%H:%M:%SZ" "$CREATED" +%s 2>/dev/null || date -d "$CREATED" +%s)
NOW_EPOCH=$(date +%s)
LEAD_HOURS=$(echo "scale=1; ($NOW_EPOCH - $CREATED_EPOCH) / 3600" | bc)

echo "## Issue #<NUMBER>: $TITLE"
echo "- Created: $CREATED"
echo "- Lead time: ${LEAD_HOURS}h (issue open → work complete)"
```

### Cycle Time (first commit on branch → last commit)

```bash
# Get cycle time from branch commits
FIRST_COMMIT=$(git log main..<BRANCH> --reverse --format="%H %aI" | head -1)
LAST_COMMIT=$(git log main..<BRANCH> --format="%H %aI" | head -1)

FIRST_TIME=$(echo "$FIRST_COMMIT" | awk '{print $2}')
LAST_TIME=$(echo "$LAST_COMMIT" | awk '{print $2}')

FIRST_EPOCH=$(date -jf "%Y-%m-%dT%H:%M:%S%z" "$FIRST_TIME" +%s 2>/dev/null || date -d "$FIRST_TIME" +%s)
LAST_EPOCH=$(date -jf "%Y-%m-%dT%H:%M:%S%z" "$LAST_TIME" +%s 2>/dev/null || date -d "$LAST_TIME" +%s)
CYCLE_MINUTES=$(echo "scale=1; ($LAST_EPOCH - $FIRST_EPOCH) / 60" | bc)

COMMIT_COUNT=$(git rev-list --count main..<BRANCH>)
FILES_CHANGED=$(git diff --stat main..<BRANCH> | tail -1)

echo "- Cycle time: ${CYCLE_MINUTES}m (first commit → last commit)"
echo "- Commits: $COMMIT_COUNT"
echo "- $FILES_CHANGED"
```

### PR Metrics (if remote workflow)

```bash
PR_JSON=$(gh pr view <PR_NUMBER> --json createdAt,mergedAt,additions,deletions,changedFiles,commits)
PR_CREATED=$(echo "$PR_JSON" | jq -r '.createdAt')
ADDITIONS=$(echo "$PR_JSON" | jq -r '.additions')
DELETIONS=$(echo "$PR_JSON" | jq -r '.deletions')
CHANGED_FILES=$(echo "$PR_JSON" | jq -r '.changedFiles')
PR_COMMITS=$(echo "$PR_JSON" | jq -r '.commits | length')

echo "- PR: +$ADDITIONS/-$DELETIONS across $CHANGED_FILES files ($PR_COMMITS commits)"
```

### Full Summary Template

Print this when handing work to the human:

```
────────────────────────────────────────
Issue #NN: <title>
────────────────────────────────────────
Status:     In review → awaiting human verification
Lead time:  X.Xh (created → now)
Cycle time: X.Xm (first commit → last commit)
Commits:    N
Changed:    N files, +A/-D lines
Branch:     <branch-name>
PR:         #NN (if applicable)
────────────────────────────────────────
```

This summary provides the raw data for velocity tracking. Over time, trends in lead time and cycle time reveal whether solution documents, tests, and architectural patterns are compounding — shorter times for similar-complexity issues means the system is working.

### Future: `gh velocity` Extension

These metrics scripts are candidates for a future `gh` CLI extension written in Go:

```
gh velocity show          # current open issues with age
gh velocity stats         # lead/cycle time stats across closed issues
gh velocity trend         # trend chart over last N issues/releases
gh velocity release v1.2  # metrics for a specific release
```

This is out of scope for now but the data model is already in place via issue timestamps, commit timestamps, and PR metadata.

## Integration with Compound Engineering Pipeline

This skill integrates with the `/lfg` pipeline:

| Pipeline Step | GitHub Action |
|---------------|--------------|
| `/workflows:plan` | Create or find issue, set **Ready** |
| `/workflows:work` | Set **In progress** on first commit |
| `/workflows:review` | Set **In review** when review agents launch |
| Completion | Print metrics summary, leave at **In review** for human |

## Rules

1. **NEVER close issues** — only humans close issues
2. **NEVER set status to Done** — closing does this automatically
3. **Always create or reference an issue** — no untracked work
4. **Always print metrics** when handing work back to the human
5. **Use `gh` CLI for all GitHub operations** — no API tokens or web UI
6. **Local branches are fine for solo work** — PRs are for collaboration or worktrees
