package calcmark

import (
	"strings"
	"testing"

	impldoc "github.com/CalcMark/go-calcmark/v2/impl/document"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
)

// Named tables work in Embedded mode too: the directive and table sit in
// the markdown outside the fences, and the fenced calc block reads them
// (go-calcmark#118, addendum). Interpolation of per-row values lands in
// the rendered HTML.
func TestNamedTables_EmbeddedMode(t *testing.T) {
	src := "# SOW\n\n<!-- table: rates (role, rate, hc) -->\n| Role | Rate | HC |\n|------|------|----|\n| Senior | $250 | 2 |\n| Junior | $150 | 5 |\n\n```cm\ncosts = rates.rate * rates.hc\ntotal = sum(costs)\n```\n\n| Role | Cost |\n|------|------|\n| Senior | {{costs}} |\n| Junior | {{costs}} |\n\nTotal: {{total}}\n"
	doc, err := NewDocumentEmbedded(src)
	if err != nil {
		t.Fatal(err)
	}
	ev := impldoc.NewEvaluator()
	if err := ev.Evaluate(doc); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	v, ok := ev.GetEnvironment().Get("rates")
	if !ok {
		t.Fatal("rates table not registered from Embedded-mode markdown")
	}
	if _, ok := v.(*types.Table); !ok {
		t.Fatalf("rates is %T", v)
	}
	total, _ := ev.GetEnvironment().Get("total")
	if total == nil || total.String() != "$1250.00" {
		t.Errorf("total = %v", total)
	}

	html, err := Convert(src, Options{Format: "html", Mode: Embedded})
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	for _, want := range []string{"$500.00", "$750.00", "$1,250.00"} {
		if !strings.Contains(html, want) {
			t.Errorf("HTML lacks %q", want)
		}
	}
}

// Convert(Mode: Embedded) now evaluates the host document as one unit,
// so a variable defined in one fence resolves in the next and prose
// interpolation works — previously each fence was evaluated alone.
func TestConvert_Embedded_CrossFenceVariablesResolve(t *testing.T) {
	src := "```cm\nprice = 100\n```\n\nThe price is {{price}}.\n\n```cm\ntax = price * 0.1\n```\n"
	md, err := Convert(src, Options{Format: "md", Mode: Embedded})
	if err != nil {
		t.Fatalf("Convert: %v\n%s", err, md)
	}
	for _, want := range []string{"tax = price * 0.1 → 10", "The price is 100."} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown lacks %q:\n%s", want, md)
		}
	}
}
