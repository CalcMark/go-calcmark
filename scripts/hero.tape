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
# Shows: frontmatter @scale, custom units (patties/buns/cans), NL sum,
# percentage widening, currency, and an error (undefined variable).
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

# --- Frontmatter ---
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

Type "  factor: 2"
Sleep 300ms
Escape
Enter
Sleep 1s

Type "  unit_categories: [Custom]"
Sleep 300ms
Escape
Enter
Sleep 1s

Type "---"
Sleep 500ms
Escape
Enter
Sleep 1.5s

# --- Content ---
Type "# BBQ Shopping"
Escape
Enter
Sleep 1s

Escape
Enter
Sleep 1s

Type "burgers = 8 patties"
Escape
Enter
Sleep 1s

Type "buns = 10 buns"
Escape
Enter
Sleep 1s

Type "drinks = 12 cans"
Escape
Enter
Sleep 1s

Type "meat = $32"
Escape
Enter
Sleep 1s

Type "sides = $15"
Escape
Enter
Sleep 1s

# NL function
Type "total = sum of meat, sides, $5"
Escape
Enter
Sleep 1s

# Percentage widening
Type "tip = total + 15%"
Escape
Enter
Sleep 1s

Escape
Enter
Sleep 1s

# Markdown context before the error
Type "## Get help when things go wrong"
Escape
Enter
Sleep 1s

Escape
Enter
Sleep 1s

# Error: undefined variable — shows diagnostic footer
Type "per_person = tip / guests"
Sleep 2s

Escape
Enter
Sleep 2s

Ctrl+C
