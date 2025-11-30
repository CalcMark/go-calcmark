# CalcMark TUI Design Specification

**Version:** 1.0.0  
**Status:** Design Complete — Ready for Implementation  
**Target:** `cm` CLI tool with REPL and Editor modes

---

## Executive Summary

This document specifies the design for CalcMark's terminal user interface, consisting of two distinct modes:

1. **Simple REPL** (`cm` with no arguments) — A minimal, fast, line-by-line calculator
2. **Document Editor** (`cm edit` or `cm <file>`) — A split-pane TUI for working with CalcMark documents

The design prioritizes keyboard-first interaction with progressive disclosure: beginners use arrow keys and Enter; power users gain speed with vim-style bindings. Mouse support exists as a fallback, not a requirement.

---

## Table of Contents

1. [Design Philosophy](#design-philosophy)
2. [CLI Entry Points](#cli-entry-points)
3. [Simple REPL Mode](#simple-repl-mode)
4. [Document Editor Mode](#document-editor-mode)
5. [Visual Design](#visual-design)
6. [State Machine](#state-machine)
7. [Component Architecture](#component-architecture)
8. [Keyboard Reference](#keyboard-reference)
9. [Implementation Notes](#implementation-notes)

---

## Design Philosophy

### Core Principles

1. **Immediate Value** — Users see calculation results instantly. No compile step, no run command.

2. **Progressive Disclosure** — The interface reveals complexity only when needed:
   - Level 1: Arrow keys, Enter, Escape, `/` (anyone can use this)
   - Level 2: Vim bindings (`j/k`, `gg/G`, `dd`) for speed
   - Level 3: Commands (`/find`, `/goto`, `/eval`) for power users

3. **Document-Centric** — CalcMark files are the source of truth. The editor shows the document and its computed results side-by-side.

4. **Keyboard-First, Mouse-Optional** — Every action is keyboard-accessible. Mouse clicks work but are never required.

5. **Focused Scope** — This is not an IDE. No plugins, no multiple buffers, no LSP. One file, edited well.

### What This Is NOT

- Not a general-purpose text editor (use vim/VS Code for that)
- Not a spreadsheet (calculations flow top-to-bottom, not in a grid)
- Not a notebook (no cell execution order, no hidden state)

---

## CLI Entry Points

```
cm                      # Simple REPL — interactive calculator
cm edit                 # Editor with file picker
cm edit budget.cm       # Editor opens specific file
cm budget.cm            # Shorthand for: cm edit budget.cm
cm eval "2 * PI * 5"    # One-shot evaluation, prints result, exits
cm fmt budget.cm        # Format file (future)
cm version              # Version info
cm help                 # Help text
```

### Argument Parsing Logic

```
if no args:
    start Simple REPL
else if first arg is "edit":
    if second arg exists:
        open Editor with that file
    else:
        open Editor with file picker
else if first arg is "eval":
    evaluate remaining args as expression, print, exit
else if first arg is "version" or "help" or other command:
    handle command
else if first arg looks like a file path:
    open Editor with that file (create if doesn't exist)
else:
    show usage error
```

---

## Simple REPL Mode

The REPL is a minimal, fast, line-by-line calculator. No split panes, no preview, no document structure. Just input → output.

### Visual Layout

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ CalcMark v0.2.0                                                             │
│                                                                             │
│ > salary = $85000                                                           │
│   $85,000.00                                                                │
│ > monthly = salary / 12                                                     │
│   $7,083.33                                                                 │
│ > monthly * 0.30                                                            │
│   $2,125.00                                                                 │
│ > avg(monthly, $6000, $8000)                                                │
│   $7,027.78                                                                 │
│ > tax_rate = 0.22                                                           │
│   0.22                                                                      │
│ > monthly * tax_rate                                                        │
│   $1,558.33                                                                 │
│ >                                                                           │
│                                                                             │
│                                                                             │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ ↑↓ history │ /help │ /vars │ /clear │ /quit                                │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Behavior

| Action | Effect |
|--------|--------|
| Type expression, press Enter | Evaluate, show result below input |
| `↑` / `↓` | Navigate history |
| `/help` | Show help overlay |
| `/vars` | List all defined variables and their values |
| `/clear` | Clear screen and history (variables persist) |
| `/reset` | Clear everything including variables |
| `/quit` or `Ctrl-C` | Exit |
| `/edit` | Switch to Editor mode (opens file picker) |
| `/edit file.cm` | Switch to Editor mode with specific file |

### REPL State

```go
type REPLState struct {
    History     []HistoryEntry  // Previous inputs and outputs
    Context     *evaluator.Context  // Variable bindings
    InputBuffer string          // Current input line
    HistoryPos  int             // Position in history (-1 = current input)
    ShowingHelp bool            // Help overlay visible
}

type HistoryEntry struct {
    Input  string
    Output string  // Formatted result or error message
    IsError bool
}
```

### Autosuggestion

As the user types, suggestions appear below the input line showing matching:
- **User-defined variables** from the current context
- **Built-in functions** (`avg`, `sqrt`, etc.)
- **Built-in units** (`meter`, `teaspoon`, `GB`, etc.)
- **Built-in constants** (`PI`, `E`)

```
> tea
Hints: team_velocity [Tab] │ teaspoon (tsp)
```

The format shows:
- Variable/function name
- `[Tab]` indicator for the first (best) match
- Abbreviation in parentheses for units that have one

**Behavior:**
- Suggestions appear after 1+ characters typed
- `Tab` completes the first suggestion
- `Tab` again cycles to next suggestion
- Suggestions filter as you type more characters
- Suggestions disappear when input matches exactly or is empty

**Priority order:**
1. User-defined variables (most relevant to current document)
2. Built-in functions
3. Built-in units/constants

```go
type Suggestion struct {
    Name         string  // "team_velocity", "teaspoon"
    Abbreviation string  // "", "tsp"
    Kind         SuggestionKind  // Variable, Function, Unit, Constant
}

type SuggestionKind int

const (
    SuggestionVariable SuggestionKind = iota
    SuggestionFunction
    SuggestionUnit
    SuggestionConstant
)
```

### REPL Result Formatting

Results are formatted for human readability:

- Numbers: `50000` → `50,000`
- Currency: `$85000` → `$85,000.00`
- Large numbers: Optional SI prefixes in future (`1500000` → `1.5M`)
- Errors: Shown inline, prefixed with `⚠`

```
> cost = undefined_var * 2
  ⚠ undefined variable: undefined_var
> cost
  ⚠ undefined variable: cost
```

---

## Document Editor Mode

The Editor is a split-pane TUI for working with CalcMark documents. The left pane shows the source document (editable), the right pane shows computed results (read-only).

### Visual Layout — Full Preview Mode

```
┌─ Source ────────────────────────────────┬─ Preview ───────────────────────┐
│                                         │ ▸ Globals (3)               [g] │
│                                         ├─────────────────────────────────┤
│ # Q4 Infrastructure Budget              │                                 │
│                                         │                                 │
│ Planning the migration to the new       │                                 │
│ cluster. Need to account for compute    │                                 │
│ and personnel costs.                    │                                 │
│                                         │                                 │
│ ## Team Costs                           │                                 │
│                                         │                                 │
│ dev_count = 4                           │ dev_count             4         │
│ dev_rate = $950/day                     │ dev_rate              $950/day  │
│ sprint_weeks = 3                        │ sprint_weeks          3         │
│ working_days = sprint_weeks * 5         │ working_days          15        │
│                                         │                                 │
│ team_cost = dev_count * dev_rate *      │ team_cost             $57,000   │
│             working_days     ← cursor   │                                 │
│                                         │                                 │
│ ## Infrastructure                       │                                 │
│                                         │                                 │
│ compute = $12,400                       │ compute               $12,400   │
│ storage = $3,200                        │ storage               $3,200    │
│ network = $1,800                        │ network               $1,800    │
│                                         │                                 │
│ infra_total = compute + storage +       │ infra_total           $17,400   │
│               network                   │                                 │
│                                         │                                 │
│ ## Summary                              │                                 │
│                                         │                                 │
│ total_cost = team_cost + infra_total    │ total_cost            $74,400   │
│                                         │                                 │
│ 2 * PI * 100                            │                       628.32    │
│                                         │                                 │
├─────────────────────────────────────────┴─────────────────────────────────┤
│ dev_count = 4 │ dev_rate = $950/day │ working_days = 15                   │
├───────────────────────────────────────────────────────────────────────────┤
│ budget.cm │ ln 16/42 │ 12 calcs │ modified │ Tab:preview  ?:help  /:cmd   │
└───────────────────────────────────────────────────────────────────────────┘
```

### Visual Layout — Minimal Preview Mode

```
┌─ Source ────────────────────────────────┬─ Preview ───────────────────────┐
│                                         │ ▸ Globals (3)               [g] │
│                                         ├─────────────────────────────────┤
│ # Q4 Infrastructure Budget              │                                 │
│                                         │                                 │
│ Planning the migration to the new       │                                 │
│ cluster. Need to account for compute    │                                 │
│ and personnel costs.                    │                                 │
│                                         │                                 │
│ ## Team Costs                           │                                 │
│                                         │                                 │
│ dev_count = 4                           │                         → 4     │
│ dev_rate = $950/day                     │                  → $950/day     │
│ sprint_weeks = 3                        │                         → 3     │
│ working_days = sprint_weeks * 5         │                        → 15     │
│                                         │                                 │
│ team_cost = dev_count * dev_rate *      │                   → $57,000     │
│             working_days     ← cursor   │                                 │
│                                         │                                 │
```

### Visual Layout — Hidden Preview Mode

```
┌─ Source ────────────────────────────────────────────────────────────────────┐
│ # Q4 Infrastructure Budget                                                  │
│                                                                             │
│ Planning the migration to the new cluster. Need to account for compute     │
│ and personnel costs.                                                        │
│                                                                             │
│ ## Team Costs                                                               │
│                                                                             │
│ dev_count = 4                                                               │
│ dev_rate = $950/day                                                         │
│ sprint_weeks = 3                                                            │
│ working_days = sprint_weeks * 5                                             │
│                                                                             │
│ team_cost = dev_count * dev_rate * working_days                ← cursor    │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ dev_count = 4 │ dev_rate = $950/day │ working_days = 15                     │
├─────────────────────────────────────────────────────────────────────────────┤
│ budget.cm │ ln 13/42 │ 12 calcs │ modified │ Tab:preview  ?:help  /:cmd     │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Layout Components

#### 1. Source Pane (Left)
- Editable document content
- Cursor lives here
- Syntax highlighting (subtle)
- Line wrapping for long expressions
- Shows frontmatter as YAML when scrolled to top

#### 2. Preview Pane (Right)
- Three modes: **Full**, **Minimal**, **Hidden**
- Vertically aligned with source (line N in source → line N in preview)
- Read-only
- Shows globals panel at top (collapsible)

#### 3. Globals Panel (Top of Preview)
- Collapsed by default, shows count: `▸ Globals (3)`
- Expanded shows all frontmatter variables
- Keyboard navigable (press `g` to toggle/enter)
- Stays pinned while scrolling document

#### 4. Context Footer
- Shows values of variables referenced in current line
- Updates as cursor moves
- Empty on prose/markdown lines
- Becomes command input when `/` is pressed

#### 5. Status Bar
- Filename
- Line position (`ln 16/42`)
- Calculation count
- Modified indicator
- Keyboard hints

### Vertical Alignment Rule

**Critical invariant:** Every source line has exactly one corresponding preview line at the same vertical position.

When a calculation spans multiple lines (due to wrapping):
- The result appears on the **first** line of the expression
- Continuation lines in the preview are blank

```
Source                                    Preview
─────────────────────────────────────     ─────────────────────────────────
team_cost = dev_count * dev_rate *        team_cost             $57,000
            working_days                                                   ← blank
```

This ensures scrolling stays synchronized.

### Preview Modes

| Mode | Display | When to Use |
|------|---------|-------------|
| **Full** | Variable name + value | Reviewing what's defined |
| **Minimal** | Arrow + value only | Focused on prose, just want numbers |
| **Hidden** | No preview pane | Full-width editing, maximum space |

Toggle with `Tab`: Full → Minimal → Hidden → Full

### Globals Panel Behavior

**Collapsed state:**
```
│ ▸ Globals (3)                       [g] │
```

**Expanded state:**
```
│ ▾ Globals                           [g] │
│   tax_rate              0.08            │
│   fiscal_year           2025            │
│   base_salary           $95,000         │
```

**Keyboard navigation:**
1. Press `g` → globals expand, focus moves to globals list
2. `↑`/`↓` to navigate within globals
3. `Enter` → jump to that variable's definition in frontmatter
4. `Escape` or `g` again → collapse and return focus to document

### Context Footer

The footer shows what feeds into the current expression:

```
│ dev_count = 4 │ dev_rate = $950/day │ working_days = 15                   │
```

On prose lines, it shows nothing or the last calculation result.

When `/` is pressed, the footer becomes the command input:

```
│ /save                                                                     │
```

With fuzzy-matched suggestions appearing above:

```
├─────────────────────────────────────────┴─────────────────────────────────┤
│ /save                                                                     │
│   /save         Save document                                             │
│   /saveas       Save as new file                                          │
└───────────────────────────────────────────────────────────────────────────┘
```

### Error Display

Errors appear inline in the preview, calmly:

```
Source                                    Preview
─────────────────────────────────────     ─────────────────────────────────
cost = base * undefined_rate              ⚠ undefined: undefined_rate
result = 10 / 0                           ⚠ division by zero
incomplete = 5 +                          ⚠ incomplete expression
```

Errors use a muted warning color (amber), not aggressive red.

### Autosuggestion in Editor

When editing a line, suggestions appear below the source pane:

```
┌─ Source ────────────────────────────────┬─ Preview ───────────────────────┐
│ ...                                     │ ...                             │
│ total = dev_cost + infra + tea█         │                                 │
│ ...                                     │ ...                             │
├─────────────────────────────────────────┴─────────────────────────────────┤
│ Hints: team_velocity [Tab] │ teaspoon (tsp) │ team_cost                   │
├───────────────────────────────────────────────────────────────────────────┤
│ dev_cost = $57,000 │ infra = $17,400                                      │
├───────────────────────────────────────────────────────────────────────────┤
│ budget.cm │ ln 16/42 │ modified │ Tab:complete  ?:help  /:cmd             │
└───────────────────────────────────────────────────────────────────────────┘
```

**Suggestion sources (in priority order):**
1. User-defined variables from current document context
2. Built-in functions (`avg`, `sqrt`, etc.)
3. Built-in units (`meter`, `teaspoon`, `GB`, `day`, etc.)
4. Built-in constants (`PI`, `E`)

**Behavior:**
- Suggestions appear after 1+ characters of a word
- First match shows `[Tab]` indicator
- Units show abbreviation: `teaspoon (tsp)`
- `Tab` completes, `Tab` again cycles through matches
- Typing more characters filters the list
- Works in both REPL and Editor modes

---

## Visual Design

### Color Palette

Use a minimal palette with semantic meaning:

| Role | Usage | Suggested Color |
|------|-------|-----------------|
| **Text** | Prose, identifiers | Default terminal foreground |
| **Dim** | Operators, structural, hints | Gray (ANSI 8 or 240) |
| **Result** | Computed values | Cyan or Amber (ANSI 6 or 3) |
| **Error** | Warnings, errors | Amber (ANSI 3) or soft Red (ANSI 1) |
| **Header** | Markdown headers | Bold + slightly brighter |
| **Accent** | Cursor, selection | Terminal cursor color |

### Typography with Lipgloss

```go
var (
    // Source pane
    textStyle       = lipgloss.NewStyle()
    dimStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
    headerStyle     = lipgloss.NewStyle().Bold(true)
    errorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))  // amber
    
    // Preview pane
    resultStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))  // cyan
    varNameStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
    
    // Panels
    paneBorderStyle = lipgloss.NewStyle().
                        Border(lipgloss.NormalBorder()).
                        BorderForeground(lipgloss.Color("240"))
    
    statusBarStyle  = lipgloss.NewStyle().
                        Background(lipgloss.Color("236")).
                        Foreground(lipgloss.Color("252"))
    
    // Globals panel
    globalsHeaderStyle = lipgloss.NewStyle().Bold(true)
    globalsItemStyle   = lipgloss.NewStyle().PaddingLeft(2)
)
```

### Spacing and Alignment

- Source pane: 2-character left padding
- Preview pane: Results right-aligned
- Globals panel: 2-character indent for items
- Status bar: 1-character padding
- Pane border: Single-line box drawing characters

### Number Formatting

All numeric output is human-formatted:

| Raw | Formatted |
|-----|-----------|
| `50000` | `50,000` |
| `1234.5` | `1,234.5` |
| `$85000` | `$85,000.00` |
| `€1234.56` | `€1,234.56` |
| `0.08` | `0.08` (or `8%` if variable name suggests percentage) |

---

## State Machine

### Application States

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│                              ┌─────────────┐                                │
│                              │   START     │                                │
│                              └──────┬──────┘                                │
│                                     │                                       │
│                      ┌──────────────┼──────────────┐                        │
│                      │              │              │                        │
│                      ▼              ▼              ▼                        │
│               ┌──────────┐   ┌──────────┐   ┌──────────┐                    │
│               │   REPL   │   │  PICKER  │   │  EDITOR  │                    │
│               └────┬─────┘   └────┬─────┘   └────┬─────┘                    │
│                    │              │              │                          │
│                    │              │    ┌─────────┴─────────┐                │
│                    │              │    │                   │                │
│                    │              │    ▼                   ▼                │
│                    │              │ ┌──────┐         ┌──────────┐           │
│                    │              │ │NORMAL│◄───────►│ EDITING  │           │
│                    │              │ └──┬───┘         └──────────┘           │
│                    │              │    │                                    │
│                    │              │    ▼                                    │
│                    │              │ ┌──────────┐                            │
│                    │              │ │ COMMAND  │                            │
│                    │              │ └──────────┘                            │
│                    │              │                                         │
│                    ▼              ▼                                         │
│               ┌─────────────────────────────────────┐                       │
│               │                EXIT                 │                       │
│               └─────────────────────────────────────┘                       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Editor Sub-States

```go
type EditorMode int

const (
    ModeNormal EditorMode = iota  // Navigating, not typing
    ModeEditing                    // Typing into a line
    ModeCommand                    // Typing a /command
    ModeGlobals                    // Navigating globals panel
    ModeHelp                       // Help overlay visible
    ModePicker                     // File picker overlay (for /open, /saveas)
)
```

### State Transitions

#### Normal Mode

| Input | Condition | Next State | Action |
|-------|-----------|------------|--------|
| `↓` or `j` | — | Normal | Move cursor down |
| `↑` or `k` | — | Normal | Move cursor up |
| `Enter` | On any line | Editing | Begin editing current line |
| `e` or `i` | On any line | Editing | Begin editing current line |
| `o` | — | Editing | Insert line below, begin editing |
| `O` | — | Editing | Insert line above, begin editing |
| `dd` | — | Normal | Delete current line |
| `/` | — | Command | Open command palette |
| `?` | — | Help | Show help overlay |
| `Tab` | — | Normal | Cycle preview mode |
| `g` | Globals collapsed | Globals | Expand globals, focus on list |
| `g` | Globals expanded | Normal | Collapse globals |
| `gg` | — | Normal | Jump to top |
| `G` | — | Normal | Jump to bottom |
| `Ctrl-C` | Unsaved changes | Normal | Prompt to save (or ignore) |
| `Ctrl-C` | No changes | Exit | Quit |

#### Editing Mode

| Input | Condition | Next State | Action |
|-------|-----------|------------|--------|
| Any printable | — | Editing | Insert character, update suggestions |
| `Backspace` | Characters exist | Editing | Delete character, update suggestions |
| `←` / `→` | — | Editing | Move cursor within line |
| `Tab` | Suggestions visible | Editing | Complete with selected suggestion |
| `Tab` | Just completed | Editing | Cycle to next suggestion |
| `Tab` | No suggestions | Editing | Insert literal tab (or ignore) |
| `Enter` | — | Editing | Commit line, move to next, continue editing |
| `Escape` | — | Normal | Commit line, return to normal |
| `Ctrl-C` | — | Normal | Revert line changes, return to normal |

#### Command Mode

| Input | Condition | Next State | Action |
|-------|-----------|------------|--------|
| Any printable | — | Command | Update command input, filter suggestions |
| `Backspace` | Characters exist | Command | Delete character |
| `Backspace` | Empty input | Normal | Cancel command mode |
| `Enter` | Valid command | Normal | Execute command |
| `Enter` | Expression | Normal | Evaluate as quick-eval, show result |
| `Escape` | — | Normal | Cancel command mode |
| `↓` / `↑` | Suggestions visible | Command | Navigate suggestions |
| `Tab` | Suggestion selected | Command | Autocomplete command |

#### Globals Mode

| Input | Condition | Next State | Action |
|-------|-----------|------------|--------|
| `↓` / `↑` | — | Globals | Navigate within globals list |
| `Enter` | On a global | Normal | Jump to definition in frontmatter |
| `Escape` | — | Normal | Collapse globals, return focus |
| `g` | — | Normal | Collapse globals, return focus |

#### Help Mode

| Input | Condition | Next State | Action |
|-------|-----------|------------|--------|
| Any key | — | Normal | Dismiss help overlay |
| `Escape` | — | Normal | Dismiss help overlay |

---

## Component Architecture

### Bubbletea Model Hierarchy

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              RootModel                                      │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         AppState                                     │   │
│  │  - Mode: REPL | PICKER | EDITOR                                     │   │
│  │  - WindowSize: width, height                                         │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────┐  ┌─────────────────────┐  ┌──────────────────┐   │
│  │     REPLModel       │  │    PickerModel      │  │   EditorModel    │   │
│  │                     │  │                     │  │                  │   │
│  │  - History          │  │  - filepicker.Model │  │  - Document      │   │
│  │  - Context          │  │  - CurrentDir       │  │  - EditorState   │   │
│  │  - InputBuffer      │  │  - AllowedTypes     │  │  - Components    │   │
│  │  - HistoryPos       │  │                     │  │                  │   │
│  │  - textinput.Model  │  │                     │  │                  │   │
│  └─────────────────────┘  └─────────────────────┘  └────────┬─────────┘   │
│                                                              │             │
│                                                              ▼             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         EditorModel                                  │   │
│  │                                                                      │   │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────┐  │   │
│  │  │  SourcePane     │  │  PreviewPane    │  │    StatusBar        │  │   │
│  │  │                 │  │                 │  │                     │  │   │
│  │  │  - Lines[]      │  │  - Mode         │  │  - Filename         │  │   │
│  │  │  - CursorLine   │  │  - Results[]    │  │  - LineInfo         │  │   │
│  │  │  - CursorCol    │  │  - GlobalsPanel │  │  - CalcCount        │  │   │
│  │  │  - ScrollOffset │  │  - ScrollOffset │  │  - Modified         │  │   │
│  │  │  - viewport     │  │  - viewport     │  │                     │  │   │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────────┘  │   │
│  │                                                                      │   │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────┐  │   │
│  │  │  ContextFooter  │  │  CommandPalette │  │    HelpOverlay      │  │   │
│  │  │                 │  │                 │  │                     │  │   │
│  │  │  - Variables[]  │  │  - Input        │  │  - KeyBindings      │  │   │
│  │  │  - Values[]     │  │  - Suggestions  │  │  - Sections         │  │   │
│  │  │                 │  │  - SelectedIdx  │  │                     │  │   │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────────┘  │   │
│  │                                                                      │   │
│  │  ┌─────────────────┐                                                 │   │
│  │  │  Autosuggester  │                                                 │   │
│  │  │                 │                                                 │   │
│  │  │  - Suggestions  │                                                 │   │
│  │  │  - SelectedIdx  │                                                 │   │
│  │  │  - Prefix       │                                                 │   │
│  │  └─────────────────┘                                                 │   │
│  │                                                                      │   │
│  │  ┌─────────────────────────────────────────────────────────────────┐│   │
│  │  │                      Document                                    ││   │
│  │  │                                                                  ││   │
│  │  │  - Filepath                                                      ││   │
│  │  │  - Frontmatter (globals)                                         ││   │
│  │  │  - Lines[]                                                       ││   │
│  │  │  - Context (evaluator.Context)                                   ││   │
│  │  │  - Results[] (per-line evaluation results)                       ││   │
│  │  │  - Diagnostics[]                                                 ││   │
│  │  │  - Modified bool                                                 ││   │
│  │  └─────────────────────────────────────────────────────────────────┘│   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Component Descriptions

#### RootModel

The top-level model that owns application state and delegates to sub-models.

```go
type RootModel struct {
    appMode    AppMode  // REPL, Picker, Editor
    windowSize tea.WindowSizeMsg
    
    repl   REPLModel
    picker PickerModel
    editor EditorModel
}

type AppMode int

const (
    AppModeREPL AppMode = iota
    AppModePicker
    AppModeEditor
)
```

#### REPLModel

Simple REPL state and logic.

```go
type REPLModel struct {
    history     []HistoryEntry
    context     *evaluator.Context
    input       textinput.Model  // from bubbles
    historyPos  int
    showingHelp bool
}

func (m REPLModel) Update(msg tea.Msg) (REPLModel, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "enter":
            return m.evaluateInput()
        case "up":
            return m.historyBack()
        case "down":
            return m.historyForward()
        case "ctrl+c":
            return m, tea.Quit
        }
    }
    
    // Delegate to textinput
    var cmd tea.Cmd
    m.input, cmd = m.input.Update(msg)
    return m, cmd
}
```

#### PickerModel

Wraps the bubbles filepicker component.

```go
type PickerModel struct {
    filepicker   filepicker.Model
    selectedFile string
    err          error
}

func NewPickerModel(startDir string) PickerModel {
    fp := filepicker.New()
    fp.CurrentDirectory = startDir
    fp.AllowedTypes = []string{".cm", ".calcmark", ".md"}
    fp.Height = 20
    
    return PickerModel{
        filepicker: fp,
    }
}
```

#### EditorModel

The main editor state.

```go
type EditorModel struct {
    // Core state
    document    Document
    mode        EditorMode
    
    // UI components
    sourcePane     SourcePane
    previewPane    PreviewPane
    statusBar      StatusBar
    contextFooter  ContextFooter
    commandPalette CommandPalette
    helpOverlay    HelpOverlay
    globalsPanel   GlobalsPanel
    
    // Layout
    windowWidth  int
    windowHeight int
    
    // Edit state
    undoStack    []Document
    redoStack    []Document
}

type EditorMode int

const (
    ModeNormal EditorMode = iota
    ModeEditing
    ModeCommand
    ModeGlobals
    ModeHelp
)
```

#### Document

The document model, separate from UI concerns.

```go
type Document struct {
    filepath     string
    frontmatter  map[string]interface{}  // Parsed YAML globals
    lines        []string
    context      *evaluator.Context
    results      []LineResult
    diagnostics  []validator.Diagnostic
    modified     bool
}

type LineResult struct {
    LineType    classifier.LineType  // CALCULATION, MARKDOWN, BLANK
    Value       types.Type           // nil for non-calculations
    VarName     string               // empty for expressions without assignment
    Error       error                // nil if no error
    References  []string             // variable names referenced in this line
}

func (d *Document) Recalculate() {
    d.context = evaluator.NewContext()
    
    // Load frontmatter globals into context
    for name, value := range d.frontmatter {
        d.context.Set(name, convertToType(value))
    }
    
    // Evaluate each line
    d.results = make([]LineResult, len(d.lines))
    for i, line := range d.lines {
        d.results[i] = d.evaluateLine(line)
    }
}
```

#### SourcePane

The left pane showing editable content.

```go
type SourcePane struct {
    viewport    viewport.Model  // from bubbles
    lines       []string
    cursorLine  int
    cursorCol   int
    editBuffer  string  // current line being edited
    isEditing   bool
}

func (p SourcePane) View() string {
    var b strings.Builder
    
    visibleStart := p.viewport.YOffset
    visibleEnd := visibleStart + p.viewport.Height
    
    for i := visibleStart; i < visibleEnd && i < len(p.lines); i++ {
        line := p.lines[i]
        
        if i == p.cursorLine {
            if p.isEditing {
                line = p.renderEditingLine()
            } else {
                line = p.renderCursorLine(line)
            }
        } else {
            line = p.renderLine(line, i)
        }
        
        b.WriteString(line)
        b.WriteString("\n")
    }
    
    return b.String()
}
```

#### PreviewPane

The right pane showing computed results.

```go
type PreviewPane struct {
    viewport     viewport.Model
    mode         PreviewMode
    results      []LineResult
    globalsPanel GlobalsPanel
}

type PreviewMode int

const (
    PreviewModeFull PreviewMode = iota
    PreviewModeMinimal
    PreviewModeHidden
)

func (p PreviewPane) View() string {
    if p.mode == PreviewModeHidden {
        return ""
    }
    
    var b strings.Builder
    
    // Globals panel
    b.WriteString(p.globalsPanel.View())
    b.WriteString("\n")
    
    // Results, line by line
    for _, result := range p.visibleResults() {
        b.WriteString(p.renderResult(result))
        b.WriteString("\n")
    }
    
    return b.String()
}

func (p PreviewPane) renderResult(r LineResult) string {
    if r.LineType != classifier.CALCULATION {
        return ""  // blank line to maintain alignment
    }
    
    if r.Error != nil {
        return errorStyle.Render(fmt.Sprintf("⚠ %s", r.Error))
    }
    
    formatted := formatValue(r.Value)
    
    switch p.mode {
    case PreviewModeFull:
        if r.VarName != "" {
            return fmt.Sprintf("%s  %s",
                varNameStyle.Render(padLeft(r.VarName, 20)),
                resultStyle.Render(formatted))
        }
        return resultStyle.Render(padLeft(formatted, 22))
        
    case PreviewModeMinimal:
        return resultStyle.Render(fmt.Sprintf("→ %s", formatted))
    }
    
    return ""
}
```

#### GlobalsPanel

The collapsible globals display.

```go
type GlobalsPanel struct {
    expanded    bool
    globals     []GlobalVar
    focusIndex  int
    focused     bool
}

type GlobalVar struct {
    Name  string
    Value types.Type
    Line  int  // line number in frontmatter for jump-to-definition
}

func (g GlobalsPanel) View() string {
    if !g.expanded {
        return fmt.Sprintf("▸ Globals (%d)                       [g]", len(g.globals))
    }
    
    var b strings.Builder
    b.WriteString("▾ Globals                           [g]\n")
    
    for i, gv := range g.globals {
        prefix := "  "
        if g.focused && i == g.focusIndex {
            prefix = "> "
        }
        
        b.WriteString(fmt.Sprintf("%s%s  %s\n",
            prefix,
            varNameStyle.Render(padLeft(gv.Name, 18)),
            resultStyle.Render(formatValue(gv.Value))))
    }
    
    return b.String()
}
```

#### ContextFooter

Shows variables referenced in current line.

```go
type ContextFooter struct {
    references []VarReference
}

type VarReference struct {
    Name  string
    Value types.Type
}

func (f ContextFooter) View(width int) string {
    if len(f.references) == 0 {
        return ""
    }
    
    parts := make([]string, len(f.references))
    for i, ref := range f.references {
        parts[i] = fmt.Sprintf("%s = %s", ref.Name, formatValue(ref.Value))
    }
    
    return strings.Join(parts, " │ ")
}
```

#### CommandPalette

The `/command` input with fuzzy matching.

```go
type CommandPalette struct {
    input        textinput.Model
    suggestions  []Command
    selectedIdx  int
    visible      bool
}

type Command struct {
    Name        string
    Description string
    Handler     func(args string) tea.Cmd
}

var commands = []Command{
    {Name: "save", Description: "Save document"},
    {Name: "saveas", Description: "Save as new file"},
    {Name: "quit", Description: "Quit (warns if unsaved)"},
    {Name: "help", Description: "Show help"},
    {Name: "globals", Description: "Toggle globals panel"},
    {Name: "preview", Description: "Cycle preview mode"},
    {Name: "find", Description: "Search document"},
    {Name: "goto", Description: "Jump to line number"},
    {Name: "eval", Description: "Evaluate expression"},
    {Name: "insert", Description: "Insert last eval result"},
    {Name: "undo", Description: "Undo last change"},
    {Name: "redo", Description: "Redo last undone change"},
    {Name: "open", Description: "Open different file"},
}

func (p CommandPalette) filterSuggestions() []Command {
    input := strings.TrimPrefix(p.input.Value(), "/")
    if input == "" {
        return commands
    }
    
    var matches []Command
    for _, cmd := range commands {
        if fuzzyMatch(cmd.Name, input) {
            matches = append(matches, cmd)
        }
    }
    return matches
}
```

#### StatusBar

Bottom status line.

```go
type StatusBar struct {
    filename    string
    currentLine int
    totalLines  int
    calcCount   int
    modified    bool
}

func (s StatusBar) View(width int, isEditing bool) string {
    left := s.filename
    if s.modified {
        left += " [modified]"
    }
    
    middle := fmt.Sprintf("ln %d/%d │ %d calcs", 
        s.currentLine, s.totalLines, s.calcCount)
    
    // Contextual hints based on mode
    var right string
    if isEditing {
        right = "Tab:complete  Esc:done  /:cmd"
    } else {
        right = "Tab:preview  ?:help  /:cmd"
    }
    
    // Distribute space
    return statusBarStyle.Render(
        distributeSpace(left, middle, right, width))
}
```

#### Autosuggester

Provides inline suggestions while editing.

```go
type Autosuggester struct {
    suggestions    []Suggestion
    selectedIndex  int
    prefix         string  // what the user has typed
    visible        bool
}

type Suggestion struct {
    Name         string          // "team_velocity", "teaspoon"
    Abbreviation string          // "", "tsp"  
    Kind         SuggestionKind  // Variable, Function, Unit, Constant
}

type SuggestionKind int

const (
    SuggestionVariable SuggestionKind = iota
    SuggestionFunction
    SuggestionUnit
    SuggestionConstant
)

func (a *Autosuggester) Update(input string, cursorPos int, context *evaluator.Context) {
    // Extract the word being typed at cursor position
    a.prefix = extractWordAtCursor(input, cursorPos)
    
    if len(a.prefix) == 0 {
        a.visible = false
        a.suggestions = nil
        return
    }
    
    a.suggestions = a.findMatches(a.prefix, context)
    a.visible = len(a.suggestions) > 0
    a.selectedIndex = 0
}

func (a *Autosuggester) findMatches(prefix string, context *evaluator.Context) []Suggestion {
    var matches []Suggestion
    prefix = strings.ToLower(prefix)
    
    // 1. User-defined variables (highest priority)
    for name := range context.Variables() {
        if strings.HasPrefix(strings.ToLower(name), prefix) {
            matches = append(matches, Suggestion{
                Name: name,
                Kind: SuggestionVariable,
            })
        }
    }
    
    // 2. Built-in functions
    for _, fn := range builtinFunctions {
        if strings.HasPrefix(fn.Name, prefix) {
            matches = append(matches, Suggestion{
                Name: fn.Name,
                Kind: SuggestionFunction,
            })
        }
    }
    
    // 3. Built-in units and constants
    for _, unit := range builtinUnits {
        if strings.HasPrefix(strings.ToLower(unit.Name), prefix) ||
           strings.HasPrefix(strings.ToLower(unit.Abbrev), prefix) {
            matches = append(matches, Suggestion{
                Name:         unit.Name,
                Abbreviation: unit.Abbrev,
                Kind:         SuggestionUnit,
            })
        }
    }
    
    // Sort: variables first, then by name
    sort.Slice(matches, func(i, j int) bool {
        if matches[i].Kind != matches[j].Kind {
            return matches[i].Kind < matches[j].Kind
        }
        return matches[i].Name < matches[j].Name
    })
    
    return matches
}

func (a *Autosuggester) View() string {
    if !a.visible || len(a.suggestions) == 0 {
        return ""
    }
    
    var parts []string
    for i, s := range a.suggestions {
        var display string
        if s.Abbreviation != "" {
            display = fmt.Sprintf("%s (%s)", s.Name, s.Abbreviation)
        } else {
            display = s.Name
        }
        
        if i == 0 {
            display += " [Tab]"
        }
        
        if i == a.selectedIndex {
            display = selectedStyle.Render(display)
        } else {
            display = hintStyle.Render(display)
        }
        
        parts = append(parts, display)
    }
    
    return hintStyle.Render("Hints: ") + strings.Join(parts, " │ ")
}

func (a *Autosuggester) Complete() string {
    if len(a.suggestions) == 0 {
        return ""
    }
    return a.suggestions[a.selectedIndex].Name
}

func (a *Autosuggester) CycleNext() {
    if len(a.suggestions) > 0 {
        a.selectedIndex = (a.selectedIndex + 1) % len(a.suggestions)
    }
}
```

**Keyboard behavior in editing mode:**

| Key | Condition | Action |
|-----|-----------|--------|
| `Tab` | Suggestions visible | Complete with selected suggestion |
| `Tab` | Already completed | Cycle to next suggestion |
| Any typing | — | Update suggestions based on new input |
| `Escape` | Suggestions visible | Dismiss suggestions (and exit editing) |

### External Dependencies

```go
import (
    // Charm ecosystem
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbles/filepicker"
    "github.com/charmbracelet/bubbles/textinput"
    "github.com/charmbracelet/bubbles/viewport"
    "github.com/charmbracelet/bubbles/key"
    "github.com/charmbracelet/bubbles/help"
    "github.com/charmbracelet/lipgloss"
    
    // CalcMark core
    "github.com/CalcMark/go-calcmark/evaluator"
    "github.com/CalcMark/go-calcmark/classifier"
    "github.com/CalcMark/go-calcmark/validator"
    "github.com/CalcMark/go-calcmark/types"
    
    // Standard library
    "gopkg.in/yaml.v3"  // For frontmatter parsing
)
```

---

## Keyboard Reference

### Universal Keys (Work Everywhere)

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate up/down |
| `←` / `→` | Navigate within line (editing) or scroll (preview) |
| `Enter` | Confirm / start editing |
| `Escape` | Cancel / stop editing / dismiss |
| `Tab` | **In normal mode:** Cycle preview mode |
| `Tab` | **In editing mode:** Autocomplete suggestion |
| `/` | Open command palette |
| `?` | Show help |
| `Ctrl-C` | Quit (with unsaved warning) |
| `PageUp` / `PageDown` | Scroll by screen |
| `Home` / `End` | Jump to top/bottom of document |

### Power User Keys (Editor Normal Mode)

| Key | Action |
|-----|--------|
| `j` / `k` | Move down/up (same as arrows) |
| `h` / `l` | Scroll preview or navigate in line |
| `gg` | Jump to top |
| `G` | Jump to bottom |
| `Ctrl-d` / `Ctrl-u` | Half-page down/up |
| `e` / `i` | Edit current line |
| `o` / `O` | Insert line below/above |
| `dd` | Delete current line |
| `yy` | Yank (copy) line |
| `p` | Paste below |
| `u` | Undo |
| `Ctrl-r` | Redo |
| `gd` | Go to definition |
| `g` | Toggle globals panel |
| `n` / `N` | Next/prev search result |

### Command Palette Commands

| Command | Description |
|---------|-------------|
| `/save` | Save document |
| `/saveas <name>` | Save as new file |
| `/quit` | Quit (warns if unsaved) |
| `/help` | Full help screen |
| `/globals` | Toggle globals panel |
| `/preview` | Cycle preview mode |
| `/preview full` | Set full preview |
| `/preview minimal` | Set minimal preview |
| `/preview hidden` | Hide preview |
| `/find <term>` | Search document |
| `/goto <line>` | Jump to line number |
| `/eval <expr>` | Evaluate expression (quick-eval) |
| `/insert` | Insert last eval result at cursor |
| `/undo` | Undo (discoverable alias for `u`) |
| `/redo` | Redo |
| `/open` | Open file picker |
| `/open <file>` | Open specific file |
| `/new` | New document |
| `/vars` | List all defined variables |

### Mouse Support (Optional)

| Action | Effect |
|--------|--------|
| Click source pane | Move cursor to that line |
| Click preview pane | Move cursor to corresponding source line |
| Click globals item | Jump to definition |
| Scroll wheel | Scroll document |
| Click status bar hint | Execute that command |

---

## Implementation Notes

### File Structure

```
cmd/cm/
├── main.go              # Entry point, argument parsing
├── root.go              # RootModel
├── repl.go              # REPLModel
├── picker.go            # PickerModel
├── editor.go            # EditorModel
├── document.go          # Document type
├── source_pane.go       # SourcePane
├── preview_pane.go      # PreviewPane
├── globals_panel.go     # GlobalsPanel
├── context_footer.go    # ContextFooter
├── command_palette.go   # CommandPalette
├── status_bar.go        # StatusBar
├── help_overlay.go      # HelpOverlay
├── styles.go            # Lipgloss styles
├── keys.go              # Key bindings
├── format.go            # Number/value formatting
└── commands.go          # Command handlers
```

### Key Implementation Details

#### 1. Line-by-Line Re-evaluation

When any line changes, re-evaluate from that line forward (not the whole document). Context flows top-to-bottom.

```go
func (d *Document) OnLineChanged(lineNum int) {
    // Re-evaluate from lineNum to end
    for i := lineNum; i < len(d.lines); i++ {
        d.results[i] = d.evaluateLine(d.lines[i])
    }
}
```

#### 2. Viewport Synchronization

Both panes must scroll together. Use a single scroll offset.

```go
func (m EditorModel) syncScroll() {
    m.previewPane.viewport.YOffset = m.sourcePane.viewport.YOffset
}
```

#### 3. Multi-line Expression Handling

Track which lines are continuations of previous lines.

```go
type LineMetadata struct {
    IsContinuation bool  // true if this continues previous line
    StartsExpr     int   // line number where this expression starts
}
```

#### 4. Frontmatter Parsing

Parse YAML frontmatter on document load.

```go
func parseFrontmatter(content string) (map[string]interface{}, string, error) {
    if !strings.HasPrefix(content, "---\n") {
        return nil, content, nil
    }
    
    end := strings.Index(content[4:], "\n---\n")
    if end == -1 {
        return nil, content, nil
    }
    
    frontmatter := content[4 : 4+end]
    body := content[4+end+5:]
    
    var data map[string]interface{}
    err := yaml.Unmarshal([]byte(frontmatter), &data)
    
    return data, body, err
}
```

#### 5. Undo/Redo Stack

Store document snapshots on each change.

```go
func (m *EditorModel) pushUndo() {
    m.undoStack = append(m.undoStack, m.document.Clone())
    m.redoStack = nil  // clear redo on new change
}

func (m *EditorModel) undo() {
    if len(m.undoStack) == 0 {
        return
    }
    
    m.redoStack = append(m.redoStack, m.document.Clone())
    m.document = m.undoStack[len(m.undoStack)-1]
    m.undoStack = m.undoStack[:len(m.undoStack)-1]
}
```

### Testing Strategy

1. **Unit tests** for Document evaluation logic
2. **Unit tests** for each component's View() output
3. **Integration tests** for state transitions
4. **Golden file tests** for visual output snapshots
5. **Manual testing** for keyboard interaction

### Performance Considerations

1. **Lazy rendering** — Only render visible lines
2. **Incremental re-evaluation** — Don't re-evaluate unchanged lines
3. **Debounced updates** — Don't re-evaluate on every keystroke; debounce by ~50ms
4. **Virtual scrolling** — For documents with 1000+ lines

---

## Appendix: Help Overlay Content

```
┌─────────────────────────── CalcMark Help ───────────────────────────────────┐
│                                                                             │
│  NAVIGATION                           EDITING                               │
│  ──────────                           ───────                               │
│  ↑/↓ or j/k    Move up/down           Enter        Start editing           │
│  ←/→           Move in line           e or i       Start editing           │
│  gg            Jump to top            o            Insert line below       │
│  G             Jump to bottom         O            Insert line above       │
│  Ctrl-d/u      Half-page scroll       dd           Delete line             │
│  PgUp/PgDn     Full-page scroll       Escape       Stop editing            │
│                                       u            Undo                     │
│  COMMANDS                             Ctrl-r       Redo                     │
│  ────────                                                                   │
│  /             Open command palette   PREVIEW                               │
│  /save         Save document          ───────                               │
│  /quit         Quit                   Tab          Cycle preview mode       │
│  /find <term>  Search                 g            Toggle globals panel     │
│  /eval <expr>  Quick evaluate                                               │
│  /help         This help screen       ?            Show this help          │
│                                                                             │
│                         Press any key to close                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Appendix: File Picker View

```
┌─────────────────────────── Open CalcMark File ──────────────────────────────┐
│                                                                             │
│  ~/projects/budgets/                                                        │
│                                                                             │
│    📁 ..                                                                    │
│    📁 2024/                                                                 │
│    📁 2025/                                                                 │
│    📄 q1-budget.cm                                                          │
│  > 📄 q2-budget.cm                   ← selected                             │
│    📄 q3-budget.cm                                                          │
│    📄 q4-budget.cm                                                          │
│    📄 annual-summary.cm                                                     │
│                                                                             │
│                                                                             │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│  ↑/↓ navigate │ Enter open │ Escape cancel │ Backspace parent              │
└─────────────────────────────────────────────────────────────────────────────┘
```

Filter to `.cm`, `.calcmark`, and `.md` files by default.

---

**End of Design Specification**
