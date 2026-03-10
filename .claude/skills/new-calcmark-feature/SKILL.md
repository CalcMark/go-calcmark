---
name: new-calcmark-feature
description: Implement a new CalcMark language feature end-to-end. Covers the full 8-layer stack from lexer token through TUI autosuggest, with TDD throughout.
allowed-tools: Agent,Bash(task:*),Bash(go:*),Bash(echo:*),Bash(./cm:*),Read,Edit,Write,Glob,Grep
---

# New CalcMark Feature Implementation

Step-by-step guide for adding a new function, keyword, operator, or type to the CalcMark language. Every feature touches up to 8 layers — skipping any one causes silent bugs caught only by cross-layer consistency tests.

## Pre-flight

Before writing any code:

1. **Read the relevant existing code** — understand the pattern by reading an analogous feature (e.g., `sqrt` for single-arg functions, `avg` for variadic, `compound` for NL + canonical hybrid).
2. **Decide: canonical only, NL only, or both?** — Most functions have a canonical form `func(args)`. Some also have NL syntax (`average of`, `sum of`, `square root of`). Decide upfront.
3. **Decide: does this need new type coercion?** — If the feature changes how types interact in binary operations (e.g., `Quantity * Currency`), that's a separate change in `impl/interpreter/operators.go`.

## Layer Reference

| # | Layer | File(s) | Purpose |
|---|-------|---------|---------|
| 1 | Lexer token | `spec/lexer/token.go` | Define `FUNC_<NAME>` constant |
| 2 | Lexer keyword | `spec/lexer/lexer.go` | Map string → token in `ReservedKeywords` |
| 3 | Parser dispatch | `spec/parser/rdparser.go` | Add token to `parsePrimary()` match + arg validation in `parseFunctionCall()` |
| 4 | Classifier | `spec/classifier/classifier.go` | Add token to `containsFunctions()` |
| 5 | Detector | `spec/document/detector.go` | Add token to `isFunctionToken()` |
| 6 | Semantic spec | `spec/semantic/function_types.go` | Add `FunctionSpec` with parameter types |
| 7 | Interpreter | `impl/interpreter/functions.go` | Add `FunctionDef` to `BuiltinFunctions`, eval function to `functionEvalMap`, and implement eval logic |
| 8 | Feature registry | `spec/features/registry.go` | Add `Feature` entry for help/autocomplete |

### Additional layers (when applicable)

| Layer | File(s) | When needed |
|-------|---------|-------------|
| NL lexer tokens | `spec/lexer/token.go` + multi-token scanning | NL alias (`average of`, `sum of`) |
| NL parser | `spec/parser/nl_functions.go` | NL function dispatch |
| Type coercion | `impl/interpreter/operators.go` | New cross-type arithmetic rules |
| Golden test | `testdata/eval/success/features/<name>.cm` | Always — demonstrates the feature for the test harness |
| Registry count | `impl/interpreter/registry_test.go` | Always — increment `expectedCount` |

## Execution Steps (TDD)

**IMPORTANT:** Follow TDD strictly. Write failing tests before any production code.

### Step 1: Write failing tests

Write tests across all affected layers before implementing anything. This ensures you know exactly what "done" looks like.

#### Parser test (`spec/parser/parser_function_test.go`)

```go
func TestParse<Name>Function(t *testing.T) {
    // Test that "<name>(<arg>)" parses to *ast.FunctionCall with correct name and arg count
}

func TestParse<Name>FunctionErrors(t *testing.T) {
    // Test that wrong arg counts produce parse errors
    // Remember: empty arg list returns early in parseFunctionCall() — check BOTH paths
}
```

#### Interpreter test (`impl/interpreter/<name>_test.go`)

Use the `interpreter_test` package (external test):

```go
package interpreter_test

import (
    "testing"
    "github.com/CalcMark/go-calcmark/impl/interpreter"
    "github.com/CalcMark/go-calcmark/spec/parser"
    "github.com/CalcMark/go-calcmark/spec/types"
)

func Test<Name>Function(t *testing.T) {
    // Parse input, create interpreter, eval, check result type and value
    nodes, err := parser.Parse(input)
    interp := interpreter.NewInterpreter()
    results, err := interp.Eval(nodes)
    // Assert on results[len(results)-1]
}
```

#### Golden test file (`testdata/eval/success/features/<name>.cm`)

Write a `.cm` file exercising the feature with markdown headings and various input types. This is evaluated by `TestEvalFilesEvaluate` automatically.

#### Type coercion test (if applicable)

Add to the interpreter test file — test both operand orderings (e.g., `Quantity * Currency` AND `Currency * Quantity`).

### Step 2: Run tests, verify they fail

```bash
go test ./spec/parser/ -run TestParse<Name> -count=1
go test ./impl/interpreter/ -run Test<Name> -count=1
```

All new tests should fail. If any pass, the test isn't testing the right thing.

### Step 3: Implement Layer 1 — Lexer token

In `spec/lexer/token.go`, add the token constant in the reserved function names block:

```go
FUNC_AVG    // "avg" - canonical name
FUNC_SQRT   // "sqrt" - canonical name
FUNC_SUM    // "sum" - canonical name
FUNC_<NAME> // "<name>" - canonical name   ← ADD HERE
```

### Step 4: Implement Layer 2 — Lexer keyword

In `spec/lexer/lexer.go`, add to `ReservedKeywords`:

```go
"avg":    FUNC_AVG,
"sqrt":   FUNC_SQRT,
"sum":    FUNC_SUM,
"<name>": FUNC_<NAME>,   // ← ADD HERE
```

### Step 5: Implement Layer 3 — Parser dispatch + validation

In `spec/parser/rdparser.go`, find `parsePrimary()` and add the token to the canonical function match:

```go
if p.match(lexer.FUNC_AVG, lexer.FUNC_SQRT, lexer.FUNC_SUM, lexer.FUNC_<NAME>) {
    return p.parseFunctionCall()
}
```

Then add argument validation in `parseFunctionCall()` in **TWO places**:

1. **Empty arg list path** (early return when `p.check(lexer.RPAREN)`):
   ```go
   if funcNameStr == "<name>" {
       return nil, p.error("<name>() requires exactly N argument(s)")
   }
   ```

2. **After parsing args** (before building the AST node):
   ```go
   if funcNameStr == "<name>" {
       if len(args) == 0 {
           return nil, p.error("<name>() requires exactly N argument(s)")
       }
       if len(args) > N {
           return nil, p.error("<name>() requires exactly N argument(s)")
       }
   }
   ```

**Common gotcha:** Forgetting the empty-args path causes `<name>()` with zero args to silently succeed.

### Step 6: Implement Layer 4 — Classifier

In `spec/classifier/classifier.go`, add to the `functionTypes` map in `containsFunctions()`:

```go
lexer.FUNC_<NAME>: true,
```

### Step 7: Implement Layer 5 — Detector

In `spec/document/detector.go`, add to the `isFunctionToken()` switch:

```go
case lexer.FUNC_AVG, lexer.FUNC_SQRT, lexer.FUNC_SUM, lexer.FUNC_<NAME>,
```

### Step 8: Implement Layer 6 — Semantic spec

In `spec/semantic/function_types.go`, add to `FunctionSpecs`:

```go
"<name>": {
    Name: "<name>",
    Params: []ParamSpec{
        {Name: "value", Type: ArgType<Type>, Examples: []string{...}},
    },
},
```

### Step 9: Implement Layer 7 — Interpreter

In `impl/interpreter/functions.go`:

1. Add `FunctionDef` to `BuiltinFunctions`:
   ```go
   {
       Name:        "<name>",
       Synonyms:    []string{},
       Description: "...",
       Signature:   "<name>(arg1, arg2)",
       Category:    "Math",  // or appropriate category
   },
   ```

2. Add to `functionEvalMap`:
   ```go
   "<name>": eval<Name>Func,
   ```

3. Implement the eval function:
   ```go
   func eval<Name>Func(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
       args, err := interp.evalAllArgs(f)
       if err != nil {
           return nil, err
       }
       return eval<Name>(args)
   }

   func eval<Name>(args []types.Type) (types.Type, error) {
       // Implementation
   }
   ```

**Pattern note:** Functions with identifier arguments (like `read`, `seek`, `rtt`) do NOT use `evalAllArgs()` — they extract AST identifiers directly from `f.Arguments`.

### Step 10: Implement Layer 8 — Feature registry

In `spec/features/registry.go`, add to `getFunctions()`:

```go
{
    Name:        "<name>",
    Category:    CategoryFunction,
    Syntax:      "<name>(arg)",
    Description: "...",
    Aliases:     nil,  // or []Alias{...} for NL forms
    Example:     "<name>(...) -> result",
},
```

### Step 11: Update registry count

In `impl/interpreter/registry_test.go`, increment `expectedCount` in `TestRegistryFunctionCount`:

```go
expectedCount := <previous_count + 1>
```

### Step 12: Implement type coercion (if applicable)

In `impl/interpreter/operators.go`, add new cases to `evalBinaryOperation()`. Place them in the correct position relative to existing type dispatch — order matters because the first matching case wins.

**Pattern for cross-type multiplication:**

```go
// <TypeA> * <TypeB> → <ResultType>
if leftA, ok := left.(*types.<TypeA>); ok {
    if rightB, ok := right.(*types.<TypeB>); ok && operator == "*" {
        return types.New<ResultType>(leftA.Value.Mul(rightB.Value), ...), nil
    }
}
```

Always implement both orderings if the operation is commutative:
- `TypeA * TypeB → Result`
- `TypeB * TypeA → Result`

### Step 13: Run full test suite

```bash
task test
```

All tests must pass. Common failures:
- `TestRegistryFunctionCount` — forgot to increment count
- `TestEveryBuiltinFunctionHasFunctionSpec` — forgot semantic spec
- `TestEveryFunctionSpecHasBuiltinFunction` — spec name doesn't match builtin name
- `TestEveryBuiltinFunctionHasEval` — forgot `functionEvalMap` entry
- `TestSuggestionTagCoversAllCategories` — new category not in known set

### Step 14: Quality and smoke test

```bash
task quality
task build && echo "<name>(<test_input>)" | ./cm --format json
```

### Step 15: Validate doceval and site build

If the feature is referenced in any site documentation:

```bash
go run ./cmd/doceval
task site:build
```

## NL Function Variant (when adding NL aliases)

If the function also needs an NL form (e.g., `"<name> of"` as an alias for `<name>()`):

1. Add multi-token type in `spec/lexer/token.go`:
   ```go
   FUNC_<NAME>_OF // "<name> of" → maps to "<name>"
   ```

2. Add multi-token scanning in `spec/lexer/lexer.go` (in the multi-token keyword detection section).

3. Add to `parsePrimary()` NL function match in `spec/parser/rdparser.go`:
   ```go
   if p.match(..., lexer.FUNC_<NAME>_OF) {
       return p.parseNaturalLanguageFunction()
   }
   ```

4. Add case to `parseNaturalLanguageFunction()` in `spec/parser/nl_functions.go`:
   ```go
   case lexer.FUNC_<NAME>_OF:
       funcName = "<name>"
   ```

5. Add to classifier and detector (both NL and canonical tokens).

6. Add `Alias` entries in `spec/features/registry.go`:
   ```go
   Aliases: []Alias{{Name: "<name> of", Parseable: true, Example: "<name> of 1, 2, 3"}},
   ```

## Checklist

Use this checklist to verify completeness before running `task test`:

- [ ] Lexer token constant added (`FUNC_<NAME>`)
- [ ] Reserved keyword mapping added
- [ ] Parser dispatch added to `parsePrimary()`
- [ ] Arg validation in BOTH paths of `parseFunctionCall()`
- [ ] Classifier `containsFunctions()` updated
- [ ] Detector `isFunctionToken()` updated
- [ ] Semantic `FunctionSpec` added
- [ ] `BuiltinFunctions` entry added with all required fields
- [ ] `functionEvalMap` entry added
- [ ] Eval function implemented
- [ ] Feature registry entry added
- [ ] `TestRegistryFunctionCount` incremented
- [ ] Golden test file created in `testdata/eval/success/features/`
- [ ] Parser tests written and passing
- [ ] Interpreter tests written and passing
- [ ] `task test` passes
- [ ] `task quality` passes
- [ ] Smoke test with `./cm --format json` works
- [ ] (If NL) Multi-token lexer, NL parser, aliases all added
- [ ] (If type coercion) Both operand orderings implemented and tested
- [ ] (If site docs changed) `go run ./cmd/doceval` and `task site:build` pass
