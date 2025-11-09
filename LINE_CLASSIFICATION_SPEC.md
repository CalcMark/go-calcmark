# MarkCalc Line Classification Specification

## Philosophy: "Calculation by Exclusion"

**Core Principle:** If a line cannot compile to a valid calculation, it's markdown.

This approach allows seamless mixing of prose and calculations without special delimiters.

## Classification Rules

A line is classified as **CALCULATION** if and only if one of these conditions is true:

### 1. Assignment Statement
```
CONTAINS '=' AND parses successfully as assignment
```

**Examples:**
```
x = 5                     ✓ CALCULATION
salary = $50000           ✓ CALCULATION
my 😁 budget = 1000       ✓ CALCULATION
x =                       ✗ MARKDOWN (parse fails)
hello = world             ✗ MARKDOWN (if "world" undefined)
```

### 2. Arithmetic Expression
```
CONTAINS operators (+, -, *, /)
AND all operands valid (literals or KNOWN variables)
AND parses completely
```

**Examples:**
```
3 + 5                     ✓ CALCULATION
$100 * 52                 ✓ CALCULATION
x * 2                     ✓ CALCULATION (if x is defined)
emoji * 2                 ✗ MARKDOWN (if emoji undefined)
I spent $100 + $50        ✗ MARKDOWN (trailing "I spent")
x * y                     ✗ MARKDOWN (if x or y undefined)
```

### 3. Comparison Expression
```
CONTAINS comparison operators (>, <, >=, <=, ==, !=)
AND all operands valid
AND parses completely
```

**Examples:**
```
1 > 0                     ✓ CALCULATION
x >= 5                    ✓ CALCULATION (if x defined)
salary < $100000          ✓ CALCULATION (if salary defined)
> my quote                ✗ MARKDOWN (no left operand)
x > y                     ✗ MARKDOWN (if x or y undefined)
```

### 4. Single Literal
```
IS exactly one of:
- Number: 42, 3.14, 1,000
- Currency: $100, $1,000.50
- Boolean: true, false, yes, no, t, f, y, n
```

**Examples:**
```
42                        ✓ CALCULATION
$1000                     ✓ CALCULATION
true                      ✓ CALCULATION
hello                     ✗ MARKDOWN (identifier, not literal)
```

### 5. Known Variable Reference
```
IS single identifier
AND identifier exists in current context
```

**Examples:**
```
# After: x = 5
x                         ✓ CALCULATION (prints 5)

# Without definition:
unknown_var               ✗ MARKDOWN
```

## Everything Else is MARKDOWN

- Unknown identifiers
- Multi-word phrases
- Incomplete expressions
- Malformed syntax
- Natural language
- URLs, emails
- Code snippets
- Special prefixes (#, >, -, etc.)

## Context Awareness

The classifier is **context-aware**:

```python
context = Context()

classify_line("x", context)           # MARKDOWN (x not defined)
evaluate("x = 5", context)
classify_line("x", context)           # CALCULATION (x now defined)
classify_line("y = x * 2", context)   # CALCULATION (x is known)
classify_line("z = unknown * 3", ctx) # MARKDOWN (unknown not defined)
```

## Edge Cases

### Trailing Text
If a valid expression is followed by non-expression tokens, it's MARKDOWN:

```
$100 budget               ✗ MARKDOWN (trailing "budget")
5 + 3 equals eight        ✗ MARKDOWN (trailing "equals eight")
```

**Implementation:** Parse must consume all tokens (except trailing whitespace/newline).

### Incomplete Expressions
Parse failures result in MARKDOWN classification:

```
x *                       ✗ MARKDOWN (incomplete)
+ 5                       ✗ MARKDOWN (no left operand)
5 +                       ✗ MARKDOWN (no right operand)
```

### Whitespace
Blank lines are classified as BLANK (subtype of markdown):

```
                          ✗ BLANK
   \t\t                  ✗ BLANK
```

### Special Prefixes (Optional Optimization)
Lines starting with markdown syntax can skip parsing:

```
# Header                  ✗ MARKDOWN (early exit)
> Quote                   ✗ MARKDOWN (early exit)
- List item               ✗ MARKDOWN (early exit)
1. Numbered               ✗ MARKDOWN (early exit)
```

**Note:** This is an optimization. Without it, these would still be classified as markdown after parse failure.

### Unicode Edge Cases

```
給料                      ✗ MARKDOWN (if undefined)
給料 = $5000              ✓ CALCULATION
給料                      ✓ CALCULATION (now defined)

💰 * 2                    ✗ MARKDOWN (if undefined)
💰 = 1000                 ✓ CALCULATION
💰 * 2                    ✓ CALCULATION (now defined)
```

### Operators in Prose

Natural language containing operator characters:

```
I need 5-10 minutes       ✗ MARKDOWN (not a valid expression)
The price is $5+tax       ✗ MARKDOWN ("tax" undefined)
```

These fail to parse as complete expressions, so become markdown.

## Implementation Strategy

```python
def classify_line(line: str, context: Context) -> LineType:
    # 1. Check empty/whitespace
    if line.strip() == "":
        return BLANK

    # 2. Optional: Check markdown prefixes (optimization)
    if line.lstrip().startswith(("#", ">", "-", "*")):
        return MARKDOWN

    # 3. Try to tokenize
    try:
        tokens = tokenize(line)
    except LexerError:
        return MARKDOWN

    # 4. Check for assignment
    if "=" in line:
        try:
            parse(line)
            return CALCULATION
        except ParseError:
            return MARKDOWN

    # 5. Check for operators
    has_operators = contains_operators(tokens)
    if has_operators:
        try:
            ast = parse(line)
            # Verify all identifiers exist
            if all_identifiers_defined(ast, context):
                return CALCULATION
            else:
                return MARKDOWN
        except ParseError:
            return MARKDOWN

    # 6. Single token cases
    if len(tokens) == 1:  # (excluding EOF)
        if is_literal(tokens[0]):
            return CALCULATION
        if is_identifier(tokens[0]) and context.has(tokens[0].value):
            return CALCULATION
        return MARKDOWN

    # 7. Default: markdown
    return MARKDOWN
```

## Testing Strategy

### Unit Tests (Classifier)

Test each rule independently:

1. Assignment detection
2. Operator expression detection
3. Literal detection
4. Variable reference detection
5. Context awareness
6. Edge cases

### Integration Tests (Documents)

Test full documents with mixed content:

1. Calculations remain calculations
2. Prose becomes markdown
3. Context flows across lines
4. Undefined vars in expressions → markdown
5. Errors don't break subsequent lines

### Stress Tests

1. **Random junk:** `@#$%^&*()`
2. **Unicode soup:** Mix of all scripts
3. **Very long identifiers:** 1000+ chars
4. **URLs:** `https://example.com?foo=bar`
5. **Emails:** `user@example.com`
6. **Math-looking prose:** "x is greater than y"
7. **Incomplete expressions:** `x * `, `+ 5`
8. **Natural language with math terms:** "multiply by 2"

## Design Goals

1. **Never crash** - Invalid input becomes markdown
2. **Predictable** - Same input always same output
3. **Context-aware** - Know what variables exist
4. **Conservative** - Err on side of markdown
5. **Fast** - Quick classification without full evaluation
6. **Pure** - No side effects in classification

## Examples: Real Documents

### Example 1: Budget

```
# My Monthly Budget                    MARKDOWN

Income:                                 MARKDOWN
salary = $5000                          CALCULATION
bonus = $500                            CALCULATION

Expenses:                               MARKDOWN
rent = $1500                            CALCULATION
food = $800                             CALCULATION
utilities = $200                        CALCULATION

Total expenses:                         MARKDOWN
expenses = rent + food + utilities      CALCULATION

Savings:                                MARKDOWN
savings = salary + bonus - expenses     CALCULATION
```

### Example 2: Edge Cases

```
Let's calculate something               MARKDOWN
x = 5                                   CALCULATION
The value of x is stored                MARKDOWN
x                                       CALCULATION (prints 5)
y = x * 2                               CALCULATION
z = undefined_var * 3                   MARKDOWN (undefined_var)
Now z is not defined                    MARKDOWN
result = y + 10                         CALCULATION
```

### Example 3: Unicode Mixed

```
給与計算 (Salary Calculation)          MARKDOWN

時給 = $25                              CALCULATION
時間 = 40                               CALCULATION
週給 = 時給 * 時間                      CALCULATION

This 💰 emoji is not defined           MARKDOWN
💰 = $1000                              CALCULATION
Now we can use 💰                       MARKDOWN
total = 💰 + 週給                       CALCULATION
```

## Success Criteria

The classifier is successful if:

1. ✓ All valid calculations are detected
2. ✓ Invalid calculations become markdown
3. ✓ Natural language becomes markdown
4. ✓ No crashes on junk input
5. ✓ Context correctly influences classification
6. ✓ Undefined variables in expressions → markdown
7. ✓ Performance: <1ms per line for typical input
