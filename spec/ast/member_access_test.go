package ast

import "testing"

func TestMemberAccess_StringAndScaleRef(t *testing.T) {
	ma := &MemberAccess{Object: &Identifier{Name: "sales"}, Field: "q1"}
	if ma.String() != "sales.q1" {
		t.Errorf("String() = %q", ma.String())
	}
	if ContainsScaleRef(ma) {
		t.Error("plain member access must not contain a scale ref")
	}
	withScale := &MemberAccess{Object: &DirectiveRef{Directive: "scale"}, Field: "x"}
	if !ContainsScaleRef(withScale) {
		t.Error("ContainsScaleRef must recurse into Object")
	}
	r := &Range{Start: Position{Line: 1, Column: 1}, End: Position{Line: 1, Column: 9}}
	SetRangeIfMissing(ma, r)
	if ma.GetRange() != r {
		t.Error("SetRangeIfMissing must cover MemberAccess")
	}
}
