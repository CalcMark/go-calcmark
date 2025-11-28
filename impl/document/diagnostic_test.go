package document

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/CalcMark/go-calcmark/spec/lexer"
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
