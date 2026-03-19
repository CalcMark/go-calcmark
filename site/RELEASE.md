# Release Process

CalcMark uses [GoReleaser](https://goreleaser.com) via GitHub Actions. Pushing a
semver tag (`v*.*.*`) triggers the `release.yml` workflow, which builds platform
binaries, creates a GitHub Release, updates the Homebrew tap, and notifies
calcmark-lark.

## Quick Release

Use the `/release` skill in Claude Code:

```bash
/release patch   # Bug fixes
/release minor   # New features
/release major   # Breaking changes
```

The skill runs tests, shows the changelog, tags, and pushes — with confirmation
at every step.

## Manual Release

```bash
# 1. Ensure you're on main with a clean tree
git checkout main && git pull
git status  # must be clean

# 2. Run quality gates
task test
task quality

# 3. Tag
git tag -a v1.8.17 -m "Release v1.8.17"

# 4. Push the tag (triggers GitHub Actions)
git push origin v1.8.17
```

## Pre-release Versions

Pre-release tags (e.g., `v2.0.0-alpha.1`, `v1.9.0-rc.1`) are tagged manually.
GoReleaser marks them as pre-release on GitHub automatically. The `/release`
skill does not support pre-release tags.

```bash
git tag -a v2.0.0-alpha.1 -m "v2.0.0-alpha.1"
git push origin v2.0.0-alpha.1
```

Pre-release tags do **not** trigger the calcmark-lark notification (the workflow
filters on tags containing `-`).

## Rolling Back a Release

**Always forward-fix. Never delete tags or releases.**

Deleting a tag and re-releasing is unreliable:

- Homebrew bottles are cached — users who already ran `brew upgrade` have the
  bad version and won't re-fetch.
- The Go module proxy caches tagged versions permanently.
- GitHub release URLs may be cached by CDNs and CI systems.
- Lark has already been notified and may have pulled the release.

### Forward-Fix Procedure

If a release has a critical bug:

```bash
# 1. Revert the bad commit(s) on main
git checkout main && git pull
git revert <bad-commit-sha>      # or fix the bug directly

# 2. Run the full quality gate
task test
task quality

# 3. Tag the next patch version
git tag -a v1.8.18 -m "Release v1.8.18 — reverts broken change in v1.8.17"
git push origin v1.8.18
```

This produces a new release that Homebrew, Go modules, and Lark all pick up
through their normal update paths. Users on the bad version get the fix on their
next `brew upgrade`.

### Timeline

| Step | Time |
|------|------|
| Push tag | 0 min |
| GitHub Actions builds release | ~3 min |
| Homebrew tap updated | ~3 min (same workflow) |
| `brew upgrade` available | immediately after tap push |
| Go module proxy caches new tag | ~5 min |
| Lark notified | ~3 min (repository dispatch) |

A forward-fix is fully propagated within 5 minutes of pushing the tag.

### When Users Report the Bad Version

Tell them:

```bash
brew update && brew upgrade calcmark/tap/calcmark
```

Or point them to the GitHub release page to download the fixed binary directly.

### What NOT to Do

- **Don't delete the tag** — `git push --delete origin v1.8.17` leaves a gap in
  the version history and breaks anyone pinned to that version.
- **Don't delete the GitHub release** — cached URLs return 404, breaking CI
  pipelines that reference the release.
- **Don't force-push the tag** — `git tag -f` + `git push -f origin v1.8.17`
  is the worst option: Go module proxy has already cached the old content under
  that version, so `go install` fetches the broken code forever.
- **Don't amend the tagged commit** — same problem as force-pushing.

## Troubleshooting

### Tag already exists locally

```bash
git tag -d v1.8.17          # delete local tag
# Then re-run /release
```

### Tag already exists on remote

If you pushed a tag but the release workflow failed (not a code bug — a CI
issue), you can delete and re-push:

```bash
git push --delete origin v1.8.17
git tag -d v1.8.17
# Fix the CI issue, then re-run /release
```

Only do this if the release **never published** (no binaries, no Homebrew
update). If binaries were published, forward-fix instead.

### GoReleaser fails

Check the [Actions tab](https://github.com/CalcMark/go-calcmark/actions) for
the failed run. Common causes:

- **`HOMEBREW_TAP_GITHUB_TOKEN` expired** — regenerate the PAT with `repo`
  scope and update the secret.
- **`LARK_DISPATCH_TOKEN` expired** — same process.
- **Go version mismatch** — `go-version-file: 'go.mod'` should handle this
  automatically.
