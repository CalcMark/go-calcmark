# Phase 8: Documentation - Research

**Researched:** 2026-02-05
**Domain:** CLI documentation, README authoring, terminal recording
**Confidence:** HIGH

## Summary

Phase 8 focuses on creating user-facing documentation that enables new users to understand, install, and use CalcMark within 5 minutes. The research covers three areas: README structure and content best practices for CLI tools, example file creation following the user's decisions, and visual asset generation using VHS (Charmbracelet's terminal recording tool).

The project already has substantial documentation in `docs/README.md` (400-line user guide) and existing example files in `docs/examples/`. The main work is restructuring the root README.md for quick onboarding, creating focused example files in `testdata/examples/`, and generating visual assets (screenshot + hero GIF).

**Primary recommendation:** Use VHS for the hero GIF (already installed on this system), create a simple tape file demonstrating live evaluation, and structure the README with "What -> Install -> Quick Start -> Examples -> Learn More" flow.

## Standard Stack

The tools and formats for this documentation phase:

### Core
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| VHS | latest (installed) | Terminal GIF recording | Charmbracelet ecosystem standard, declarative tape files, reproducible |
| Markdown | CommonMark | README format | GitHub native, universal |
| PNG | - | Screenshots | Lossless, GitHub renders inline |
| GIF | - | Animated demo | GitHub README compatible, no player needed |

### Supporting
| Tool | Purpose | When to Use |
|------|---------|-------------|
| `vhs themes` | List available themes | Choosing theme for recordings |
| `vhs record` | Interactive tape creation | Initial capture before editing |
| macOS Screenshot (Cmd+Shift+4) | Static screenshots | Capturing TUI for README |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| VHS | asciinema + agg | More steps, less control over output |
| GIF | WebM/MP4 | Better quality but GitHub README doesn't auto-play video |
| PNG | WebP | Smaller but less universal support |

**Installation:** VHS already installed at `/opt/homebrew/bin/vhs`. Dependencies (ttyd, ffmpeg) assumed present.

## Architecture Patterns

### Recommended README Structure

Based on CONTEXT.md decisions and CLI documentation best practices:

```
README.md
├── # CalcMark (title + tagline)
├── Screenshot/GIF (hero visual)
├── ## What is CalcMark? (2-3 sentences + motivation)
├── ## Installation (one-liners: brew, binary download)
├── ## Quick Start (step-by-step + copy-paste example)
├── ## Examples (links to testdata/examples/)
├── ## Learn More (links to docs/README.md, help commands)
├── ## License
```

### Example File Structure (per CONTEXT.md)

```
testdata/examples/
├── budget.cm          # Budget/finance - simple, no front matter
├── unit-conversion.cm # Unit conversion - simple, no front matter
└── engineering.cm     # Engineering/capacity - uses YAML front matter
```

Each file: 10-20 lines, one concept, real-world recognizable use case.

### Visual Asset Structure (per CONTEXT.md)

```
docs/images/
├── hero.gif           # Animated: typing + results updating live
└── tui-screenshot.png # Static: source and results side-by-side
```

### Pattern 1: README "What is it?" Section

**What:** Opening section that explains CalcMark in 2-3 sentences
**When to use:** Required per CONTEXT.md - must come before installation

**Example:**
```markdown
# CalcMark

**Calculations embedded in markdown documents.**

CalcMark is a terminal-based calculation language that blends seamlessly with
markdown. Write your thinking in plain text, add calculations that reference
each other, and watch results update as you type.

Unlike spreadsheets, CalcMark files are human-readable, diffable, and live in
your text editor or terminal.
```

### Pattern 2: One-Liner Installation

**What:** Installation commands that fit on one line
**When to use:** Required per CONTEXT.md - no multi-step instructions

**Example:**
```markdown
## Installation

**macOS/Linux (Homebrew):**
```bash
brew install calcmark/tap/calcmark
```

**Download binary:**
- [macOS (Apple Silicon)](https://github.com/CalcMark/go-calcmark/releases/latest)
- [macOS (Intel)](https://github.com/CalcMark/go-calcmark/releases/latest)
- [Linux (x64)](https://github.com/CalcMark/go-calcmark/releases/latest)
- [Windows (x64)](https://github.com/CalcMark/go-calcmark/releases/latest)
```

### Pattern 3: Quick Start Step-by-Step

**What:** Three-step workflow showing create -> write -> run
**When to use:** Required per CONTEXT.md - show full workflow

**Example:**
```markdown
## Quick Start

1. Create a file `budget.cm`:
```
monthly_income = $5000
rent = $1500
savings_rate = 20%
savings = monthly_income * savings_rate
remaining = monthly_income - rent - savings
```

2. Open in the editor:
```bash
cm budget.cm
```

3. Or evaluate from command line:
```bash
cm eval budget.cm
```
```

### Anti-Patterns to Avoid
- **Wall of text before visuals:** Put hero GIF/screenshot near top
- **Multi-step installation:** Keep to one-liners
- **Too many examples:** Link to examples, don't inline them all
- **Technical jargon first:** Lead with "what it does" not "how it works"

## Don't Hand-Roll

Problems with existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Terminal GIF recording | Screen recording + conversion | VHS tape files | Reproducible, scriptable, consistent output |
| Reference docs | Manual copy from code | `cm help functions`, `cm help constants` | Already implemented, stays in sync |
| Installation docs | Custom install scripts | GoReleaser + Homebrew tap | Already configured in Phase 7 |

**Key insight:** The project already has substantial infrastructure (help commands, GoReleaser, existing examples). Phase 8 is assembly and restructuring, not new tooling.

## Common Pitfalls

### Pitfall 1: Outdated Screenshots After Code Changes

**What goes wrong:** Screenshots show old UI, create confusion
**Why it happens:** Manual screenshots not regenerated after updates
**How to avoid:** Use VHS tape files that can be re-run; store tape files in repo
**Warning signs:** Screenshots showing features that no longer exist or look different

### Pitfall 2: GIF File Size Too Large

**What goes wrong:** Hero GIF is 10MB+, slow to load on README
**Why it happens:** Recording too long, dimensions too large, framerate too high
**How to avoid:** Keep recordings under 15 seconds, use 800-1000px width, 10-15fps
**Warning signs:** GIF over 2MB; GitHub shows loading spinner

### Pitfall 3: Example Files Break on Version Changes

**What goes wrong:** Example files have syntax errors after language changes
**Why it happens:** Examples not included in test suite
**How to avoid:** Place examples in testdata/ so they're validated by tests
**Warning signs:** `cm eval testdata/examples/*.cm` fails

### Pitfall 4: Missing Binary Download Links

**What goes wrong:** Users can't find correct binary for their platform
**Why it happens:** Links hardcoded to specific version
**How to avoid:** Use GitHub releases/latest URL pattern, let GoReleaser name files
**Warning signs:** 404 errors on download links

### Pitfall 5: Copy-Paste Examples Don't Work

**What goes wrong:** User copies example, gets error
**Why it happens:** Example has invisible characters, wrong encoding, or depends on undefined variables
**How to avoid:** Test every README code block by copying and running
**Warning signs:** Examples work in file but not when pasted

## Code Examples

### VHS Tape File for Hero GIF

**Source:** VHS documentation + local testing

```tape
# hero.tape - CalcMark demo GIF
Output docs/images/hero.gif

Require cm

Set Shell "bash"
Set FontSize 20
Set Width 900
Set Height 500
Set Theme "Catppuccin Mocha"
Set Padding 20
Set TypingSpeed 75ms
Set Framerate 15

# Show file being opened
Type "cm budget.cm"
Sleep 500ms
Enter
Sleep 1s

# Type some calculations (simulating TUI editing)
Type "monthly_income = $5000"
Sleep 300ms
Enter
Sleep 200ms
Type "rent = $1500"
Sleep 300ms
Enter
Sleep 200ms
Type "savings = monthly_income * 0.20"
Sleep 300ms
Enter
Sleep 1s

# Show final state
Sleep 2s
```

**Note:** VHS cannot interact with TUI apps directly. For hero GIF showing live TUI, use `vhs record` to capture real interaction, then edit the tape file.

### Simple Example File (Budget)

**Format:** Plain CalcMark, no front matter, 10-20 lines

```calcmark
# Monthly Budget

monthly_income = $5000
rent = $1500
utilities = $200
groceries = $400
savings_rate = 20%

total_fixed = rent + utilities + groceries
savings = monthly_income * savings_rate
remaining = monthly_income - total_fixed - savings
```

### Example File with YAML Front Matter (Engineering)

**Format:** YAML front matter with constants, 10-20 lines

```calcmark
---
growth_rate: 0.15
buffer_percent: 0.20
---
# Capacity Planning

current_users = 1M
daily_requests = current_users * 10
requests_per_second = daily_requests / 86400

next_year_users = current_users * (1 + growth_rate)
peak_rps = requests_per_second * 3

servers_needed = peak_rps at 1000 req/s per server with 20% buffer
```

### Taking a Static Screenshot (macOS)

**Process:**
1. Run `cm docs/examples/household-budget.cm` to open TUI
2. Resize terminal to desired dimensions (recommend 120x30)
3. Press Cmd+Shift+4, then Space, then click terminal window
4. Save to `docs/images/tui-screenshot.png`

**Alternative:** Use VHS `Screenshot` command in tape file for reproducible capture.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual screen recordings | VHS tape files | 2022 | Reproducible demos |
| GitHub wiki docs | README + inline docs | Ongoing | Better discoverability |
| ASCII art headers | Simple text + badges | 2020s | Cleaner, more professional |
| Long feature lists | Quick start + links | 2020s | Faster onboarding |

**Current best practices:**
- Hero visual (GIF or screenshot) near top of README
- Installation in one line
- Quick start in 3 steps or less
- Link to detailed docs instead of inlining everything

## Open Questions

Things resolved through research:

1. **VHS vs asciinema for GIF recording**
   - Resolved: VHS is preferred (already installed, simpler workflow, Charmbracelet ecosystem match with bubbletea TUI)

2. **Where to put example files**
   - Resolved: `testdata/examples/` per CONTEXT.md (validated by test suite)

3. **How to handle binary download links**
   - Resolved: Use GitHub releases/latest pattern, GoReleaser handles naming

## Sources

### Primary (HIGH confidence)
- VHS GitHub repository - https://github.com/charmbracelet/vhs
- VHS installed locally - `/opt/homebrew/bin/vhs`, tested with `vhs new`
- Existing project docs - `/Users/bitsbyme/projects/go-calcmark/docs/README.md`
- GoReleaser config - `/Users/bitsbyme/projects/go-calcmark/.goreleaser.yaml`
- Current README - `/Users/bitsbyme/projects/go-calcmark/README.md`
- Existing examples - `/Users/bitsbyme/projects/go-calcmark/docs/examples/`

### Secondary (MEDIUM confidence)
- README best practices - [Make a README](https://www.makeareadme.com/)
- CLI documentation patterns - [Tilburg Science Hub](https://www.tilburgsciencehub.com/topics/collaborate-share/share-your-work/content-creation/readme-best-practices/)
- Awesome README examples - [GitHub](https://github.com/matiassingers/awesome-readme)

### Tertiary (LOW confidence)
- None - all findings verified with local tools and official documentation

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - VHS installed and tested, tools verified
- Architecture: HIGH - Structure based on user decisions in CONTEXT.md
- Pitfalls: MEDIUM - Based on common documentation issues, not project-specific data

**Research date:** 2026-02-05
**Valid until:** Indefinite - documentation patterns are stable
