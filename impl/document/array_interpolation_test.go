package document

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/format/display"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/shopspring/decimal"
)

// `{{table.column}}` resolves through the table; `{{array_var}}` inside a
// markdown table row substitutes the element for that data row
// (go-calcmark#118, R13–R15).

func TestInterpolateLine_DottedTableReference(t *testing.T) {
	rate, _ := types.NewArray([]types.Type{
		types.NewCurrency(decimal.NewFromInt(250), "$"),
		types.NewCurrency(decimal.NewFromInt(150), "$"),
	})
	rates, _ := types.NewTable("rates", []string{"rate"}, map[string]*types.Array{"rate": rate})
	env := map[string]types.Type{"rates": rates}
	got := interpolateLine("Rates: {{rates.rate}}", env, display.DefaultFormatter(), nil, nil, false)
	if got != "Rates: [$250.00, $150.00]" {
		t.Errorf("got %q", got)
	}
	if got := interpolateLine("{{rates.nope}}", env, display.DefaultFormatter(), nil, nil, false); got != "{{rates.nope}}" {
		t.Errorf("unknown column must stay unresolved, got %q", got)
	}
}

const sowDoc = `<!-- table: rates (role, rate, hc) -->
| Role   | Rate | HC |
|--------|------|----|
| Senior | $250 | 2  |
| Junior | $150 | 5  |

costs = rates.rate * rates.hc
total = sum(costs)

| Role   | Line cost |
|--------|-----------|
| Senior | {{costs}} |
| Junior | {{costs}} |

Grand total: {{total}}. All costs: {{costs}}.
`

func interpolatedText(t *testing.T, src string) ([]string, []document.Diagnostic) {
	t.Helper()
	doc, err := document.NewDocument(src)
	if err != nil {
		t.Fatal(err)
	}
	_ = NewEvaluator().Evaluate(doc)
	var lines []string
	var diags []document.Diagnostic
	for _, n := range doc.GetBlocks() {
		if tb, ok := n.Block.(*document.TextBlock); ok {
			lines = append(lines, tb.InterpolatedSourceText())
			diags = append(diags, tb.Diagnostics()...)
		}
	}
	return lines, diags
}

func TestArrayInterpolation_PerRowInTable(t *testing.T) {
	lines, diags := interpolatedText(t, sowDoc)
	text := strings.Join(lines, "\n")
	if !strings.Contains(text, "| Senior | $500.00 |") || !strings.Contains(text, "| Junior | $750.00 |") {
		t.Errorf("rows did not get their own element:\n%s", text)
	}
	if !strings.Contains(text, "Grand total: $1,250.00.") {
		t.Errorf("scalar interpolation broke:\n%s", text)
	}
	if !strings.Contains(text, "All costs: [$500.00, $750.00].") {
		t.Errorf("array in prose should render as a list:\n%s", text)
	}
	if len(diags) != 0 {
		t.Errorf("unexpected diagnostics: %+v", diags)
	}
}

func TestArrayInterpolation_LengthMismatchLeavesTagsAndDiagnoses(t *testing.T) {
	src := "<!-- table: t (n) -->\n| N |\n|---|\n| 1 |\n| 2 |\n\ndoubled = t.n * 2\n\n| Row | Value |\n|-----|-------|\n| a | {{doubled}} |\n| b | {{doubled}} |\n| c | {{doubled}} |\n"
	lines, diags := interpolatedText(t, src)
	text := strings.Join(lines, "\n")
	if strings.Count(text, "{{doubled}}") != 3 {
		t.Errorf("mismatched array must leave every tag unresolved:\n%s", text)
	}
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "3 row") || !strings.Contains(diags[0].Message, "2 value") {
		t.Errorf("want one length-mismatch diagnostic naming 3 rows and 2 values, got %+v", diags)
	}
}

func TestArrayInterpolation_ScalarInTableRepeats(t *testing.T) {
	src := "x = 7\n\n| A | B |\n|---|---|\n| 1 | {{x}} |\n| 2 | {{x}} |\n"
	lines, _ := interpolatedText(t, src)
	text := strings.Join(lines, "\n")
	if strings.Count(text, "| 7 |") != 2 {
		t.Errorf("scalar must repeat on every row:\n%s", text)
	}
}
