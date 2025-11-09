# MarkCalc Document Model Specification

## Overview

This specification defines the structural concepts of **Documents**, **Worksheets**, and **Lines** in MarkCalc. These concepts describe how MarkCalc content is organized, processed, and presented to users.

**Relationship to Language Spec:** The MarkCalc language defines valid statements (Assignment, Expression, etc.) and their semantics. This document model defines how those statements are organized into consumable units.

## Inspiration

Similar to how CommonMark defines a document as "a sequence of blocks," MarkCalc defines a worksheet as "an ordered sequence of lines." However, MarkCalc lines are simpler than CommonMark blocks:

- **Lines are newline-delimited** - Always exactly one line per unit
- **Lines never nest** - Flat structure, unlike CommonMark's container blocks
- **Lines process sequentially** - Top-to-bottom with accumulating context

## Core Concepts

### Document

A **Document** is the top-level container for MarkCalc content.

**Current Scope:** In version 1.0, a Document contains exactly one Worksheet. The terms "Document" and "Worksheet" are currently synonymous.

**Future Scope:** Documents may contain multiple worksheets, imports, or cross-worksheet references.

**Representation:**
```typescript
interface Document {
  id: string;
  title: string;
  content: string;           // Raw text content
  created_at: datetime;
  updated_at: datetime;
}
```

### Worksheet

A **Worksheet** is an ordered sequence of Lines with a shared execution context.

**Properties:**
- **Ordered** - Line order is significant
- **Sequential execution** - Lines process top-to-bottom
- **Shared context** - Variables defined in earlier lines are available to later lines
- **Newline-delimited** - Content splits on `\n` characters

**Representation:**
```typescript
interface Worksheet {
  lines: Line[];            // Ordered array of lines
  context: ExecutionContext; // Shared variable state
}
```

### Line

A **Line** is the fundamental unit of content in a worksheet.

**Definition:** A line is a sequence of characters terminated by a newline (`\n`) or end-of-file, excluding the newline character itself.

**Types:** Every line is classified as exactly one of:

1. **Calculation Line** - Contains a MarkCalc statement
2. **Markdown Line** - Contains prose, markdown syntax, or unrecognized content
3. **Blank Line** - Empty or contains only whitespace

**Classification:** Lines are classified using the rules defined in [LINE_CLASSIFICATION_SPEC.md](./LINE_CLASSIFICATION_SPEC.md).

**Representation:**
```typescript
interface Line {
  line_number: number;      // 0-indexed position in worksheet
  content: string;          // Raw text of the line
  type: LineType;           // 'calculation' | 'markdown' | 'blank'

  // For calculation lines:
  statement?: ASTNode;      // Parsed AST
  result?: ResultValue;     // Evaluation result
  variables?: Variable[];   // Variables defined on this line

  // For all lines:
  diagnostics?: Diagnostic[]; // Errors, warnings, info
}

type LineType = 'calculation' | 'markdown' | 'blank';

interface Variable {
  name: string;
  value: string;            // String representation
  type: 'number' | 'currency' | 'boolean';
}

interface ResultValue {
  value: string;            // Display value (e.g., "$1,000.00")
  type: 'number' | 'currency' | 'boolean';
  raw_value: string;        // Raw numeric value
}
```

## Execution Model

### Sequential Processing

Worksheets execute line-by-line from top to bottom:

```
Input Worksheet:
  Line 0: salary = $5000
  Line 1: # Deductions
  Line 2: tax = salary * 0.2
  Line 3: net = salary - tax

Processing:
  Line 0: CALCULATION → Evaluates → Context: {salary: $5000}
  Line 1: MARKDOWN    → Skipped   → Context: {salary: $5000}
  Line 2: CALCULATION → Evaluates → Context: {salary: $5000, tax: $1000}
  Line 3: CALCULATION → Evaluates → Context: {salary: $5000, tax: $1000, net: $4000}
```

### Context Accumulation

The **Execution Context** is a mapping of variable names to values:

```typescript
interface ExecutionContext {
  variables: Map<string, MarkCalcValue>;
}
```

**Rules:**
1. Context starts empty
2. Each calculation line may define new variables (via Assignment)
3. Variables persist for all subsequent lines
4. Variables can be reassigned (last assignment wins)
5. Undefined variable references produce semantic errors (see SEMANTIC_VALIDATION.md)

### Line Independence

While lines share context, they are otherwise independent:

- A line's classification doesn't depend on previous lines (except for variable context)
- Errors in one line don't affect processing of other lines
- Lines can be inserted, deleted, or reordered

## Line Processing Pipeline

Each line goes through this pipeline:

```
1. TOKENIZATION
   Input:  "salary = $5000"
   Output: [IDENTIFIER("salary"), ASSIGN, CURRENCY("$5000"), EOF]

2. CLASSIFICATION
   Input:  Tokens + Context
   Output: LineType.CALCULATION
   Rules:  See LINE_CLASSIFICATION_SPEC.md

3. PARSING (for calculations only)
   Input:  Tokens
   Output: Assignment(name="salary", value=CurrencyLiteral(5000, "$"))

4. SEMANTIC VALIDATION (optional)
   Input:  AST + Context
   Output: ValidationResult (errors, warnings)
   Rules:  See SEMANTIC_VALIDATION.md

5. EVALUATION (for calculations only)
   Input:  AST + Context
   Output: Result value + Updated context
```

## API Contract

### Request: Evaluate Document

```typescript
POST /api/v1/markcalc/evaluate

Request:
{
  "content": string  // Raw worksheet content
}

Response:
{
  "lines": Line[]    // Structured line data
}
```

### Response Structure

The API returns a structured representation preserving line information:

```json
{
  "lines": [
    {
      "line_number": 0,
      "content": "salary = $5000",
      "type": "calculation",
      "statement": { /* AST */ },
      "result": {
        "value": "$5,000.00",
        "type": "currency",
        "raw_value": "5000"
      },
      "variables": [
        {
          "name": "salary",
          "value": "$5,000.00",
          "type": "currency"
        }
      ],
      "diagnostics": []
    },
    {
      "line_number": 1,
      "content": "# Deductions",
      "type": "markdown",
      "diagnostics": []
    },
    {
      "line_number": 2,
      "content": "tax = salary * 0.2",
      "type": "calculation",
      "statement": { /* AST */ },
      "result": {
        "value": "$1,000.00",
        "type": "currency",
        "raw_value": "1000"
      },
      "variables": [
        {
          "name": "tax",
          "value": "$1,000.00",
          "type": "currency"
        }
      ],
      "diagnostics": []
    }
  ]
}
```

### Benefits of Line-Based API

1. **No frontend parsing** - Backend provides all structural information
2. **Exact variable locations** - Each line knows which variables it defines
3. **Per-line diagnostics** - Errors map directly to display positions
4. **Simpler frontend** - Consume structured data instead of regex parsing
5. **Enables features** - Line-level operations (insert, delete, move, fold)

## Frontend Consumption

### Display Model

The frontend can render each line independently:

```svelte
<!-- Worksheet.svelte -->
{#each lines as line, i (line.line_number)}
  <Line
    content={line.content}
    type={line.type}
    variables={line.variables}
    result={line.result}
    diagnostics={line.diagnostics}
    lineNumber={line.line_number}
  />
{/each}
```

### State Management

The document store manages line-based state:

```typescript
interface DocumentState {
  document: Document;
  lines: Line[];           // Structured line data from API
  isDirty: boolean;
  isSaving: boolean;
}
```

No need for:
- Regex parsing to find variable definitions
- Manual line splitting
- Complex state derivation

## Relationship to Other Specs

### LINE_CLASSIFICATION_SPEC.md

Defines the **rules** for determining line type:
- When is a line CALCULATION vs MARKDOWN?
- Context-aware classification
- Edge cases and examples

**This spec** defines the **data model** for representing classified lines.

### SEMANTIC_VALIDATION.md

Defines **semantic error detection**:
- Undefined variable references
- Type mismatches
- Rich diagnostic information

**This spec** defines where diagnostics are **stored** (in Line.diagnostics).

### Language Spec (Implicit)

Defines **valid statements**:
- Assignment syntax
- Expression syntax
- Type system

**This spec** defines how statements are **organized** into worksheets.

## Design Principles

### 1. Simplicity

Lines are simpler than CommonMark blocks:
- Always newline-delimited (no complex boundary rules)
- Never nest (flat structure)
- Sequential processing (no backtracking)

### 2. Explicitness

Every structural concept is explicit:
- Document (top-level container)
- Worksheet (ordered lines + context)
- Line (content + metadata + type)

### 3. Separation of Concerns

Clear boundaries:
- **Language spec** - What statements are valid
- **Document model** - How statements are organized
- **API contract** - How data is exchanged
- **Frontend** - How data is displayed

### 4. Future-Proof

Room for evolution:
- Multi-worksheet documents
- Cross-worksheet references
- Line-level permissions
- Collaborative editing

## Examples

### Example 1: Simple Budget

```
Input (raw text):
# Monthly Budget

income = $5000
expenses = $3000
savings = income - expenses

Output (structured):
lines: [
  { line_number: 0, type: 'markdown', content: '# Monthly Budget' },
  { line_number: 1, type: 'blank', content: '' },
  { line_number: 2, type: 'calculation', content: 'income = $5000',
    variables: [{name: 'income', value: '$5,000.00'}] },
  { line_number: 3, type: 'calculation', content: 'expenses = $3000',
    variables: [{name: 'expenses', value: '$3,000.00'}] },
  { line_number: 4, type: 'calculation', content: 'savings = income - expenses',
    variables: [{name: 'savings', value: '$2,000.00'}] }
]
```

### Example 2: Error Handling

```
Input:
x = 5
y = undefined_var * 2
z = x + 10

Output:
lines: [
  { line_number: 0, type: 'calculation', content: 'x = 5',
    variables: [{name: 'x', value: '5'}], diagnostics: [] },
  { line_number: 1, type: 'calculation', content: 'y = undefined_var * 2',
    variables: [], diagnostics: [
      {
        severity: 'error',
        message: 'Undefined variable: undefined_var',
        range: { start: { line: 1, column: 4 }, end: { line: 1, column: 17 } }
      }
    ]},
  { line_number: 2, type: 'calculation', content: 'z = x + 10',
    variables: [{name: 'z', value: '15'}], diagnostics: [] }
]
```

Note: Line 2 still evaluates successfully despite line 1's error.

## Migration Path

### Current State

- API returns flat `{results: [], variables: {}}`
- Frontend parses content with regex to find variable locations
- No formal line representation

### Migration Steps

1. **Backend:** Update evaluator to track line numbers from tokens
2. **Backend:** Create `Line` model with all metadata
3. **Backend:** Update API to return `{lines: Line[]}`
4. **Frontend:** Update document store to consume line-based structure
5. **Frontend:** Create `Line.svelte` component (optional)
6. **Frontend:** Remove regex parsing logic

### Backward Compatibility

For v1 migration, consider supporting both response formats:
- Old: `{results, variables}` (deprecated)
- New: `{lines}` (preferred)

Use API version headers or separate endpoints during transition.

## Success Criteria

This document model is successful if:

1. ✓ Backend naturally exposes parser's line structure
2. ✓ Frontend eliminates regex parsing
3. ✓ Variable locations are exact (no guessing)
4. ✓ Errors map precisely to display positions
5. ✓ API responses are self-describing
6. ✓ Implementation matches spec concepts 1:1
7. ✓ Future features are unblocked (multi-doc, collaboration, etc.)

## References

- [LINE_CLASSIFICATION_SPEC.md](./LINE_CLASSIFICATION_SPEC.md) - Classification rules
- [SEMANTIC_VALIDATION.md](./SEMANTIC_VALIDATION.md) - Validation and diagnostics
- [CommonMark Spec](https://spec.commonmark.org/) - Inspiration for document structure
