package features

import (
	"strings"
	"testing"
)

func TestKeywordSuggestions(t *testing.T) {
	byName := func(sugs []Suggestion) map[string]Suggestion {
		m := make(map[string]Suggestion, len(sugs))
		for _, s := range sugs {
			m[s.Name] = s
		}
		return m
	}

	t.Run("empty prefix returns only the insertable keyword forms", func(t *testing.T) {
		got := byName(KeywordSuggestions(""))
		for _, want := range []string{"of", "in", "as % of"} {
			if _, ok := got[want]; !ok {
				t.Errorf("expected keyword %q in suggestions, got %v", want, keys(got))
			}
		}
		// Keywords without a snippet InsertText must NOT surface as slash forms.
		for _, absent := range []string{"per", "over", "at", "as", "as napkin", "as precise"} {
			if _, ok := got[absent]; ok {
				t.Errorf("keyword %q has no InsertText and must not surface", absent)
			}
		}
	})

	t.Run("percent-of snippet keeps the %% inside the first tab stop", func(t *testing.T) {
		got := byName(KeywordSuggestions(""))
		if it := got["of"].InsertText; it != "${1:23%} of ${2:1000}" {
			t.Errorf("of InsertText = %q, want the %% inside stop 1", it)
		}
		if it := got["as % of"].InsertText; it != "${1:23} as a % of ${2:43}" {
			t.Errorf("as %% of InsertText = %q", it)
		}
		if it := got["in"].InsertText; it != "${1:32 g} in ${2:oz}" {
			t.Errorf("in InsertText = %q", it)
		}
	})

	t.Run("prefix filters by keyword name", func(t *testing.T) {
		got := byName(KeywordSuggestions("of"))
		if _, ok := got["of"]; !ok {
			t.Errorf("prefix 'of' should match 'of', got %v", keys(got))
		}
		if _, ok := got["in"]; ok {
			t.Errorf("prefix 'of' should not match 'in'")
		}
	})

	t.Run("every suggestion carries the keyword category and a description", func(t *testing.T) {
		for _, s := range KeywordSuggestions("") {
			if s.Category != string(CategoryKeyword) {
				t.Errorf("%q Category = %q, want %q", s.Name, s.Category, CategoryKeyword)
			}
			if s.Description == "" {
				t.Errorf("%q has empty Description", s.Name)
			}
		}
	})
}

func keys(m map[string]Suggestion) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

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

// TestDateSuggestions_ExcludesTemplateNames verifies that template-
// form entries (like `this weekday`, `next month name`) are NOT
// surfaced as literal completions. Users are expected to type a
// specific weekday (`this Friday`) or month (`next April`) — offering
// the bare template as a one-shot completion would insert invalid
// calcmark source.
func TestDateSuggestions_ExcludesTemplateNames(t *testing.T) {
	banned := []string{
		"next weekday", "this weekday", "last weekday",
		"next month name", "this month name", "last month name",
	}
	// Use prefixes that would otherwise match these names.
	for _, prefix := range []string{"", "this", "last", "next"} {
		sugg := DateSuggestions(prefix)
		for _, b := range banned {
			for _, s := range sugg {
				if s.Name == b {
					t.Errorf("DateSuggestions(%q) unexpectedly surfaced template %q", prefix, b)
				}
			}
		}
	}
}

// TestDateSuggestions_KeepsLiteralAliases confirms valid literal
// entries in the features registry survive the template filter.
func TestDateSuggestions_KeepsLiteralAliases(t *testing.T) {
	required := []string{"today", "this quarter"}
	sugg := DateSuggestions("")
	for _, want := range required {
		found := false
		for _, s := range sugg {
			if s.Name == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DateSuggestions(\"\") should still surface literal %q", want)
		}
	}
}

// TestDateSuggestions_ThisPrefixCoversNamedPeriods verifies that
// typing `this` in a calc block surfaces every named time period the
// user can anchor with `this`, not just `this quarter`. The lexer
// recognizes `this week / this month / this year / this quarter` as
// relative date keywords, so completion must match.
func TestDateSuggestions_ThisPrefixCoversNamedPeriods(t *testing.T) {
	required := []string{"this week", "this month", "this year", "this quarter"}
	sugg := DateSuggestions("this")
	for _, want := range required {
		found := false
		for _, s := range sugg {
			if s.Name == want {
				found = true
				if s.Category != "Date" {
					t.Errorf("suggestion %q has category %q, want Date", s.Name, s.Category)
				}
				break
			}
		}
		if !found {
			names := make([]string, len(sugg))
			for i, s := range sugg {
				names[i] = s.Name
			}
			t.Errorf("DateSuggestions(\"this\") missing %q; got %v", want, names)
		}
	}
}

// TestDateSuggestions_QPrefixSurfacesCalendarQuarters verifies that
// typing `Q` in a calc block surfaces the four calendar-quarter
// literal forms the lexer accepts (`Q1`-`Q4`). Reported 2026-04-28
// against calcmark-web: typing `FQ` or `Q` produces zero suggestions
// despite the lexer recognizing `CALENDAR_QUARTER_LITERAL` /
// `FISCAL_QUARTER_LITERAL` and the evaluator computing them
// correctly. This test pins the contract: every literal token kind
// the lexer accepts must be discoverable via prefix completion.
func TestDateSuggestions_QPrefixSurfacesCalendarQuarters(t *testing.T) {
	required := []string{"Q1", "Q2", "Q3", "Q4"}
	got := DateSuggestions("Q")
	for _, want := range required {
		found := false
		for _, s := range got {
			if s.Name == want {
				found = true
				if s.Category != "Date" {
					t.Errorf("suggestion %q has category %q, want Date", s.Name, s.Category)
				}
				break
			}
		}
		if !found {
			names := make([]string, len(got))
			for i, s := range got {
				names[i] = s.Name
			}
			t.Errorf("DateSuggestions(\"Q\") missing %q; got %v", want, names)
		}
	}
}

// TestDateSuggestions_QPrefixCaseInsensitive — calcmark identifiers
// are case-insensitive in the lexer (q1 → CALENDAR_QUARTER_LITERAL).
// Completion mirrors that: lowercase prefix returns the same set.
func TestDateSuggestions_QPrefixCaseInsensitive(t *testing.T) {
	got := DateSuggestions("q")
	if len(got) < 4 {
		t.Errorf("DateSuggestions(\"q\") should return at least 4 quarter items; got %d", len(got))
	}
}

// TestDateSuggestions_Q1ExactPrefixSelectsOne — typing the literal
// itself narrows to exactly that entry.
func TestDateSuggestions_Q1ExactPrefixSelectsOne(t *testing.T) {
	got := DateSuggestions("Q1")
	count := 0
	for _, s := range got {
		if s.Name == "Q1" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("DateSuggestions(\"Q1\") should match exactly one Q1 entry; got %d matches in %v", count, got)
	}
}

// TestDateSuggestions_Q5ReturnsNothingMatchingQ5 — the parser rejects
// Q5 (`invalid calendar quarter`); completion must not offer it.
func TestDateSuggestions_Q5ReturnsNothingMatchingQ5(t *testing.T) {
	got := DateSuggestions("Q5")
	for _, s := range got {
		if s.Name == "Q5" {
			t.Errorf("DateSuggestions(\"Q5\") unexpectedly surfaced %q", s.Name)
		}
	}
}

// TestDateSuggestions_QuarterEntriesHaveDescriptionAndExample —
// every new entry must carry the full registry shape so hover docs
// and the LSP detail panel have content to show.
func TestDateSuggestions_QuarterEntriesHaveDescriptionAndExample(t *testing.T) {
	got := DateSuggestions("Q")
	for _, s := range got {
		if s.Name != "Q1" && s.Name != "Q2" && s.Name != "Q3" && s.Name != "Q4" {
			continue
		}
		if s.Description == "" {
			t.Errorf("calendar-quarter %q has empty Description", s.Name)
		}
		if s.Syntax == "" {
			t.Errorf("calendar-quarter %q has empty Syntax", s.Name)
		}
	}
}

// TestDateSuggestions_FQPrefixSurfacesFiscalQuarters — typing `FQ`
// returns FQ1-FQ4. Symmetric to Q1-Q4 but documented as requiring
// `fiscal_year_starts` frontmatter (the interpreter raises a
// fiscal-required runtime error otherwise; surfacing that
// requirement in the completion description prevents user
// confusion).
func TestDateSuggestions_FQPrefixSurfacesFiscalQuarters(t *testing.T) {
	required := []string{"FQ1", "FQ2", "FQ3", "FQ4"}
	got := DateSuggestions("FQ")
	for _, want := range required {
		found := false
		for _, s := range got {
			if s.Name == want {
				found = true
				if s.Category != "Date" {
					t.Errorf("suggestion %q has category %q, want Date", s.Name, s.Category)
				}
				if !strings.Contains(s.Description, "fiscal_year_starts") {
					t.Errorf("FQ-family suggestion %q description should mention fiscal_year_starts; got %q",
						s.Name, s.Description)
				}
				break
			}
		}
		if !found {
			names := make([]string, len(got))
			for i, s := range got {
				names[i] = s.Name
			}
			t.Errorf("DateSuggestions(\"FQ\") missing %q; got %v", want, names)
		}
	}
}

// TestDateSuggestions_FQ0AndFQ5RejectedByLexer — completion mirrors
// what the lexer accepts. FQ0 / FQ5 are not valid lexer tokens, so
// they must not appear as suggestions.
func TestDateSuggestions_FQ0AndFQ5RejectedByLexer(t *testing.T) {
	got := DateSuggestions("FQ")
	for _, s := range got {
		if s.Name == "FQ0" || s.Name == "FQ5" {
			t.Errorf("DateSuggestions(\"FQ\") unexpectedly surfaced %q", s.Name)
		}
	}
}

// TestDateSuggestions_FiscalQuarterShorthandAliases — the lexer
// already accepts `this FQ` / `next FQ` / `last FQ` (verified in
// spec/lexer/date_tokenizer_test.go). The registry should reflect
// that by listing them as Parseable aliases on `fiscal quarter`.
// This test exercises the alias path through DateSuggestions.
func TestDateSuggestions_FiscalQuarterShorthandAliases(t *testing.T) {
	wanted := []string{"this FQ", "next FQ", "last FQ"}
	got := DateSuggestions("this F")
	gotNext := DateSuggestions("next F")
	gotLast := DateSuggestions("last F")
	all := append(append(append([]Suggestion{}, got...), gotNext...), gotLast...)
	for _, want := range wanted {
		found := false
		for _, s := range all {
			if s.Name == want {
				found = true
				break
			}
		}
		if !found {
			names := make([]string, len(all))
			for i, s := range all {
				names[i] = s.Name
			}
			t.Errorf("expected alias %q across this/next/last F prefix queries; got %v", want, names)
		}
	}
}

// TestDateSuggestions_FiscalYearShorthandAliases — `this FY` /
// `next FY` / `last FY` are lexed equivalently; same registry
// alias treatment.
func TestDateSuggestions_FiscalYearShorthandAliases(t *testing.T) {
	wanted := []string{"this FY", "next FY", "last FY"}
	all := append(append(append(
		[]Suggestion{},
		DateSuggestions("this F")...),
		DateSuggestions("next F")...),
		DateSuggestions("last F")...)
	for _, want := range wanted {
		found := false
		for _, s := range all {
			if s.Name == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected alias %q across this/next/last F prefix queries", want)
		}
	}
}

// TestDateSuggestions_FYPrefixSurfacesYearLiteralSnippet — typing
// `FY` returns a single snippet completion that fills in the current
// calendar year as a placeholder. The lexer recognizes `FY26` and
// `FY2026` as FISCAL_YEAR_LITERAL; this entry is the discoverability
// path. The snippet's InsertText carries `${1:NNNN}` so editors can
// honor it as a placeholder. The registry stores the snippet text;
// LSP-layer conversion to `InsertTextFormat: Snippet` happens
// downstream in lsp/completion.go (covered by U3's lsp/ test).
func TestDateSuggestions_FYPrefixSurfacesYearLiteralSnippet(t *testing.T) {
	got := DateSuggestions("FY")
	found := false
	for _, s := range got {
		// Allow a single FY entry whose InsertText contains
		// `${1:` (the snippet placeholder marker).
		if s.Name == "FY" && strings.Contains(s.InsertText, "${1:") {
			found = true
			if !strings.Contains(s.Description, "fiscal_year_starts") {
				t.Errorf("FY snippet description should mention fiscal_year_starts; got %q", s.Description)
			}
			break
		}
	}
	if !found {
		t.Errorf("DateSuggestions(\"FY\") missing FY snippet entry with ${1:NNNN} placeholder; got %v", got)
	}
}

// TestDateSuggestions_FYSnippetDefaultsToCurrentYear — the
// placeholder default is `now.Year()` so users see a sensible
// pre-filled value. Format: `FY${1:2026}`.
func TestDateSuggestions_FYSnippetDefaultsToCurrentYear(t *testing.T) {
	got := DateSuggestions("FY")
	for _, s := range got {
		if s.Name != "FY" {
			continue
		}
		// The exact year value depends on `time.Now()` at call time;
		// verify the placeholder is exactly 4 digits in the default
		// position so editors can interpret it as a year.
		if !strings.Contains(s.InsertText, "${1:20") {
			t.Errorf("FY snippet should default to a current-millennium year (FY${1:20YY}); got %q", s.InsertText)
		}
		return
	}
	t.Fatalf("no FY entry returned by DateSuggestions(\"FY\")")
}

// TestDateSuggestions_CYPrefixSurfacesYearLiteralSnippet — symmetric
// to FY, for calendar years.
func TestDateSuggestions_CYPrefixSurfacesYearLiteralSnippet(t *testing.T) {
	got := DateSuggestions("CY")
	found := false
	for _, s := range got {
		if s.Name == "CY" && strings.Contains(s.InsertText, "${1:") {
			found = true
			if s.Description == "" {
				t.Errorf("CY snippet description should be non-empty")
			}
			break
		}
	}
	if !found {
		t.Errorf("DateSuggestions(\"CY\") missing CY snippet entry; got %v", got)
	}
}

// TestDateSuggestions_EndPrefixSynthesizesEndOfPeriods — typing
// `end` returns synthesized `end of <period>` items, derived from
// the registered period-bearing date features. The synthesis
// handler skips non-period date entries (`today`, `tomorrow`,
// `days`, `weeks`, etc.) via a local skip-list. New period kinds
// added to the registry automatically flow through this handler
// without manual registry expansion.
func TestDateSuggestions_EndPrefixSynthesizesEndOfPeriods(t *testing.T) {
	got := DateSuggestions("end")
	required := []string{
		"end of Q1", "end of Q2", "end of Q3", "end of Q4",
		"end of FQ1", "end of FQ2", "end of FQ3", "end of FQ4",
		"end of this month", "end of this quarter", "end of this year",
		"end of fiscal quarter", "end of fiscal year",
	}
	for _, want := range required {
		found := false
		for _, s := range got {
			if s.Name == want {
				found = true
				break
			}
		}
		if !found {
			names := make([]string, len(got))
			for i, s := range got {
				names[i] = s.Name
			}
			t.Errorf("DateSuggestions(\"end\") missing %q; got %v", want, names)
		}
	}
}

// TestDateSuggestions_EndOfPrefixEquivalentToEnd — typing `end of`
// returns the same synthesized set as `end` (the trailing space + of
// don't change which period set is offered).
func TestDateSuggestions_EndOfPrefixEquivalentToEnd(t *testing.T) {
	gotEnd := DateSuggestions("end")
	gotEndOf := DateSuggestions("end of")
	endNames := nameSet(gotEnd)
	endOfNames := nameSet(gotEndOf)
	for name := range endNames {
		if !endOfNames[name] {
			t.Errorf("DateSuggestions(\"end of\") missing %q present in DateSuggestions(\"end\")", name)
		}
	}
}

// TestDateSuggestions_EndOfQNarrowsToCalendarQuarters — partial
// prefix narrows the synthesized set.
func TestDateSuggestions_EndOfQNarrowsToCalendarQuarters(t *testing.T) {
	got := DateSuggestions("end of Q")
	required := []string{"end of Q1", "end of Q2", "end of Q3", "end of Q4"}
	for _, want := range required {
		found := false
		for _, s := range got {
			if s.Name == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DateSuggestions(\"end of Q\") missing %q", want)
		}
	}
	// Should NOT include FQ-family items (different prefix).
	for _, s := range got {
		if strings.HasPrefix(s.Name, "end of FQ") {
			t.Errorf("DateSuggestions(\"end of Q\") unexpectedly surfaced FQ-family item %q", s.Name)
		}
	}
}

// TestDateSuggestions_StartPrefixMirrorsEnd — `start of <period>`
// is the symmetric form. Same synthesized set with `start` prefix.
func TestDateSuggestions_StartPrefixMirrorsEnd(t *testing.T) {
	got := DateSuggestions("start")
	required := []string{
		"start of Q1", "start of FQ1",
		"start of this month", "start of fiscal quarter",
	}
	for _, want := range required {
		found := false
		for _, s := range got {
			if s.Name == want {
				found = true
				break
			}
		}
		if !found {
			names := make([]string, len(got))
			for i, s := range got {
				names[i] = s.Name
			}
			t.Errorf("DateSuggestions(\"start\") missing %q; got %v", want, names)
		}
	}
}

// TestDateSuggestions_EndPrefixSkipsNonPeriodEntries — synthesis
// excludes non-period date features (today, tomorrow, days, weeks)
// because `end of today` is not a meaningful expression.
func TestDateSuggestions_EndPrefixSkipsNonPeriodEntries(t *testing.T) {
	got := DateSuggestions("end")
	for _, s := range got {
		for _, banned := range []string{
			"end of today", "end of tomorrow", "end of yesterday", "end of now",
			"end of days", "end of weeks", "end of months", "end of years",
			"end of from", "end of ago",
		} {
			if s.Name == banned {
				t.Errorf("DateSuggestions(\"end\") unexpectedly synthesized %q", banned)
			}
		}
	}
}

// TestDateSuggestions_EndingDoesNotMatchSynthesis — the literal
// English word `ending` should NOT trigger synthesis. The handler
// must use a word-boundary check, not a naive prefix match.
func TestDateSuggestions_EndingDoesNotMatchSynthesis(t *testing.T) {
	got := DateSuggestions("ending")
	for _, s := range got {
		if strings.HasPrefix(s.Name, "end of") {
			t.Errorf("DateSuggestions(\"ending\") incorrectly synthesized %q", s.Name)
		}
	}
}

// TestDateSuggestions_EndOfFYPropagatesSnippet — the year-bearing
// FY/CY entries carry snippet placeholders. The synthesis handler
// must propagate them so `end of FY${1:NNNN}` is offered (rather
// than `end of FY` with no placeholder).
func TestDateSuggestions_EndOfFYPropagatesSnippet(t *testing.T) {
	got := DateSuggestions("end of FY")
	found := false
	for _, s := range got {
		if s.Name == "end of FY" && strings.Contains(s.InsertText, "${1:") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("DateSuggestions(\"end of FY\") should offer snippet `end of FY${1:NNNN}`; got %v", got)
	}
}

// nameSet — small helper for set-membership comparisons.
func nameSet(ss []Suggestion) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s.Name] = true
	}
	return m
}

// TestDateSuggestions_BareFQSurfacesThisFQAlias — typing `FQ`
// alone should surface `this FQ` (and friends) as suggestions
// even though the lexer rejects bare `FQ`. The dropdown shows
// the alias-name (`this FQ`), inserts the alias-name (which the
// lexer accepts), and the user gets "current fiscal quarter"
// semantics — what they probably wanted by typing FQ.
//
// Lexer-design context: spec/lexer/date_keywords.go:26 explicitly
// excludes bare CY/FY/CQ/FQ from DateKeywords because they collide
// with the notation parser (`FQ` would consume out of `FQ1`).
// This suffix-matching path bridges the discoverability gap
// without changing the lexer.
func TestDateSuggestions_BareFQSurfacesThisFQAlias(t *testing.T) {
	got := DateSuggestions("FQ")
	wanted := []string{"this FQ", "next FQ", "last FQ"}
	for _, want := range wanted {
		found := false
		for _, s := range got {
			if s.Name == want {
				found = true
				break
			}
		}
		if !found {
			names := make([]string, len(got))
			for i, s := range got {
				names[i] = s.Name
			}
			t.Errorf("DateSuggestions(\"FQ\") missing %q; got %v", want, names)
		}
	}
}

// TestDateSuggestions_BareFYSurfacesThisFYAlias — same shape for FY.
func TestDateSuggestions_BareFYSurfacesThisFYAlias(t *testing.T) {
	got := DateSuggestions("FY")
	wanted := []string{"this FY", "next FY", "last FY"}
	for _, want := range wanted {
		found := false
		for _, s := range got {
			if s.Name == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DateSuggestions(\"FY\") missing %q", want)
		}
	}
}

// TestDateSuggestions_BareCYSurfacesThisCYAlias — symmetric for
// calendar year. `CY` aliases need to be registered first; this
// test will exercise that wiring.
func TestDateSuggestions_BareCYSurfacesThisCYAlias(t *testing.T) {
	got := DateSuggestions("CY")
	want := "this CY"
	found := false
	for _, s := range got {
		if s.Name == want {
			found = true
			break
		}
	}
	if !found {
		// CY aliases not yet registered — this test will pass once
		// the next commit adds them. Fail loudly so it doesn't slip.
		t.Errorf("DateSuggestions(\"CY\") missing %q (CY aliases need to be registered on `this year`)", want)
	}
}

// TestDateSuggestions_BareCQSurfacesThisCQAlias — calendar quarter
// abbreviation.
func TestDateSuggestions_BareCQSurfacesThisCQAlias(t *testing.T) {
	got := DateSuggestions("CQ")
	want := "this CQ"
	found := false
	for _, s := range got {
		if s.Name == want {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("DateSuggestions(\"CQ\") missing %q (CQ aliases need to be registered on `this quarter`)", want)
	}
}
