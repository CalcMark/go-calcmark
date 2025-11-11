# CalcMark Demo - SvelteKit 5 Application

A SvelteKit 5 application demonstrating CalcMark syntax highlighting, evaluation, and diagnostics with server-side WASM processing.

## Features

- **Server-side WASM**: CalcMark WASM runs on the Node.js server (no 3MB client download)
- **Real-time processing**: Syntax highlighting, evaluation, and diagnostics as you type
- **Svelte 5 runes**: Modern reactive patterns with `$state`, `$derived`, `$effect`
- **Block-based markdown rendering**: Proper rendering of markdown structures (lists, blockquotes, etc.)
- **Token-based highlighting**: Color-coded tokens with contextual styling
- **Live diagnostics**: Errors, warnings, and hints displayed inline
- **Evaluation results**: See calculated values for each line
- **Interactive tooltips**: Hover over calculation lines to see results and diagnostics

## Prerequisites

- **Node.js 18+** (for running the SvelteKit dev server)
- **Go 1.21+** (for building the WASM module)

## Setup

### 1. Build the WASM module

**IMPORTANT**: The demo requires WASM files to be built first. These files are **NOT** in git and must be generated locally.

```bash
# From the project root, build the calcmark CLI tool first
cd /Users/bitsbyme/projects/go-calcmark
go build -o calcmark ./impl/cmd/calcmark

# Then use it to build WASM files directly into demo/static/
./calcmark wasm demo/static/
```

This creates two files in `demo/static/` (both are git-ignored):
- `calcmark.wasm` - The compiled WASM binary (~3MB)
- `wasm_exec.js` - Go's WASM runtime loader

**Why these aren't in git**: The WASM file is large (~3MB) and should be built locally for your Go version.

### 2. Install dependencies

```bash
cd demo
npm install
```

### 3. Run the development server

```bash
npm run dev
```

Then open http://localhost:5173 in your browser.

## Project Structure

```
demo/
├── static/
│   ├── calcmark.wasm          # CalcMark WASM binary (generated, not in git)
│   └── wasm_exec.js           # Go WASM runtime (generated, not in git)
├── src/
│   ├── app.html               # SvelteKit HTML template
│   ├── lib/
│   │   ├── components/
│   │   │   ├── CalcMarkEditor.svelte     # Main editor component
│   │   │   ├── SyntaxHighlighter.svelte  # Syntax highlighting logic
│   │   │   ├── CalcMarkBlock.svelte      # Block grouping and rendering
│   │   │   ├── CalculationLine.svelte    # Individual calculation line
│   │   │   ├── CalcToken.svelte          # Token rendering
│   │   │   ├── ValueTooltip.svelte       # Hover tooltips
│   │   │   ├── EvaluationResults.svelte  # Results panel
│   │   │   ├── LineClassifications.svelte # Classification panel
│   │   │   └── TokenizationDetails.svelte # Tokens panel
│   │   └── server/
│   │       └── calcmark.ts    # Server-side WASM loader
│   └── routes/
│       ├── +layout.svelte     # Root layout
│       ├── +page.svelte       # Home page
│       └── api/
│           └── process/
│               └── +server.ts # CalcMark processing API endpoint
├── package.json
├── svelte.config.js           # SvelteKit configuration
└── vite.config.js             # Vite configuration
```

## How It Works

### Server-Side Architecture

The demo uses **server-side WASM processing** to avoid sending 3MB of WASM to every client:

1. **WASM Initialization** (`src/lib/server/calcmark.ts`)
   - Loads WASM once when server starts
   - Initializes `global.calcmark` API
   - Single-pass evaluation for O(n) performance

2. **API Endpoint** (`src/routes/api/process/+server.ts`)
   - POST `/api/process` with `{ input: string }`
   - Returns: classifications, tokens, evaluationResults, diagnostics

3. **Client Components** (Svelte 5 with runes)
   - Editor calls API on input change (debounced 150ms)
   - Receives processed data and updates UI reactively
   - Uses `$state`, `$derived`, `$effect` for reactivity

### Data Flow

```
User types in editor
    ↓
CalcMarkEditor.svelte (debounced)
    ↓
POST /api/process
    ↓
Server: processCalcMark() in Node.js
    ├── classifyLines()
    ├── tokenize()
    ├── evaluate() (single pass)
    └── validate()
    ↓
Response: { classifications, tokens, evaluationResults, diagnostics }
    ↓
SyntaxHighlighter.svelte ($derived.by)
    ↓
CalcMarkBlock.svelte (groups lines into blocks)
    ├── Markdown blocks → {@html marked.parse(...)}
    └── Calculation blocks → CalculationLine components
```

### Svelte 5 Migration Patterns

This project demonstrates proper Svelte 5 migration:

```javascript
// OLD (Svelte 4)               →  NEW (Svelte 5)
export let prop                 →  let { prop = defaultValue } = $props()
let variable                    →  let variable = $state(value)
$: derived = expression         →  const derived = $derived(expression)
$: { ...complex logic }         →  const derived = $derived.by(() => {...})
onMount(() => {...})            →  $effect(() => {...})
on:event                        →  onevent
```

## API Reference

The server-side CalcMark API (`src/lib/server/calcmark.ts`) exposes:

```typescript
interface CalcMarkAPI {
  tokenize(source: string): { tokens: string; error: string | null };
  evaluate(source: string, useGlobalContext: boolean): { results: string; error: string | null };
  validate(source: string): { diagnostics: string; error: string | null };
  classifyLines(lines: string[]): { classifications: string; error: string | null };
  resetContext(): void;
  getVersion(): string;
}
```

## Building for Production

```bash
npm run build
```

This creates optimized files in `build/`:
- Minified JavaScript
- Server-side Node.js application
- Static assets in `build/client/`

Deploy using:
```bash
node build
```

Or use the Node adapter for your hosting platform (Vercel, Netlify, etc.).

## Performance Notes

- **No client WASM download**: Server processes everything
- **Single-pass evaluation**: O(n) instead of O(n²)
- **Debounced input**: 150ms delay prevents excessive API calls
- **Block-based rendering**: Efficient markdown parsing
- **Caching**: Server maintains WASM instance across requests

## Browser Support

Works in all modern browsers:
- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## Troubleshooting

### WASM files not found

**Error**: `ENOENT: no such file or directory, open 'static/calcmark.wasm'`

**Solution**:
```bash
# From project root
go build -o calcmark ./impl/cmd/calcmark
./calcmark wasm demo/static/
```

This builds and copies both `calcmark.wasm` and `wasm_exec.js` to the correct location.

### Server initialization fails

**Error**: `CalcMark API not initialized`

**Solution**:
- Ensure Go WASM was built with correct Go version: `go version`
- The `calcmark wasm` command automatically uses the correct `wasm_exec.js` for your Go version
- Rebuild if needed: `./calcmark wasm demo/static/`

### API endpoint returns 500

Check server logs for errors. Common issues:
- WASM files missing or corrupted
- Go runtime version mismatch
- Invalid input format

### Syntax highlighting not working

1. Open browser console and check for errors
2. Verify API endpoint returns data: `curl -X POST http://localhost:5173/api/process -H "Content-Type: application/json" -d '{"input": "x = 5"}'`
3. Check that `tokens` array is populated in response

## Development Tips

### Adding console logging

Server-side (visible in terminal):
```typescript
// src/lib/server/calcmark.ts
console.log('Processing input:', input);
```

Client-side (visible in browser console):
```svelte
<!-- src/lib/components/CalcMarkEditor.svelte -->
<script>
  $effect(() => {
    console.log('Tokens received:', tokens);
  });
</script>
```

### Testing the API directly

```bash
# Simple calculation
curl -X POST http://localhost:5173/api/process \
  -H "Content-Type: application/json" \
  -d '{"input": "x = 5 + 10"}'

# With markdown
curl -X POST http://localhost:5173/api/process \
  -H "Content-Type: application/json" \
  -d '{"input": "# Test\n\nx = 5\ny = x + 10"}'
```

## Next Steps

Potential enhancements:
1. **Autocomplete**: Use tokenization to suggest variables
2. **Code folding**: Collapse markdown sections
3. **Multi-document**: Support multiple CalcMark files
4. **Export**: Generate PDF/HTML reports from CalcMark documents
5. **Collaborative editing**: Real-time multi-user editing

## Syntax Highlighting Reference

To ensure syntax highlighting stays in sync with the language grammar, use the formal EBNF specification:

```bash
# From project root - outputs EBNF grammar with all token types, reserved keywords, and operators
go run spec/cmd/cmspec/main.go
```

This is the source of truth for:
- Token types that need CSS styling in `CalcToken.svelte`
- Reserved keywords
- Operator precedence
- Terminal patterns (numbers, identifiers, etc.)

The grammar is auto-generated from the parser implementation, so it's always accurate.

## Related Documentation

- [CalcMark WASM API](../impl/wasm/README.md)
- [CalcMark Syntax Specification](../spec/SYNTAX_SPEC.md)
- [SvelteKit Documentation](https://kit.svelte.dev/)
- [Svelte 5 Migration Guide](https://svelte-5-preview.vercel.app/docs/breaking-changes)
