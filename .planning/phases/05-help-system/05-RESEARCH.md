# Phase 5: Help System - Research

**Researched:** 2026-02-03
**Domain:** CLI help commands, shell completions, TUI help overlay, status bar
**Confidence:** HIGH

## Summary

This phase implements comprehensive help and discoverability features for CalcMark. The research covers four distinct areas: CLI help commands (HELP-02, HELP-03, HELP-04, HELP-05), shell completions (HELP-01), TUI help overlay (HELP-06), and status bar enhancements (HELP-07, HELP-08).

The codebase already uses Cobra (v1.10.2) for CLI commands and Bubble Tea (v1.3.10) with Bubbles (v0.21.0) for the TUI. Key findings show that shell completions are currently disabled in root.go (`CompletionOptions.DisableDefaultCmd = true`), functions are defined in a switch statement in `impl/interpreter/functions.go`, and the existing status bar component in `cmd/calcmark/tui/components/statusbar.go` already supports most required fields.

**Primary recommendation:** Use Cobra's built-in completion generation (remove DisableDefaultCmd), create a function registry that centralizes function metadata, leverage the existing bubbles/help package for the TUI overlay, and extend the existing StatusBarState to track evaluation progress.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| spf13/cobra | v1.10.2 | CLI framework with shell completion | Already in use, industry standard |
| charmbracelet/bubbles/help | v0.21.0 | Help panel component for Bubble Tea | Official component, already available |
| charmbracelet/bubbles/key | v0.21.0 | Key binding definitions | Provides KeyMap interface for help |
| charmbracelet/lipgloss | v1.1.1 | Terminal styling | Already in use for all TUI styling |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| text/tabwriter | stdlib | Formatted text tables | CLI help output alignment |
| charmbracelet/glamour | v0.10.0 | Markdown rendering | Rich help text formatting |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| bubbles/help | Custom overlay | More control but duplicates work |
| tabwriter | lipgloss | lipgloss better for TUI, tabwriter better for CLI pipes |

**Installation:**
All dependencies already present in go.mod. No new packages needed.

## Architecture Patterns

### Recommended Project Structure
```
cmd/calcmark/
├── cmd/
│   ├── root.go           # Enable completions, add help command
│   ├── help.go           # NEW: help subcommand (functions, constants)
│   └── completion.go     # NEW: explicit completion command
├── tui/
│   ├── editor/
│   │   ├── model.go      # Add StateHelp mode, evaluation tracking
│   │   ├── view.go       # Add help overlay rendering
│   │   └── help_overlay.go # NEW: help overlay component
│   ├── components/
│   │   └── statusbar.go  # Add EvalInProgress field
│   └── shared/
│       └── keys.go       # Already has KeyMap, add FullHelp/ShortHelp
impl/interpreter/
├── registry.go           # NEW: function metadata registry
└── functions.go          # Existing function implementations
spec/units/
└── canonical.go          # Existing unit definitions (has Description field)
```

### Pattern 1: Function Registry for Help Discovery
**What:** Centralized metadata registry for all CalcMark functions
**When to use:** When extracting function names, descriptions, and synonyms for `cm help functions`
**Example:**
```go
// impl/interpreter/registry.go

// FunctionInfo contains metadata about a CalcMark function
type FunctionInfo struct {
    Name        string   // Primary name (e.g., "avg")
    Synonyms    []string // Alternative names (e.g., "average")
    Description string   // Human-readable description
    Signature   string   // e.g., "avg(value1, value2, ...)"
    Category    string   // e.g., "Math", "Network", "Storage"
}

// FunctionRegistry is the single source of truth for function metadata
var FunctionRegistry = []FunctionInfo{
    {
        Name:        "avg",
        Synonyms:    []string{"average"},
        Description: "Calculate the average of numbers",
        Signature:   "avg(value1, value2, ...)",
        Category:    "Math",
    },
    {
        Name:        "sqrt",
        Synonyms:    []string{},
        Description: "Calculate square root of a number",
        Signature:   "sqrt(value)",
        Category:    "Math",
    },
    // ... more functions
}

// GetAllFunctions returns all registered functions sorted by name
func GetAllFunctions() []FunctionInfo {
    sorted := make([]FunctionInfo, len(FunctionRegistry))
    copy(sorted, FunctionRegistry)
    sort.Slice(sorted, func(i, j int) bool {
        return sorted[i].Name < sorted[j].Name
    })
    return sorted
}
```

### Pattern 2: Cobra Shell Completion
**What:** Enable Cobra's built-in completion generation
**When to use:** HELP-01 - shell completions
**Example:**
```go
// cmd/calcmark/cmd/root.go
func init() {
    // REMOVE this line to enable default completion command:
    // rootCmd.CompletionOptions.DisableDefaultCmd = true

    // OR create explicit completion command for more control
}

// cmd/calcmark/cmd/completion.go
var completionCmd = &cobra.Command{
    Use:   "completion [bash|zsh|fish|powershell]",
    Short: "Generate shell completion scripts",
    Long: `Generate shell completion scripts for CalcMark.

To load completions:

Bash:
  $ source <(cm completion bash)
  # Or add to ~/.bashrc for persistence

Zsh:
  $ cm completion zsh > "${fpath[1]}/_cm"
  # Then restart shell

Fish:
  $ cm completion fish > ~/.config/fish/completions/cm.fish

PowerShell:
  PS> cm completion powershell | Out-String | Invoke-Expression
`,
    DisableFlagsInUseLine: true,
    ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
    Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
    RunE: func(cmd *cobra.Command, args []string) error {
        switch args[0] {
        case "bash":
            return rootCmd.GenBashCompletion(os.Stdout)
        case "zsh":
            return rootCmd.GenZshCompletion(os.Stdout)
        case "fish":
            return rootCmd.GenFishCompletion(os.Stdout, true)
        case "powershell":
            return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
        default:
            return fmt.Errorf("unknown shell: %s", args[0])
        }
    },
}
```

### Pattern 3: TUI Help Overlay with bubbles/help
**What:** Modal overlay using InputState to manage help display
**When to use:** HELP-06 - in-TUI help overlay
**Example:**
```go
// cmd/calcmark/tui/editor/model.go
import "github.com/charmbracelet/bubbles/help"

// InputState already exists with StateHelp
const (
    StateDefault      InputState = iota
    StateGlobals
    StateHelp         // Already defined - use this for help overlay
    // ...
)

// Model additions
type Model struct {
    // ... existing fields ...
    helpModel help.Model // Add help component
}

// NewModel constructor
func New(doc *document.Document) Model {
    m := Model{
        // ... existing fields ...
        helpModel: help.New(),
    }
    m.helpModel.ShowAll = true // Start with full help
    return m
}

// Update handler
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    // Check for help key (F1 or ?)
    switch {
    case key.Matches(msg, m.keys.Help):
        if m.mode == StateHelp {
            m.mode = StateDefault // Toggle off
        } else {
            m.mode = StateHelp // Toggle on
        }
        return m, nil
    }
    // ... existing handlers ...
}

// View renders help overlay when StateHelp active
func (m Model) View() string {
    // ... existing rendering ...

    if m.mode == StateHelp {
        // Render help overlay on top
        helpView := m.renderHelpOverlay()
        // Use lipgloss.Place to position overlay
        return lipgloss.Place(m.width, m.height,
            lipgloss.Center, lipgloss.Center,
            helpView,
            lipgloss.WithWhitespaceChars(" "),
            lipgloss.WithWhitespaceForeground(lipgloss.Color("237")),
        )
    }
    // ... existing rendering ...
}

func (m Model) renderHelpOverlay() string {
    // Use bubbles/help with custom KeyMap
    return m.helpModel.View(m.keys)
}
```

### Pattern 4: Status Bar Evaluation Tracking
**What:** Track evaluation state with debounce message
**When to use:** HELP-07, HELP-08 - status bar with eval progress
**Example:**
```go
// cmd/calcmark/tui/components/statusbar.go
type StatusBarState struct {
    // ... existing fields ...
    EvalInProgress bool  // NEW: true when evaluation is pending
    Column         int   // NEW: cursor column (for line:col display)
}

// RenderStatusBar - modify center section
center := style.Position.Render(
    fmt.Sprintf("L%d:%d | %d calcs", state.Line, state.Column, state.CalcCount),
)

// If evaluating, show indicator
if state.EvalInProgress {
    center = style.Position.Render(
        fmt.Sprintf("L%d:%d | EVAL...", state.Line, state.Column),
    )
}

// cmd/calcmark/tui/editor/model.go
func (m *Model) GetStatusBarState() components.StatusBarState {
    return components.StatusBarState{
        // ... existing fields ...
        EvalInProgress: m.userIsTyping, // userIsTyping already tracks debounce state
        Column:         m.cursorCol + 1, // 1-indexed for display
    }
}
```

### Anti-Patterns to Avoid
- **Hand-rolling completion scripts:** Cobra generates correct, maintained scripts
- **Hardcoding function lists:** Use registry pattern for single source of truth
- **Rendering help inside View():** Use InputState to toggle, not conditional rendering
- **Blocking evaluation tracking:** Use existing debounce pattern, not new timers

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Shell completions | Custom bash/zsh scripts | Cobra GenBashCompletion | Maintained, handles edge cases |
| Help text formatting | Custom string building | tabwriter (CLI) or lipgloss (TUI) | Proper alignment, terminal-aware |
| Key binding help | Manual list maintenance | bubbles/help + KeyMap interface | Auto-updates when keys change |
| Modal overlay positioning | Manual coordinate math | lipgloss.Place | Handles centering, margins |

**Key insight:** The existing codebase already has most infrastructure in place - the task is connecting components, not building new ones.

## Common Pitfalls

### Pitfall 1: Completion Scripts Not Working in Piped Output
**What goes wrong:** Completion works in terminal but not when piped to file
**Why it happens:** Shell-specific quirks with output buffering
**How to avoid:** Use Cobra's GenXxxCompletion methods which handle this correctly
**Warning signs:** Works in terminal, fails in `cm completion bash > script.sh`

### Pitfall 2: Help Output Pagination Breaks
**What goes wrong:** Long help output doesn't work with `less` or `more`
**Why it happens:** ANSI codes or improper line buffering
**How to avoid:** Ensure output is plain text or use standard pager-compatible formatting
**Warning signs:** Colors disappear in pager, or output appears all at once

### Pitfall 3: Function Registry Drift
**What goes wrong:** Registry doesn't match actual function implementations
**Why it happens:** Adding new functions without updating registry
**How to avoid:** Add test that verifies registry matches switch statement in evalFunctionCall
**Warning signs:** Help shows function that doesn't work, or function exists but not in help

### Pitfall 4: Status Bar Height Changes
**What goes wrong:** Flickering or artifacts when status bar content changes
**Why it happens:** Bubbletea rendering issues when view height varies
**How to avoid:** Keep statusBarHeight constant (already 2 in statusbar.go)
**Warning signs:** Visual glitches when switching between "EVAL..." and normal display

### Pitfall 5: Help Overlay Keyboard Conflicts
**What goes wrong:** Help key toggles don't work or conflict with editing
**Why it happens:** Key handling order in Update function
**How to avoid:** Check help toggle BEFORE mode-specific handling
**Warning signs:** Can open help but not close it, or typing "?" in editor opens help

## Code Examples

Verified patterns from official sources:

### Cobra Completion Command (From Cobra docs)
```go
// Source: https://cobra.dev/docs/how-to-guides/shell-completion/
var completionCmd = &cobra.Command{
    Use:   "completion [bash|zsh|fish|powershell]",
    Short: "Generate completion script",
    Long: `To load completions:

Bash:
  $ source <(cm completion bash)

Zsh:
  $ cm completion zsh > "${fpath[1]}/_cm"

Fish:
  $ cm completion fish | source

PowerShell:
  PS> cm completion powershell | Out-String | Invoke-Expression
`,
    DisableFlagsInUseLine: true,
    ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
    Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
    RunE: func(cmd *cobra.Command, args []string) error {
        switch args[0] {
        case "bash":
            return cmd.Root().GenBashCompletion(os.Stdout)
        case "zsh":
            return cmd.Root().GenZshCompletion(os.Stdout)
        case "fish":
            return cmd.Root().GenFishCompletion(os.Stdout, true)
        case "powershell":
            return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
        }
        return nil
    },
}

func init() {
    rootCmd.AddCommand(completionCmd)
}
```

### Bubbles Help Integration (From Bubbles docs)
```go
// Source: https://pkg.go.dev/github.com/charmbracelet/bubbles/help
import (
    "github.com/charmbracelet/bubbles/help"
    "github.com/charmbracelet/bubbles/key"
)

// KeyMap satisfies help.KeyMap interface
type keyMap struct {
    Up    key.Binding
    Down  key.Binding
    Help  key.Binding
    Quit  key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
    return []key.Binding{k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
    return [][]key.Binding{
        {k.Up, k.Down},
        {k.Help, k.Quit},
    }
}

var keys = keyMap{
    Up: key.NewBinding(
        key.WithKeys("up", "k"),
        key.WithHelp("↑/k", "move up"),
    ),
    Down: key.NewBinding(
        key.WithKeys("down", "j"),
        key.WithHelp("↓/j", "move down"),
    ),
    Help: key.NewBinding(
        key.WithKeys("?"),
        key.WithHelp("?", "toggle help"),
    ),
    Quit: key.NewBinding(
        key.WithKeys("q", "esc", "ctrl+c"),
        key.WithHelp("q", "quit"),
    ),
}

type model struct {
    keys keyMap
    help help.Model
}

func (m model) View() string {
    return m.help.View(m.keys)
}
```

### Extracting Units from Canonical Registry
```go
// spec/units/canonical.go already provides this structure:
type UnitMapping struct {
    Canonical   string   // e.g., "meter"
    Symbol      string   // e.g., "m"
    Aliases     []string // e.g., ["meter", "meters", "metre"]
    System      string   // e.g., "SI"
    Quantity    string   // e.g., "Length"
    Description string   // e.g., "SI base unit of length"
}

// For help output, iterate StandardUnits:
func GetAllUnits() []UnitMapping {
    seen := make(map[string]bool)
    var units []UnitMapping
    for _, u := range StandardUnits {
        if !seen[u.Canonical] {
            units = append(units, u)
            seen[u.Canonical] = true
        }
    }
    sort.Slice(units, func(i, j int) bool {
        return units[i].Canonical < units[j].Canonical
    })
    return units
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual completion scripts | Cobra built-in generation | Cobra 1.0+ | Automatic multi-shell support |
| Custom help rendering | bubbles/help component | Bubbles 0.14+ | Consistent styling, auto-truncation |
| Vim-style modes | InputState for overlays | CalcMark current | Non-modal editing with overlay support |

**Deprecated/outdated:**
- Cobra's GenBashCompletionFile - use GenBashCompletion to stdout, then redirect
- Manual ANSI escape codes - use lipgloss for all styling

## Open Questions

Things that couldn't be fully resolved:

1. **Help Key Binding Choice**
   - What we know: `?` is common for help, F1 is standard in some apps
   - What's unclear: Whether `?` conflicts with calc expressions
   - Recommendation: Use F1 as primary, `?` only when not in calc context

2. **Constants Scope**
   - What we know: spec/units/canonical.go has unit constants with descriptions
   - What's unclear: Are there other "constants" beyond units (math constants like pi)?
   - Recommendation: Start with units, add math constants if interpreter has them

3. **Function Synonym Source**
   - What we know: functions.go has `case "avg", "average":` showing synonyms
   - What's unclear: Is there a formal synonym spec or just code convention?
   - Recommendation: Extract from code, document in registry as source of truth

## Sources

### Primary (HIGH confidence)
- Cobra v1.10.2 shell completion: https://cobra.dev/docs/how-to-guides/shell-completion/
- Bubbles help package: https://pkg.go.dev/github.com/charmbracelet/bubbles/help
- CalcMark codebase: `/Users/bitsbyme/projects/go-calcmark/`

### Secondary (MEDIUM confidence)
- Cobra GitHub: https://github.com/spf13/cobra
- Bubbletea examples: https://github.com/charmbracelet/bubbletea/blob/main/examples/help/main.go

### Tertiary (LOW confidence)
- None - all findings verified with official documentation

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - using existing dependencies
- Architecture: HIGH - patterns match existing codebase style
- Pitfalls: HIGH - verified against Bubble Tea rendering behavior

**Research date:** 2026-02-03
**Valid until:** 2026-03-03 (stable libraries, low churn expected)
