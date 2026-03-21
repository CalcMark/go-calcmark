---
title: "feat: Integrate gh-velocity for automated project metrics"
type: feat
status: completed
date: 2026-03-21
---

# feat: Integrate gh-velocity for automated project metrics

## Overview

Replace custom velocity bash scripts with the `gh-velocity` GitHub CLI extension. Add three GitHub Actions workflows: weekly Discussion metrics, PR merge metrics insertion, and issue close metrics insertion. Update the `/release` command to use `gh-velocity` instead of `release-velocity.sh`.

## Problem Statement / Motivation

The current velocity system is six custom bash scripts (release-velocity.sh, issue-summary.sh, cycle-time.sh, lead-time.sh, pr-metrics.sh, helpers.sh) that only fire at release time. This means:

- No ongoing visibility into project health between releases
- No metrics on individual PRs or issues when they close
- The scripts duplicate what `gh-velocity` provides out of the box with richer statistical analysis (P90, P95, outlier detection, quality ratios)
- Maintaining custom date-parsing and GraphQL logic is unnecessary overhead

## Proposed Solution

### 1. Install and configure gh-velocity

Run `gh velocity config preflight --write` to auto-generate `.gh-velocity.yml` at the repository root. The preflight command examines existing labels, project boards, and recent activity to produce an optimal configuration.

### 2. Three new GitHub Actions workflows

**a) `.github/workflows/velocity-weekly.yml`** — Weekly Discussion post (Monday 09:00 UTC)
- Runs `gh velocity report --since 7d --post -r markdown`
- Posts to Announcements Discussion category
- Includes `workflow_dispatch` for manual testing
- Uses `--post` for idempotent updates (keyed on date range, won't duplicate)

**b) `.github/workflows/velocity-pr.yml`** — PR merge metrics
- Triggers on `pull_request: types: [closed]`
- Guards with `if: github.event.pull_request.merged == true`
- Runs `gh velocity pr $PR_NUMBER -r markdown`
- Appends metrics to PR body wrapped in `<!-- gh-velocity-start -->` / `<!-- gh-velocity-end -->` sentinel markers
- Replaces existing block on re-run (idempotent)
- Skips bot PRs (check `github.actor` against known bots)

**c) `.github/workflows/velocity-issue.yml`** — Issue close metrics
- Triggers on `issues: types: [closed]`
- Guards with `if: github.event.issue.state_reason == 'completed'` (skips "not planned")
- Runs `gh velocity issue $ISSUE_NUMBER -r markdown`
- Appends metrics to issue body with same sentinel marker pattern
- Idempotent on re-run

### 3. Update `/release` command

Replace Step 9 in `.claude/commands/release.md` to call `gh velocity quality release vX.Y.Z --post -r markdown` instead of `release-velocity.sh`.

### 4. Keep local scripts for agent use (no deletion)

The `issue-summary.sh` is used by the `github-project` skill for local agent metrics. These scripts remain available. The new workflows are additive CI automation, not a script removal.

## Technical Considerations

### Permissions

| Workflow | Permissions needed |
|----------|--------------------|
| Weekly report | `contents: read`, `discussions: write` |
| PR merge | `contents: read`, `pull-requests: write` |
| Issue close | `contents: read`, `issues: write` |

The default `GITHUB_TOKEN` covers all these scopes. A `GH_VELOCITY_TOKEN` PAT is only needed if using Projects v2 board data for velocity iterations — not required for the core metrics (lead time, cycle time, throughput, quality).

### Idempotency

PR and issue body edits use HTML comment sentinels:
```
<!-- gh-velocity-start -->
...metrics markdown...
<!-- gh-velocity-end -->
```

The workflow checks for existing markers and replaces the block rather than appending. This handles workflow re-runs gracefully.

### Race condition: PR merge closing an issue

When a PR closes an issue via "Closes #N", both workflows fire concurrently. This is safe because they edit **different resources** (PR body vs issue body). No lock needed.

### gh-velocity installation in CI

Each workflow installs the extension fresh:
```yaml
- run: gh extension install dvhthomas/gh-velocity
  env:
    GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

Installation takes ~3 seconds — acceptable for event-driven workflows. Pin to a version tag when one is available.

### Rate limits

`gh velocity pr` or `gh velocity issue` costs 1-3 API calls — well within GitHub's 1000 requests/hour for Actions. The weekly report may cost 5-50 calls depending on volume. The `api_throttle_seconds` config option is available if needed.

### Bot exclusion

```yaml
exclude_users:
  - "dependabot[bot]"
  - "renovate[bot]"
```

This keeps automated dependency PRs from skewing velocity metrics.

## System-Wide Impact

- **Interaction graph**: PR merge → velocity-pr workflow → edits PR body. Issue close → velocity-issue workflow → edits issue body. Weekly cron → velocity-weekly workflow → creates/updates Discussion. Release tag push → release.yml → GoReleaser (unchanged). The `/release` command's Step 9 now calls `gh velocity` instead of `release-velocity.sh`.
- **Error propagation**: Workflow failures are isolated — a failed metrics insertion does not affect merges, releases, or other workflows. Non-zero exit from `gh velocity` fails the workflow step but the PR/issue is already closed.
- **State lifecycle risks**: No persistent state. Metrics are computed on-demand from GitHub's event timeline. Sentinel markers in PR/issue bodies are the only state, and they're idempotent.
- **API surface parity**: The `github-project` skill continues to call `issue-summary.sh` locally. CI automation uses `gh-velocity`. Two parallel systems, but they serve different contexts (local agent vs CI).

## Acceptance Criteria

- [ ] `.gh-velocity.yml` exists at repo root with quality categories matching existing labels
- [ ] `velocity-weekly.yml` workflow posts a Discussion every Monday at 09:00 UTC with 7-day metrics
- [ ] `velocity-weekly.yml` supports `workflow_dispatch` for manual testing
- [ ] `velocity-pr.yml` workflow appends metrics to PR body on merge (not on close-without-merge)
- [ ] `velocity-issue.yml` workflow appends metrics to issue body on close-as-completed (not "not planned")
- [ ] PR/issue body edits are idempotent (sentinel markers, no duplication on re-run)
- [ ] Bot PRs are excluded from metrics (dependabot, renovate)
- [ ] `/release` command Step 9 calls `gh velocity` instead of `release-velocity.sh`
- [ ] Existing local scripts remain functional (no breaking changes to `github-project` skill)
- [ ] All workflows have `workflow_dispatch` triggers for testing

## Success Metrics

- Weekly Discussion posts appear consistently every Monday
- Every merged PR has velocity metrics in its body within 2 minutes of merge
- Every completed issue has velocity metrics in its body within 2 minutes of close
- Release velocity Discussions continue to post (now via `gh velocity quality release`)

## Dependencies & Risks

| Risk | Mitigation |
|------|------------|
| `gh-velocity` extension unavailable or breaking change | Pin version; existing scripts remain as fallback |
| `GITHUB_TOKEN` insufficient for Discussions | Test with `workflow_dispatch` before relying on cron |
| Rate limiting on busy days (many concurrent merges) | `api_throttle_seconds: 2` in config |
| `gh velocity` output format changes | Sentinel markers isolate metrics block; format changes are cosmetic only |

## Implementation Phases

### Phase 1: Configuration and weekly report
1. Create `.gh-velocity.yml` with quality categories, bot exclusions
2. Create `.github/workflows/velocity-weekly.yml`
3. Test with `workflow_dispatch`

### Phase 2: PR and issue metrics
4. Create `.github/workflows/velocity-pr.yml` with sentinel-based body editing
5. Create `.github/workflows/velocity-issue.yml` with state_reason guard
6. Test both with manual workflow runs

### Phase 3: Release integration
7. Update `.claude/commands/release.md` Step 9 to use `gh velocity quality release`
8. Test with a dry-run release

## Files to Create/Modify

| Action | File |
|--------|------|
| Create | `.gh-velocity.yml` |
| Create | `.github/workflows/velocity-weekly.yml` |
| Create | `.github/workflows/velocity-pr.yml` |
| Create | `.github/workflows/velocity-issue.yml` |
| Modify | `.claude/commands/release.md` (Step 9) |

## Sources & References

- gh-velocity documentation: https://dvhthomas.github.io/gh-velocity/
- Existing velocity scripts: `.claude/skills/github-project/scripts/release-velocity.sh`
- Existing release command: `.claude/commands/release.md` (lines 153-161)
- Existing workflows: `.github/workflows/release.yml`
