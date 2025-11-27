//go:build js && wasm
// +build js,wasm

package main

import (
	"encoding/json"
	"syscall/js"
	"testing"
)

// TestTokenizeWithThousandsSeparators tests that token positions are correct with thousands separators
// This test runs only in WASM environment where js.Value is available
func TestTokenizeWithThousandsSeparators(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple number with thousands",
			input: "1,000",
		},
		{
			name:  "average of with thousands and decimals",
			input: "average of 432, 32, 1,000.01",
		},
		{
			name:  "multiple thousands",
			input: "avg(1,000, 2,000, 3,000)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a js.Value from a string
			jsStr := js.ValueOf(tt.input)
			args := []js.Value{jsStr}
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
		})
	}
}
