#!/usr/bin/env bash
# cycle-time.sh — Measure coding duration for a GitHub issue.
#
# Finds commits associated with an issue through multiple signals:
#   1. Commits mentioning (#N) in the message (conventional commit trailers)
#   2. Commits linked to a PR that closes the issue
#   3. Commits on a branch passed as optional second argument
#   4. Commit SHAs mentioned in the issue body or comments
#
# Usage: ./cycle-time.sh <ISSUE_NUMBER> [BRANCH]
set -euo pipefail

ISSUE_NUMBER="${1:?Usage: cycle-time.sh <ISSUE_NUMBER> [BRANCH]}"
BRANCH="${2:-}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/helpers.sh"

# Collect commit hashes from all sources (deduped later)
COMMIT_HASHES=""

# Source 1: Commits referencing (#N) in the message
PATTERN_COMMITS=$(git log --all --grep="(#${ISSUE_NUMBER})" --format="%H" 2>/dev/null || true)
if [ -n "$PATTERN_COMMITS" ]; then
  COMMIT_HASHES="$PATTERN_COMMITS"
fi

# Source 2: Commits on an explicit branch (relative to main)
if [ -n "$BRANCH" ] && [ "$BRANCH" != "main" ]; then
  BRANCH_COMMITS=$(git log main.."$BRANCH" --format="%H" 2>/dev/null || true)
  if [ -n "$BRANCH_COMMITS" ]; then
    COMMIT_HASHES=$(printf "%s\n%s" "$COMMIT_HASHES" "$BRANCH_COMMITS")
  fi
fi

# Source 3: Commits from PRs that reference the issue
PR_NUMBERS=$(gh issue view "$ISSUE_NUMBER" --json timelineItems 2>/dev/null | \
  jq -r '[.timelineItems[] | select(.source.__typename == "PullRequest") | .source.number] | unique | .[]' 2>/dev/null || true)

for PR in $PR_NUMBERS; do
  PR_COMMITS=$(gh pr view "$PR" --json commits 2>/dev/null | \
    jq -r '.commits[].oid' 2>/dev/null || true)
  if [ -n "$PR_COMMITS" ]; then
    COMMIT_HASHES=$(printf "%s\n%s" "$COMMIT_HASHES" "$PR_COMMITS")
  fi
done

# Source 4: Commit SHAs mentioned in the issue body or comments
ISSUE_TEXT=$(gh issue view "$ISSUE_NUMBER" --json body,comments \
  --jq '[.body, (.comments[]?.body // empty)] | join("\n")' 2>/dev/null || true)
if [ -n "$ISSUE_TEXT" ]; then
  # Match 7-40 char hex strings that are valid git objects
  MENTIONED_SHAS=$(echo "$ISSUE_TEXT" | grep -oE '\b[0-9a-f]{7,40}\b' | while read -r sha; do
    # Verify it's a real commit in this repo
    if git cat-file -t "$sha" 2>/dev/null | grep -q "commit"; then
      git rev-parse "$sha" 2>/dev/null
    fi
  done || true)
  if [ -n "$MENTIONED_SHAS" ]; then
    COMMIT_HASHES=$(printf "%s\n%s" "$COMMIT_HASHES" "$MENTIONED_SHAS")
  fi
fi

# Deduplicate and filter empty lines
COMMIT_HASHES=$(echo "$COMMIT_HASHES" | sort -u | sed '/^$/d')

if [ -z "$COMMIT_HASHES" ]; then
  echo "No commits found for issue #${ISSUE_NUMBER}"
  echo "Looked for: commits with '(#${ISSUE_NUMBER})' in message, linked PRs${BRANCH:+, branch $BRANCH}"
  exit 1
fi

# Get timestamps for all commits, sort chronologically
COMMIT_TIMES=$(echo "$COMMIT_HASHES" | while read -r hash; do
  git log -1 --format="%aI %H" "$hash" 2>/dev/null || true
done | sort)

FIRST_LINE=$(echo "$COMMIT_TIMES" | head -1)
LAST_LINE=$(echo "$COMMIT_TIMES" | tail -1)
FIRST_TIME=$(echo "$FIRST_LINE" | awk '{print $1}')
LAST_TIME=$(echo "$LAST_LINE" | awk '{print $1}')
FIRST_HASH=$(echo "$FIRST_LINE" | awk '{print $2}')
LAST_HASH=$(echo "$LAST_LINE" | awk '{print $2}')
COMMIT_COUNT=$(echo "$COMMIT_HASHES" | wc -l | tr -d ' ')

FIRST_EPOCH=$(parse_iso_epoch "$FIRST_TIME")
LAST_EPOCH=$(parse_iso_epoch "$LAST_TIME")
DIFF_SECONDS=$((LAST_EPOCH - FIRST_EPOCH))

FIRST_SHORT=$(git log -1 --format="%h %s" "$FIRST_HASH")
LAST_SHORT=$(git log -1 --format="%h %s" "$LAST_HASH")

echo "- Cycle time: $(format_duration $DIFF_SECONDS) (first commit → last commit)"
echo "- Commits: $COMMIT_COUNT"
echo "- First: $FIRST_SHORT"
echo "- Last:  $LAST_SHORT"
# Machine-readable last hash for callers (e.g., issue-summary.sh release detection)
echo "- LastHash: $LAST_HASH"
