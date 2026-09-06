package document

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/document"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
)

// A markdown table preceded by `<!-- table: name (cols) -->` registers a
// *types.Table in the environment during evaluation (go-calcmark#118,
// R1–R6). Tables without a directive stay inert markdown.

const ratesDoc = `# SOW

<!-- table: rates (role, rate, hc) -->
| Role   | Rate | Headcount |
|--------|------|-----------|
| Senior | $250 | 3         |
| Junior | $150 | 5         |

Some prose.
`

func evalDoc(t *testing.T, src string) (*Evaluator, *document.Document) {
	t.Helper()
	doc, err := document.NewDocument(src)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	ev := NewEvaluator()
	_ = ev.Evaluate(doc)
	return ev, doc
}

func lookupTable(t *testing.T, ev *Evaluator, name string) *types.Table {
	t.Helper()
	v, ok := ev.GetEnvironment().Get(name)
	if !ok {
		t.Fatalf("table %q not registered in environment", name)
	}
	tbl, ok := v.(*types.Table)
	if !ok {
		t.Fatalf("%q is %T, want *types.Table", name, v)
	}
	return tbl
}

func textBlockDiagnostics(doc *document.Document) []document.Diagnostic {
	var out []document.Diagnostic
	for _, n := range doc.GetBlocks() {
		if tb, ok := n.Block.(*document.TextBlock); ok {
			out = append(out, tb.Diagnostics()...)
		}
	}
	return out
}

func TestNamedTable_RegistersColumnsAsArrays(t *testing.T) {
	ev, doc := evalDoc(t, ratesDoc)
	tbl := lookupTable(t, ev, "rates")
	if tbl.RowCount != 2 || strings.Join(tbl.ColumnOrder, ",") != "role,rate,hc" {
		t.Errorf("table = %s, columns %v", tbl, tbl.ColumnOrder)
	}
	rate, _ := tbl.Column("rate")
	if rate.ElementType != "Currency" || rate.String() != "[$250.00, $150.00]" {
		t.Errorf("rate column = %q (%s)", rate.String(), rate.ElementType)
	}
	hc, _ := tbl.Column("hc")
	if hc.ElementType != "Number" || hc.String() != "[3, 5]" {
		t.Errorf("hc column = %q (%s)", hc.String(), hc.ElementType)
	}
	role, _ := tbl.Column("role")
	if role.ElementType != "Text" || role.String() != "[Senior, Junior]" {
		t.Errorf("role column = %q (%s)", role.String(), role.ElementType)
	}
	if d := textBlockDiagnostics(doc); len(d) != 0 {
		t.Errorf("unexpected diagnostics: %+v", d)
	}
}

func TestNamedTable_DirectiveNamesAreNormalized(t *testing.T) {
	ev, _ := evalDoc(t, "<!-- table: Q1 Sales (Region, Total Rev) -->\n| R | T |\n|---|---|\n| East | 10 |\n")
	tbl := lookupTable(t, ev, "q1_sales")
	if _, ok := tbl.Column("total_rev"); !ok {
		t.Errorf("columns = %v, want total_rev", tbl.ColumnOrder)
	}
}

func TestNamedTable_ColumnCountMismatchIsDiagnosed(t *testing.T) {
	_, doc := evalDoc(t, "<!-- table: t (a, b) -->\n| A | B | C |\n|---|---|---|\n| 1 | 2 | 3 |\n")
	diags := textBlockDiagnostics(doc)
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "2 column") || diags[0].Line != 1 {
		t.Errorf("want one column-count diagnostic on line 1, got %+v", diags)
	}
}

func TestNamedTable_MixedTypeColumnIsDiagnosedAtTheCell(t *testing.T) {
	_, doc := evalDoc(t, "<!-- table: t (a) -->\n| A |\n|---|\n| $100 |\n| 50% |\n")
	diags := textBlockDiagnostics(doc)
	if len(diags) != 1 {
		t.Fatalf("want one diagnostic, got %+v", diags)
	}
	d := diags[0]
	if d.Line != 5 || d.Column == 0 || d.EndColumn <= d.Column {
		t.Errorf("diagnostic should point at the `50%%` cell on line 5: %+v", d)
	}
}

func TestNamedTable_DirectiveWithoutTableIsDiagnosed(t *testing.T) {
	_, doc := evalDoc(t, "<!-- table: t (a) -->\n\nJust prose here.\n")
	diags := textBlockDiagnostics(doc)
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "no markdown table") {
		t.Errorf("want a missing-table diagnostic, got %+v", diags)
	}
}

func TestNamedTable_DuplicateNameIsDiagnosed(t *testing.T) {
	src := "<!-- table: t (a) -->\n| A |\n|---|\n| 1 |\n\n<!-- table: t (a) -->\n| A |\n|---|\n| 2 |\n"
	_, doc := evalDoc(t, src)
	diags := textBlockDiagnostics(doc)
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "already") {
		t.Errorf("want a duplicate-name diagnostic, got %+v", diags)
	}
}

func TestNamedTable_NameCollidingWithVariableIsDiagnosed(t *testing.T) {
	src := "t = 5\n\n<!-- table: t (a) -->\n| A |\n|---|\n| 1 |\n"
	_, doc := evalDoc(t, src)
	diags := textBlockDiagnostics(doc)
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "variable") {
		t.Errorf("want a collision diagnostic, got %+v", diags)
	}
}

func TestNamedTable_InvalidIdentifierIsDiagnosed(t *testing.T) {
	_, doc := evalDoc(t, "<!-- table: 2fast (a) -->\n| A |\n|---|\n| 1 |\n")
	diags := textBlockDiagnostics(doc)
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "identifier") {
		t.Errorf("want an invalid-identifier diagnostic, got %+v", diags)
	}
}

func TestNamedTable_TableWithoutDirectiveIsInert(t *testing.T) {
	ev, doc := evalDoc(t, "| A | B |\n|---|---|\n| 1 | 2 |\n")
	if _, ok := ev.GetEnvironment().Get("a"); ok {
		t.Error("undirected table must not register anything")
	}
	if d := textBlockDiagnostics(doc); len(d) != 0 {
		t.Errorf("unexpected diagnostics: %+v", d)
	}
}
