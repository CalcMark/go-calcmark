# Phase 6: Differentiators - Research

**Researched:** 2026-02-03
**Domain:** YAML front matter integration, function metadata refactor, TUI autocomplete
**Confidence:** HIGH

## Summary

Phase 6 implements two differentiating features: document-level constants via YAML front matter, and TUI autocomplete for functions/constants/units. Both features build on existing infrastructure.

**Key discovery:** YAML front matter parsing is ALREADY IMPLEMENTED in `spec/document/frontmatter.go` with full support for `globals:` (user-defined constants) and `exchange:` (currency rates). The integration with the evaluator is complete. The primary work is UI exposure (clear error messages, autocomplete for globals).

**Function metadata refactor:** The user has decided (STATE.md) to move from a separate registry to a single source of truth where metadata lives WITH function implementations. This enables compile-time safety, unified data for help/autocomplete, and better error hints.

**Autocomplete:** The `components/suggest.go` already provides `SuggestionSource` interface, `AutosuggestState`, and rendering functions. The work is trigger detection (Tab/Ctrl+Space), popup positioning, and creating suggestion sources for functions, units, and variables.

**Primary recommendation:** Split Phase 6 into two clear plans: (1) Function metadata refactor + YAML front matter error messages, (2) TUI autocomplete implementation.

## Standard Stack

The established libraries/tools for this domain:

### Core (Already In Use)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| gopkg.in/yaml.v3 | v3.0.1 | YAML parsing (frontmatter) | Already a dependency, used by frontmatter.go |
| charmbracelet/bubbletea | v1.3.10 | TUI framework | Standard for Go TUI apps |
| charmbracelet/bubbles | v0.21.0 | TUI components | Provides key handling, styling |
| charmbracelet/lipgloss | v1.x | Styling/layout | Already deeply integrated |

### Supporting (Already In Place)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| components/suggest.go | N/A (internal) | Autocomplete UI | Suggestion rendering, filtering |
| impl/interpreter/registry.go | N/A (internal) | Function metadata | Will be refactored |
| spec/document/frontmatter.go | N/A (internal) | YAML parsing | Already complete |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| gopkg.in/yaml.v3 | adrg/frontmatter | frontmatter lib adds overhead; direct yaml.v3 already works |
| Custom autocomplete | bubbles/textinput suggestions | textinput's built-in suggestions work differently; custom dropdown gives more control |
| Separate FunctionRegistry | Go interfaces with metadata methods | Interfaces add boilerplate; struct-based registration is simpler |

**Installation:**
```bash
# No new dependencies needed - all infrastructure exists
```

## Architecture Patterns

### YAML Front Matter (ALREADY IMPLEMENTED)

The existing implementation in `spec/document/frontmatter.go` is complete and well-tested:

```yaml
---
globals:
  tax_rate: 0.08
  base_price: $100
exchange:
  USD_EUR: 0.92
---
```

**Integration points:**
1. `ParseFrontmatter()` extracts YAML, returns remaining content
2. `Document.ApplyFrontmatter()` injects values into interpreter environment
3. `ParseGlobals()` converts string values to typed CalcMark values
4. Evaluator calls `ApplyFrontmatter()` before block evaluation

**Gap:** Error messages for malformed YAML need line numbers (currently returns string error).

### Function Metadata Refactor Pattern

**Current state:** `FunctionRegistry` in `registry.go` is separate from implementations in `functions.go`. AST-parsing test detects drift at runtime.

**Recommended pattern: Struct-based registration**

```go
// In functions.go - metadata WITH implementation
type FunctionDef struct {
    Name        string
    Synonyms    []string
    Description string
    Signature   string
    Category    string
    Eval        func(interp *Interpreter, args []types.Type) (types.Type, error)
}

var BuiltinFunctions = []FunctionDef{
    {
        Name:        "avg",
        Synonyms:    []string{"average", "mean"},
        Description: "Calculate the average of numbers",
        Signature:   "avg(value1, value2, ...)",
        Category:    "Math",
        Eval:        evalAverage,
    },
    // ... more functions
}

// evalFunctionCall uses BuiltinFunctions instead of switch
func (interp *Interpreter) evalFunctionCall(f *ast.FunctionCall) (types.Type, error) {
    for _, fn := range BuiltinFunctions {
        if f.Name == fn.Name || contains(fn.Synonyms, f.Name) {
            return fn.Eval(interp, args)
        }
    }
    return nil, fmt.Errorf("unknown function: %s", f.Name)
}
```

**Benefits:**
1. Compile-time errors if function added without metadata
2. Single source of truth for help, autocomplete, diagnostics
3. Function signatures available for error hints

### Autocomplete Pattern

```
[User types "= av"]
      |
      v
[Trigger detection] -- Tab key or Ctrl+Space or automatic (after =)
      |
      v
[Extract prefix] -- "av" from cursor position
      |
      v
[Query suggestion sources] -- FunctionSuggestionSource, UnitSuggestionSource, VariableSuggestionSource
      |
      v
[Filter and rank] -- Prefix match + synonym match
      |
      v
[Render dropdown] -- RenderDropdownSuggestions() positioned at cursor
      |
      v
[Selection] -- Up/Down to navigate, Tab/Enter to accept, Esc to dismiss
      |
      v
[Insert completion] -- Replace prefix with selected suggestion
```

### Recommended Project Structure

```
impl/interpreter/
├── functions.go          # Function implementations + metadata (refactored)
├── function_registry.go  # Registry lookup helpers (renamed from registry.go)
├── registry_test.go      # Validates completeness (simplified)

cmd/calcmark/tui/
├── components/
│   ├── suggest.go        # Existing suggestion rendering
│   └── autocomplete.go   # NEW: Suggestion sources for functions/units/variables
├── editor/
│   ├── model.go          # Add StateAutocomplete mode, dropdown state
│   └── view.go           # Render autocomplete dropdown

spec/document/
├── frontmatter.go        # Already complete
├── frontmatter_test.go   # Already complete
├── globals.go            # Already complete
```

### Anti-Patterns to Avoid

- **Don't implement custom YAML parsing:** frontmatter.go already handles `---` delimiters correctly
- **Don't use interface-based function registration:** Adds boilerplate without compile-time safety benefit
- **Don't trigger autocomplete on every keystroke:** Use explicit triggers (Tab, Ctrl+Space) or smart detection (after `=`)
- **Don't build separate popup layer:** Render dropdown as part of View() with lipgloss positioning

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| YAML front matter parsing | Custom `---` delimiter handling | `spec/document/frontmatter.go` | Already tested, handles edge cases |
| Prefix matching for autocomplete | Custom string matching | `components.FilterSuggestions()` | Already implemented with case-insensitive matching |
| Suggestion rendering | Custom dropdown rendering | `components.RenderDropdownSuggestions()` | Already styled, handles truncation |
| Function metadata | Hardcoded help strings | Struct-based registration | Single source of truth |
| Input state management | Custom state flags | InputState enum (StateAutocomplete) | Already exists in model.go |

**Key insight:** 80% of the infrastructure exists. The work is integration and wiring, not new implementation.

## Common Pitfalls

### Pitfall 1: Ignoring Existing Frontmatter Implementation
**What goes wrong:** Building new YAML parsing when frontmatter.go is complete
**Why it happens:** Not reading the codebase before implementing
**How to avoid:** frontmatter.go has full parsing, globals.go has type conversion
**Warning signs:** Writing `---` delimiter detection code

### Pitfall 2: Triggering Autocomplete Too Aggressively
**What goes wrong:** Autocomplete popup appears while typing prose (in text blocks)
**Why it happens:** Trigger detection doesn't consider block context
**How to avoid:** Only trigger in calc blocks (after `=` or on explicit Tab/Ctrl+Space)
**Warning signs:** Suggestions appearing while typing markdown text

### Pitfall 3: Function Metadata Without Compile-Time Checks
**What goes wrong:** Adding functions to BuiltinFunctions without metadata causes runtime errors
**Why it happens:** Struct-based registration requires discipline
**How to avoid:** Delete the old registry.go test that AST-parses functions.go; the new structure IS the test
**Warning signs:** Functions returning "unknown function" at runtime

### Pitfall 4: Autocomplete Steals Navigation Keys
**What goes wrong:** Arrow keys move in dropdown when user wants to navigate document
**Why it happens:** StateAutocomplete captures keys intended for editing
**How to avoid:** Escape dismisses dropdown cleanly; only Tab/Enter accept selection
**Warning signs:** Users can't arrow past the dropdown position

### Pitfall 5: YAML Error Messages Without Line Numbers
**What goes wrong:** "invalid YAML" without position information
**Why it happens:** yaml.v3 errors contain line info but need extraction
**How to avoid:** Extract yaml.TypeError line info and format user-friendly message
**Warning signs:** Error messages that don't tell users WHERE the problem is

## Code Examples

Verified patterns from the existing codebase:

### Frontmatter Usage (Existing)
```go
// Source: spec/document/frontmatter.go
fm, remaining, err := ParseFrontmatter(source)
if err != nil {
    return nil, fmt.Errorf("frontmatter: %w", err)
}

// Frontmatter globals are available via fm.Globals map[string]string
// Applied to evaluator via doc.ApplyFrontmatter(env)
```

### Suggestion Source Interface (Existing)
```go
// Source: cmd/calcmark/tui/components/suggest.go
type SuggestionSource interface {
    GetSuggestions(prefix string) []Suggestion
}

type Suggestion struct {
    Name        string
    Category    string
    Description string
    Syntax      string
}
```

### Function Registry Pattern (Proposed)
```go
// Source: impl/interpreter/functions.go (to be refactored)
type FunctionDef struct {
    Name        string
    Synonyms    []string
    Description string
    Signature   string
    Category    string
    // Eval is the implementation
    Eval func(interp *Interpreter, args []types.Type) (types.Type, error)
}

// BuiltinFunctions is the single source of truth
var BuiltinFunctions = []FunctionDef{
    {
        Name:        "avg",
        Synonyms:    []string{"average", "mean"},
        Description: "Calculate the average of numbers",
        Signature:   "avg(value1, value2, ...)",
        Category:    "Math",
        Eval:        func(interp *Interpreter, args []types.Type) (types.Type, error) {
            return evalAverage(args)
        },
    },
    // ... all 12 functions with metadata
}
```

### Autocomplete State (Proposed)
```go
// In cmd/calcmark/tui/editor/model.go
const (
    StateDefault      InputState = iota
    StateAutocomplete            // NEW: Autocomplete dropdown active
    // ... other states
)

type Model struct {
    // ... existing fields

    // Autocomplete state
    autocompleteState components.AutosuggestState
    suggestionSources []components.SuggestionSource
}

func (m Model) handleTabKey() (tea.Model, tea.Cmd) {
    if m.mode == StateAutocomplete {
        // Accept selection
        return m.acceptAutocomplete()
    }

    // Trigger autocomplete
    prefix := m.getCurrentWordPrefix()
    suggestions := m.getSuggestions(prefix)
    if len(suggestions) > 0 {
        m.mode = StateAutocomplete
        m.autocompleteState = components.AutosuggestState{
            Suggestions: suggestions,
            Selected:    0,
            Visible:     true,
            Prefix:      prefix,
        }
    }
    return m, nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Separate registry file | Metadata with implementation | Phase 6 (user decision) | Compile-time safety, unified data |
| No autocomplete | Tab-triggered suggestions | Phase 6 | IDE-like experience |
| Silent frontmatter errors | Line-numbered error messages | Phase 6 | Better user experience |

**Current in CalcMark:**
- Frontmatter parsing: Complete
- Function metadata: Separate registry (needs refactor)
- Autocomplete: Components exist, not wired in

## Open Questions

Things that couldn't be fully resolved:

1. **Autocomplete trigger keys**
   - What we know: Tab is common, Ctrl+Space is IDE standard
   - What's unclear: Should autocomplete auto-trigger after `=`? After 2 characters?
   - Recommendation: Start with explicit Tab trigger, add auto-trigger if users request

2. **Synonym display in autocomplete**
   - What we know: User wants "mean" to suggest "average (mean)"
   - What's unclear: Format for multiple synonyms? "avg (average, mean)" vs "avg/average/mean"?
   - Recommendation: Use "avg (mean, average)" format - primary name first, parenthetical synonyms

3. **Autocomplete in calculation context**
   - What we know: Should work after `=` in a calculation
   - What's unclear: Should it work mid-expression? After `+`? In function arguments?
   - Recommendation: Start with word-boundary trigger (after `=`, space, operators), expand based on feedback

## Sources

### Primary (HIGH confidence)
- spec/document/frontmatter.go - Full implementation examined
- spec/document/frontmatter_test.go - 600+ lines of tests
- spec/document/globals.go - Type conversion for globals
- impl/interpreter/registry.go - Current function metadata
- impl/interpreter/functions.go - Current function implementations
- cmd/calcmark/tui/components/suggest.go - Suggestion rendering infrastructure
- .planning/STATE.md - User decision on function metadata refactor

### Secondary (MEDIUM confidence)
- [Bubble Tea pkg.go.dev](https://pkg.go.dev/github.com/charmbracelet/bubbletea) - TUI patterns
- [bubbles/textinput suggestions](https://pkg.go.dev/github.com/charmbracelet/bubbles/textinput) - Built-in autocomplete pattern

### Tertiary (LOW confidence)
- WebSearch results for "Bubble Tea autocomplete dropdown" - Community patterns

## Metadata

**Confidence breakdown:**
- YAML Frontmatter: HIGH - Already implemented, well-tested
- Function metadata refactor: HIGH - Clear pattern, user decision made
- Autocomplete: MEDIUM - Infrastructure exists, integration is new

**Research date:** 2026-02-03
**Valid until:** 30 days (stable domain, well-understood patterns)

## Key Implementation Notes for Planner

### YAML Frontmatter (Minimal Work)

1. **Parsing is DONE** - `ParseFrontmatter()` works
2. **Integration is DONE** - `ApplyFrontmatter()` injects into evaluator
3. **What's needed:**
   - Improve error messages with line numbers from yaml.v3 errors
   - Test that SC1 "tax_rate available in calculations" already passes
   - Catwalk test for malformed YAML error display

### Function Metadata Refactor (Medium Work)

1. **Create FunctionDef struct** with Eval field
2. **Migrate 12 functions** from switch to BuiltinFunctions slice
3. **Update evalFunctionCall** to iterate BuiltinFunctions
4. **Update help commands** to use BuiltinFunctions
5. **Delete old registry_test.go AST parsing** - no longer needed

### Autocomplete (Larger Work)

1. **Create FunctionSuggestionSource** using BuiltinFunctions
2. **Create UnitSuggestionSource** using spec/units/canonical.go
3. **Create VariableSuggestionSource** from evaluator environment
4. **Add StateAutocomplete** to InputState enum
5. **Wire Tab/Ctrl+Space** handlers to trigger autocomplete
6. **Render dropdown** in View() when StateAutocomplete
7. **Handle Up/Down/Enter/Esc** for dropdown navigation
