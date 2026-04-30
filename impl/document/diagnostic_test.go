package document

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/document"
	"github.com/CalcMark/go-calcmark/v2/spec/lexer"
)

func TestLooksLikeFailedCalculation(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantLikely bool
	}{
		// Should detect as likely failed calculation
		{"assignment with invalid comment", "x = 10 # comment", true},
		{"assignment with incomplete expression", "y = 5 +", true},
		{"assignment missing value", "z =", true},

		// Reserved keyword used as variable name
		{"reserved keyword end", "end = April 26", true},
		{"reserved keyword for", "for = 10", true},
		{"reserved keyword if", "if = 5", true},
		{"reserved keyword while", "while = true", true},
		{"reserved keyword let", "let = 100", true},

		// Should NOT detect as failed calculation (valid markdown/text)
		{"markdown heading", "# This is a heading", false},
		{"plain text", "This is just text", false},
		{"prose with equals", "two empty lines = hard boundary", false},
		{"bullet point", "- list item", false},
		{"empty line", "", false},
		{"whitespace only", "   ", false},

		// Should NOT detect (valid calculations that parse)
		{"valid assignment", "x = 10", false},
		{"valid expression", "a = 5 + 3", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLikely, _ := looksLikeFailedCalculation(tt.line)
			if gotLikely != tt.wantLikely {
				t.Errorf("looksLikeFailedCalculation(%q) = %v, want %v", tt.line, gotLikely, tt.wantLikely)
			}
		})
	}
}

// TestEvalErrorDiagnosticLine verifies that eval errors carry the correct
// block-relative line number. When line 2 of a calc block fails, the diagnostic
// must have Line=2, not Line=1.
// Regression test for misaligned diagnostics in the TUI results pane.
func TestEvalErrorDiagnosticLine(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantLine int    // Expected 1-indexed block-relative line
		wantCode string // Expected diagnostic code
		wantMsg  string // Substring of diagnostic message
	}{
		{
			name: "error on line 2 of calc block",
			source: `compound $1000 by 5% over 10
compound $1000 by 5% monthly over 10 ye
`,
			wantLine: 2,
			wantCode: "eval_error",
			wantMsg:  `invalid duration unit`,
		},
		{
			name: "error on line 1 of calc block",
			source: `compound $1000 by 5% monthly over 10 ye
compound $1000 by 5% over 10
`,
			wantLine: 1,
			wantCode: "eval_error",
			wantMsg:  `invalid duration unit`,
		},
		{
			name: "error on line 3 of calc block",
			source: `a = 1 + 1
b = a * 2
c = undefined_var + 1
`,
			wantLine: 3,
			wantCode: "undefined_variable",
			wantMsg:  `undefined_var`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := document.NewDocument(tt.source)
			if err != nil {
				t.Fatalf("NewDocument error: %v", err)
			}

			eval := NewEvaluator()
			_ = eval.Evaluate(doc) // errors expected

			// Check diagnostics on the calc block
			blocks := doc.GetBlocks()
			if len(blocks) == 0 {
				t.Fatal("expected at least one block")
			}

			cb, ok := blocks[0].Block.(*document.CalcBlock)
			if !ok {
				t.Fatal("expected first block to be CalcBlock")
			}

			diags := cb.Diagnostics()
			if len(diags) == 0 {
				t.Fatal("expected at least one diagnostic")
			}

			diag := diags[0]
			if diag.Line != tt.wantLine {
				t.Errorf("diagnostic.Line = %d, want %d", diag.Line, tt.wantLine)
			}
			if diag.Code != tt.wantCode {
				t.Errorf("diagnostic.Code = %q, want %q", diag.Code, tt.wantCode)
			}
			if !strings.Contains(diag.Message, tt.wantMsg) {
				t.Errorf("diagnostic.Message = %q, want substring %q", diag.Message, tt.wantMsg)
			}
		})
	}
}

// TestRedefinitionLineNumberWithFrontmatter verifies that the "first defined
// at line N" message in variable redefinition errors uses document-absolute
// line numbers, not block-relative. With 4 lines of frontmatter, block-relative
// line 1 becomes document line 5.
func TestRedefinitionLineNumberWithFrontmatter(t *testing.T) {
	source := "---\nexchange:\n  USD_EUR: 1.1\n---\nx = 10\nx = 20\n"
	// Frontmatter: lines 1-4 (---\nexchange:\n  USD_EUR: 1.1\n---\n)
	// x = 10 at document line 5
	// x = 20 at document line 6 (redefinition)

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument error: %v", err)
	}

	eval := NewEvaluator()
	_ = eval.Evaluate(doc) // redefinition error expected

	// Find the calc block
	var cb *document.CalcBlock
	for _, node := range doc.GetBlocks() {
		if b, ok := node.Block.(*document.CalcBlock); ok {
			cb = b
			break
		}
	}
	if cb == nil {
		t.Fatal("expected a calc block")
	}

	diags := cb.Diagnostics()
	if len(diags) == 0 {
		t.Fatal("expected a redefinition diagnostic")
	}

	msg := diags[0].Message
	// Should say "line 5" (document-absolute), not "line 1" (block-relative)
	if !strings.Contains(msg, "line 5") {
		t.Errorf("Expected 'line 5' (document-absolute) in message, got: %s", msg)
	}
	if strings.Contains(msg, "line 1") {
		t.Errorf("Should NOT contain block-relative 'line 1', got: %s", msg)
	}

	// DocLine should carry the document-absolute line of the error itself (line 6)
	if diags[0].DocLine != 6 {
		t.Errorf("DocLine = %d, want 6 (document-absolute line of redefinition)", diags[0].DocLine)
	}
	// Line should still be block-relative for internal use
	if diags[0].Line != 2 {
		t.Errorf("Line = %d, want 2 (block-relative)", diags[0].Line)
	}
}

// TestRedefinitionLineNumberWithoutFrontmatter verifies that without frontmatter,
// the line number in the redefinition message is the plain document line.
func TestRedefinitionLineNumberWithoutFrontmatter(t *testing.T) {
	source := "x = 10\nx = 20\n"
	// x = 10 at document line 1
	// x = 20 at document line 2 (redefinition)

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument error: %v", err)
	}

	eval := NewEvaluator()
	_ = eval.Evaluate(doc)

	var cb *document.CalcBlock
	for _, node := range doc.GetBlocks() {
		if b, ok := node.Block.(*document.CalcBlock); ok {
			cb = b
			break
		}
	}
	if cb == nil {
		t.Fatal("expected a calc block")
	}

	diags := cb.Diagnostics()
	if len(diags) == 0 {
		t.Fatal("expected a redefinition diagnostic")
	}

	msg := diags[0].Message
	if !strings.Contains(msg, "line 1") {
		t.Errorf("Expected 'line 1' in message, got: %s", msg)
	}
}

func TestEvaluatorDiagnostics(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		wantDiagCount  int
		wantDiagCode   string
		wantDiagInLine string // substring to match in diagnostic source
	}{
		{
			name: "detects failed assignment in text block",
			source: `# Header

x = 10 # this looks like a calculation

More text here.
`,
			wantDiagCount:  1,
			wantDiagCode:   DiagLikelyCalculation,
			wantDiagInLine: "x = 10 #",
		},
		{
			name: "no diagnostics for valid document",
			source: `# Header

x = 10

Some text.
`,
			wantDiagCount: 0,
		},
		{
			name: "no diagnostics for pure markdown",
			source: `# Header

This is just text with = signs like two = two.

- List item
- Another item
`,
			wantDiagCount: 0,
		},
		{
			name: "detects multiple failed assignments",
			source: `# Test

a = #
b = # also broken
`,
			wantDiagCount: 2,
			wantDiagCode:  DiagLikelyCalculation,
		},
		{
			name: "detects reserved keyword as variable name",
			source: `start = Apr 22
end = April 26
`,
			wantDiagCount:  1,
			wantDiagCode:   DiagLikelyCalculation,
			wantDiagInLine: "end = April 26",
		},
		{
			name:           "reserved keyword diagnostic has helpful message",
			source:         "end = April 26\n",
			wantDiagCount:  1,
			wantDiagCode:   DiagLikelyCalculation,
			wantDiagInLine: "end = April 26",
		},
		{
			name:           "today keyword warns when used as variable",
			source:         "today = Mar 3 2021\n",
			wantDiagCount:  1,
			wantDiagCode:   DiagLikelyCalculation,
			wantDiagInLine: "today = Mar 3 2021",
		},
		{
			name:           "tomorrow keyword warns when used as variable",
			source:         "tomorrow = Mar 4 2021\n",
			wantDiagCount:  1,
			wantDiagCode:   DiagLikelyCalculation,
			wantDiagInLine: "tomorrow = Mar 4 2021",
		},
		{
			name:           "yesterday keyword warns when used as variable",
			source:         "yesterday = Mar 2 2021\n",
			wantDiagCount:  1,
			wantDiagCode:   DiagLikelyCalculation,
			wantDiagInLine: "yesterday = Mar 2 2021",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := document.NewDocument(tt.source)
			if err != nil {
				t.Fatalf("NewDocument error: %v", err)
			}

			evaluator := NewEvaluator()
			_ = evaluator.Evaluate(doc) // Ignore eval errors for this test

			diags := evaluator.Diagnostics()

			if len(diags) != tt.wantDiagCount {
				t.Errorf("got %d diagnostics, want %d", len(diags), tt.wantDiagCount)
				for i, d := range diags {
					t.Logf("  diag %d: %s: %s (source: %q)", i, d.Code, d.Message, d.Source)
				}
				return
			}

			if tt.wantDiagCount > 0 && tt.wantDiagCode != "" {
				if diags[0].Code != tt.wantDiagCode {
					t.Errorf("diagnostic code = %q, want %q", diags[0].Code, tt.wantDiagCode)
				}
			}

			if tt.wantDiagInLine != "" && tt.wantDiagCount > 0 {
				if !strings.Contains(diags[0].Source, tt.wantDiagInLine) {
					t.Errorf("diagnostic source %q doesn't contain %q", diags[0].Source, tt.wantDiagInLine)
				}
			}
		})
	}
}

func TestStartsLikeAssignment(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"x = 10", true},
		{"myVar = something", true},
		{"_private = value", true},
		{"X = Y", true},

		{"= something", false},       // no identifier before =
		{"two words = value", false}, // space in identifier part
		{"", false},
		{"no equals here", false},
		{"123 = value", false}, // starts with digit
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := startsLikeAssignment(tt.input)
			if got != tt.want {
				t.Errorf("startsLikeAssignment(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCalculationIndicators(t *testing.T) {
	// Verify the indicator list is not empty and properly documented
	indicators := GetCalculationIndicators()
	if len(indicators) == 0 {
		t.Fatal("expected at least one calculation indicator")
	}

	for _, ind := range indicators {
		if ind.Name == "" {
			t.Error("indicator has empty name")
		}
		if ind.Description == "" {
			t.Errorf("indicator %q has empty description", ind.Name)
		}
		if ind.Check == nil {
			t.Errorf("indicator %q has nil Check function", ind.Name)
		}
	}
}

func TestAssignmentIndicator(t *testing.T) {
	// Test the assignment indicator specifically
	indicators := GetCalculationIndicators()

	var assignmentIndicator *CalculationIndicator
	for i := range indicators {
		if indicators[i].Name == "assignment" {
			assignmentIndicator = &indicators[i]
			break
		}
	}

	if assignmentIndicator == nil {
		t.Fatal("assignment indicator not found")
	}

	tests := []struct {
		name string
		line string
		want bool
	}{
		// Should match assignment indicator
		{"simple assignment", "x = 10", true},
		{"assignment with unit", "distance = 100 meters", true},
		{"assignment incomplete", "y =", true},

		// Should NOT match assignment indicator
		{"plain number", "42", false},
		{"expression", "5 + 3", false},
		{"markdown heading", "# Title", false},
		// Note: "two = two is math" DOES match the indicator (identifier = pattern)
		// but looksLikeFailedCalculation won't flag it because it parses successfully
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lex := lexer.NewLexer(tt.line)
			tokens, err := lex.Tokenize()
			if err != nil {
				// Can't tokenize - indicator check would be skipped
				t.Skipf("line %q failed to tokenize: %v", tt.line, err)
			}

			meaningful := filterMeaningful(tokens)
			got := assignmentIndicator.Check(meaningful)
			if got != tt.want {
				t.Errorf("assignment indicator for %q = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestReservedKeywordIndicator(t *testing.T) {
	indicators := GetCalculationIndicators()

	var indicator *CalculationIndicator
	for i := range indicators {
		if indicators[i].Name == "reserved_keyword_assignment" {
			indicator = &indicators[i]
			break
		}
	}

	if indicator == nil {
		t.Fatal("reserved_keyword_assignment indicator not found")
	}

	tests := []struct {
		name string
		line string
		want bool
	}{
		{"end assignment", "end = April 26", true},
		{"for assignment", "for = 10", true},
		{"if assignment", "if = 5", true},
		{"let assignment", "let = 100", true},

		// Should NOT match
		{"normal identifier", "x = 10", false},
		{"keyword without assign", "end", false},
		{"plain number", "42", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lex := lexer.NewLexer(tt.line)
			tokens, err := lex.Tokenize()
			if err != nil {
				t.Skipf("line %q failed to tokenize: %v", tt.line, err)
			}

			meaningful := filterMeaningful(tokens)
			got := indicator.Check(meaningful)
			if got != tt.want {
				t.Errorf("reserved_keyword_assignment indicator for %q = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestReservedKeywordDiagnosticMessage(t *testing.T) {
	source := "end = April 26\n"

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument error: %v", err)
	}

	evaluator := NewEvaluator()
	_ = evaluator.Evaluate(doc)

	// Check evaluator-level diagnostics
	evalDiags := evaluator.Diagnostics()
	if len(evalDiags) != 1 {
		t.Fatalf("got %d evaluator diagnostics, want 1", len(evalDiags))
	}

	msg := evalDiags[0].Message
	if !strings.Contains(msg, "reserved keyword") {
		t.Errorf("diagnostic message should mention 'reserved keyword', got: %s", msg)
	}
	if !strings.Contains(msg, `"end"`) {
		t.Errorf("diagnostic message should mention the keyword name, got: %s", msg)
	}

	// Check TextBlock-level diagnostics (used by TUI footer)
	var tb *document.TextBlock
	for _, node := range doc.GetBlocks() {
		if b, ok := node.Block.(*document.TextBlock); ok {
			tb = b
			break
		}
	}
	if tb == nil {
		t.Fatal("expected a text block")
	}
	blockDiags := tb.Diagnostics()
	if len(blockDiags) != 1 {
		t.Fatalf("got %d block diagnostics, want 1", len(blockDiags))
	}
	if blockDiags[0].Detailed == "" {
		t.Error("diagnostic Detailed field should not be empty")
	}
	if !strings.Contains(blockDiags[0].Detailed, "end_val") {
		t.Errorf("diagnostic hint should suggest an alternative name, got: %s", blockDiags[0].Detailed)
	}
}

func TestDateKeywordDiagnosticHints(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		keyword  string
		wantHint string
	}{
		{"today suggests start_date", "today = Mar 3 2021\n", "today", "start_date"},
		{"tomorrow suggests next_day", "tomorrow = Mar 4 2021\n", "tomorrow", "next_day"},
		{"yesterday suggests prev_day", "yesterday = Mar 2 2021\n", "yesterday", "prev_day"},
		{"end still suggests end_val", "end = April 26\n", "end", "end_val"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := document.NewDocument(tt.source)
			if err != nil {
				t.Fatalf("NewDocument error: %v", err)
			}
			evaluator := NewEvaluator()
			_ = evaluator.Evaluate(doc)

			diags := evaluator.Diagnostics()
			if len(diags) != 1 {
				t.Fatalf("got %d diagnostics, want 1", len(diags))
			}
			if !strings.Contains(diags[0].Message, "reserved keyword") {
				t.Errorf("message should mention 'reserved keyword', got: %s", diags[0].Message)
			}
			if !strings.Contains(diags[0].Message, tt.keyword) {
				t.Errorf("message should mention %q, got: %s", tt.keyword, diags[0].Message)
			}

			// Check the hint in the text block diagnostic
			var tb *document.TextBlock
			for _, node := range doc.GetBlocks() {
				if b, ok := node.Block.(*document.TextBlock); ok {
					tb = b
					break
				}
			}
			if tb == nil {
				t.Fatal("expected a text block")
			}
			blockDiags := tb.Diagnostics()
			if len(blockDiags) != 1 {
				t.Fatalf("got %d block diagnostics, want 1", len(blockDiags))
			}
			if !strings.Contains(blockDiags[0].Detailed, tt.wantHint) {
				t.Errorf("hint should suggest %q, got: %s", tt.wantHint, blockDiags[0].Detailed)
			}
		})
	}
}

func TestIndicatorTriggersWarning(t *testing.T) {
	// Test that each indicator actually triggers warnings when appropriate
	testCases := []struct {
		name          string
		indicatorName string
		failingLine   string // Line that matches indicator but fails to parse
		validLine     string // Line that matches indicator and parses OK
	}{
		{
			name:          "assignment indicator",
			indicatorName: "assignment",
			failingLine:   "x = 10 # broken",
			validLine:     "x = 10",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test failing line triggers detection
			likely, err := looksLikeFailedCalculation(tc.failingLine)
			if !likely {
				t.Errorf("failing line %q should be detected as likely calculation", tc.failingLine)
			}
			if err == nil {
				t.Errorf("failing line %q should have parse error", tc.failingLine)
			}

			// Test valid line does NOT trigger
			likely, _ = looksLikeFailedCalculation(tc.validLine)
			if likely {
				t.Errorf("valid line %q should NOT be detected as failed calculation", tc.validLine)
			}
		})
	}
}
