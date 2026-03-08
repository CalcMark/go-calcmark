#!/usr/bin/env bash
# lead-time.sh — Calculate lead time snapshot for a GitHub issue (created → now).
# True lead time is measured at release (created → included in release).
# This snapshot shows elapsed time since filing — useful during development.
# Usage: ./lead-time.sh <ISSUE_NUMBER>
set -euo pipefail

ISSUE_NUMBER="${1:?Usage: lead-time.sh <ISSUE_NUMBER>}"

ISSUE_JSON=$(gh issue view "$ISSUE_NUMBER" --json createdAt,title,number,closedAt,state)
CREATED=$(echo "$ISSUE_JSON" | jq -r '.createdAt')
TITLE=$(echo "$ISSUE_JSON" | jq -r '.title')
NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Calculate lead time in hours (macOS date -jf, Linux date -d)
CREATED_EPOCH=$(date -jf "%Y-%m-%dT%H:%M:%SZ" "$CREATED" +%s 2>/dev/null || date -d "$CREATED" +%s)
NOW_EPOCH=$(date +%s)
LEAD_HOURS=$(echo "scale=1; ($NOW_EPOCH - $CREATED_EPOCH) / 3600" | bc)

echo "## Issue #${ISSUE_NUMBER}: $TITLE"
echo "- Created: $CREATED"
echo "- Lead time: ${LEAD_HOURS}h (issue open → work complete)"
