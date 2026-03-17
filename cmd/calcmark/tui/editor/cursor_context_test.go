package editor

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/types"
)

func TestGetCursorContext(t *testing.T) {
	tests := []struct {
		name          string
		line          string
		cursorCol     int
		wantInFunc    bool
		wantFuncName  string
		wantArgIndex  int
		wantParamName string
		wantParamType types.ArgType
	}{
		{
			name:       "not in function call",
			line:       "x = 10",
			cursorCol:  4,
			wantInFunc: false,
		},
		{
			name:          "inside accumulate first arg",
			line:          "accumulate(",
			cursorCol:     11,
			wantInFunc:    true,
			wantFuncName:  "accumulate",
			wantArgIndex:  0,
			wantParamName: "rate",
			wantParamType: types.ArgTypeRate,
		},
		{
			name:          "inside accumulate second arg",
			line:          "accumulate(10 MB/s, ",
			cursorCol:     20,
			wantInFunc:    true,
			wantFuncName:  "accumulate",
			wantArgIndex:  1,
			wantParamName: "duration",
			wantParamType: types.ArgTypeDuration,
		},
		{
			name:          "inside avg variadic",
			line:          "avg(1, 2, ",
			cursorCol:     10,
			wantInFunc:    true,
			wantFuncName:  "avg",
			wantArgIndex:  2,
			wantParamName: "values",
			wantParamType: types.ArgTypeAny,
		},
		{
			name:          "nested function - inner",
			line:          "avg(sqrt(",
			cursorCol:     9,
			wantInFunc:    true,
			wantFuncName:  "sqrt",
			wantArgIndex:  0,
			wantParamName: "value",
			wantParamType: types.ArgTypeNumber,
		},
		{
			name:       "after closed paren - not in func",
			line:       "avg(1, 2) + ",
			cursorCol:  12,
			wantInFunc: false,
		},
		{
			name:          "downtime with percentage",
			line:          "downtime(99.9%, ",
			cursorCol:     16,
			wantInFunc:    true,
			wantFuncName:  "downtime",
			wantArgIndex:  1,
			wantParamName: "duration",
			wantParamType: types.ArgTypeDuration,
		},
		{
			name:         "unknown function",
			line:         "unknown_func(",
			cursorCol:    13,
			wantInFunc:   true,
			wantFuncName: "unknown_func",
			// No param info for unknown functions
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := GetCursorContext(tt.line, tt.cursorCol)

			if ctx.InFunctionCall != tt.wantInFunc {
				t.Errorf("InFunctionCall = %v, want %v", ctx.InFunctionCall, tt.wantInFunc)
			}

			if tt.wantInFunc {
				if ctx.FunctionName != tt.wantFuncName {
					t.Errorf("FunctionName = %q, want %q", ctx.FunctionName, tt.wantFuncName)
				}

				if ctx.ArgIndex != tt.wantArgIndex {
					t.Errorf("ArgIndex = %d, want %d", ctx.ArgIndex, tt.wantArgIndex)
				}

				if tt.wantParamName != "" {
					if ctx.ParamSpec == nil {
						t.Errorf("ParamSpec is nil, want param %q", tt.wantParamName)
					} else {
						if ctx.ParamSpec.Name != tt.wantParamName {
							t.Errorf("ParamSpec.Name = %q, want %q", ctx.ParamSpec.Name, tt.wantParamName)
						}
						if ctx.ParamSpec.Type != tt.wantParamType {
							t.Errorf("ParamSpec.Type = %q, want %q", ctx.ParamSpec.Type, tt.wantParamType)
						}
					}
				}
			}
		})
	}
}

func TestFormatParamHelp(t *testing.T) {
	spec := types.GetFunctionSpec("accumulate")
	if spec == nil {
		t.Fatal("accumulate spec not found")
	}

	param := spec.GetParamAtIndex(0) // rate
	help := FormatParamHelp(param)

	if help == "" {
		t.Error("FormatParamHelp returned empty string")
	}

	// Should contain parameter name
	if !contains(help, "rate") {
		t.Errorf("Help should contain param name 'rate': %s", help)
	}

	// Should contain examples
	if !contains(help, "MB/s") {
		t.Errorf("Help should contain example 'MB/s': %s", help)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
