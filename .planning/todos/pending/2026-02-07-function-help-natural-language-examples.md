---
created: 2026-02-07T20:29
title: Add natural language examples to function help messages
area: interpreter
files:
  - spec/semantic/function_types.go
---

## Problem

Function help messages should include natural language examples that show how users typically invoke functions in CalcMark's natural language syntax. Currently, functions may only show formal syntax.

Users write things like:
- `average of 2, 4, 3`
- `a = 3` then `average of 3, a, 43`

Function help should display these natural language examples alongside or instead of the formal function signature to make the language more approachable and show idiomatic usage.

## Solution

TBD - Review how function metadata is stored (FunctionDef struct) and add a NaturalLanguageExample field or similar. Update help display to show these examples.
