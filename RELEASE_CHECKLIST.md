# Release Checklist

Before the first CalcMark release, complete these setup steps.

## Homebrew Tap Setup (One-Time)

1. **Create the Homebrew tap repository**
   - Go to https://github.com/new
   - Repository name: `homebrew-tap`
   - Owner: `CalcMark` (or your org)
   - Visibility: **Public** (required for Homebrew taps)
   - Initialize with a README

2. **Create a Personal Access Token (PAT)**
   - Go to https://github.com/settings/tokens
   - Generate new token (classic)
   - Name: `homebrew-tap-access`
   - Scopes: check `repo` (full control of private repositories)
   - Copy the token immediately (you won't see it again)

3. **Add the secret to go-calcmark repository**
   - Go to https://github.com/CalcMark/go-calcmark/settings/secrets/actions
   - Click "New repository secret"
   - Name: `HOMEBREW_TAP_GITHUB_TOKEN`
   - Value: paste the PAT from step 2

## Creating a Release

Once Homebrew tap is set up:

```bash
# Ensure you're on main with all changes committed
git checkout main
git pull

# Tag the release (use semantic versioning)
git tag v0.3.0

# Push the tag to trigger the release workflow
git push --tags
```

The GitHub Actions workflow will:
- Build binaries for 7 platforms (macOS, Linux, Windows)
- Generate checksums
- Create GitHub release with artifacts
- Update the Homebrew formula in `calcmark/homebrew-tap`

## Verifying the Release

After the workflow completes:

1. **Check GitHub Releases**
   - Visit https://github.com/CalcMark/go-calcmark/releases
   - Verify all 7 binaries are present
   - Verify checksums.txt exists

2. **Test Homebrew installation** (on macOS/Linux)
   ```bash
   brew tap calcmark/tap
   brew install calcmark
   cm --version
   ```

3. **Test binary download**
   - Download a binary from releases
   - Extract and run `./cm --version`

## Notes

- GoReleaser config: `.goreleaser.yaml`
- Release workflow: `.github/workflows/release.yml`
- Homebrew tap syntax: `brew install calcmark/tap/calcmark`
