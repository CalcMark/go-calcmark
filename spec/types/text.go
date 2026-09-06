package types

// Text is a display-only value: a table cell that is not a CalcMark
// literal (a role name, a label). It can be counted and interpolated but
// never takes part in arithmetic — ToDecimal rejects it like any other
// non-numeric type. Text exists only as an Array element; there is no
// literal syntax for it.
type Text struct {
	Value string
}

// NewText creates a Text value.
func NewText(value string) *Text {
	return &Text{Value: value}
}

// String returns the text verbatim.
func (t *Text) String() string {
	return t.Value
}
