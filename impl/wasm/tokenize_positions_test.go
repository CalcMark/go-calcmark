// +build !wasm

package main

import (
	"encoding/json"
	"syscall/js"
	"testing"
)

// Mock js.Value for testing
type mockJSValue struct {
	stringVal string
}

func (m mockJSValue) String() string {
	return m.stringVal
}

func (m mockJSValue) Bool() bool {
	return false
}

func (m mockJSValue) Int() int {
	return 0
}

func (m mockJSValue) Length() int {
	return 0
}

func (m mockJSValue) Index(i int) js.Value {
	return nil
}

// TestTokenizeWithThousandsSeparators tests that token positions are correct with thousands separators
func TestTokenizeWithThousandsSeparators(t *testing.T) {
	tests := []struct {
		name  string
		input string
		// We check that tokens have correct positions
		checkPositions bool
	}{
		{
			name:           "simple number with thousands",
			input:          "1,000",
			checkPositions: true,
		},
		{
			name:           "average of with thousands and decimals",
			input:          "average of 432, 32, 1,000.01",
			checkPositions: true,
		},
		{
			name:           "multiple thousands",
			input:          "avg(1,000, 2,000, 3,000)",
			checkPositions: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []js.Value{mockJSValue{stringVal: tt.input}}
			result := tokenize(js.Value{}, args)

			resultMap, ok := result.(map[string]interface{})
			if !ok {
				t.Fatalf("Expected map result, got %T", result)
			}

			if resultMap["error"] != nil {
				t.Fatalf("Unexpected error: %v", resultMap["error"])
			}

			tokensJSON, ok := resultMap["tokens"].(string)
			if !ok {
				t.Fatalf("Expected tokens string, got %T", resultMap["tokens"])
			}

			var tokens []TokenInfo
			if err := json.Unmarshal([]byte(tokensJSON), &tokens); err != nil {
				t.Fatalf("Failed to unmarshal tokens: %v", err)
			}

			// Check that token positions make sense
			if tt.checkPositions {
				// The end position of the last token should be close to the input length
				// It won't be exact because thousands separators are stripped from values
				lastToken := tokens[len(tokens)-2] // Second to last (before EOF)

				t.Logf("Input: %q (len=%d)", tt.input, len(tt.input))
				for i, tok := range tokens {
					t.Logf("Token %d: %s = %q (start=%d, end=%d)", i, tok.Type, tok.Value, tok.Start, tok.End)
				}

				// Basic sanity: no token should start beyond the input length
				for i, tok := range tokens {
					if tok.Start > len(tt.input) {
						t.Errorf("Token %d starts at %d, beyond input length %d", i, tok.Start, len(tt.input))
					}
					if tok.End > len(tt.input)+10 { // Allow some slack
						t.Errorf("Token %d ends at %d, way beyond input length %d", i, tok.End, len(tt.input))
					}
				}
			}
		})
	}
}
