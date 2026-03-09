#!/usr/bin/env bash
# issue-summary.sh — Print full completion metrics for a GitHub issue.
#
# Automatically detects if the issue's commits shipped in a release tag.
# If so, lead time measures created → release. Otherwise created → now.
#
# Usage: ./issue-summary.sh <ISSUE_NUMBER> [BRANCH] [PR_NUMBER]
set -euo pipefail

ISSUE_NUMBER="${1:?Usage: issue-summary.sh <ISSUE_NUMBER> [BRANCH] [PR_NUMBER]}"
BRANCH="${2:-}"
PR_NUMBER="${3:-}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/helpers.sh"

# Gather issue info
ISSUE_JSON=$(gh issue view "$ISSUE_NUMBER" --json createdAt,title,number,state)
TITLE=$(echo "$ISSUE_JSON" | jq -r '.title')
CREATED=$(echo "$ISSUE_JSON" | jq -r '.createdAt')
CREATED_EPOCH=$(parse_iso_epoch "$CREATED")

# Cycle time via cycle-time.sh (issue-based)
CYCLE_OUTPUT=$("$SCRIPT_DIR/cycle-time.sh" "$ISSUE_NUMBER" "$BRANCH" 2>/dev/null || echo "- Cycle time: N/A (no commits found)")

# Extract fields from cycle-time output
CYCLE_TIME=$(echo "$CYCLE_OUTPUT" | head -1 | sed 's/^- Cycle time: //')
COMMIT_LINE=$(echo "$CYCLE_OUTPUT" | grep "^- Commits:" | sed 's/^- Commits: //' || echo "0")
FIRST_COMMIT=$(echo "$CYCLE_OUTPUT" | grep "^- First:" || true)
LAST_COMMIT_LINE=$(echo "$CYCLE_OUTPUT" | grep "^- Last:" || true)
LAST_HASH=$(echo "$CYCLE_OUTPUT" | grep "^- LastHash:" | sed 's/^- LastHash: //' || true)

# Detect release: find earliest semver tag containing the last commit
RELEASE_TAG=""
STATUS_LINE="In review → awaiting human verification"
if [ -n "$LAST_HASH" ]; then
  RELEASE_TAG=$(find_release_tag "$LAST_HASH")
fi

if [ -n "$RELEASE_TAG" ]; then
  END_EPOCH=$(tag_epoch "$RELEASE_TAG")
  LEAD_SECONDS=$((END_EPOCH - CREATED_EPOCH))
  LEAD_LABEL="(created → $RELEASE_TAG)"
  STATUS_LINE="Shipped in $RELEASE_TAG"
else
  NOW_EPOCH=$(date +%s)
  LEAD_SECONDS=$((NOW_EPOCH - CREATED_EPOCH))
  LEAD_LABEL="(created → now; not yet released)"
  STATUS_LINE="In review → awaiting human verification"
fi

# Print summary
echo "────────────────────────────────────────"
echo "Issue #${ISSUE_NUMBER}: $TITLE"
echo "────────────────────────────────────────"
echo "Status:     $STATUS_LINE"
echo "Lead time:  $(format_duration $LEAD_SECONDS) $LEAD_LABEL"
echo "Cycle time: ${CYCLE_TIME}"
echo "Commits:    $COMMIT_LINE"

if [ -n "$FIRST_COMMIT" ]; then
  echo "$FIRST_COMMIT"
  echo "$LAST_COMMIT_LINE"
fi

if [ -n "$PR_NUMBER" ]; then
  PR_LINE=$("$SCRIPT_DIR/pr-metrics.sh" "$PR_NUMBER" 2>/dev/null || echo "- PR: #${PR_NUMBER}")
  echo "PR:         #${PR_NUMBER}"
  echo "$PR_LINE"
fi

echo "────────────────────────────────────────"
