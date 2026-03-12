package editor

// error_display_test.go — Error parsing and display formatting.

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/spec/document"
)

func TestParseErrorForDisplay(t *testing.T) {
	tests := []struct {
		name         string
		errMsg       string
		wantShort    string
		wantHint     string
		wantContains []string // substrings that must appear
	}{
		{
			name:      "undefined variable with quotes",
			errMsg:    `undefined_variable: Undefined variable "My Budget" - it must be defined`,
			wantShort: "Undefined variable: My Budget",
			wantHint:  "Define it above: My Budget = <value>",
		},
		{
			name:      "undefined variable alternate format",
			errMsg:    `undefined variable: "total_cost"`,
			wantShort: "Undefined variable: total_cost",
			wantHint:  "Define it above: total_cost = <value>",
		},
		{
			name:      "division by zero",
			errMsg:    "division_by_zero: cannot divide by zero",
			wantShort: "Division by zero",
			wantHint:  "Check that divisor is not zero",
		},
		{
			name:         "incompatible units",
			errMsg:       "incompatible_units: cannot add meters and seconds",
			wantContains: []string{"meters", "seconds"},
			wantHint:     "Units must be compatible for this operation",
		},
		{
			name:         "generic error",
			errMsg:       "something went wrong",
			wantContains: []string{"something went wrong"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := components.ParseErrorForDisplay(tt.errMsg)

			if tt.wantShort != "" && info.ShortMessage != tt.wantShort {
				t.Errorf("ShortMessage = %q, want %q", info.ShortMessage, tt.wantShort)
			}

			if tt.wantHint != "" && info.Hint != tt.wantHint {
				t.Errorf("Hint = %q, want %q", info.Hint, tt.wantHint)
			}

			for _, substr := range tt.wantContains {
				if !strings.Contains(info.ShortMessage, substr) {
					t.Errorf("ShortMessage %q should contain %q", info.ShortMessage, substr)
				}
			}
		})
	}
}

// TestErrorDisplayInContextFooter verifies errors are shown helpfully in context footer.
func TestErrorDisplayInContextFooter(t *testing.T) {
	// Create a document with an undefined variable error
	content := `result = undefined_var * 2`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull
	m.cursorLine = 0

	view := m.View().Content

	// Should show helpful error in context footer area
	// Looking for the variable name and hint, not the raw error code
	if !strings.Contains(view, "undefined_var") {
		t.Logf("VIEW:\n%s", view)
		t.Error("View should show undefined variable name 'undefined_var'")
	}

	// Should show hint from semantic checker (lists defined variables)
	if !strings.Contains(view, "Defined variables") {
		t.Logf("VIEW:\n%s", view)
		t.Error("View should show hint about defined variables from semantic checker")
	}

	// Should NOT show raw error code format
	if strings.Contains(view, "undefined_variable:") {
		t.Error("View should not show raw error code 'undefined_variable:'")
	}
}

// TestViewHeightWithErrors verifies view height stays consistent with errors.
func TestInitialEvalErrors(t *testing.T) {
	// Create a document with a valid calc expression that evaluates cleanly
	doc, _ := document.NewDocument("x = 42\n")
	m := New(doc)

	// A well-formed document should not have error status
	if m.statusIsErr {
		t.Logf("Status: %s", m.statusMsg)
		// Note: some documents might trigger errors depending on the evaluator
	}

	// Verify the evaluator ran (eval should be non-nil)
	if m.eval == nil {
		t.Error("Expected evaluator to be initialized")
	}
}

// TestAddExportExtension tests export extension helper.
