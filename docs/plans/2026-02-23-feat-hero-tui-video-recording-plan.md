---
title: "feat: Replace hero GIF and add feature demo GIFs"
type: feat
status: completed
date: 2026-02-23
deepened: 2026-02-23
approach: mix-of-demos
---

# Replace Hero GIF and Add Feature Demo GIFs

## Overview

Replace the broken hero GIF with a polished `cm eval` demo, and add smaller feature-specific GIFs to the homepage feature cards. All recordings use scripted VHS tape files for reproducibility.

## Problem Statement

The current hero at `site/static/images/hero.gif` is a 88KB GIF that barely shows anything (dark frame with faint prompt). The homepage has 6 feature cards with text-only descriptions. Neither the hero nor the features demonstrate CalcMark's capabilities visually.

## Proposed Solution: Mix of Demos

### 1. Hero GIF — `cm eval` Budget Demo (Reliable)

A polished `cm eval` demo showing a budget calculation with variables, expressions, natural language functions, and formatted currency output. This is the proven VHS approach (non-interactive) and will produce a small, crisp GIF.

### 2. Feature Card GIFs — Focused TUI Segments

Smaller, focused GIFs showing specific TUI features for relevant feature cards:

| Feature Card | Current Text | Proposed GIF |
|-------------|-------------|-------------|
| "Variables flow downward" | Text only | TUI: type `income = $5000`, then `rent = income * 0.30` — preview updates live |
| "Units are first-class" | Text only | `cm eval`: `5 miles in km` + `20 celsius in fahrenheit` |
| "Export anywhere" | Text only | TUI: Ctrl+E → export overlay with format options |
| "Terminal-native" | Text only | TUI: typing with autocomplete popup appearing |
| "Currencies built in" | Text only | Keep text (or `cm eval` showing `$100 + 50 EUR`) |
| "Markdown is ignored" | Text only | Keep text (concept is better explained verbally) |

Not every card needs a GIF — only where animation adds understanding. Start with 2-3 feature GIFs and the hero.

## Technical Approach

### VHS Tape Files

All recordings use scripted `.tape` files for reproducibility. VHS CAN drive Bubble Tea TUI apps (the old project docs were wrong). Key commands:

- `Type`, `Enter`, `Tab`, `Ctrl+E` — keystroke sending
- `Wait+Screen@5s /pattern/` — wait for TUI to render (v0.9.0+)
- `Hide`/`Show` — hide setup from recording
- `Sleep` — natural pauses

### VHS TUI Rendering Caveats

- **Issue #412**: Inconsistent Bubble Tea rendering (blank viewport). Run multiple times.
- **Issue #362**: Outdated `ttyd` causes Lip Gloss styling bugs. Ensure `ttyd` is current.
- **Build prerequisites**: Background bleed-through fix (commit `c5ba7a0`) and overlay compositing fix (commit `6583511`) must be present.
- **Terminal/theme match**: Recording terminal background must match Catppuccin Mocha.

### Security

- Use `/tmp/demo.cm` — avoid leaking local directory structure
- VHS `Set Shell "bash"` with minimal prompt
- Review output for paths in status bar, title bar, footer

### Architecture

- Single source of truth: tapes and output in `docs/images/`
- Copy to `site/static/images/` via Taskfile target
- `site/static/images/` is correct (not `site/assets/`)

## Demo Scripts

### Hero: `docs/images/hero.tape`

```
# hero.tape - CalcMark hero demo
# Regenerate: vhs docs/images/hero.tape
Output docs/images/hero.gif

Set Shell "bash"
Set FontSize 18
Set Width 900
Set Height 500
Set Theme "Catppuccin Mocha"
Set Padding 15
Set TypingSpeed 50ms
Set Framerate 12

Require cm

# Create a demo file with calculations
Hide
Type "cat << 'EOF' > /tmp/demo.cm"
Enter
Type "# Monthly Budget"
Enter
Type ""
Enter
Type "income = $5000"
Enter
Type "rent = $1500"
Enter
Type "groceries = $400"
Enter
Type "savings = income * 0.20"
Enter
Type "remaining = income - rent - groceries - savings"
Enter
Type ""
Enter
Type "average of income, rent, groceries"
Enter
Type "EOF"
Enter
Show
Sleep 300ms

# Evaluate the file
Type "cm eval /tmp/demo.cm"
Enter
Sleep 3s

# Clean up
Hide
Type "rm /tmp/demo.cm"
Enter
Show
Sleep 500ms
```

### Feature: Variables Flow Downward — `docs/images/feature-variables.tape`

```
# feature-variables.tape - Shows live preview updating as variables are typed
Output docs/images/feature-variables.gif

Set Shell "bash"
Set FontSize 16
Set Width 700
Set Height 350
Set Theme "Catppuccin Mocha"
Set Padding 10
Set TypingSpeed 80ms
Set Framerate 12

Require cm

Hide
Type "cm edit /tmp/vars-demo.cm"
Enter
Show

Wait+Screen@5s /CalcMark/
Sleep 300ms

Type "income = $5000"
Enter
Sleep 800ms
Type "rent = income * 0.30"
Enter
Sleep 800ms
Type "remaining = income - rent"
Sleep 1s

Ctrl+C
Hide
Type "rm -f /tmp/vars-demo.cm"
Enter
Show
```

### Feature: Terminal-native (Autocomplete) — `docs/images/feature-autocomplete.tape`

```
# feature-autocomplete.tape - Shows autocomplete popup
Output docs/images/feature-autocomplete.gif

Set Shell "bash"
Set FontSize 16
Set Width 700
Set Height 350
Set Theme "Catppuccin Mocha"
Set Padding 10
Set TypingSpeed 80ms
Set Framerate 12

Require cm

Hide
Type "cm edit /tmp/ac-demo.cm"
Enter
Show

Wait+Screen@5s /CalcMark/
Sleep 300ms

Type "x = 100"
Enter
Sleep 500ms
Type "ro"
Sleep 1s
Tab
Type "x / 3)"
Sleep 1.5s

Ctrl+C
Hide
Type "rm -f /tmp/ac-demo.cm"
Enter
Show
```

### Feature: Export Anywhere — `docs/images/feature-export.tape`

```
# feature-export.tape - Shows export overlay
Output docs/images/feature-export.gif

Set Shell "bash"
Set FontSize 16
Set Width 700
Set Height 400
Set Theme "Catppuccin Mocha"
Set Padding 10
Set TypingSpeed 80ms
Set Framerate 12

Require cm

Hide
Type@0 "echo -e 'price = $42.50\ntax = price * 0.08\ntotal = price + tax' > /tmp/export-demo.cm"
Enter
Type "cm edit /tmp/export-demo.cm"
Enter
Show

Wait+Screen@5s /CalcMark/
Sleep 1s

Ctrl+E
Sleep 2s
Escape
Sleep 500ms

Ctrl+C
Hide
Type "rm -f /tmp/export-demo.cm"
Enter
Show
```

## Homepage Changes

### HTML: Add GIFs to Feature Cards

Update `site/layouts/_default/home.html` to add `<img>` tags in selected feature cards:

```html
<div class="feature-card">
  <h3>Variables flow downward</h3>
  <p>Define once, reference anywhere below. Change one number and watch everything update.</p>
  <img src="{{ "images/feature-variables.gif" | relURL }}"
       alt="CalcMark editor showing variables updating as you type"
       class="feature-gif" loading="lazy">
</div>
```

Feature card GIFs use `loading="lazy"` (they're below the fold). Hero GIF removes `loading="lazy"` (above the fold).

### CSS: Feature Card GIF Styling

Add to `site/assets/css/components.css`:

```css
.feature-gif {
  width: 100%;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  margin-top: var(--space-3);
}

@media (prefers-reduced-motion: reduce) {
  .feature-gif,
  .hero-gif {
    /* Browser hides GIF animation; first frame shown as static image */
  }
}
```

### Alt Text Updates

- Hero: `"CalcMark evaluating a monthly budget with variables, expressions, and natural language functions"`
- Variables: `"CalcMark editor showing variables updating as you type"`
- Autocomplete: `"CalcMark editor autocomplete suggesting functions"`
- Export: `"CalcMark editor export overlay showing format options"`

## Acceptance Criteria

- [x] Hero GIF shows `cm eval` with budget demo (variables, expressions, `average of`)
- [x] At least 2 feature card GIFs demonstrate TUI capabilities
- [x] All recordings are scripted VHS tape files (reproducible via `vhs <file>.tape`)
- [x] No sensitive information visible in any recording
- [x] GIF file sizes: hero < 1MB, feature cards < 500KB each
- [x] `task build` passes (Hugo site builds)
- [x] `task record-demos` regenerates all GIFs
- [x] Alt text is descriptive on all images

## Implementation Phases

### Phase 1: Verify Expressions and Record Hero

1. `task build`
2. Verify `average of income, rent, groceries` works with currency variables
3. Write and test `docs/images/hero.tape`
4. Run `vhs docs/images/hero.tape` — verify output quality and file size
5. Copy to `site/static/images/hero.gif`

### Phase 2: Record Feature GIFs and Update Site

1. Write and test feature tape files (start with 2: variables + autocomplete)
2. For TUI tapes: run multiple times to verify consistent rendering
3. If TUI rendering is inconsistent, fall back to `cm eval` variants
4. Copy feature GIFs to `site/static/images/`
5. Update `site/layouts/_default/home.html` — add feature card GIFs, update alt text, fix `loading`
6. Add `.feature-gif` CSS to `site/assets/css/components.css`
7. Add `record-hero` Taskfile target
8. `hugo --source site` to verify build

## Taskfile Target

```yaml
record-hero:
  desc: Regenerate all demo GIFs for the website
  cmds:
    - vhs docs/images/hero.tape
    - vhs docs/images/feature-variables.tape
    - vhs docs/images/feature-autocomplete.tape
    - vhs docs/images/feature-export.tape
    - cp docs/images/hero.gif site/static/images/
    - cp docs/images/feature-variables.gif site/static/images/
    - cp docs/images/feature-autocomplete.gif site/static/images/
    - cp docs/images/feature-export.gif site/static/images/
  sources:
    - docs/images/*.tape
  generates:
    - site/static/images/hero.gif
    - site/static/images/feature-*.gif
```

## Dependencies & Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| `average of` fails with variable refs | Medium | High | Verify first; fall back to `avg()` |
| VHS renders TUI inconsistently | Medium | High | Run multiple times; fall back to `cm eval` for that feature |
| Feature GIFs too large | Medium | Medium | Reduce dimensions (700x350), reduce framerate |
| `ttyd` outdated | Low | High | Update `ttyd` before recording |
| TUI changes make GIFs stale | High | Low | `task record-hero` makes regeneration trivial |

## References

### Internal
- Current hero tape: `docs/images/hero.tape`
- Homepage layout: `site/layouts/_default/home.html`
- Hero CSS: `site/assets/css/components.css:188-192`
- TUI autocomplete: `cmd/calcmark/tui/editor/autocomplete.go`
- Export overlay: `cmd/calcmark/tui/editor/export_overlay.go`
- Background bleed-through fixes: `docs/solutions/ui-bugs/lipgloss-background-bleed-through.md`
- Overlay compositing fixes: `docs/solutions/ui-bugs/overlay-compositing-ansi-state-bleed-through.md`

### External
- VHS: https://github.com/charmbracelet/vhs
- VHS Issue #362: https://github.com/charmbracelet/vhs/issues/362
- VHS Issue #412: https://github.com/charmbracelet/vhs/issues/412
