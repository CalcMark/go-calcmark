package types

import (
	"strings"
	"testing"
)

func mustArray(t *testing.T, els []Type) *Array {
	t.Helper()
	arr, err := NewArray(els)
	if err != nil {
		t.Fatal(err)
	}
	return arr
}

func TestTable_ColumnLookupAndSummary(t *testing.T) {
	cols := map[string]*Array{
		"rate": mustArray(t, nums(250, 150)),
		"hc":   mustArray(t, nums(3, 5)),
	}
	tbl, err := NewTable("rates", []string{"rate", "hc"}, cols)
	if err != nil {
		t.Fatal(err)
	}
	if tbl.RowCount != 2 {
		t.Errorf("RowCount = %d, want 2", tbl.RowCount)
	}
	if hc, ok := tbl.Column("hc"); !ok || hc.String() != "[3, 5]" {
		t.Errorf("Column(hc) = %v, %v", hc, ok)
	}
	if _, ok := tbl.Column("nonexistent"); ok {
		t.Error("Column(nonexistent) must report false")
	}
	if tbl.String() != "table rates (2 rows, 2 columns)" {
		t.Errorf("String() = %q", tbl.String())
	}
}

func TestTable_HeaderOnlyIsValid(t *testing.T) {
	tbl, err := NewTable("empty", []string{"a"}, map[string]*Array{"a": mustArray(t, nil)})
	if err != nil {
		t.Fatal(err)
	}
	if tbl.RowCount != 0 {
		t.Errorf("RowCount = %d, want 0", tbl.RowCount)
	}
}

func TestTable_RaggedColumnsRejected(t *testing.T) {
	_, err := NewTable("t", []string{"a", "b"}, map[string]*Array{
		"a": mustArray(t, nums(1, 2)),
		"b": mustArray(t, nums(1)),
	})
	if err == nil || !strings.Contains(err.Error(), "rows") {
		t.Errorf("want ragged-columns error, got %v", err)
	}
}

func TestTable_MissingColumnDataRejected(t *testing.T) {
	_, err := NewTable("t", []string{"a", "b"}, map[string]*Array{"a": mustArray(t, nums(1))})
	if err == nil {
		t.Error("want error for column without data")
	}
}
