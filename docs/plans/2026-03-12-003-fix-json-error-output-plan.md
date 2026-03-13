---
title: "fix: JSON error output on stdout for agent/pipeline consumption"
type: fix
status: active
date: 2026-03-12
---

# fix: JSON error output on stdout for agent/pipeline consumption

## Overview

When `cm` encounters an error with `--format json`, the error goes to stderr as plain text while stdout gets nothing. Agents and pipelines piping `cm` output to a JSON parser get empty input and a confusing parse error instead of the actual CalcMark error.

Fix: when `--format json` is active, output a JSON error envelope on stdout so parsers always receive valid JSON. Exit code still signals the error for shell scripts.

Closes #53.

## Problem Statement

```bash
echo "x = 10\nx = 20" | cm --format json
# stdout: (empty)
# stderr: Error: evaluation error: line 2: variable_redefinition: cannot reassign 'x'
# exit code: 1
```

An agent piping this to `jq` gets a parse error instead of the actual CalcMark error.

## Proposed Solution

In `cmd/calcmark/cmd/eval.go`, catch errors from the evaluation pipeline. When `--format json` is active, serialize the error as a JSON object to stdout before returning the error (which still goes to stderr via Cobra and sets exit code 1).

### JSON Error Envelope

```json
{
  "error": {
    "type": "evaluation_error",
    "message": "cannot reassign 'x'",
    "line": 2,
    "code": "variable_redefinition"
  }
}
```

Error types map to the existing error wrapping prefixes:
- `"parse_error"` — from `fmt.Errorf("parse error: %w", ...)`
- `"evaluation_error"` — from `fmt.Errorf("evaluation error: %w", ...)`
- `"frontmatter_error"` — from `fmt.Errorf("frontmatter error: %w", ...)`

### Architecture

The change is localized to the command layer — the error handling in `eval.go` and `root.go` (piped stdin path). No changes to the format package, eval pipeline, or public API.

**Error parsing approach**: The error messages from `eval.go` follow a structured pattern: `"<category> error: line N: <code>: <message>"`. Parse this with a regex or string splitting to extract type, line, code, and message. If parsing fails, fall back to the raw error string with type `"unknown_error"`.

The JSON formatter's existing `JSONDiagnostic` struct is close but not quite right — it's for per-block semantic diagnostics, not top-level fatal errors. A new lightweight `JSONError` struct at the command layer is cleaner.

## Technical Approach

### Phase 1: JSON Error Output

**Tasks:**

- [ ] Add `JSONErrorEnvelope` and `JSONError` structs in `cmd/calcmark/cmd/eval.go` (or a small helper file)
  ```go
  type JSONErrorEnvelope struct {
      Error JSONError `json:"error"`
  }
  type JSONError struct {
      Type    string `json:"type"`
      Message string `json:"message"`
      Line    int    `json:"line,omitempty"`
      Code    string `json:"code,omitempty"`
  }
  ```
- [ ] Add `parseEvalError(err error) JSONError` function that extracts type/line/code/message from the structured error string
- [ ] Add `writeJSONError(w io.Writer, err error)` function that serializes the error envelope
- [ ] In `runEval()`: when format is JSON and an error occurs, call `writeJSONError(os.Stdout, err)` before returning the error
- [ ] In root `RunE` (piped stdin path): same treatment — when format is JSON and `runEval` would be called, handle errors the same way
- [ ] Cobra still prints to stderr and sets exit code 1 — this is additive, not replacing
- [ ] Table-driven tests for `parseEvalError` covering all error patterns
- [ ] Integration test: pipe invalid input with `--format json`, verify stdout contains valid JSON error

**Files:**
- `cmd/calcmark/cmd/eval.go` (modify)
- `cmd/calcmark/cmd/root.go` (modify — piped stdin path calls runEval, same handling)
- `cmd/calcmark/cmd/eval_test.go` (new or modify)

### Phase 2: Patch Release

- [ ] Verify `task test` passes
- [ ] Verify `task quality` passes
- [ ] Tag `v1.8.1` (patch release — backwards-compatible fix)
- [ ] Push tag to trigger GoReleaser workflow

## Acceptance Criteria

- [ ] `echo "x = 10\nx = 20" | cm --format json` outputs valid JSON on stdout with error details
- [ ] Exit code is still 1 on error
- [ ] Stderr still shows the error (Cobra behavior unchanged)
- [ ] `echo "1 + 1" | cm --format json` still outputs normal JSON (no regression)
- [ ] JSON error envelope includes `type`, `message`, and where available `line` and `code`
- [ ] All existing tests pass
- [ ] `task quality` passes

## Sources & References

- Issue: https://github.com/CalcMark/go-calcmark/issues/53
- Error wrapping: `eval.go` lines using `fmt.Errorf("parse error: %w", ...)` etc.
- JSON format: `format/json_formatter.go`
- Command error handling: `cmd/calcmark/cmd/eval.go`, `cmd/calcmark/cmd/root.go`
