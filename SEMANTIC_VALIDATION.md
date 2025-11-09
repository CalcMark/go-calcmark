# MarkCalc Semantic Validation

## Overview

The MarkCalc language now has a **semantic validator** that enables application developers to provide rich error feedback to users, including exact positions for highlighting issues (like red squiggles in an editor).

## Key Distinction

MarkCalc separates three distinct concepts:

1. **Classification (Intent Detection)** - "Is this INTENDED to be a calculation?"
2. **Semantic Validation (Static Analysis)** - "Are there issues I can detect before running?"
3. **Evaluation (Runtime)** - "Compute the result"

This separation enables flexible application development where apps can choose how to handle issues at each stage.

## Language-Level vs Application-Level

### Language Provides

The MarkCalc library provides:

- **Classification API**: Determine if a line is CALCULATION, MARKDOWN, or BLANK
- **Validation API**: Check for semantic errors without evaluating
- **Evaluation API**: Execute calculations and return results
- **Rich Diagnostics**: Position information, error codes, serializable to JSON

### Applications Decide

Applications consume these and decide:

- **How** to present diagnostics (red squiggles, tooltips, inline messages)
- **When** to validate (as user types, on save, on demand)
- **Whether** to evaluate invalid calculations or skip them
- **What** to do with different severity levels (errors vs warnings)

## Core Philosophy

> `alice = bob + 2` where `bob` is undefined is a **valid calculation with a semantic error**, not markdown.

Just like in Python, `x = y + 2` is valid Python syntax even if `y` doesn't exist - you just get a `NameError` when you run it. The same principle applies here.

## APIs for Application Developers

### 1. Single Line Validation

```python
from calcdown.markcalc.validator import validate_calculation
from calcdown.markcalc.evaluator import Context

ctx = Context()
result = validate_calculation("alice = bob + 2", ctx)

if not result.is_valid:
    for error in result.errors:
        print(f"{error.message} at {error.range}")
        # error.range.start.column -> where to start red squiggle
        # error.range.end.column   -> where to end red squiggle
        # error.variable_name      -> "bob"
```

### 2. Document Validation

```python
from calcdown.markcalc.validator import validate_document

document = """
x = 5
y = x + 2
z = undefined * 3
"""

results = validate_document(document)
# results = {3: ValidationResult(...)}  # Only line 3 has errors

for line_num, validation in results.items():
    print(f"Line {line_num} has errors:")
    for error in validation.errors:
        print(f"  - {error.message}")
```

### 3. Complete Editor Pattern

```python
from calcdown.markcalc.classifier import classify_line, LineType
from calcdown.markcalc.validator import validate_calculation
from calcdown.markcalc.evaluator import evaluate, Context

context = Context()

for line_num, line in enumerate(document.lines, start=1):
    # Step 1: Classify
    line_type = classify_line(line, context)

    if line_type == LineType.CALCULATION:
        # Step 2: Validate
        validation = validate_calculation(line, context)

        if not validation.is_valid:
            # Show diagnostics (red squiggles, etc.)
            for error in validation.errors:
                editor.show_squiggle(
                    line_num,
                    error.range.start.column,
                    error.range.end.column,
                    error.message
                )
        else:
            # Step 3: Evaluate
            try:
                result = evaluate(line, context)
                editor.show_result(line_num, result)
            except Exception as e:
                editor.show_error(line_num, str(e))
    else:
        # Render as markdown
        editor.render_markdown(line_num, line)
```

## Data Structures

### Diagnostic

```python
@dataclass
class Diagnostic:
    severity: DiagnosticSeverity  # ERROR, WARNING, INFO
    code: DiagnosticCode          # UNDEFINED_VARIABLE, SYNTAX_ERROR, etc.
    message: str                  # Human-readable description
    range: Range                  # Exact position (line:col to line:col)
    variable_name: str | None     # For undefined variable errors

    def to_dict(self) -> dict:
        """Convert to JSON-serializable dict."""
```

### ValidationResult

```python
@dataclass
class ValidationResult:
    diagnostics: list[Diagnostic]

    @property
    def is_valid(self) -> bool:
        """True if no errors (warnings are OK)."""

    @property
    def has_errors(self) -> bool:
        """True if any error-level diagnostics."""

    @property
    def errors(self) -> list[Diagnostic]:
        """Get only error diagnostics."""

    @property
    def warnings(self) -> list[Diagnostic]:
        """Get only warning diagnostics."""

    def __bool__(self) -> bool:
        """Truthy if valid."""
```

### Range & Position

```python
@dataclass
class Position:
    line: int     # 1-indexed
    column: int   # 1-indexed

@dataclass
class Range:
    start: Position
    end: Position
```

## Example: Budget with Typo

```python
document = """
salary = $5000
bonus = $500
savings = salry + bonus  # Typo: 'salry' instead of 'salary'
"""

results = validate_document(document)
error = results[3].errors[0]

# error.message = "Undefined variable: salry"
# error.range.start = Position(line=3, column=11)
# error.range.end = Position(line=3, column=16)
# error.variable_name = "salry"

# Application shows red squiggle under 'salry' from column 11 to 16
```

## JSON Serialization

Diagnostics can be serialized for API responses:

```python
result = validate_calculation("result = foo + bar")

response = {
    "line": "result = foo + bar",
    "diagnostics": [d.to_dict() for d in result.diagnostics]
}

# Returns:
{
  "diagnostics": [
    {
      "severity": "error",
      "code": "undefined_variable",
      "message": "Undefined variable: foo",
      "range": {
        "start": {"line": 1, "column": 10},
        "end": {"line": 1, "column": 13}
      },
      "variable_name": "foo"
    },
    ...
  ]
}
```

## Files Added

- `src/calcdown/markcalc/diagnostics.py` - Diagnostic classes and error types
- `src/calcdown/markcalc/validator.py` - Semantic validation functions
- `tests/unit/markcalc/test_validator.py` - Comprehensive test suite (32 tests)
- `examples/validator_demo.py` - Interactive demonstration

## Files Modified

- `src/calcdown/markcalc/ast_nodes.py` - Added optional `range: Range | None` to all AST nodes
- `src/calcdown/markcalc/parser.py` - Updated to populate position information when creating AST nodes

## Test Coverage

- **267 total tests** passing
- **92% code coverage** overall
- **93% coverage** of validator module
- All edge cases tested: undefined variables, syntax errors, context flow, document validation

## Future Enhancements

The validator framework is extensible for future features:

1. **Type checking**: Detect type mismatches (e.g., "Cannot multiply currency by currency")
2. **Static division by zero**: Detect `x / 0` where 0 is a literal
3. **Warnings**: Unused variables, shadowing, etc.
4. **Auto-fix suggestions**: "Did you mean 'salary' instead of 'salry'?"
5. **Severity levels**: INFO level for suggestions

All of these can be added without breaking existing code, thanks to the extensible diagnostic system.
