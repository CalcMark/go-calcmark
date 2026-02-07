# Phase 10: Preview Pane - Research

**Researched:** 2026-02-06
**Domain:** TUI rendering, result formatting, side-by-side layout
**Confidence:** HIGH

## Summary

Phase 10 is a refinement phase that enhances the existing preview pane rendering rather than building from scratch. The codebase already has:
- A working two-pane side-by-side layout (`sidebyside.go`)
- Line alignment computation (`aligned.go`, `AlignedModel`)
- Result formatting (`format/display/display.go`)
- Error display infrastructure (`components/errors.go`, `contextfooter.go`)

The phase focuses on:
1. Adjusting width ratios from current 55/45 to 60/40
2. Changing preview header from "Preview" to "Results"
3. Enhancing result formatting with tilde prefix for napkin estimates
4. Adding locale-aware thousand separators
5. Unifying currency display logic
6. Improving error cascade handling

**Primary recommendation:** Extend existing rendering infrastructure rather than rebuild. The `display.Format()` function and `AlignedModel` computation are the main extension points.

## Standard Stack

### Core (Already in Use)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| charmbracelet/lipgloss | latest | TUI styling | Already used throughout codebase |
| charmbracelet/bubbletea | latest | TUI framework | Existing Model/Update/View pattern |
| mattn/go-runewidth | latest | Unicode width | Already in geometry.go |
| shopspring/decimal | latest | Precise math | Core type system uses it |

### Supporting (Already in Use)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| charmbracelet/glamour | latest | Markdown rendering | TextBlock preview |
| cockroachdb/datadriven | latest | Test framework | Catwalk tests |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom wrapping | lipgloss.Width | Use lipgloss - already handles ANSI codes |
| locale package | golang.org/x/text/language | Could add for proper i18n, but overkill for now |

**Installation:** No new dependencies required.

## Architecture Patterns

### Existing Project Structure
```
cmd/calcmark/tui/
├── editor/
│   ├── view.go              # Main rendering (View() method)
│   ├── aligned.go           # AlignedModel computation
│   ├── sidebyside.go        # Two-pane layout helper
│   ├── results.go           # LineResult extraction
│   └── model.go             # Editor state
├── components/
│   ├── contextfooter.go     # Error display
│   └── errors.go            # Error formatting
└── geometry/
    └── geometry.go          # Text wrapping

format/
├── display/
│   ├── display.go           # Format() entry point
│   └── normalize.go         # Unit normalization
└── formatter.go             # Formatter interface
```

### Pattern 1: LineResult to Visual Line
**What:** The existing flow from document to rendered preview:
```
Document -> GetLineResults() -> AlignedModel -> renderCalcLine() -> View
```
**When to use:** All preview rendering
**Example:**
```go
// Source: results.go
type LineResult struct {
    LineNum    int
    Source     string
    IsCalc     bool
    VarName    string    // "" for anonymous calculations
    Value      string    // display.Format(result)
    Error      string
    Diagnostic *document.Diagnostic
    WasChanged bool
}
```

### Pattern 2: Centralized Formatting
**What:** All value display goes through `display.Format()`
**When to use:** Whenever converting a types.Type to string for display
**Example:**
```go
// Source: format/display/display.go
func Format(t types.Type) string {
    switch v := t.(type) {
    case *types.Number:
        return FormatNumber(v.Value)
    case *types.Quantity:
        return FormatQuantity(v)
    case *types.Currency:
        return FormatCurrency(v)
    // ...
    }
}
```

### Pattern 3: AlignedModel for Sync Scroll
**What:** Pre-compute visual line structure to ensure source/preview alignment
**When to use:** Whenever both panes must scroll together
**Example:**
```go
// Source: aligned.go
type AlignedModel struct {
    SourceLines    []AlignedLine
    PreviewLines   []AlignedLine
    SourceToVisual map[int]int  // source line -> first visual line
}
```

### Anti-Patterns to Avoid
- **Computing layout per-render:** Use AlignedModel cache to avoid cycles
- **Mixing styling into model:** Keep lipgloss styles in config/theme.go
- **Hard-coding colors:** Use theme config for all colors (errors, values, etc.)

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Text wrapping | Custom char-by-char | geometry.WrapText() | Handles unicode, CJK |
| Visual width | len(string) | lipgloss.Width() | Handles ANSI codes |
| ANSI reset handling | Manual string ops | stripResetCodes() in sidebyside.go | Already debugged |
| Line alignment | Manual counting | ComputeAlignedModel() | Handles wrapping both panes |
| Decimal formatting | float64 | shopspring/decimal | Precision requirements |

**Key insight:** Most of the hard problems (alignment, wrapping, ANSI handling) are already solved. The phase is about format string changes and ratio adjustments.

## Common Pitfalls

### Pitfall 1: ANSI Code Leakage
**What goes wrong:** Background colors bleed across pane boundaries
**Why it happens:** ANSI reset codes (\x1b[0m) clear all formatting including backgrounds
**How to avoid:** Use `stripResetCodes()` in sidebyside.go (already implemented)
**Warning signs:** Terminal default color showing through between panes

### Pitfall 2: Width Calculation Mismatch
**What goes wrong:** Text overflows pane or leaves gaps
**Why it happens:** Using `len(string)` instead of `lipgloss.Width()` for styled content
**How to avoid:** Always use lipgloss.Width for any styled string
**Warning signs:** Divider position varies across lines

### Pitfall 3: AlignedModel Cache Invalidation
**What goes wrong:** Preview doesn't update when content changes
**Why it happens:** Cache key doesn't include all relevant inputs (e.g., editBuf)
**How to avoid:** Include editBuf in computeCacheKey() (already done)
**Warning signs:** Stale preview content after typing

### Pitfall 4: Glamour Empty Line Stripping
**What goes wrong:** Empty lines between blocks disappear in preview
**Why it happens:** glamour (markdown renderer) strips empty lines
**How to avoid:** Use line-by-line rendering with explicit empty line handling
**Warning signs:** Content blocks appear merged together

### Pitfall 5: Cascading Error Display
**What goes wrong:** Same error repeated on every dependent line
**Why it happens:** Each dependent line evaluates and fails with same root cause
**How to avoid:** Track root cause, show "blocked" for dependents
**Warning signs:** Screen filled with repeated error messages

## Code Examples

Verified patterns from existing codebase:

### Result Display (Current Implementation)
```go
// Source: view.go:822-880
func (m Model) renderCalcLine(r LineResult, width int) string {
    if r.Error != "" && isActuallyCalc {
        errHint := extractErrorHint(r.Error, width-4)
        return errStyle.Render("⚠ " + errHint)
    }

    if r.VarName != "" {
        // "varName → value"
        return m.styles.CalcVarName.Render(r.VarName) + " " +
               m.styles.CalcArrow.Render("→") + " " +
               valueStyle.Render(r.Value)
    }
    // Anonymous calculation - just show value
    return valueStyle.Render(r.Value)
}
```

### Width Ratio Configuration (Current)
```go
// Source: model.go:98-102
var DefaultPaneWidths = map[PreviewMode]PaneWidthConfig{
    PreviewFull:    {SourcePercent: 55, PreviewPercent: 45},  // Change to 60/40
    PreviewMinimal: {SourcePercent: 75, PreviewPercent: 25},
    PreviewHidden:  {SourcePercent: 100, PreviewPercent: 0},
}
```

### Thousand Separator (To Be Added)
```go
// Pattern for locale-aware formatting (not yet implemented)
import "golang.org/x/text/language"
import "golang.org/x/text/message"

func FormatNumberLocale(value decimal.Decimal, locale language.Tag) string {
    p := message.NewPrinter(locale)
    f, _ := value.Float64()
    return p.Sprintf("%v", f)  // Uses locale-specific separators
}
```

### Napkin Tilde Prefix (Needed)
```go
// Pattern for napkin estimate display
func FormatQuantity(q *types.Quantity) string {
    if q.IsNapkin() {  // Flag to be added or detected
        return "~" + formatWithSuffix(q.Value, q.Unit)
    }
    // ... existing logic
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Modal editing | Non-modal editing | Phase 9.1 | Always editable |
| Per-line rendering | AlignedModel | Recent | Stable alignment |
| Hard-coded colors | Theme config | Recent | User customizable |

**Deprecated/outdated:**
- None identified - current codebase is modern

## Open Questions

Things that couldn't be fully resolved:

1. **Napkin Estimate Detection**
   - What we know: `as napkin` conversion exists in AST
   - What's unclear: How to propagate "is napkin" flag through evaluation to display
   - Recommendation: Add IsNapkin field to types or use NapkinConversion result type

2. **Locale Configuration**
   - What we know: golang.org/x/text/language provides locale support
   - What's unclear: Where to store user's locale preference
   - Recommendation: Add to ThemeConfig or create separate LocaleConfig

3. **Currency Symbol Positioning**
   - What we know: Current code has SymbolToCode/CodeToSymbol maps
   - What's unclear: Whether $ goes before or after varies by locale
   - Recommendation: Use golang.org/x/text/currency for proper i18n when needed

4. **Cascading Error Root Cause**
   - What we know: Diagnostics have Code and Message
   - What's unclear: How to detect "this error is caused by another line's error"
   - Recommendation: Track undefined variables from prior errors in evaluation context

## Sources

### Primary (HIGH confidence)
- `/Users/bitsbyme/projects/go-calcmark/cmd/calcmark/tui/editor/view.go` - Main rendering
- `/Users/bitsbyme/projects/go-calcmark/cmd/calcmark/tui/editor/aligned.go` - Alignment computation
- `/Users/bitsbyme/projects/go-calcmark/cmd/calcmark/tui/editor/sidebyside.go` - Two-pane layout
- `/Users/bitsbyme/projects/go-calcmark/format/display/display.go` - Value formatting
- `/Users/bitsbyme/projects/go-calcmark/cmd/calcmark/config/theme.go` - Style configuration

### Secondary (MEDIUM confidence)
- `/Users/bitsbyme/projects/go-calcmark/format/OUTPUT_FORMATTERS.md` - Formatter architecture
- `/Users/bitsbyme/projects/go-calcmark/cmd/calcmark/tui/editor/TESTING.md` - Test patterns

### Tertiary (LOW confidence)
- golang.org/x/text/language documentation for locale handling (not yet in codebase)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries already in use
- Architecture: HIGH - Patterns documented from existing code
- Pitfalls: HIGH - Observed in test files and comments

**Research date:** 2026-02-06
**Valid until:** 60 days (stable codebase, no fast-moving dependencies)
