---
status: complete
priority: p3
issue_id: "005"
tags: [code-review, agent-native, cli]
dependencies: []
---

# Add --format Flag to cm eval for Agent Consumption

## Problem Statement

`cm eval` currently outputs plain text. Agents and scripts would benefit from structured output (JSON, CSV) for programmatic consumption.

## Findings

- Source: agent-native-reviewer agent
- `cm eval` hardcodes `format.GetFormatter("text", "")` in eval.go
- JSONFormatter already exists but isn't accessible via `cm eval`
- Related: `cm convert` could also benefit from stdin support

## Proposed Solutions

### Option A: Add --format flag
- **Pros:** Agents get structured output, consistent with other CLI tools
- **Cons:** Adds flag complexity
- **Effort:** Small
- **Risk:** None

## Acceptance Criteria

- [ ] `cm eval --format json` produces JSON output
- [ ] `cm eval --format text` (default) produces current text output
- [ ] Test coverage for both formats
