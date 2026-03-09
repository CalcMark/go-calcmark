---
name: github-project
description: Manage GitHub issues and project status for CalcMark work. Use when starting work, transitioning status, creating issues, or finishing features. Handles both local-only workflows (merge to main) and remote PR workflows.
allowed-tools: Bash(gh:*),Bash(git:*),Bash(jq:*),Bash(echo:*),Bash(printf:*),Bash(.claude/skills/github-project/scripts/*),Read,Glob,Grep
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

When work is finished (status set to "In review" and handed to human), run the metrics scripts to print a completion summary.

**Definitions:**

- **Lead time** = issue created → included in a public release. Measures the full idea-to-delivery pipeline.
- **Cycle time** = work started (issue → "In progress" / first commit) → included in a public release. Measures active development speed.

Both metrics share the same end point: the work appears in a release that users can see. The difference is the start point — lead time starts when the idea is filed, cycle time starts when coding begins.

### Scripts

All metric scripts live in `.claude/skills/github-project/scripts/`:

| Script | Purpose | Usage |
|--------|---------|-------|
| `lead-time.sh` | Issue created → now (snapshot; true lead time measured at release) | `.claude/skills/github-project/scripts/lead-time.sh <ISSUE_NUMBER>` |
| `cycle-time.sh` | First commit → last commit for an issue (active work duration) | `.claude/skills/github-project/scripts/cycle-time.sh <ISSUE_NUMBER> [BRANCH]` |
| `pr-metrics.sh` | PR size (additions, deletions, files) | `.claude/skills/github-project/scripts/pr-metrics.sh <PR_NUMBER>` |
| `issue-summary.sh` | Full completion summary (auto-detects release) | `.claude/skills/github-project/scripts/issue-summary.sh <ISSUE_NUMBER> [BRANCH] [PR_NUMBER]` |
| `release-velocity.sh` | Post velocity Discussion for a release | `.claude/skills/github-project/scripts/release-velocity.sh <TAG>` |
| `helpers.sh` | Shared functions (sourced by other scripts) | `source .claude/skills/github-project/scripts/helpers.sh` |

`cycle-time.sh` finds commits associated with an issue through multiple signals:
1. Commits mentioning `(#N)` in the message (conventional commit trailers)
2. Commits linked to PRs that reference the issue
3. Commits on an explicit branch (optional fallback)
4. Commit SHAs mentioned in the issue body or comments

Run the summary script when handing work to the human:

```bash
.claude/skills/github-project/scripts/issue-summary.sh <ISSUE_NUMBER> [BRANCH] [PR_NUMBER]
```

### Release Velocity Discussion

After every release, post a velocity report as a GitHub Discussion in the Announcements category:

```bash
.claude/skills/github-project/scripts/release-velocity.sh v1.6.5
```

This is called automatically at the end of the `/release` skill (Step 9). It:
1. Finds all issues referenced in commits between the previous tag and this tag
2. Calculates lead time (created → release) and cycle time (first → last commit) for each
3. Posts a Discussion with a metrics table and link to release notes

The `/release` skill MUST call this script after Step 8 (post-release summary).

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
