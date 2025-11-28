# CalcMark Implementation Plan

This document tracks implementation work identified in the gap analysis (Nov 2025).

## Priority 1: Enable Existing Features

### 1.1 Fix Date Tests (dates.cm)
- **Status**: TODO
- **Issue**: `testdata/golden/valid/features/dates.cm` is entirely commented out
- **Task**: Uncomment working date expressions and verify they pass
- **Files**: `testdata/golden/valid/features/dates.cm`

### 1.2 Implement PI and E Constants
- **Status**: TODO
- **Issue**: TODO comment in `impl/interpreter/environment.go:29`
- **Task**: Add mathematical constants PI (3.14159...) and E (2.71828...) to environment
- **Files**: `impl/interpreter/environment.go`

## Priority 2: Complete Partial Features

### 2.1 Implement "X from Y" Date Syntax
- **Status**: TODO
- **Issue**: Parser doesn't handle "2 weeks from today" syntax
- **Evidence**: `spec/parser/date_test.go:75-76` has skip: true
- **Files**: `spec/parser/rdparser.go`, `spec/parser/date_test.go`

### 2.2 Add Semantic Validation for Date Bounds
- **Status**: TODO
- **Issue**: "December 32" parses but should fail semantic validation
- **Evidence**: `spec/parser/date_test.go:148` has skip: true
- **Files**: `spec/semantic/checker.go`, `spec/semantic/date_validation.go` (new)

### 2.3 Add Semantic Checking for Missing AST Nodes
- **Status**: TODO
- **Issue**: Semantic checker doesn't validate Rate, UnitConversion, NapkinConversion, PercentageOf
- **Files**: `spec/semantic/checker.go`

## Priority 3: Implement Logical Operators

### 3.1 Implement AND/OR/NOT Operators
- **Status**: TODO
- **Issue**: Tokens exist but no parser/interpreter support
- **Files**:
  - `spec/parser/rdparser.go` - add parsing
  - `impl/interpreter/operators.go` - add evaluation
  - `spec/ast/nodes.go` - may need LogicalOp node or extend BinaryOp

## Out of Scope (Decided Against)

The following features have reserved tokens but will NOT be implemented:

- **Control flow**: IF/THEN/ELSE/ELIF/END, FOR, WHILE, RETURN, BREAK, CONTINUE
- **Variable modifiers**: LET, CONST

These tokens remain reserved to prevent their use as variable names.

## Completion Tracking

| Item | Status | Date |
|------|--------|------|
| 1.1 Fix Date Tests | | |
| 1.2 PI and E Constants | | |
| 2.1 "X from Y" Syntax | | |
| 2.2 Date Bound Validation | | |
| 2.3 Semantic Node Coverage | | |
| 3.1 Logical Operators | | |
