package document

import (
	"regexp"

	"github.com/CalcMark/go-calcmark/format/display"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/CalcMark/go-calcmark/spec/transform"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// interpolationPattern matches {{variable_name}} with optional whitespace.
// Only word characters (letters, digits, underscore) are valid variable names.
var interpolationPattern = regexp.MustCompile(`\{\{\s*(\w+)\s*\}\}`)

// interpolateTextBlocks resolves {{var}} tags in all TextBlocks against
// the final environment. Called after evaluation completes.
func interpolateTextBlocks(doc *document.Document, env map[string]types.Type, df display.Formatter) {
	fm := doc.GetFrontmatter()

	for _, node := range doc.GetBlocks() {
		tb, ok := node.Block.(*document.TextBlock)
		if !ok {
			continue
		}
		interpolated := make([]string, len(tb.Source()))
		changed := false
		for i, line := range tb.Source() {
			resolved := interpolateLine(line, env, df, fm)
			interpolated[i] = resolved
			if resolved != line {
				changed = true
			}
		}
		if changed {
			tb.SetInterpolatedSource(interpolated)
			tb.SetDirty(true) // Force HTML re-render
		}
	}
}

// interpolateLine replaces all {{var}} tags in a single line.
// Unresolved tags are left as-is.
func interpolateLine(line string, env map[string]types.Type, df display.Formatter, fm *document.Frontmatter) string {
	return interpolationPattern.ReplaceAllStringFunc(line, func(match string) string {
		// Extract variable name from the match
		submatch := interpolationPattern.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		varName := submatch[1]

		value, ok := env[varName]
		if !ok || value == nil {
			return match // Leave unresolved tags as-is
		}

		// Apply scale/convert_to transforms to match CalcBlock display
		if fm != nil {
			value = transform.Apply(value, fm.Scale, fm.ConvertTo)
		}

		return df.Format(value)
	})
}
