package document

import (
	"regexp"
	"strings"

	"github.com/CalcMark/go-calcmark/format/display"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/CalcMark/go-calcmark/spec/transform"
	"github.com/CalcMark/go-calcmark/spec/types"
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
		interpolated := make([]string, len(tb.Source()))
		interpolatedHTML := make([]string, len(tb.Source()))
		changed := false
		for i, line := range tb.Source() {
			plain := interpolateLine(line, env, df, fm, parsedGlobals, false)
			interpolated[i] = plain
			interpolatedHTML[i] = interpolateLine(line, env, df, fm, parsedGlobals, true)
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
func interpolateLine(line string, env map[string]types.Type, df display.Formatter, fm *document.Frontmatter, parsedGlobals map[string]types.Type, wrapHTML bool) string {
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
			value = env[ref]
		}

		if value == nil {
			return match // Leave unresolved tags as-is
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
