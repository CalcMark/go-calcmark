#!/usr/bin/env bash
# lead-time.sh — Calculate lead time for a GitHub issue.
#
# If the issue's commits are included in a release tag, measures created → release.
# Otherwise measures created → now (snapshot).
#
# Usage: ./lead-time.sh <ISSUE_NUMBER>
set -euo pipefail

ISSUE_NUMBER="${1:?Usage: lead-time.sh <ISSUE_NUMBER>}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/helpers.sh"

ISSUE_JSON=$(gh issue view "$ISSUE_NUMBER" --json createdAt,title,number,state)
CREATED=$(echo "$ISSUE_JSON" | jq -r '.createdAt')
TITLE=$(echo "$ISSUE_JSON" | jq -r '.title')

CREATED_EPOCH=$(parse_iso_epoch "$CREATED")

# Try to find a release containing this issue's commits
RELEASE_TAG=""
LAST_COMMIT=$(git log --all --grep="(#${ISSUE_NUMBER})" --format="%H" | head -1)
if [ -n "$LAST_COMMIT" ]; then
  RELEASE_TAG=$(find_release_tag "$LAST_COMMIT")
fi

if [ -n "$RELEASE_TAG" ]; then
  END_EPOCH=$(tag_epoch "$RELEASE_TAG")
  END_LABEL="$RELEASE_TAG release"
else
  END_EPOCH=$(date +%s)
  END_LABEL="now (not yet released)"
fi

LEAD_SECONDS=$((END_EPOCH - CREATED_EPOCH))

echo "## Issue #${ISSUE_NUMBER}: $TITLE"
echo "- Created: $CREATED"
echo "- Lead time: $(format_duration $LEAD_SECONDS) (created → $END_LABEL)"
