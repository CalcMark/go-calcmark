package document

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/CalcMark/go-calcmark/v2/impl/interpreter"
	"github.com/CalcMark/go-calcmark/v2/spec/ast"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
	"github.com/CalcMark/go-calcmark/v2/spec/parser"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
)

// Named tables (go-calcmark#118): a markdown table preceded by
//
//	<!-- table: rates (role, rate, hc) -->
//
// registers a *types.Table in the environment under `rates`, with one
// *types.Array per declared column. Tables without a directive are inert
// markdown. Extraction runs during evaluation, in document order, so
// calc blocks below a table can reference it and blocks above cannot.

// Diagnostic codes for table extraction.
const (
	DiagTableDirective   = "table_directive"       // malformed directive / bad identifier
	DiagTableMissing     = "table_missing"         // directive with no table after it
	DiagTableColumns     = "table_column_mismatch" // declared vs actual column count
	DiagTableMixedTypes  = "table_mixed_types"     // a column mixes value types
	DiagTableNameClash   = "table_name_collision"  // name already used by a variable or table
	tableDirectivePrefix = "<!--"
)

// tableSeparatorPattern matches the `|---|:--:|` row under a header.
var tableSeparatorPattern = regexp.MustCompile(`^\s*\|?\s*:?-{1,}:?\s*(\|\s*:?-{1,}:?\s*)*\|?\s*$`)

// extractNamedTables scans one TextBlock for table directives, builds a
// Table for each, and registers it in env. Diagnostics are attached to the
// block with block-relative Line plus document-absolute DocLine/EndLine.
//
// `seen` tracks table names registered earlier in the same full pass so a
// second directive with the same name is diagnosed; pass nil for
// incremental re-evaluation, where re-registering a block's own table is
// the expected path.
func extractNamedTables(blockID string, tb *document.TextBlock, env *interpreter.Environment, seen map[string]bool, lineOff int) {
	lines := tb.Source()
	for i, line := range lines {
		if !strings.Contains(line, tableDirectivePrefix) {
			continue
		}
		m := document.TableDirectivePattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		report := func(code, msg string, lineIdx, col, endCol int) {
			tb.AddDiagnostic(document.Diagnostic{
				BlockID:   blockID,
				Severity:  "error",
				Code:      code,
				Message:   msg,
				Line:      lineIdx + 1,
				Column:    col,
				EndColumn: endCol,
				DocLine:   lineIdx + 1 + lineOff,
				EndLine:   lineIdx + 1 + lineOff,
			})
		}

		name, columns, err := parseTableDirective(m[1], m[2])
		if err != nil {
			report(DiagTableDirective, err.Error(), i, 0, 0)
			continue
		}

		grid, gridErr := findMarkdownTable(lines, i+1)
		if gridErr != nil {
			report(DiagTableMissing, gridErr.Error(), i, 0, 0)
			continue
		}
		if len(grid.header) != len(columns) {
			report(DiagTableColumns, fmt.Sprintf(
				"directive declares %d column%s (%s) but the table has %d",
				len(columns), plural(len(columns)), strings.Join(columns, ", "), len(grid.header)), i, 0, 0)
			continue
		}

		table, cellErr := buildTable(name, columns, grid)
		if cellErr != nil {
			report(cellErr.code, cellErr.msg, cellErr.line, cellErr.col, cellErr.endCol)
			continue
		}

		if existing, ok := env.Get(name); ok {
			if _, isTable := existing.(*types.Table); !isTable {
				report(DiagTableNameClash, fmt.Sprintf("table name %q is already a variable; pick another table name", name), i, 0, 0)
				continue
			}
			if seen != nil && seen[name] {
				report(DiagTableNameClash, fmt.Sprintf("a table named %q is already declared above", name), i, 0, 0)
				continue
			}
		}
		if seen != nil {
			seen[name] = true
		}
		env.Set(name, table)
	}
}

// parseTableDirective normalizes and validates the name and column list.
func parseTableDirective(rawName, rawColumns string) (string, []string, error) {
	name := document.NormalizeTableName(rawName)
	if !isValidIdentifier(name) {
		return "", nil, fmt.Errorf("table name %q is not a valid identifier after normalization (%q)", strings.TrimSpace(rawName), name)
	}
	var columns []string
	seen := map[string]bool{}
	for raw := range strings.SplitSeq(rawColumns, ",") {
		col := document.NormalizeTableName(raw)
		if col == "" {
			continue
		}
		if !isValidIdentifier(col) {
			return "", nil, fmt.Errorf("column name %q is not a valid identifier after normalization (%q)", strings.TrimSpace(raw), col)
		}
		if seen[col] {
			return "", nil, fmt.Errorf("column %q is declared twice", col)
		}
		seen[col] = true
		columns = append(columns, col)
	}
	if len(columns) == 0 {
		return "", nil, fmt.Errorf("table %q declares no columns; write <!-- table: %s (col1, col2) -->", name, name)
	}
	return name, columns, nil
}

func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if r == '_' || unicode.IsLetter(r) {
			continue
		}
		if i > 0 && unicode.IsDigit(r) {
			continue
		}
		return false
	}
	return true
}

// tableCell is one data cell with its 1-based rune column span in the
// source line, so diagnostics can underline the exact cell.
type tableCell struct {
	text   string
	col    int
	endCol int
}

type markdownGrid struct {
	header   []tableCell
	rows     [][]tableCell
	rowLines []int // source line index of each data row
}

// findMarkdownTable locates the pipe table that follows a directive:
// optional blank lines, a header row, a separator row, then data rows.
func findMarkdownTable(lines []string, from int) (*markdownGrid, error) {
	i := from
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	if i >= len(lines) || !isPipeRow(lines[i]) {
		return nil, fmt.Errorf("no markdown table follows the directive (expected a `| header |` row)")
	}
	grid := &markdownGrid{header: splitCells(lines[i])}
	i++
	if i >= len(lines) || !tableSeparatorPattern.MatchString(lines[i]) {
		return nil, fmt.Errorf("the table header must be followed by a `|---|` separator row")
	}
	i++
	for i < len(lines) && isPipeRow(lines[i]) {
		grid.rows = append(grid.rows, splitCells(lines[i]))
		grid.rowLines = append(grid.rowLines, i)
		i++
	}
	return grid, nil
}

func isPipeRow(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "|")
}

// splitCells splits a pipe row into trimmed cells with rune columns.
// Leading and trailing pipes are optional; `\|` is not special (R4 —
// data cells are literals, never markup).
func splitCells(line string) []tableCell {
	runes := []rune(line)
	var cells []tableCell
	start := 0
	flush := func(end int) {
		raw := string(runes[start:end])
		trimmedLeft := strings.TrimLeftFunc(raw, unicode.IsSpace)
		leading := len([]rune(raw)) - len([]rune(trimmedLeft))
		text := strings.TrimRightFunc(trimmedLeft, unicode.IsSpace)
		col := start + leading + 1
		cells = append(cells, tableCell{text: text, col: col, endCol: col + len([]rune(text))})
	}
	for idx, r := range runes {
		if r == '|' {
			if idx > start || (idx == start && idx != 0 && cells != nil) {
				flush(idx)
			}
			start = idx + 1
		}
	}
	// Drop the empty segment before a leading pipe and after a trailing pipe.
	if start < len(runes) && strings.TrimSpace(string(runes[start:])) != "" {
		flush(len(runes))
	}
	// A leading pipe leaves an empty first segment only when text preceded
	// it; a trailing pipe leaves nothing. Remove empty edge cells that came
	// from indentation.
	if len(cells) > 0 && cells[0].text == "" && strings.TrimSpace(string(runes[:cells[0].col-1])) == "" {
		cells = cells[1:]
	}
	return cells
}

type cellError struct {
	code, msg         string
	line, col, endCol int
}

// buildTable parses every cell as a CalcMark literal (falling back to
// Text) and assembles homogeneous column arrays.
func buildTable(name string, columns []string, grid *markdownGrid) (*types.Table, *cellError) {
	colArrays := make(map[string]*types.Array, len(columns))
	for c, col := range columns {
		elements := make([]types.Type, 0, len(grid.rows))
		firstType := ""
		for r, row := range grid.rows {
			cell := tableCell{}
			if c < len(row) {
				cell = row[c]
			}
			value := parseCellValue(cell.text)
			elements = append(elements, value)
			tn := types.TypeNameOf(value)
			if r == 0 {
				firstType = tn
			} else if tn != firstType {
				return nil, &cellError{
					code: DiagTableMixedTypes,
					msg: fmt.Sprintf("column %q mixes %s and %s — every cell in a column must be the same kind of value",
						col, firstType, tn),
					line: grid.rowLines[r], col: cell.col, endCol: cell.endCol,
				}
			}
		}
		arr, err := types.NewArray(elements)
		if err != nil {
			return nil, &cellError{code: DiagTableMixedTypes, msg: err.Error()}
		}
		colArrays[col] = arr
	}
	table, err := types.NewTable(name, columns, colArrays)
	if err != nil {
		return nil, &cellError{code: DiagTableDirective, msg: err.Error()}
	}
	return table, nil
}

// parseCellValue turns one cell into a value. A cell that parses and
// evaluates as a CalcMark literal (`$250`, `3`, `2 weeks`, `10 GB/s`)
// becomes that value; anything else (`Senior`, `n/a`, an empty cell) is
// Text. The evaluation uses an empty environment so a cell can never
// reference a variable (R4: literals only).
func parseCellValue(text string) types.Type {
	text = strings.TrimSpace(text)
	if text == "" {
		return types.NewText("")
	}
	nodes, err := parser.Parse(text + "\n")
	if err != nil || len(nodes) != 1 {
		return types.NewText(text)
	}
	if _, isAssign := nodes[0].(*ast.Assignment); isAssign {
		return types.NewText(text)
	}
	interp := interpreter.NewInterpreterWithEnv(interpreter.NewEnvironment())
	results, err := interp.Eval(nodes)
	if err != nil || len(results) != 1 || results[0] == nil {
		return types.NewText(text)
	}
	return results[0]
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
