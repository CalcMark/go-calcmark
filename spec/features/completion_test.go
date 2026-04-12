package features

import (
	"testing"
)

func TestFunctionSuggestions(t *testing.T) {
	// Build implementedNames from known function names
	implementedNames := map[string]bool{
		"avg": true, "sum": true, "sqrt": true, "number": true,
		"accumulate": true, "convert_rate": true, "downtime": true,
		"rtt": true, "throughput": true, "transfer_time": true,
		"read": true, "seek": true, "compress": true, "capacity": true,
		"compound": true, "grow": true, "depreciate": true,
	}

	t.Run("prefix matches function name", func(t *testing.T) {
		suggestions := FunctionSuggestions("av", implementedNames)
		found := false
		for _, s := range suggestions {
			if s.InsertText == "avg" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected to find avg for prefix 'av'")
		}
	})

	t.Run("synonym match", func(t *testing.T) {
		suggestions := FunctionSuggestions("mean", implementedNames)
		found := false
		for _, s := range suggestions {
			if s.InsertText == "avg" {
				found = true
				if s.Name != "avg (mean)" {
					t.Errorf("expected Name='avg (mean)', got %q", s.Name)
				}
				break
			}
		}
		if !found {
			t.Error("expected to find avg via synonym 'mean'")
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		suggestions := FunctionSuggestions("AV", implementedNames)
		found := false
		for _, s := range suggestions {
			if s.InsertText == "avg" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected case-insensitive match for 'AV'")
		}
	})

	t.Run("empty prefix returns all", func(t *testing.T) {
		suggestions := FunctionSuggestions("", implementedNames)
		if len(suggestions) < 17 { // 17 functions + NL rows
			t.Errorf("expected at least 17 suggestions for empty prefix, got %d", len(suggestions))
		}
	})

	t.Run("NL rows included", func(t *testing.T) {
		suggestions := FunctionSuggestions("av", implementedNames)
		var nlRow *Suggestion
		for i := range suggestions {
			if suggestions[i].Category == "example" && suggestions[i].InsertText == "average of 1, 2, 3" {
				nlRow = &suggestions[i]
				break
			}
		}
		if nlRow == nil {
			t.Fatal("expected NL row for avg")
		}
		if nlRow.SortCategory != "Math" {
			t.Errorf("NL row SortCategory = %q, want 'Math'", nlRow.SortCategory)
		}
	})

	t.Run("NL keyword prefix matches NL row only", func(t *testing.T) {
		suggestions := FunctionSuggestions("squa", implementedNames)
		var hasFn, hasNL bool
		for _, s := range suggestions {
			if s.InsertText == "sqrt" {
				hasFn = true
			}
			if s.Category == "example" && s.InsertText == "square root of 16" {
				hasNL = true
			}
		}
		if hasFn {
			t.Error("fn row for sqrt should NOT match prefix 'squa'")
		}
		if !hasNL {
			t.Error("expected NL row for 'square root of' with prefix 'squa'")
		}
	})

	t.Run("nil implementedNames includes all", func(t *testing.T) {
		all := FunctionSuggestions("", nil)
		filtered := FunctionSuggestions("", implementedNames)
		if len(all) < len(filtered) {
			t.Errorf("nil implementedNames should include at least as many as filtered: all=%d filtered=%d", len(all), len(filtered))
		}
	})

	t.Run("no match returns empty", func(t *testing.T) {
		suggestions := FunctionSuggestions("xyz", implementedNames)
		if len(suggestions) != 0 {
			t.Errorf("expected 0 suggestions for 'xyz', got %d", len(suggestions))
		}
	})

	t.Run("FunctionName is canonical for paren-form", func(t *testing.T) {
		suggestions := FunctionSuggestions("av", implementedNames)
		for _, s := range suggestions {
			if s.InsertText == "avg" && s.Category != "example" {
				if s.FunctionName != "avg" {
					t.Errorf("paren-form FunctionName = %q, want %q", s.FunctionName, "avg")
				}
				return
			}
		}
		t.Fatal("expected paren-form suggestion for avg")
	})

	t.Run("FunctionName is canonical for NL-example rows", func(t *testing.T) {
		suggestions := FunctionSuggestions("av", implementedNames)
		for _, s := range suggestions {
			if s.Category == "example" && s.FunctionName == "avg" {
				return // found it
			}
		}
		t.Fatal("expected NL-example row with FunctionName='avg'")
	})

	t.Run("FunctionName is canonical for grow NL-example", func(t *testing.T) {
		suggestions := FunctionSuggestions("gro", implementedNames)
		for _, s := range suggestions {
			if s.Category == "example" && s.FunctionName == "grow" {
				return
			}
		}
		t.Fatal("expected NL-example row with FunctionName='grow'")
	})

	t.Run("FunctionName is canonical for sum NL-example", func(t *testing.T) {
		suggestions := FunctionSuggestions("sum", implementedNames)
		for _, s := range suggestions {
			if s.Category == "example" && s.FunctionName == "sum" {
				return
			}
		}
		t.Fatal("expected NL-example row with FunctionName='sum'")
	})

	t.Run("NLExample fallback for functions without parseable aliases", func(t *testing.T) {
		suggestions := FunctionSuggestions("accum", implementedNames)
		var nlRow *Suggestion
		for i := range suggestions {
			if suggestions[i].Category == "example" {
				nlRow = &suggestions[i]
				break
			}
		}
		if nlRow == nil {
			t.Fatal("expected NL row for accumulate via NLExample")
		}
		if nlRow.InsertText != "100 MB/s over 1 day" {
			t.Errorf("NL InsertText = %q, want %q", nlRow.InsertText, "100 MB/s over 1 day")
		}
	})
}

func TestUnitSuggestions(t *testing.T) {
	t.Run("prefix matches canonical name", func(t *testing.T) {
		suggestions := UnitSuggestions("met")
		found := false
		for _, s := range suggestions {
			if s.InsertText == "meter" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected to find meter for prefix 'met'")
		}
	})

	t.Run("prefix matches symbol", func(t *testing.T) {
		suggestions := UnitSuggestions("kg")
		found := false
		for _, s := range suggestions {
			if s.InsertText == "kilogram" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected to find kilogram for prefix 'kg'")
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		suggestions := UnitSuggestions("MET")
		found := false
		for _, s := range suggestions {
			if s.InsertText == "meter" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected case-insensitive match for 'MET'")
		}
	})

	t.Run("deduplicates by canonical name", func(t *testing.T) {
		suggestions := UnitSuggestions("m")
		seen := make(map[string]int)
		for _, s := range suggestions {
			seen[s.InsertText]++
			if seen[s.InsertText] > 1 {
				t.Errorf("duplicate suggestion for %q", s.InsertText)
			}
		}
	})
}

func TestVariableSuggestions(t *testing.T) {
	vars := map[string]string{
		"price":    "100",
		"quantity": "5",
		"total":    "500",
	}

	t.Run("prefix match", func(t *testing.T) {
		suggestions := VariableSuggestions(vars, "pr", 10, nil)
		found := false
		for _, s := range suggestions {
			if s.InsertText == "price" {
				found = true
				if s.Description != "100" {
					t.Errorf("expected description '100', got %q", s.Description)
				}
				break
			}
		}
		if !found {
			t.Error("expected to find price for prefix 'pr'")
		}
	})

	t.Run("empty prefix returns all", func(t *testing.T) {
		suggestions := VariableSuggestions(vars, "", 10, nil)
		if len(suggestions) != 3 {
			t.Errorf("expected 3 suggestions, got %d", len(suggestions))
		}
	})

	t.Run("position filtering excludes variables at or after cursor", func(t *testing.T) {
		definedLines := map[string]int{
			"price":    0,
			"quantity": 1,
			"total":    2,
		}
		suggestions := VariableSuggestions(vars, "", 1, definedLines)
		for _, s := range suggestions {
			if s.InsertText == "quantity" || s.InsertText == "total" {
				t.Errorf("should not suggest %q defined at or after cursor line 1", s.InsertText)
			}
		}
		found := false
		for _, s := range suggestions {
			if s.InsertText == "price" {
				found = true
			}
		}
		if !found {
			t.Error("should suggest 'price' defined before cursor line 1")
		}
	})

	t.Run("nil vars returns nil", func(t *testing.T) {
		suggestions := VariableSuggestions(nil, "pr", 0, nil)
		if suggestions != nil {
			t.Errorf("expected nil, got %v", suggestions)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		suggestions := VariableSuggestions(vars, "PR", 10, nil)
		found := false
		for _, s := range suggestions {
			if s.InsertText == "price" {
				found = true
			}
		}
		if !found {
			t.Error("expected case-insensitive match for 'PR'")
		}
	})
}

func TestDirectiveSuggestions(t *testing.T) {
	t.Run("@s prefix matches @scale", func(t *testing.T) {
		suggestions := DirectiveSuggestions("@s", "3", nil)
		if len(suggestions) != 1 {
			t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
		}
		if suggestions[0].InsertText != "@scale" {
			t.Errorf("expected InsertText=@scale, got %q", suggestions[0].InsertText)
		}
	})

	t.Run("no @ prefix returns nothing", func(t *testing.T) {
		suggestions := DirectiveSuggestions("sc", "3", nil)
		if len(suggestions) != 0 {
			t.Errorf("expected 0 suggestions, got %d", len(suggestions))
		}
	})

	t.Run("no scale returns nothing for @s", func(t *testing.T) {
		suggestions := DirectiveSuggestions("@s", "", nil)
		if len(suggestions) != 0 {
			t.Errorf("expected 0 suggestions, got %d", len(suggestions))
		}
	})

	t.Run("@g matches @globals", func(t *testing.T) {
		globals := map[string]string{"tax_rate": "0.32", "budget": "$5000"}
		suggestions := DirectiveSuggestions("@g", "", globals)
		if len(suggestions) != 1 {
			t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
		}
		if suggestions[0].InsertText != "@globals." {
			t.Errorf("expected InsertText=@globals., got %q", suggestions[0].InsertText)
		}
	})

	t.Run("@globals. shows fields", func(t *testing.T) {
		globals := map[string]string{"tax_rate": "0.32", "budget": "$5000"}
		suggestions := DirectiveSuggestions("@globals.", "", globals)
		if len(suggestions) != 2 {
			t.Fatalf("expected 2 field suggestions, got %d", len(suggestions))
		}
	})

	t.Run("@globals.t narrows to tax_rate", func(t *testing.T) {
		globals := map[string]string{"tax_rate": "0.32", "budget": "$5000"}
		suggestions := DirectiveSuggestions("@globals.t", "", globals)
		if len(suggestions) != 1 {
			t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
		}
		if suggestions[0].InsertText != "@globals.tax_rate" {
			t.Errorf("expected InsertText=@globals.tax_rate, got %q", suggestions[0].InsertText)
		}
	})

	t.Run("bare @ returns scale and globals", func(t *testing.T) {
		globals := map[string]string{"tax_rate": "0.32"}
		suggestions := DirectiveSuggestions("@", "3", globals)
		if len(suggestions) != 2 {
			t.Errorf("expected 2 suggestions (@scale + @globals), got %d", len(suggestions))
		}
	})

	t.Run("no globals returns nothing for @g", func(t *testing.T) {
		suggestions := DirectiveSuggestions("@g", "", nil)
		if len(suggestions) != 0 {
			t.Errorf("expected 0 suggestions, got %d", len(suggestions))
		}
	})
}

func TestExtractPrefix(t *testing.T) {
	tests := []struct {
		name string
		line string
		col  int
		want string
	}{
		{"simple word", "avg", 3, "avg"},
		{"word in expression", "x = avg", 7, "avg"},
		{"mid word", "average", 3, "ave"},
		{"empty at col 0", "", 0, ""},
		{"@scale directive", "@scale", 6, "@scale"},
		{"@s prefix", "@s", 2, "@s"},
		{"@globals", "@globals", 8, "@globals"},
		{"@globals.field", "@globals.tax_rate", 17, "@globals.tax_rate"},
		{"@globals. (dot only)", "@globals.", 9, "@globals."},
		{"directive in expression", "a = @s", 6, "@s"},
		{"email not directive", "email@example", 13, "example"},
		{"bare @", "@", 1, "@"},
		{"col beyond line", "avg", 10, "avg"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractPrefix(tc.line, tc.col)
			if got != tc.want {
				t.Errorf("ExtractPrefix(%q, %d) = %q, want %q", tc.line, tc.col, got, tc.want)
			}
		})
	}
}

func TestCleanAliasName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"transfer...across", "transfer across"},
		{"average of", "average of"},
	}
	for _, tc := range tests {
		if got := cleanAliasName(tc.input); got != tc.want {
			t.Errorf("cleanAliasName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFirstWord(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"average of", "average"},
		{"transfer...across", "transfer"},
		{"accumulate", "accumulate"},
	}
	for _, tc := range tests {
		if got := firstWord(tc.input); got != tc.want {
			t.Errorf("firstWord(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
