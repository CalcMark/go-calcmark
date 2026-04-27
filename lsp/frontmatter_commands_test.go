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

// User-flagged 2026-04-27: every mutation auto-chases itself with
// a format command from the client (FrontmatterPanel.dispatch). If
// the round-trip ever stutters a `---` into the output, the format
// pass exposes it on the very next save. Pin idempotency directly:
// applying ANY mutation TWICE produces the same shape as applying
// it once.
func TestApplyFrontmatterMutation_IdempotentUnderFormatChase(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{
			name:   "minimal fence + body",
			source: "---\nfiscal_year_starts: October\n---\n\nend of Q1\n",
		},
		{
			name:   "fence with multiple keys",
			source: "---\ntitle: Report\nfiscal_year_starts: October\n---\n\nbody\n",
		},
		{
			name:   "fence with no body",
			source: "---\nfiscal_year_starts: October\n---\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			once, err := applyFrontmatterMutation(tc.source, fmFormat())
			if err != nil {
				t.Fatalf("first apply: %v", err)
			}
			twice, err := applyFrontmatterMutation(once, fmFormat())
			if err != nil {
				t.Fatalf("second apply: %v", err)
			}
			if once != twice {
				t.Errorf(
					"format is not idempotent — second pass changed the source.\nFIRST:\n%s\nSECOND:\n%s",
					once, twice,
				)
			}
		})
	}
}

// User-flagged 2026-04-27: saving fiscal_year_starts produced a
// document with TWO `---` separators where there should be one,
// rendering as horizontal rules in the body. Pin the bug at the
// helper level so any encoder/regex regression trips the test.
// TestDocEndPosition pins the (line, character) computation that
// drives the WorkspaceEdit's range. The client's
// applyTextEditsToString validates these values; a wrong answer
// here either rejects the edit (best case) or applies it against a
// truncated range (silent corruption — the user's repro shape).
func TestDocEndPosition(t *testing.T) {
	cases := []struct {
		source        string
		wantLine      int
		wantCharacter int
	}{
		{"", 0, 0},
		{"hello", 0, 5},
		{"hello\n", 1, 0},
		{"a\nb\n", 2, 0},
		{"a\nb", 1, 1},
		{"---\nfoo\n---\n", 3, 0},
		{"---\nfoo\n---\n\nbody\n", 5, 0},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			line, ch := docEndPosition(tc.source)
			if line != tc.wantLine || ch != tc.wantCharacter {
				t.Errorf("docEndPosition(%q) = (%d, %d), want (%d, %d)",
					tc.source, line, ch, tc.wantLine, tc.wantCharacter)
			}
		})
	}
}

func TestApplyFrontmatterMutation_NoDoubledFenceOnReplace(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{
			name:   "minimal fence + body",
			source: "---\nfiscal_year_starts: October\n---\n\nend of Q1\n",
		},
		{
			name:   "fence with multiple keys",
			source: "---\ntitle: Report\nfiscal_year_starts: October\n---\n\nbody\n",
		},
		{
			name:   "fence with no body",
			source: "---\nfiscal_year_starts: October\n---\n",
		},
		{
			name:   "fence followed by `---` text in body (legitimate hr)",
			source: "---\nfiscal_year_starts: October\n---\n\nbody\n\n---\n\nmore body\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := applyFrontmatterMutation(tc.source, fmSetFiscalYearStarts(&commandArgs{
				Map: map[string]any{"month": "January"},
			}))
			if err != nil {
				t.Fatalf("apply: %v", err)
			}
			// Only the leading and closing `---` of the fence should
			// appear in the first three lines. A doubled fence
			// produces three or four `---` lines back-to-back.
			lines := strings.SplitN(got, "\n", 6)
			fenceLines := 0
			for i, line := range lines {
				if i > 4 {
					break
				}
				if line == "---" {
					fenceLines++
				}
			}
			if fenceLines > 2 {
				t.Errorf("expected exactly 2 `---` lines (fence open + close), got %d:\n%s", fenceLines, got)
			}
		})
	}
}
