#!/usr/bin/env bash
# cycle-time.sh — Snapshot cycle time from branch commits (first commit → last commit).
# True cycle time ends when work ships in a public release.
# This snapshot shows active coding duration — useful during development.
# Usage: ./cycle-time.sh <BRANCH>
set -euo pipefail

BRANCH="${1:?Usage: cycle-time.sh <BRANCH>}"

FIRST_COMMIT=$(git log main.."$BRANCH" --reverse --format="%H %aI" | head -1)
LAST_COMMIT=$(git log main.."$BRANCH" --format="%H %aI" | head -1)

if [ -z "$FIRST_COMMIT" ] || [ -z "$LAST_COMMIT" ]; then
  echo "No commits found on branch $BRANCH relative to main"
  exit 1
fi

FIRST_TIME=$(echo "$FIRST_COMMIT" | awk '{print $2}')
LAST_TIME=$(echo "$LAST_COMMIT" | awk '{print $2}')

# macOS date -jf, Linux date -d
FIRST_EPOCH=$(date -jf "%Y-%m-%dT%H:%M:%S%z" "$FIRST_TIME" +%s 2>/dev/null || date -d "$FIRST_TIME" +%s)
LAST_EPOCH=$(date -jf "%Y-%m-%dT%H:%M:%S%z" "$LAST_TIME" +%s 2>/dev/null || date -d "$LAST_TIME" +%s)
CYCLE_MINUTES=$(echo "scale=1; ($LAST_EPOCH - $FIRST_EPOCH) / 60" | bc)

COMMIT_COUNT=$(git rev-list --count main.."$BRANCH")
FILES_CHANGED=$(git diff --stat main.."$BRANCH" | tail -1)

echo "- Cycle time: ${CYCLE_MINUTES}m (first commit → last commit)"
echo "- Commits: $COMMIT_COUNT"
echo "- $FILES_CHANGED"
