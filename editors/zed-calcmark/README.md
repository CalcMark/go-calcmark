# CalcMark for Zed

Language support for [CalcMark](https://calcmark.org) — calculations and markdown in one document.

## Features

- Diagnostics (errors and warnings with precise positions)
- Autocomplete (functions, units, variables, NL syntax)
- Hover info (variable values, function signatures, unit descriptions)
- Go-to-definition (jump to variable assignment)
- Document symbols (outline view)
- Code actions (quick fixes for undefined variables)
- Signature help (parameter hints inside function calls)
- Semantic highlighting (context-aware calc vs markdown)

## Setup

1. Install the `cm` binary ([installation guide](https://calcmark.org/docs/getting-started/))
2. Install this extension in Zed
3. Add to your Zed settings (**Cmd+,**):

```json
{
  "languages": {
    "CalcMark": {
      "semantic_tokens": "full"
    }
  }
}
```

This enables context-aware syntax highlighting. Without it, CalcMark files appear as plain text.
