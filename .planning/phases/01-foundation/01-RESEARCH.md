# Phase 1: Foundation - Research

**Researched:** 2026-02-02
**Domain:** Go project infrastructure, pure geometry extraction, dependency management
**Confidence:** HIGH

## Summary

Phase 1 addresses three foundational problems: (1) the CI release workflow pins Go 1.21 while go.mod requires 1.24.4, guaranteeing build failures on release; (2) geometry computation for two-column alignment is deeply embedded in the TUI editor package with lipgloss dependencies, making it impossible to test without the full TUI stack; and (3) dependencies need updating and `adrg/frontmatter` needs to be added for future YAML front matter parsing.

The core technical challenge is extracting a `geometry` package with **zero TUI framework dependencies**. The existing `WrapText` function in `linemodel.go` uses `lipgloss.Width()` for unicode width measurement. The geometry package must use `mattn/go-runewidth` (already an indirect dependency via lipgloss) directly instead. The `CalculateRowGeometry` algorithm from `code.sh` uses `muesli/reflow/wordwrap` (also already an indirect dependency), but implementing wrapping with `go-runewidth.StringWidth` is better because it avoids pulling in the full reflow library as a direct dependency and matches the existing `WrapText` approach.

The existing codebase already has a hand-rolled frontmatter parser in `spec/document/frontmatter.go` that works correctly. The requirement FOUND-05 says to add `adrg/frontmatter` to go.mod -- this is for Phase 6 (YAML front matter) but should be added now to validate dependency compatibility. The existing hand-rolled parser will likely be replaced or augmented in Phase 6.

**Primary recommendation:** Extract geometry functions to `cmd/calcmark/tui/geometry/` package using `go-runewidth` for width calculation (already an indirect dep, promotes to direct), keeping zero dependency on lipgloss/bubbletea/bubbles.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.24.12 | Language runtime | Latest 1.24.x patch (released 2026-01-15). Security fixes in crypto/tls, net/url, archive/zip. go.mod currently says 1.24.4; update to 1.24.12. |
| mattn/go-runewidth | v0.0.16 | Unicode-aware string width | Already an indirect dep. Provides `StringWidth()` and `RuneWidth()` for CJK, emoji, etc. Replaces `lipgloss.Width()` in pure geometry code. |
| actions/setup-go | v5 | CI Go installation | Current in workflow. Use `go-version-file: 'go.mod'` instead of hardcoded version. |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| adrg/frontmatter | latest | YAML front matter parsing | Add to go.mod now (FOUND-05). Actual integration in Phase 6. Uses gopkg.in/yaml.v3 internally (already a dep). |
| spf13/cobra | v1.10.2 | CLI framework | Update from v1.10.1. Minor bug fixes. |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| mattn/go-runewidth (direct) | lipgloss.Width() | lipgloss.Width is a thin wrapper around runewidth. Using runewidth directly avoids TUI framework dependency in geometry package. |
| muesli/reflow/wordwrap | Custom WrapText with runewidth | code.sh uses wordwrap but existing codebase has custom WrapText. Custom approach is already tested and handles edge cases (CJK, emoji). Keep custom approach. |
| go-version: '1.24.12' | go-version-file: 'go.mod' | `go-version-file` is better -- auto-tracks go.mod, no future drift. |

**Installation:**
```bash
# Promote indirect dep to direct
go get github.com/mattn/go-runewidth@v0.0.16

# Add new dependency
go get github.com/adrg/frontmatter

# Update existing dependencies
go get github.com/spf13/cobra@v1.10.2
go get golang.org/x/text@latest
go get golang.org/x/net@latest
go get golang.org/x/sys@latest
go get golang.org/x/term@latest

# Tidy
go mod tidy
```

## Architecture Patterns

### Recommended Project Structure
```
cmd/calcmark/tui/
├── geometry/             # NEW: Pure geometry computation (zero TUI deps)
│   ├── geometry.go       # CalculateRowGeometry, WrapText
│   ├── geometry_test.go  # Comprehensive unit tests
│   └── doc.go            # Package documentation
├── editor/               # TUI editor (imports geometry)
│   ├── aligned.go        # Uses geometry.WrapText instead of local WrapText
│   ├── linemodel.go      # Uses geometry.WrapText instead of local WrapText
│   ├── view.go           # Rendering (still uses lipgloss for styling)
│   └── ...
├── components/           # Shared TUI components
├── repl/                 # REPL model
└── shared/               # Shared types
```

### Pattern 1: Pure Geometry Functions
**What:** Functions that take dimensions and content as plain strings/ints and return layout data as plain structs. No lipgloss, no bubbletea, no terminal concepts.
**When to use:** All geometry computation -- wrapping, alignment, row height calculation.
**Example:**
```go
// Package geometry provides pure layout computation for two-column rendering.
// It has zero dependencies on TUI frameworks (lipgloss, bubbletea, bubbles).
package geometry

import "github.com/mattn/go-runewidth"

// RowGeometry describes the visual layout of a single logical row
// when rendered in a two-column layout with text wrapping.
type RowGeometry struct {
    Height     int      // Number of visual lines needed (max of left, right)
    LeftLines  []string // Left column visual lines (padded to Height)
    RightLines []string // Right column visual lines (padded to Height)
}

// CalculateRowGeometry computes the visual layout for a single logical row.
// srcLine: source text for the left column
// resultContent: result text for the right column (empty string if no result)
// leftWidth: available width for left column
// rightWidth: available width for right column
func CalculateRowGeometry(srcLine, resultContent string, leftWidth, rightWidth int) RowGeometry {
    leftWrapped := WrapText(srcLine, leftWidth)

    var rightWrapped []string
    if resultContent != "" {
        rightWrapped = WrapText(resultContent, rightWidth)
    }

    h := len(leftWrapped)
    if len(rightWrapped) > h {
        h = len(rightWrapped)
    }
    if h == 0 {
        h = 1
    }

    finalLeft := make([]string, h)
    finalRight := make([]string, h)

    for i := 0; i < h; i++ {
        if i < len(leftWrapped) {
            finalLeft[i] = leftWrapped[i]
        }
        if i < len(rightWrapped) {
            finalRight[i] = rightWrapped[i]
        }
    }

    return RowGeometry{Height: h, LeftLines: finalLeft, RightLines: finalRight}
}

// WrapText wraps text to fit within maxWidth using unicode-aware width calculation.
// Returns a slice of strings, each fitting within maxWidth.
// Uses runewidth.StringWidth for correct CJK, emoji, and combining character handling.
func WrapText(text string, maxWidth int) []string {
    if maxWidth <= 0 {
        return []string{text}
    }
    if len(text) == 0 {
        return []string{""}
    }
    if runewidth.StringWidth(text) <= maxWidth {
        return []string{text}
    }

    // Word-boundary-aware wrapping with hard-break fallback
    // (port from editor/linemodel.go WrapText, replacing lipgloss.Width with runewidth.StringWidth)
    // ...
}
```

### Pattern 2: Dependency Direction for Geometry
**What:** geometry package is imported by editor, never the reverse. geometry has zero imports from cmd/calcmark/tui/editor.
**When to use:** Always. This is the core architectural constraint.
**Dependency graph:**
```
geometry (pure: mattn/go-runewidth only)
    ^
    |
editor (TUI: lipgloss, bubbletea, bubbles, glamour + geometry)
```

### Pattern 3: Table-Driven Tests for Geometry
**What:** All geometry functions tested with table-driven tests covering edge cases.
**When to use:** Every geometry function.
**Example:**
```go
func TestCalculateRowGeometry(t *testing.T) {
    tests := []struct {
        name       string
        srcLine    string
        result     string
        leftWidth  int
        rightWidth int
        wantHeight int
        wantLeft   []string
        wantRight  []string
    }{
        {
            name:       "single line both sides",
            srcLine:    "hello",
            result:     "world",
            leftWidth:  10,
            rightWidth: 10,
            wantHeight: 1,
            wantLeft:   []string{"hello"},
            wantRight:  []string{"world"},
        },
        {
            name:       "left wraps right does not",
            srcLine:    "this is a longer line that wraps",
            result:     "short",
            leftWidth:  10,
            rightWidth: 10,
            wantHeight: 4, // depends on exact wrapping
        },
        {
            name:       "right wraps left does not",
            srcLine:    "short",
            result:     "1234567890",
            leftWidth:  10,
            rightWidth: 5,
            wantHeight: 2,
        },
        {
            name:       "empty result",
            srcLine:    "hello",
            result:     "",
            leftWidth:  10,
            rightWidth: 10,
            wantHeight: 1,
            wantLeft:   []string{"hello"},
            wantRight:  []string{""},
        },
        {
            name:       "both wrap asymmetrically",
            srcLine:    "long source content here",
            result:     "even longer result that wraps more than source",
            leftWidth:  10,
            rightWidth: 10,
            wantHeight: 5, // right wraps to 5 lines
        },
    }
    // ...
}
```

### Anti-Patterns to Avoid
- **Importing lipgloss in geometry:** The entire point of extraction is zero TUI deps. Use `runewidth.StringWidth` not `lipgloss.Width`.
- **Duplicating WrapText:** Do not keep WrapText in both geometry and editor. Editor must import from geometry. The editor's `linemodel.go` WrapText should be removed and replaced with `geometry.WrapText`.
- **Testing geometry through the TUI:** Geometry tests must not require bubbletea message passing or terminal dimensions. Pure in, pure out.
- **Circular imports:** geometry must not import anything from editor. If geometry needs a type defined in editor, move the type to geometry or to a shared package.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Unicode string width measurement | Custom rune width tables | `mattn/go-runewidth` v0.0.16 | CJK double-width, combining characters, emoji zwj sequences. Thousands of edge cases. |
| YAML front matter parsing | Manual `---` delimiter parsing (exists in frontmatter.go) | `adrg/frontmatter` for Phase 6 | Current hand-rolled parser works but adrg/frontmatter handles TOML/JSON front matter too and is better tested. For now, just add the dependency. |
| CI Go version management | Hardcoded `go-version: '1.21'` | `go-version-file: 'go.mod'` | Never drifts. Single source of truth. |
| Word wrapping | Simple `strings.Split` + length | Port existing `WrapText` from `linemodel.go` with `runewidth.StringWidth` | Must handle word boundaries, hard breaks when no space exists, and unicode width correctly. The existing implementation handles all of this. |

**Key insight:** The geometry extraction is a refactoring, not a rewrite. The algorithms already exist in `linemodel.go` (`WrapText`, `ComputeLineModel`) and `aligned.go` (`ComputeAlignedModel`). The task is to extract `WrapText` and `CalculateRowGeometry` to a separate package, replacing `lipgloss.Width` with `runewidth.StringWidth`, and updating imports.

## Common Pitfalls

### Pitfall 1: lipgloss.Width vs runewidth.StringWidth Behavioral Differences
**What goes wrong:** `lipgloss.Width` strips ANSI escape codes before measuring width. `runewidth.StringWidth` does not.
**Why it happens:** lipgloss.Width is designed for styled terminal strings. runewidth is designed for plain text.
**How to avoid:** The geometry package works with **plain text only** (no ANSI codes). This matches its purpose -- geometry computation happens before styling. Ensure all inputs to geometry functions are plain strings, never styled strings.
**Warning signs:** If geometry functions receive strings containing `\x1b[`, something is wrong. The geometry layer should never see ANSI escape codes.

### Pitfall 2: go.mod Go Version vs CI Go Version Drift
**What goes wrong:** go.mod says `go 1.24.4`, CI installs Go 1.21, build fails because 1.24 features (like range-over-func) aren't available.
**Why it happens:** Someone updated go.mod but not the CI workflow.
**How to avoid:** Use `go-version-file: 'go.mod'` in GitHub Actions. This reads the version from go.mod automatically.
**Warning signs:** CI failures with "unexpected syntax" or "undefined: ..." errors that pass locally.

### Pitfall 3: Circular Import Between Geometry and Editor
**What goes wrong:** Editor package imports geometry, but geometry needs a type from editor.
**Why it happens:** Types like `LineResult`, `PreviewMode` are defined in editor but used in alignment computation.
**How to avoid:** The geometry package defines its OWN types. `CalculateRowGeometry` uses plain strings and ints, not editor types. The `ComputeAlignedModel` function stays in editor (it uses `LineResult`, `PreviewMode`, render callbacks) and calls `geometry.WrapText` internally.
**Warning signs:** Import cycle compilation errors.

### Pitfall 4: Forgetting to Update All WrapText Callers
**What goes wrong:** WrapText is extracted to geometry but some callers in editor still reference the old local function, causing compilation errors or worse -- a stale duplicate remains.
**Why it happens:** WrapText is called from multiple places in the editor package.
**How to avoid:** After extracting, delete the old `WrapText` from `linemodel.go` and fix all compilation errors. The compiler will find all callers. Search for `WrapText(` in the editor package to verify.
**Warning signs:** Two definitions of WrapText in the codebase.

### Pitfall 5: Existing Test Failures Blocking Phase Completion
**What goes wrong:** Three editor tests currently fail (`TestEditorCatwalk`, `TestViewportDoesNotExceedHeight`, `TestViewportHeightWithLargeContent`). These are viewport height bugs unrelated to geometry extraction.
**Why it happens:** The editor renders 25 lines for a 24-line terminal height, causing terminal bleed-through.
**How to avoid:** Phase 1 success criteria says `task test` must pass. These pre-existing failures must be fixed or the tests must be marked as known issues. Recommendation: fix the viewport height bug (off-by-one in content height calculation) as part of this phase since it blocks the success criteria.
**Warning signs:** `task test` fails even after all Phase 1 changes.

### Pitfall 6: adrg/frontmatter Replacing Existing Frontmatter Parser Prematurely
**What goes wrong:** Adding adrg/frontmatter and immediately replacing the hand-rolled parser in `spec/document/frontmatter.go` breaks the existing frontmatter tests and behavior.
**Why it happens:** The requirement says "adrg/frontmatter added" but the actual migration is Phase 6 work.
**How to avoid:** Phase 1 only adds the dependency to go.mod. It does NOT replace the existing parser. The existing `ParseFrontmatter` function in `spec/document/frontmatter.go` continues to work as-is.
**Warning signs:** Changes to `spec/document/frontmatter.go` in Phase 1.

## Code Examples

Verified patterns from the existing codebase:

### Existing WrapText (to be ported)
```go
// Source: cmd/calcmark/tui/editor/linemodel.go:238-300
// This is the function to port to geometry package.
// Replace lipgloss.Width() calls with runewidth.StringWidth().
func WrapText(text string, maxWidth int) []string {
    if maxWidth <= 0 {
        return []string{text}
    }
    if len(text) == 0 {
        return []string{""}
    }
    if runewidth.StringWidth(text) <= maxWidth { // was: lipgloss.Width(text)
        return []string{text}
    }
    // ... rest of wrapping logic unchanged, just swap width function
}
```

### CalculateRowGeometry from code.sh (to be implemented)
```go
// Source: code.sh lines 42-77
// This is the algorithm to implement. Adapted from bash/Go prototype.
// Uses WrapText instead of wordwrap.String for consistency with existing codebase.
func CalculateRowGeometry(srcLine, resultContent string, leftW, rightW int) RowGeometry {
    leftWrapped := WrapText(srcLine, leftW)

    var rightWrapped []string
    if resultContent != "" {
        rightWrapped = WrapText(resultContent, rightW)
    }

    h := len(leftWrapped)
    if len(rightWrapped) > h {
        h = len(rightWrapped)
    }
    if h == 0 {
        h = 1
    }

    finalLeft := make([]string, h)
    finalRight := make([]string, h)
    for i := 0; i < h; i++ {
        if i < len(leftWrapped) {
            finalLeft[i] = leftWrapped[i]
        }
        if i < len(rightWrapped) {
            finalRight[i] = rightWrapped[i]
        }
    }

    return RowGeometry{Height: h, LeftLines: finalLeft, RightLines: finalRight}
}
```

### CI Workflow Fix
```yaml
# Source: .github/workflows/release.yml
# BEFORE (broken):
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.21'

# AFTER (correct):
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version-file: 'go.mod'
```

### go.mod Updates
```go
// Current:
go 1.24.4

// Updated:
go 1.24.12
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Hardcoded `go-version` in CI | `go-version-file: 'go.mod'` | actions/setup-go v4+ | No version drift between go.mod and CI |
| `lipgloss.Width()` for string width | `runewidth.StringWidth()` in pure packages | N/A (lipgloss wraps runewidth) | Enables pure packages without TUI deps |
| Go 1.24.4 | Go 1.24.12 | 2026-01-15 | Security fixes: crypto/tls, net/url, archive/zip, compiler, runtime, os |
| cobra v1.10.1 | cobra v1.10.2 | Latest stable | Minor bug fixes |

**Deprecated/outdated:**
- `go-version: '1.21'` in release.yml: Must be removed. Go 1.21 cannot build code requiring Go 1.24 features.
- `gopkg.in/yaml.v3`: Marked unmaintained (April 2025). Fork at `go.yaml.in/yaml/v3` exists. BUT: do NOT migrate now -- adrg/frontmatter uses gopkg.in/yaml.v3, and CalcMark already depends on it. Migration is a post-v1 task.

## Open Questions

Things that could not be fully resolved:

1. **Where exactly should the geometry package live?**
   - What we know: Must be under `cmd/calcmark/tui/` since it serves the TUI. Cannot be in `spec/` (spec never depends on impl or TUI).
   - Options: `cmd/calcmark/tui/geometry/` or a top-level `geometry/` package.
   - Recommendation: `cmd/calcmark/tui/geometry/` because it is TUI-specific layout computation. It has no lipgloss dep but the concept of "two-column terminal layout" is TUI-specific. If other consumers (WASM) needed it, it could be promoted later.

2. **Should ComputeAlignedModel also move to geometry?**
   - What we know: `ComputeAlignedModel` in `aligned.go` is a pure function but takes `renderCalcLine` and `renderMarkdown` callbacks that are editor-specific. It also uses `LineResult` and `PreviewMode` types defined in editor.
   - Recommendation: Keep `ComputeAlignedModel` in editor for Phase 1. Extract only `CalculateRowGeometry` and `WrapText`. Phase 2 can evaluate whether to move more functions.

3. **Go 1.24.12 vs 1.25.6?**
   - What we know: Go 1.25.6 is the latest stable. go.mod says `go 1.24.4`. The research summary recommends staying on 1.24.x for v1 release stability.
   - Recommendation: Update to 1.24.12 (latest 1.24.x). Do not jump to 1.25 during this phase -- that is a larger change that could introduce regressions.

4. **Pre-existing test failures**
   - What we know: 3 tests fail in `cmd/calcmark/tui/editor`: `TestEditorCatwalk`, `TestViewportDoesNotExceedHeight`, `TestViewportHeightWithLargeContent`. These are viewport height off-by-one bugs.
   - Recommendation: Fix these as part of Phase 1 since the success criteria requires `task test` to pass. The fix is likely a one-line change in the content height calculation.

## Sources

### Primary (HIGH confidence)
- Codebase analysis: `go.mod`, `Taskfile.yml`, `.github/workflows/release.yml`, `code.sh`, `cmd/calcmark/tui/editor/linemodel.go`, `cmd/calcmark/tui/editor/aligned.go`
- [Go Release History](https://go.dev/doc/devel/release) - verified Go 1.24.12 as latest 1.24.x (released 2026-01-15)
- `mattn/go-runewidth` v0.0.16 API: verified via `go doc` -- provides `StringWidth`, `RuneWidth`, `Wrap`
- `muesli/reflow` v0.3.0: already an indirect dep in go.sum, provides `wordwrap.String` used in code.sh
- [adrg/frontmatter GitHub](https://github.com/adrg/frontmatter) - verified API: `frontmatter.Parse(reader, &struct)`, MIT license, 173 stars

### Secondary (MEDIUM confidence)
- `.planning/research/STACK.md` and `.planning/research/SUMMARY.md` - prior research conducted same day
- `.planning/codebase/STRUCTURE.md` and `.planning/codebase/ARCHITECTURE.md` - codebase analysis docs

### Tertiary (LOW confidence)
- None. All findings verified from codebase or official sources.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all versions verified from go.dev release history, go.mod, and package documentation
- Architecture: HIGH - based on direct codebase analysis of existing code (linemodel.go, aligned.go, view.go) and the code.sh reference
- Pitfalls: HIGH - based on actual current test failures and direct code inspection showing lipgloss.Width usage

**Research date:** 2026-02-02
**Valid until:** 2026-03-04 (30 days -- stable domain, no fast-moving APIs)
