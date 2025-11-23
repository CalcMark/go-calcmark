package document

import (
	"testing"
)

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "LF (Unix)",
			input:    "line1\nline2\nline3",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "CRLF (Windows)",
			input:    "line1\r\nline2\r\nline3",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "CR (Old Mac)",
			input:    "line1\rline2\rline3",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "Mixed line endings",
			input:    "line1\nline2\r\nline3\rline4",
			expected: []string{"line1", "line2", "line3", "line4"},
		},
		{
			name:     "Unicode line separator",
			input:    "line1\u2028line2",
			expected: []string{"line1", "line2"},
		},
		{
			name:     "Unicode paragraph separator",
			input:    "para1\u2029para2",
			expected: []string{"para1", "para2"},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "No newlines",
			input:    "single line",
			expected: []string{"single line"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitLines(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d lines, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Line %d: expected %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

func TestIsEmptyLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty string", "", true},
		{"spaces", "   ", true},
		{"tabs", "\t\t", true},
		{"mixed whitespace", " \t ", true},
		{"Unicode whitespace", "\u00A0\u2000\u2009", true}, // NBSP, various spaces
		{"text", "hello", false},
		{"text with spaces", "  hello  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEmptyLine(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
