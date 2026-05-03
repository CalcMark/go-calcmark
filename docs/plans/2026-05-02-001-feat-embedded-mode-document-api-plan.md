---
title: 'feat: structured Document API for Embedded-mode sources'
type: feat
status: active
date: 2026-05-02
---

# feat: structured Document API for Embedded-mode sources

## Summary

Add `calcmark.NewDocumentEmbedded(source string) (*specDoc.Document, error)` — a peer to `calcmark.NewDocument` for the existing `calcmark.Mode` distinction. Where `NewDocument` parses a flat CalcMark source, `NewDocumentEmbedded` parses standard markdown with `cm`/`calcmark` fenced code blocks and returns a real `*specDoc.Document` whose `GetBlocks()` returns properly typed `*specDoc.CalcBlock` per fence and `*specDoc.TextBlock` per passthrough segment. The whole document shares one evaluator session — variables defined in any fence resolve in subsequent fences and in passthrough interpolation. Renders pipeline (`Convert(Embedded, ...)`) is untouched; the new API is a structural-parsing peer that lights up consumers (calcmark-web LSP/document model, future Lark consumer, agent skills) by giving them the same Document shape regardless of source mode.

---

## Problem Frame

calcmark-web is shipping a Phase 1 plan to read Embedded-mode sources as a first-class wire format. Empirical probe in calcmark-web (`web/document_model.go` consumer) confirmed that today's `specDoc.NewDocument` does NOT recognize `` ```cm ``/`` ```calcmark `` fences — it treats them as plain markdown content and returns one big text block. Every consumer downstream of the parser (var resolver, transform configs, line ranges, frontmatter projection, the rendering protocol Block emission to the editor's BlockList / kind-aware affordances / gutter preview) consumes `[]*specDoc.BlockNode` and depends on each calc fence being a `*CalcBlock` with proper structure.

Today, the only Embedded-mode support in go-calcmark is on the **rendering** path (`convert.go` → `convertEmbedded` → `evalEmbeddedBlock` per fence, evaluator state isolated per fence). There is no **structural-parsing** path that returns a `*specDoc.Document`. Consumers that need block structure — not just rendered HTML/Markdown — currently have nowhere to go in the public API.

The wrong fix (which a calcmark-web maintainer almost wrote): replicate `embedded.Scan` + per-segment block projection in calcmark-web. That forks language semantics across two repos. The right fix is the upstream API in this plan — every calcmark consumer gets the same Document shape regardless of source mode.

A load-bearing semantic decision falls out of this plan. Today's `convertEmbedded` evaluates each fence in a fresh `*Document`, so cross-fence variables don't resolve. That's tolerable for the rendering pipeline (each fence renders independently) but is a UX regression for consumers (calcmark-web's editor today resolves `{{ price }}` in prose adjacent to a fence defining `price`). This plan locks **whole-doc evaluator scoping** for the new structural API: all calc fences become CalcBlocks in one `*Document`, so the existing `impldoc.Evaluator.Evaluate(doc)` naturally sees them all and resolves vars across fences and into passthrough interpolation. The rendering pipeline's per-fence-isolation behavior is unchanged (separate code path); only the new structural API gets whole-doc scoping.

---

## Requirements

- **R1.** Public function `calcmark.NewDocumentEmbedded(source string) (*specDoc.Document, error)` in the top-level `calcmark` package, peer to `calcmark.NewDocument` and `calcmark.Convert`. Single-source-string-in, document-or-error-out signature matching the established public-facade shape.
- **R2.** For Embedded sources (markdown with one or more `cm`/`calcmark` fences), the returned Document's `GetBlocks()` returns a list where each `BlockNode.Block` is either a `*specDoc.CalcBlock` (one per fence, source = the fence's inner content) or a `*specDoc.TextBlock` (one per passthrough markdown segment).
- **R3.** Frontmatter handling: a leading `---...---` region of the source is parsed as `*specDoc.Frontmatter` (via existing `spec/document.ParseFrontmatter`), exposed via `Document.GetFrontmatter()` exactly as `NewDocument` does today. The frontmatter region does NOT appear as a passthrough TextBlock.
- **R4.** **Whole-doc evaluator scoping.** When `impldoc.NewEvaluator().Evaluate(doc)` runs against a Document produced by `NewDocumentEmbedded`, variables defined in any CalcBlock are visible to all subsequent CalcBlocks in document order (matching today's flat-CM behavior). Passthrough interpolation (`{{ varName }}` in prose) resolves against the same evaluator state when the consumer evaluates the doc.
- **R5.** Document-absolute line ranges: each block's `BlockNode.Block.Lines()` and any diagnostics emitted during evaluation carry 1-indexed host-document line numbers (NOT line numbers relative to a fence's inner content). The `embedded.Segment.OpenLine` field (1-based) plus per-segment line counts is sufficient to compute these.
- **R6.** Tolerates flat-CM sources by returning a single `TextBlock` for the whole source. (Sources with zero `cm`/`calcmark` fences are equivalent to a single Passthrough segment per `embedded.Scan` semantics.) This makes mode-detection optional for the consumer — they can call `NewDocumentEmbedded` always if they want a uniform call site, at the cost of legacy CM semantics for fence-less sources. Mode-aware consumers like calcmark-web continue to dispatch on `DetectSourceMode` and call the right function.
- **R7.** Reuses `embedded.Scan` from `impl/embedded/scanner.go` as the canonical fence segmenter. No parallel scanner anywhere.
- **R8.** Reuses `spec/document.NewDetector().DetectBlocks` and `spec/document.ParseFrontmatter` as per-segment primitives where applicable. No parallel block-detection logic.
- **R9.** Reuses `impl/document.NewEvaluator()` unchanged — the new API produces a Document that the existing evaluator handles identically to one produced by `NewDocument`.
- **R10.** Zero impact on existing CM-mode behavior. `calcmark.NewDocument`, `calcmark.Convert`, `calcmark.Eval`, and every existing test continue to behave identically. The new API is purely additive.
- **R11.** Spec-impl dependency rule preserved: `spec/document` does NOT import `impl/embedded` or any `impl/...` package. The new API lives where `impl` imports are legal: top-level `calcmark` package (already imports both spec and impl) for the public face; `impl/embedded` (extends today's package) for the structural builder primitive.
- **R12.** Static-rendering escape hatch is the existing language spec convention — any fenced code block whose info-string is NOT exactly `cm` or `calcmark` (e.g., `` ```text ``, `` ```go ``, `` ```output ``) is a regular markdown code block and projects as TextBlock content within the surrounding passthrough segment, NOT as a CalcBlock. No new spec construct, no new info-string. This plan does not invent any language-level affordance.

---

## Scope Boundaries

- **Non-goal: changing the rendering pipeline.** `Convert(input, Options{Mode: Embedded, Format: ...})` keeps its current behavior — per-fence isolated evaluation, fences replaced with rendered output in `Format: "md"` and `Format: "html"`. The structural API and the rendering pipeline are independent code paths with deliberately different semantics for Embedded mode (whole-doc vs. per-fence). This is a real spec-shape divergence; called out in Open Questions but accepted in this plan.
- **Non-goal: new info-strings.** The "any non-cm info-string is passthrough" rule already covers the static-rendering case. Do not invent `calcmark-source`, `calcmark-static`, etc.
- **Non-goal: changing `embedded.Scan`'s segmentation rules.** Scanner is the canonical spec implementation; this plan only adds a new consumer of its existing output.
- **Non-goal: streaming / chunked / incremental parsing.** Full source in memory; single pass.
- **Non-goal: a new Mode option on the existing `NewDocument`.** Considered (`NewDocument(source, opts ...Option)` with `WithMode(Embedded)`) and rejected — see Key Technical Decisions.
- **Non-goal: changes to incremental-update primitives** (`Document.ReplaceBlockSource`, `InsertBlock`, `DeleteBlock`). The new API produces a Document compatible with these, but their behavior is not redesigned for Embedded sources in this plan; that's a follow-up if a consumer needs it.
- **Non-goal: `Convert(Mode: CM)` with fenced sources.** A flat-CM-mode caller passing a fenced source still gets today's behavior (one big text block). Consumers route to the right Mode via their own logic (e.g., calcmark-web's `DetectSourceMode`).
- **Non-goal: performance optimization beyond a single-pass scan + per-segment projection.** Current `Convert(Embedded, ...)` is single-pass; the new API matches that.

### Deferred to Follow-Up Work

- **Convergence of rendering-pipeline scoping with structural-API scoping** — if the per-fence-isolated rendering path proves out of step with consumer expectations, that's a separate plan: either teach `convertEmbedded` to use whole-doc scoping (potentially a breaking change for existing rendering consumers), or document the divergence loudly. Tracked as a real semantic question; not solved here.
- **calcmark-web Phase 1 consumer plan completion** — `calcmark-web/docs/plans/2026-05-02-002-feat-fenced-markdown-wire-format-plan.md` resumes once this plan ships. Its U3 (Embedded-mode parse branch in `BuildDocumentModel`) becomes a one-line change: `if mode == Embedded { doc, err = calcmark.NewDocumentEmbedded(source) }`.
- **Future API: `Mode` option on `NewDocument`** — if more parse-time options accumulate later, consider migrating to `NewDocument(source, opts ...Option)` with backward-compat. Not justified yet by a single addition.

---

## Context & Research

### Relevant Code and Patterns

- **`convert.go`** at the top-level `calcmark` package — establishes the facade pattern this plan extends. `Convert(input, Options{Mode: ...})` dispatches to `convertCM` or `convertEmbedded`; both compose `spec/document` and `impl/...` packages. The new `NewDocumentEmbedded` is a parallel facade in the same package.
- **`spec/document/document.go:38` `NewDocument(source string) (*Document, error)`** — today's CM-mode entrypoint. Parses frontmatter, calls `Detector.DetectBlocks`, wraps blocks in `BlockNode`s with UUIDs, calls `rebuildDependencies`. The new API mirrors this construction pattern but drives detection per-segment instead of per-source.
- **`spec/document/detector.go:32` `Detector.DetectBlocks(source string) ([]Block, error)`** — the segmentation primitive. Use per-segment for passthrough text where multi-block markdown structure (heading + paragraph) is desired; bypass for calc fences where the fence IS the boundary truth.
- **`spec/document/frontmatter.go:616` `ParseFrontmatter(source string) (*Frontmatter, string, error)`** — pulls leading `---...---` and returns the rest. Reuse for the leading-passthrough frontmatter check.
- **`impl/embedded/scanner.go` `Scan(input string) []Segment`** — the canonical fence segmenter. `Segment.Kind` is `Passthrough` or `CalcMarkBlock`; `Segment.Text` is the segment's text (calc fence inner content for CalcMarkBlock; raw markdown for Passthrough); `Segment.OpenLine` is the 1-based host-document line of the opening fence (0 for Passthrough).
- **`impl/document/evaluator.go:115` `Evaluator.Evaluate(doc *spec/document.Document) error`** — the evaluator. Takes a Document, mutates it to populate Results. Handles whole-doc var scoping by virtue of all blocks living in one Document.
- **`convert.go:225` `evalEmbeddedBlock(source string, openLine int, df display.Formatter) embeddedBlockResult`** — the existing per-fence evaluator used by the rendering path. **This plan deliberately does NOT reuse it** — it constructs a fresh Document per fence, which would defeat whole-doc var scoping. Its existence shows the per-fence pattern; the new API uses the opposite pattern.

### Institutional Learnings

- **`docs/solutions/`** has 10 categorized subdirs. None directly cover Embedded-mode parsing or fence-aware Document construction (verified by grep for "embedded", "fence", "markdown.code"). Closest adjacent learning: `docs/solutions/integration-issues/cm-watch-html-templating-css-and-live-reload.md` (Hugo + cm fence rendering on the docs site) — informs how the docs site already consumes Embedded mode for rendering, but doesn't directly apply to the structural API.

### External References

- **CommonMark spec** — fenced code blocks (sections 4.5). Backtick (`` ``` ``) and tilde (`~~~`) fences with arbitrary info-strings. The `embedded.Scan` implementation already honors these correctly; this plan does not need to revisit fence-syntax handling.
- **Goldmark** (github.com/yuin/goldmark, used elsewhere in `convert.go`) — referenced for context but not consumed by this plan; the new structural API does not invoke goldmark.

### Cross-repo references

- **calcmark-web Phase 1 plan:** `calcmark-web/docs/plans/2026-05-02-002-feat-fenced-markdown-wire-format-plan.md` — the consumer plan whose U3 unblocks once this plan ships. Its DetectSourceMode helper (already shipped in calcmark-web/web/source_mode.go) chooses which API to call.
- **calcmark-web prove-it findings:** `calcmark-web/prove-it/current/index.html` (run 2026-05-02-1315) — three calcmark-web bugs (heuristic segmentation bleed, stale chip arrow during edit, leading content disappears after a calc edit) motivate the consumer-side architectural shift this plan substrates. Findings are calcmark-web-side; this plan does not address them directly.

---

## Key Technical Decisions

- **Public API location: top-level `calcmark` package.** Lives next to `Convert` and `Eval` in a new file `embedded_document.go` (or similar). Justification: the spec-impl dependency rule (`spec/` cannot import `impl/`) forbids the originally-proposed `specDoc.NewDocumentEmbedded` location. The top-level `calcmark` package already imports both spec and impl and is the established facade home (per `convert.go`'s pattern). Consumers do `import calcmark "github.com/CalcMark/go-calcmark/v2"` and call `calcmark.NewDocumentEmbedded(...)` — no `impl/...` imports leaked.
- **API shape: separate function `NewDocumentEmbedded`, NOT options on `NewDocument`.** Considered `NewDocument(source string, opts ...Option)` with `WithMode(Embedded)`. Rejected because: (a) options-on-existing-function expands surface area we don't yet need; (b) the function name `NewDocumentEmbedded` advertises its semantics in the call site without requiring the reader to look up Options; (c) future Mode options can still be added later without breaking either function. Composability via two peer functions wins on readability now; options can come if more parse-time configuration accumulates.
- **Internal builder lives at `impl/embedded/document.go`.** Function: `BuildDocument(source string) (*specDoc.Document, error)`. The top-level `NewDocumentEmbedded` is a one-line facade calling `embedded.BuildDocument(source)`. This colocates the Embedded-mode logic with the existing scanner (`embedded.Scan`) and respects the dependency rule (impl can import spec, not vice versa). The split also lets internal tests in `impl/embedded` exercise the builder directly without going through the top-level package.
- **Whole-doc evaluator scoping (R4) — load-bearing decision, user-confirmed.** All CalcBlocks land in the same `*specDoc.Document`. The existing `impldoc.Evaluator.Evaluate(doc)` then naturally sees them all and resolves vars across fences. Passthrough interpolation (`{{ varName }}` in prose) resolves against the same state. Diverges from `convertEmbedded`'s per-fence-isolated rendering behavior; the divergence is intentional and documented in Open Questions and the public API doc. Rationale: matches today's flat-CM evaluator semantics that downstream consumers (calcmark-web's interpolation) depend on; per-fence isolation would break the consumer's UX.
- **Per-segment projection rules:**
  - **CalcMarkBlock segment** → produce a SINGLE `*CalcBlock` containing the fence's inner content as the block source. The fence is the boundary; do NOT call `Detector.DetectBlocks` on the inner content (which could re-segment via heuristics). Construct the block directly via the `spec/document` block-construction primitives.
  - **Passthrough segment** → produce a single `*TextBlock` containing the segment's markdown text. Like CalcMarkBlock, do NOT re-segment via `Detector.DetectBlocks` — that would reintroduce the heuristic-bleed bug calcmark-web's prove-it run surfaced. The fence boundaries from the scanner are the truth; passthrough text is one TextBlock per segment.
  - **Frontmatter exception (R3)** — if the FIRST passthrough segment begins with `---\n`, parse it via `ParseFrontmatter`. Store the resulting `*Frontmatter` on the Document; the frontmatter does NOT also produce a TextBlock. Any text remaining after the closing `---` (rare; usually the Embedded scanner places it in the next passthrough segment) is appended to the next passthrough TextBlock.
- **Block UUID scheme matches `NewDocument`.** Use `uuid.New().String()` per block, populating `BlockNode.ID`, mirroring `NewDocument`'s loop at `spec/document/document.go:60-67`. Consumers that key on block IDs (e.g., for incremental updates) treat Embedded-Document blocks identically to CM-Document blocks.
- **Document construction visibility.** The Document constructor `*Document{...}` literal in `NewDocument` is package-private (lowercase fields). The new builder lives in `impl/embedded` which is a different package. **Implementation question:** does the spec/document package need to expose a constructor like `NewBlankDocument() *Document` or `NewDocumentFromBlocks(fm, blocks) *Document` for impl/embedded to use, or can the builder copy the construction pattern? Resolved during planning: yes, expose a small `spec/document.NewDocumentFromBlocks(fm *Frontmatter, blocks []Block) *Document` helper. It's a thin convenience constructor that wraps blocks in `BlockNode`s with UUIDs and calls `rebuildDependencies` — same logic as `NewDocument`'s body sans the parsing. Reusing this helper from impl/embedded keeps the construction logic in one place and respects the dependency rule.

---

## Open Questions

### Resolved during planning

- **Where does the public API live?** Resolved: top-level `calcmark` package, function `NewDocumentEmbedded`. See Key Technical Decisions for the spec-impl dependency rationale.
- **API shape (peer function vs. options)?** Resolved: peer function. See Key Technical Decisions.
- **Variable scoping (per-fence vs. whole-doc)?** Resolved: whole-doc, user-confirmed. See R4 + Key Technical Decisions.
- **Static-rendering escape hatch?** Resolved: existing language spec already supports it (any non-cm info-string is passthrough). No new construct needed.
- **Document construction primitive sharing?** Resolved: add `spec/document.NewDocumentFromBlocks(fm, blocks) *Document` so impl/embedded composes it without copying the constructor body. See Key Technical Decisions.

### Deferred to implementation

- **`Detector.DetectBlocks` per-segment use vs. forced single-block projection.** The plan locks "one block per segment" at decision time (KTD), but during U2 implementation, edge cases may surface (e.g., a passthrough segment with embedded HTML that would parse weirdly). Implementer should write the test first; if forced single-block projection breaks a real case, escalate before introducing per-segment Detector calls.
- **Frontmatter end-of-segment handling.** If a frontmatter `---...---` ends mid-segment (the closing `---` is followed by more text in the SAME passthrough segment), the trailing text needs to project as a TextBlock. The implementer verifies via test fixture; the leading-frontmatter behavior is the common case but the trailing-text case must be handled (probably by re-using `ParseFrontmatter` which already returns the "remaining" string).
- **CalcBlock construction signature.** `spec/document.CalcBlock` is a struct (verified at `block.go:57`) but the exact constructor (or struct-literal pattern) for impl/embedded to use needs to be confirmed during implementation — possibly requires a small `NewCalcBlock(lines []string) *CalcBlock` helper if the existing struct literal isn't impl-package-friendly.
- **Incremental-update behavior.** `Document.ReplaceBlockSource`, `InsertBlock`, `DeleteBlock` are documented for CM-mode docs. Whether they make sense for an Embedded-Document (where the source has fence delimiters that block-level operations don't preserve) is not addressed in this plan. The implementer should NOT extend their behavior; if a consumer hits an unexpected case, escalate to a follow-up plan.
- **Diagnostic line offsets (R5) edge cases.** `Diagnostic.DocLine` is 1-indexed document-absolute. The builder needs to add `seg.OpenLine` (1-based, opening fence line) to per-block line numbers to map to host-doc coordinates. The exact offset arithmetic — particularly the `+1` for "the line AFTER the opening fence" — needs a test that pins the expected DocLine for a known fixture.

---

## High-Level Technical Design

> *This illustrates the intended call chain and is directional guidance for review, not implementation specification. The implementing agent should treat it as context, not code to reproduce.*

```
Consumer call site (e.g., calcmark-web/web/document_model.go)
  ↓
calcmark.NewDocumentEmbedded(source)                           [top-level facade — embedded_document.go]
  ↓
embedded.BuildDocument(source)                                 [impl/embedded/document.go — internal builder]
  ↓
embedded.Scan(source) []Segment                                [impl/embedded/scanner.go — existing, unchanged]
  ↓
for each segment:
  Passthrough (leading + matches "---\n...\n---") →
    spec/document.ParseFrontmatter(seg.Text) → *Frontmatter    [spec/document/frontmatter.go — existing]
  Passthrough (other) →
    construct one *TextBlock with seg.Text as source           [spec/document/block.go — TextBlock struct]
  CalcMarkBlock →
    construct one *CalcBlock with seg.Text as source           [spec/document/block.go — CalcBlock struct]
  ↓
spec/document.NewDocumentFromBlocks(fm, blocks)                [spec/document/document.go — new helper]
  → wraps blocks in BlockNodes with UUIDs
  → calls rebuildDependencies()
  → returns *Document
  ↓
return to consumer
```

Consumer then runs the existing evaluator unchanged:

```
impl/document.NewEvaluator().Evaluate(doc)                     [impl/document/evaluator.go — existing, unchanged]
  → mutates doc to populate per-block Results
  → variables defined in any CalcBlock are visible in subsequent CalcBlocks (R4)
```

The block-list shape downstream consumers see — `[]*BlockNode` with each `.Block` being either `*CalcBlock` or `*TextBlock` — is identical to what `NewDocument` produces today. The only difference is which segments become which kinds, driven by the Embedded scanner's fence boundaries instead of the heuristic Detector.

---

## Implementation Units

- U1. **Public API surface + signature freeze**

**Goal:** Lock the public API: `calcmark.NewDocumentEmbedded(source string) (*specDoc.Document, error)` exists at the top-level `calcmark` package, returns a placeholder/empty document for now, and has a thin test pinning the signature shape so subsequent units can build out behavior without renaming.

**Requirements:** R1, R10.

**Dependencies:** None.

**Files:**
- Create: `embedded_document.go` (top-level `calcmark` package)
- Test: `embedded_document_test.go`

**Approach:**
- Function lives in a new file in the top-level `calcmark` package, alongside `convert.go` and `eval.go`.
- For U1 only, the body returns `nil, errors.New("not yet implemented")` — purely a signature freeze. The test asserts the function exists with the right shape and returns the sentinel error.
- Doc comment establishes the contract: parses Embedded-mode markdown, returns `*Document` with proper block kinds, whole-doc evaluator scoping, peer to `NewDocument`. Reference Convert's `Mode: Embedded` for the rendering counterpart.

**Execution note:** Test-first. The signature is the contract.

**Patterns to follow:**
- `eval.go:42` `Eval(input string) (*Result, error)` — top-level facade signature pattern.
- `convert.go:92-115` `Convert` — package-level documentation tone.

**Test scenarios:**
- *Happy path:* `calcmark.NewDocumentEmbedded("")` returns `(nil, error)` matching the sentinel — proves the function exists and is callable. Test will be updated in U2 to assert real behavior.
- *Signature shape:* compile-time check that the function returns `(*specDoc.Document, error)` — implicitly enforced by Go's type system; the test just needs to type-assert the return value's first slot.

**Verification:** `go test ./...` passes; `task quality` clean; the function appears in `go doc github.com/CalcMark/go-calcmark/v2 NewDocumentEmbedded`.

---

- U2. **Internal builder + per-segment projection**

**Goal:** Implement the structural builder in `impl/embedded` that drives the new public API. For each Embedded segment, produces exactly one block of the matching kind (one `*CalcBlock` per fence, one `*TextBlock` per non-frontmatter passthrough). Stitches blocks into a real `*specDoc.Document` via the new `NewDocumentFromBlocks` helper.

**Requirements:** R2, R6, R7, R8, R11.

**Dependencies:** U1 (so the public-facade signature is locked and U2's builder is reachable from there).

**Files:**
- Create: `impl/embedded/document.go` — exports `BuildDocument(source string) (*specDoc.Document, error)`.
- Modify: `embedded_document.go` (top-level) — replace U1's stub body with `return embedded.BuildDocument(source)`.
- Create: `spec/document/document_from_blocks.go` (or extend `document.go`) — exports `NewDocumentFromBlocks(fm *Frontmatter, blocks []Block) *Document`. Pure helper that wraps blocks in BlockNodes with UUIDs, sets frontmatter, calls `rebuildDependencies`. Does NOT parse a source; takes pre-built blocks.
- Test: `impl/embedded/document_test.go` — exhaustive parallel-fixture coverage.
- Test: `embedded_document_test.go` (top-level) — happy-path integration test through the public facade.
- Test: `spec/document/document_from_blocks_test.go` — pin the new helper's behavior independently.

**Approach:**
- `BuildDocument` calls `Scan(source)` to get segments.
- Iterate segments in order:
  - `CalcMarkBlock` → construct one `*CalcBlock` with `seg.Text` as its source content. Use existing `spec/document.CalcBlock` constructor (or struct literal if accessible from impl; if not, add a small `NewCalcBlock(lines []string) *CalcBlock` helper to spec/document — sized as a deferred-to-implementation question above).
  - `Passthrough` → construct one `*TextBlock` with `seg.Text` as its source content.
  - **Frontmatter exception:** if the first segment is Passthrough AND `seg.Text` begins with `---\n`, call `ParseFrontmatter(seg.Text)`. Store the returned `*Frontmatter` for the Document; if `ParseFrontmatter`'s "remaining" return is non-empty, project it as a TextBlock.
- Call `spec/document.NewDocumentFromBlocks(fm, blocks)` to produce the final Document.
- Return.

**Execution note:** Test-first against parallel fixtures. For each fixture, write a flat-CM equivalent (where one exists) and an Embedded version, assert they produce structurally equivalent block-kind sequences. Discrepancies are bugs.

**Patterns to follow:**
- `spec/document/document.go:38-79` `NewDocument` — block-wrapping + UUID pattern + `rebuildDependencies` call. The new `NewDocumentFromBlocks` extracts that logic into a pure helper.
- `convert.go:225-252` `evalEmbeddedBlock` — shows per-fence Document construction (which this plan deliberately does NOT use; reference for understanding the existing pattern, not for reuse).

**Test scenarios:**
- *Happy path:* source with one heading + prose + one `cm` fence + trailing prose → builds Document with three blocks: TextBlock (heading + prose, single block), CalcBlock (fence inner content), TextBlock (trailing prose).
- *Happy path:* two `cm` fences separated by prose → five blocks: TextBlock, CalcBlock, TextBlock, CalcBlock, TextBlock (or fewer if the leading/trailing passthrough is empty).
- *Happy path:* source with `cm` fence first, no leading prose → two blocks: CalcBlock, then trailing TextBlock (or one block if no trailing prose).
- *Happy path:* `calcmark` (long-form) info-string → CalcBlock, same as `cm`.
- *Happy path:* tilde fence (`~~~cm`) → CalcBlock (parity with backtick).
- *Happy path:* attribute info-string (`` ```cm {.highlight} ``) → CalcBlock; attribute is ignored at the block-content level.
- *Edge case:* empty source → empty `*Document` with zero blocks (matches today's `NewDocument("")`).
- *Edge case:* source with only prose, no fences → single TextBlock with the whole source as its content (R6 — flat-CM equivalent).
- *Edge case:* `text`-fenced (non-cm) block → projects as part of the surrounding TextBlock (the scanner returns it as Passthrough text); NOT a CalcBlock.
- *Edge case:* `cm-extra` info-string → Passthrough per scanner, projects as TextBlock content.
- *Edge case:* unclosed fence (no closing `` ``` ``) → scanner returns it as Passthrough; projects as TextBlock content. Does not crash.
- *Edge case:* `cm` fence with empty inner content → CalcBlock with empty source. Does not crash.
- *Integration:* source with `cm` fence containing `price = 100`; running `impldoc.NewEvaluator().Evaluate(doc)` against the returned Document populates the CalcBlock's Results. Doc structure flows through the existing evaluator unchanged.

**Verification:** `go test ./impl/embedded/... ./spec/document/...` passes; the parallel-fixture matrix shows structural equivalence between flat-CM and equivalent Embedded sources; `task test` clean; `task quality` clean.

---

- U3. **Whole-doc evaluator scoping verification**

**Goal:** Verify that variables defined in any CalcBlock of an Embedded-Document resolve in subsequent CalcBlocks. This is automatic by virtue of all blocks living in one `*Document` (U2 ensures that), but the test scenarios pin the contract so a future refactor doesn't accidentally split the Document into per-fence pieces.

**Requirements:** R4.

**Dependencies:** U2 (need a working builder).

**Files:**
- Test: `impl/embedded/document_scoping_test.go` — dedicated test file for the scoping invariant.

**Approach:**
- No new code — this unit is purely test coverage. The builder from U2 already produces a single Document; the existing evaluator (`impl/document/evaluator.go`) handles whole-doc scoping naturally.
- Tests construct Embedded sources where `cm` fences depend on each other and assert the evaluator resolves them correctly.

**Execution note:** Test-first.

**Patterns to follow:**
- `impl/document/evaluator_test.go` (or equivalent) — existing evaluator-level test conventions for asserting block Results.

**Test scenarios:**
- *Happy path:* fence A defines `price = 100`; fence B defines `tax = price * 0.1`. After `Evaluator.Evaluate(doc)`, fence B's `tax` Result is `10`.
- *Happy path:* fence A defines `price = 100`; fence B defines `tax = price * 0.1`; fence C defines `total = price + tax`. After evaluation, all three Results resolve correctly.
- *Edge case:* fence A defines `price = 100`; fence B defines `price = 200` (redefinition). Evaluator behaves identically to a flat-CM doc with the same redefinition (whatever today's spec says — typically a warning + last-wins or first-wins; pin whichever).
- *Edge case:* fence B references `undefinedVar`; evaluator emits a diagnostic on fence B's block; fence A is unaffected.
- *Integration:* evaluation order matches document order (fence A precedes fence B in the source means A is evaluated first, so B's reference to A's vars resolves).

**Verification:** Tests pass; the var-scoping contract is documented in the test file's comments so future readers understand the intent.

---

- U4. **Frontmatter handling for Embedded sources**

**Goal:** A leading `---\n...\n---\n` segment of an Embedded source is parsed as `*specDoc.Frontmatter` (via existing `ParseFrontmatter`), exposed via `Document.GetFrontmatter()`, and does NOT also appear as a passthrough TextBlock.

**Requirements:** R3.

**Dependencies:** U2 (the builder needs to recognize the leading-frontmatter case).

**Files:**
- Modify: `impl/embedded/document.go` — add the leading-frontmatter detection branch.
- Test: `impl/embedded/document_frontmatter_test.go` — frontmatter-specific test scenarios.

**Approach:**
- In the builder's segment loop, when iterating the FIRST segment AND it's Passthrough AND its text begins with `---\n`:
  - Call `spec/document.ParseFrontmatter(seg.Text)`.
  - On success: store the `*Frontmatter` for the Document. If the "remaining" return value (post-frontmatter text in the same segment) is non-empty, project it as a TextBlock immediately.
  - On failure: fall back to projecting the segment as a TextBlock (matching today's CM-mode behavior of "broken frontmatter still renders the body").

**Execution note:** Test-first.

**Patterns to follow:**
- `spec/document/document.go:38-50` — how `NewDocument` calls `ParseFrontmatter` and stores the result.
- `spec/document/frontmatter_registry.go` — the source of truth for CalcMark frontmatter keys; the parsing primitive (`ParseFrontmatter`) consults it. No need to touch the registry.

**Test scenarios:**
- *Happy path:* source begins with `---\nglobals:\n  price: 100\n---\n` followed by `cm` fence and prose → `Document.GetFrontmatter()` returns the parsed Frontmatter with `globals.price = 100`; first block is the CalcBlock (no synthesized TextBlock for the frontmatter region).
- *Happy path:* source begins with `---\n...\n---\nSome leading text\n` followed by `cm` fence → frontmatter parsed; first block is a TextBlock containing "Some leading text"; second block is the CalcBlock.
- *Edge case:* source begins with malformed frontmatter (unclosed `---`) → `ParseFrontmatter` returns an error; builder falls back to projecting the whole segment as a TextBlock; `Document.GetFrontmatter()` returns nil.
- *Edge case:* source has no frontmatter (begins with content directly) → `Document.GetFrontmatter()` returns nil; builder skips the frontmatter branch entirely.
- *Edge case:* source's frontmatter region appears MID-source (e.g., `# heading\n---\n...\n---\n`) → does NOT trigger frontmatter handling. Frontmatter is leading-only per CalcMark spec; mid-source `---` blocks are markdown horizontal rules. (Verify CalcMark's CM-mode behavior matches this assumption; if not, this plan's behavior should match CM-mode parity.)
- *Integration:* an Embedded source with frontmatter `globals:` AND a `cm` fence using the global → after evaluation, the global is available in the calc block's evaluator state (matches CM-mode behavior of `globals` being visible to body calc).

**Verification:** Tests pass; frontmatter behavior parity with CM-mode is pinned by parallel fixtures.

---

- U5. **Document-absolute line ranges + diagnostic offsets**

**Goal:** Every block's `Lines()` and any diagnostics emitted during evaluation carry 1-indexed host-document line numbers, NOT line numbers relative to a fence's inner content.

**Requirements:** R5.

**Dependencies:** U2 (builder produces blocks; this unit ensures their line metadata is host-doc-correct).

**Files:**
- Modify: `impl/embedded/document.go` — add per-block line-offset arithmetic during projection.
- Test: `impl/embedded/document_line_ranges_test.go` — line-offset-specific scenarios.

**Approach:**
- The scanner's `Segment.OpenLine` is 1-based and points to the opening fence's host-document line. For a CalcMarkBlock, the inner content's first line is at host-doc line `OpenLine + 1`.
- For a Passthrough segment, the segment text starts at the host-document line that follows the previous segment's closing line.
- During U2's projection, for each block, add the segment's host-doc start line to the block's internal line numbering BEFORE constructing the BlockNode.
- Pre-existing convention from spec/document.Diagnostic (`DocLine int` is 1-indexed document-absolute) is the target.

**Execution note:** Test-first. Pin specific DocLine values for known fixtures so off-by-one bugs surface immediately.

**Patterns to follow:**
- `spec/document/document.go:107-118` `Diagnostic` struct — `Line` is block-relative, `DocLine` is doc-absolute. The two-value pattern is the existing contract.
- `convert.go:226` `contentLine := openLine + 1` — the existing per-fence line offset in `evalEmbeddedBlock`. Same arithmetic applies here.

**Test scenarios:**
- *Happy path:* a `cm` fence opens at host-doc line 5 (1-based) with content `price = 100\ntax = price * 0.1`. The CalcBlock's first content line is at DocLine 6; the second at DocLine 7.
- *Happy path:* a passthrough segment spans host-doc lines 1-3 (heading + blank + paragraph) followed by a `cm` fence opening at line 5. Passthrough TextBlock starts at DocLine 1; CalcBlock content starts at DocLine 6.
- *Happy path:* a fence at line 1 (no leading frontmatter, no leading prose) → CalcBlock content starts at DocLine 2.
- *Error path:* a calc inside a fence at host-doc line 6 has a syntax error. The diagnostic's `DocLine` is 6 (the error's doc-absolute line), NOT 1 (which would be the line relative to the fence's inner content).
- *Edge case:* multiple consecutive fences without intervening prose. Each fence's content lines are correctly offset by the cumulative line count of previous segments.
- *Integration:* run `Evaluator.Evaluate(doc)` on a source with a known-bad calc inside a fence; assert `evaluator.Diagnostics()` returns at least one diagnostic whose `DocLine` matches the host-doc line of the bad calc.

**Verification:** Tests pass; off-by-one bugs in line offset arithmetic are caught at test time, not at consumer-integration time.

---

- U6. **Documentation: surface the new API + the static-rendering convention**

**Goal:** Future contributors and consumers can find the new API in the published docs site. The static-rendering convention ("any non-cm info-string is passthrough") is documented loudly so downstream consumers (calcmark-web, Lark, future agent skills) don't reinvent escape-hatch info-strings.

**Requirements:** R12.

**Dependencies:** U1, U2, U3, U4, U5 (so the doc reflects what's actually shipped).

**Files:**
- Modify: `site/content/docs/go-package.md` — add a section documenting `NewDocumentEmbedded` alongside `NewDocument` and `Convert`. Include the call signature, the whole-doc scoping behavior (R4), and the relationship to `Convert(Mode: Embedded)` (rendering pipeline vs. structural API — different scoping deliberately).
- Modify: `site/content/docs/language-reference.md` — add a sentence under the fenced-code-block section noting the static-rendering escape hatch (any info-string that isn't `cm` or `calcmark` is rendered as plain code, not evaluated). Reference where this is enforced (`embedded.Scan`).
- Modify: `embedded_document.go` (top-level) — package-level doc comment on the new function reiterates the static-rendering convention so it's discoverable from `go doc`.

**Test expectation:** none — documentation-only unit. Verification is by reading the diff and running `task` site preview if needed.

**Patterns to follow:**
- Existing structure of `site/content/docs/go-package.md`.
- AGENTS.md tone: terse, decision-oriented, links to source.

**Verification:** Diff is reviewable; `go doc github.com/CalcMark/go-calcmark/v2 NewDocumentEmbedded` shows the contract; the docs site (when previewed via `task site` or equivalent) renders the new sections without errors.

---

## System-Wide Impact

- **Interaction graph:** The new API is purely additive. Existing entrypoints (`NewDocument`, `Convert`, `Eval`) are unchanged. A new `NewDocumentFromBlocks` helper lands in `spec/document` (consumed only by `impl/embedded`'s builder) — exposed publicly because Go has no "package-friend" visibility, but its contract is "internal helper for parser front-ends."
- **Error propagation:** The new builder returns errors from `ParseFrontmatter` (frontmatter parsing) and from any per-segment block construction. The public facade just forwards them. Errors include enough host-doc line context for the caller to anchor diagnostics.
- **State lifecycle risks:** None new. `*specDoc.Document` is a pure data structure (per its package doc); the new API constructs one without persistent state. The evaluator runs against it identically to today.
- **API surface parity:** Three external surfaces matter:
  - `calcmark.NewDocument` (existing) — unchanged.
  - `calcmark.NewDocumentEmbedded` (new) — peer.
  - `calcmark.Convert(Mode: Embedded)` (existing rendering pipeline) — unchanged. Note the **deliberate scoping divergence** between the rendering pipeline (per-fence isolated) and the new structural API (whole-doc). Documented in U6 + the public function doc.
- **Integration coverage:** The parallel-fixture testing pattern (U2) is the load-bearing assurance — every Embedded fixture asserts structural equivalence to its flat-CM counterpart where one exists. The cross-fence scoping tests (U3) assure the whole-doc behavior. The line-range tests (U5) assure the host-doc line discipline.
- **Unchanged invariants:**
  - Spec-impl dependency rule (`spec/` cannot import `impl/`) — preserved. New API at top-level + `impl/embedded` for the builder.
  - `embedded.Scan` segmentation rules — unchanged.
  - `Convert(Mode: Embedded, ...)` per-fence rendering behavior — unchanged.
  - `NewDocument` flat-CM parsing — unchanged.
  - `Evaluator.Evaluate` — unchanged; works identically against either kind of Document.

---

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| Constructing `*specDoc.CalcBlock` / `*specDoc.TextBlock` from the impl/embedded package requires struct-literal access to private fields the spec package doesn't expose | Add a small constructor helper (`NewCalcBlock(lines []string) *CalcBlock`, `NewTextBlock(lines []string) *TextBlock`) in `spec/document` if needed, gated by U2's implementation discovery. Surfaced as a deferred-to-implementation question. |
| `NewDocumentFromBlocks` helper expands the spec/document API surface in a way that future contributors mis-use (e.g., bypassing parsing entirely for sources that should go through `NewDocument`) | Doc comment on the helper clearly scopes its use: "for parser front-ends that have already done their own segmentation, e.g., the Embedded-mode parser." Discoverability is bounded — most consumers won't see it. |
| Whole-doc scoping in the structural API diverges from per-fence scoping in `Convert(Mode: Embedded)`, confusing consumers who expect consistent semantics | Documented loudly in the public function doc (U6) and in `convert.go`'s doc comment for `convertEmbedded` (cross-reference the new API). The divergence is intentional; the docs name it explicitly so consumers don't trip over it. Tracked as Deferred to Follow-Up Work — convergence is a future plan if it proves needed. |
| Line-offset arithmetic (U5) has off-by-one edge cases that escape unit tests and surface at consumer-integration time (e.g., calcmark-web's gutter diagnostic markers point to wrong lines) | U5 includes specific DocLine pinning for known fixtures, including the leading-fence-at-line-1 corner case. The integration test runs the real evaluator and asserts `Diagnostics().DocLine` matches expected host-doc lines. |
| The implementer accidentally introduces `Detector.DetectBlocks` calls per-segment, reintroducing the heuristic-bleed bug calcmark-web's prove-it surfaced | Key Technical Decisions explicitly forbid per-segment Detector use. U2's tests assert "one block per segment" so any implementer drift is caught at PR review. |
| Backward-compatibility regression: existing CM-mode tests fail unexpectedly because `NewDocumentFromBlocks` introduces a subtle behavioral change to `rebuildDependencies` | `task test` runs the full suite (per AGENTS.md). Any CM-mode regression surfaces immediately. The new helper extracts existing logic verbatim from `NewDocument`'s body — a refactor, not a redesign. |
| The frontmatter "leading passthrough segment" detection in U4 mishandles edge cases (e.g., a `---` horizontal rule on line 1 mistaken for frontmatter) | Reuse `ParseFrontmatter`'s own logic for distinguishing frontmatter from horizontal rules — it already handles this for CM-mode and parity is the goal. The U4 tests pin the parity. |

---

## Documentation / Operational Notes

- **`site/content/docs/go-package.md`** — new section for `NewDocumentEmbedded`. Cross-reference `Convert(Mode: Embedded)` and explicitly call out the structural-vs-rendering scoping divergence.
- **`site/content/docs/language-reference.md`** — sentence on the static-rendering escape hatch in the fenced-code-block section.
- **CHANGELOG.md** — entry on first stable release after this plan ships, calling out the new public API and consumer migration guidance.
- **Cross-repo handoff** — once this plan ships, the calcmark-web Phase 1 plan (`calcmark-web/docs/plans/2026-05-02-002-feat-fenced-markdown-wire-format-plan.md`) resumes. Its U3 collapses to a one-line dispatch on `DetectSourceMode(source)` calling `calcmark.NewDocumentEmbedded` for fenced sources.

---

## Sources & References

- **Consumer plan:** `calcmark-web/docs/plans/2026-05-02-002-feat-fenced-markdown-wire-format-plan.md` (the calling-side plan that consumes this API).
- **Motivating evidence:** `calcmark-web/prove-it/current/index.html` (calcmark-web run 2026-05-02-1315) — three findings (heuristic bleed, stale chip arrow, leading content disappearing) the consumer plan addresses via this substrate.
- **Existing rendering pipeline:** `convert.go` (top-level), specifically `convertEmbedded` (lines 164-221) and `evalEmbeddedBlock` (lines 225-252). Reference for understanding the existing per-fence behavior; this plan does NOT modify it.
- **Existing scanner:** `impl/embedded/scanner.go`; `impl/embedded/scanner_test.go` for info-string spec edge cases.
- **Existing Document constructor:** `spec/document/document.go:38-79` `NewDocument`. The new API mirrors its construction pattern.
- **Existing detector:** `spec/document/detector.go`; per Key Technical Decisions, NOT used per-segment by the new builder.
- **Existing evaluator:** `impl/document/evaluator.go`; consumed unchanged by callers of the new API.
- **AGENTS.md / CLAUDE.md** in this repo — TDD discipline; spec-impl dependency rule; `task test` / `task quality` are the validation gates; `docs/solutions/` is the institutional learning repository.
