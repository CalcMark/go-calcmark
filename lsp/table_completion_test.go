package lsp

import (
	"slices"
	"testing"
)

// After `table.` the LSP completes the table's column names and nothing
// else (go-calcmark#118, R16). Table names themselves arrive through the
// ordinary variable path because tables live in the environment.
func TestCompletion_TableColumnsAfterDot(t *testing.T) {
	source := "<!-- table: rates (role, rate, hc) -->\n| Role | Rate | HC |\n|------|------|----|\n| Senior | $250 | 2 |\n\ncost = rates."
	s, uri := prepareServerDoc(t, source)
	items := completionAt(t, s, uri, 5, uint32(len("cost = rates.")))
	labels := itemLabels(items)
	for _, want := range []string{"role", "rate", "hc"} {
		if !slices.Contains(labels, want) {
			t.Errorf("want column %q in completions, got %v", want, labels)
		}
	}
	for _, l := range labels {
		if l == "sum" || l == "avg" || l == "rates" {
			t.Errorf("only columns belong after `rates.`, got %v", labels)
			break
		}
	}
}

func TestCompletion_TableColumnsFilterByPrefix(t *testing.T) {
	source := "<!-- table: rates (role, rate, hc) -->\n| Role | Rate | HC |\n|------|------|----|\n| Senior | $250 | 2 |\n\ncost = rates.r"
	s, uri := prepareServerDoc(t, source)
	labels := itemLabels(completionAt(t, s, uri, 5, uint32(len("cost = rates.r"))))
	if !slices.Contains(labels, "role") || !slices.Contains(labels, "rate") || slices.Contains(labels, "hc") {
		t.Errorf("prefix `r` should keep role and rate only, got %v", labels)
	}
}

func TestCompletion_TableNameIsOfferedAsVariable(t *testing.T) {
	source := "<!-- table: rates (rate) -->\n| Rate |\n|------|\n| $1 |\n\ncost = ra"
	s, uri := prepareServerDoc(t, source)
	labels := itemLabels(completionAt(t, s, uri, 5, uint32(len("cost = ra"))))
	if !slices.Contains(labels, "rates") {
		t.Errorf("table name should complete like a variable, got %v", labels)
	}
}
