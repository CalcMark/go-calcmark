package types

import "fmt"

// Table is a named markdown table registered in the environment by a
// `<!-- table: name (col1, col2) -->` directive. Each column is an Array;
// ColumnOrder preserves declaration order for display and completion.
type Table struct {
	Name        string
	ColumnOrder []string
	Columns     map[string]*Array
	RowCount    int
}

// NewTable builds a Table from ordered column names and their arrays.
// Every column must be present and have the same length.
func NewTable(name string, columnOrder []string, columns map[string]*Array) (*Table, error) {
	if len(columnOrder) == 0 {
		return nil, fmt.Errorf("table %q declares no columns", name)
	}
	rows := -1
	for _, col := range columnOrder {
		arr, ok := columns[col]
		if !ok || arr == nil {
			return nil, fmt.Errorf("table %q: column %q has no data", name, col)
		}
		if rows == -1 {
			rows = arr.Len()
		} else if arr.Len() != rows {
			return nil, fmt.Errorf("table %q: column %q has %d rows but %q has %d",
				name, col, arr.Len(), columnOrder[0], rows)
		}
	}
	return &Table{Name: name, ColumnOrder: columnOrder, Columns: columns, RowCount: rows}, nil
}

// Column returns the named column's array.
func (t *Table) Column(name string) (*Array, bool) {
	arr, ok := t.Columns[name]
	return arr, ok
}

// String returns a short summary; a table is never displayed inline.
func (t *Table) String() string {
	return fmt.Sprintf("table %s (%d rows, %d columns)", t.Name, t.RowCount, len(t.ColumnOrder))
}
