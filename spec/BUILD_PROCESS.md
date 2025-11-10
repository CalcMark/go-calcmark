# Syntax Spec Build Process

This document explains how the `SYNTAX_HIGHLIGHTER_SPEC.json` is generated, embedded, and distributed.

## Overview

The syntax highlighter specification follows this workflow:

```
Go Code (lexer/parser) → Generator → JSON Files → Embedded in Library → Distributed
                                         ↓
                                   spec/ (docs)
                                   syntax/ (embedded)
```

## Components

### 1. Generator (`cmd/calcmark generate`)

The `calcmark` CLI includes a `generate` subcommand that creates the JSON spec from the Go implementation.

**Run manually:**
```bash
calcmark generate
```

**Run via go generate:**
```bash
go generate ./syntax
```

This generates TWO copies of the file:
- `spec/SYNTAX_HIGHLIGHTER_SPEC.json` - For documentation
- `syntax/SYNTAX_HIGHLIGHTER_SPEC.json` - For embedding

### 2. Embedded Spec (`syntax/embed.go`)

Uses Go 1.16+ `//go:embed` directive to embed the JSON file into the compiled library.

**Key features:**
- `SyntaxHighlighterSpec` - String containing the JSON
- `SyntaxHighlighterSpecBytes()` - Function returning `[]byte`
- `//go:generate` directive to regenerate before embedding

### 3. CLI Tool (`cmd/calcmark`)

A command-line tool with subcommands for working with the spec.

**Install:**
```bash
go install github.com/CalcMark/go-calcmark/cmd/calcmark@latest
```

**Usage:**
```bash
calcmark spec                  # Print to stdout
calcmark spec > syntax.json    # Save to file
calcmark generate              # Regenerate the spec files
```

## Build Workflow

### For Library Development

When making changes to the language syntax:

1. **Update the Go code** (lexer, parser, etc.)
2. **Regenerate the spec:**
   ```bash
   calcmark generate
   # Or: go generate ./syntax
   ```
3. **Run tests to verify sync:**
   ```bash
   go test ./lexer -run TestSyntaxSpec
   ```
4. **The spec is automatically embedded on next build**

### For Library Consumers

When using go-calcmark in your project:

```go
import "github.com/CalcMark/go-calcmark/syntax"

// The spec is already embedded - just access it:
jsonSpec := syntax.SyntaxHighlighterSpec
```

**Example: CalcMark Server serving the spec over HTTP:**

```go
package main

import (
    "net/http"
    "github.com/CalcMark/go-calcmark/syntax"
)

func main() {
    http.HandleFunc("/syntax", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write(syntax.SyntaxHighlighterSpecBytes())
    })

    http.ListenAndServe(":8080", nil)
}
```

No need to:
- Copy JSON files
- Manage file paths
- Worry about file distribution
- Keep files in sync

The spec is **always** embedded in the library.

## Validation

The JSON spec is validated by tests in `lexer/syntax_spec_test.go`:

- ✅ File exists
- ✅ Valid JSON
- ✅ Contains all reserved keywords from Go code
- ✅ Contains multi-token functions
- ✅ Contains boolean values
- ✅ Contains operators
- ✅ Documents breaking changes

These tests run on **every** test run, ensuring the spec never drifts from the implementation.

## Distribution

The spec is distributed in three ways:

1. **Embedded in library** - Import `github.com/CalcMark/go-calcmark/syntax`
2. **Via CLI tool** - Run `calcmark spec`
3. **As JSON file** - Check out the repository and find `spec/SYNTAX_HIGHLIGHTER_SPEC.json`

## Version Synchronization

The spec version matches the library version:

- `spec/SYNTAX_HIGHLIGHTER_SPEC.json` - `version` field
- `cmd/calcmark/main.go` - `version` constant
- `cmd/generate-syntax-spec/main.go` - `version` constant

When releasing a new version, update all three locations.

## Testing

Tests verify the complete workflow:

```bash
# Test generation
calcmark generate
go test ./lexer -run TestSyntaxSpec

# Test embedding
go test ./syntax

# Test CLI
go build -o /tmp/calcmark ./cmd/calcmark
/tmp/calcmark spec | jq .version

# Test all
go test ./...
```

All tests pass: ✅ 178 tests (173 original + 3 syntax tests + 2 from other packages)
