#!/usr/bin/env bash
# release-velocity.sh — Post a GitHub Discussion with velocity metrics for a release.
#
# Finds all issues included in a release (via commit messages between tags),
# calculates lead time and cycle time for each, and posts an Announcements
# discussion linking to the release notes.
#
# Usage: ./release-velocity.sh <TAG>
#   e.g.: ./release-velocity.sh v1.6.5
set -euo pipefail

TAG="${1:?Usage: release-velocity.sh <TAG>}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/helpers.sh"

# Validate tag exists
if ! git rev-parse "$TAG" >/dev/null 2>&1; then
  echo "Error: tag $TAG not found"
  exit 1
fi

# Find the previous semver tag
PREV_TAG=$(git tag --sort=-version:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | grep -A1 "^${TAG}$" | tail -1)
if [ "$PREV_TAG" = "$TAG" ]; then
  # TAG is the oldest — use all commits up to it
  COMMIT_RANGE="$TAG"
  PREV_TAG="(initial)"
else
  COMMIT_RANGE="${PREV_TAG}..${TAG}"
fi

# Extract unique issue numbers from commits in this release
ISSUE_NUMBERS=$(git log "$COMMIT_RANGE" --oneline --no-merges \
  | grep -oE '\(#[0-9]+\)' \
  | grep -oE '[0-9]+' \
  | sort -un)

if [ -z "$ISSUE_NUMBERS" ]; then
  echo "No issues found in commits between $PREV_TAG and $TAG"
  exit 1
fi

ISSUE_COUNT=$(echo "$ISSUE_NUMBERS" | wc -l | tr -d ' ')
COMMIT_COUNT=$(git rev-list --count "$COMMIT_RANGE" --no-merges 2>/dev/null || echo "?")
RELEASE_DATE=$(git log -1 --format="%ai" "$TAG" | cut -d' ' -f1)

# Build the metrics table
TABLE_ROWS=""
for ISSUE_NUM in $ISSUE_NUMBERS; do
  # Get issue title
  TITLE=$(gh issue view "$ISSUE_NUM" --json title --jq '.title' 2>/dev/null || echo "(unknown)")

  # Get cycle time output
  CYCLE_OUTPUT=$("$SCRIPT_DIR/cycle-time.sh" "$ISSUE_NUM" 2>/dev/null || true)
  CYCLE_TIME=$(echo "$CYCLE_OUTPUT" | head -1 | sed 's/^- Cycle time: //' | sed 's/ (first commit.*$//' || echo "N/A")
  ISSUE_COMMITS=$(echo "$CYCLE_OUTPUT" | grep "^- Commits:" | sed 's/^- Commits: //' || echo "?")

  # Get lead time (created → release tag)
  CREATED=$(gh issue view "$ISSUE_NUM" --json createdAt --jq '.createdAt' 2>/dev/null || true)
  LEAD_TIME="N/A"
  if [ -n "$CREATED" ]; then
    CREATED_EPOCH=$(parse_iso_epoch "$CREATED")
    RELEASE_EPOCH=$(tag_epoch "$TAG")
    LEAD_SECONDS=$((RELEASE_EPOCH - CREATED_EPOCH))
    if [ "$LEAD_SECONDS" -ge 0 ]; then
      LEAD_TIME=$(format_duration $LEAD_SECONDS)
    fi
  fi

  TABLE_ROWS="${TABLE_ROWS}| #${ISSUE_NUM} | ${TITLE} | ${LEAD_TIME} | ${CYCLE_TIME} | ${ISSUE_COMMITS} |
"
done

# Compose the discussion body
BODY="## Release Velocity: ${TAG}

**Released:** ${RELEASE_DATE}
**Commits:** ${COMMIT_COUNT} (since ${PREV_TAG})
**Issues closed:** ${ISSUE_COUNT}

### Metrics

| Issue | Title | Lead Time | Cycle Time | Commits |
|-------|-------|-----------|------------|---------|
${TABLE_ROWS}
**Lead time** = issue created → included in release
**Cycle time** = first commit → last commit for the issue

### Definitions

- **Lead time** measures the full idea-to-delivery pipeline. Shorter lead times mean faster response to reported issues.
- **Cycle time** measures active development duration. The gap between lead and cycle time reveals queue/wait time.

### Release Notes

https://github.com/CalcMark/go-calcmark/releases/tag/${TAG}"

TITLE="Release Velocity: ${TAG}"

# GraphQL IDs
REPO_ID="R_kgDOQSei8w"
CATEGORY_ID="DIC_kwDOQSei884C4Bnr"  # Announcements

# Post the discussion
RESULT=$(gh api graphql -f query='
  mutation($repoId: ID!, $categoryId: ID!, $title: String!, $body: String!) {
    createDiscussion(input: {
      repositoryId: $repoId
      categoryId: $categoryId
      title: $title
      body: $body
    }) {
      discussion {
        url
        number
      }
    }
  }
' -f repoId="$REPO_ID" -f categoryId="$CATEGORY_ID" -f title="$TITLE" -f body="$BODY")

DISCUSSION_URL=$(echo "$RESULT" | jq -r '.data.createDiscussion.discussion.url')
DISCUSSION_NUM=$(echo "$RESULT" | jq -r '.data.createDiscussion.discussion.number')

echo "Posted Discussion #${DISCUSSION_NUM}: ${TITLE}"
echo "  ${DISCUSSION_URL}"
