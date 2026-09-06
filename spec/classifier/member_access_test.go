package classifier

import "testing"

type namesResolver map[string]bool

func (n namesResolver) Has(name string) bool { return n[name] }

// `table.column` expressions classify as calculations when the table is
// a known name (go-calcmark#118, unit 10 — the classifier is the most
// commonly missed layer).
func TestClassifyLine_MemberAccess(t *testing.T) {
	env := namesResolver{"rates": true}
	lt, err := ClassifyLine("rates.rate * 2", env)
	if err != nil || lt != Calculation {
		t.Errorf("known table: got %v, %v", lt, err)
	}
	lt, _ = ClassifyLine("ghost.rate * 2", namesResolver{})
	if lt == Calculation {
		t.Errorf("unknown table must not classify as calculation")
	}
}
