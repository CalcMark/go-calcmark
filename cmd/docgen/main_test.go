package main

import (
	"encoding/json"
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/features"
)

func TestKebabCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"avg", "avg"},
		{"transfer_time", "transfer-time"},
		{"as napkin", "as-napkin"},
		{"+", ""},
		{"convert_rate", "convert-rate"},
		{"  spaces  ", "spaces"},
	}
	for _, tt := range tests {
		got := kebabCase(tt.input)
		if got != tt.want {
			t.Errorf("kebabCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGenerateJSON(t *testing.T) {
	reg := features.NewRegistry()

	output := make(JSONOutput)
	for _, cat := range reg.Categories() {
		feats := reg.ByCategory(cat)
		jsonFeats := make([]JSONFeature, 0, len(feats))
		for _, f := range feats {
			jf := JSONFeature{
				Name:        f.Name,
				Category:    string(f.Category),
				Syntax:      f.Syntax,
				Description: f.Description,
				Example:     f.Example,
				Anchor:      kebabCase(f.Name),
			}
			for _, a := range f.Aliases {
				jf.Aliases = append(jf.Aliases, JSONAlias{
					Name:      a.Name,
					Parseable: a.Parseable,
				})
			}
			jsonFeats = append(jsonFeats, jf)
		}
		output[string(cat)] = jsonFeats
	}

	// Verify JSON is valid
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Verify it can round-trip
	var parsed JSONOutput
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify expected categories exist
	expectedCats := []string{"function", "keyword", "operator", "unit", "date", "network", "storage", "compression"}
	for _, cat := range expectedCats {
		if _, ok := parsed[cat]; !ok {
			t.Errorf("Missing expected category %q", cat)
		}
	}

	// Verify functions category has all expected functions
	functions := parsed["function"]
	expectedFuncs := []string{"avg", "sqrt", "accumulate", "convert_rate", "capacity", "downtime", "rtt", "throughput", "transfer_time", "read", "seek", "compress"}
	funcNames := make(map[string]bool)
	for _, f := range functions {
		funcNames[f.Name] = true
	}
	for _, name := range expectedFuncs {
		if !funcNames[name] {
			t.Errorf("Missing expected function %q in JSON output", name)
		}
	}

	// Verify every function has an anchor
	for _, f := range functions {
		if f.Anchor == "" {
			t.Errorf("Function %q has empty anchor", f.Name)
		}
	}

	// Verify keywords include over, as napkin, at
	keywords := parsed["keyword"]
	keywordNames := make(map[string]bool)
	for _, k := range keywords {
		keywordNames[k.Name] = true
	}
	for _, name := range []string{"over", "as napkin", "at"} {
		if !keywordNames[name] {
			t.Errorf("Missing expected keyword %q in JSON output", name)
		}
	}

	t.Logf("Generated %d bytes, %d categories", len(data), len(parsed))
	for cat, feats := range parsed {
		t.Logf("  %s: %d features", cat, len(feats))
	}
}

// TestCrossRegistryCompleteness verifies that every function in the interpreter's
// BuiltinFunctions has a corresponding entry in the features registry.
// This prevents drift between the two registries.
func TestCrossRegistryCompleteness(t *testing.T) {
	reg := features.NewRegistry()
	registryFuncs := reg.ByCategory(features.CategoryFunction)

	// Build a set of registry function names (including aliases)
	registryNames := make(map[string]bool)
	for _, f := range registryFuncs {
		registryNames[f.Name] = true
		for _, a := range f.Aliases {
			registryNames[a.Name] = true
		}
	}

	// Check every interpreter function exists in the registry
	for _, fn := range interpreter.BuiltinFunctions {
		if !registryNames[fn.Name] {
			t.Errorf("Interpreter function %q has no entry in features registry", fn.Name)
		}
	}
}
