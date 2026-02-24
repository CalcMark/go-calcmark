---
title: "Consolidate unwieldy Taskfile.yml: 35 to 15 tasks, fix race condition in release, remove orphaned tasks"
date: 2026-02-24
problem_type: build-errors
severity: high
components:
  - Taskfile.yml
  - CONTRIBUTING.md
  - impl/README.md
tags:
  - build-system
  - task-automation
  - race-condition
  - technical-debt
  - documentation
related_issues: []
resolution_time: "4-6 hours"
root_cause: "Taskfile.yml grew to 35 public tasks with 12 orphaned internal tasks, parallel deps execution caused race condition between clean and test/quality, double test execution in bench task, and documentation became stale"
slug: "consolidate-taskfile-race-condition-orphaned-tasks"
---

## Problem

The project's Taskfile.yml grew to 35 public tasks over time. During a consolidation effort and subsequent multi-agent code review, five issues were discovered:

1. **Race condition in `release`**: The `release` task used `deps:` (parallel execution) for `clean`, `quality`, `test`, and `security`. Because `clean` runs `go clean -cache`, it could wipe the build cache while other tasks were mid-execution.
2. **Double test execution**: The `bench` task ran `go test ./... -bench=. -benchmem` without `-run='^$'`, causing all unit tests to execute alongside benchmarks. When `release` called both `test` and `bench`, every test ran twice.
3. **12 orphaned internal tasks**: Tasks marked `internal: true` that no other task referenced became unreachable dead code.
4. **Quality gate too slow**: Including benchmarks in the pre-commit `quality` task made it impractical for frequent use.
5. **Stale documentation**: CONTRIBUTING.md and impl/README.md referenced tasks that were now internal or deleted.

## Root Cause

Incremental task additions over time without holistic review of task dependencies, execution order, and discoverability.

**Race condition**: Taskfile v3 `deps:` runs dependencies in parallel by default. The `release` task listed `clean`, `quality`, `test`, and `security` as deps. Because `clean` runs `go clean -cache`, it could wipe the build cache while `quality`, `test`, or `security` were mid-execution. This is a non-deterministic failure that only manifests under certain timing conditions.

**Double test execution**: The `-bench` flag adds benchmark execution on top of the default test run. Without `-run='^$'` to suppress the test matcher, all unit tests execute in addition to the benchmarks.

**Orphaned tasks**: Tasks marked `internal: true` are hidden from `task --list`. When no other task references them, they become unreachable dead code that still occupies the config and can mislead contributors.

## Solution

### 1. Sequential release pipeline

Changed `release` from parallel `deps` to sequential `cmds`:

```yaml
# Before (parallel deps -- race condition)
release:
  deps:
    - clean
    - quality
    - test
    - security
  cmds:
    - task: build:all

# After (sequential cmds -- deterministic order)
release:
  desc: Build release binaries (runs all quality checks first)
  cmds:
    - task: clean
    - task: quality
    - task: test
    - task: bench
    - task: security
    - task: build:all
    - echo 'Release binaries built in dist/'
```

### 2. Benchmark-only execution

Added `-run='^$'` to suppress unit test execution during benchmarks:

```yaml
# Before (runs all tests AND benchmarks)
bench:
  cmds:
    - go test ./... -bench=. -benchmem

# After (benchmarks only)
bench:
  internal: true
  cmds:
    - go test ./... -run='^$' -bench=. -benchmem
```

### 3. Removed 12 orphaned internal tasks

Deleted entirely from Taskfile.yml: `test:lexer`, `test:parser`, `test:semantic`, `test:interpreter`, `test:integration`, `test:e2e`, `dev:repl`, `dev:eval`, `vet`, `lint:strict`, `bench:lexer`, `bench:parser`, `deps`.

### 4. Separated bench from quality

Removed `bench` from the `quality` task (used as a pre-commit gate) and kept it only in `release` where it serves as a pre-release verification step.

### 5. Updated documentation

- CONTRIBUTING.md: Rewrote "Common Tasks" section to list only visible `task --list` targets
- impl/README.md: Replaced `task test:interpreter` with `go test ./impl/interpreter/... -v`

## Verification

1. Run `task release` multiple times -- no intermittent cache failures
2. Run `task bench` -- output contains only `Benchmark` lines, no `--- PASS` test output
3. Run `task --list` -- exactly 15 public tasks
4. Run `grep -c "test:lexer\|dev:repl\|lint:strict" Taskfile.yml` -- returns 0
5. Run `task test` -- full suite passes, no regressions

## Prevention Strategies

1. **Prefer `cmds` over `deps` for ordered pipelines**: In Taskfile v3, `deps` runs in parallel. Use `cmds` with `task:` calls when execution order matters (especially when one step modifies shared state like the build cache).

2. **Audit internal tasks periodically**: When marking tasks `internal: true`, grep for all references. If a task has zero callers, delete it entirely rather than hiding it.

3. **Always use `-run='^$'` with `-bench`**: Go's `go test -bench=.` runs matching tests before benchmarks. To run benchmarks only, add `-run='^$'` to match no test names.

4. **Keep commit gates fast**: Pre-commit quality gates (`task quality`) should complete in under 60 seconds. Slow checks like benchmarks and fuzzing belong in release pipelines or CI, not developer workflows.

5. **Update docs alongside config changes**: When modifying Taskfile.yml, search CONTRIBUTING.md and README files for task references. Stale documentation is worse than no documentation because it actively misleads.

## Related Documentation

- [CONTRIBUTING.md](../../CONTRIBUTING.md) -- Developer workflow documentation
- [impl/README.md](../../impl/README.md) -- Implementation package documentation
- [Taskfile v3 docs](https://taskfile.dev/) -- Official Taskfile documentation
