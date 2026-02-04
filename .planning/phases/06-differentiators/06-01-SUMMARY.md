---
phase: 06-differentiators
plan: 01
subsystem: interpreter
tags: [functions, metadata, tui, yaml, error-handling]

dependency_graph:
  requires: [05-02]
  provides: [function-metadata-refactor, function-result-display-fix, yaml-error-messages]
  affects: [06-02]

tech_stack:
  added: []
  patterns: [single-source-of-truth, init-function-pattern]

key_files:
  created:
    - cmd/calcmark/tui/editor/function_result_display_test.go
  modified:
    - impl/interpreter/functions.go
    - impl/interpreter/registry.go
    - impl/interpreter/registry_test.go
    - spec/document/detector.go
    - spec/document/frontmatter.go
    - spec/document/frontmatter_test.go
    - testdata/eval/success/features/functions.cm

decisions:
  - id: struct-based-registration
    description: "FunctionDef struct with metadata + Eval field using init() to avoid cycle"
    rationale: "Go init cycles prevented direct function references; init() pattern cleanly resolves"
  - id: function-token-detection
    description: "Added isFunctionToken to detector's looksLikeCalculation"
    rationale: "Built-in function calls were not being recognized as calculations"

metrics:
  duration: 12min
  completed: 2026-02-04
---

# Phase 06 Plan 01: Function Metadata Refactor and Bug Fixes Summary

**One-liner:** Unified function metadata to single source of truth, fixed function result display bug, and added YAML error line numbers.

## Changes Made

### Task 1: Refactor function metadata to single source of truth
- Created `FunctionDef` struct in `functions.go` with metadata fields (Name, Synonyms, Description, Signature, Category) plus `Eval` function pointer
- Created `BuiltinFunctions` slice as the authoritative list of all 12 CalcMark functions
- Used `init()` function to populate `Eval` fields, avoiding Go initialization cycle
- Refactored `evalFunctionCall` from switch statement to loop over `BuiltinFunctions`
- Updated `registry.go` to derive `FunctionInfo` from `BuiltinFunctions`
- Added "mean" as synonym for avg (required for SC4 autocomplete)
- Removed AST-parsing registry sync test (no longer needed - struct IS the test)
- Added `GetFunctionByName` helper function

### Task 2: Fix function result display bug in TUI preview pane
- Root cause: `looksLikeCalculation` in detector.go did not recognize lines starting with function tokens (FUNC_AVG, FUNC_SQRT, etc.)
- Added `isFunctionToken` helper to detect built-in function tokens
- Updated `looksLikeCalculation` to return true for function calls
- Fixed `functions.cm` testdata to define variables x, y, z, a, b used in examples
- Created comprehensive tests for function result display

### Task 3: Improve YAML frontmatter error messages with line numbers
- Added `formatYAMLError` helper to extract line info from yaml.v3 errors
- Handles both `yaml.TypeError` (structured errors) and syntax errors (string message)
- Error format: "frontmatter YAML error: line N: [message]"
- Added tests verifying line numbers appear in error messages

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| Use init() to populate Eval fields | Go compiler detects initialization cycles when function values reference the slice that contains them; init() runs after all declarations |
| Added isFunctionToken to detector | Built-in function tokens (FUNC_AVG, FUNC_SQRT) weren't recognized as starting valid calculations |
| Define variables in functions.cm | Test file used undefined variables that were previously treated as markdown but are now correctly identified as calculations |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Variable definitions in functions.cm**
- **Found during:** Task 2 verification
- **Issue:** After fixing detector, `avg(x, y, z)` lines were correctly identified as calculations but failed evaluation due to undefined variables
- **Fix:** Added variable definitions (x=10, y=20, z=30, a=5, b=4) to testdata file
- **Files modified:** testdata/eval/success/features/functions.cm
- **Commit:** 51114cd

## Test Results

- All interpreter tests pass (12 functions with metadata verification)
- Function result display tests pass (avg, sqrt, bare expressions)
- YAML error message tests pass (line numbers present)
- Full test suite passes

## Verification

```bash
# Function metadata unification
./cm help functions | grep -E "avg|mean|average"
# Output: avg (average, mean)

# Function evaluation with synonyms
echo "x = avg(2,4,4)" | cm eval
# Output: 3.333333

echo "x = mean(2,4,4)" | cm eval
# Output: 3.333333

# YAML error with line number
echo '---
globals:
  rate: [invalid
---' > test.cm && cm eval test.cm
# Output: Error: ... frontmatter YAML error: line 1: did not find expected ',' or ']'
```

## Next Phase Readiness

**Blockers:** None

**Ready for 06-02:** Yes - function metadata is now unified and accessible for autocomplete implementation via `BuiltinFunctions` and `GetFunctionByName`.
