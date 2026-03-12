# hero.tape - CalcMark hero demo GIF (TUI editor session)
# Regenerate: task record-demos
#
# IMPORTANT: VHS Enter keystrokes can get swallowed when the TUI is busy
# evaluating. Use Sleep 1s minimum after each Enter before the next Type.
# For lines that trigger heavy evaluation (frontmatter, many variables),
# use Sleep 1.5s or more.
#
# IMPORTANT: Escape before Enter dismisses any autocomplete popup that may
# have appeared while typing. Without it, Enter accepts the suggestion
# instead of creating a newline.
#
# Shows: @scale, @convert_to, {{interpolation}}, NL sum, percentage widening,
# @scale directive reference, unit conversion.
Output site/assets/images/hero.gif

Set Shell "bash"
Set FontSize 18
Set Width 900
Set Height 560
Set Theme "Catppuccin Mocha"
Set Padding 15
Set TypingSpeed 55ms
Set Framerate 12

Require ./cm

# Launch editor with no file (opens blank document)
Hide
Type "./cm"
Enter
Show

# Wait for the TUI editor to render
Sleep 2s

# Frontmatter — type the opening delimiter and content
Type "---"
Sleep 500ms
Escape
Enter
Sleep 1s

Type "scale:"
Sleep 300ms
Escape
Enter
Sleep 1s

Type "  factor: 4"
Sleep 300ms
Escape
Enter
Sleep 1s

Type "  unit_categories: [Mass, Volume]"
Sleep 300ms
Escape
Enter
Sleep 1s

Type "convert_to: si"
Sleep 300ms
Escape
Enter
Sleep 1s

Type "---"
Sleep 500ms
Escape
Enter
Sleep 1.5s

# Markdown heading
Type "# Party Guacamole"
Escape
Enter
Sleep 1s

Escape
Enter
Sleep 1s

# Prose with interpolation placeholders
Type "Makes {{servings}} -- needs {{avocados}} of avocados."
Escape
Enter
Sleep 1s

Escape
Enter
Sleep 1s

# Section heading
Type "## Ingredients (1 batch)"
Escape
Enter
Sleep 1s

Escape
Enter
Sleep 1s

# Recipe variables (scaled + converted by frontmatter)
Type "avocados = 3 avocados"
Escape
Enter
Sleep 1s

Type "lime_juice = 2 tablespoons"
Escape
Enter
Sleep 1s

Type "onion = 0.25 cups"
Escape
Enter
Sleep 1s

Type "servings = 4 servings"
Escape
Enter
Sleep 1s

Escape
Enter
Sleep 1s

# Cost section
Type "## Cost"
Escape
Enter
Sleep 1s

Escape
Enter
Sleep 1s

Type "ingredients = $6.50"
Escape
Enter
Sleep 1s

Type "with_tax = ingredients + 8%"
Escape
Enter
Sleep 1s

Type "per_batch = with_tax / @scale"
Escape
Enter
Sleep 1s

# NL function: sum
Type "sum of ingredients, $2, $1.50"
Escape
Enter
Sleep 1s

Escape
Enter
Sleep 1s

# Unit conversions
Type "## Quick math"
Escape
Enter
Sleep 1s

Escape
Enter
Sleep 1s

Type "500 grams in oz"
Escape
Enter
Sleep 1s

Type "350 fahrenheit in celsius"
Sleep 1.5s

Escape
Enter
Sleep 2s

Ctrl+C
