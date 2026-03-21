---
title: "Worktree binary path confusion: ./cm vs system cm"
category: build-errors
date: 2026-03-20
severity: medium
tags:
  - worktree
  - binary
  - path
  - debugging
  - task-build
module: Build / CLI
symptom: "Feature appears broken in TUI but works in all tests and CLI — running wrong binary"
root_cause: "Shell PATH resolves `cm` to /opt/homebrew/bin/cm (installed release) instead of ./cm (worktree build)"
---

# Worktree binary path confusion: `./cm` vs system `cm`

## Problem

After running `task build` in a git worktree, the freshly built `./cm` binary is not the one that executes when you type `cm` without a path prefix. The shell resolves `cm` to the globally installed release binary (e.g., `/opt/homebrew/bin/cm`) instead of the local `./cm` in the worktree root.

### Exact symptoms

- `task build` succeeds and produces `./cm` in the worktree directory
- All Go tests pass (`task test` reports no failures)
- Piping input to `./cm` works correctly and reflects the latest code changes
- Running `cm` (no `./` prefix) launches the TUI using an **older release binary** that lacks recent fixes
- A feature or fix appears broken in the TUI but works in every programmatic test

## Root cause

`task build` writes the binary to the current working directory as `./cm`. The shell's `PATH` resolution ignores the current directory (`.` is not in `PATH` by default on macOS) and finds `/opt/homebrew/bin/cm` first. The two binaries are different versions with different behavior.

This is standard Unix behavior, not a bug in `task` or Go. It is especially easy to overlook in worktrees because the worktree path is long and unfamiliar, making it less obvious that you are not in the main repository checkout.

## The debugging red herring

This issue consumed 30+ minutes of investigation during the `@scale` directive feature implementation:

1. A `{{@scale}}` interpolation fix was implemented and all Go unit and integration tests passed.
2. Manual verification via `echo "..." | ./cm --format json` confirmed the fix worked.
3. The user launched the TUI by typing `cm file.cm` (without `./`) to do a final visual check.
4. The TUI showed the old, broken behavior — `{{@scale}}` was not interpolating.
5. This triggered a deep investigation into TUI rendering paths, cache invalidation, glamour markdown rendering, and Side-by-Side vs Rendered mode differences.
6. The actual cause was simply that `cm` resolved to the old Homebrew-installed release binary, which did not contain the fix.

The red herring was convincing because: tests passed, the CLI worked when explicitly invoked as `./cm`, and the TUI is a complex subsystem where subtle rendering bugs are plausible.

## Solution

### Always use `./cm` explicitly in worktrees

```bash
# Correct — runs the worktree build
./cm file.cm
./cm --format json < input.cm

# Wrong — runs whatever the shell finds in PATH
cm file.cm
```

### Verify before debugging

Before investigating any "TUI-only" or "runtime-only" bug, confirm which binary you are running:

```bash
# Check which binary the shell resolves
which cm
# → /opt/homebrew/bin/cm  ← WRONG (installed release)

# Check your local build
./cm version
# → v1.9.3-12-g24e366b   ← RIGHT (worktree build)

# Compare modification times
ls -la ./cm "$(which cm)"
```

If the version strings differ, you are running the wrong binary.

## Prevention

1. **Always use `./cm` in worktrees.** Never bare `cm` when testing local changes.
2. **Check `./cm version` after `task build`** to confirm the binary reflects your latest commits.
3. **If a fix "works in tests but not in TUI"**, the first check should be `which cm` — not a deep dive into rendering code.

## Related

- `docs/solutions/build-errors/consolidate-taskfile-race-condition-orphaned-tasks.md` — Other Taskfile build pipeline issues
- `docs/solutions/test-failures/user-config-leaks-into-tests.md` — Environment leakage pattern (same class of problem: wrong environment, right code)
