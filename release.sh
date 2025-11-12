#!/usr/bin/env bash
# release.sh - Build and publish go-calcmark releases with WASM artifacts
#
# Usage:
#   ./release.sh          # Build and publish to GitHub (requires gh CLI)
#   ./release.sh --local  # Build artifacts only (no GitHub publish)

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse arguments
LOCAL_ONLY=false
if [ "$1" = "--local" ]; then
    LOCAL_ONLY=true
fi

# Extract version from version.go
VERSION=$(grep 'const Version =' version.go | cut -d'"' -f2)
if [ -z "$VERSION" ]; then
    echo -e "${RED}Error: Could not extract version from version.go${NC}"
    exit 1
fi

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  go-calcmark Release v${VERSION}$(printf '%*s' $((22 - ${#VERSION}))║)${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo

# 1. Check for clean working tree
echo -e "${BLUE}[1/6]${NC} Checking working tree..."
if [ -n "$(git status --porcelain)" ]; then
    echo -e "${RED}Error: Working tree is not clean. Commit or stash changes first.${NC}"
    git status --short
    exit 1
fi
echo -e "${GREEN}✓ Working tree is clean${NC}"
echo

# 2. Verify version tag matches
echo -e "${BLUE}[2/6]${NC} Verifying version tag..."
CURRENT_TAG=$(git describe --tags --exact-match 2>/dev/null || echo "")
EXPECTED_TAG="v${VERSION}"

if [ -z "$CURRENT_TAG" ]; then
    echo -e "${RED}Error: HEAD is not tagged. Create tag first:${NC}"
    echo -e "  ${YELLOW}git tag -a \"${EXPECTED_TAG}\" -m \"Release ${EXPECTED_TAG}\"${NC}"
    exit 1
fi

if [ "$CURRENT_TAG" != "$EXPECTED_TAG" ]; then
    echo -e "${RED}Error: Current tag (${CURRENT_TAG}) doesn't match version.go (${EXPECTED_TAG})${NC}"
    echo -e "Update version.go or retag with:"
    echo -e "  ${YELLOW}git tag -d \"${CURRENT_TAG}\"${NC}"
    echo -e "  ${YELLOW}git tag -a \"${EXPECTED_TAG}\" -m \"Release ${EXPECTED_TAG}\"${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Tag ${EXPECTED_TAG} matches version.go${NC}"
echo

# 3. Run tests
echo -e "${BLUE}[3/6]${NC} Running tests..."
if ! go test ./... -v; then
    echo -e "${RED}Error: Tests failed. Fix tests before releasing.${NC}"
    exit 1
fi
echo -e "${GREEN}✓ All tests passed${NC}"
echo

# 4. Build CLI tools
echo -e "${BLUE}[4/6]${NC} Building CLI tools..."
go build -o calcmark ./impl/cmd/calcmark
go build -o cmspec ./spec/cmd/cmspec

# Verify version commands work
CALCMARK_VERSION=$(./calcmark version | awk '{print $3}')
CMSPEC_VERSION=$(./cmspec version | awk '{print $3}')

if [ "$CALCMARK_VERSION" != "$VERSION" ] || [ "$CMSPEC_VERSION" != "$VERSION" ]; then
    echo -e "${RED}Error: Built binaries report wrong version${NC}"
    echo "  calcmark: $CALCMARK_VERSION (expected $VERSION)"
    echo "  cmspec: $CMSPEC_VERSION (expected $VERSION)"
    exit 1
fi
echo -e "${GREEN}✓ CLI tools built (calcmark, cmspec)${NC}"
echo

# 5. Build WASM artifacts
echo -e "${BLUE}[5/6]${NC} Building WASM artifacts..."
RELEASE_DIR="./release-artifacts"
rm -rf "$RELEASE_DIR"
mkdir -p "$RELEASE_DIR"

./calcmark wasm "$RELEASE_DIR"

# Verify artifacts were created
WASM_FILE="$RELEASE_DIR/calcmark-${VERSION}.wasm"
JS_FILE="$RELEASE_DIR/wasm_exec.js"

if [ ! -f "$WASM_FILE" ] || [ ! -f "$JS_FILE" ]; then
    echo -e "${RED}Error: WASM artifacts not created${NC}"
    exit 1
fi

echo -e "${GREEN}✓ WASM artifacts built:${NC}"
echo -e "  • ${WASM_FILE} ($(du -h "$WASM_FILE" | cut -f1))"
echo -e "  • ${JS_FILE} ($(du -h "$JS_FILE" | cut -f1))"
echo

# 6. Publish to GitHub (unless --local)
if [ "$LOCAL_ONLY" = true ]; then
    echo -e "${BLUE}[6/6]${NC} Skipping GitHub publish (--local mode)"
    echo
    echo -e "${YELLOW}Local build complete. To publish:${NC}"
    echo -e "  1. Push tag: ${BLUE}git push origin ${EXPECTED_TAG}${NC}"
    echo -e "  2. Run release: ${BLUE}./release.sh${NC}"
    echo
    echo "Or manually create release at:"
    echo "  https://github.com/CalcMark/go-calcmark/releases/new?tag=${EXPECTED_TAG}"
else
    echo -e "${BLUE}[6/6]${NC} Publishing to GitHub..."

    # Check if gh CLI is installed
    if ! command -v gh &> /dev/null; then
        echo -e "${RED}Error: 'gh' CLI not found${NC}"
        echo "Install with: brew install gh"
        echo "Or run with --local and publish manually"
        exit 1
    fi

    # Check if already released
    if gh release view "$EXPECTED_TAG" &> /dev/null; then
        echo -e "${YELLOW}Warning: Release ${EXPECTED_TAG} already exists on GitHub${NC}"
        read -p "Delete and recreate? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            gh release delete "$EXPECTED_TAG" --yes
        else
            echo "Aborting."
            exit 1
        fi
    fi

    # Push tag if not already on remote
    if ! git ls-remote --tags origin | grep -q "refs/tags/${EXPECTED_TAG}"; then
        echo "Pushing tag ${EXPECTED_TAG} to origin..."
        git push origin "$EXPECTED_TAG"
    fi

    # Create release
    echo "Creating GitHub release..."
    gh release create "$EXPECTED_TAG" \
        --title "v${VERSION}" \
        --generate-notes \
        "$WASM_FILE" \
        "$JS_FILE"

    echo
    echo -e "${GREEN}✓ Release ${EXPECTED_TAG} published to GitHub${NC}"
    echo "View at: https://github.com/CalcMark/go-calcmark/releases/tag/${EXPECTED_TAG}"
fi

# Cleanup
rm -f ./calcmark ./cmspec

echo
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo -e "${GREEN}  Release ${EXPECTED_TAG} Complete!${NC}"
echo -e "${GREEN}════════════════════════════════════════${NC}"
