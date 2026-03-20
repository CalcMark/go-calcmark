---
title: "cm watch: HTML templating, CSS extraction, goldmark, and live reload"
category: integration-issues
date: 2026-03-19
tags: [watch, css, fsnotify, goldmark, websocket, template, embedded-mode, footnotes, go-embed, bluemonday]
modules: [cmd/calcmark/cmd/watch.go, format/html_formatter.go, format/templates/calcmark.css, format/templates/default.gohtml, convert.go, spec/document/markdown.go]
symptoms:
  - watch page renders unstyled content
  - file changes not detected after editor save
  - no diagnostic output on stderr
  - CSS duplicated across templates
  - embedded mode HTML missing styles
  - footnotes render as literal text
  - ZgotmplZ appears in style tags
root_cause: "Multiple compounding issues: watch template lacked CSS for content classes; fsnotify watched file inodes which break on atomic saves; goldmark missing Footnote extension; html/template escapes raw CSS strings unless typed as template.CSS"
---

# cm watch: HTML templating, CSS extraction, goldmark, and live reload

## Problem

The `cm watch` command served a live preview that was broken in multiple ways:
unstyled content, file changes silently ignored after the first editor save,
no diagnostic logging, footnotes rendering as literal `[^1]` text, and
`ZgotmplZ` appearing in style tags after attempting to share CSS.

## Root Cause Analysis

Six interconnected issues across the watch server and HTML formatting pipeline:

1. **Missing CSS in watch page.** `watchPageTemplate` had 5 lines of CSS but
   rendered content used classes (`.calc-block`, `.text-block`, `.frontmatter`)
   with no matching rules. The full 200-line stylesheet existed in
   `default.gohtml` but wasn't shared.

2. **Atomic saves break file-level fsnotify watches.** `watcher.Add(filename)`
   watches a specific inode. Editors like vim and VS Code write to a temp file
   then rename, removing the original inode. The watch silently stops delivering
   events — no error, no notification.

3. **No logging.** The watch server only printed startup messages. The atomic
   save breakage was invisible because there was no logging for change
   detection, re-renders, or WebSocket lifecycle.

4. **Embedded mode produces different HTML structure.** Embedded mode renders
   through goldmark, producing bare `<h1>`, `<p>`, `<pre><code>` elements.
   CM mode wraps content in `<div class="text-block">` and
   `<div class="calc-block">`. The scoped `.text-block h1` CSS rules never
   matched embedded mode output.

5. **Footnotes not rendering.** goldmark was configured with `extension.GFM`
   only. Footnote syntax (`[^1]`) requires `extension.Footnote` as a separate
   registration.

6. **`ZgotmplZ` escaping.** Go's `html/template` treats `string` values inside
   `<style>` tags as unsafe and replaces them with the sentinel `ZgotmplZ`.
   CSS must be typed as `template.CSS`.

## Solution

### CSS extraction to single source of truth

Created `format/templates/calcmark.css` as the canonical stylesheet, embedded
at compile time:

```go
//go:embed templates/calcmark.css
var styleCSS string

func StyleCSS() string { return styleCSS }
```

**default.gohtml** uses a template variable with `template.CSS` type (not
`string`) to avoid ZgotmplZ:

```go
data := struct {
    Style template.CSS  // NOT string
    // ...
}{
    Style: template.CSS(styleCSS),
}
```

**watch.go** injects CSS at init via simple string replacement (no template
engine needed for the watch shell):

```go
var watchPageTemplate string

func init() {
    watchPageTemplate = strings.Replace(watchPageShell, "{{STYLE}}", format.StyleCSS(), 1)
}
```

### Directory-level file watching

Watch the parent directory, filter events by absolute path:

```go
func (s *watchServer) addWatch(watcher *fsnotify.Watcher) error {
    absPath, err := filepath.Abs(s.filename)
    if err != nil {
        return err
    }
    s.filename = absPath
    return watcher.Add(filepath.Dir(absPath))
}
```

In `watchLoop`, filter by filename and handle Write, Create, AND Rename events:

```go
if event.Name != s.filename { continue }
if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
    // debounced re-render
}
```

### Debounced logging

Log inside the debounce callback, not on every raw event. A single editor
save can produce 3-4 fsnotify events (Write, Chmod, Write). Logging each
one creates noise; logging once inside the 100ms debounce callback produces
exactly one line per logical save.

### Dual-scope CSS for embedded and CM modes

The shared `calcmark.css` contains both scoped and unscoped rules:

```css
/* CM mode: content wrapped in .text-block divs */
.text-block h1 { margin-top: 1.5em; }

/* Embedded mode: bare HTML from goldmark, scoped under #content */
#content h1 { margin-top: 1.5em; }

/* Embedded calcmark code blocks get the blue border */
#content pre:has(> .language-calcmark) {
    border-left: 4px solid #0066cc;
}
```

### Goldmark footnote extension

Added `extension.Footnote` to both goldmark instances:

```go
var embeddedMarkdown = goldmark.New(
    goldmark.WithExtensions(extension.GFM, extension.Footnote),
)
```

Applied to both `convert.go` (embedded mode runtime) and
`spec/document/markdown.go` (CM mode text block rendering).

## Key Decisions

1. **Embed CSS at compile time** rather than serving as a separate HTTP asset.
   Keeps the watch server as a single self-contained binary.

2. **`strings.Replace` for watch shell, `html/template` for default.gohtml.**
   The watch page is a simple HTML shell with WebSocket JS — a full template
   engine is unnecessary. But default.gohtml needs Go template features for
   block rendering, so it uses `html/template` with `template.CSS`.

3. **Watch directory, not file.** This is the only correct pattern for fsnotify
   with atomic-save editors. File-level watching is fundamentally broken.

4. **`#content`-scoped selectors for embedded mode.** Avoids polluting global
   scope while ensuring bare goldmark HTML gets styled in the watch preview.

## Prevention

- **Never inline CSS in Go templates.** All styles go in embedded `.css` files
  referenced via `go:embed`. Flag any `<style>` block in a `.gohtml` file
  during review.

- **Always watch directories with fsnotify, never files.** Document this as a
  project convention. Write a test that verifies `WatchList()` contains the
  directory, not the file.

- **Use `template.CSS` for CSS in Go templates.** If you see `ZgotmplZ` in
  output, a type assertion is missing. Add a test that renders templates and
  asserts no `ZgotmplZ` appears.

- **Maintain a golden file for every supported markdown feature.** If footnotes
  are supported, a golden test must exercise them. The test breaks immediately
  when an extension is removed.

- **Debounce side effects.** In event-driven handlers with debouncing, logging
  and state changes belong inside the debounced callback, not in the raw
  event receiver.

## Related

- GitHub issue: [#86](https://github.com/CalcMark/go-calcmark/issues/86)
- Plan: `docs/plans/2026-03-19-002-fix-cm-watch-live-preview-plan.md`
- goldmark footnote extension: `github.com/yuin/goldmark/extension.Footnote`
- Go template type safety: `template.CSS`, `template.HTML`, `template.JS`
