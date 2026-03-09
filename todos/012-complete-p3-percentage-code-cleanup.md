---
status: complete
priority: p3
issue_id: "012"
tags: [code-review, cleanup, simplicity]
dependencies: []
---

# Percentage Code Cleanup: Stale Comments, Dead Code, Dispatch Consolidation

## Problem Statement

Multiple review agents identified minor cleanup items in the percentage implementation.

## Findings

1. **Stale comment** at `operators.go:131-138` — describes pre-Percentage behavior, now handled by dedicated widening block 50 lines earlier. Misleading for future maintainers.

2. **Dead code** `NewPercentageFromWhole` in `spec/types/percentage.go:26-30` — defined but never called anywhere. Both literal evaluators handle `%` suffix themselves. YAGNI violation.

3. **Percentage+Percentage dispatch** split across two locations (lines 77-89 and 287-296 in operators.go). Could be consolidated into the widening section for clarity.

4. **Transform.go comment** at `spec/transform/transform.go:54` should include "Percentage" in the list of unchanged types.

5. **`decOne` allocation** in `evalPercentageWidening` — `decimal.NewFromInt(1)` allocated per call, should use package-level `decOne` (already exists in growth_functions.go).

## Acceptance Criteria

- [ ] Stale comments removed from operators.go
- [ ] `NewPercentageFromWhole` removed
- [ ] Percentage+Percentage consolidated near widening block
- [ ] Transform.go comment updated
- [ ] `decOne` reused from package-level var
