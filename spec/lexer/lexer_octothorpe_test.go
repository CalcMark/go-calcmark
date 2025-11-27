package lexer

import (
	"strings"
	"testing"
)

// TestOctothorpeError tests that # mid-line produces a clear error
func TestOctothorpeError(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "octothorpe after assignment",
			input: "x = 10 # comment",
		},
		{
			name:  "octothorpe after expression",
			input: "100 + 50 # note",
		},
		{
			name:  "octothorpe with napkin annotation",
			input: "small_num = 47 as napkin # → ~47",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			_, err := lexer.Tokenize()

			if err == nil {
				t.Errorf("Expected error for input %q, got nil", tt.input)
				return
			}

			// Check that error message mentions octothorpe
			errMsg := err.Error()
			if !strings.Contains(errMsg, "octothorpe") && !strings.Contains(errMsg, "#") {
				t.Errorf("Expected error to mention octothorpe or #, got: %s", errMsg)
			}

			// Check that error mentions it's not supported inline
			if !strings.Contains(errMsg, "not supported") {
				t.Errorf("Expected error to say not supported, got: %s", errMsg)
			}

			// Check that error mentions markdown heading alternative
			if !strings.Contains(errMsg, "Markdown heading") {
				t.Errorf("Expected error to mention Markdown heading alternative, got: %s", errMsg)
			}
		})
	}
}

// TestLeadingOctothorpeAllowed verifies that # at start of line is OK (markdown)
func TestLeadingOctothorpeAllowed(t *testing.T) {
	// NOTE: The lexer doesn't see leading # because the document detector
	// splits text blocks from calc blocks before lexing individual lines.
	// This test documents that behavior.

	// If we try to lex "# Heading" as a calculation, it would error
	// But the detector won't send this to the lexer - it's a text block
	input := "# This is a markdown heading"
	lexer := NewLexer(input)
	_, err := lexer.Tokenize()

	// Should error because # is not valid in a calculation
	if err == nil {
		t.Error("Lexer should error on # even at start (detector prevents this case)")
	}
}

// TestValidIdentifiersAndUnits ensures we don't break existing functionality
func TestValidIdentifiersAndUnits(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "custom unit",
			input: "x = 10 apples",
		},
		{
			name:  "emoji identifier",
			input: "🍎 = 5",
		},
		{
			name:  "unicode identifier",
			input: "café = 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			_, err := lexer.Tokenize()

			if err != nil {
				t.Errorf("Valid input %q should not error, got: %v", tt.input, err)
			}
		})
	}
}
