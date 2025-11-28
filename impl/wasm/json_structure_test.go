//go:build !wasm
// +build !wasm

package main

import (
	"encoding/json"
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/impl/types"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// evaluateSourceNonWasm is a helper function to evaluate source code and return results.
// This is the non-WASM version for testing JSON output.
func evaluateSourceNonWasm(source string, env *interpreter.Environment) ([]types.Type, error) {
	nodes, err := parser.Parse(source)
	if err != nil {
		return nil, err
	}
	interp := interpreter.NewInterpreterWithEnv(env)
	return interp.Eval(nodes)
}

// TestEvaluateJSONStructure tests what the JSON output looks like when we marshal evaluation results
func TestEvaluateJSONStructure(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]any // Expected JSON structure
	}{
		{
			name:  "number result",
			input: "x = 5",
			expected: map[string]any{
				"Value":        map[string]any{}, // decimal.Decimal structure
				"SourceFormat": "",
			},
		},
		{
			name:  "currency result",
			input: "budget = $1000",
			expected: map[string]any{
				"Value":        map[string]any{}, // decimal.Decimal structure
				"Symbol":       "$",
				"SourceFormat": "$1000",
			},
		},
		{
			name:  "boolean result",
			input: "flag = true",
			expected: map[string]any{
				"Value": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := interpreter.NewEnvironment()
			results, err := evaluateSourceNonWasm(tt.input, ctx)
			if err != nil {
				t.Fatalf("Evaluate failed: %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			// Marshal to JSON
			jsonBytes, err := json.Marshal(results)
			if err != nil {
				t.Fatalf("JSON marshal failed: %v", err)
			}

			t.Logf("Input: %s", tt.input)
			t.Logf("Result type: %T", results[0])
			t.Logf("JSON: %s", string(jsonBytes))

			// Unmarshal to see structure
			var unmarshaled []map[string]any
			err = json.Unmarshal(jsonBytes, &unmarshaled)
			if err != nil {
				t.Fatalf("JSON unmarshal failed: %v", err)
			}

			if len(unmarshaled) != 1 {
				t.Fatalf("Expected 1 unmarshaled result, got %d", len(unmarshaled))
			}

			result := unmarshaled[0]
			t.Logf("Unmarshaled keys: %v", getKeysNonWasm(result))
			t.Logf("Unmarshaled structure: %+v", result)

			// Check that we have the expected keys
			for key := range tt.expected {
				if _, exists := result[key]; !exists {
					t.Errorf("Expected key %q not found in result", key)
				}
			}
		})
	}
}

// TestActualResultStructure directly tests what we get from types
func TestActualResultStructure(t *testing.T) {
	t.Run("Number JSON", func(t *testing.T) {
		num, _ := types.NewNumberWithFormat("5", "5")
		jsonBytes, _ := json.Marshal(num)
		t.Logf("Number JSON: %s", string(jsonBytes))

		var result map[string]any
		json.Unmarshal(jsonBytes, &result)
		t.Logf("Number keys: %v", getKeysNonWasm(result))
		t.Logf("Number structure: %+v", result)
	})

	t.Run("Currency JSON", func(t *testing.T) {
		curr, _ := types.NewCurrencyWithFormat("1000", "$", "$1000")
		jsonBytes, _ := json.Marshal(curr)
		t.Logf("Currency JSON: %s", string(jsonBytes))

		var result map[string]any
		json.Unmarshal(jsonBytes, &result)
		t.Logf("Currency keys: %v", getKeysNonWasm(result))
		t.Logf("Currency structure: %+v", result)
	})

	t.Run("Boolean JSON", func(t *testing.T) {
		boolean, _ := types.NewBoolean(true)
		jsonBytes, _ := json.Marshal(boolean)
		t.Logf("Boolean JSON: %s", string(jsonBytes))

		var result map[string]any
		json.Unmarshal(jsonBytes, &result)
		t.Logf("Boolean keys: %v", getKeysNonWasm(result))
		t.Logf("Boolean structure: %+v", result)
	})
}

// TestWASMEvaluateOutput tests the actual WASM evaluate function output
func TestWASMEvaluateOutput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"number", "x = 5"},
		{"currency", "budget = $1000"},
		{"boolean", "flag = true"},
		{"number with format", "amount = 1,000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := interpreter.NewEnvironment()
			results, err := evaluateSourceNonWasm(tt.input, ctx)
			if err != nil {
				t.Fatalf("Evaluate failed: %v", err)
			}

			// This is what the WASM code does
			jsonBytes, jsonErr := json.Marshal(results)
			if jsonErr != nil {
				t.Fatalf("JSON marshal failed: %v", jsonErr)
			}

			t.Logf("\nInput: %s", tt.input)
			t.Logf("JSON output: %s", string(jsonBytes))

			// Parse it back
			var parsed []any
			json.Unmarshal(jsonBytes, &parsed)
			if len(parsed) > 0 {
				t.Logf("Parsed result type: %T", parsed[0])
				if m, ok := parsed[0].(map[string]any); ok {
					t.Logf("Available keys: %v", getKeysNonWasm(m))
					for key, val := range m {
						t.Logf("  %s = %v (type: %T)", key, val, val)
					}
				}
			}
		})
	}
}

func getKeysNonWasm(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
