package document

import (
	"slices"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/format/display"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

func TestInterpolateLine(t *testing.T) {
	env := map[string]types.Type{
		"x": types.NewNumber(decimal.NewFromInt(42)),
	}
	df := display.DefaultFormatter()

	got := interpolateLine("Result: {{x}}", env, df, nil, nil, false)
	want := "Result: 42"
	if got != want {
		t.Errorf("interpolateLine() = %q, want %q", got, want)
	}
}

func TestInterpolateLineMultipleTags(t *testing.T) {
	env := map[string]types.Type{
		"a": types.NewNumber(decimal.NewFromInt(10)),
		"b": types.NewNumber(decimal.NewFromInt(20)),
	}
	df := display.DefaultFormatter()

	got := interpolateLine("{{a}} and {{b}}", env, df, nil, nil, false)
	want := "10 and 20"
	if got != want {
		t.Errorf("interpolateLine() = %q, want %q", got, want)
	}
}

func TestInterpolateLineMissingVar(t *testing.T) {
	env := map[string]types.Type{}
	df := display.DefaultFormatter()

	got := interpolateLine("Value: {{unknown}}", env, df, nil, nil, false)
	want := "Value: {{unknown}}"
	if got != want {
		t.Errorf("interpolateLine() = %q, want %q", got, want)
	}
}

func TestInterpolateLineNoTags(t *testing.T) {
	env := map[string]types.Type{
		"x": types.NewNumber(decimal.NewFromInt(1)),
	}
	df := display.DefaultFormatter()

	input := "Just plain text"
	got := interpolateLine(input, env, df, nil, nil, false)
	if got != input {
		t.Errorf("interpolateLine() = %q, want %q", got, input)
	}
}

func TestInterpolateLineDisplayFormatted(t *testing.T) {
	env := map[string]types.Type{
		"cost":    types.NewCurrency(decimal.NewFromFloat(1200000), "USD"),
		"pct":     types.NewPercentage(decimal.NewFromFloat(0.28)),
		"widgets": types.NewQuantity(decimal.NewFromInt(14), "people"),
	}
	df := display.DefaultFormatter()

	tests := []struct {
		input string
		want  string
	}{
		{"Total: {{cost}}", "Total: $1.2M"},
		{"Margin: {{pct}}", "Margin: 28%"},
		{"Team: {{widgets}}", "Team: 14 people"},
	}
	for _, tt := range tests {
		got := interpolateLine(tt.input, env, df, nil, nil, false)
		if got != tt.want {
			t.Errorf("interpolateLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestInterpolateLineAdjacentTags(t *testing.T) {
	env := map[string]types.Type{
		"a": types.NewNumber(decimal.NewFromInt(1)),
		"b": types.NewNumber(decimal.NewFromInt(2)),
	}
	df := display.DefaultFormatter()

	got := interpolateLine("{{a}}{{b}}", env, df, nil, nil, false)
	want := "12"
	if got != want {
		t.Errorf("interpolateLine() = %q, want %q", got, want)
	}
}

func TestInterpolateLineInTable(t *testing.T) {
	env := map[string]types.Type{
		"rev": types.NewCurrency(decimal.NewFromFloat(4200000), "USD"),
	}
	df := display.DefaultFormatter()

	got := interpolateLine("| Revenue | {{rev}} |", env, df, nil, nil, false)
	want := "| Revenue | $4.2M |"
	if got != want {
		t.Errorf("interpolateLine() = %q, want %q", got, want)
	}
}

func TestInterpolateLineInHeading(t *testing.T) {
	env := map[string]types.Type{
		"total": types.NewNumber(decimal.NewFromInt(500)),
	}
	df := display.DefaultFormatter()

	got := interpolateLine("# Summary: {{total}}", env, df, nil, nil, false)
	want := "# Summary: 500"
	if got != want {
		t.Errorf("interpolateLine() = %q, want %q", got, want)
	}
}

func TestInterpolateLineWhitespace(t *testing.T) {
	env := map[string]types.Type{
		"x": types.NewNumber(decimal.NewFromInt(99)),
	}
	df := display.DefaultFormatter()

	tests := []struct {
		input string
		want  string
	}{
		{"{{ x }}", "99"},
		{"{{  x  }}", "99"},
		{"{{ x}}", "99"},
		{"{{x }}", "99"},
	}
	for _, tt := range tests {
		got := interpolateLine(tt.input, env, df, nil, nil, false)
		if got != tt.want {
			t.Errorf("interpolateLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestInterpolateLinePartialBraces(t *testing.T) {
	env := map[string]types.Type{
		"x": types.NewNumber(decimal.NewFromInt(1)),
	}
	df := display.DefaultFormatter()

	// These should NOT be matched
	tests := []string{
		"{x}",       // Single braces
		"{{}}",      // Empty tag
		"{{x",       // Unclosed
		"x}}",       // No opening
		"{{a + b}}", // Expression (space prevents \w+ match)
	}
	for _, input := range tests {
		got := interpolateLine(input, env, df, nil, nil, false)
		if got != input {
			t.Errorf("interpolateLine(%q) = %q, should be unchanged", input, got)
		}
	}
}

func TestInterpolateLineBackticks(t *testing.T) {
	env := map[string]types.Type{
		"x": types.NewNumber(decimal.NewFromInt(42)),
	}
	df := display.DefaultFormatter()

	tests := []struct {
		input string
		want  string
	}{
		{"`{{x}}`", "42"},                  // Backticks stripped, bold wrapped
		{"Value: `{{x}}`", "Value: 42"},    // Inline backtick-wrapped
		{"`{{ x }}`", "42"},                // Whitespace + backticks
		{"{{x}}", "42"},                    // No backticks still works
		{"`{{unknown}}`", "`{{unknown}}`"}, // Missing var: backticks preserved
	}
	for _, tt := range tests {
		got := interpolateLine(tt.input, env, df, nil, nil, false)
		if got != tt.want {
			t.Errorf("interpolateLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestInterpolateLineWithTransform(t *testing.T) {
	// Scale requires unit_categories to be set — use Currency with explicit category
	env := map[string]types.Type{
		"cost": types.NewCurrency(decimal.NewFromInt(500), "USD"),
	}
	df := display.DefaultFormatter()

	fm := &document.Frontmatter{
		Scale: &document.ScaleConfig{
			Factor:         decimal.NewFromInt(1000),
			UnitCategories: []string{"Currency"},
		},
	}

	got := interpolateLine("Total: {{cost}}", env, df, fm, nil, false)
	// 500 * 1000 = 500,000 → formatted as $500K
	want := "Total: $500K"
	if got != want {
		t.Errorf("interpolateLine() with scale = %q, want %q", got, want)
	}
}

func TestInterpolateTextBlocks(t *testing.T) {
	// Build a document with a TextBlock containing {{var}} followed by a CalcBlock
	doc, err := document.NewDocument("Total: {{x}}\n\n\nx = 42\n")
	if err != nil {
		t.Fatal(err)
	}

	env := map[string]types.Type{
		"x": types.NewNumber(decimal.NewFromInt(42)),
	}
	df := display.DefaultFormatter()

	interpolateTextBlocks(doc, env, df)

	// Find the TextBlock
	for _, node := range doc.GetBlocks() {
		tb, ok := node.Block.(*document.TextBlock)
		if !ok {
			continue
		}
		got := tb.InterpolatedSource()
		if slices.Contains(got, "Total: 42") {
			// Raw source unchanged
			if slices.Contains(tb.Source(), "Total: {{x}}") {
				return // Success
			}
			t.Error("Source() should still contain {{x}}")
			return
		}
		t.Errorf("InterpolatedSource() should contain 'Total: 42', got %v", got)
		return
	}
	t.Error("expected a TextBlock in document")
}

func TestInterpolateLineHTML(t *testing.T) {
	env := map[string]types.Type{
		"rev": types.NewCurrency(decimal.NewFromFloat(4200000), "USD"),
	}
	df := display.DefaultFormatter()

	got := interpolateLine("Revenue: {{rev}}", env, df, nil, nil, true)
	want := "Revenue: \x02$4.2M\x03"
	if got != want {
		t.Errorf("interpolateLine(wrapHTML=true) = %q, want %q", got, want)
	}
}

func TestInterpolateLineHTMLMissingVar(t *testing.T) {
	env := map[string]types.Type{}
	df := display.DefaultFormatter()

	got := interpolateLine("Value: {{unknown}}", env, df, nil, nil, true)
	want := "Value: {{unknown}}"
	if got != want {
		t.Errorf("interpolateLine(wrapHTML=true, missing) = %q, want %q", got, want)
	}
}

func TestInterpolateTextBlocksHTMLSource(t *testing.T) {
	doc, err := document.NewDocument("Total: {{x}}\n\n\nx = 42\n")
	if err != nil {
		t.Fatal(err)
	}

	env := map[string]types.Type{
		"x": types.NewNumber(decimal.NewFromInt(42)),
	}
	df := display.DefaultFormatter()

	interpolateTextBlocks(doc, env, df)

	for _, node := range doc.GetBlocks() {
		tb, ok := node.Block.(*document.TextBlock)
		if !ok {
			continue
		}
		// Plain source should have bold wrapping
		plain := tb.InterpolatedSource()
		if plain[0] != "Total: 42" {
			t.Errorf("InterpolatedSource()[0] = %q, want %q", plain[0], "Total: 42")
		}
		// HTML source should have sentinels (no bold)
		html := tb.InterpolatedHTMLSourceText()
		if !strings.Contains(html, "Total: \x0242\x03") {
			t.Errorf("InterpolatedHTMLSourceText() = %q, should contain sentinel-wrapped value", html)
		}
		return
	}
	t.Error("expected a TextBlock in document")
}

func TestInterpolateTextBlocksNoChange(t *testing.T) {
	doc, err := document.NewDocument("No tags here\n")
	if err != nil {
		t.Fatal(err)
	}

	env := map[string]types.Type{
		"x": types.NewNumber(decimal.NewFromInt(1)),
	}
	df := display.DefaultFormatter()

	interpolateTextBlocks(doc, env, df)

	for _, node := range doc.GetBlocks() {
		tb, ok := node.Block.(*document.TextBlock)
		if !ok {
			continue
		}
		// Should fall back to Source since no changes
		got := tb.InterpolatedSource()
		if got[0] != "No tags here" {
			t.Errorf("InterpolatedSource should fall back, got %v", got)
		}
		return
	}
}

func TestInterpolateLineDirectiveScale(t *testing.T) {
	env := map[string]types.Type{}
	df := display.DefaultFormatter()
	fm := &document.Frontmatter{
		Scale: &document.ScaleConfig{
			Factor:         decimal.NewFromInt(3),
			UnitCategories: []string{"All"},
		},
	}

	got := interpolateLine("Scale factor: {{ @scale }}", env, df, fm, nil, false)
	want := "Scale factor: 3"
	if got != want {
		t.Errorf("interpolateLine(@scale) = %q, want %q", got, want)
	}
}

func TestInterpolateLineDirectiveGlobals(t *testing.T) {
	env := map[string]types.Type{}
	df := display.DefaultFormatter()
	fm := &document.Frontmatter{
		Globals: map[string]string{
			"tax_rate": "0.32",
		},
	}
	parsed, _ := document.ParseGlobals(fm.Globals)

	got := interpolateLine("Tax: {{ @globals.tax_rate }}", env, df, fm, parsed.Values, false)
	want := "Tax: 0.32"
	if got != want {
		t.Errorf("interpolateLine(@globals.tax_rate) = %q, want %q", got, want)
	}
}

func TestInterpolateLineDirectiveNoFrontmatter(t *testing.T) {
	env := map[string]types.Type{}
	df := display.DefaultFormatter()

	// No frontmatter — directive should remain unresolved
	got := interpolateLine("Factor: {{ @scale }}", env, df, nil, nil, false)
	want := "Factor: {{ @scale }}"
	if got != want {
		t.Errorf("interpolateLine(@scale, no fm) = %q, want %q", got, want)
	}
}

func TestInterpolateLineDirectiveNotScaled(t *testing.T) {
	// Verify that @scale value is NOT itself scaled (no double-scaling)
	env := map[string]types.Type{}
	df := display.DefaultFormatter()
	fm := &document.Frontmatter{
		Scale: &document.ScaleConfig{
			Factor:         decimal.NewFromInt(1000),
			UnitCategories: []string{"All"},
		},
	}

	got := interpolateLine("Factor: {{ @scale }}", env, df, fm, nil, false)
	// Should be 1000 (displayed as 1K), NOT 1000*1000=1000000 (1M)
	want := "Factor: 1K"
	if got != want {
		t.Errorf("interpolateLine(@scale, should not double-scale) = %q, want %q", got, want)
	}
}

// TestInterpolateLine_BoldSourceDoesNotDouble verifies that when the user
// writes **{{ var }}** in their markdown (already bold), the interpolation
// does not produce ****value**** (doubled bold markers).
func TestInterpolateLine_BoldSourceDoesNotDouble(t *testing.T) {
	env := map[string]types.Type{
		"out": types.NewNumber(decimal.NewFromInt(42)),
	}
	df := display.DefaultFormatter()

	got := interpolateLine("We leave on **{{ out }}**", env, df, nil, nil, false)
	// Should be 42, not **42**
	if strings.Contains(got, "****") {
		t.Errorf("interpolateLine() doubled bold markers: %q", got)
	}
	// The value should appear bold (either from user's ** or from interpolation, not both)
	if !strings.Contains(got, "42") {
		t.Errorf("interpolateLine() should contain 42, got: %q", got)
	}
}
