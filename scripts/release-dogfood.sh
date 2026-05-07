#!/usr/bin/env bash
#
# release-dogfood.sh — verify a candidate go-calcmark change against cmw's
# full test suite BEFORE tagging.
#
# Why this exists: tagging v2.2.2 from a green go-calcmark suite shipped a
# regression that cmw's `task check` would have caught immediately
# (TestLSPDispatcher_AtGlobalsDot_ReturnsGlobalsFields regressed because
# the LSP returned no completions for in-flight `@globals.`). Three
# follow-up patch tags later (v2.2.3, v2.2.4), the version history is
# uglier than it needed to be. This script encodes the "dogfood the
# downstream first" lesson into the workflow.
#
# Reliability contract — what this script guarantees:
#
#   1. **No flaky e2e.** It runs cmw's `task check`, which is the
#      pre-push gate (fmt + lint + build + unit + Go API + tdd-gate).
#      `task test:e2e` (Playwright) is intentionally NOT invoked —
#      that's the manual surface and the only known flaky path.
#
#   2. **Tests against the LOCAL working tree, not a published tag.**
#      Uses a `replace` directive in cmw's go.mod pointing at the
#      go-calcmark working directory you invoke this from. The tag
#      doesn't need to exist yet; in fact, this script is meant to
#      run BEFORE you tag.
#
#   3. **Always restores cmw's go.mod / go.sum on exit.** Whether the
#      check passes, fails, or you Ctrl-C in the middle, an EXIT trap
#      runs `git checkout -- go.mod go.sum` so the cmw working tree
#      is left exactly as you found it.
#
#   4. **Refuses to run on a dirty cmw working tree.** If cmw already
#      has uncommitted go.mod or go.sum changes, the trap-based
#      restore would clobber them. Bail with an actionable message
#      instead.
#
# Configuration (env vars):
#   CMW_PATH        Absolute path to a cmw checkout. Defaults to
#                   `$(dirname go-calcmark)/cmw` (i.e., a sibling
#                   directory at `../cmw`).
#
# Exit codes:
#   0  cmw passes against the local go-calcmark — safe to tag.
#   1  cmw fails — do NOT tag. (Or precondition not met; the message
#      prints which it is.)
#
# Usage:
#   task release:dogfood
#   CMW_PATH=/custom/path/to/cmw task release:dogfood

set -euo pipefail

# ── Resolve paths ────────────────────────────────────────────────────────────

GO_CALCMARK_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CMW_PATH="${CMW_PATH:-$(cd "$GO_CALCMARK_PATH/.." && pwd)/cmw}"

# ── Preconditions ────────────────────────────────────────────────────────────

if [ ! -d "$CMW_PATH" ]; then
	echo "✖ cmw checkout not found at: $CMW_PATH"
	echo
	echo "  Either:"
	echo "    - clone cmw at \$(dirname go-calcmark)/cmw"
	echo "    - or set CMW_PATH=/path/to/cmw"
	exit 1
fi

if [ ! -f "$CMW_PATH/go.mod" ]; then
	echo "✖ no go.mod at $CMW_PATH/go.mod — is this really a cmw checkout?"
	exit 1
fi

if [ ! -f "$CMW_PATH/Taskfile.yml" ]; then
	echo "✖ no Taskfile.yml at $CMW_PATH — is this really a cmw checkout?"
	exit 1
fi

# Refuse to run on a dirty cmw tree (we can't restore over uncommitted changes).
if ! (cd "$CMW_PATH" && git diff --quiet go.mod go.sum 2>/dev/null); then
	echo "✖ cmw has uncommitted changes to go.mod or go.sum"
	echo
	echo "  We restore these from the index after the check, which would clobber"
	echo "  your in-progress edits. Commit or stash them in cmw first:"
	echo
	echo "    cd $CMW_PATH"
	echo "    git status"
	exit 1
fi

# ── Restore cmw's go.mod / go.sum on any exit ───────────────────────────────
#
# Defensive: trap fires on success, failure, or signal. The `git checkout --`
# restores from the index, which is exactly what cmw had committed before
# we started. Combined with the dirty-check above, this leaves cmw clean.

cleanup() {
	local exit_code=$?
	if [ -d "$CMW_PATH" ]; then
		(cd "$CMW_PATH" && git checkout -- go.mod go.sum 2>/dev/null) || true
		echo "→ cmw go.mod / go.sum restored"
	fi
	exit $exit_code
}
trap cleanup EXIT

# ── Bump cmw to the local go-calcmark via replace ───────────────────────────

echo "→ pointing cmw at local go-calcmark"
echo "    go-calcmark: $GO_CALCMARK_PATH"
echo "    cmw:         $CMW_PATH"

cd "$CMW_PATH"
go mod edit -replace="github.com/CalcMark/go-calcmark/v2=$GO_CALCMARK_PATH"
go mod tidy

# ── Run cmw's full check ─────────────────────────────────────────────────────
#
# `task check` is cmw's pre-push gate. From cmw/CLAUDE.md:
#   "task check (full pre-push gate — fmt + lint + build + test + hook smoke tests)"
# It deliberately omits Playwright e2e tests, which are the known flaky surface.

echo
echo "→ running cmw task check against local go-calcmark"
echo

if task check; then
	echo
	echo "✓ cmw passes against the local go-calcmark — safe to tag the release"
	exit 0
fi

echo
echo "✖ cmw failed task check against the local go-calcmark"
echo
echo "  Do NOT tag. Investigate the failures above. Common causes:"
echo "    - exported API changed in a way cmw consumes incorrectly"
echo "    - LSP behaviour shift surfaced in cmw's integration tests"
echo "    - go-calcmark dependency added that cmw's go.mod doesn't have"
echo
echo "  If you suspect a flake, re-run \`task release:dogfood\`. We do NOT"
echo "  retry inside this script — flakes are bugs to fix, not noise to mask."
exit 1
