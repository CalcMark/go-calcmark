#!/usr/bin/env bash
# helpers.sh — Shared functions for metric scripts.
# Source this file: source "$(dirname "$0")/helpers.sh"

# Parse ISO 8601 timestamp to epoch seconds (portable: macOS + Linux).
# Handles both Z suffix (2026-03-09T15:12:06Z) and offset (2026-03-09T09:12:06-06:00).
parse_iso_epoch() {
  local ts="$1"
  local ts_clean
  ts_clean=$(echo "$ts" | sed 's/Z$/+0000/' | sed 's/\([+-][0-9][0-9]\):\([0-9][0-9]\)$/\1\2/')
  date -jf "%Y-%m-%dT%H:%M:%S%z" "$ts_clean" +%s 2>/dev/null || date -d "$ts" +%s
}

# Format seconds as human-readable duration: "3d 14h", "2h 8m", "28m", "5s".
format_duration() {
  local secs=$1
  if [ "$secs" -ge 86400 ]; then
    echo "$((secs / 86400))d $((secs % 86400 / 3600))h"
  elif [ "$secs" -ge 3600 ]; then
    echo "$((secs / 3600))h $((secs % 3600 / 60))m"
  elif [ "$secs" -ge 60 ]; then
    echo "$((secs / 60))m"
  else
    echo "${secs}s"
  fi
}

# Find the earliest semver release tag that contains a given commit.
# Returns the tag name, or empty string if no release contains it yet.
find_release_tag() {
  local commit="$1"
  # Get all semver tags containing this commit, sorted by version ascending
  git tag --contains "$commit" 2>/dev/null \
    | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' \
    | sort -V \
    | head -1
}

# Get the epoch timestamp of a git tag (tagger date for annotated, commit date for lightweight).
tag_epoch() {
  local tag="$1"
  git log -1 --format="%ct" "$tag" 2>/dev/null
}
