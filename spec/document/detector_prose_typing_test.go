package document

import "testing"

// Pins down a class of bugs reported by users:
// while typing prose like `also *this* is a test` into an empty / new
// block, intermediate substrings (`also *t`, `also *th`, `also *this`)
// were being classified as calculations because the lexer sees
// `IDENT MULT IDENT` and the structural rule "identifier followed by
// operator = calculation" fires. The first identifier (`also`) is not
// a known name, no `=` follows — so these lines must be treated as
// prose (text), not calculation.
//
// The rule we encode here, distilled from the user's statement:
//
//   `something *` is NOT a calculation unless `something` is an
//   existing variable name, function, or keyword. That sits alongside
//   `*…*` and `_…_` (single delimiter) being recognised as markdown
//   emphasis the same way `**bold**` already is.
//
// The current Detector misclassifies these as calc, which propagates
// to the editor and makes `*` highlight/parse as multiply while the
// user is trying to write a sentence.

func TestProseWithSingleAsteriskEmphasisIsNotCalculation(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name string
		line string
	}{
		// Final user-visible string.
		{"sentence with single-asterisk emphasis", "also *this* is a test"},
		// Same shape, different leading word — proves the rule isn't
		// `also`-specific.
		{"sentence with leading 'and' and emphasis", "and *this* is some text"},
		// Single-underscore italic is also markdown.
		{"sentence with single-underscore emphasis", "also _this_ is a test"},
		// Plain prose with no emphasis at all.
		{"plain prose sentence", "This is some prose"},
		// Each intermediate keystroke from typing
		// `also *this* is a test`. Every step must stay text — anything
		// else means the editor flips the block kind mid-typing.
		{"keystroke: also *", "also *"},
		{"keystroke: also *t", "also *t"},
		{"keystroke: also *th", "also *th"},
		{"keystroke: also *thi", "also *thi"},
		{"keystroke: also *this", "also *this"},
		{"keystroke: also *this*", "also *this*"},
		{"keystroke: also *this* ", "also *this* "},
		{"keystroke: also *this* i", "also *this* i"},
		{"keystroke: also *this* is", "also *this* is"},
		{"keystroke: also *this* is ", "also *this* is "},
		{"keystroke: also *this* is a", "also *this* is a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isCalc, err := d.IsCalculation(tt.line)
			if err != nil {
				t.Fatalf("IsCalculation(%q) returned err=%v", tt.line, err)
			}
			if isCalc {
				t.Errorf("IsCalculation(%q) = true, want false (this line is prose)", tt.line)
			}
		})
	}
}

// `something *` (identifier followed by an operator) is the canonical
// shape that today's "structural" rule promotes to calculation. The
// user's clarification: this should only happen when `something` is a
// recognised name. At the line level, the only "recognised" leading
// identifiers are built-in function names and language keywords —
// everything else is prose.
//
// These cases isolate the rule without any markdown-emphasis cues so
// the markdown fix and the leading-name fix can be verified
// independently.
func TestUnknownLeadingIdentifierBeforeOperatorIsNotCalculation(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name string
		line string
	}{
		// User typed `something *` mid-sentence; nothing about
		// `something` says "calculation".
		{"unknown ident then multiply", "something * else"},
		{"unknown ident then plus", "something + else"},
		{"unknown ident then minus", "something - else"},
		// Looks structurally like an expression but no operand is a
		// known name.
		{"two unknown idents joined by op", "alpha * beta"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isCalc, err := d.IsCalculation(tt.line)
			if err != nil {
				t.Fatalf("IsCalculation(%q) returned err=%v", tt.line, err)
			}
			if isCalc {
				t.Errorf("IsCalculation(%q) = true, want false (leading identifier is not a known name)", tt.line)
			}
		})
	}
}

// Companion: lines that genuinely ARE calculations must keep classifying
// as calc, so the new heuristic doesn't over-correct. Each leading token
// satisfies the rule via a different leg:
//   - `=` follows         (assignment)
//   - leading is built-in (function or keyword)
//   - leading is literal  (number / currency / paren / unary / directive)
func TestGenuineCalculationsStayCalculation(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name string
		line string
	}{
		{"assignment", "x = 5"},
		{"assignment with expression", "tax_rate = price * 0.1"},
		{"number literal expression", "1 + 2"},
		{"built-in function call", "sqrt(2)"},
		{"date literal keyword", "today"},
		{"directive reference", "@globals.tax_rate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isCalc, err := d.IsCalculation(tt.line)
			if err != nil {
				t.Fatalf("IsCalculation(%q) returned err=%v", tt.line, err)
			}
			if !isCalc {
				t.Errorf("IsCalculation(%q) = false, want true (this line is a real calculation)", tt.line)
			}
		})
	}
}

// Registry-driven recognition: function-call shape is calc when (and
// only when) the leading identifier is registered in the feature
// registry. The detector does not hard-code names; it asks the
// registry, which is the single source of truth.
//
// Pinning these cases here means a future contributor who renames a
// function or removes one from the registry will see this test fail
// and think about the consequences for block classification, instead
// of silently flipping every doc that calls the removed name.
func TestRegistryDrivenFunctionCallRecognition(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name   string
		line   string
		isCalc bool
	}{
		// Registered function with no dedicated lexer token (lexes as
		// IDENT LPAREN). Must classify as calc so evaluation can run
		// and the user sees real diagnostics about the call.
		{"registered IDENT-callable", "accumulate(5mb, 1 hour)", true},
		// Built-in function with dedicated lexer token. Calc by the
		// dedicated-token path, regardless of the IDENT rule.
		{"built-in function with dedicated token", "sqrt(16)", true},
		// NL trigger keyword used as a function call.
		{"NL trigger as function call", "compound(1000, 5%, 10 years)", true},
		// Truly unknown name in function-call shape: cannot be
		// promoted on shape alone — registry doesn't know it, doc
		// hasn't defined it, so the line is prose to us. Any "did
		// you mean…?" diagnostic for unknown names is a separate
		// concern (parser/LSP), not a text/calc decision.
		{"unknown name in function-call shape", "frobnicate(x)", false},
		// Registered function name followed by an operator and a
		// number: lexes as IDENT MULT NUMBER. The `*` is multiply.
		{"registered IDENT name followed by operator", "accumulate * 2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.IsCalculation(tt.line)
			if err != nil {
				t.Fatalf("IsCalculation(%q) returned err=%v", tt.line, err)
			}
			if got != tt.isCalc {
				t.Errorf("IsCalculation(%q) = %v, want %v", tt.line, got, tt.isCalc)
			}
		})
	}
}

// Block-level integration test. A user typing prose into a fresh block
// should never see that block flip to a CalcBlock at any keystroke.
//
// Walks the prefix from 1 char up to the full string. Every prefix
// must remain a single TextBlock — not just the final string.
func TestTypingProseNeverFlipsBlockKind(t *testing.T) {
	d := NewDetector()
	full := "also *this* is a test"

	for i := 1; i <= len(full); i++ {
		prefix := full[:i]
		t.Run(prefix, func(t *testing.T) {
			blocks, err := d.DetectBlocks(prefix)
			if err != nil {
				t.Fatalf("DetectBlocks(%q) error = %v", prefix, err)
			}
			if len(blocks) != 1 {
				t.Fatalf("DetectBlocks(%q): expected 1 block, got %d", prefix, len(blocks))
			}
			if got := blocks[0].Type(); got != BlockText {
				t.Errorf("DetectBlocks(%q): block type = %v, want BlockText", prefix, got)
			}
		})
	}
}
