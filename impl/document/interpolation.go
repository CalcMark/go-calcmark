package document

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/CalcMark/go-calcmark/v2/format/display"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
	"github.com/CalcMark/go-calcmark/v2/spec/transform"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
)

// interpolationPattern matches {{variable_name}}, {{@scale}}, or {{@globals.field}}
// with optional whitespace and optional surrounding backticks.
// Backticks are consumed so interpolated values render as plain inline text.
var interpolationPattern = regexp.MustCompile("`?" + `\{\{\s*(@?\w+(?:\.\w+)?)\s*\}\}` + "`?")

// interpolateTextBlocks resolves {{var}} tags in all TextBlocks against
// the final environment. Called after evaluation completes.
func interpolateTextBlocks(doc *document.Document, env map[string]types.Type, df display.Formatter) {
	fm := doc.GetFrontmatter()

	// Pre-parse globals once for the entire interpolation pass to avoid
	// redundant ParseGlobals calls on every {{ @globals.field }} match.
	var parsedGlobals map[string]types.Type
	if fm != nil && len(fm.Globals) > 0 {
		parsed, err := document.ParseGlobals(fm.Globals)
		if err == nil {
			parsedGlobals = parsed.Values
		}
	}

	for _, node := range doc.GetBlocks() {
		tb, ok := node.Block.(*document.TextBlock)
		if !ok {
			continue
		}
		lines := tb.Source()
		rows := markdownTableRows(lines)
		interpolated := make([]string, len(lines))
		interpolatedHTML := make([]string, len(lines))
		changed := false
		mismatches := map[string]bool{} // "var@tableStart" → reported once per table
		lineOff := blockLineOffset(doc, node.ID)
		for i, line := range lines {
			row := rows[i]
			onMismatch := func(name string, have int) {
				key := fmt.Sprintf("%s@%d", name, row.tableStart)
				if mismatches[key] {
					return
				}
				mismatches[key] = true
				tb.AddDiagnostic(document.Diagnostic{
					BlockID:  node.ID,
					Severity: "error",
					Code:     DiagInterpolationRows,
					Message: fmt.Sprintf("{{%s}} has %d value%s but this table has %d row%s — one value per data row is required",
						name, have, plural(have), row.rowCount, plural(row.rowCount)),
					Line:    row.tableStart + 1,
					DocLine: row.tableStart + 1 + lineOff,
					EndLine: row.tableStart + 1 + lineOff,
				})
			}
			plain := interpolateLineInRow(line, env, df, fm, parsedGlobals, false, row, onMismatch)
			interpolated[i] = plain
			interpolatedHTML[i] = interpolateLineInRow(line, env, df, fm, parsedGlobals, true, row, onMismatch)
			if plain != line {
				changed = true
			}
		}
		if changed {
			tb.SetInterpolatedSource(interpolated)
			tb.SetInterpolatedHTMLSource(interpolatedHTML)
			tb.SetDirty(true) // Force HTML re-render
		}
	}
}

// DiagInterpolationRows: an array interpolated into a markdown table has
// a different length than the table has data rows (go-calcmark#118, R15).
const DiagInterpolationRows = "interpolation_row_mismatch"

// tableRow describes a source line's place in a markdown table: which
// data row it is (0-based, -1 when not a data row), where that table's
// first data row sits, and how many data rows the table has.
type tableRow struct {
	index      int
	tableStart int
	rowCount   int
}

var notInTable = tableRow{index: -1, tableStart: -1}

// markdownTableRows maps every line to its tableRow. A table is a pipe
// row followed by a `|---|` separator row and then consecutive pipe rows.
func markdownTableRows(lines []string) []tableRow {
	rows := make([]tableRow, len(lines))
	for i := range rows {
		rows[i] = notInTable
	}
	for i := 0; i+1 < len(lines); i++ {
		if !isPipeRow(lines[i]) || !tableSeparatorPattern.MatchString(lines[i+1]) {
			continue
		}
		start := i + 2
		end := start
		for end < len(lines) && isPipeRow(lines[end]) {
			end++
		}
		for r := start; r < end; r++ {
			rows[r] = tableRow{index: r - start, tableStart: start, rowCount: end - start}
		}
		i = end - 1
	}
	return rows
}

// interpolateLine replaces all {{var}} tags in a single line.
// Unresolved tags are left as-is. When wrapHTML is true, resolved values
// are wrapped with STX/ETX sentinels for post-processing into <span> tags.
func interpolateLine(line string, env map[string]types.Type, df display.Formatter, fm *document.Frontmatter, parsedGlobals map[string]types.Type, wrapHTML bool) string {
	return interpolateLineInRow(line, env, df, fm, parsedGlobals, wrapHTML, notInTable, nil)
}

// interpolateLineInRow is interpolateLine with table-row awareness: an
// Array value inside a data row substitutes the element for that row
// (R15). A length mismatch leaves the tag unresolved and reports through
// onMismatch. Outside a table an Array renders as `[a, b]`.
func interpolateLineInRow(line string, env map[string]types.Type, df display.Formatter, fm *document.Frontmatter, parsedGlobals map[string]types.Type, wrapHTML bool, row tableRow, onMismatch func(name string, have int)) string {
	return interpolationPattern.ReplaceAllStringFunc(line, func(match string) string {
		// Extract variable name from the match
		submatch := interpolationPattern.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		ref := submatch[1]

		var value types.Type
		isDirective := strings.HasPrefix(ref, "@")
		if isDirective {
			value = resolveDirectiveForInterpolation(ref, fm, parsedGlobals)
		} else {
			value = resolveEnvRef(ref, env)
		}

		if value == nil {
			return match // Leave unresolved tags as-is
		}

		if arr, ok := value.(*types.Array); ok && row.index >= 0 {
			if arr.Len() != row.rowCount {
				if onMismatch != nil {
					onMismatch(ref, arr.Len())
				}
				return match
			}
			value = arr.Elements[row.index]
		}

		// Apply scale/convert_to transforms to match CalcBlock display.
		// Directive values are already the raw factor/value — skip transform
		// to avoid double-scaling (consistent with R3: explicit @scale opts out).
		if fm != nil && !isDirective {
			value = transform.Apply(value, fm.Scale, fm.ConvertTo)
		}

		formatted := df.Format(value)
		if wrapHTML {
			return "\x02" + formatted + "\x03"
		}
		return formatted
	})
}

// resolveEnvRef looks up a plain `name` or a dotted `table.column`
// reference (go-calcmark#118, R13). Unknown names and columns resolve to
// nil so the tag stays visible in the output.
func resolveEnvRef(ref string, env map[string]types.Type) types.Type {
	if v, ok := env[ref]; ok {
		return v
	}
	tableName, column, dotted := strings.Cut(ref, ".")
	if !dotted {
		return nil
	}
	table, ok := env[tableName].(*types.Table)
	if !ok {
		return nil
	}
	arr, ok := table.Column(column)
	if !ok {
		return nil
	}
	return arr
}

// resolveDirectiveForInterpolation resolves @scale or @globals.field
// from frontmatter for use in {{ }} interpolation tags.
// parsedGlobals is a pre-parsed cache of globals values (may be nil).
func resolveDirectiveForInterpolation(ref string, fm *document.Frontmatter, parsedGlobals map[string]types.Type) types.Type {
	if fm == nil {
		return nil
	}

	// Strip the leading "@"
	name := ref[1:]

	if name == "scale" {
		if fm.Scale == nil {
			return nil
		}
		return types.NewNumber(fm.Scale.Factor)
	}

	// @globals.field
	if field, ok := strings.CutPrefix(name, "globals."); ok {
		if parsedGlobals != nil {
			return parsedGlobals[field]
		}
		return nil
	}

	return nil
}
