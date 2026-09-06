package document

import "testing"

// A table directive makes its name a known name for the rest of the
// document, so `rates.rate * 2` classifies as a calculation the same way
// `x * 2` does after `x = 1` (go-calcmark#118, unit 10).
func TestDetector_TableNameIsKnownForDotAccessLines(t *testing.T) {
	src := "<!-- table: rates (rate, hc) -->\n| Rate | HC |\n|------|----|\n| $1 | 2 |\n\nrates.rate * 2\nsum(rates.rate)\n"
	doc, err := NewDocument(src)
	if err != nil {
		t.Fatal(err)
	}
	var calc, text int
	for _, n := range doc.GetBlocks() {
		switch n.Block.(type) {
		case *CalcBlock:
			calc++
		case *TextBlock:
			text++
		}
	}
	if calc != 1 || text != 1 {
		t.Errorf("want 1 text block (directive+table) and 1 calc block, got text=%d calc=%d", text, calc)
	}
}

func TestDetector_ProseWithDotStaysText(t *testing.T) {
	src := "The rates.rate column is shown below.\nSee config.yaml for details.\n"
	doc, err := NewDocument(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range doc.GetBlocks() {
		if _, ok := n.Block.(*CalcBlock); ok {
			t.Errorf("prose with dotted words must not classify as calc: %q", n.Block.Source())
		}
	}
}

func TestTableDirectiveName(t *testing.T) {
	cases := map[string]string{
		"<!-- table: rates (rate, hc) -->": "rates",
		"  <!--table: Q1 Sales (a) -->":    "q1_sales",
		"<!-- not a table directive -->":   "",
		"| rates.rate | a cell |":          "",
	}
	for line, want := range cases {
		got, _ := TableDirectiveName(line)
		if got != want {
			t.Errorf("TableDirectiveName(%q) = %q, want %q", line, got, want)
		}
	}
}
