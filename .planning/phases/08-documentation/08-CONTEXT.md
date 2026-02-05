# Phase 8: Documentation - Context

**Gathered:** 2026-02-04
**Status:** Ready for planning

<domain>
## Phase Boundary

README, examples, and references that help new users understand and start using CalcMark within 5 minutes. Focus on getting users productive quickly with clear installation and quick-start content.

</domain>

<decisions>
## Implementation Decisions

### README structure
- Lead with "What it is" explanation before installation
- Position as "Calculations embedded in markdown documents" (not Jupyter comparison)
- Installation: One-liners only (brew install, binary download links)
- Quick start: Include step-by-step walkthrough AND example file AND link to testdata examples
- Brief motivation explaining WHY CalcMark exists (vs spreadsheets) — one or two sentences

### Example files
- Three example files covering: Budget/finance, Unit conversion, Engineering
- Simple and focused: 10-20 lines each, one concept per file
- Location: testdata/examples/ subdirectory
- Front matter usage: One example uses YAML front matter, others are plain calculations

### Tone & audience
- Primary audience: Developers familiar with CLI tools, markdown, terminal workflows
- Tone: Friendly and helpful — approachable but professional
- Skip CONTRIBUTING.md for now — focus on user-facing docs first

### Visual elements
- Screenshots: Required — at least one showing TUI with source and results side-by-side
- Animated GIF: One hero GIF showing typing and results updating live
- Asset location: docs/images/ directory
- Header: Simple text "# CalcMark" with tagline, no ASCII art

### Claude's Discretion
- Exact README section ordering after the core structure
- Screenshot capture tool and image format
- GIF recording approach (vhs, asciinema, screen recording)
- Exact wording of tagline and motivation paragraph

</decisions>

<specifics>
## Specific Ideas

- Quick start should show the full workflow: create file, add calculations, run cm
- Examples should demonstrate real-world use cases developers would recognize
- Keep docs scannable — developers want to copy-paste and go

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 08-documentation*
*Context gathered: 2026-02-04*
