---
title: Diagnostic Detailed Field End-to-End Pipeline
category: code-organization
date: 2026-03-12
tags: [diagnostics, semantic-parser, tui, hints, architecture]
components: [spec/semantic/diagnostics.go, spec/document/document.go, impl/document/evaluator.go, components/errors.go]
---

# Diagnostic Detailed Field End-to-End Pipeline

## Problem

Hint definitions for diagnostic errors (e.g., "Defined variables: income, tax_rate") were scattered across the TUI layer (`components/errors.go`) and the semantic parser (`spec/semantic/`). The TUI was computing hints from error message strings using regex-like parsing, duplicating logic that the semantic checker already had.

The semantic checker has full context — it knows which variables are defined, which units are valid, etc. The TUI should consume this information, not re-derive it.

## Solution

### 1. Hints defined in the semantic layer

`spec/semantic/diagnostics.go` is the single source of truth for all diagnostic hints:

```go
func HintForDiagnostic(code, message string) string {
    switch code {
    case DiagUndefinedVariable:
        varName := extractQuoted(message)
        if varName != "" {
            return "Define it above: " + varName + " = <value>"
        }
    case DiagDivisionByZero:
        return "Check that divisor is not zero"
    case DiagFrontmatterValidation:
        if idx := strings.Index(message, "valid categories: "); idx >= 0 {
            return message[idx:]
        }
        return "Check frontmatter YAML syntax"
    // ...
    }
}
```

### 2. `Detailed` field flows through the pipeline

The semantic checker populates `Detailed` with rich context (e.g., "Defined variables: income, tax_rate, expenses"). This field flows through three `Diagnostic` types:

```
semantic.Diagnostic   →   document.Diagnostic   →   calcmark.Diagnostic
(spec layer)              (bridge layer)             (public API)
  .Detailed                 .Detailed                 .Detailed
```

Carry sites in the evaluator:
- `evaluateCalcBlockSelective` (~line 363): `Detailed: diag.Detailed,`
- `evaluateCalcBlockWithDoc` (~line 482): `Detailed: diag.Detailed,`
- `convertDiagnostics` in `eval.go`: `Detailed: d.Detailed,`

### 3. TUI prefers `Detailed`, falls back to semantic hints

```go
func GetHintForDiagnostic(diag *document.Diagnostic) string {
    if diag.Detailed != "" {
        return diag.Detailed  // Rich context from checker
    }
    return semantic.HintForDiagnostic(diag.Code, diag.Message)  // Fallback
}
```

### Architecture

```
Semantic Checker (has full context)
  → populates Detailed: "Defined variables: income, tax_rate"
  → populates Code: "undefined_variable"
  → Evaluator carries both through document.Diagnostic
  → TUI reads Detailed directly (no parsing)
  → Fallback: semantic.HintForDiagnostic(code, message)
```

## Prevention

- **New hint types** go in `spec/semantic/diagnostics.go`, never in TUI code.
- **Rich context** (variable lists, valid options) goes in the `Detailed` field, populated by the checker which has full evaluation context.
- **Simple hints** (static guidance strings) go in `HintForDiagnostic` switch cases.
- When adding a new `Diagnostic` conversion site, always include `Detailed: diag.Detailed` — grep for existing conversion sites to find them all.
