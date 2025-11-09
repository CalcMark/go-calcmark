# CalcMark Language Specification

This directory contains the **platform-independent** CalcMark language specification.

## Files

### `LANGUAGE_SPEC.md`
**The authoritative CalcMark language specification.**

Defines:
- Syntax and grammar
- Semantics (what expressions mean)
- Type system rules
- Operator precedence
- Reserved keywords
- Examples

Anyone implementing CalcMark in any language should follow this spec.

### `SYNTAX_HIGHLIGHTER_SPEC.json`
**Machine-readable syntax specification for editor integrations.**

This JSON file is:
- **Generated from** the Go implementation (validated by tests)
- **Used by** TypeScript/JavaScript syntax highlighters
- **Served at** `/syntax` HTTP endpoint (optional)
- **Validated** automatically on every test run

Contains:
- Reserved keywords
- Token patterns
- Operators
- Multi-token function patterns
- Breaking changes

### `SYNTAX_HIGHLIGHTER_README.md`
**Integration guide** for using `SYNTAX_HIGHLIGHTER_SPEC.json` in TypeScript/JavaScript clients.

Shows how to:
- Import the JSON spec
- Extract keywords for highlighting
- Build a simple tokenizer
- Handle case-insensitive keywords

## Usage

### For Language Implementers
Read `LANGUAGE_SPEC.md` to understand CalcMark semantics and implement in your language of choice.

### For Syntax Highlighter Authors
Use `SYNTAX_HIGHLIGHTER_SPEC.json` + `SYNTAX_HIGHLIGHTER_README.md` to build editor integrations.

### For API Servers
Serve `SYNTAX_HIGHLIGHTER_SPEC.json` at `/syntax` endpoint for dynamic client updates:

```go
http.HandleFunc("/syntax", func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "spec/SYNTAX_HIGHLIGHTER_SPEC.json")
})
```

## Relationship to Go Implementation

The Go implementation (in `..` parent directory) is **one implementation** of this spec.

```
spec/                    ← Language definition (platform-independent)
  LANGUAGE_SPEC.md      ← What CalcMark means

../                      ← Go implementation (platform-specific)
  lexer/                ← How we tokenize in Go
  parser/               ← How we parse in Go
  evaluator/            ← How we evaluate in Go
```

The spec files are validated against the Go implementation via tests in `../lexer/syntax_spec_test.go`.

## Version

Current version: **1.0.0** (draft, implementation in progress)
