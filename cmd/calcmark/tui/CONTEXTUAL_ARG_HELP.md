# Contextual Argument Help Plan

## Problem
When typing `accumulate(`, the user needs to know:
1. What type each argument expects (rate, time_period, quantity, etc.)
2. How to create values of that type in CalcMark syntax

Currently we show `accumulate(rate, duration)` but no examples of how to write a rate or duration.

## Goal
Show contextual examples based on argument position:
```
accumulate(|
           ↑ rate: e.g., 10 MB/s, 5 km/h, 100 requests/minute
```

```
accumulate(10 MB/s, |
                    ↑ duration: e.g., 2 hours, 30 minutes, 1 day
```

## Type System Analysis

### Core Value Types in CalcMark
From `spec/types/` and `impl/types/`:

| Type | CalcMark Syntax Examples | Description |
|------|-------------------------|-------------|
| `number` | `42`, `3.14`, `1e6` | Plain numbers |
| `quantity` | `10 meters`, `5.5 kg`, `100 MB` | Number + unit |
| `rate` | `10 MB/s`, `60 km/h`, `5 requests/minute` | Quantity per time |
| `duration` | `2 hours`, `30 minutes`, `1.5 days` | Time quantity |
| `percentage` | `15%`, `0.5%` | Percentage values |
| `currency` | `$100`, `€50` | Money values |

### Function Argument Types
From `impl/interpreter/registry.go` - functions have typed parameters:

```go
type FunctionDef struct {
    Name        string
    Synonyms    []string
    Category    string
    Description string
    Signature   string      // e.g., "avg(values...)"
    ParamTypes  []ParamType // NEW: type info per parameter
    // ...
}

type ParamType struct {
    Name     string   // "rate", "duration", "values"
    Type     string   // "rate", "duration", "number", "quantity", "any"
    Variadic bool     // true for "values..."
    Examples []string // ["10 MB/s", "5 km/h"]
}
```

## Implementation Plan

### Phase 1: Extend Function Metadata
1. Add `ParamTypes` to `FunctionDef` in `impl/interpreter/registry.go`
2. Populate examples for each parameter type
3. Create `TypeExamples` map for common types

```go
var TypeExamples = map[string][]string{
    "number":     {"42", "3.14", "1000"},
    "quantity":   {"10 meters", "5 kg", "100 MB"},
    "rate":       {"10 MB/s", "60 km/h", "5 req/min"},
    "duration":   {"2 hours", "30 min", "1 day"},
    "percentage": {"15%", "50%", "0.5%"},
    "any":        {"<any value>"},
}
```

### Phase 2: Detect Cursor Context
In `editor/model.go`, add function to detect:
- Are we inside a function call? `func_name(`
- Which argument position? Count commas before cursor
- What's the expected type for that position?

```go
type CursorContext struct {
    InFunctionCall bool
    FunctionName   string
    ArgIndex       int      // 0-based
    ExpectedType   string   // "rate", "duration", etc.
    Examples       []string // ["10 MB/s", "5 km/h"]
}

func (m *Model) GetCursorContext() CursorContext
```

### Phase 3: Update Context Footer
When cursor is inside a function call:
- Show: `arg_name (type): example1, example2, example3`
- Highlight which argument position

```
accumulate(|
  → rate: 10 MB/s, 60 km/h, 5 requests/minute
```

### Phase 4: Autocomplete for Types
When user presses TAB inside function argument:
- If they've typed a number, suggest units that match the expected type
- If expected type is `rate`, suggest rate-compatible units
- If expected type is `duration`, suggest time units

## Files to Modify

1. **`impl/interpreter/registry.go`**
   - Add `ParamTypes` to `FunctionDef`
   - Add `TypeExamples` map
   - Populate metadata for all functions

2. **`cmd/calcmark/tui/editor/context.go`** (NEW)
   - `GetCursorContext()` - parse cursor position
   - `ParseFunctionCall()` - extract function name and arg position

3. **`cmd/calcmark/tui/components/contextfooter.go`**
   - Add `ArgumentContext` to state
   - Render argument help when inside function call

4. **`cmd/calcmark/tui/editor/autocomplete.go`**
   - Type-aware suggestions based on expected argument type

## Example Interaction

```
User types: accumulate(
Footer shows: → rate: 10 MB/s, 60 km/h, 5 requests/minute

User types: accumulate(10 MB/s,
Footer shows: → duration: 2 hours, 30 minutes, 1 day

User types: accumulate(10 MB/s, 2 h
Autocomplete popup: hours, hectares (filtered by 'h')
Footer shows: → duration: 2 hours, 30 minutes, 1 day
```

## Success Criteria
- [ ] Function metadata includes parameter types and examples
- [ ] Cursor context detection works for nested function calls
- [ ] Context footer shows relevant examples for current argument
- [ ] Autocomplete suggests type-appropriate completions
- [ ] Works for all built-in functions

## Priority Order
1. Phase 1 (metadata) - foundation
2. Phase 2 (cursor context) - detection
3. Phase 3 (footer display) - immediate value
4. Phase 4 (smart autocomplete) - enhanced UX
