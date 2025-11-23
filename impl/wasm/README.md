# CalcMark WASM

WebAssembly bindings for the CalcMark library, enabling CalcMark parsing, evaluation, and syntax highlighting in web browsers.

## Building

```bash
./build.sh
```

This will:
1. Compile the Go code to WebAssembly (`calcmark.wasm`)
2. Download the required `wasm_exec.js` helper from the Go repository

## Exported Functions

The WASM module exposes the following functions on `window.calcmark`:

### `tokenize(sourceCode: string)`
Tokenizes CalcMark source code.

**Returns:** `{tokens: string, error: string|null}`
- `tokens`: JSON-encoded array of token objects with `type`, `value`, `start`, `end`, `line`
- `error`: Error message if tokenization failed, otherwise `null`

**Example:**
```javascript
const result = window.calcmark.tokenize("x = 5 + 3");
if (!result.error) {
  const tokens = JSON.parse(result.tokens);
  console.log(tokens);
}
```

### `parse(sourceCode: string)`
Parses CalcMark source code into an Abstract Syntax Tree (AST).

**Returns:** `{ast: string, error: string|null}`
- `ast`: JSON-encoded AST node array
- `error`: Error message if parsing failed, otherwise `null`

### `evaluate(sourceCode: string, useGlobalContext: boolean)`
Evaluates CalcMark source code and returns results.

**Returns:** `{results: string, error: string|null}`
- `results`: JSON-encoded array of evaluation results
- `error`: Error message if evaluation failed, otherwise `null`
- `useGlobalContext`: If `true`, maintains variables across calls. If `false`, uses a fresh context.

**Example:**
```javascript
const result = window.calcmark.evaluate("x = 5\ny = x + 3", true);
if (!result.error) {
  const results = JSON.parse(result.results);
  console.log(results); // [5, 8]
}
```

### `validate(sourceCode: string)`
Validates CalcMark source code and returns diagnostics.

**Returns:** `{diagnostics: string, error: string|null}`
- `diagnostics`: JSON-encoded validation result with diagnostic codes
- `error`: Error message if validation system failed, otherwise `null`

### `classifyLine(line: string)`
Classifies a single line as CALCULATION, MARKDOWN, or BLANK.

**Returns:** `{lineType: string, error: string|null}`
- `lineType`: One of "CALCULATION", "MARKDOWN", or "BLANK"
- `error`: Error message if classification failed, otherwise `null`

### `classifyLines(lines: string[])`
Classifies multiple lines with context awareness.

**Returns:** `{classifications: string, error: string|null}`
- `classifications`: JSON-encoded array of classification results
- `error`: Error message if classification failed, otherwise `null`

**Example:**
```javascript
const lines = ["x = 5", "y = x + 3", "# Header"];
const result = window.calcmark.classifyLines(lines);
if (!result.error) {
  const classifications = JSON.parse(result.classifications);
  // [
  //   {lineType: "CALCULATION", line: "x = 5", index: 0},
  //   {lineType: "CALCULATION", line: "y = x + 3", index: 1},
  //   {lineType: "MARKDOWN", line: "# Header", index: 2}
  // ]
}
```

### `resetContext()`
Resets the global evaluation context, clearing all variables.

**Returns:** `void`

### `getVersion()`
Returns the CalcMark library version.

**Returns:** `string` (e.g., "0.1.1")

## Integration

### Basic HTML

```html
<!DOCTYPE html>
<html>
<head>
  <script src="wasm_exec.js"></script>
  <script>
    const go = new Go();
    WebAssembly.instantiateStreaming(fetch("calcmark.wasm"), go.importObject)
      .then(result => {
        go.run(result.instance);

        // Now you can use window.calcmark
        const tokenResult = window.calcmark.tokenize("salary = $50000");
        console.log(JSON.parse(tokenResult.tokens));
      });
  </script>
</head>
<body>
  <h1>CalcMark WASM Example</h1>
</body>
</html>
```

### With Module Bundler (Vite, Webpack, etc.)

The WASM module can be integrated into any module bundler setup. Import the `wasm_exec.js` and load the `.wasm` file according to your bundler's asset handling configuration.

## File Sizes

- `calcmark.wasm`: ~2-3 MB (compressed: ~500-700 KB with gzip)
- `wasm_exec.js`: ~17 KB

## Browser Compatibility

Requires browsers with WebAssembly support (all modern browsers):
- Chrome 57+
- Firefox 52+
- Safari 11+
- Edge 16+

## Development

To rebuild after changing Go code:

```bash
cd /path/to/go-calcmark
# Make changes to the library
cd wasm
./build.sh
```

## TypeScript Definitions (Optional)

For TypeScript projects, consider creating a `.d.ts` file:

```typescript
// calcmark.d.ts
export interface TokenInfo {
  type: string;
  value: string;
  start: number;
  end: number;
  line: number;
}

export interface CalcMarkAPI {
  tokenize(source: string): { tokens: string; error: string | null };
  parse(source: string): { ast: string; error: string | null };
  evaluate(source: string, useGlobalContext: boolean): { results: string; error: string | null };
  validate(source: string): { diagnostics: string; error: string | null };
  classifyLine(line: string): { lineType: string; error: string | null };
  classifyLines(lines: string[]): { classifications: string; error: string | null };
  resetContext(): void;
  getVersion(): string;
}

declare global {
  interface Window {
    calcmark: CalcMarkAPI;
  }
}
```
