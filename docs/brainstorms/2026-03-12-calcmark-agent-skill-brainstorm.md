# Brainstorm: Distributable CalcMark Agent Skill

**Date:** 2026-03-12
**Status:** Draft

## What We're Building

A multi-platform agent skill that teaches AI coding agents how to use CalcMark as a calculation and document-creation tool. The skill enables agents to:

1. **Perform calculations** during research and analysis tasks (cost modeling, capacity planning, comparisons, date arithmetic, unit conversions)
2. **Create rich CalcMark documents** as deliverables (budgets, proposals, reports) and convert them to HTML, Markdown, or JSON
3. **Save `.cm` files** as evidence of analytical work and reproducible calculation artifacts

The skill lives in a **new repo under the CalcMark GitHub org** (e.g., `github.com/CalcMark/agent-skill`) and targets **two platforms** at launch: Claude Code and Cursor. OpenAI GPTs deferred until a hosted CalcMark API exists.

## Why This Approach

**Shared Core + Platform Adapters** — a single `CALCMARK-AGENT.md` core document contains all CalcMark knowledge, syntax patterns, workflow guidance, and example references. Thin platform-specific wrappers adapt it to each target:

| Platform | Format | Distribution |
|----------|--------|-------------|
| Claude Code | `SKILL.md` with frontmatter referencing core doc | Compound Engineering marketplace, manual install |
| Cursor | `.cursor/rules/calcmark.mdc` | Cursor rules directory, shareable |
| OpenAI GPT | Instructions text + optional actions | GPT Builder, GPT Store |

This avoids duplicating CalcMark knowledge across platforms while giving each platform its native format.

### Knowledge Strategy: Hybrid

The skill embeds enough CalcMark knowledge inline for common calculations, and supplements with:

- **Runtime discovery**: `cm help functions`, `cm help constants` — the agent runs these to learn available capabilities
- **Online examples**: Agent fetches from `https://calcmark.org/docs/examples/` for deeper worked examples when needed
- **Agent integration docs**: `https://calcmark.org/docs/agent-integration/` as the canonical reference for JSON output structure and CLI patterns

### Installation Strategy

The skill instructs the agent to check for `cm` in PATH and install it if missing:

- **macOS/Linux**: `brew install calcmark/tap/calcmark`
- **Other platforms**: Download binary from GitHub releases (`github.com/CalcMark/go-calcmark/releases`)

## Key Decisions

1. **New repo at `github.com/CalcMark/agent-skill`** — clean separation from the interpreter, independent versioning, external contributors welcome
2. **Shared Core + Adapters** over MCP server or template generator — simplest approach, single source of truth, no build step
3. **Two platforms at launch**: Claude Code and Cursor — GPTs deferred until hosted API exists
4. **Hybrid knowledge**: inline basics + `cm help` runtime discovery + website examples — works offline for basics, always up-to-date for advanced features
5. **Agent installs `cm` autonomously** — skill provides platform-appropriate install commands

## Core Document Structure (CALCMARK-AGENT.md)

The shared core document should cover:

### 1. What is CalcMark
One paragraph: blends markdown and calculations in one document. Think "spreadsheet meets markdown."

### 2. Installation Check & Setup
Platform-detect and install `cm` if not present.

### 3. Essential Syntax
Inline the basics the agent needs for 80% of tasks:
- Variables and expressions: `price = 100 USD`, `total = price * 12`
- Units and conversions: `distance = 5 km to miles`
- Dates and durations: `deadline = 2026-04-01`, `remaining = deadline - today`
- Rates: `speed = 60 km/hr`, `time = 100 km at speed`
- Napkin math: `~1M users * ~500 bytes`
- Functions: `round()`, `min()`, `max()`, `abs()`, and NL equivalents
- Frontmatter: `---\ntitle: Budget\nlocale: en-US\n---`

### 4. CLI Patterns for Agents
- Pipe: `echo "expr" | cm --format json`
- Heredoc for multi-line documents
- File eval: `cm eval budget.cm --format json`
- Convert: `cm convert doc.cm --to=html`
- Feature discovery: `cm help functions`, `cm help constants`

### 5. Workflow Patterns
- **Quick calculation**: Pipe a one-liner, extract result from JSON
- **Research artifact**: Write a `.cm` file, eval it, save both source and results
- **Document deliverable**: Write `.cm` file, convert to HTML/MD for the user
- **Iterative analysis**: Build up a `.cm` file across multiple steps, re-eval as variables are refined

### 6. JSON Output Reference
Document the response structure so the agent can parse results programmatically.

### 7. Learn More
Link to `https://calcmark.org/docs/agent-integration/` and `https://calcmark.org/docs/examples/`

## Platform Adapter Details

### Claude Code (SKILL.md)
```yaml
---
name: calcmark
description: Use CalcMark to perform calculations, create computational documents, and produce analysis artifacts (.cm files, HTML, JSON, Markdown).
allowed-tools: Bash(cm:*), Bash(brew:*), Bash(curl:*), Read, Write, WebFetch
---
```
References or inlines `CALCMARK-AGENT.md` content. Uses `WebFetch` for pulling examples from calcmark.org.

### Cursor (.cursor/rules/calcmark.mdc)
Cursor rules use a similar markdown format. The core content maps directly; the frontmatter differs slightly (Cursor uses `description` and `globs` fields).

### OpenAI GPT (Deferred)
Requires a hosted CalcMark API for GPTs to execute calculations. Revisit when API infrastructure exists.

## Repo Structure

```
github.com/CalcMark/agent-skill/
  README.md                    # Overview, quick start, platform links
  CALCMARK-AGENT.md            # Shared core knowledge document
  platforms/
    claude-code/
      SKILL.md                 # Claude Code skill wrapper
    cursor/
      calcmark.mdc             # Cursor rules file
  examples/
    quick-calc.cm              # Agent workflow examples
    research-artifact.cm
    budget-deliverable.cm
```

## Resolved Questions

1. **GPT Store distribution**: Dropped from initial scope. GPTs can't run local CLI tools, and standing up a hosted API adds infrastructure burden. Focus on Claude Code + Cursor first; revisit GPTs when a hosted CalcMark API exists.

2. **Version pinning**: No version checking. Graceful degradation — if a `cm` command fails, the agent sees the descriptive error and adapts. Simpler and avoids maintenance of version compatibility matrices.

3. **Platform scope revised**: Two platforms at launch (Claude Code + Cursor), not three. GPT dropped per Q1 above.

## Open Questions

1. **Cursor marketplace**: Does Cursor have a rules marketplace or sharing mechanism, or is distribution purely manual (copy the `.mdc` file)?

2. **Compound Engineering marketplace**: What's the submission/publishing process for Claude Code skills? Is there a registry, review process, or just a repo convention?
