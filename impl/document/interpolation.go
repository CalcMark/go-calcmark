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
		interpolatedHTML := make([]string, len(tb.Source()))
		changed := false
		for i, line := range tb.Source() {
			plain := interpolateLine(line, env, df, fm, false)
			interpolated[i] = plain
			interpolatedHTML[i] = interpolateLine(line, env, df, fm, true)
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

// interpolateLine replaces all {{var}} tags in a single line.
// Unresolved tags are left as-is. When wrapHTML is true, resolved values
// are wrapped with STX/ETX sentinels for post-processing into <span> tags.
func interpolateLine(line string, env map[string]types.Type, df display.Formatter, fm *document.Frontmatter, wrapHTML bool) string {
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

		formatted := df.Format(value)
		if wrapHTML {
			return "\x02" + formatted + "\x03"
		}
		return formatted
	})
}
