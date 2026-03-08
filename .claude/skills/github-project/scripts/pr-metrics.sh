#!/usr/bin/env bash
# pr-metrics.sh — Fetch PR size metrics from GitHub.
# Usage: ./pr-metrics.sh <PR_NUMBER>
set -euo pipefail

PR_NUMBER="${1:?Usage: pr-metrics.sh <PR_NUMBER>}"

PR_JSON=$(gh pr view "$PR_NUMBER" --json createdAt,mergedAt,additions,deletions,changedFiles,commits)
ADDITIONS=$(echo "$PR_JSON" | jq -r '.additions')
DELETIONS=$(echo "$PR_JSON" | jq -r '.deletions')
CHANGED_FILES=$(echo "$PR_JSON" | jq -r '.changedFiles')
PR_COMMITS=$(echo "$PR_JSON" | jq -r '.commits | length')

echo "- PR: +$ADDITIONS/-$DELETIONS across $CHANGED_FILES files ($PR_COMMITS commits)"
