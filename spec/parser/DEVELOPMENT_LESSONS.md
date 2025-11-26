# Parser Development Lessons Learned

## Core Principles Established

### 1. **Validate Before Consume** (Critical Pattern)

**Problem:** Calling `p.advance()` before validation creates irreversible state changes.

**Solution:** Always peek→validate→consume:
```go
// ❌ BAD: Side effect before validation
if p.check(lexer.IDENTIFIER) {
    tok := p.advance()  // Consumed!
    if !isValid(tok) {
        // Too late to undo
    }
}

// ✅ GOOD: Validate first
if p.check(lexer.IDENTIFIER) {
    value := string(p.peek().Value)  // Inspect without side effect
    if isValid(value) {
        p.advance()  // Only consume if valid
    }
}
```

**Time Complexity:** O(1) - peek and validate are both constant time operations.

### 2. **Test Full Pipeline, Not Just Units**

**Problem:** Parser tests passed (19/19), CLI failed. Isolated unit tests missed integration issues.

**Solution:** Test through the full execution path:
```go
// Unit test (insufficient)
func TestParser(t *testing.T) {
    nodes, _ := parser.Parse("input")
    // Only tests parser in isolation
}

// Integration test (catches real bugs)
func TestFullPipeline(t *testing.T) {
    doc, _ := document.NewDocument("input")
    eval := NewEvaluator()
    eval.Evaluate(doc)
    // Tests lexer→parser→semantic→interpreter→output
}
```

### 3. **State Management in Parsers**

**Rules for Parser State:**
1. **Immutable Inspection:** Use `p.check()` and `p.peek()` liberally - they're free
2. **Documented Mutations:** Every `p.advance()` should have a comment explaining WHY
3. **Rollback Capability:** For complex patterns, save position: `saved := p.current`
4. **Clear Ownership:** Each function owns its consumption - don't consume tokens meant for caller

**Time Complexity Note:** Parser state mutations (advance, rollback) are O(1) operations. The concern is logical correctness, not performance.

### 4. **Natural Syntax Pattern Detection**

**Challenge:** Natural syntax keywords can conflict with identifiers.

**Solution:** Order checks from most specific to most general:
```go
// ✅ CORRECT ORDER:
// 1. Check specific patterns first (e.g., "downtime per month")
if isDowntimePattern() { ... }

// 2. Then check general patterns (e.g., "X per Y" = rate)
if p.match(lexer.PER) { ... }

// 3. Finally, treat as identifier
return parseIdentifier()
```

### 5. **Error Messages as User Hints**

**Insight:** Parser errors should guide users toward correct syntax.

**Levels:**
- **ERROR:** Syntax violation, cannot continue
- **HINT:** "Did you mean to use 'downtime per month' instead of treating 'downtime' as a unit?"
- **SUGGESTION:** "Common time units: second, minute, hour, day, week, month, year"

**Implementation:** Use semantic checker's diagnostic system, not parser errors.

### 6. **Debug Tracing Strategy**

**What Worked:**
1. Add `fmt.Fprintf(os.Stderr, "DEBUG: ...")` at decision points
2. Log token position: "current token='%s' at index %d"
3. Log AST node types: "created %T"
4. Compare expected vs actual flow

**Cleanup:** Remove debug statements after fix, rely on tests for regression detection.

### 7. **When to Refactor vs. Fix**

**During Bug Fix:**
- ❌ Don't refactor while debugging - changes hide the root cause
- ✅ Add CRITICAL comments explaining the issue
- ✅ Document refactoring opportunities separately

**After Bug Fix:**
- ✅ Safe refactoring: Extract helpers, improve names, add comments
- ✅ Document lessons learned
- ❌ Don't change logic without new tests

### 8. **Time Complexity Considerations**

**Parser Performance Rules:**
1. **Token Operations:** `peek()`, `check()`, `advance()` are all O(1)
2. **Lookahead:** Limited lookahead (1-2 tokens) maintains O(n) overall parsing
3. **Backtracking:** Save/restore position is O(1), but avoid in hot paths
4. **String Operations:** Be careful with `string(tok.Value)` in tight loops

**Safe Patterns:**
- ✅ Peek at next token: O(1)
- ✅ Check token type: O(1)
- ✅ Extract token value: O(1) for small tokens
- ❌ Scan forward through many tokens: O(n)

### 9. **Pure vs. Stateful Functions**

**Pure Functions (Preferred):**
```go
// No parser state mutation
func isTimeUnit(unit string) bool {
    return timeUnits[unit]  // Pure lookup
}
```

**Stateful Functions (Use Carefully):**
```go
// Mutates parser state - document clearly
func (p *Parser) tryConsumeUnit() (string, bool) {
    // MUTATES: Advances cursor if unit found
    if unit, found := validateUnit(); found {
        p.advance()  // Side effect documented
        return unit, true
    }
    return "", false
}
```

**Rule:** If function mutates state, name should indicate action (`consume`, `advance`, `skip`).

### 10. **Documentation Standards**

**Critical Sections Need:**
1. **WHY comment:** Explains reasoning, not just what
2. **EXAMPLE comment:** Shows typical input/output
3. **CRITICAL/IMPORTANT markers:** Flags non-obvious logic
4. **Time complexity:** For non-trivial operations

**Example:**
```go
// CRITICAL: Must validate unit BEFORE consuming identifier.
// Otherwise identifiers like "downtime" after percentages
// get consumed as unit names in "99.9% downtime per month".
//
// Time complexity: O(1) - single hash lookup
// Example: "10 meters" → consumes "meters", returns QuantityLiteral
//          "10 downtime" → doesn't consume, returns NumberLiteral
```

## Checklist for Parser Changes

Before committing parser changes:

- [ ] All token consumption has `p.advance()` AFTER validation
- [ ] Complex patterns have rollback capability
- [ ] New natural syntax has full-pipeline tests
- [ ] Critical sections have explanatory comments
- [ ] No O(n) operations in token lookahead
- [ ] State mutations are documented
- [ ] Error messages guide users to correct syntax

## Common Pitfalls

1. **Consuming before validating** ← This bug
2. **Forgetting to test full pipeline** ← Caused delayed detection
3. **Modifying parser while debugging** ← Can hide root cause
4. **Not documenting validation logic** ← Makes bugs hard to find
5. **Assuming isolated tests are sufficient** ← Miss integration issues

## Future Improvements

1. **Parser Cursor Tracker:** Log consumption history for debugging
2. **Diagnostic Hints:** Surface helpful errors without breaking parsing
3. **Helper Extractors:** Create `tryConsumeX()` helpers for common patterns
4. **Performance Profiling:** Ensure O(n) overall complexity maintained
5. **State Guards:** Add assertions that consumption assumptions hold

---

**Session:** Phase 4 Downtime Function  
**Date:** 2025-11-26  
**Bug:** Parser consumed identifier before validation  
**Impact:** 6 hours debugging, 20 commits  
**Lesson:** Validate before consume - O(1) pattern established  
