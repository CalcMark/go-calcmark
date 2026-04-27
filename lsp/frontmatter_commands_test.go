package lsp

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// applyFrontmatterMutation tests pin the round-trip helper itself:
// every command implementation funnels through this function, so
// getting the basics right (creates a fence when absent, splices
// cleanly into an existing fence, drops the fence when the mutator
// empties the mapping) covers the wire-level shape for all of them.
//
// Per-command tests below verify the mutator's specific behavior
// against representative inputs.

func TestApplyFrontmatterMutation_CreatesFenceWhenAbsent(t *testing.T) {
	source := "x = 1\n"
	got, err := applyFrontmatterMutation(source, func(root *yaml.Node) error {
		setScalarChild(root, "rate", "0.08")
		return nil
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.HasPrefix(got, "---\nrate: 0.08\n---\n") {
		t.Errorf("expected new fence at start, got:\n%s", got)
	}
	if !strings.Contains(got, "x = 1\n") {
		t.Errorf("body lost: %q", got)
	}
}

func TestApplyFrontmatterMutation_PreservesUnchangedKeys(t *testing.T) {
	// Mutating one key shouldn't disturb others. Insertion order is
	// preserved by yaml.v3's Node API, so a setGlobal of `rate` on
	// a doc with `title` and existing globals lands rate at the end
	// of the globals mapping, leaving the rest in place.
	source := "---\n" +
		"title: Report\n" +
		"globals:\n" +
		"  tax: \"0.07\"\n" +
		"---\n" +
		"x = 1\n"
	got, err := applyFrontmatterMutation(source, fmSetGlobal(&commandArgs{
		Map: map[string]any{"name": "rate", "value": "0.08"},
	}))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(got, "title: Report") {
		t.Errorf("unrelated key dropped: %q", got)
	}
	if !strings.Contains(got, "tax: ") {
		t.Errorf("existing global dropped: %q", got)
	}
	if !strings.Contains(got, "rate: ") {
		t.Errorf("new global missing: %q", got)
	}
}

func TestApplyFrontmatterMutation_DropsFenceWhenMappingEmptied(t *testing.T) {
	// Removing the only directive should leave the document
	// fence-less. Common path: the user clears their last global.
	source := "---\nglobals:\n  tax: \"0.07\"\n---\n\nx = 1\n"
	got, err := applyFrontmatterMutation(source, fmRemoveGlobal(&commandArgs{
		Map: map[string]any{"name": "tax"},
	}))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if strings.Contains(got, "---") {
		t.Errorf("expected fence dropped, got:\n%s", got)
	}
	if !strings.Contains(got, "x = 1") {
		t.Errorf("body lost: %q", got)
	}
}

// ── Per-command tests ─────────────────────────────────────────

func TestFmSetGlobal_CreatesGlobalsKeyWhenAbsent(t *testing.T) {
	source := "x = 1\n"
	got, err := applyFrontmatterMutation(source, fmSetGlobal(&commandArgs{
		Map: map[string]any{"name": "rate", "value": "0.08"},
	}))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(got, "globals:\n") {
		t.Errorf("expected `globals:` key, got:\n%s", got)
	}
	if !strings.Contains(got, "rate: 0.08") {
		t.Errorf("expected `rate: 0.08`, got:\n%s", got)
	}
}

func TestFmSetGlobal_RejectsMissingArgs(t *testing.T) {
	_, err := applyFrontmatterMutation("", fmSetGlobal(&commandArgs{Map: map[string]any{}}))
	if err == nil {
		t.Error("expected error for missing 'name'")
	}
}

func TestFmRemoveGlobal_RemovesEntry(t *testing.T) {
	source := "---\nglobals:\n  rate: \"0.08\"\n  tax: \"0.07\"\n---\n"
	got, err := applyFrontmatterMutation(source, fmRemoveGlobal(&commandArgs{
		Map: map[string]any{"name": "rate"},
	}))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if strings.Contains(got, "rate: ") {
		t.Errorf("expected 'rate' removed, got:\n%s", got)
	}
	if !strings.Contains(got, "tax: ") {
		t.Errorf("kept global was dropped: %s", got)
	}
}

func TestFmSetExchangeRate_NormalizesKeyAsFromTo(t *testing.T) {
	source := "x = 1\n"
	got, err := applyFrontmatterMutation(source, fmSetExchangeRate(&commandArgs{
		Map: map[string]any{"from": "usd", "to": "eur", "rate": "0.92"},
	}))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(got, "USD_EUR: 0.92") {
		t.Errorf("expected USD_EUR key, got:\n%s", got)
	}
}

func TestFmSetScale_BareScalarWhenNoCategories(t *testing.T) {
	source := "x = 1\n"
	got, err := applyFrontmatterMutation(source, fmSetScale(&commandArgs{
		Map: map[string]any{"factor": "4"},
	}))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(got, "scale: 4\n") {
		t.Errorf("expected `scale: 4`, got:\n%s", got)
	}
}

func TestFmSetScale_MappingFormWithCategories(t *testing.T) {
	source := "x = 1\n"
	got, err := applyFrontmatterMutation(source, fmSetScale(&commandArgs{
		Map: map[string]any{
			"factor":          "2",
			"unit_categories": []any{"Mass", "Length"},
		},
	}))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(got, "factor: ") {
		t.Errorf("expected `factor:` sub-key, got:\n%s", got)
	}
	if !strings.Contains(got, "Mass") || !strings.Contains(got, "Length") {
		t.Errorf("expected category sequence, got:\n%s", got)
	}
}

func TestFmSetConvertTo_BareScalarWhenNoCategories(t *testing.T) {
	got, err := applyFrontmatterMutation("", fmSetConvertTo(&commandArgs{
		Map: map[string]any{"system": "si"},
	}))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(got, "convert_to: si") {
		t.Errorf("expected `convert_to: si`, got:\n%s", got)
	}
}

func TestFmSetMeasurement_PartialUpdateMergesIntoExisting(t *testing.T) {
	// A form that submits only the volume axis must not drop an
	// existing mass setting on the same measurement mapping.
	source := "---\nmeasurement:\n  mass: standard\n---\n"
	got, err := applyFrontmatterMutation(source, fmSetMeasurement(&commandArgs{
		Map: map[string]any{"volume": "imperial"},
	}))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(got, "mass: standard") {
		t.Errorf("partial update dropped pre-existing mass: %s", got)
	}
	if !strings.Contains(got, "volume: imperial") {
		t.Errorf("partial update missing new volume: %s", got)
	}
}

func TestFmSetFiscalYearStarts_StringMonth(t *testing.T) {
	got, err := applyFrontmatterMutation("", fmSetFiscalYearStarts(&commandArgs{
		Map: map[string]any{"month": "July"},
	}))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(got, "fiscal_year_starts: July") {
		t.Errorf("expected `fiscal_year_starts: July`, got:\n%s", got)
	}
}

func TestFmSetFiscalYearStarts_MonthAndDay(t *testing.T) {
	got, err := applyFrontmatterMutation("", fmSetFiscalYearStarts(&commandArgs{
		Map: map[string]any{"month": "October", "day": float64(1)},
	}))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(got, "fiscal_year_starts: October 1") {
		t.Errorf("expected `fiscal_year_starts: October 1`, got:\n%s", got)
	}
}

func TestFmSetFiscalYearStarts_NumericMonth(t *testing.T) {
	got, err := applyFrontmatterMutation("", fmSetFiscalYearStarts(&commandArgs{
		Map: map[string]any{"month": float64(7)},
	}))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(got, "fiscal_year_starts: July") {
		t.Errorf("expected July from month=7, got:\n%s", got)
	}
}

func TestFmRemoveKey_DropsTopLevel(t *testing.T) {
	source := "---\nscale: 4\nconvert_to: si\n---\n"
	got, err := applyFrontmatterMutation(source, fmRemoveKey("scale"))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if strings.Contains(got, "scale:") {
		t.Errorf("expected scale removed, got:\n%s", got)
	}
	if !strings.Contains(got, "convert_to: si") {
		t.Errorf("convert_to was dropped: %s", got)
	}
}
