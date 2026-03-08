#!/usr/bin/env bash
# issue-summary.sh — Print full completion metrics for an issue + branch.
# Usage: ./issue-summary.sh <ISSUE_NUMBER> <BRANCH> [PR_NUMBER]
set -euo pipefail

ISSUE_NUMBER="${1:?Usage: issue-summary.sh <ISSUE_NUMBER> <BRANCH> [PR_NUMBER]}"
BRANCH="${2:?Usage: issue-summary.sh <ISSUE_NUMBER> <BRANCH> [PR_NUMBER]}"
PR_NUMBER="${3:-}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Gather issue info
ISSUE_JSON=$(gh issue view "$ISSUE_NUMBER" --json createdAt,title,number,state)
TITLE=$(echo "$ISSUE_JSON" | jq -r '.title')
CREATED=$(echo "$ISSUE_JSON" | jq -r '.createdAt')
NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Lead time
CREATED_EPOCH=$(date -jf "%Y-%m-%dT%H:%M:%SZ" "$CREATED" +%s 2>/dev/null || date -d "$CREATED" +%s)
NOW_EPOCH=$(date +%s)
LEAD_HOURS=$(echo "scale=1; ($NOW_EPOCH - $CREATED_EPOCH) / 3600" | bc)

# Cycle time
FIRST_COMMIT=$(git log main.."$BRANCH" --reverse --format="%H %aI" | head -1)
LAST_COMMIT=$(git log main.."$BRANCH" --format="%H %aI" | head -1)

CYCLE_MINUTES="N/A"
COMMIT_COUNT="0"
FILES_CHANGED="no changes"

if [ -n "$FIRST_COMMIT" ] && [ -n "$LAST_COMMIT" ]; then
  FIRST_TIME=$(echo "$FIRST_COMMIT" | awk '{print $2}')
  LAST_TIME=$(echo "$LAST_COMMIT" | awk '{print $2}')
  FIRST_EPOCH=$(date -jf "%Y-%m-%dT%H:%M:%S%z" "$FIRST_TIME" +%s 2>/dev/null || date -d "$FIRST_TIME" +%s)
  LAST_EPOCH=$(date -jf "%Y-%m-%dT%H:%M:%S%z" "$LAST_TIME" +%s 2>/dev/null || date -d "$LAST_TIME" +%s)
  CYCLE_MINUTES=$(echo "scale=1; ($LAST_EPOCH - $FIRST_EPOCH) / 60" | bc)
  COMMIT_COUNT=$(git rev-list --count main.."$BRANCH")
  FILES_CHANGED=$(git diff --stat main.."$BRANCH" | tail -1)
fi

# Print summary
echo "────────────────────────────────────────"
echo "Issue #${ISSUE_NUMBER}: $TITLE"
echo "────────────────────────────────────────"
echo "Status:     In review → awaiting human verification"
echo "Lead time:  ${LEAD_HOURS}h elapsed (created → now; ends at release)"
echo "Cycle time: ${CYCLE_MINUTES}m coding (first commit → last commit; ends at release)"
echo "Commits:    $COMMIT_COUNT"
echo "Changed:    $FILES_CHANGED"
echo "Branch:     $BRANCH"

if [ -n "$PR_NUMBER" ]; then
  PR_LINE=$("$SCRIPT_DIR/pr-metrics.sh" "$PR_NUMBER" 2>/dev/null || echo "- PR: #${PR_NUMBER}")
  echo "PR:         #${PR_NUMBER}"
  echo "$PR_LINE"
fi

echo "────────────────────────────────────────"
