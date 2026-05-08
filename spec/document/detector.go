package document

import (
	"strings"

	"github.com/CalcMark/go-calcmark/v2/spec/features"
	"github.com/CalcMark/go-calcmark/v2/spec/lexer"
	"github.com/CalcMark/go-calcmark/v2/spec/parser"
)

// Detector analyzes source text and splits it into blocks.
//
// `recognisedIdentLeads` is the union of:
//   - NL trigger keywords (registry-derived) like `compound`, `read`, …
//   - Registered function names that lex as IDENTIFIER (registry-derived)
//     like `accumulate`, `transfer_time`, `convert_rate`, …
//
// Built-in functions with dedicated lexer tokens (`avg`, `sqrt`, `sum`,
// `number`) are NOT in this set — they are caught earlier in
// `looksLikeCalculation` via `isFunctionToken` and never reach the
// IDENT path.
//
// The classifier consults this set for the IDENT-leading line shapes
// that have markdown ambiguity: `name * other`, `name(args)`, etc. A
// line that lexes as `IDENT op …` or `IDENT LPAREN …` is admitted as
// calculation only when the leading name is in this set OR in the
// doc-scoped `knownNames` (assigned earlier in the same document).
// Anything outside both sets is prose typed by a user who happened to
// hit a token that the lexer recognises (e.g., `also *this` lexing as
// IDENT MULTIPLY IDENT).
type Detector struct {
	nlTriggers           map[string]bool
	recognisedIdentLeads map[string]bool
}

// NewDetector creates a new block detector.
func NewDetector() *Detector {
	r := features.DefaultRegistry()
	triggers := make(map[string]bool)
	for _, kw := range r.NLTriggerKeywords() {
		triggers[kw] = true
	}
	leads := make(map[string]bool, len(triggers))
	for k := range triggers {
		leads[k] = true
	}
	for _, name := range r.IdentCallableFunctionNames() {
		leads[name] = true
	}
	return &Detector{nlTriggers: triggers, recognisedIdentLeads: leads}
}

// DetectBlocks splits source into blocks using these rules:
// - 2 consecutive empty lines = block boundary
// - 1 empty line = part of current block
// - Calculations vs text determined by parsing each line
//
// Unicode-aware: handles all line terminators (LF, CRLF, CR, U+2028, U+2029).
func (d *Detector) DetectBlocks(source string) ([]Block, error) {
	lines := splitLines(source) // Unicode-aware line splitting
	blocks := []Block{}

	currentBlockLines := []string{}
	currentBlockType := BlockText // Default to text
	emptyLineCount := 0
	var pendingEmpties []string // Track trailing empties for TUI line preservation

	// Names defined earlier in the document. Each successful assignment
	// (`name = …`) on a calc line adds its LHS to this set. The
	// classifier consults it to decide whether a leading-identifier
	// line like `x * 5` is calc (when `x` was assigned earlier) or
	// prose (when it was not). Built-in functions and keywords have
	// dedicated lexer token types and are recognised separately, so
	// this set is doc-scoped user names only.
	knownNames := map[string]bool{}

	// Fenced code block state machine
	inFencedCodeBlock := false
	fenceMarker := "" // "```" or "~~~" — the opening fence pattern

	for _, line := range lines {
		isEmpty := isEmptyLine(line) // Unicode-aware empty check

		// Inside a fenced code block, all lines are text regardless of content
		if inFencedCodeBlock {
			if !isEmpty {
				trimmed := strings.TrimSpace(line)
				// Check for closing fence (must match opening marker type)
				if isMatchingCloseFence(trimmed, fenceMarker) {
					inFencedCodeBlock = false
				}
			}
			// Accumulate into current text block (reset empty count to avoid splitting)
			emptyLineCount = 0
			currentBlockLines = append(currentBlockLines, line)
			continue
		}

		if isEmpty {
			emptyLineCount++

			// 2 consecutive empty lines = block boundary
			if emptyLineCount >= 2 {
				// Flush current block (if not empty)
				if len(currentBlockLines) > 0 && !allEmpty(currentBlockLines) {
					blocks = append(blocks, d.createBlock(currentBlockType, currentBlockLines))
					currentBlockLines = []string{}
				}

				// Reset for next block
				emptyLineCount = 0
				// Track this empty line in pendingEmpties - they'll be added to the
				// next block or preserved at end of document for TUI line tracking
				pendingEmpties = append(pendingEmpties, line)
				continue
			}

			// 1 empty line - include in current block
			currentBlockLines = append(currentBlockLines, line)

		} else {
			// Non-empty line
			emptyLineCount = 0

			// Handle pending empties (empty lines beyond block separator)
			if len(pendingEmpties) > 0 {
				if len(blocks) > 0 {
					// Append pending empties to the last block to preserve line count
					lastBlock := blocks[len(blocks)-1]
					switch b := lastBlock.(type) {
					case *CalcBlock:
						b.source = append(b.source, pendingEmpties...)
					case *TextBlock:
						b.source = append(b.source, pendingEmpties...)
					}
				} else {
					// No previous block - these empties are at the start of the document
					// Add them to current block lines so they become part of the first block
					currentBlockLines = append(currentBlockLines, pendingEmpties...)
				}
			}
			pendingEmpties = nil

			// Check for fenced code block opening
			trimmed := strings.TrimSpace(line)
			if marker := getFenceMarker(trimmed); marker != "" {
				inFencedCodeBlock = true
				fenceMarker = marker
				// Force this line and everything until close into a text block
				if len(currentBlockLines) > 0 && currentBlockType != BlockText {
					// Flush current calc block before starting text block
					blocks = append(blocks, d.createBlock(currentBlockType, currentBlockLines))
					currentBlockLines = []string{}
				}
				currentBlockType = BlockText
				currentBlockLines = append(currentBlockLines, line)
				continue
			}

			// Determine if this line is a calculation. Pass the
			// running set of names assigned earlier in the doc so a
			// later `x * 5` line can be promoted to calc when `x`
			// was previously defined; without context, leading-
			// identifier-then-operator lines demote to prose.
			isCalc, err := d.isCalculationWithKnownNames(line, knownNames)
			if err != nil {
				// Lexer error on calc-like line - propagate immediately
				return nil, err
			}
			if isCalc {
				if name := assignmentTargetOf(line); name != "" {
					knownNames[name] = true
				}
			}

			// If first line of new block, set type
			if len(currentBlockLines) == 0 {
				currentBlockType = BlockText
				if isCalc {
					currentBlockType = BlockCalculation
				}
			} else {
				// Check if block type changes
				expectedType := BlockText
				if isCalc {
					expectedType = BlockCalculation
				}

				// If type changes, start new block
				if expectedType != currentBlockType {
					// Flush current block
					blocks = append(blocks, d.createBlock(currentBlockType, currentBlockLines))
					currentBlockLines = []string{}
					currentBlockType = expectedType
				}
			}

			currentBlockLines = append(currentBlockLines, line)
		}
	}

	// Flush remaining block (if not empty)
	if len(currentBlockLines) > 0 && !allEmpty(currentBlockLines) {
		blocks = append(blocks, d.createBlock(currentBlockType, currentBlockLines))
	} else if len(currentBlockLines) > 0 {
		// currentBlockLines is all empty - add to pendingEmpties for preservation
		pendingEmpties = append(pendingEmpties, currentBlockLines...)
	}

	// Handle trailing empty lines (pendingEmpties) - these are empties at end of document
	// that need to be preserved for TUI line tracking
	if len(pendingEmpties) > 0 {
		if len(blocks) > 0 {
			// Append to last block to preserve line count
			lastBlock := blocks[len(blocks)-1]
			switch b := lastBlock.(type) {
			case *CalcBlock:
				b.source = append(b.source, pendingEmpties...)
			case *TextBlock:
				b.source = append(b.source, pendingEmpties...)
			}
		} else {
			// No previous block - create a text block for the empty lines
			blocks = append(blocks, NewTextBlock(pendingEmpties))
		}
	}

	return blocks, nil
}

// allEmpty checks if all lines in a slice are empty.
func allEmpty(lines []string) bool {
	for _, line := range lines {
		if !isEmptyLine(line) {
			return false
		}
	}
	return true
}

// IsCalculation checks if a line is a valid calculation.
// The approach: if a line parses successfully as a calculation, it's a calculation.
// If it fails to parse, it's text (markdown).
//
// Returns (true, nil) for valid calculation lines.
// Returns (false, nil) for text lines (including invalid syntax - treated as markdown).
//
// This is the public API for determining line type. Used by the TUI to
// decide how to render lines in the preview pane.
//
// Has no doc context, so the classifier treats every leading user
// identifier as unknown. `DetectBlocks` calls the context-aware
// `isCalculationWithKnownNames` so prior `name = …` lines promote later
// `name * other` lines correctly.
func (d *Detector) IsCalculation(line string) (bool, error) {
	return d.isCalculationWithKnownNames(line, nil)
}

// IsCalculationWithKnownNames classifies a single line like IsCalculation,
// but admits lines whose leading identifier is in the doc-scoped
// `knownNames` set (e.g. previously-assigned variables) as calculations.
//
// Use this from contexts that already know the document's defined
// names — notably the LSP, where suppressing unit / function completions
// on prose lines hinges on the same calc-vs-text decision the detector
// makes when splitting blocks.
//
// A `nil` `knownNames` is equivalent to calling `IsCalculation`.
func (d *Detector) IsCalculationWithKnownNames(line string, knownNames map[string]bool) (bool, error) {
	return d.isCalculationWithKnownNames(line, knownNames)
}

// LooksLikeCalculation returns true when the token stream at the start of
// a line has the syntactic shape of a CalcMark calculation. Like
// IsCalculation it consults `knownNames` to admit references to
// document-defined identifiers, but unlike IsCalculation it does NOT
// require the line to parse end-to-end.
//
// Use this from in-flight-typing contexts (notably the LSP completion
// provider) where the user's partial input — `accumulate(`, `compound 1000`,
// `revenue * ` — fails the strict parser but is unambiguously calc-shape.
// The token-shape check is what `DetectBlocks` uses internally as its
// final step; this method exposes it directly for callers that already
// have the line and want the detector's verdict without a full parse.
//
// `lineText` may be a single line (no terminating newline required).
// Returns false for empty input. `knownNames` may be nil.
func (d *Detector) LooksLikeCalculation(lineText string, knownNames map[string]bool) bool {
	trimmed := strings.TrimSpace(lineText)
	if trimmed == "" {
		return false
	}
	if hasIndentedCodePrefix(lineText) {
		return false
	}
	if isMarkdownPattern(trimmed) {
		return false
	}
	// Unambiguous calc-shape starters that the lexer rejects in their
	// in-flight-typing form. Catch these BEFORE tokenising so callers
	// (notably the LSP) get the right verdict for partial input the
	// strict lexer chokes on.
	//
	// `@<word>.` (e.g. `@globals.`) is the documented incomplete shape
	// for a directive field reference — the lexer raises
	// `expected field name after '@globals.' (e.g., @globals.tax_rate)`
	// and returns no tokens. There is no markdown construct that
	// starts with `@` followed by a letter, so admit unconditionally.
	// The follow-on completion path will surface the directive's
	// fields, which is exactly the user's intent.
	if isDirectiveLeadingShape(trimmed) {
		return true
	}
	lex := lexer.NewLexer(trimmed)
	tokens, err := lex.Tokenize()
	if err != nil {
		// In-flight `name = @globals.` on the RHS of an assignment
		// also lex-errors on the trailing dot. The presence of an
		// assignment `=` (not `==`/`!=`/`<=`/`>=`) is itself an
		// unambiguous calc signal, so admit when we can find one.
		// This covers the common "user typed `x = @globals.` and
		// expected globals-field completions" path.
		if hasAssignmentEquals(trimmed) {
			return true
		}
		return false
	}
	meaningful := filterNonNewlineTokens(tokens)
	if len(meaningful) == 0 {
		return false
	}
	return d.looksLikeCalculation(meaningful, knownNames)
}

// LooksLikeCalculationForCompletion is the LSP-completion variant of
// `LooksLikeCalculation`. It returns true for everything the strict
// classifier already admits AND admits one extra shape: a single bare
// IDENTIFIER whose value is a case-insensitive prefix of any
// registered IDENT-callable function name (or NL trigger keyword).
//
// Why a separate method? `LooksLikeCalculation` is used by
// `DetectBlocks` to decide whether a line in a document becomes a
// calc block. A paragraph in prose containing the single word
// `compound` on its own line should stay a text block — the strict
// classifier's "single bare IDENT → prose" rule is correct there.
//
// But the LSP completion provider has the user typing in-flight: the
// cursor sits at the end of `com` on a fresh line and the user
// expects `compound` to autosuggest. The same conservative rule that
// protects block detection silences autocomplete on the very first
// character of a function name — the regression cmw flagged on
// 2026-05-07. The lenient variant admits the bare-prefix case only
// when the prefix matches the registered function-name set, so
// arbitrary prose words like `alpha` or `hello` still classify as
// markdown.
//
// Block-detection callers MUST keep using `LooksLikeCalculation`.
// LSP completion classifiers SHOULD use this variant.
func (d *Detector) LooksLikeCalculationForCompletion(lineText string, knownNames map[string]bool) bool {
	if d.LooksLikeCalculation(lineText, knownNames) {
		return true
	}
	trimmed := strings.TrimSpace(lineText)
	if trimmed == "" {
		return false
	}
	if hasIndentedCodePrefix(lineText) {
		return false
	}
	if isMarkdownPattern(trimmed) {
		return false
	}
	lex := lexer.NewLexer(trimmed)
	tokens, err := lex.Tokenize()
	if err != nil {
		return false
	}
	meaningful := filterNonNewlineTokens(tokens)
	if len(meaningful) != 1 {
		return false
	}
	if meaningful[0].Type != lexer.IDENTIFIER {
		return false
	}
	return d.matchesRecognisedIdentLeadPrefix(meaningful[0].Value)
}

// matchesRecognisedIdentLeadPrefix returns true when `prefix` is a
// case-insensitive prefix of any registered IDENT-callable function
// name or NL trigger keyword (the union held in
// `recognisedIdentLeads`). Used by `LooksLikeCalculationForCompletion`
// to admit the bare-IDENT-prefix completion case.
//
// Pure scan — no allocation in the hot path beyond what `strings.HasPrefix`
// requires. The recognisedIdentLeads map is already lower-cased at
// construction (`NewDetector`).
func (d *Detector) matchesRecognisedIdentLeadPrefix(prefix string) bool {
	if prefix == "" {
		return false
	}
	lower := strings.ToLower(prefix)
	for name := range d.recognisedIdentLeads {
		if strings.HasPrefix(name, lower) {
			return true
		}
	}
	return false
}

// hasAssignmentEquals returns true when `s` contains a `=` that isn't part of
// a comparison operator (`==`, `!=`, `<=`, `>=`). Used as a fast calc-admit
// for in-flight typing where the lexer chokes — the presence of a plain `=`
// is enough signal that we're inside a calc assignment regardless of what
// comes after.
func hasAssignmentEquals(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != '=' {
			continue
		}
		// Skip `==`.
		if i+1 < len(s) && s[i+1] == '=' {
			i++ // step past the second `=`
			continue
		}
		// Skip operators that precede `=`: `!`, `<`, `>`.
		if i > 0 {
			prev := s[i-1]
			if prev == '!' || prev == '<' || prev == '>' {
				continue
			}
		}
		return true
	}
	return false
}

// isDirectiveLeadingShape returns true when `trimmed` begins with `@<letter>`
// — the shape of a directive reference (`@scale`, `@globals.tax_rate`,
// `@convert_to`). Catches the in-flight `@globals.` case where the lexer's
// strict directive validation rejects the dot without a following field.
//
// Pure: no side effects, no allocation. Used as a calc-admit fast path
// from `LooksLikeCalculation`.
func isDirectiveLeadingShape(trimmed string) bool {
	if len(trimmed) < 2 {
		return false
	}
	if trimmed[0] != '@' {
		return false
	}
	c := trimmed[1]
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func (d *Detector) isCalculationWithKnownNames(line string, knownNames map[string]bool) (bool, error) {
	trimmed := strings.TrimSpace(line)

	if trimmed == "" {
		return false, nil
	}

	// Indented code blocks: 4+ spaces or tab prefix = markdown, never a calculation
	if hasIndentedCodePrefix(line) {
		return false, nil
	}

	// Explicit markdown patterns are never calculations
	if isMarkdownPattern(trimmed) {
		return false, nil
	}

	// Try to parse the line as a CalcMark expression/statement.
	source := trimmed
	if !strings.HasSuffix(source, "\n") {
		source += "\n"
	}
	_, parseErr := parser.Parse(source)

	// Tokenize for the shape check below + the parser-failure fallback.
	lex := lexer.NewLexer(trimmed)
	tokens, lexErr := lex.Tokenize()
	if lexErr != nil {
		// Lexer rejected the line — historically classified as text
		// at the BLOCK level. Pre-existing tests (`octothorpe in
		// calc expressions` etc.) depend on this behaviour: a calc
		// line with a stray `#` lexes-error and stays as text. Don't
		// add the LooksLikeCalculation fast paths here; they belong
		// at the line-shape API for in-flight typing, not at block
		// detection where the parser is the strict gate.
		return false, nil
	}

	meaningfulTokens := filterNonNewlineTokens(tokens)
	if len(meaningfulTokens) == 0 {
		return false, nil
	}

	if parseErr != nil {
		// User-asked 2026-05-07: `5% of` mid-typing classified as
		// text because the strict parser rejected the partial input.
		// But `5% of <RHS>` is unambiguously calc-shape, so the user
		// expects the editor to stay in calc context the whole way
		// through.
		//
		// Trade-off: we can't fall through to `looksLikeCalculation`
		// unconditionally — NL-trigger English prose like
		// `Read more about this topic` lexes with `Read` as a
		// recognised IDENT lead and would over-admit (regression
		// against `TestNLFunctionVariableDetection`). The parser's
		// rejection of those lines IS the right signal for prose
		// disambiguation.
		//
		// Narrow the fall-through to leading shapes that English
		// prose never starts with: number-shaped tokens (`5`, `5%`,
		// `5K`), an opening paren, a leading `=` (anonymous calc),
		// a unary minus, or a currency literal. IDENT-leading
		// partial parses stay TEXT, preserving the prose
		// classification path.
		first := meaningfulTokens[0]
		if isCalcUnambiguousLeader(first.Type) {
			return true, nil
		}
		return false, nil
	}

	// Parser succeeded — this is a well-formed calc statement. But
	// `Hello` parses as a bare identifier; the token-shape check
	// (`looksLikeCalculation`) is what filters those out as prose.
	return d.looksLikeCalculation(meaningfulTokens, knownNames), nil
}

// isCalcUnambiguousLeader returns true when the given leading token
// type is shape-only enough to admit a partial line as calc even when
// the parser rejects it. Lines starting with these tokens are not
// confusable with English prose.
//
// Used by `isCalculationWithKnownNames` to admit in-flight typing
// like `5% of` (NUMBER_PERCENT lead) or `(1 + 2` (LPAREN lead) as
// calc while keeping NL-trigger prose (`Read more...`) classified as
// text per the parser's rejection.
func isCalcUnambiguousLeader(t lexer.TokenType) bool {
	if isNumberToken(t) {
		return true
	}
	switch t {
	case lexer.ASSIGN, // anonymous calc: `= expr`
		lexer.LPAREN,       // parenthesised expression
		lexer.AT_SIGN,      // directive reference
		lexer.CURRENCY,     // ISO currency code: USD, EUR
		lexer.CURRENCY_SYM, // currency symbol: $, €
		lexer.QUANTITY,     // `5kg`, `100 USD` (lexer pre-folds)
		lexer.MINUS:        // unary negation: `-5`
		return true
	}
	return false
}

// filterNonNewlineTokens returns tokens excluding NEWLINE and EOF.
// Pure function: no side effects.
func filterNonNewlineTokens(tokens []lexer.Token) []lexer.Token {
	result := make([]lexer.Token, 0, len(tokens))
	for _, t := range tokens {
		if t.Type != lexer.NEWLINE && t.Type != lexer.EOF {
			result = append(result, t)
		}
	}
	return result
}

// looksLikeCalculation checks if tokens represent a calculation structure.
// Deterministic, no side effects. Uses nlTriggers from the feature registry
// to recognize NL function patterns without hard-coding keyword lists.
//
// `knownNames` is the doc-scoped set of identifiers assigned earlier in
// the document (built up by `DetectBlocks`). It only matters for the
// leading-IDENTIFIER case: a line that lexes as `IDENT op …` is treated
// as a calculation only when the leading identifier is in `knownNames`,
// is followed by `=` (assignment), or matches an NL-trigger pattern.
// Everything else (prose like `also *this*`, `something + else`) is
// classified as text.
//
// Pass `nil` for `knownNames` to apply the rule at the line level with
// no doc context.
func (d *Detector) looksLikeCalculation(tokens []lexer.Token, knownNames map[string]bool) bool {
	if len(tokens) == 0 {
		return false
	}

	first := tokens[0]

	// Anonymous calculation: = expression
	// Pattern: ASSIGN followed by expression tokens
	if first.Type == lexer.ASSIGN && len(tokens) >= 2 {
		return true
	}

	// Assignment: identifier = ...
	if first.Type == lexer.IDENTIFIER && len(tokens) >= 2 && tokens[1].Type == lexer.ASSIGN {
		return true
	}

	// Expression starting with number (including multiplier suffixes like 5K, 3M)
	if isNumberToken(first.Type) {
		return true
	}

	// Expression starting with quantity (e.g., "10 meters")
	if first.Type == lexer.QUANTITY {
		return true
	}

	// Expression starting with currency symbol or currency token
	if first.Type == lexer.CURRENCY || first.Type == lexer.CURRENCY_SYM {
		return true
	}

	// Expression starting with paren
	if first.Type == lexer.LPAREN {
		return true
	}

	// Function call (built-in functions like avg, sqrt)
	if isFunctionToken(first.Type) {
		return true
	}

	// Boolean literal
	if first.Type == lexer.BOOLEAN {
		return true
	}

	// Unary operators (not, -)
	if first.Type == lexer.NOT || first.Type == lexer.MINUS {
		return true
	}

	// Directive references (@scale, @globals.name)
	if first.Type == lexer.AT_SIGN {
		return true
	}

	// Date literals and keywords
	if isDateToken(first.Type) {
		return true
	}

	// Period-bearing operator (end of / start of) followed by a
	// literal period-bearing token. The look-ahead is required —
	// `end of the day` is common English prose (END_OF + IDENTIFIER)
	// and must not classify as a calculation.
	if isPeriodOperatorToken(first.Type) {
		return looksLikePeriodOperator(tokens)
	}

	// Identifier alone or followed by operator/function call:
	// historically classified as a calculation, but a leading user
	// identifier with no `=` and no doc-context match is almost
	// always prose (e.g., `also *this* is a test`, `alpha * beta`).
	// The user-stated rule: `something *` is calc only when
	// `something` is a known name (defined earlier in the doc) or
	// the line is an assignment.
	if first.Type == lexer.IDENTIFIER {
		// Single identifier is ambiguous - could be variable reference or prose.
		// In document context, treat as prose (text) since undefined vars
		// are common text.
		if len(tokens) == 1 {
			return false
		}
		second := tokens[1]
		leadingName := strings.ToLower(string(first.Value))
		// A leading IDENT is "recognised" if it's:
		//   - in the registry (NL trigger or IDENT-callable function), or
		//   - assigned earlier in this document.
		//
		// Built-in functions like `avg`/`sum`/`sqrt`/`number` have
		// dedicated lexer tokens and are caught by `isFunctionToken`
		// above — they never reach this branch. Anything else IS a
		// user identifier the parser/evaluator has nothing to say
		// about, so we cannot promote it to calc on shape alone.
		leadingRecognised := d.recognisedIdentLeads[leadingName] || knownNames[leadingName]

		// `name(args)` shape: registry-recognised function call OR a
		// reference to a doc-defined name → calc; otherwise the user
		// typed something function-call-shaped using an unknown name
		// (`frobnicate(x)`), which is prose to us — there is no
		// `name(...)` construct in markdown but there is also no
		// CalcMark obligation to evaluate truly unknown names. The
		// downstream layers (parser/LSP) own the "did you mean…?"
		// diagnostic for those.
		if second.Type == lexer.LPAREN {
			return leadingRecognised
		}
		// Identifier followed by an arithmetic / comparison operator
		// (`*`, `+`, `-`, `==`, …) is a calculation only when the
		// leading identifier is recognised. Otherwise this is the
		// prose-typing pattern (`also *t`, `alpha * beta`) and must
		// classify as text.
		if isOperatorToken(second.Type) {
			return leadingRecognised
		}
		// Identifier followed by another identifier = likely prose,
		// EXCEPT NL function patterns where the first token is a known
		// NL trigger keyword (derived from the feature registry).
		// The parser already validated the line parses successfully,
		// so we only need to confirm the trigger keyword matches.
		if second.Type == lexer.IDENTIFIER {
			return d.nlTriggers[leadingName]
		}
		// Identifier followed by a CalcMark keyword (`in`, `as`, …):
		// recognised leads are calc, others are prose ("this is some
		// text" lexes as IDENT KEYWORD IDENT and must not promote).
		if isKeywordToken(second.Type) {
			return leadingRecognised
		}
		// Identifier followed by anything else the parser accepted
		// (NUMBER, QUANTITY, CURRENCY, KEYWORD-flavoured tokens for
		// NL function args like `100 MB from ssd`): only recognise as
		// calc when the leading identifier is recognised. NL triggers
		// are the dominant case here.
		return leadingRecognised
	}

	return false
}

// isNumberToken checks if a token type is a number variant.
// Pure function.
func isNumberToken(t lexer.TokenType) bool {
	switch t {
	case lexer.NUMBER, lexer.NUMBER_PERCENT, lexer.NUMBER_K,
		lexer.NUMBER_M, lexer.NUMBER_B, lexer.NUMBER_T, lexer.NUMBER_SCI,
		lexer.FRACTION:
		return true
	}
	return false
}

// isDateToken checks if a token type is a date-related token.
// Pure function.
func isDateToken(t lexer.TokenType) bool {
	switch t {
	case lexer.DATE_TODAY, lexer.DATE_TOMORROW, lexer.DATE_YESTERDAY,
		lexer.DATE_THIS_WEEK, lexer.DATE_THIS_MONTH, lexer.DATE_THIS_YEAR,
		lexer.DATE_NEXT_WEEK, lexer.DATE_NEXT_MONTH, lexer.DATE_NEXT_YEAR,
		lexer.DATE_LAST_WEEK, lexer.DATE_LAST_MONTH, lexer.DATE_LAST_YEAR,
		lexer.DATE_WEEKDAY, lexer.DATE_THIS_WEEKDAY, lexer.DATE_NEXT_WEEKDAY, lexer.DATE_LAST_WEEKDAY,
		lexer.DATE_THIS_MONTH_NAME, lexer.DATE_NEXT_MONTH_NAME, lexer.DATE_LAST_MONTH_NAME,
		lexer.DATE_THIS_QUARTER, lexer.DATE_NEXT_QUARTER, lexer.DATE_LAST_QUARTER,
		lexer.DATE_THIS_FISCAL_QUARTER, lexer.DATE_NEXT_FISCAL_QUARTER, lexer.DATE_LAST_FISCAL_QUARTER,
		lexer.DATE_THIS_FISCAL_YEAR, lexer.DATE_NEXT_FISCAL_YEAR, lexer.DATE_LAST_FISCAL_YEAR,
		lexer.AGO, lexer.CALENDAR_QUARTER_LITERAL, lexer.FISCAL_QUARTER_LITERAL,
		lexer.FISCAL_YEAR_LITERAL, lexer.CALENDAR_YEAR_LITERAL,
		lexer.DATE_LITERAL, lexer.DURATION_LITERAL:
		return true
	}
	return false
}

// isOperatorToken checks if a token type is an operator.
// Pure function.
func isOperatorToken(t lexer.TokenType) bool {
	switch t {
	case lexer.PLUS, lexer.MINUS, lexer.MULTIPLY, lexer.DIVIDE,
		lexer.MODULUS, lexer.EXPONENT, lexer.ASSIGN,
		lexer.GREATER_THAN, lexer.LESS_THAN, lexer.GREATER_EQUAL,
		lexer.LESS_EQUAL, lexer.EQUAL, lexer.NOT_EQUAL,
		lexer.AND, lexer.OR:
		return true
	}
	return false
}

// isKeywordToken checks if a token type is a CalcMark keyword.
// Pure function.
func isKeywordToken(t lexer.TokenType) bool {
	switch t {
	case lexer.AS, lexer.FROM, lexer.IN, lexer.OF, lexer.PER, lexer.OVER, lexer.WITH:
		return true
	}
	return false
}

// isFunctionToken checks if a token type is a built-in function.
// Pure function.
func isFunctionToken(t lexer.TokenType) bool {
	switch t {
	case lexer.FUNC_AVG, lexer.FUNC_SQRT, lexer.FUNC_SUM, lexer.FUNC_NUMBER,
		lexer.FUNC_AVERAGE_OF, lexer.FUNC_SQUARE_ROOT_OF, lexer.FUNC_SUM_OF:
		return true
	}
	return false
}

// isPeriodOperatorToken reports whether t is a period-bearing
// operator that takes a period inner expression (e.g., `end of <P>`,
// `start of <P>`, `length of <P>`, `days in <P>`) or a custom-period
// constructor (`between A and B`, `from A to B`). Pure function.
//
// LENGTH_OF / DAYS_IN take Period inner — same look-ahead rule as
// END_OF (next token must be period-bearing). BETWEEN / FROM take
// arbitrary expressions; their first operand can be any date-bearing
// or period-bearing token, so the same look-ahead admits them.
func isPeriodOperatorToken(t lexer.TokenType) bool {
	switch t {
	case lexer.END_OF, lexer.START_OF, lexer.LENGTH_OF, lexer.DAYS_IN,
		lexer.BETWEEN, lexer.FROM:
		return true
	}
	return false
}

// looksLikePeriodOperator reports whether a token stream beginning
// with a period-bearing operator (END_OF / START_OF / ...) is a
// calculation. Pure function. The decision rule is look-ahead on
// tokens[1]: literal period-bearing tokens → calc; anything else
// (notably IDENTIFIER) → prose. This keeps `end of the day` (common
// English) out of calc classification while admitting `end of Q1`,
// `end of this fiscal year`, `end of April`, etc.
//
// IDENTIFIER is rejected even though it would unlock the R9
// variable-bound path (`q = Q1; e = end of q`). R9 is deferred to a
// later PR; bare-line variable-bound forms revisit when R9 lands.
// Assignment forms (`x = end of q`) classify via the assignment
// shape, so the deferred path isn't blocked for users who write the
// natural `x = ...` form.
func looksLikePeriodOperator(tokens []lexer.Token) bool {
	if len(tokens) < 2 {
		return false
	}
	return isDateToken(tokens[1].Type)
}

// hasIndentedCodePrefix checks if a line starts with 4+ spaces or a tab,
// indicating a CommonMark indented code block.
func hasIndentedCodePrefix(line string) bool {
	if len(line) == 0 {
		return false
	}
	if line[0] == '\t' {
		return true
	}
	if len(line) >= 4 && line[0] == ' ' && line[1] == ' ' && line[2] == ' ' && line[3] == ' ' {
		return true
	}
	return false
}

// isHorizontalRule checks if a line is a CommonMark horizontal rule.
// Rules: 3+ of the same char (-, *, _) with optional spaces between.
func isHorizontalRule(line string) bool {
	if len(line) < 3 {
		return false
	}
	// Determine the rule character (must be -, *, or _)
	var ruleChar byte
	for i := range len(line) {
		if line[i] != ' ' {
			ruleChar = line[i]
			break
		}
	}
	if ruleChar != '-' && ruleChar != '*' && ruleChar != '_' {
		return false
	}
	count := 0
	for i := range len(line) {
		if line[i] == ruleChar {
			count++
		} else if line[i] != ' ' {
			return false
		}
	}
	return count >= 3
}

// assignmentTargetOf returns the LHS identifier of `name = expr` lines,
// or "" when the line is not a top-level assignment. Lower-cased to
// match the case-folding used by `knownNames` lookups in
// `looksLikeCalculation`. Pure function.
//
// Distinguishes assignment (`=`) from comparison (`==`) by requiring
// the character after `=` to be anything other than another `=`.
func assignmentTargetOf(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}
	// Walk an identifier prefix. Must start with a letter or `_`,
	// then letters/digits/`_`.
	end := 0
	for i, r := range trimmed {
		if i == 0 {
			if !(r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
				return ""
			}
			end = i + 1
			continue
		}
		if r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			end = i + 1
			continue
		}
		break
	}
	if end == 0 {
		return ""
	}
	name := trimmed[:end]
	rest := strings.TrimLeft(trimmed[end:], " \t")
	if !strings.HasPrefix(rest, "=") {
		return ""
	}
	// Reject `==` (comparison) — must be a single `=`.
	if len(rest) >= 2 && rest[1] == '=' {
		return ""
	}
	return strings.ToLower(name)
}

// isFencedCodeFence checks if a line is a fenced code block delimiter.
// Pattern: 3+ backticks or 3+ tildes, optionally followed by info string.
func isFencedCodeFence(line string) bool {
	if len(line) < 3 {
		return false
	}
	if strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~") {
		return true
	}
	return false
}

// isSetextUnderline checks if a line consists only of = or - characters (3+).
// Used for setext-style heading detection.
func isSetextUnderline(line string) bool {
	if len(line) < 3 {
		return false
	}
	char := line[0]
	if char != '=' && char != '-' {
		return false
	}
	for i := range len(line) {
		if line[i] != char {
			return false
		}
	}
	return true
}

// isMarkdownPattern checks if a line matches common markdown patterns.
// These patterns explicitly indicate the line is NOT a calculation.
func isMarkdownPattern(line string) bool {
	// Headers
	if strings.HasPrefix(line, "#") {
		return true
	}

	// Unordered lists (but *= and -= are calculations)
	if strings.HasPrefix(line, "*") && !strings.HasPrefix(line, "*=") {
		return true
	}
	if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "-=") &&
		len(line) > 1 && line[1] == ' ' {
		return true
	}

	// Plus list marker: "+ text" (but not += which is a calculation)
	if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+=") &&
		len(line) > 1 && line[1] == ' ' {
		return true
	}

	// Ordered lists: "1. ", "2. ", "10. " etc.
	// Pattern: digits followed by ". " (dot and space)
	if isOrderedListItem(line) {
		return true
	}

	// Blockquotes
	if strings.HasPrefix(line, ">") {
		return true
	}

	// Images: ![alt](src)
	if strings.HasPrefix(line, "![") {
		return true
	}

	// Inline links: [text](url)
	if strings.HasPrefix(line, "[") && strings.Contains(line, "](") {
		return true
	}

	// Reference-style link definitions: [id]: url
	if strings.HasPrefix(line, "[") && strings.Contains(line, "]:") {
		return true
	}

	// Fenced code block fences
	if isFencedCodeFence(line) {
		return true
	}

	// Horizontal rules (---, ***, ___)
	if isHorizontalRule(line) {
		return true
	}

	// Setext heading underlines (===, ---)
	if isSetextUnderline(line) {
		return true
	}

	// Inline bold/italic: **text** or *text* (not surrounded by spaces like " * ")
	// This catches markdown formatting in prose
	if hasInlineMarkdownFormatting(line) {
		return true
	}

	return false
}

// isOrderedListItem checks if a line is a markdown ordered list item.
// Pattern: one or more digits, followed by ". " (dot and space).
// Examples: "1. First", "10. Tenth", "999. Item"
func isOrderedListItem(line string) bool {
	// Find the dot position
	dotIdx := strings.Index(line, ".")
	if dotIdx <= 0 || dotIdx >= len(line)-1 {
		return false
	}

	// Check that everything before the dot is digits
	for i := range dotIdx {
		if line[i] < '0' || line[i] > '9' {
			return false
		}
	}

	// Check that dot is followed by a space
	return line[dotIdx+1] == ' '
}

// hasInlineMarkdownFormatting detects **bold** and *italic* markdown patterns.
// These are NOT arithmetic operators when immediately adjacent to word characters.
func hasInlineMarkdownFormatting(line string) bool {
	// Look for **text** pattern (bold)
	// The key difference from power operator: ** immediately followed by non-space
	for i := 0; i < len(line)-2; i++ {
		if line[i] == '*' && line[i+1] == '*' {
			// Check if this looks like bold (not power operator)
			// Bold: **word (no space after **)
			// Power: x ** y (spaces around **)
			if i+2 < len(line) && line[i+2] != ' ' && line[i+2] != '*' {
				// Check for closing **
				closeIdx := strings.Index(line[i+2:], "**")
				if closeIdx > 0 {
					return true
				}
			}
		}
	}
	return false
}

// getFenceMarker returns the fence marker ("```" or "~~~") if the line opens a
// fenced code block. Returns "" if not a fence. The marker is the run of
// backticks or tildes (without any info string).
func getFenceMarker(line string) string {
	if len(line) < 3 {
		return ""
	}
	char := line[0]
	if char != '`' && char != '~' {
		return ""
	}
	count := 0
	for i := range len(line) {
		if line[i] == char {
			count++
		} else {
			break
		}
	}
	if count >= 3 {
		return line[:count]
	}
	return ""
}

// isMatchingCloseFence checks if a line is a closing fence that matches the
// opening fence marker. The closing fence must use the same character and have
// at least as many characters as the opening, with no other non-space content.
func isMatchingCloseFence(line string, openMarker string) bool {
	if len(line) < len(openMarker) {
		return false
	}
	char := openMarker[0]
	if len(line) == 0 || line[0] != char {
		return false
	}
	count := 0
	for i := range len(line) {
		if line[i] == char {
			count++
		} else if line[i] != ' ' {
			return false // non-space, non-fence char = not a close fence
		}
	}
	return count >= len(openMarker)
}

// createBlock creates the appropriate block type.
func (d *Detector) createBlock(blockType BlockType, lines []string) Block {
	switch blockType {
	case BlockCalculation:
		return NewCalcBlock(lines)
	case BlockText:
		return NewTextBlock(lines)
	default:
		return NewTextBlock(lines)
	}
}
