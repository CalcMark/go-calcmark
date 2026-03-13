package cmd

import (
	"fmt"
	"testing"
)

func TestParseEvalError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantTyp string
		wantMsg string
		wantLn  int
		wantCd  string
	}{
		{
			name:    "evaluation_error_with_line_and_code",
			err:     fmt.Errorf("evaluation error: line 2: variable_redefinition: cannot reassign 'x'"),
			wantTyp: "evaluation_error",
			wantMsg: "cannot reassign 'x'",
			wantLn:  2,
			wantCd:  "variable_redefinition",
		},
		{
			name:    "evaluation_error_with_extended_detail",
			err:     fmt.Errorf("evaluation error: line 1: undefined_variable: undefined variable \"y\" — did you mean 'x'?"),
			wantTyp: "evaluation_error",
			wantMsg: "undefined variable \"y\" — did you mean 'x'?",
			wantLn:  1,
			wantCd:  "undefined_variable",
		},
		{
			name:    "parse_error_plain",
			err:     fmt.Errorf("parse error: unexpected token"),
			wantTyp: "parse_error",
			wantMsg: "unexpected token",
			wantLn:  0,
			wantCd:  "",
		},
		{
			name:    "parse_error_with_line_and_code",
			err:     fmt.Errorf("parse error: line 5: syntax_error: unexpected ')'"),
			wantTyp: "parse_error",
			wantMsg: "unexpected ')'",
			wantLn:  5,
			wantCd:  "syntax_error",
		},
		{
			name:    "frontmatter_error",
			err:     fmt.Errorf("frontmatter error: invalid YAML"),
			wantTyp: "frontmatter_error",
			wantMsg: "invalid YAML",
			wantLn:  0,
			wantCd:  "",
		},
		{
			name:    "format_error",
			err:     fmt.Errorf("format error: template execution failed"),
			wantTyp: "format_error",
			wantMsg: "template execution failed",
			wantLn:  0,
			wantCd:  "",
		},
		{
			name:    "unknown_error",
			err:     fmt.Errorf("something unexpected happened"),
			wantTyp: "unknown_error",
			wantMsg: "something unexpected happened",
			wantLn:  0,
			wantCd:  "",
		},
		{
			name:    "read_file_error",
			err:     fmt.Errorf("read file: no such file or directory"),
			wantTyp: "unknown_error",
			wantMsg: "read file: no such file or directory",
			wantLn:  0,
			wantCd:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEvalError(tt.err)
			if got.Type != tt.wantTyp {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantTyp)
			}
			if got.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", got.Message, tt.wantMsg)
			}
			if got.Line != tt.wantLn {
				t.Errorf("Line = %d, want %d", got.Line, tt.wantLn)
			}
			if got.Code != tt.wantCd {
				t.Errorf("Code = %q, want %q", got.Code, tt.wantCd)
			}
		})
	}
}
